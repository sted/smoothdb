package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
)

// routesHelper is the part of Helper that InitAdminUI and InitTestRouter use.
type routesHelper struct {
	Helper
	router *heligo.Router
	dbe    *database.DbEngine
}

func (h routesHelper) GetRouter() *heligo.Router  { return h.router }
func (h routesHelper) GetDBE() *database.DbEngine { return h.dbe }

// headerCounter counts the status lines a handler writes.
type headerCounter struct {
	*httptest.ResponseRecorder
	headers int
}

func (w *headerCounter) WriteHeader(code int) {
	w.headers++
	w.ResponseRecorder.WriteHeader(code)
}

func (w *headerCounter) Write(b []byte) (int, error) {
	if w.headers == 0 {
		w.headers++
	}
	return w.ResponseRecorder.Write(b)
}

// serveCounting serves req and returns the response, the number of status
// lines written and the status the handler reported to the middleware chain.
func serveCounting(t *testing.T, router *heligo.Router, req *http.Request, reported *int) (*headerCounter, int) {
	t.Helper()
	w := &headerCounter{ResponseRecorder: httptest.NewRecorder()}
	router.ServeHTTP(w, req)
	return w, *reported
}

// GET /ui wrote the redirect, then fell through to ServeFileFS, which wrote
// a second time (a superfluous WriteHeader and index.html after the redirect
// body), and the handler reported 200 for a 302.
func TestUIRedirectWritesOnce(t *testing.T) {
	router := heligo.New()
	var reported int
	router.Use(func(next heligo.Handler) heligo.Handler {
		return func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
			status, err := next(c, w, r)
			reported = status
			return status, err
		}
	})
	InitAdminUI(routesHelper{router: router})

	// control: a UI page is served with index.html
	w, status := serveCounting(t, router, httptest.NewRequest("GET", "/ui/databases", nil), &reported)
	if w.Code != http.StatusOK || status != http.StatusOK || !strings.Contains(strings.ToLower(w.Body.String()), "<html") {
		t.Fatalf("control: expected 200 with index.html, got %d (reported %d)", w.Code, status)
	}

	w, status = serveCounting(t, router, httptest.NewRequest("GET", "/ui", nil), &reported)
	if w.Code != http.StatusFound || w.Header().Get("Location") != defaultUIRoute {
		t.Fatalf("expected a 302 to %s, got %d %q", defaultUIRoute, w.Code, w.Header().Get("Location"))
	}
	if w.headers != 1 {
		t.Errorf("expected one status line, the handler wrote %d", w.headers)
	}
	if strings.Contains(strings.ToLower(w.Body.String()), "<html") {
		t.Errorf("index.html was written after the redirect: %q", w.Body.String())
	}
	if status != http.StatusFound {
		t.Errorf("the handler reported %d for the redirect", status)
	}
}

// /debug/pprof/:cmd sent every name to pprof.Index, which serves the named
// profiles but answers 404 for cmdline, profile, symbol and trace, which have
// handlers of their own.
func TestPprofRoutes(t *testing.T) {
	router := heligo.New()
	InitTestRouter(routesHelper{router: router})
	for _, path := range []string{
		"/debug/pprof/",     // control: the index
		"/debug/pprof/heap", // control: a named profile
		"/debug/pprof/cmdline",
		"/debug/pprof/symbol",
		"/debug/pprof/trace?seconds=0.05",
		"/debug/pprof/profile?seconds=1",
	} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != http.StatusOK {
			t.Errorf("GET %s: expected 200, got %d %s", path, w.Code, strings.TrimSpace(w.Body.String()))
		}
	}
}

// /test/prepare ignored the AcquireConnection error and released the nil
// connection: a nil-pointer panic whenever no connection could be had, as
// for a client that hung up while waiting for one.
func TestPrepareStressTestWithoutConnection(t *testing.T) {
	config := database.DefaultConfig()
	config.URL = "postgresql://postgres:postgres@localhost:5432/postgres"
	config.MinPoolConnections = 1
	config.MaxPoolConnections = 1
	dbe, err := database.InitDbEngine(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer dbe.Close()
	router := heligo.New()
	InitTestRouter(routesHelper{router: router, dbe: dbe})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // the acquire fails at once
	req := httptest.NewRequest("GET", "/test/prepare/x", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("the handler panicked: %v", r)
			}
		}()
		router.ServeHTTP(w, req)
	}()
	if w.Code < 500 {
		t.Errorf("expected a server error without a connection, got %d", w.Code)
	}
}
