package server

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// With LoginMode "db", /token took a successful connection as the proof of
// the password. Under trust, peer or ident authentication PostgreSQL never
// asks for it, and /token signed a token for any LOGIN role with any
// password. A token is now issued only when PostgreSQL asked for a password,
// unless LoginAllowTrust is on (SMOOTHDB_DEBUG sets it). An empty password
// and the authenticator's own name are refused before any connection.
func TestTokenNeedsAPasswordCheck(t *testing.T) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, superuserURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	adminTxExec(t, conn, `DROP ROLE IF EXISTS trustchk_user`,
		`CREATE ROLE trustchk_user LOGIN PASSWORD 'right-password'`)
	defer adminTxExec(t, conn, `DROP ROLE IF EXISTS trustchk_user`)

	// probe: does PostgreSQL ask this host for a password?
	trust := false
	if c, err := pgx.Connect(ctx, "postgresql://trustchk_user:wrong-password@localhost:5432/postgres"); err == nil {
		trust = true
		c.Close(ctx)
	}
	t.Logf("PostgreSQL trusts this host: %v", trust)

	for i, allowTrust := range []bool{false, true} {
		addr := []string{"localhost:9304", "localhost:9305"}[i]
		s := newTestServer(t, map[string]any{
			"Address":          addr,
			"LoginMode":        "db",
			"JWTSecret":        "login-trust-test-secret-0123456789",
			"LoginRateLimit":   0,
			"LoginAllowTrust":  allowTrust,
			"EnableAdminRoute": false,
		})
		done := startTestServer(t, s)
		url := "http://" + addr + "/token"
		cases := []struct {
			name, user, password string
			ok                   bool
		}{
			{"right password", "trustchk_user", "right-password", !trust || allowTrust},
			{"wrong password", "trustchk_user", "wrong-password", trust && allowTrust},
			{"empty password", "trustchk_user", "", false},
			{"the authenticator", "postgres", "postgres", false},
		}
		for _, c := range cases {
			status, body := postCredentials(t, url, c.user, c.password)
			issued := status == 200 && strings.Contains(body, "access_token")
			if issued != c.ok {
				t.Errorf("LoginAllowTrust %v, %s: status %d, token issued %v, want %v: %s",
					allowTrust, c.name, status, issued, c.ok, body)
			}
			if !c.ok && status != 400 {
				t.Errorf("LoginAllowTrust %v, %s: status %d, want 400", allowTrust, c.name, status)
			}
		}
		s.Shutdown(context.Background())
		<-done
	}
}
