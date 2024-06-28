package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/ui"
)

const defaultUIRoute = "/ui/databases"

func InitAdminUI(apiHelper Helper) {

	api := apiHelper.GetRouter().Group("/ui")
	content := ui.UIFiles()

	handler := func(ctx context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var r2 *http.Request
		if r.Request.URL.Path == "/ui" {
			r2 = StripPrefix(r.Request, "/ui")
		} else {
			r2 = StripPrefix(r.Request, "/ui/")
		}
		if r2 != nil {
			p := r2.URL.Path
			if p == "" {
				http.Redirect(w, r.Request, defaultUIRoute, http.StatusFound)
			} else if !strings.HasPrefix(p, "assets/") && p != "index.html" {
				p = "index.html"
			}
			http.ServeFileFS(w, r2, content, p)
			return http.StatusOK, nil

		} else {
			return http.StatusNotFound, nil
		}
	}

	api.Handle("GET", "", handler)
	api.Handle("GET", "/", handler)
	api.Handle("GET", "/*path", handler)

	// Redirect home to default UI route when AdminUI is enabled
	router := apiHelper.GetRouter()
	router.Handle("GET", "/", heligo.Adapt(http.RedirectHandler(defaultUIRoute, http.StatusFound)))
}
