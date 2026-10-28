package api

import (
	"context"
	"net/http"
	"net/http/pprof"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
)

func InitTestRouter(api Helper) {

	dbe := api.GetDBE()
	router := api.GetRouter()

	router.Handle("GET", "/test", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		return writeString(w, "text/html; charset=utf-8", "smoothdb at your service", http.StatusOK)
	})

	router.Handle("GET", "/test/prepare/:test", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		//test := c.Param("test")
		conn, err := dbe.AcquireConnection(c)
		if err != nil {
			return WriteServerError(w, err)
		}
		defer conn.Release()
		database.PrepareStressTest(conn)
		return http.StatusOK, nil
	})
	router.Handle("GET", "/test/go/:test", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		//test := c.Param("test")
		database.StressTest()
		return http.StatusOK, nil
	})
	router.Handle("GET", "/test/clean/:test", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		//test := c.Param("test")
		database.CleanStressTest()
		return http.StatusOK, nil
	})

	// Register pprof handlers
	router.Handle("GET", "/debug/pprof/", heligo.AdaptFunc(pprof.Index))
	// pprof.Index serves the named profiles (heap, goroutine, ...); these four
	// have handlers of their own, and Index answers 404 for them
	pprofHandlers := map[string]heligo.Handler{
		"cmdline": heligo.AdaptFunc(pprof.Cmdline),
		"profile": heligo.AdaptFunc(pprof.Profile),
		"symbol":  heligo.AdaptFunc(pprof.Symbol),
		"trace":   heligo.AdaptFunc(pprof.Trace),
	}
	index := heligo.AdaptFunc(pprof.Index)
	router.Handle("GET", "/debug/pprof/:cmd", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		if h, ok := pprofHandlers[r.Param("cmd")]; ok {
			return h(c, w, r)
		}
		return index(c, w, r)
	})
}
