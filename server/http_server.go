package server

import (
	"net/http"
	"time"

	"github.com/sted/heligo"
)

func (server *Server) initHTTPServer() {
	router := heligo.New()
	//router.Use(gin.Recovery())
	router.Use(HTTPLogger(server.Logger))

	// corsConfig := cors.DefaultConfig()
	// corsConfig.AllowAllOrigins = true
	// //config.AllowCredentials = true
	// corsConfig.AllowHeaders = []string{"Authorization", "X-Client-Info", "Accept-Profile"}
	// router.Use(cors.New(corsConfig))
	// root := router.Group("/")

	server.HTTP = &http.Server{
		Addr:         server.Config.Address,
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	if server.Config.EnableAdminRoute {
		server.initAdminRouter()
	}
	if server.Config.EnableAPIRoute {
		server.initSourcesRouter()
	}
	server.initTestRouter()
}

func (server *Server) GetRouter() *heligo.Router {
	return server.HTTP.Handler.(*heligo.Router)
}
