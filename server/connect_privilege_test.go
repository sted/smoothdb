package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sted/smoothdb/authn"
)

// SmoothDB connects to every database as the authenticator and then switches
// to the request role, so PostgreSQL never checks the CONNECT privilege of
// that role. A role that has no CONNECT on a database reached it through
// SmoothDB anyway. The check now runs in SmoothDB, before any SQL on the
// database; a REVOKE takes effect at the next schema cache reload.
func TestConnectPrivilegeIsChecked(t *testing.T) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, superuserURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	cleanup := []string{
		`DROP DATABASE IF EXISTS connchk_db WITH (FORCE)`,
		`DROP ROLE IF EXISTS connchk_user`, `DROP ROLE IF EXISTS connchk_other`, `DROP ROLE IF EXISTS connchk_anon`,
	}
	adminTxExec(t, conn, cleanup...)
	adminTxExec(t, conn,
		`CREATE ROLE connchk_user NOLOGIN`, `CREATE ROLE connchk_other NOLOGIN`, `CREATE ROLE connchk_anon NOLOGIN`,
		`CREATE DATABASE connchk_db`,
		`REVOKE CONNECT ON DATABASE connchk_db FROM PUBLIC`,
		`GRANT CONNECT ON DATABASE connchk_db TO connchk_user`)
	defer adminTxExec(t, conn, cleanup...)

	const secret = "connect-privilege-test-secret-0123456789"
	userToken, _ := authn.GenerateToken("connchk_user", secret)
	otherToken, _ := authn.GenerateToken("connchk_other", secret)

	addr := "localhost:8196"
	s := newPinnedTestServer(t, map[string]any{
		"Address":           addr,
		"AllowAnon":         true,
		"Database.AnonRole": "connchk_anon",
		"JWTSecret":         secret,
		"EnableAdminRoute":  true,
		"AdminRoles":        []string{"*"},
	})
	done := startTestServer(t, s)
	stopped := false
	stop := func() {
		if !stopped {
			stopped = true
			s.Shutdown(context.Background())
			<-done
		}
	}
	// the server holds a pool on connchk_db: stop it before the DROP DATABASE
	defer stop()

	get := func(path, token string) (int, string, string) {
		t.Helper()
		req, _ := http.NewRequest("GET", "http://"+addr+path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := slowClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		var e struct {
			Code string `json:"code"`
		}
		json.Unmarshal(b, &e)
		return resp.StatusCode, e.Code, string(b)
	}

	cases := []struct {
		who, path, token string
		status           int
		code             string
	}{
		{"granted role, data API", "/api/connchk_db/", userToken, 200, ""},
		{"granted role, admin API", "/admin/databases/connchk_db/tables", userToken, 200, ""},
		{"role without CONNECT, data API", "/api/connchk_db/", otherToken, 403, "42501"},
		{"role without CONNECT, admin API", "/admin/databases/connchk_db/tables", otherToken, 403, "42501"},
		{"anonymous without CONNECT", "/api/connchk_db/", "", 401, "42501"},
		{"a template database", "/api/template1/", userToken, 404, ""},
	}
	for _, c := range cases {
		t.Run(c.who, func(t *testing.T) {
			status, code, body := get(c.path, c.token)
			if status != c.status || (c.code != "" && code != c.code) {
				t.Errorf("status %d code %q, want %d %q: %s", status, code, c.status, c.code, body)
			}
		})
	}

	// a REVOKE reaches a cached session at the next schema cache reload
	t.Run("revoke after reload", func(t *testing.T) {
		adminTxExec(t, conn, `REVOKE CONNECT ON DATABASE connchk_db FROM connchk_user`,
			`NOTIFY smoothdb, 'reload schema'`)
		deadline := time.Now().Add(5 * time.Second)
		for {
			status, _, body := get("/api/connchk_db/", userToken)
			if status == 403 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("status %d after the reload, want 403: %s", status, body)
			}
			time.Sleep(100 * time.Millisecond)
		}
	})
	stop()
}
