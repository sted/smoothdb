package server

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/sted/heligo"
)

func (server *Server) initHTTPServer() {
	server.router = heligo.New()
	//router.Use(gin.Recovery())
	server.router.Use(HTTPLogger(server.Logger))

	cfg := server.Config
	if len(cfg.CORSAllowedOrigins) != 0 {
		server.initCORS()
	}
	if cfg.EnableAdminRoute {
		server.initAdminRouter()
	}
	if cfg.EnableAPIRoute {
		server.initSourcesRouter()
	}
	if cfg.EnableDebugRoute {
		server.initTestRouter()
	}

	if cfg.CertFile != "" {
		// Load certificates
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err == nil {
			server.tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
		} else {
			server.Logger.Warn().Err(err)
		}
	}

	server.HTTP = &http.Server{
		Addr:         cfg.Address,
		Handler:      server.router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
}

func (server *Server) startHTTPServer() error {
	if server.tlsConfig == nil {
		return server.HTTP.ListenAndServe()
	} else {
		server.HTTP.TLSConfig = server.tlsConfig
		return server.HTTP.ListenAndServeTLS("", "")
	}
}

func (server *Server) GetRouter() *heligo.Router {
	return server.router
}
