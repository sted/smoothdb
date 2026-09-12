package server

import (
	"context"
	"testing"
	"time"
)

// ensureAnonRole creates a NOLOGIN, non-superuser role to use as the anonymous
// role, switched into from the pool's own (here superuser) role.
func ensureAnonRole(t *testing.T, s *Server, name string) {
	t.Helper()
	ctx := context.Background()
	conn, err := s.DBE.AcquireConnection(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "DROP ROLE IF EXISTS "+name); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, "CREATE ROLE "+name+" NOLOGIN"); err != nil {
		t.Fatal(err)
	}
}

// acquireWithin reports whether the main pool hands out a connection within d.
func acquireWithin(t *testing.T, s *Server, d time.Duration) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	conn, err := s.DBE.AcquireConnection(ctx)
	if err != nil {
		return false
	}
	conn.Release()
	return true
}

// In "role" mode the prepared connection stays attached to the session for
// about a second after the request, so pool demand tracks active sessions
// (CollHub measured a 20-connection pool collapsing under 40 users). "claims"
// mode keeps the verified claims but returns the connection at request end.
// With a pool of one connection the difference is directly observable.
func TestSessionModeConnectionRetention(t *testing.T) {
	cases := []struct {
		mode      string
		port      string
		available bool // is the only connection back in the pool right after a request?
	}{
		{"role", "8097", false},
		{"claims", "8098", true},
	}
	for _, c := range cases {
		t.Run(c.mode, func(t *testing.T) {
			s := newSingleConnectionServer(t, map[string]any{
				"Address":           "localhost:" + c.port,
				"AllowAnon":         true,
				"Database.AnonRole": "sessmode_anon",
				"SessionMode":       c.mode,
				"EnableAdminRoute":  false,
			})
			// The anonymous request below must acquire a connection to exercise
			// retention; an empty AnonRole is now refused before any connection
			// is taken, so configure a non-superuser role to switch into.
			ensureAnonRole(t, s, "sessmode_anon")
			done := startTestServer(t, s)
			defer func() {
				s.Shutdown(context.Background())
				<-done
			}()
			// any request through the API middleware; its status is irrelevant
			statusWithin(t, "http://localhost:"+c.port+"/api/postgres/whatever")
			if got := acquireWithin(t, s, 300*time.Millisecond); got != c.available {
				t.Fatalf("SessionMode %q: connection available right after the request = %v, want %v", c.mode, got, c.available)
			}
		})
	}
}
