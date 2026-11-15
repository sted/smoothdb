package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sted/smoothdb/authn"
)

// The raw SQL fields of the admin API (view definitions, column types and
// defaults, function return types, policy expressions) go into the DDL as
// written. The DDL went to PostgreSQL through the simple protocol, which runs
// every statement of the string: a second statement ran after the DDL, and a
// RESET ROLE in it returned to the authenticator. Each case below is a valid
// multi-statement string whose second statement creates admsec_mark.
var adminMultiStatementCases = []struct {
	name, path, body string
	setup            []string
}{
	{
		name: "view definition", path: "/admin/databases/postgres/views",
		body: `{"name": "admsec_v", "definition": "SELECT 1; CREATE TABLE admsec_mark (id int)"}`,
	},
	{
		name: "column default in a new table", path: "/admin/databases/postgres/tables",
		body: `{"name": "admsec_t", "columns": [{"name": "id", "type": "int",
			"default": "1); CREATE TABLE admsec_mark (id int); CREATE TABLE admsec_t2 (x int"}]}`,
	},
	{
		name: "column type in a new column", path: "/admin/databases/postgres/tables/admsec_c/columns",
		body:  `{"name": "y", "type": "int; CREATE TABLE admsec_mark (id int)"}`,
		setup: []string{`CREATE TABLE admsec_c (id int)`},
	},
	{
		name: "function return type", path: "/admin/databases/postgres/functions",
		body: `{"name": "admsec_f", "definition": "SELECT 1",
			"returns": "int LANGUAGE sql AS $x$ SELECT 1 $x$; CREATE TABLE admsec_mark (id int); CREATE FUNCTION admsec_f2() RETURNS int"}`,
	},
	{
		name: "policy expression", path: "/admin/databases/postgres/tables/admsec_p/policies",
		body:  `{"name": "p", "using": "true); CREATE TABLE admsec_mark (id int); CREATE POLICY p2 ON admsec_p USING (true"}`,
		setup: []string{`CREATE TABLE admsec_p (id int)`},
	},
}

const adminSecurityCleanup = `DROP TABLE IF EXISTS admsec_mark, admsec_t, admsec_t2, admsec_c, admsec_p;
	DROP VIEW IF EXISTS admsec_v; DROP FUNCTION IF EXISTS admsec_f(), admsec_f2()`

func adminSecurityPost(t *testing.T, addr, path, body, token string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest("POST", "http://"+addr+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := slowClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestAdminDDLRunsOneStatement(t *testing.T) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, superuserURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)

	addr := "localhost:8190"
	s := newTestServer(t, map[string]any{
		"Address":           addr,
		"AllowAnon":         true,
		"Database.AnonRole": "postgres",
		"EnableAdminRoute":  true,
		"AdminRoles":        []string{"*"},
	})
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()

	for _, c := range adminMultiStatementCases {
		t.Run(c.name, func(t *testing.T) {
			adminTxExec(t, conn, adminSecurityCleanup)
			adminTxExec(t, conn, c.setup...)
			defer adminTxExec(t, conn, adminSecurityCleanup)

			status, body := adminSecurityPost(t, addr, c.path, c.body, "")
			if status != http.StatusBadRequest {
				t.Errorf("status %d, want 400: %s", status, body)
			}
			var marked bool
			if err := conn.QueryRow(ctx, `SELECT to_regclass('public.admsec_mark') IS NOT NULL`).Scan(&marked); err != nil {
				t.Fatal(err)
			}
			if marked {
				t.Error("the second statement ran")
			}
		})
	}

	// positive control: one statement, with a trailing semicolon, still runs
	t.Run("one statement", func(t *testing.T) {
		adminTxExec(t, conn, adminSecurityCleanup)
		defer adminTxExec(t, conn, adminSecurityCleanup)
		status, body := adminSecurityPost(t, addr, "/admin/databases/postgres/views",
			`{"name": "admsec_v", "definition": "SELECT 1 AS one;"}`, "")
		if status != http.StatusCreated {
			t.Errorf("status %d, want 201: %s", status, body)
		}
	})
}

// Even one statement can store code that switches role when it runs (a
// function body that calls set_config('role', ...)), so only the roles in
// AdminRoles may use the admin API. "*" admits every role, an empty list none.
func TestAdminRolesGate(t *testing.T) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, superuserURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	adminTxExec(t, conn,
		`DROP ROLE IF EXISTS admsec_admin`, `DROP ROLE IF EXISTS admsec_user`, `DROP ROLE IF EXISTS admsec_anon`,
		`CREATE ROLE admsec_admin NOLOGIN`, `CREATE ROLE admsec_user NOLOGIN`, `CREATE ROLE admsec_anon NOLOGIN`)
	defer adminTxExec(t, conn,
		`DROP ROLE IF EXISTS admsec_admin`, `DROP ROLE IF EXISTS admsec_user`, `DROP ROLE IF EXISTS admsec_anon`)

	const secret = "admin-roles-gate-test-secret-0123456789"
	adminToken, _ := authn.GenerateToken("admsec_admin", secret)
	userToken, _ := authn.GenerateToken("admsec_user", secret)

	cases := []struct {
		roles             []string
		admin, user, anon int
	}{
		{[]string{"admsec_admin"}, 200, 403, 401},
		{[]string{"*"}, 200, 200, 200},
		{[]string{}, 403, 403, 401},
	}
	for i, c := range cases {
		t.Run(fmt.Sprint(c.roles), func(t *testing.T) {
			addr := fmt.Sprintf("localhost:%d", 8191+i)
			s := newPinnedTestServer(t, map[string]any{
				"Address":           addr,
				"AllowAnon":         true,
				"Database.AnonRole": "admsec_anon",
				"JWTSecret":         secret,
				"EnableAdminRoute":  true,
				"AdminRoles":        c.roles,
			})
			done := startTestServer(t, s)
			defer func() {
				s.Shutdown(context.Background())
				<-done
			}()
			get := func(token string) (int, string) {
				req, _ := http.NewRequest("GET", "http://"+addr+"/admin/databases/postgres/schemas", nil)
				if token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}
				resp, err := slowClient.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()
				b, _ := io.ReadAll(resp.Body)
				return resp.StatusCode, string(b)
			}
			for _, r := range []struct {
				who, token string
				want       int
			}{{"admsec_admin", adminToken, c.admin}, {"admsec_user", userToken, c.user}, {"anon", "", c.anon}} {
				if status, body := get(r.token); status != r.want {
					t.Errorf("%s: status %d, want %d: %s", r.who, status, r.want, body)
				}
			}
			// a refused role cannot write either
			if c.user != 200 {
				status, body := adminSecurityPost(t, addr, "/admin/databases/postgres/schemas",
					`{"name": "admsec_schema"}`, userToken)
				if status != 403 {
					t.Errorf("POST as admsec_user: status %d, want 403: %s", status, body)
				}
			}
		})
	}
}
