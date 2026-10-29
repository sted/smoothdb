package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sted/smoothdb/database"
	"github.com/sted/smoothdb/test"
)

// slowClient waits longer than probeClient: in SessionMode "role" a single
// connection is retained by the previous session until the watcher releases
// it, about two seconds.
var slowClient = &http.Client{Timeout: 15 * time.Second}

type requestSettings struct {
	User               string `json:"user"`
	StatementTimeout   string `json:"statement_timeout"`
	WorkMem            string `json:"work_mem"`
	MaintenanceWorkMem string `json:"maintenance_work_mem"`
	Isolation          string `json:"isolation"`
}

func settingsOf(t *testing.T, base, token string) requestSettings {
	t.Helper()
	req, _ := http.NewRequest("GET", base+"/rpc/authz_settings", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := slowClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("authz_settings: %d %s", resp.StatusCode, body)
	}
	var s requestSettings
	if err := json.Unmarshal(body, &s); err != nil {
		t.Fatalf("authz_settings: %s: %v", body, err)
	}
	return s
}

// PostgREST applies the settings of the impersonated role (ALTER ROLE ... SET,
// read from pg_roles.rolconfig for the roles granted to the authenticator:
// Config/Database.hs, queryRoleSettings) to each of its requests, with
// set_config(..., true) before the role switch (Query/PreQuery.hs), and runs
// the transaction at the role's default_transaction_isolation (MainTx.hs,
// planIsoLvl). Verified against PostgREST 16.4: a role with statement_timeout
// = 150ms and work_mem = 7MB sees both, its sleep of 0.5 s fails with 57014
// (500), an anonymous request right after sees the defaults, a role with
// default_transaction_isolation = serializable runs serializable, and a
// per-database setting (ALTER ROLE ... IN DATABASE) is not applied.
//
// A pool of one connection makes every request reuse the connection of the
// previous one, so a setting that survived its request would show up in the
// next one, run as another role.
func TestRoleSettingsApplyToTheirRequestsOnly(t *testing.T) {
	authzSetup(t)
	cases := []struct{ name, port, tx, session string }{
		{"autocommit, retained connections", "8114", "none", "role"},
		{"autocommit, no sessions", "8115", "none", "none"},
		{"transactions, retained connections", "8116", "rollback", "role"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
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
			iso := signClaims(t, map[string]any{"role": "authz_iso"}, postgrestJWTSecret)
			defaults := requestSettings{"authz_anon", "0", "4MB", "64MB", "read committed"}

			if got := settingsOf(t, base, ""); got != defaults {
				t.Errorf("anonymous, first: got %+v, want %+v", got, defaults)
			}
			want := requestSettings{"authz_user", "500ms", "7MB", "64MB", "read committed"}
			if got := settingsOf(t, base, user); got != want {
				t.Errorf("authz_user: got %+v, want %+v", got, want)
			}
			if got := settingsOf(t, base, ""); got != defaults {
				t.Errorf("anonymous after authz_user: got %+v, want %+v", got, defaults)
			}
			want = requestSettings{"authz_iso", "0", "4MB", "64MB", "serializable"}
			if got := settingsOf(t, base, iso); got != want {
				t.Errorf("authz_iso: got %+v, want %+v", got, want)
			}
			if got := settingsOf(t, base, ""); got != defaults {
				t.Errorf("anonymous after authz_iso: got %+v, want %+v", got, defaults)
			}

			// the statement timeout is in force: PostgREST answers 500 57014
			// (the time is not a measure: in "role" mode the request first
			// waits for the connection the previous session retains)
			req, _ := http.NewRequest("POST", base+"/rpc/authz_sleep", strings.NewReader(`{"s": 2}`))
			req.Header.Set("Authorization", "Bearer "+user)
			req.Header.Set("Content-Type", "application/json")
			resp, err := slowClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode != http.StatusInternalServerError || !strings.Contains(string(body), `"57014"`) {
				t.Errorf("sleep past the role's statement timeout: got %d %s", resp.StatusCode, body)
			}
			if got := settingsOf(t, base, ""); got != defaults {
				t.Errorf("anonymous after the timeout: got %+v, want %+v", got, defaults)
			}
		})
	}
}

// The role settings are read with the schema cache, so the schema reload
// notification picks up an ALTER ROLE as PostgREST's config reload does.
func TestRoleSettingsReloadWithTheSchemaCache(t *testing.T) {
	authzSetup(t)
	s := newAuthzServer(t, "8117", map[string]any{
		"SessionMode":              "none",
		"Database.TransactionMode": "rollback",
	})
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()
	base := "http://localhost:8117/api/postgres"
	user := signClaims(t, map[string]any{"role": "authz_user"}, postgrestJWTSecret)
	if got := settingsOf(t, base, user).WorkMem; got != "7MB" {
		t.Fatalf("work_mem before the change: got %s, want 7MB", got)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, superuserURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	for _, q := range []string{"ALTER ROLE authz_user SET work_mem = '8MB'", "NOTIFY smoothdb, 'reload schema'"} {
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(5 * time.Second)
	for settingsOf(t, base, user).WorkMem != "8MB" {
		if time.Now().After(deadline) {
			t.Fatal("work_mem never followed the ALTER ROLE after a schema reload")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// PostgREST bounds the wait for a pool connection (db-pool-acquisition-timeout,
// 10 s by default) and answers 504 PGRST003 when it runs out (Error.hs:
// AcquisitionTimeoutUsageError). Verified against PostgREST 16.4 with a pool of
// one held by a slow request: 504 after 1.04 s with a 1 s timeout.
func TestPoolAcquisitionTimeout(t *testing.T) {
	var defaults map[string]any
	raw, _ := json.Marshal(database.DefaultConfig())
	json.Unmarshal(raw, &defaults)
	if got := defaults["PoolAcquisitionTimeout"]; got != float64(10) {
		t.Errorf("default PoolAcquisitionTimeout: got %v, want PostgREST's 10", got)
	}
	s := newSingleConnectionServer(t, map[string]any{
		"Address":                         "localhost:8113",
		"AllowAnon":                       true,
		"Database.AnonRole":               "postgres",
		"Database.PoolAcquisitionTimeout": 1,
		"EnableAdminRoute":                false,
	})
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()
	base := "http://localhost:8113/api/postgres"

	conn, err := s.DBE.AcquireConnection(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	test.Execute(t, test.Config{BaseUrl: base}, []test.Test{
		{
			Description: "a saturated pool answers 504 once the acquisition timeout runs out",
			Query:       "",
			Expected:    `{"subsystem":"database","message":"Timed out acquiring connection from connection pool.","code":"PGRST003","hint":"","details":null,"position":0}`,
			Status:      504,
		},
	})
	if elapsed := time.Since(start); elapsed < 900*time.Millisecond || elapsed > 3*time.Second {
		t.Errorf("the request waited %v, want about the 1 s timeout", elapsed)
	}
	conn.Release()

	// positive control: with the connection back the same request is served
	test.Execute(t, test.Config{BaseUrl: base}, []test.Test{
		{
			Description: "the pool serves again once the connection is back",
			Query:       "",
			Status:      200,
		},
	})
}

// superuserSetup runs stmts with a superuser connection of its own and drop
// when the test ends (before authzSetup's teardown, registered earlier).
func superuserSetup(t *testing.T, drop, stmts []string) {
	t.Helper()
	ctx := context.Background()
	run := func(qs []string) {
		conn, err := pgx.Connect(ctx, superuserURL)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close(ctx)
		for _, q := range qs {
			if _, err := conn.Exec(ctx, q); err != nil {
				t.Errorf("%q: %v", q, err)
			}
		}
	}
	run(append(drop, stmts...))
	t.Cleanup(func() { run(drop) })
}

func getBody(t *testing.T, url, token string) string {
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
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: %d %s", url, resp.StatusCode, body)
	}
	return string(body)
}

// A role's search_path applies to its requests, and the next role on the
// connection gets Database.SchemaSearchPath back, as each PostgREST request
// sets the role's settings transaction-locally over the connection's own
// search path. With autocommit the role settings are session settings that
// the release RESETs: the configured search path was a session value too, so
// RESET dropped it for the server's default ("$user", public), and every
// later request on the connection ran without it.
func TestRoleSearchPathResetsToTheConfiguredOne(t *testing.T) {
	authzSetup(t)
	superuserSetup(t, []string{
		"DROP FUNCTION IF EXISTS role_search_path()",
		"DROP ROLE IF EXISTS authz_path",
	}, []string{
		"CREATE ROLE authz_path NOLOGIN",
		"GRANT authz_path TO authz_authn",
		"ALTER ROLE authz_path SET search_path = public, authz_nowhere",
		"CREATE FUNCTION role_search_path() RETURNS text LANGUAGE sql AS $$ SELECT current_setting('search_path') $$",
	})
	s := newAuthzServer(t, "8121", map[string]any{
		"SessionMode":                 "none",
		"Database.TransactionMode":    "none",
		"Database.SchemaSearchPath":   []string{"public", "information_schema"},
		"Database.MinPoolConnections": int32(1),
		"Database.MaxPoolConnections": int32(1),
	})
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()
	url := "http://localhost:8121/api/postgres/rpc/role_search_path"
	path := signClaims(t, map[string]any{"role": "authz_path"}, postgrestJWTSecret)

	configured := `"public, information_schema"`
	if got := getBody(t, url, ""); got != configured {
		t.Errorf("anonymous, first: got %s, want %s", got, configured)
	}
	if got := getBody(t, url, path); got != `"public, authz_nowhere"` {
		t.Errorf("authz_path: got %s, want the role's search path", got)
	}
	if got := getBody(t, url, ""); got != configured {
		t.Errorf("anonymous after authz_path: got %s, want %s", got, configured)
	}
}
