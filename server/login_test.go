package server

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func postCredentials(t *testing.T, url, user, password string) (int, string) {
	t.Helper()
	body := `{"email":"` + user + `","password":"` + password + `"}`
	resp, err := probeClient.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

// In db login mode /token, reachable without credentials, answered a failed
// login with the text of the pgx ConnectError: the database user, the
// database name, the dial address and the server's FATAL message.
func TestTokenDoesNotLeakConnectionDetails(t *testing.T) {
	s := newTestServer(t, map[string]any{
		"Address":          "localhost:9303",
		"LoginMode":        "db",
		"JWTSecret":        "login-test-secret",
		"LoginRateLimit":   0,
		"EnableAdminRoute": false,
	})
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()
	url := "http://localhost:9303/token"

	// control: the configured user logs in (trust or its password)
	if status, body := postCredentials(t, url, "postgres", "postgres"); status != http.StatusOK || !strings.Contains(body, "access_token") {
		t.Fatalf("control: expected 200 with a token, got %d %s", status, body)
	}

	status, body := postCredentials(t, url, "smoothdb_no_such_login", "x")
	if status != http.StatusBadRequest {
		t.Errorf("expected 400 for a failed login, got %d", status)
	}
	for _, s := range []string{"smoothdb_no_such_login", "database=", "postgres", "5432", "127.0.0.1", "localhost", "FATAL", "SQLSTATE"} {
		if strings.Contains(body, s) {
			t.Errorf("the body contains %q: %s", s, body)
		}
	}
}
