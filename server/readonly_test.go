package server

import (
	"context"
	"testing"

	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/test"
)

// readOnlyProbeSetup builds, on the server's main (postgres) database, the
// ro_session schema: a table, a VOLATILE writer and a STABLE function that
// writes through it (PostgreSQL accepts the marker; the READ ONLY transaction
// is what catches it).
func readOnlyProbeSetup(t *testing.T, s *Server) {
	t.Helper()
	ctx := context.Background()
	conn, err := s.DBE.AcquireConnection(ctx)
	if err != nil {
		t.Fatal(err)
	}
	stmts := []string{
		"DROP SCHEMA IF EXISTS ro_session CASCADE",
		"CREATE SCHEMA ro_session",
		"CREATE TABLE ro_session.audit (note text)",
		`CREATE FUNCTION ro_session.write_audit(note text) RETURNS text
			LANGUAGE sql VOLATILE AS $$
			INSERT INTO ro_session.audit (note) VALUES ($1) RETURNING note
			$$`,
		`CREATE FUNCTION ro_session.stable_liar(note text) RETURNS text
			LANGUAGE sql STABLE AS $$ SELECT ro_session.write_audit($1) $$`,
	}
	for _, q := range stmts {
		if _, err := conn.Exec(ctx, q); err != nil {
			conn.Release()
			t.Fatalf("probe setup %q: %v", q, err)
		}
	}
	// the reload takes its own connection: give back the pool's only one first
	conn.Release()
	db, err := s.DBE.GetMainDatabase(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ReloadSchemaCache(ctx); err != nil {
		t.Fatal(err)
	}
}

// GET and HEAD run READ ONLY, POST does not — and in "role" session mode the
// prepared connection stays attached to the session between requests, so the
// access mode must follow each request on the SAME connection: a GET followed
// by a POST must write, a POST followed by a GET must be read-only again, and
// a POST on a STABLE function (run READ ONLY like a GET) must not leave the
// connection read-only for the next write. The two transaction modes are two
// different mechanisms (a session setting with TransactionMode "none", the
// transaction's own access mode otherwise), so the same sequence runs under
// both.
func TestReadOnlyFollowsEachRequestOnARetainedConnection(t *testing.T) {
	cases := []struct {
		txMode string
		port   string
	}{
		{"none", "8089"},
		{"commit", "8090"},
	}
	for _, c := range cases {
		t.Run(c.txMode, func(t *testing.T) {
			s := newSingleConnectionServer(t, map[string]any{
				"Address":                  "localhost:" + c.port,
				"AllowAnon":                false,
				"JWTSecret":                "read-only-test-secret",
				"SessionMode":              "role",
				"EnableAdminRoute":         false,
				"Database.TransactionMode": c.txMode,
			})
			readOnlyProbeSetup(t, s)
			done := startTestServer(t, s)
			defer func() {
				s.Shutdown(context.Background())
				<-done
			}()

			token, err := authn.GenerateToken("postgres", s.JWTSecret())
			if err != nil {
				t.Fatal(err)
			}
			// one token: every request below lands on the same session
			cfg := test.Config{
				BaseUrl: "http://localhost:" + c.port + "/api/postgres",
				CommonHeaders: test.Headers{
					"Authorization":   {token},
					"Accept-Profile":  {"ro_session"},
					"Content-Profile": {"ro_session"},
				},
			}
			test.Execute(t, cfg, []test.Test{
				{
					Description:     "GET on a function that writes is refused",
					Method:          "GET",
					Query:           "/rpc/write_audit?note=g1",
					ExpectedHeaders: map[string]string{"Allow": "POST"},
					Status:          405,
				},
				{
					Description: "POST on the same function writes on the same connection",
					Method:      "POST",
					Query:       "/rpc/write_audit",
					Body:        `{"note": "p1"}`,
					Expected:    `"p1"`,
					Status:      200,
				},
				{
					Description: "GET on the table after the write",
					Method:      "GET",
					Query:       "/audit?select=note",
					Expected:    `[{"note":"p1"}]`,
					Status:      200,
				},
				{
					Description: "POST on the table after the read",
					Method:      "POST",
					Query:       "/audit",
					Body:        `{"note": "p2"}`,
					Status:      201,
				},
				{
					Description:     "HEAD on a function that writes is refused",
					Method:          "HEAD",
					Query:           "/rpc/write_audit?note=h1",
					ExpectedHeaders: map[string]string{"Allow": "POST"},
					Status:          405,
				},
				{
					Description:     "POST on a STABLE function that writes is refused",
					Method:          "POST",
					Query:           "/rpc/stable_liar",
					Body:            `{"note": "s1"}`,
					ExpectedHeaders: map[string]string{"Allow": "POST"},
					Status:          405,
				},
				{
					Description: "POST on the VOLATILE writer after the refused STABLE call",
					Method:      "POST",
					Query:       "/rpc/write_audit",
					Body:        `{"note": "p3"}`,
					Expected:    `"p3"`,
					Status:      200,
				},
				{
					Description: "only the writes done through POST on VOLATILE functions or tables are there",
					Method:      "GET",
					Query:       "/audit?select=note&order=note",
					Expected:    `[{"note":"p1"},{"note":"p2"},{"note":"p3"}]`,
					Status:      200,
				},
			})
		})
	}
}
