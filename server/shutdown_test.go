package server

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/test"
)

// newTestServer builds a server for shutdown tests. The database URL defaults
// to the one used in CI; NewServerWithConfig itself honors SMOOTHDB_DATABASE_URL,
// so a local override just works.
func newTestServer(t *testing.T, config map[string]any) *Server {
	t.Helper()
	base := map[string]any{
		"Database.URL":        "postgresql://postgres:postgres@localhost:5432/postgres",
		"Logging.FileLogging": false,
		"Logging.StdOut":      false,
	}
	for k, v := range config {
		base[k] = v
	}
	s, err := NewServerWithConfig(base, &ConfigOptions{
		ConfigFilePath: filepath.Join(t.TempDir(), "config.jsonc"),
		SkipFlags:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// startTestServer runs the server and its signal handler — with the signal
// channel registered before the listener starts, as in Run() — and waits for
// the HTTP listener to accept connections. The returned channel receives the
// result of Start() when the server stops.
func startTestServer(t *testing.T, s *Server) chan error {
	t.Helper()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	done := make(chan error, 1)
	go func() { done <- s.Start() }()
	go s.stopHandler(c)

	test.WaitForServer("http://" + s.Config.Address)
	return done
}

// waitStopped asserts that the server stops with http.ErrServerClosed (the
// graceful path) within the given timeout.
func waitStopped(t *testing.T, done chan error, timeout time.Duration) {
	t.Helper()
	select {
	case err := <-done:
		if err != http.ErrServerClosed {
			t.Fatalf("expected http.ErrServerClosed, got %v", err)
		}
	case <-time.After(timeout):
		t.Fatal("server did not shut down in time")
	}
}

// A raw SIGTERM — the default stop signal of Docker, Kubernetes and systemd —
// must start the same graceful shutdown as SIGINT/Ctrl-C. Without the handler
// the runtime default applies and the process is terminated immediately
// (which here would kill the whole test binary).
func TestSIGTERMStartsGracefulShutdown(t *testing.T) {
	s := newTestServer(t, map[string]any{"Address": "localhost:8091"})
	done := startTestServer(t, s)

	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	waitStopped(t, done, 10*time.Second)
}

func getStatus(t *testing.T, url string) int {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// During the drain window (DrainDelay) /ready must report 503 so load
// balancers deregister the instance, while /live stays 200 and the listener
// keeps serving; only afterwards does the server stop accepting connections.
func TestReadyReportsDrainingDuringShutdown(t *testing.T) {
	s := newTestServer(t, map[string]any{
		"Address":    "localhost:8093",
		"DrainDelay": int64(2),
	})
	done := startTestServer(t, s)
	base := "http://localhost:8093"

	if got := getStatus(t, base+"/ready"); got != http.StatusOK {
		t.Fatalf("expected 200 from /ready before shutdown, got %d", got)
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	// Readiness flips before the drain sleep starts, so the 503 must show up
	// almost immediately — poll briefly, then check /live in the same breath,
	// well inside the 2s drain window.
	deadline := time.Now().Add(1 * time.Second)
	for getStatus(t, base+"/ready") != http.StatusServiceUnavailable {
		if time.Now().After(deadline) {
			t.Fatal("/ready did not report draining within the drain window")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := getStatus(t, base+"/live"); got != http.StatusOK {
		t.Fatalf("expected 200 from /live while draining, got %d", got)
	}
	waitStopped(t, done, 10*time.Second)
}

// The drain window is for orchestrated stops: an interactive Ctrl-C (SIGINT)
// must skip DrainDelay and shut down immediately.
func TestSIGINTSkipsDrainDelay(t *testing.T) {
	s := newTestServer(t, map[string]any{
		"Address":    "localhost:8094",
		"DrainDelay": int64(30),
	})
	done := startTestServer(t, s)

	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}
	waitStopped(t, done, 5*time.Second)
}

// A second signal during the drain window must cut it short and proceed with
// the graceful shutdown right away.
func TestSecondSignalSkipsDrainWindow(t *testing.T) {
	s := newTestServer(t, map[string]any{
		"Address":    "localhost:8095",
		"DrainDelay": int64(30),
	})
	done := startTestServer(t, s)

	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	// Wait until the handler has consumed the first signal (readiness flips
	// right after) — a second signal sent while the first still sits in the
	// channel's one-slot buffer would be dropped, not queued.
	deadline := time.Now().Add(1 * time.Second)
	for getStatus(t, "http://localhost:8095/ready") != http.StatusServiceUnavailable {
		if time.Now().After(deadline) {
			t.Fatal("/ready did not report draining")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	waitStopped(t, done, 5*time.Second)
}

// Shutdown must wait for in-flight requests up to GracefulShutdownTimeout.
// With the previous hardcoded 1s window this request (2s) was cut off with a
// connection reset instead of completing.
func TestShutdownWaitsForInflightRequests(t *testing.T) {
	s := newTestServer(t, map[string]any{
		"Address":                 "localhost:8092",
		"GracefulShutdownTimeout": int64(10),
	})
	s.GetRouter().Handle("GET", "/slow", func(ctx context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		time.Sleep(2 * time.Second)
		return heligo.WriteJSON(w, http.StatusOK, map[string]string{"status": "done"})
	})
	done := startTestServer(t, s)

	type result struct {
		status int
		err    error
	}
	resCh := make(chan result, 1)
	go func() {
		resp, err := http.Get("http://localhost:8092/slow")
		if err != nil {
			resCh <- result{0, err}
			return
		}
		resp.Body.Close()
		resCh <- result{resp.StatusCode, nil}
	}()
	// Let the slow request reach its handler before stopping the server.
	time.Sleep(300 * time.Millisecond)
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}

	res := <-resCh
	if res.err != nil {
		t.Fatalf("in-flight request failed during shutdown: %v", res.err)
	}
	if res.status != http.StatusOK {
		t.Fatalf("expected status 200 for in-flight request, got %d", res.status)
	}
	waitStopped(t, done, 10*time.Second)
}
