package api

import (
	"context"
	"net/http"

	"github.com/sted/heligo"
)

// InitHealthRoutes registers the health check endpoints
func InitHealthRoutes(apiHelper Helper) {
	router := apiHelper.GetRouter()

	router.Handle("GET", "/live", LiveHandler)
	router.Handle("GET", "/ready", ReadyHandler(apiHelper))
}

// LiveHandler handles the /live endpoint: the process is up. It deliberately
// keeps answering 200 while the server drains — a graceful shutdown must not
// look like a hung process.
func LiveHandler(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
	return heligo.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ReadyHandler returns the handler for the /ready endpoint: route traffic to
// this instance. It flips to 503 as soon as a shutdown begins, so load
// balancers deregister the instance while it is still serving.
func ReadyHandler(apiHelper Helper) heligo.Handler {
	return func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		if apiHelper.IsDraining() {
			return heligo.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "draining"})
		}
		return heligo.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
