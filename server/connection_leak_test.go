package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

// A request that hits the session with a token expired meanwhile drops the
// session, and must leave no connection behind. "role" kept the connection on
// the session for about a second, and a dropped session took it out of the
// watcher's sight; it is now a deprecated alias of "claims", which keeps no
// connection, and the test checks that the alias does not either. The
// control is the same flow with a valid token.
func TestExpiredTokenOnSessionHitReleasesConnection(t *testing.T) {
	s := newSingleConnectionServer(t, map[string]any{
		"Address":          "localhost:8179",
		"AllowAnon":        false,
		"JWTSecret":        "connection-leak-test-secret",
		"SessionMode":      "role",
		"EnableAdminRoute": false,
	})
	ensureAnonRole(t, s, "leak_user")
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()
	url := "http://localhost:8179/api/postgres/whatever"

	// control: a valid token, the watcher hands the connection back
	valid := signClaims(t, jwt.MapClaims{"role": "leak_user"}, s.JWTSecret())
	getWithToken(t, url, valid)
	if !acquireWithin(t, s, 3*time.Second) {
		t.Fatal("control: the watcher did not release the idle session connection")
	}

	// a token that expires between two requests within the retention window:
	// with the 30 s clock skew (PostgREST's, on whole seconds) a token is
	// expired once now - 30 > exp, so exp is set 30 s back and the second
	// request is sent once the clock has turned the next second
	edge := time.Now().Truncate(time.Second).Add(time.Second)
	time.Sleep(time.Until(edge.Add(-400 * time.Millisecond)))
	exp := edge.Add(-time.Second).Unix() - 30
	expiring := signClaims(t, jwt.MapClaims{"role": "leak_user", "exp": exp}, s.JWTSecret())
	if status := getWithToken(t, url, expiring); status == http.StatusUnauthorized {
		t.Fatalf("first request: the token is still valid, got %d", status)
	}
	time.Sleep(time.Until(edge) + 100*time.Millisecond)
	if status := getWithToken(t, url, expiring); status != http.StatusUnauthorized {
		t.Fatalf("second request: expected 401 for the expired token, got %d", status)
	}
	if !acquireWithin(t, s, 3*time.Second) {
		t.Fatal("the connection of the dropped session was never returned to the pool")
	}
}
