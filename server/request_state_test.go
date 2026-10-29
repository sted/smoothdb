package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// pooledState is what a connection taken from the pool carries: the role, the
// claims GUC, a role setting (authz_user sets work_mem) and the access mode.
type pooledState struct {
	User, Claims, WorkMem, ReadOnly string
}

func pooledStateOf(t *testing.T, s *Server) pooledState {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := s.DBE.AcquireConnection(ctx)
	if err != nil {
		t.Fatalf("no connection in the pool after the request: %v", err)
	}
	defer conn.Release()
	var st pooledState
	err = conn.QueryRow(ctx, `SELECT current_user::text,
		coalesce(current_setting('request.jwt.claims', true), ''),
		current_setting('work_mem'), current_setting('default_transaction_read_only')`).
		Scan(&st.User, &st.Claims, &st.WorkMem, &st.ReadOnly)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// A connection goes back to the pool with nothing of the request that used
// it: not its role, not its claims, not its role settings, not its access
// mode. The pool also serves the schema reload, the activation of a database
// and the library's ContextWithDb, which take a connection without preparing
// it (or with an empty role), and would run as the last client otherwise.
// PostgREST cannot leak them: every request variable is set with
// set_config(..., true) inside the request's transaction (Query/PreQuery.hs,
// txVarQuery; SqlFragment.hs, setConfigWithConstantName -> setConfigLocal).
//
// smoothdb set them session-level: with a TransactionMode other than "none"
// nothing reset them at the end of the request, and with "none" the reset left
// the claims. A pool of one connection makes the probe take the connection the
// request just used. "role" with TransactionMode "none" keeps the connection
// on the session on purpose, and is not a case here.
func TestPooledConnectionCarriesNothingOfTheRequest(t *testing.T) {
	authzSetup(t)
	cases := []struct{ session, tx, port string }{
		{"none", "none", "8150"},
		{"claims", "none", "8151"},
		{"none", "commit", "8152"},
		{"role", "commit", "8153"},
		{"none", "rollback-allow-override", "8154"},
	}
	clean := pooledState{"authz_authn", "", "4MB", "off"}
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
			user := signClaims(t, map[string]any{"role": "authz_user", "sub": "leak-probe"}, postgrestJWTSecret)

			// a read (run READ ONLY) and a call, as authz_user
			if got := settingsOf(t, base, user); got.User != "authz_user" || got.WorkMem != "7MB" {
				t.Fatalf("the request itself: got %+v", got)
			}
			if got := pooledStateOf(t, s); got != clean {
				t.Errorf("after a request: the pooled connection carries %+v, want %+v", got, clean)
			}
			// positive control: the probe sees what a request sets when the
			// request is still running it, so it can tell a leak
			if got := settingsOf(t, base, user); got.User != "authz_user" {
				t.Fatalf("second request: got %+v", got)
			}
		})
	}
}

// statusAndCode sends a GET with a bearer token and returns the status and the
// code of the error body.
func statusAndCode(t *testing.T, url, token string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := slowClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var e struct{ Code string }
	json.Unmarshal(body, &e)
	return resp.StatusCode, e.Code
}

// PostgREST plans a request from the schema cache before it opens the
// transaction that switches the role (App.hs: Plan.actionPlan, then
// MainTx.mainTx with the txVarQuery), so an error of the request itself comes
// before an error of the role. Verified against PostgREST 16.4 with a token
// naming a role that does not exist:
//
//	GET /items?id=xx.1     -> 400 PGRST100 (failed to parse filter)
//	GET /rpc/no_such_fn    -> 404 PGRST202
//	GET /items             -> 401 22023 (role "nobody_role_x" does not exist)
//
// smoothdb switched the role in the middleware, before the handler, and
// answered 401 to all three. A filter it cannot parse is refused before any
// statement now, with 400 (the PGRST100 body is card 32494's, error body);
// an unknown function is looked up by the database, not in the schema cache,
// so there the role still fails first (401 22023, not asserted here).
//
// The cases are checked by status, and by code where smoothdb has PostgREST's.
func TestRequestErrorsComeBeforeTheRoleSwitch(t *testing.T) {
	authzSetup(t)
	cases := []struct{ session, tx, port string }{
		{"none", "none", "8155"},
		{"role", "none", "8156"},
		{"none", "commit", "8157"},
	}
	for _, c := range cases {
		t.Run(c.session+"/"+c.tx, func(t *testing.T) {
			s := newAuthzServer(t, c.port, map[string]any{
				"SessionMode":              c.session,
				"Database.TransactionMode": c.tx,
			})
			done := startTestServer(t, s)
			defer func() {
				s.Shutdown(context.Background())
				<-done
			}()
			base := "http://localhost:" + c.port + "/api/postgres"
			missing := signClaims(t, map[string]any{"role": "authz_nobody"}, postgrestJWTSecret)
			for _, want := range []struct {
				path   string
				status int
				code   string
			}{
				{"/authz_open?note=xx.1", 400, ""},
				// control: a request that reaches the database fails on the role
				{"/authz_open", 401, "22023"},
				// and again, now that the session (in "role" mode) exists
				{"/authz_open?note=xx.1", 400, ""},
				{"/authz_open", 401, "22023"},
			} {
				if status, code := statusAndCode(t, base+want.path, missing); status != want.status || (want.code != "" && code != want.code) {
					t.Errorf("GET %s with a role that does not exist: got %d %s, want %d %s",
						want.path, status, code, want.status, want.code)
				}
			}
		})
	}
}

// A role reaches a schema its authenticator cannot use: authz_authn is
// NOINHERIT and has no USAGE on authz_hidden, authz_user has. The statement
// must be parsed as the role, also the first time the connection sees it (no
// prepared statement yet: the parse is where USAGE is checked), and the
// second time, from the connection's statement cache. Anonymous has no USAGE
// either: its request is refused.
func TestRoleReachesASchemaItsAuthenticatorCannot(t *testing.T) {
	authzSetup(t)
	superuserSetup(t, []string{"DROP SCHEMA IF EXISTS authz_hidden CASCADE"}, []string{
		"CREATE SCHEMA authz_hidden",
		"CREATE TABLE authz_hidden.notes (note text)",
		"INSERT INTO authz_hidden.notes VALUES ('hidden')",
		"CREATE FUNCTION authz_hidden.echo(n text) RETURNS text LANGUAGE sql AS $$ SELECT n $$",
		"GRANT USAGE ON SCHEMA authz_hidden TO authz_user",
		"GRANT SELECT, INSERT ON authz_hidden.notes TO authz_user",
	})
	cases := []struct{ session, tx, port string }{
		{"none", "none", "8158"},
		{"role", "none", "8159"},
		{"none", "commit", "8160"},
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
			user := signClaims(t, map[string]any{"role": "authz_user"}, postgrestJWTSecret)
			do := func(method, path, token, body string) (int, string) {
				var r io.Reader
				if body != "" {
					r = strings.NewReader(body)
				}
				req, _ := http.NewRequest(method, base+path, r)
				req.Header.Set("Accept-Profile", "authz_hidden")
				req.Header.Set("Content-Profile", "authz_hidden")
				req.Header.Set("Content-Type", "application/json")
				if token != "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}
				resp, err := slowClient.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				b, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				return resp.StatusCode, string(b)
			}
			for i := 0; i < 2; i++ {
				if status, body := do("GET", "/notes?select=note&note=eq.hidden", user, ""); status != 200 || body != `[{"note":"hidden"}]` {
					t.Errorf("GET %d as authz_user: got %d %s", i, status, body)
				}
				if status, body := do("GET", "/rpc/echo?n=x", user, ""); status != 200 || body != `"x"` {
					t.Errorf("rpc %d as authz_user: got %d %s", i, status, body)
				}
				if status, body := do("POST", "/notes", user, `{"note":"w"}`); status != 201 {
					t.Errorf("POST %d as authz_user: got %d %s", i, status, body)
				}
				if status, body := do("GET", "/notes?select=note", "", ""); status != 401 {
					t.Errorf("GET %d anonymous: got %d %s, want 401 (no USAGE)", i, status, body)
				}
			}
		})
	}
}
