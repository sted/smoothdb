package api

import (
	"context"
	"net/http"

	"github.com/sted/heligo"
)

// InitHealthRoutes registers the health check endpoints
func InitHealthRoutes(apiHelper Helper) {
	router := apiHelper.GetRouter()

	// Register both health check endpoints with the same handler
	router.Handle("GET", "/live", HealthHandler)
	router.Handle("GET", "/ready", HealthHandler)
}

// HealthHandler handles both /live and /ready endpoints
func HealthHandler(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
	// If this handler is executing, the server is up and ready
	return heligo.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
