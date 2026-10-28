package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/sted/heligo"
)

// loginHelper is the part of Helper that InitLoginRoute uses.
type loginHelper struct {
	Helper
	router *heligo.Router
}

func (h loginHelper) GetRouter() *heligo.Router { return h.router }
func (h loginHelper) LoginRateLimit() int       { return 0 }
func (h loginHelper) RequestMaxBytes() int64    { return 1 << 20 }

// postToken posts credentials to a /token route in gotrue mode backed by
// authURL, and returns the recorded response. It fails the test when the
// handler does not answer within limit.
func postToken(t *testing.T, authURL string, limit time.Duration) *httptest.ResponseRecorder {
	t.Helper()
	h := loginHelper{router: heligo.New()}
	InitLoginRoute(h, "gotrue", authURL, "secret", 0)
	req := httptest.NewRequest("POST", "/token", strings.NewReader(`{"email":"a@b.c","password":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		h.router.ServeHTTP(rec, req)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(limit):
		t.Fatalf("/token did not answer within %v", limit)
	}
	return rec
}

// The gotrue client had no timeout: an auth server that accepts the
// connection and never answers held the /token request forever.
func TestGotrueTokenTimesOut(t *testing.T) {
	old := loginTimeout
	loginTimeout = 200 * time.Millisecond
	t.Cleanup(func() { loginTimeout = old })

	release := make(chan struct{})
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
	}))
	defer stub.Close()
	defer close(release)

	rec := postToken(t, stub.URL, 3*time.Second)
	if rec.Code == http.StatusOK {
		t.Fatalf("expected an error from a silent auth server, got 200 %s", rec.Body)
	}
	if strings.Contains(rec.Body.String(), stub.URL) {
		t.Errorf("the body names the auth server: %s", rec.Body)
	}
}

// The gotrue reply was read with an unbounded io.ReadAll.
func TestGotrueTokenReplyIsBounded(t *testing.T) {
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"`))
		w.Write([]byte(strings.Repeat("x", 4<<20)))
		w.Write([]byte(`"}`))
	}))
	defer stub.Close()

	rec := postToken(t, stub.URL, 5*time.Second)
	if rec.Code == http.StatusOK {
		t.Fatalf("expected an oversized auth reply to be refused, got 200 (%d bytes)", rec.Body.Len())
	}

	// control: a normal reply goes through
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"access_token":"tok","token_type":"bearer"}`))
	}))
	defer ok.Close()
	rec = postToken(t, ok.URL, 5*time.Second)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"access_token":"tok"`) {
		t.Fatalf("control: expected 200 with the token, got %d %s", rec.Code, rec.Body)
	}
}

// WriteServerError wrote the text of a pgx ConnectError — user, database and
// dial address — whatever VerboseErrors said.
func TestWriteServerErrorHidesConnectError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// nothing listens on port 1: a real ConnectError naming user, db and address
	_, err := pgx.Connect(ctx, "postgresql://leakuser:pw@127.0.0.1:1/leakdb?connect_timeout=2")
	if err == nil {
		t.Fatal("expected the connection to fail")
	}
	leaks := []string{"leakuser", "leakdb", "127.0.0.1"}

	old := verboseErrors
	t.Cleanup(func() { verboseErrors = old })

	// control: verbose errors show the detail, so the probe can see it
	verboseErrors = true
	rec := httptest.NewRecorder()
	WriteServerError(rec, err)
	if !strings.Contains(rec.Body.String(), "leakdb") {
		t.Fatalf("control: expected the verbose body to carry the detail, got %s", rec.Body)
	}

	verboseErrors = false
	rec = httptest.NewRecorder()
	status, _ := WriteServerError(rec, err)
	if status != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", status)
	}
	for _, s := range leaks {
		if strings.Contains(rec.Body.String(), s) {
			t.Errorf("the body contains %q: %s", s, rec.Body)
		}
	}
}
