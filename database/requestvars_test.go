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
