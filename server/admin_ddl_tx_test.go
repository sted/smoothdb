package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// The /admin DDL routes run in the request's transaction like every other
// route. The multi-statement helpers (CreateTable, UpdateTable, UpdateColumn,
// UpdateRole, CreateUser, UpdateSchema, UpdateDatabase) committed that
// transaction themselves, so their change persisted under TransactionMode
// "rollback" and under Prefer: tx=rollback.

type adminTxCase struct {
	name, method, path, body string
	setup, cleanup           []string
	// trace answers true when the change of the route exists
	trace string
}

var adminTxCases = []adminTxCase{
	{
		name: "CreateTable", method: "POST", path: "/admin/databases/postgres/tables",
		body:    `{"name": "admtx_t", "comment": "c", "columns": [{"name": "id", "type": "int"}]}`,
		trace:   `SELECT to_regclass('public.admtx_t') IS NOT NULL`,
		cleanup: []string{`DROP TABLE IF EXISTS admtx_t`},
	},
	{
		name: "UpdateTable", method: "PATCH", path: "/admin/databases/postgres/tables/admtx_u",
		body:    `{"comment": "updated", "rowsecurity": true}`,
		setup:   []string{`CREATE TABLE admtx_u (id int)`},
		trace:   `SELECT coalesce(obj_description('public.admtx_u'::regclass), '') = 'updated'`,
		cleanup: []string{`DROP TABLE IF EXISTS admtx_u`},
	},
	{
		name: "UpdateColumn", method: "PATCH", path: "/admin/databases/postgres/tables/admtx_c/columns/id",
		body:  `{"notnull": true, "comment": "x"}`,
		setup: []string{`CREATE TABLE admtx_c (id int)`},
		trace: `SELECT attnotnull FROM pg_attribute
			WHERE attrelid = 'public.admtx_c'::regclass AND attname = 'id'`,
		cleanup: []string{`DROP TABLE IF EXISTS admtx_c`},
	},
	{
		name: "UpdateRole", method: "PATCH", path: "/admin/roles/admtx_role",
		body:    `{"cancreatedatabases": true}`,
		setup:   []string{`CREATE ROLE admtx_role NOLOGIN`},
		trace:   `SELECT rolcreatedb FROM pg_roles WHERE rolname = 'admtx_role'`,
		cleanup: []string{`DROP ROLE IF EXISTS admtx_role`},
	},
	{
		name: "CreateUser", method: "POST", path: "/admin/users",
		body:    `{"name": "admtx_user"}`,
		trace:   `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'admtx_user')`,
		cleanup: []string{`DROP ROLE IF EXISTS admtx_user`},
	},
	{
		name: "UpdateSchema", method: "PATCH", path: "/admin/databases/postgres/schemas/admtx_s",
		body:    `{"name": "admtx_s2"}`,
		setup:   []string{`CREATE SCHEMA admtx_s`},
		trace:   `SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'admtx_s2')`,
		cleanup: []string{`DROP SCHEMA IF EXISTS admtx_s`, `DROP SCHEMA IF EXISTS admtx_s2`},
	},
	{
		name: "UpdateDatabase", method: "PATCH", path: "/admin/databases/admtx_db",
		body:  `{"owner": "admtx_owner"}`,
		setup: []string{`CREATE ROLE admtx_owner NOLOGIN`, `CREATE DATABASE admtx_db`},
		trace: `SELECT pg_get_userbyid(datdba) = 'admtx_owner' FROM pg_database
			WHERE datname = 'admtx_db'`,
		cleanup: []string{`DROP DATABASE IF EXISTS admtx_db`, `DROP ROLE IF EXISTS admtx_owner`},
	},
}

func adminTxExec(t *testing.T, conn *pgx.Conn, sqls ...string) {
	t.Helper()
	for _, sql := range sqls {
		if _, err := conn.Exec(context.Background(), sql); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
	}
}

func TestAdminDDLFollowsTheRequestTransaction(t *testing.T) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, superuserURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)

	modes := []struct {
		tx, prefer string
		persists   bool
	}{
		{"rollback", "", false},
		{"commit", "", true},
		{"none", "", true},
		{"commit-allow-override", "tx=rollback", false},
		{"rollback-allow-override", "tx=commit", true},
	}
	for i, m := range modes {
		t.Run(m.tx+"/"+m.prefer, func(t *testing.T) {
			addr := fmt.Sprintf("localhost:%d", 8146+i)
			s := newTestServer(t, map[string]any{
				"Address":                  addr,
				"AllowAnon":                true,
				"Database.AnonRole":        "postgres",
				"Database.TransactionMode": m.tx,
				"EnableAdminRoute":         true,
			})
			done := startTestServer(t, s)
			defer func() {
				s.Shutdown(context.Background())
				<-done
			}()
			for _, c := range adminTxCases {
				t.Run(c.name, func(t *testing.T) {
					adminTxExec(t, conn, c.cleanup...)
					adminTxExec(t, conn, c.setup...)
					defer adminTxExec(t, conn, c.cleanup...)

					req, _ := http.NewRequest(c.method, "http://"+addr+c.path, strings.NewReader(c.body))
					req.Header.Set("Content-Type", "application/json")
					if m.prefer != "" {
						req.Header.Set("Prefer", m.prefer)
					}
					resp, err := slowClient.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					if resp.StatusCode >= 300 {
						t.Fatalf("%s %s: %d %s", c.method, c.path, resp.StatusCode, body)
					}
					var exists bool
					if err := conn.QueryRow(ctx, c.trace).Scan(&exists); err != nil {
						t.Fatal(err)
					}
					if exists != m.persists {
						t.Errorf("%s with TransactionMode %q and Prefer %q: persisted = %v, want %v",
							c.name, m.tx, m.prefer, exists, m.persists)
					}
				})
			}
		})
	}
}
