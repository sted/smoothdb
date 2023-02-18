package api

import (
	"context"
	"net/http/pprof"

	"github.com/smoothdb/smoothdb/database"

	"github.com/gin-gonic/gin"
)

func InitTestRouter(root *gin.RouterGroup, dbe *database.DbEngine) *gin.RouterGroup {

	test := root.Group("/test")

	test.GET("/prepare/:test", func(c *gin.Context) {
		//test := c.Param("test")
		context := context.Background()
		conn := dbe.AcquireConnection(context)
		defer conn.Release()

		database.PrepareStressTest(conn)
	})
	test.GET("/go/:test", func(c *gin.Context) {
		//test := c.Param("test")
		database.StressTest()
	})
	test.GET("/clean/:test", func(c *gin.Context) {
		//test := c.Param("test")
		database.CleanStressTest()
	})

	debug := root.Group("/debug")

	// Register pprof handlers
	debug.GET("/pprof/", gin.WrapF(pprof.Index))
	debug.GET("/pprof/:cmd", gin.WrapF(pprof.Index))
	debug.GET("/pprof/cmdline", gin.WrapF(pprof.Cmdline))
	debug.GET("/pprof/profile", gin.WrapF(pprof.Profile))
	debug.GET("/pprof/symbol", gin.WrapF(pprof.Symbol))
	debug.GET("/pprof/trace", gin.WrapF(pprof.Trace))

	return test
}
