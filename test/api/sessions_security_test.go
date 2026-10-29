package test_api

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sted/smoothdb/test"
)

// The test server runs with SessionMode "role": a session keyed by the token is
// created before the token is verified and, on a hit, verification is skipped.
// A half-built session left behind by a failed request must not be reachable
// by the next one, and an expired token must be refused even on a hit.

func getStatus(t *testing.T, token, query string) int {
	t.Helper()
	cfg := test.Config{BaseUrl: "http://localhost:8082/api", CommonHeaders: test.Headers{"Authorization": {token}}}
	_, _, status, err := test.Exec(test.InitClient(), cfg, &test.Command{Query: query})
	if err != nil {
		t.Fatal(err)
	}
	return status
}

// Before the fix the second request found the session with nil claims and
// panicked (500 from the recovery handler).
func TestInvalidTokenIsUnauthorizedOnEveryRequest(t *testing.T) {
	for i := 1; i <= 3; i++ {
		if s := getStatus(t, "Bearer not.a.valid.token", "/dbtest"); s != http.StatusUnauthorized {
			t.Fatalf("request %d with an invalid token: expected 401, got %d", i, s)
		}
	}
}

// Before the fix the second request found the session with a nil database,
// took a connection from the main pool and listed the main database instead.
func TestMissingDatabaseIsNotFoundOnEveryRequest(t *testing.T) {
	for i := 1; i <= 3; i++ {
		if s := getStatus(t, adminToken, "/no_such_database_xyz"); s != http.StatusNotFound {
			t.Fatalf("request %d for a missing database: expected 404, got %d", i, s)
		}
	}
}

// A token about to expire opens a session; the session lives ~5 s after its
// last use, so a request after expiry is a hit and used to be accepted. With
// the 30 s clock skew (PostgREST's, on whole seconds) a token is expired once
// now - 30 > exp: exp is set 30 s back, and the second request is sent once
// the clock has turned the next second.
func TestExpiredTokenIsRefusedOnASessionHit(t *testing.T) {
	edge := time.Now().Truncate(time.Second).Add(time.Second)
	time.Sleep(time.Until(edge.Add(-400 * time.Millisecond)))
	exp := edge.Add(-time.Second).Unix() - 30
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"role": "admin", "exp": exp}).
		SignedString([]byte(srv.JWTSecret()))
	if err != nil {
		t.Fatal(err)
	}
	token = "Bearer " + token
	if s := getStatus(t, token, "/dbtest"); s != http.StatusOK {
		t.Fatalf("fresh token: expected 200, got %d", s)
	}
	time.Sleep(time.Until(edge) + 100*time.Millisecond)
	if s := getStatus(t, token, "/dbtest"); s != http.StatusUnauthorized {
		t.Fatalf("expired token on a session hit: expected 401, got %d", s)
	}
}
