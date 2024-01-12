package server

import (
	"net/http"
	"time"

	"github.com/sted/heligo"
)

func (server *Server) initHTTPServer() {
	server.router = heligo.New()
	//router.Use(gin.Recovery())
	server.router.Use(HTTPLogger(server.Logger))

	if len(server.Config.CORSAllowedOrigins) != 0 {
		server.initCORS()
	}
	if server.Config.EnableAdminRoute {
		server.initAdminRouter()
	}
	if server.Config.EnableAPIRoute {
		server.initSourcesRouter()
	}
	server.initTestRouter()

	server.HTTP = &http.Server{
		Addr:         server.Config.Address,
		Handler:      server.router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
}

func (server *Server) GetRouter() *heligo.Router {
	return server.router
}
