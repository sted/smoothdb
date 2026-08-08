package server

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// A leaked connection does not make the next request fail, it makes it wait in
// the pool forever — so a client without a deadline turns a red test into a
// hung one. Every request here gets a bound.
var probeClient = &http.Client{Timeout: 5 * time.Second}

func statusWithin(t *testing.T, url string) int {
	t.Helper()
	resp, err := probeClient.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// A pool of exactly one connection makes both properties below observable with
// a single acquire, and makes a single leaked connection fatal instead of
// merely costly.
func newSingleConnectionServer(t *testing.T, config map[string]any) *Server {
	t.Helper()
	base := map[string]any{
		"Database.MinPoolConnections": int32(1),
		"Database.MaxPoolConnections": int32(1),
	}
	for k, v := range config {
		base[k] = v
	}
	return newTestServer(t, base)
}

// Readiness means "route traffic here", so it must answer for the database
// layer too: an instance whose pool hands out no connection serves nothing,
// and a 200 keeps a load balancer sending requests into the hole.
func TestReadyReportsAnUnservableDatabase(t *testing.T) {
	s := newSingleConnectionServer(t, map[string]any{"Address": "localhost:8095"})
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()
	base := "http://localhost:8095"

	if got := statusWithin(t, base+"/ready"); got != http.StatusOK {
		t.Fatalf("expected 200 from /ready on a healthy pool, got %d", got)
	}

	// Hold the only connection: every further acquire, the probe's included,
	// now blocks until its deadline.
	conn, err := s.DBE.AcquireConnection(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := statusWithin(t, base+"/ready"); got != http.StatusServiceUnavailable {
		conn.Release()
		t.Fatalf("expected 503 from /ready with the pool exhausted, got %d", got)
	}
	// Liveness answers a different question: the process is fine, killing it
	// would not bring the database back.
	if got := statusWithin(t, base+"/live"); got != http.StatusOK {
		conn.Release()
		t.Fatalf("expected 200 from /live with the pool exhausted, got %d", got)
	}

	conn.Release()
	if got := statusWithin(t, base+"/ready"); got != http.StatusOK {
		t.Fatalf("expected 200 from /ready once the connection is back, got %d", got)
	}
}

// A request that fails while the role is being set returns before the release
// path runs. The connection it took has to go back to the pool anyway: with
// MaxPoolConnections at 1, one leak makes the instance serve nothing ever
// again, which is exactly how a live server dies after enough client hangups.
func TestAFailedRoleSwitchReturnsTheConnection(t *testing.T) {
	s := newSingleConnectionServer(t, map[string]any{
		"Address":           "localhost:8096",
		"AllowAnon":         true,
		"Database.AnonRole": "smoothdb_no_such_role",
		"EnableAdminRoute":  false,
	})
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()
	base := "http://localhost:8096"

	// SET ROLE on a role that does not exist fails, so each of these takes a
	// connection and then errors out of the acquire path.
	for i := range 3 {
		if got := statusWithin(t, base+"/api/postgres/whatever"); got == http.StatusOK {
			t.Fatalf("request %d unexpectedly succeeded with a bogus anon role", i)
		}
	}

	// The pool is the assertion: the readiness probe needs a connection.
	deadline := time.Now().Add(5 * time.Second)
	for {
		if statusWithin(t, base+"/ready") == http.StatusOK {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the pool never recovered: a failed role switch leaked its connection")
		}
		time.Sleep(50 * time.Millisecond)
	}
}
