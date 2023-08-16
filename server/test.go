package server

import (
	"context"
	"net/http/pprof"

	"github.com/smoothdb/smoothdb/database"
)

func InitTestRouter(router *Router, dbe *database.DbEngine) {

	router.Handle("GET", "/test/prepare/:test", func(c context.Context, w ResponseWriter, r *Request) {
		//test := c.Param("test")
		conn, _ := dbe.AcquireConnection(c)
		defer conn.Release()

		database.PrepareStressTest(conn)
	})
	router.Handle("GET", "/test/go/:test", func(c context.Context, w ResponseWriter, r *Request) {
		//test := c.Param("test")
		database.StressTest()
	})
	router.Handle("GET", "/test/clean/:test", func(c context.Context, w ResponseWriter, r *Request) {
		//test := c.Param("test")
		database.CleanStressTest()
	})

	// Register pprof handlers
	router.Handle("GET", "/debug/pprof/", Adapt(pprof.Index))
	router.Handle("GET", "/debug/pprof/:cmd", Adapt(pprof.Index))
	// router.Handle("GET", "/debug/pprof/cmdline", Adapt(pprof.Cmdline))
	// router.Handle("GET", "/debug/pprof/profile", Adapt(pprof.Profile))
	// router.Handle("GET", "/debug/pprof/symbol", Adapt(pprof.Symbol))
	// router.Handle("GET", "/debug/pprof/trace", Adapt(pprof.Trace))
}
