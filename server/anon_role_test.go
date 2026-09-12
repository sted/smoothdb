package server

import (
	"context"
	"testing"

	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/test"
)

// anonProbeSetup creates, on the server's main (postgres) database, a
// non-superuser anon role and a probe function that reports the role the
// request runs as (current_user) and the role carried in the request.jwt.claims
// GUC. The connection is the pool's own (the authenticator, a superuser here),
// so the DDL needs no extra grants; the function is EXECUTE-able by PUBLIC by
// default, so any switched-to role can call it.
func anonProbeSetup(t *testing.T, s *Server) {
	t.Helper()
	ctx := context.Background()
	conn, err := s.DBE.AcquireConnection(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	stmts := []string{
		"DROP FUNCTION IF EXISTS anon_probe_whoami()",
		"DROP ROLE IF EXISTS anon_probe_role",
		"DROP ROLE IF EXISTS anon_probe_user",
		"CREATE ROLE anon_probe_role NOLOGIN",
		"CREATE ROLE anon_probe_user NOLOGIN",
		`CREATE FUNCTION anon_probe_whoami() RETURNS TABLE(current_role_name text, claims_role text)
			LANGUAGE sql AS $$
			SELECT current_user::text,
			       nullif(current_setting('request.jwt.claims', true), '')::json->>'role'
			$$`,
	}
	for _, q := range stmts {
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("probe setup %q: %v", q, err)
		}
	}
}

// PostgREST refuses anonymous access when no anon role is configured
// (test/spec/Feature/Auth/NoAnonSpec.hs, "responds with error when user does
// not attempt auth" -> 401 PGRST302 "Anonymous access is disabled"). smoothdb
// with AllowAnon:true but an empty AnonRole used to run the request as the
// connecting (authenticator) role instead: the anonymous Claims carried an
// empty role and PrepareConnection skips the SET ROLE (and the claims GUC) on
// an empty role, so the request ran with the pool's own privileges — here a
// superuser.
func TestAnonymousWithEmptyAnonRoleIsRefused(t *testing.T) {
	s := newTestServer(t, map[string]any{
		"Address":           "localhost:8086",
		"AllowAnon":         true,
		"Database.AnonRole": "",
		"JWTSecret":         "anon-role-test-secret",
		"SessionMode":       "none",
		"EnableAdminRoute":  false,
	})
	anonProbeSetup(t, s)
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()

	anon := test.Config{BaseUrl: "http://localhost:8086/api/postgres"}
	test.Execute(t, anon, []test.Test{
		{
			Description: "anonymous request with an empty anon role is refused",
			Method:      "GET",
			Query:       "/rpc/anon_probe_whoami",
			Expected:    `{"error":"Anonymous access is disabled"}`,
			Status:      401,
		},
	})

	// positive control: an authenticated request is not blocked by the anon
	// gate; it switches to its own role and carries its own claims.
	token, err := authn.GenerateToken("postgres", s.JWTSecret())
	if err != nil {
		t.Fatal(err)
	}
	authed := test.Config{
		BaseUrl:       "http://localhost:8086/api/postgres",
		CommonHeaders: test.Headers{"Authorization": {token}},
	}
	test.Execute(t, authed, []test.Test{
		{
			Description: "authenticated request is not blocked by the anon gate",
			Method:      "GET",
			Query:       "/rpc/anon_probe_whoami",
			Expected:    `[{"current_role_name":"postgres","claims_role":"postgres"}]`,
			Status:      200,
		},
	})
}

// With an anon role configured, an anonymous request runs as that role and
// carries the claims GUC, exactly as PostgREST sets it for every request
// (src/library/PostgREST/Query/PreQuery.hs: request.jwt.claims = the claims
// with "role" inserted, i.e. {"role":"<anon>"} for an anonymous request). The
// role switch already happened for a non-empty AnonRole, but the anonymous
// Claims left RawClaims empty, so request.jwt.claims was never set and any RLS
// policy keying off it evaluated with no claims. current_role_name asserts the
// request never runs as the authenticator; claims_role asserts the GUC carries
// the role.
func TestAnonymousRunsAsConfiguredRoleWithClaims(t *testing.T) {
	s := newTestServer(t, map[string]any{
		"Address":           "localhost:8087",
		"AllowAnon":         true,
		"Database.AnonRole": "anon_probe_role",
		"JWTSecret":         "anon-role-test-secret",
		"SessionMode":       "none",
		"EnableAdminRoute":  false,
	})
	anonProbeSetup(t, s)
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()

	anon := test.Config{BaseUrl: "http://localhost:8087/api/postgres"}
	test.Execute(t, anon, []test.Test{
		{
			Description: "anonymous request runs as the anon role and sets the claims GUC",
			Method:      "GET",
			Query:       "/rpc/anon_probe_whoami",
			Expected:    `[{"current_role_name":"anon_probe_role","claims_role":"anon_probe_role"}]`,
			Status:      200,
		},
	})

	// positive control: an authenticated request switches to its own role and
	// carries its own claims — this path was never broken.
	token, err := authn.GenerateToken("anon_probe_user", s.JWTSecret())
	if err != nil {
		t.Fatal(err)
	}
	authed := test.Config{
		BaseUrl:       "http://localhost:8087/api/postgres",
		CommonHeaders: test.Headers{"Authorization": {token}},
	}
	test.Execute(t, authed, []test.Test{
		{
			Description: "authenticated request switches to its own role with claims",
			Method:      "GET",
			Query:       "/rpc/anon_probe_whoami",
			Expected:    `[{"current_role_name":"anon_probe_user","claims_role":"anon_probe_user"}]`,
			Status:      200,
		},
	})
}
