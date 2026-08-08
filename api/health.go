package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/sted/heligo"
)

// readinessProbeTimeout bounds the database check so a starved pool answers the
// probe instead of hanging on it: an orchestrator that waits is an orchestrator
// that never replaces the instance.
const readinessProbeTimeout = 2 * time.Second

// InitHealthRoutes registers the health check endpoints
func InitHealthRoutes(apiHelper Helper) {
	router := apiHelper.GetRouter()

	router.Handle("GET", "/live", LiveHandler)
	router.Handle("GET", "/ready", ReadyHandler(apiHelper))
}

// LiveHandler handles the /live endpoint: the process is up. It deliberately
// keeps answering 200 while the server drains — a graceful shutdown must not
// look like a hung process — and it never touches the database, so a database
// outage cannot get the process killed.
func LiveHandler(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
	return heligo.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ReadyHandler returns the handler for the /ready endpoint: route traffic to
// this instance. It flips to 503 as soon as a shutdown begins, so load
// balancers deregister the instance while it is still serving, and also when
// the database layer cannot serve a request — an instance whose pool hands out
// no connection answers nothing, and saying "ready" would keep traffic on it.
func ReadyHandler(apiHelper Helper) heligo.Handler {
	return func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		if apiHelper.IsDraining() {
			return heligo.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "draining"})
		}
		if err := probeDatabase(c, apiHelper); err != nil {
			// The cause is already in the log with its own error entry; repeating it
			// here would put connection details in a response anyone can request.
			return heligo.WriteJSON(w, http.StatusServiceUnavailable,
				map[string]string{"status": "unavailable", "reason": "database"})
		}
		return heligo.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// probeDatabase answers "can this instance serve a request right now": it takes
// a connection from the main pool, which fails when the pool is exhausted, and
// pings it, which fails when the backend is gone.
func probeDatabase(c context.Context, apiHelper Helper) error {
	ctx, cancel := context.WithTimeout(c, readinessProbeTimeout)
	defer cancel()

	db, err := apiHelper.GetMainDatabase(ctx)
	if err != nil {
		return err
	}
	if db == nil {
		return errors.New("no main database")
	}
	conn, err := db.AcquireConnection(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	return conn.Ping(ctx)
}
