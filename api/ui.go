package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/ui"
)

const defaultUIRoute = "/ui/databases"

// UIConfig is the part of the server configuration that the UI reads at start,
// from window.SMOOTHDB_CONFIG.
type UIConfig struct {
	LoginMode   string `json:"loginMode"`
	AllowAnon   bool   `json:"allowAnon"`
	AdminURL    string `json:"adminURL"`
	APIURL      string `json:"apiURL"`
	ShortAPIURL bool   `json:"shortAPIURL"`
}

// indexWithConfig returns index.html with a script that sets
// window.SMOOTHDB_CONFIG, placed before the module scripts in <head>.
func indexWithConfig(content fs.FS, cfg UIConfig) ([]byte, error) {
	index, err := fs.ReadFile(content, "index.html")
	if err != nil {
		return nil, err
	}
	// json.Marshal escapes <, > and &: a value cannot close the script element.
	js, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	script := []byte("<script>window.SMOOTHDB_CONFIG = " + string(js) + ";</script>\n  ")
	at := bytes.Index(index, []byte("<script"))
	if at < 0 {
		at = bytes.Index(index, []byte("</head>"))
	}
	if at < 0 {
		return append(script, index...), nil
	}
	return append(index[:at:at], append(script, index[at:]...)...), nil
}

func InitAdminUI(apiHelper Helper, cfg UIConfig) {

	api := apiHelper.GetRouter().Group("/ui")
	content := ui.UIFiles()

	index, err := indexWithConfig(content, cfg)
	if err != nil {
		panic("ui index.html not found: " + err.Error())
	}

	handler := func(ctx context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var r2 *http.Request
		if r.Request.URL.Path == "/ui" {
			r2 = StripPrefix(r.Request, "/ui")
		} else {
			r2 = StripPrefix(r.Request, "/ui/")
		}
		if r2 == nil {
			return http.StatusNotFound, nil
		}
		p := r2.URL.Path
		if p == "" {
			http.Redirect(w, r.Request, defaultUIRoute, http.StatusFound)
			return http.StatusFound, nil
		}
		if strings.HasPrefix(p, "assets/") {
			http.ServeFileFS(w, r2, content, p)
			return http.StatusOK, nil
		}
		// every other path is a client-side route
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(index)
		return http.StatusOK, nil
	}

	api.Handle("GET", "", handler)
	api.Handle("GET", "/", handler)
	api.Handle("GET", "/*path", handler)

	// Redirect home to default UI route when AdminUI is enabled
	router := apiHelper.GetRouter()
	router.Handle("GET", "/", heligo.Adapt(http.RedirectHandler(defaultUIRoute, http.StatusFound)))
}
