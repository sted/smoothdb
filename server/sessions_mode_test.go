package server

import (
	"context"
	"testing"
	"time"
)

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
				"Address":          "localhost:" + c.port,
				"AllowAnon":        true,
				"SessionMode":      c.mode,
				"EnableAdminRoute": false,
			})
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
