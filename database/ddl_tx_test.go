package database

import (
	"context"
	"testing"
)

// The multi-statement DDL helpers group their statements in a transaction.
// A request connection is already in a transaction when TransactionMode is
// not "none" (PrepareConnection begins it). A second BEGIN there only gives a
// warning, and the helper's COMMIT then ended the request's transaction: the
// DDL persisted under "rollback", and the rest of the request ran without the
// transaction-local role. These tests call each helper inside a transaction
// that the test opens with a role of its own, as PrepareConnection does.

// ddlTxRole is the role of the "request". It differs from the authenticator
// (postgres), so that a transaction that ends too early shows in current_user.
const ddlTxRole = "ddltx_req"

type ddlTxCase struct {
	name string
	// setup runs as the authenticator before the case, and commits
	setup []string
	run   func(ctx context.Context) error
	// trace answers true when the change of the helper exists
	trace string
	// cleanup runs as the authenticator after the case
	cleanup []string
}

func ptrTo[T any](v T) *T { return &v }

func ddlTxCases() []ddlTxCase {
	return []ddlTxCase{
		{
			name: "CreateTable",
			run: func(ctx context.Context) error {
				_, err := CreateTable(ctx, &Table{Name: "ddltx_t", Comment: ptrTo("c"),
					Columns: []Column{{Name: "id", Type: "int", Comment: ptrTo("id")}}})
				return err
			},
			trace:   `SELECT to_regclass('public.ddltx_t') IS NOT NULL`,
			cleanup: []string{`DROP TABLE IF EXISTS ddltx_t`},
		},
		{
			name:  "UpdateTable",
			setup: []string{`CREATE TABLE ddltx_u (id int)`},
			run: func(ctx context.Context) error {
				return UpdateTable(ctx, "ddltx_u", &TableUpdate{Comment: ptrTo("updated"), RowSecurity: ptrTo(true)})
			},
			trace:   `SELECT coalesce(obj_description('public.ddltx_u'::regclass), '') = 'updated'`,
			cleanup: []string{`DROP TABLE IF EXISTS ddltx_u`},
		},
		{
			name:  "UpdateColumn",
			setup: []string{`CREATE TABLE ddltx_c (id int)`},
			run: func(ctx context.Context) error {
				return UpdateColumn(ctx, "ddltx_c", "id", &ColumnUpdate{NotNull: ptrTo(true), Comment: ptrTo("x")})
			},
			trace: `SELECT attnotnull FROM pg_attribute
				WHERE attrelid = 'public.ddltx_c'::regclass AND attname = 'id'`,
			cleanup: []string{`DROP TABLE IF EXISTS ddltx_c`},
		},
		{
			name:  "UpdateRole",
			setup: []string{`CREATE ROLE ddltx_role NOLOGIN`},
			run: func(ctx context.Context) error {
				return UpdateRole(ctx, "ddltx_role", &RoleUpdate{CanCreateDatabases: ptrTo(true)})
			},
			trace:   `SELECT rolcreatedb FROM pg_roles WHERE rolname = 'ddltx_role'`,
			cleanup: []string{`DROP ROLE IF EXISTS ddltx_role`},
		},
		{
			name: "CreateUser",
			run: func(ctx context.Context) error {
				_, err := CreateUser(ctx, &User{Name: "ddltx_user"})
				return err
			},
			trace:   `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'ddltx_user')`,
			cleanup: []string{`DROP ROLE IF EXISTS ddltx_user`},
		},
		{
			name:  "UpdateSchema",
			setup: []string{`CREATE SCHEMA ddltx_s`},
			run: func(ctx context.Context) error {
				return UpdateSchema(ctx, "ddltx_s", &SchemaUpdate{Name: ptrTo("ddltx_s2")})
			},
			trace:   `SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'ddltx_s2')`,
			cleanup: []string{`DROP SCHEMA IF EXISTS ddltx_s`, `DROP SCHEMA IF EXISTS ddltx_s2`},
		},
		{
			name:  "UpdateDatabase",
			setup: []string{`CREATE ROLE ddltx_owner NOLOGIN`, `CREATE DATABASE ddltx_db`},
			run: func(ctx context.Context) error {
				return dbe.UpdateDatabase(ctx, "ddltx_db", &DatabaseUpdate{Owner: ptrTo("ddltx_owner")})
			},
			trace: `SELECT pg_get_userbyid(datdba) = 'ddltx_owner' FROM pg_database
				WHERE datname = 'ddltx_db'`,
			cleanup: []string{`DROP DATABASE IF EXISTS ddltx_db`, `DROP ROLE IF EXISTS ddltx_owner`},
		},
	}
}

// ddlTxSetup gives a context on a fresh database as the authenticator, with
// the request role created; both go when the test ends.
func ddlTxSetup(t *testing.T, dbname string) context.Context {
	t.Helper()
	ctx := lane6DB(t, dbname)
	ddlTxExec(t, ctx, `DROP ROLE IF EXISTS `+ddlTxRole, `CREATE ROLE `+ddlTxRole+` NOLOGIN SUPERUSER`)
	t.Cleanup(func() {
		if _, err := GetConn(ctx).Exec(context.Background(), `DROP ROLE IF EXISTS `+ddlTxRole); err != nil {
			t.Errorf("dropping %s: %v", ddlTxRole, err)
		}
	})
	return ctx
}

func ddlTxExec(t *testing.T, ctx context.Context, sqls ...string) {
	t.Helper()
	for _, sql := range sqls {
		if _, err := GetConn(ctx).Exec(ctx, sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
}

func ddlTxBool(t *testing.T, ctx context.Context, sql string) bool {
	t.Helper()
	var b bool
	if err := GetConn(ctx).QueryRow(ctx, sql).Scan(&b); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	return b
}

func ddlTxCurrentUser(t *testing.T, ctx context.Context) string {
	t.Helper()
	var u string
	if err := GetConn(ctx).QueryRow(ctx, `SELECT current_user`).Scan(&u); err != nil {
		t.Fatal(err)
	}
	return u
}

// ddlTxRun prepares the case, runs it in the given mode and returns whether
// the change persisted. "rollback" and "commit" run the helper inside a
// transaction of the test, with a transaction-local role, and end it as the
// mode says; "none" runs it on an idle connection with a session role.
func ddlTxRun(t *testing.T, ctx context.Context, c ddlTxCase, mode string) bool {
	t.Helper()
	conn := GetConn(ctx)
	ddlTxExec(t, ctx, c.cleanup...)
	ddlTxExec(t, ctx, c.setup...)
	defer func() {
		if conn.PgConn().TxStatus() != 'I' {
			conn.Exec(context.Background(), `ROLLBACK`)
		}
		conn.Exec(context.Background(), `RESET ROLE`)
		for _, sql := range c.cleanup {
			if _, err := conn.Exec(context.Background(), sql); err != nil {
				t.Errorf("%s: %v", sql, err)
			}
		}
	}()

	if mode == "none" {
		ddlTxExec(t, ctx, `SET ROLE `+ddlTxRole)
	} else {
		ddlTxExec(t, ctx, `BEGIN`, `SET LOCAL ROLE `+ddlTxRole)
	}
	if err := c.run(ctx); err != nil {
		t.Fatalf("%s: %v", c.name, err)
	}
	if mode == "none" {
		if s := conn.PgConn().TxStatus(); s != 'I' {
			t.Errorf("%s left the idle connection in transaction status %q", c.name, s)
		}
	} else if s := conn.PgConn().TxStatus(); s != 'T' {
		t.Errorf("%s ended the caller's transaction: status %q, want 'T'", c.name, s)
	}
	if u := ddlTxCurrentUser(t, ctx); u != ddlTxRole {
		t.Errorf("current_user after %s: %q, want %q (the request role)", c.name, u, ddlTxRole)
	}
	switch mode {
	case "rollback":
		ddlTxExec(t, ctx, `ROLLBACK`)
	case "commit":
		ddlTxExec(t, ctx, `COMMIT`)
	default:
		ddlTxExec(t, ctx, `RESET ROLE`)
	}
	return ddlTxBool(t, ctx, c.trace)
}

func TestDDLHelpersInsideARolledBackTransaction(t *testing.T) {
	ctx := ddlTxSetup(t, "ddltx_helpers_rb")
	for _, c := range ddlTxCases() {
		t.Run(c.name, func(t *testing.T) {
			if ddlTxRun(t, ctx, c, "rollback") {
				t.Errorf("%s persisted after the caller's ROLLBACK", c.name)
			}
		})
	}
}

func TestDDLHelpersInsideACommittedTransaction(t *testing.T) {
	ctx := ddlTxSetup(t, "ddltx_helpers_co")
	for _, c := range ddlTxCases() {
		t.Run(c.name, func(t *testing.T) {
			if !ddlTxRun(t, ctx, c, "commit") {
				t.Errorf("%s did not persist after the caller's COMMIT", c.name)
			}
		})
	}
}

func TestDDLHelpersWithoutATransaction(t *testing.T) {
	ctx := ddlTxSetup(t, "ddltx_helpers_none")
	for _, c := range ddlTxCases() {
		t.Run(c.name, func(t *testing.T) {
			if !ddlTxRun(t, ctx, c, "none") {
				t.Errorf("%s did not persist with autocommit", c.name)
			}
		})
	}
}

// A helper that fails inside the caller's transaction undoes its own
// statements only: the caller's work before it survives, and the transaction
// stays usable.
func TestDDLHelperFailureKeepsTheCallersTransaction(t *testing.T) {
	ctx := ddlTxSetup(t, "ddltx_helpers_fail")
	conn := GetConn(ctx)
	ddlTxExec(t, ctx, `CREATE TABLE ddltx_f (id int)`)
	ddlTxExec(t, ctx, `BEGIN`, `SET LOCAL ROLE `+ddlTxRole, `CREATE TABLE ddltx_marker (id int)`)
	defer conn.Exec(context.Background(), `ROLLBACK`)

	// the comment runs, then OWNER TO a role that does not exist fails
	err := UpdateTable(ctx, "ddltx_f", &TableUpdate{Comment: ptrTo("lost"), Owner: ptrTo("ddltx_no_such_role")})
	if err == nil {
		t.Fatal("UpdateTable to an unknown owner: no error")
	}
	if s := conn.PgConn().TxStatus(); s != 'T' {
		t.Fatalf("after the failed helper: transaction status %q, want 'T'", s)
	}
	if !ddlTxBool(t, ctx, `SELECT to_regclass('public.ddltx_marker') IS NOT NULL`) {
		t.Error("the caller's table before the failed helper is gone")
	}
	if ddlTxBool(t, ctx, `SELECT coalesce(obj_description('public.ddltx_f'::regclass), '') = 'lost'`) {
		t.Error("the failed helper kept its comment")
	}
	if u := ddlTxCurrentUser(t, ctx); u != ddlTxRole {
		t.Errorf("current_user after the failed helper: %q, want %q", u, ddlTxRole)
	}
}

// Positive control: on an idle connection a failing helper rolls back all
// its statements, before and after the change.
func TestDDLHelperFailureWithoutATransaction(t *testing.T) {
	ctx := ddlTxSetup(t, "ddltx_helpers_fail_none")
	conn := GetConn(ctx)
	ddlTxExec(t, ctx, `CREATE TABLE ddltx_f (id int)`)
	err := UpdateTable(ctx, "ddltx_f", &TableUpdate{Comment: ptrTo("lost"), Owner: ptrTo("ddltx_no_such_role")})
	if err == nil {
		t.Fatal("UpdateTable to an unknown owner: no error")
	}
	if s := conn.PgConn().TxStatus(); s != 'I' {
		t.Fatalf("after the failed helper: transaction status %q, want 'I'", s)
	}
	if ddlTxBool(t, ctx, `SELECT coalesce(obj_description('public.ddltx_f'::regclass), '') = 'lost'`) {
		t.Error("the failed helper kept its comment")
	}
}
