package server

import (
	"net/http"
	"time"

	"github.com/smoothdb/smoothdb/api"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (server *Server) initHTTPServer() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	//router.Use(gin.Logger())
	router.Use(HTTPLogger(server.Logger))

	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	//config.AllowCredentials = true
	config.AllowHeaders = []string{"Authorization", "X-Client-Info", "Accept-Profile"}

	router.Use(cors.New(config))
	root := router.Group("/")
	authMiddleware := server.Middleware()
	if server.Config.EnableAdminRoute {
		api.InitAdminRouter(root, server.DBE, server.Config.BaseAdminURL, authMiddleware)
	}
	api.InitSourcesRouter(root, server.Config.BaseAPIURL, authMiddleware)
	api.InitTestRouter(root, server.DBE)

	server.HTTP = &http.Server{
		Addr:         server.Config.Address,
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
}

func (server *Server) GetRouter() *gin.Engine {
	return server.HTTP.Handler.(*gin.Engine)
}
