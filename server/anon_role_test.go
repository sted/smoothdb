package server

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/test"
)

// anonProbeSetup creates, on the server's main (postgres) database, a
// non-superuser anon role and a probe function that reports the role the
// request runs as (current_user) and the role carried in the request.jwt.claims
// GUC. The connection is the pool's own (the authenticator, a superuser here),
// so the DDL needs no extra grants; the function is EXECUTE-able by PUBLIC by
// default, so any switched-to role can call it. anon_probe_claims returns the
// whole GUC, and anon_probe_private is a table no probe role may read, as
// authors_only is for the anonymous role in PostgREST's fixtures.
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
		"DROP FUNCTION IF EXISTS anon_probe_claims()",
		"DROP TABLE IF EXISTS anon_probe_private",
		"DROP ROLE IF EXISTS anon_probe_role",
		"DROP ROLE IF EXISTS anon_probe_user",
		"CREATE ROLE anon_probe_role NOLOGIN",
		"CREATE ROLE anon_probe_user NOLOGIN",
		`CREATE FUNCTION anon_probe_whoami() RETURNS TABLE(current_role_name text, claims_role text)
			LANGUAGE sql AS $$
			SELECT current_user::text,
			       nullif(current_setting('request.jwt.claims', true), '')::json->>'role'
			$$`,
		`CREATE FUNCTION anon_probe_claims() RETURNS json
			LANGUAGE sql AS $$
			SELECT nullif(current_setting('request.jwt.claims', true), '')::json
			$$`,
		"CREATE TABLE anon_probe_private (secret text)",
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

// postgrestJWTSecret is the secret of PostgREST's test suite: the AuthSpec
// tokens quoted below are signed with it, and are sent here as they are.
const postgrestJWTSecret = "reallyreallyreallyreallyverysafe"

// The AuthSpec tokens without a role claim. token body: {"id":"jdoe"}
const noRoleToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6Impkb2UifQ.RVlZDaSyKbFPvxUf3V_NQXybfRB4dlBIkAUQXVXLUAI"

// token body: {}
const noClaimsToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.CUIP5V9thWsGGFsFyGijSZf1fJMfarLHI9CEJL-TGNk"

// signClaims signs claims GenerateToken cannot express, such as an empty role.
func signClaims(t *testing.T, claims jwt.MapClaims, secret string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func bearer(base, token string) test.Config {
	return test.Config{BaseUrl: base, CommonHeaders: test.Headers{"Authorization": {"Bearer " + token}}}
}

// PostgREST runs a token without a role claim as the anonymous role
// (src/library/PostgREST/Auth/Jwt.hs, parseClaims: the role is the claim <|>
// db-anon-role) and sets request.jwt.claims to the token's claims with "role"
// inserted (Query/PreQuery.hs). smoothdb took the token's missing role as an
// empty one, and PrepareConnection skips the SET ROLE and the claims GUC on an
// empty role: the request ran as the connecting (authenticator) role with the
// pool's own privileges, here a superuser that reads any table. Every session
// mode is covered, and the probe is sent twice, so that the second request
// is served from the session the first one built.
func TestTokenWithoutRoleRunsAsAnonRole(t *testing.T) {
	modes := []struct{ mode, port string }{{"none", "8102"}, {"role", "8103"}, {"claims", "8104"}}
	for _, m := range modes {
		t.Run(m.mode, func(t *testing.T) {
			s := newTestServer(t, map[string]any{
				"Address":           "localhost:" + m.port,
				"AllowAnon":         true,
				"Database.AnonRole": "anon_probe_role",
				"JWTSecret":         postgrestJWTSecret,
				"SessionMode":       m.mode,
				"EnableAdminRoute":  false,
			})
			anonProbeSetup(t, s)
			done := startTestServer(t, s)
			defer func() {
				s.Shutdown(context.Background())
				<-done
			}()
			base := "http://localhost:" + m.port + "/api/postgres"
			denied := `{"subsystem":"database","message":"permission denied for table anon_probe_private","code":"42501","hint":"","details":null,"position":0}`

			test.Execute(t, bearer(base, noRoleToken), []test.Test{
				{
					Description: "a token without a role claim runs as the anon role",
					Method:      "GET",
					Query:       "/rpc/anon_probe_whoami",
					Expected:    `[{"current_role_name":"anon_probe_role","claims_role":"anon_probe_role"}]`,
					Status:      200,
				},
				// it "hides tables from users with JWT that contain no claims about role" $ do
				//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6Impkb2UifQ.RVlZDaSyKbFPvxUf3V_NQXybfRB4dlBIkAUQXVXLUAI"
				//   request methodGet "/authors_only" [auth] ""
				//     `shouldRespondWith` 401
				{
					Description: "hides tables from users with JWT that contain no claims about role",
					Method:      "GET",
					Query:       "/anon_probe_private",
					Expected:    denied,
					Status:      401,
				},
				{
					Description: "a token without a role claim runs as the anon role, from its session",
					Method:      "GET",
					Query:       "/rpc/anon_probe_whoami",
					Expected:    `[{"current_role_name":"anon_probe_role","claims_role":"anon_probe_role"}]`,
					Status:      200,
				},
				{
					Description: "the claims GUC is the token's claims with the anon role inserted",
					Method:      "GET",
					Query:       "/rpc/anon_probe_claims",
					Expected:    `{"id":"jdoe","role":"anon_probe_role"}`,
					Status:      200,
				},
			})

			test.Execute(t, bearer(base, noClaimsToken), []test.Test{
				{
					Description: "a token with no claims runs as the anon role",
					Method:      "GET",
					Query:       "/rpc/anon_probe_whoami",
					Expected:    `[{"current_role_name":"anon_probe_role","claims_role":"anon_probe_role"}]`,
					Status:      200,
				},
				// it "should fail when jwt contains no claims" $ do
				//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.CUIP5V9thWsGGFsFyGijSZf1fJMfarLHI9CEJL-TGNk"
				//   request methodGet "/authors_only" [auth] ""
				//     `shouldRespondWith` 401
				{
					Description: "should fail when jwt contains no claims",
					Method:      "GET",
					Query:       "/anon_probe_private",
					Expected:    denied,
					Status:      401,
				},
			})

			// An empty role claim is not a missing one: PostgREST takes it as the
			// role, and PostgreSQL refuses it with 22023 `role "" does not exist`,
			// which both map to 401.
			emptyRole := signClaims(t, jwt.MapClaims{"role": "", "id": "jdoe"}, postgrestJWTSecret)
			test.Execute(t, bearer(base, emptyRole), []test.Test{
				{
					Description: "a token with an empty role claim is refused",
					Method:      "GET",
					Query:       "/rpc/anon_probe_whoami",
					Expected:    `{"error":"role \"\" does not exist"}`,
					Status:      401,
				},
			})

			// positive control: a token with a role runs as that role with its
			// own claims, untouched by the fallback.
			withRole := signClaims(t, jwt.MapClaims{"role": "anon_probe_user", "id": "jdoe"}, postgrestJWTSecret)
			test.Execute(t, bearer(base, withRole), []test.Test{
				{
					Description: "a token with a role runs as that role",
					Method:      "GET",
					Query:       "/rpc/anon_probe_whoami",
					Expected:    `[{"current_role_name":"anon_probe_user","claims_role":"anon_probe_user"}]`,
					Status:      200,
				},
				{
					Description: "a token with a role keeps its claims",
					Method:      "GET",
					Query:       "/rpc/anon_probe_claims",
					Expected:    `{"id":"jdoe","role":"anon_probe_user"}`,
					Status:      200,
				},
			})
		})
	}
}

// With anonymous access disabled, PostgREST refuses a token without a role
// claim as it refuses a request without a token (Auth/Jwt.hs, parseClaims:
// JwtTokenRequired, 401 PGRST302 "Anonymous access is disabled"). In smoothdb
// anonymous access is off when AllowAnon is false or Database.AnonRole is
// empty, and the token ran as the authenticator in both cases.
func TestTokenWithoutRoleIsRefusedWithoutAnonRole(t *testing.T) {
	cases := []struct {
		name      string
		port      string
		allowAnon bool
		anonRole  string
	}{
		{"empty anon role", "8105", true, ""},
		{"anonymous access off", "8106", false, "anon_probe_role"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newTestServer(t, map[string]any{
				"Address":           "localhost:" + c.port,
				"AllowAnon":         c.allowAnon,
				"Database.AnonRole": c.anonRole,
				"JWTSecret":         postgrestJWTSecret,
				"SessionMode":       "role",
				"EnableAdminRoute":  false,
			})
			anonProbeSetup(t, s)
			done := startTestServer(t, s)
			defer func() {
				s.Shutdown(context.Background())
				<-done
			}()
			base := "http://localhost:" + c.port + "/api/postgres"

			for _, token := range []string{noRoleToken, noClaimsToken} {
				test.Execute(t, bearer(base, token), []test.Test{
					{
						Description: "a token without a role claim is refused",
						Method:      "GET",
						Query:       "/rpc/anon_probe_whoami",
						Expected:    `{"error":"Anonymous access is disabled"}`,
						Status:      401,
					},
				})
			}

			// positive control: a token with a role is not refused.
			token, err := authn.GenerateToken("anon_probe_user", s.JWTSecret())
			if err != nil {
				t.Fatal(err)
			}
			test.Execute(t, bearer(base, token), []test.Test{
				{
					Description: "a token with a role runs as that role",
					Method:      "GET",
					Query:       "/rpc/anon_probe_whoami",
					Expected:    `[{"current_role_name":"anon_probe_user","claims_role":"anon_probe_user"}]`,
					Status:      200,
				},
			})
		})
	}
}
