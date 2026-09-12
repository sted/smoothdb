package test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The readiness probe must recognise smoothdb, not just a listener: a port
// taken by an unrelated process would otherwise pass the probe and the suite
// would run against the wrong server.
func TestWaitForServerAcceptsSmoothdb(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/live" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer ts.Close()

	if err := WaitForServer(ts.URL); err != nil {
		t.Fatalf("expected the smoothdb answer to pass the probe, got %v", err)
	}
}

func TestWaitForServerRejectsAForeignListener(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "fasthttp")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("<!doctype html><html><body>not smoothdb</body></html>"))
	}))
	defer ts.Close()

	err := WaitForServer(ts.URL)
	if err == nil {
		t.Fatal("expected the probe to reject a 200 from a foreign listener")
	}
	// The error must say who answered: that is what turns a baffling suite
	// failure into "the port is taken".
	for _, want := range []string{"fasthttp", "not smoothdb"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected the error to mention %q, got %v", want, err)
		}
	}
}

// Nothing listening used to be a printed warning and a normal return; the
// suite then ran anyway and failed on every request instead of on the cause.
func TestWaitForServerFailsWhenNothingListens(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()

	if err := WaitForServer("http://" + addr); err == nil {
		t.Fatal("expected an error with nothing listening on the port")
	}
}
