package server

import (
	"net/http"
	"time"

	"github.com/smoothdb/smoothdb/api"
	"github.com/smoothdb/smoothdb/database"

	"github.com/gin-gonic/gin"
)

func (server *Server) initHTTPServer(dbe *database.DbEngine) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	//router.Use(gin.Logger())
	router.Use(ZeroLogger(server.Logger))

	root := router.Group("/")

	root.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "SmoothDB")
	})
	authMiddleware := server.middleware()

	if server.Config.EnableAdminRoute {
		api.InitAdminRouter(root, dbe, authMiddleware)
	}
	api.InitSourcesRouter(root, authMiddleware)
	api.InitTestRouter(root, dbe)

	server.HTTP = &http.Server{
		Addr:         server.Config.Address,
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
}
