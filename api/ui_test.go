package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/sted/heligo"
)

var configScript = regexp.MustCompile(`<script>window\.SMOOTHDB_CONFIG = (.*?);</script>`)

// The UI reads the server configuration from window.SMOOTHDB_CONFIG, which the
// server writes into every page it serves.
func TestUIInjectsConfig(t *testing.T) {
	router := heligo.New()
	cfg := UIConfig{
		LoginMode: "db",
		AllowAnon: false,
		AdminURL:  "/admin",
		APIURL:    "/api",
	}
	InitAdminUI(routesHelper{router: router}, cfg)

	for _, path := range []string{"/ui/databases", "/ui/data/db/table", "/ui/index.html"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", path, w.Code)
		}
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
			t.Errorf("%s: expected text/html, got %q", path, ct)
		}
		body := w.Body.String()
		m := configScript.FindStringSubmatch(body)
		if m == nil {
			t.Fatalf("%s: no SMOOTHDB_CONFIG script in %q", path, body)
		}
		var got UIConfig
		if err := json.Unmarshal([]byte(m[1]), &got); err != nil {
			t.Fatalf("%s: the config is not JSON: %v", path, err)
		}
		if got != cfg {
			t.Errorf("%s: expected %+v, got %+v", path, cfg, got)
		}
		// the config comes before the module script that reads it
		if strings.Index(body, "SMOOTHDB_CONFIG") > strings.Index(body, `type="module"`) {
			t.Errorf("%s: the config script follows the module script", path)
		}
	}
}

// A value from the configuration cannot close the script element.
func TestUIConfigIsEscaped(t *testing.T) {
	router := heligo.New()
	InitAdminUI(routesHelper{router: router}, UIConfig{AdminURL: "</script><script>alert(1)</script>"})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/ui/databases", nil))
	if strings.Contains(w.Body.String(), "alert(1)</script>") {
		t.Errorf("the config value closed the script element: %q", w.Body.String())
	}
}

// The assets are served as they are, without the config.
func TestUIServesAssets(t *testing.T) {
	router := heligo.New()
	InitAdminUI(routesHelper{router: router}, UIConfig{})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/ui/databases", nil))
	js := regexp.MustCompile(`src="/ui/(assets/[^"]+\.js)"`).FindStringSubmatch(w.Body.String())
	if js == nil {
		t.Fatalf("no script asset in the page: %q", w.Body.String())
	}
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/ui/"+js[1], nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for %s, got %d", js[1], w.Code)
	}
	if strings.Contains(w.Body.String(), "SMOOTHDB_CONFIG = {") {
		t.Errorf("the config was written into %s", js[1])
	}
}
