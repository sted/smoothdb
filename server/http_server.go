package server

import (
	"net/http"
	"time"
)

func (server *Server) initHTTPServer() {
	//gin.SetMode(gin.ReleaseMode)
	//router := gin.New()
	router := NewRouter(server)
	//router.Use(gin.Recovery())
	//router.Use(gin.Logger())
	//router.Use(HTTPLogger(server.Logger))

	// corsConfig := cors.DefaultConfig()
	// corsConfig.AllowAllOrigins = true
	// //config.AllowCredentials = true
	// corsConfig.AllowHeaders = []string{"Authorization", "X-Client-Info", "Accept-Profile"}

	// router.Use(cors.New(corsConfig))
	// root := router.Group("/")

	if server.Config.EnableAdminRoute {
		InitAdminRouter(router, server.DBE, server.Config.BaseAdminURL)
	}
	InitSourcesRouter(router, server.Config.BaseAPIURL)
	InitTestRouter(router, server.DBE)

	server.HTTP = &http.Server{
		Addr:         server.Config.Address,
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
}

func (server *Server) GetRouter() *http.Handler {
	return &server.HTTP.Handler
}
