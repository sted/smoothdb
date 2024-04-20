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
		conn, _ := dbe.AcquireConnection(c)
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
	router.Handle("GET", "/debug/pprof/:cmd", heligo.AdaptFunc(pprof.Index))
	// router.Handle("GET", "/debug/pprof/cmdline", pprof.Cmdline)
	// router.Handle("GET", "/debug/pprof/profile", pprof.Profile)
	// router.Handle("GET", "/debug/pprof/symbol", pprof.Symbol)
	// router.Handle("GET", "/debug/pprof/trace", pprof.Trace)
}
