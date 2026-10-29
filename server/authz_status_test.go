package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/test"
)

const superuserURL = "postgresql://postgres:postgres@localhost:5432/postgres"

// authzSetup creates, with a superuser connection of its own, a
// non-superuser authenticator and the roles it may and may not become: a
// superuser can SET ROLE to anything, so a refused role switch needs an
// authenticator that is not one. authz_private is a table no role may read,
// authz_open one authz_user may.
// The objects are dropped when the test ends.
func authzSetup(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, superuserURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	drop := []string{
		"DROP TABLE IF EXISTS authz_private",
		"DROP TABLE IF EXISTS authz_open",
		"DROP ROLE IF EXISTS authz_authn",
		"DROP ROLE IF EXISTS authz_anon",
		"DROP ROLE IF EXISTS authz_user",
		"DROP ROLE IF EXISTS authz_denied",
	}
	stmts := append(drop,
		"CREATE ROLE authz_authn LOGIN PASSWORD 'authz' NOINHERIT",
		"CREATE ROLE authz_anon NOLOGIN",
		"CREATE ROLE authz_user NOLOGIN",
		"CREATE ROLE authz_denied NOLOGIN",
		"GRANT authz_anon, authz_user TO authz_authn",
		"CREATE TABLE authz_private (secret text)",
		"CREATE TABLE authz_open (note text)",
		"GRANT SELECT ON authz_open TO authz_user",
	)
	for _, q := range stmts {
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("authz setup %q: %v", q, err)
		}
	}
	t.Cleanup(func() {
		conn, err := pgx.Connect(ctx, superuserURL)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close(ctx)
		for _, q := range drop {
			if _, err := conn.Exec(ctx, q); err != nil {
				t.Errorf("authz teardown %q: %v", q, err)
			}
		}
	})
}

func newAuthzServer(t *testing.T, port string, config map[string]any) *Server {
	t.Helper()
	base := map[string]any{
		"Address":           "localhost:" + port,
		"Database.URL":      "postgresql://authz_authn:authz@localhost:5432/postgres",
		"AllowAnon":         true,
		"Database.AnonRole": "authz_anon",
		"JWTSecret":         postgrestJWTSecret,
		"EnableAdminRoute":  false,
	}
	for k, v := range config {
		base[k] = v
	}
	return newTestServer(t, base)
}

// The status of an authorization outcome follows PostgREST (Error.hs,
// mapSQLtoHTTP and the PgError headers; App.hs: authenticated means the role is
// not the anon role). Verified against PostgREST 16.4 with a non-superuser
// authenticator:
//
//	anonymous, 42501           -> 401, WWW-Authenticate: Bearer
//	authenticated, 42501       -> 403, no challenge
//	token naming the anon role -> 401, WWW-Authenticate: Bearer
//	refused role switch        -> 403 (42501 "permission denied to set role")
//	role that does not exist   -> 401, WWW-Authenticate: Bearer (22023)
//
// The errors raised while the middleware prepares the connection are reported
// like every other database error: the SmoothError body, the status of
// pgErrorStatus. They used to be 500 with {"error": ...}.
func TestAuthorizationStatusFollowsPostgREST(t *testing.T) {
	authzSetup(t)
	modes := []struct{ mode, port string }{{"none", "8110"}, {"role", "8111"}}
	for _, m := range modes {
		t.Run(m.mode, func(t *testing.T) {
			s := newAuthzServer(t, m.port, map[string]any{"SessionMode": m.mode})
			done := startTestServer(t, s)
			defer func() {
				s.Shutdown(context.Background())
				<-done
			}()
			base := "http://localhost:" + m.port + "/api/postgres"
			denied := `{"subsystem":"database","message":"permission denied for table authz_private","code":"42501","hint":"","details":null,"position":0}`
			user := signClaims(t, map[string]any{"role": "authz_user"}, postgrestJWTSecret)
			anonToken := signClaims(t, map[string]any{"role": "authz_anon"}, postgrestJWTSecret)
			refused := signClaims(t, map[string]any{"role": "authz_denied"}, postgrestJWTSecret)
			missing := signClaims(t, map[string]any{"role": "authz_nobody"}, postgrestJWTSecret)

			test.Execute(t, test.Config{BaseUrl: base}, []test.Test{
				{
					Description:     "anonymous: insufficient privilege is 401 with the challenge",
					Query:           "/authz_private",
					Expected:        denied,
					ExpectedHeaders: map[string]string{"WWW-Authenticate": "Bearer"},
					Status:          401,
				},
			})
			test.Execute(t, bearer(base, anonToken), []test.Test{
				{
					Description:     "a token naming the anon role is anonymous",
					Query:           "/authz_private",
					Expected:        denied,
					ExpectedHeaders: map[string]string{"WWW-Authenticate": "Bearer"},
					Status:          401,
				},
			})
			test.Execute(t, bearer(base, user), []test.Test{
				{
					Description:     "authenticated: insufficient privilege is 403 without a challenge",
					Query:           "/authz_private",
					Expected:        denied,
					ExpectedHeaders: map[string]string{"WWW-Authenticate": ""},
					Status:          403,
				},
			})
			test.Execute(t, bearer(base, refused), []test.Test{
				{
					Description:     "a role the authenticator cannot become is 403",
					Query:           "/authz_private",
					Expected:        `{"subsystem":"database","message":"permission denied to set role \"authz_denied\"","code":"42501","hint":"","details":null,"position":0}`,
					ExpectedHeaders: map[string]string{"WWW-Authenticate": ""},
					Status:          403,
				},
			})
			test.Execute(t, bearer(base, missing), []test.Test{
				{
					Description:     "a role that does not exist is 401 with the challenge",
					Query:           "/authz_private",
					Expected:        `{"subsystem":"database","message":"role \"authz_nobody\" does not exist","code":"22023","hint":"","details":null,"position":0}`,
					ExpectedHeaders: map[string]string{"WWW-Authenticate": "Bearer"},
					Status:          401,
				},
			})
			// positive control: the authenticator can become authz_user, and
			// the request runs.
			test.Execute(t, bearer(base, user), []test.Test{
				{
					Description: "a role the authenticator can become runs",
					Query:       "/authz_open",
					Expected:    `[]`,
					Status:      200,
				},
			})
		})
	}
}

// PostgREST answers a refused token with the RFC 6750 challenge carrying the
// message (Error.hs:686, invalidTokenHeader: `show` quotes it), and a request
// that must authenticate with a bare one (requiredTokenHeader). Both bodies
// have the SmoothError shape and the PostgREST code.
func TestAuthRefusalsCarryTheChallenge(t *testing.T) {
	cases := []struct {
		name, port string
		allowAnon  bool
	}{
		{"anonymous access off", "8112", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newTestServer(t, map[string]any{
				"Address":           "localhost:" + c.port,
				"AllowAnon":         c.allowAnon,
				"Database.AnonRole": "",
				"JWTSecret":         postgrestJWTSecret,
				"SessionMode":       "role",
				"EnableAdminRoute":  false,
				"JQ.Enabled":        true,
			})
			done := startTestServer(t, s)
			defer func() {
				s.Shutdown(context.Background())
				<-done
			}()
			base := "http://localhost:" + c.port

			// NoAnonSpec: it "responds with error when user does not attempt auth"
			//   get "/items" `shouldRespondWith` [json|{"hint": null, "details": null,
			//     "code": "PGRST302", "message":"Anonymous access is disabled"}|]
			//     { matchStatus = 401, matchHeaders = ["WWW-Authenticate" <:> "Bearer"] }
			for _, path := range []string{"/api/postgres/items", "/jq"} {
				test.Execute(t, test.Config{BaseUrl: base}, []test.Test{
					{
						Description:     "a request without a token is refused with the bare challenge " + path,
						Method:          "POST",
						Query:           path,
						Body:            `{}`,
						Expected:        `{"subsystem":"auth","message":"Anonymous access is disabled","code":"PGRST302","hint":"","details":null,"position":0}`,
						ExpectedHeaders: map[string]string{"WWW-Authenticate": "Bearer"},
						Status:          401,
					},
				})
			}

			// AuthSpec: it "hides tables from users with invalid JWT"
			//   ... `shouldRespondWith` [json| {"message":"Expected 3 parts in JWT; got 2","code":"PGRST301",...} |]
			//     { matchStatus = 401, matchHeaders = [ "WWW-Authenticate" <:>
			//         "Bearer error=\"invalid_token\", error_description=\"Expected 3 parts in JWT; got 2\"" ] }
			// AuthSpec: it "fails with an expired token"
			//   ... `shouldRespondWith` [json| {"message":"JWT expired","code":"PGRST303",...} |]
			//     { matchStatus = 401, matchHeaders = [ "WWW-Authenticate" <:>
			//         "Bearer error=\"invalid_token\", error_description=\"JWT expired\"" ] }
			// The messages are the verifier's own; the challenge must carry
			// whichever message the body has.
			expired := signClaims(t, map[string]any{"role": "postgres", "exp": 1000}, postgrestJWTSecret)
			tokens := []struct{ name, token, code string }{
				{"invalid", "ey9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.y4vZuu1dDdwAl0-S00MCRWRYMlJ5YAMSir6Es6WtWx0", "PGRST301"},
				{"expired", expired, "PGRST303"},
			}
			for _, tk := range tokens {
				for _, path := range []string{"/api/postgres/items", "/jq"} {
					req, _ := http.NewRequest("POST", base+path, strings.NewReader(`{}`))
					req.Header.Set("Authorization", "Bearer "+tk.token)
					resp, err := probeClient.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					if resp.StatusCode != http.StatusUnauthorized {
						t.Errorf("%s token on %s: expected 401, got %d %s", tk.name, path, resp.StatusCode, body)
						continue
					}
					var e struct{ Subsystem, Message, Code string }
					if err := json.Unmarshal(body, &e); err != nil {
						t.Errorf("%s token on %s: body %s: %v", tk.name, path, body, err)
						continue
					}
					if e.Subsystem != "auth" || e.Code != tk.code || e.Message == "" {
						t.Errorf("%s token on %s: expected an auth error with code %s, got %s", tk.name, path, tk.code, body)
					}
					want := `Bearer error="invalid_token", error_description=` + strconv.Quote(e.Message)
					if got := resp.Header.Get("WWW-Authenticate"); got != want {
						t.Errorf("%s token on %s: WWW-Authenticate\n\twant %s\n\tgot  %s", tk.name, path, want, got)
					}
				}
			}

			// positive control: a valid token passes the same gate.
			token, err := authn.GenerateToken("postgres", s.JWTSecret())
			if err != nil {
				t.Fatal(err)
			}
			test.Execute(t, bearer(base, token), []test.Test{
				{
					Description: "a valid token passes",
					Query:       "/api/postgres",
					Status:      200,
				},
			})
		})
	}
}
