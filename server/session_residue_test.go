package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sted/smoothdb/database"
)

// residueSetup adds to authzSetup the functions a client calls to leave its
// mark on the session of its connection: an invoker function may switch the
// role (to one the authenticator may become, or back to the authenticator
// itself), rewrite the claims or, with a superuser authenticator, change the
// session authorization, all session-level, past the end of its request.
func residueSetup(t *testing.T) {
	t.Helper()
	authzSetup(t)
	superuserSetup(t, []string{
		"DROP FUNCTION IF EXISTS authz_whoami()",
		"DROP FUNCTION IF EXISTS authz_poison(text, text)",
		"DROP FUNCTION IF EXISTS authz_become(text)",
		"DROP FUNCTION IF EXISTS authz_ro()",
	}, []string{
		`CREATE FUNCTION authz_whoami() RETURNS json LANGUAGE sql AS $$
			SELECT json_build_object('user', current_user,
				'sub', coalesce(current_setting('request.jwt.claims', true)::json->>'sub', ''))
			$$`,
		`CREATE FUNCTION authz_poison(r text, c text) RETURNS text LANGUAGE sql AS $$
			SELECT set_config('role', r, false) || set_config('request.jwt.claims', c, false)
			$$`,
		`CREATE FUNCTION authz_become(u text) RETURNS text LANGUAGE sql AS $$
			SELECT set_config('session_authorization', u, false)
			$$`,
		`CREATE FUNCTION authz_ro() RETURNS text LANGUAGE sql AS $$
			SELECT set_config('default_transaction_read_only', 'on', false)
			$$`,
	})
}

func whoami(t *testing.T, base, token string) (user, sub string) {
	t.Helper()
	var who struct{ User, Sub string }
	if err := json.Unmarshal([]byte(getBody(t, base+"/rpc/authz_whoami", token)), &who); err != nil {
		t.Fatal(err)
	}
	return who.User, who.Sub
}

// poison calls authz_poison (or authz_become) and checks it ran.
func poison(t *testing.T, url, token string) {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := slowClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("%s: %d %s", url, resp.StatusCode, body)
	}
}

// identityOf reports what a connection taken from the pool outside any
// request (as the schema reload or ContextWithDb take one) runs as.
func identityOf(t *testing.T, s *Server) (session, current, claims string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := s.DBE.AcquireConnection(ctx)
	if err != nil {
		t.Fatalf("no connection in the pool: %v", err)
	}
	defer conn.Release()
	err = conn.QueryRow(ctx, `SELECT session_user::text, current_user::text,
		coalesce(current_setting('request.jwt.claims', true), '')`).Scan(&session, &current, &claims)
	if err != nil {
		t.Fatal(err)
	}
	return
}

// In "role" mode a session keeps its connection, switched to its role, from
// one request to the next, and hands it to a waiting request of its token.
// A request may switch the role of the session itself (back to the
// authenticator, or to another role the authenticator may become) or rewrite
// its claims, and the next request of the token on that connection ran with
// them. Every request must run as its token's role with its token's claims.
func TestRetainedConnectionRunsEachRequestAsItsToken(t *testing.T) {
	residueSetup(t)
	s := newAuthzServer(t, "8166", map[string]any{
		"SessionMode":                 "role",
		"Database.MinPoolConnections": int32(1),
		"Database.MaxPoolConnections": int32(1),
	})
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()
	base := "http://localhost:8166/api/postgres"
	user := signClaims(t, map[string]any{"role": "authz_user", "sub": "me"}, postgrestJWTSecret)
	for _, attack := range []string{
		"/rpc/authz_poison?r=none&c=%7B%7D",                              // back to the authenticator
		"/rpc/authz_poison?r=authz_anon&c=%7B%22sub%22%3A%22other%22%7D", // another role, other claims
	} {
		// twice: the second time from the connection's statement cache
		for i := 0; i < 2; i++ {
			if u, sub := whoami(t, base, user); u != "authz_user" || sub != "me" {
				t.Fatalf("before %s: %s %s", attack, u, sub)
			}
			poison(t, base+attack, user)
			if u, sub := whoami(t, base, user); u != "authz_user" || sub != "me" {
				t.Errorf("after %s (%d): the next request of the token ran as %s with sub %q", attack, i, u, sub)
			}
		}
	}
}

// Without sessions a request's variables are transaction-local, and the
// connection goes back to the pool without a reset when the request set
// nothing session-level itself. But a function may: its role and claims then
// stay on the connection, and whoever takes it outside a request (the schema
// reload, the activation of a database, a library caller) ran as them. A
// connection that served a request is cleaned before such a user gets it.
func TestPooledConnectionKeepsNothingAFunctionSet(t *testing.T) {
	residueSetup(t)
	cases := []struct{ session, tx, port string }{
		{"none", "none", "8167"},
		{"none", "commit", "8168"},
	}
	for _, c := range cases {
		t.Run(c.session+"/"+c.tx, func(t *testing.T) {
			s := newAuthzServer(t, c.port, map[string]any{
				"SessionMode":                 c.session,
				"Database.TransactionMode":    c.tx,
				"Database.MinPoolConnections": int32(1),
				"Database.MaxPoolConnections": int32(1),
			})
			done := startTestServer(t, s)
			defer func() {
				s.Shutdown(context.Background())
				<-done
			}()
			base := "http://localhost:" + c.port + "/api/postgres"
			user := signClaims(t, map[string]any{"role": "authz_user", "sub": "me"}, postgrestJWTSecret)
			// twice: the second time from the connection's statement cache
			for i := 0; i < 2; i++ {
				poison(t, base+"/rpc/authz_poison?r=authz_anon&c=%7B%22sub%22%3A%22other%22%7D", user)
				if session, current, claims := identityOf(t, s); session != "authz_authn" || current != "authz_authn" || claims != "" {
					t.Errorf("after a function set role and claims (%d): the pool hands out %s/%s with claims %q", i, session, current, claims)
				}
			}
			// the schema reload, which takes its connection the same way
			poison(t, base+"/rpc/authz_poison?r=authz_anon&c=%7B%22sub%22%3A%22other%22%7D", user)
			db, err := s.DBE.GetMainDatabase(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if err := db.ReloadSchemaCache(context.Background()); err != nil {
				t.Errorf("schema reload after a function set the role: %v", err)
			}
		})
	}
}

// With a superuser authenticator (which smoothdb warns against) a function
// can change the session authorization itself: RESET ROLE then goes back to
// the new session user, not to the authenticator. The connection must go
// back to the pool as the authenticator.
func TestPooledConnectionKeepsNoSessionAuthorization(t *testing.T) {
	residueSetup(t)
	cases := []struct{ session, port string }{{"none", "8169"}, {"role", "8170"}}
	for _, c := range cases {
		t.Run(c.session, func(t *testing.T) {
			s := newTestServer(t, map[string]any{
				"Address":                     "localhost:" + c.port,
				"AllowAnon":                   true,
				"Database.AnonRole":           "authz_anon",
				"JWTSecret":                   postgrestJWTSecret,
				"EnableAdminRoute":            false,
				"SessionMode":                 c.session,
				"Database.MinPoolConnections": int32(1),
				"Database.MaxPoolConnections": int32(1),
			})
			done := startTestServer(t, s)
			defer func() {
				s.Shutdown(context.Background())
				<-done
			}()
			base := "http://localhost:" + c.port + "/api/postgres"
			for i := 0; i < 2; i++ {
				poison(t, base+"/rpc/authz_become?u=authz_user", "")
				// "role" mode: the connection is the session's until the
				// watcher idles it out (about two seconds)
				deadline := time.Now().Add(5 * time.Second)
				var session, current string
				for {
					session, current, _ = identityOf(t, s)
					if session == "postgres" && current == "postgres" || time.Now().After(deadline) {
						break
					}
					time.Sleep(200 * time.Millisecond)
				}
				if session != "postgres" || current != "postgres" {
					t.Errorf("after a function changed the session authorization (%d): the pool hands out %s/%s", i, session, current)
				}
				if u, _ := whoami(t, base, ""); u != "authz_anon" {
					t.Errorf("the next anonymous request ran as %s", u)
				}
			}
		})
	}
}

// The cleanup of a connection runs when the request ends. It ran inside the
// request's transaction, before its COMMIT or ROLLBACK: a rolled back
// request (Prefer: tx=rollback) undid it, together with the record that the
// connection needed one, and the next internal user of the pool got the role
// and claims a function of an earlier request had left on the session.
func TestCleanupOutlivesARolledBackTransaction(t *testing.T) {
	residueSetup(t)
	s := newAuthzServer(t, "8173", map[string]any{
		"SessionMode":                 "none",
		"Database.TransactionMode":    "commit-allow-override",
		"Database.MinPoolConnections": int32(1),
		"Database.MaxPoolConnections": int32(1),
	})
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()
	base := "http://localhost:8173/api/postgres"
	user := signClaims(t, map[string]any{"role": "authz_user", "sub": "me"}, postgrestJWTSecret)
	poison(t, base+"/rpc/authz_poison?r=authz_anon&c=%7B%22sub%22%3A%22other%22%7D", user)
	// a request whose function leaves a setting the release resets, rolled back
	req, _ := http.NewRequest("POST", base+"/rpc/authz_ro", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+user)
	req.Header.Set("Prefer", "tx=rollback")
	resp, err := slowClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("authz_ro: %d %s", resp.StatusCode, body)
	}
	if session, current, claims := identityOf(t, s); session != "authz_authn" || current != "authz_authn" || claims != "" {
		t.Errorf("after a rolled back request: the pool hands out %s/%s with claims %q", session, current, claims)
	}
}

// database.AcquireConnection is exported: a library user takes a connection
// with it and runs its own statements, with no request variables. It was the
// middleware's acquire, which skips the cleanup since a request applies its
// own role and claims: the library user got whatever a function of an
// earlier request had left on the session.
func TestLibraryAcquireGetsACleanConnection(t *testing.T) {
	residueSetup(t)
	s := newAuthzServer(t, "8174", map[string]any{
		"SessionMode":                 "none",
		"Database.MinPoolConnections": int32(1),
		"Database.MaxPoolConnections": int32(1),
	})
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()
	base := "http://localhost:8174/api/postgres"
	user := signClaims(t, map[string]any{"role": "authz_user", "sub": "me"}, postgrestJWTSecret)
	// twice: the second time from the connection's statement cache
	for i := 0; i < 2; i++ {
		poison(t, base+"/rpc/authz_poison?r=authz_anon&c=%7B%22sub%22%3A%22other%22%7D", user)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		conn, err := database.AcquireConnection(ctx, nil)
		if err != nil {
			cancel()
			t.Fatal(err)
		}
		var current, claims string
		err = conn.QueryRow(ctx, "SELECT current_user::text, coalesce(current_setting('request.jwt.claims', true), '')").Scan(&current, &claims)
		conn.Release()
		cancel()
		if err != nil {
			t.Fatal(err)
		}
		if current != "authz_authn" || claims != "" {
			t.Errorf("a library acquire (%d) got %s with claims %q", i, current, claims)
		}
	}
}
