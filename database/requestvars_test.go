package database

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// requestVarsSetup creates an authenticator that cannot use the rv_hidden
// schema and a role it can become that can, and connects as the
// authenticator. Everything is dropped when the test ends.
func requestVarsSetup(t *testing.T) *DbConn {
	t.Helper()
	ctx := context.Background()
	admin, err := dbe.AcquireConnection(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Release()
	drop := []string{
		"DROP SCHEMA IF EXISTS rv_hidden CASCADE",
		"DROP ROLE IF EXISTS rv_authn",
		"DROP ROLE IF EXISTS rv_user",
	}
	for _, q := range append(drop,
		"CREATE ROLE rv_authn LOGIN PASSWORD 'rv' NOINHERIT",
		"CREATE ROLE rv_user NOLOGIN",
		"GRANT rv_user TO rv_authn",
		"CREATE SCHEMA rv_hidden",
		"CREATE TABLE rv_hidden.t (n int)",
		"INSERT INTO rv_hidden.t VALUES (1)",
		"GRANT USAGE ON SCHEMA rv_hidden TO rv_user",
		"GRANT SELECT ON rv_hidden.t TO rv_user",
	) {
		if _, err := admin.Exec(ctx, q); err != nil {
			t.Fatalf("%q: %v", q, err)
		}
	}
	conn, err := pgx.Connect(ctx, "postgresql://rv_authn:rv@localhost:5432/postgres")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Close(ctx)
		admin, err := dbe.AcquireConnection(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		defer admin.Release()
		for _, q := range drop {
			admin.Exec(ctx, q)
		}
	})
	return conn
}

// The variables of a request ride in front of a statement only when the
// connection has it prepared: pgx prepares a new one before the batch, as the
// authenticator. When the connection dropped a statement the requests still
// take for prepared, that prepare fails (here: no USAGE on the schema) before
// anything runs, and the statement must run all the same, behind the
// variables applied on their own.
func TestRequestVarsRecoverAStatementTheConnectionDropped(t *testing.T) {
	conn := requestVarsSetup(t)
	ctx := contextImpl(context.Background(), dbe.mainDb, conn, "rv_user")
	gi := GetSmoothContext(ctx)
	gi.vars = &requestVars{role: "rv_user", claims: `{"role":"rv_user"}`}
	sql := "SELECT n FROM rv_hidden.t"

	// positive control: the authenticator alone cannot even prepare it
	if _, err := conn.Prepare(ctx, "rv_probe", sql); err == nil {
		t.Fatal("the authenticator prepared a statement on a schema it cannot use: the test proves nothing")
	}
	gi.remember1(sql) // taken for prepared, as after a statement pgx later evicted
	if _, err := gi.exec(ctx, sql); err != nil {
		t.Fatalf("a statement the connection dropped: %v", err)
	}
	if !gi.vars.applied {
		t.Error("the statement ran without the variables applied on their own: it did not take the way back")
	}
	if err := resetSession(ctx, conn); err != nil {
		t.Fatal(err)
	}

	// and a statement the connection has prepared under the role runs behind
	// the variables, which leave nothing on the connection
	gi.vars = &requestVars{role: "rv_user", claims: `{"role":"rv_user"}`}
	if _, err := gi.exec(ctx, sql); err != nil {
		t.Fatal(err)
	}
	if gi.vars.applied {
		t.Error("a prepared statement did not carry the variables in its batch")
	}
	var user, claims string
	if err := conn.QueryRow(ctx, "SELECT current_user::text, coalesce(current_setting('request.jwt.claims', true), '')").Scan(&user, &claims); err != nil {
		t.Fatal(err)
	}
	if user != "rv_authn" || claims != "" {
		t.Errorf("after the batch the connection carries %s %q", user, claims)
	}
}

// With TransactionMode "none" a write that sends back its rows begins a
// transaction in the batch of its statement (writeRead), on every way the
// batch can take: the variables applied first on their own (a statement not
// prepared yet, or variables already applied), the variables in the batch
// (a prepared statement), and the way back of a statement the connection
// dropped. The transaction stays open for the middleware to end, and the
// write runs as the request's role.
func TestWriteTxBeginsInTheBatch(t *testing.T) {
	if dbe.config.TransactionMode != "none" {
		t.Skip("needs TransactionMode none")
	}
	conn := requestVarsSetup(t)
	admin, err := dbe.AcquireConnection(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Release()
	if _, err := admin.Exec(context.Background(), "GRANT INSERT ON rv_hidden.t TO rv_user"); err != nil {
		t.Fatal(err)
	}
	count := func(n int) int {
		var c int
		if err := admin.QueryRow(context.Background(), "SELECT count(*) FROM rv_hidden.t WHERE n = $1", n).Scan(&c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	newRequest := func() (context.Context, *SmoothContext) {
		ctx := contextImpl(context.Background(), dbe.mainDb, conn, "rv_user")
		gi := GetSmoothContext(ctx)
		gi.vars = &requestVars{role: "rv_user", claims: `{"role":"rv_user"}`}
		return ctx, gi
	}
	// write runs sql through writeRead and checks that it left the
	// transaction open, as the role, with the row not visible outside
	write := func(ctx context.Context, way, sql string, n int) {
		t.Helper()
		out, rows, err := writeRead(ctx, sql, nil)
		if err != nil || rows != 1 {
			t.Fatalf("%s: %q %d %v", way, out, rows, err)
		}
		if s := conn.PgConn().TxStatus(); s != 'T' {
			t.Fatalf("%s: transaction status %q after the write, want T (open)", way, s)
		}
		var user string
		if err := conn.QueryRow(ctx, "SELECT current_user::text").Scan(&user); err != nil {
			t.Fatal(err)
		}
		if user != "rv_user" {
			t.Errorf("%s: the transaction runs as %s, want rv_user", way, user)
		}
		if count(n) != 0 {
			t.Errorf("%s: the row is visible before the end of the transaction", way)
		}
	}
	end := func(way, sql string, n, want int) {
		t.Helper()
		if _, err := conn.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
		if err := resetSession(context.Background(), conn); err != nil {
			t.Fatal(err)
		}
		if got := count(n); got != want {
			t.Errorf("%s: %d rows after %s, want %d", way, got, sql, want)
		}
	}

	insert2 := "INSERT INTO rv_hidden.t VALUES (2) RETURNING n"

	// not prepared yet: the variables go first, session-level, then BEGIN
	ctx, _ := newRequest()
	write(ctx, "not prepared", insert2, 2)
	end("not prepared", "ROLLBACK", 2, 0)

	// prepared (under the role, above): BEGIN and the variables in the batch
	ctx, gi := newRequest()
	write(ctx, "prepared", insert2, 2)
	if !gi.vars.applied {
		t.Error("prepared: the variables did not ride in the batch of the BEGIN")
	}
	end("prepared", "COMMIT", 2, 1)

	// variables applied already: a plain BEGIN
	ctx, gi = newRequest()
	if _, err := gi.conn(ctx); err != nil {
		t.Fatal(err)
	}
	write(ctx, "applied", "INSERT INTO rv_hidden.t VALUES (4) RETURNING n", 4)
	end("applied", "ROLLBACK", 4, 0)

	// a statement the connection dropped: the way back begins it again
	insert3 := "INSERT INTO rv_hidden.t VALUES (3) RETURNING n"
	ctx, gi = newRequest()
	gi.remember1(insert3)
	write(ctx, "dropped", insert3, 3)
	end("dropped", "ROLLBACK", 3, 0)

	// positive control: without writeRead the write commits with its batch
	ctx, _ = newRequest()
	if _, _, err := querySerialize(ctx, "INSERT INTO rv_hidden.t VALUES (5) RETURNING n", nil); err != nil {
		t.Fatal(err)
	}
	if s := conn.PgConn().TxStatus(); s != 'I' || count(5) != 1 {
		t.Errorf("a read-write without writeRead: status %q, %d rows, want I and 1", s, count(5))
	}
	resetSession(context.Background(), conn)
}
