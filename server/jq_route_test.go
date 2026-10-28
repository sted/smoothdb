package server

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const jqProbeBody = `{"evals":[{"program":".a","input":{"a":1}}]}`

// postJQ sends a jq batch, with a bearer token when one is given, and returns
// the status and the body. The client is bounded: a request stuck waiting
// for a pool connection must fail the test, not hang it.
func postJQ(t *testing.T, url, token string) (int, string) {
	t.Helper()
	req, err := http.NewRequest("POST", url, strings.NewReader(jqProbeBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

// POST /jq is pure CPU, but it was grouped under the database middleware: each
// call acquired a pool connection, ran SET ROLE and set_config on it and held
// it for the whole batch. With the pool exhausted by another request, a jq
// call waited for a connection it never used.
func TestJQDoesNotNeedAConnection(t *testing.T) {
	s := newSingleConnectionServer(t, map[string]any{
		"Address":           "localhost:9300",
		"AllowAnon":         true,
		"Database.AnonRole": "jq_anon",
		"JQ.Enabled":        true,
		"EnableAdminRoute":  false,
	})
	ensureAnonRole(t, s, "jq_anon")
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()
	url := "http://localhost:9300/jq"

	// control: with the pool free the call answers
	if status, body := postJQ(t, url, ""); status != http.StatusOK || body != `[{"output":1}]` {
		t.Fatalf("control: expected 200 [{\"output\":1}], got %d %s", status, body)
	}

	conn, err := s.DBE.AcquireConnection(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if status, body := postJQ(t, url, ""); status != http.StatusOK || body != `[{"output":1}]` {
		t.Fatalf("with the pool exhausted: expected 200 [{\"output\":1}], got %d %s", status, body)
	}
}

// Without the database middleware the route keeps its authentication: a token
// is verified and the anonymous-access rules are the same as for every other
// route.
func TestJQAuthentication(t *testing.T) {
	const secret = "jq-route-test-secret"
	s := newSingleConnectionServer(t, map[string]any{
		"Address":          "localhost:9301",
		"AllowAnon":        false,
		"JWTSecret":        secret,
		"JQ.Enabled":       true,
		"EnableAdminRoute": false,
	})
	// the role exists so that the control passes on either middleware
	ensureAnonRole(t, s, "jq_user")
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()
	url := "http://localhost:9301/jq"

	// control: a valid token with a role is served
	valid := signClaims(t, jwt.MapClaims{"role": "jq_user"}, secret)
	if status, body := postJQ(t, url, valid); status != http.StatusOK {
		t.Fatalf("valid token: expected 200, got %d %s", status, body)
	}
	cases := []struct {
		name  string
		token string
	}{
		{"no token", ""},
		{"bad signature", signClaims(t, jwt.MapClaims{"role": "jq_user"}, "another-secret")},
		{"no role claim, anonymous access off", signClaims(t, jwt.MapClaims{"id": "jdoe"}, secret)},
		{"empty role claim", signClaims(t, jwt.MapClaims{"role": ""}, secret)},
		{"expired", signClaims(t, jwt.MapClaims{"role": "jq_user", "exp": time.Now().Add(-time.Minute).Unix()}, secret)},
	}
	for _, c := range cases {
		if status, body := postJQ(t, url, c.token); status != http.StatusUnauthorized {
			t.Errorf("%s: expected 401, got %d %s", c.name, status, body)
		}
	}
}

// Anonymous access on but no anonymous role configured: refused, as by the
// database middleware (PostgREST PGRST302).
func TestJQAnonymousWithoutAnonRole(t *testing.T) {
	s := newSingleConnectionServer(t, map[string]any{
		"Address":           "localhost:9302",
		"AllowAnon":         true,
		"Database.AnonRole": "",
		"JQ.Enabled":        true,
		"EnableAdminRoute":  false,
	})
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()
	if status, body := postJQ(t, "http://localhost:9302/jq", ""); status != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d %s", status, body)
	}
}
