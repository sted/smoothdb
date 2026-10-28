package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/sted/heligo"
)

// getWithToken sends a GET with a bearer token and returns the status.
func getWithToken(t *testing.T, url, token string) int {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := probeClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// A panic in a handler is caught by heligo.Recover, outside the auth
// middleware, so the middleware's release of the request connection must run
// on the way out of the panic too. It was a plain call after next(): every
// panicking request kept its connection, still inside its transaction, and
// with a pool of one the next request could not get any.
func TestPanicInHandlerReleasesConnection(t *testing.T) {
	s := newSingleConnectionServer(t, map[string]any{
		"Address":           "localhost:8178",
		"AllowAnon":         true,
		"Database.AnonRole": "leak_anon",
		"SessionMode":       "none",
		"EnableAdminRoute":  false,
	})
	ensureAnonRole(t, s, "leak_anon")
	s.GetRouter().Group("/panic", s.MiddlewareDBE()).Handle("GET", "/", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		panic("handler panic")
	})
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()

	if status := statusWithin(t, "http://localhost:8178/panic/"); status != http.StatusInternalServerError {
		t.Fatalf("expected 500 from the recovered panic, got %d", status)
	}
	if !acquireWithin(t, s, 500*time.Millisecond) {
		t.Fatal("the connection of the panicking request was not returned to the pool")
	}
}
