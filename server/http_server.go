package server

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/api"
)

func (s *Server) initHTTPServer() {
	s.router = heligo.New()
	//router.Use(gin.Recovery())
	s.router.Use(HTTPLogger(s.logger))

	cfg := s.Config
	if len(cfg.CORSAllowedOrigins) != 0 {
		s.initCORS()
	}
	if cfg.EnableAdminRoute {
		api.InitAdminRouter(s)
	}
	if cfg.EnableAPIRoute {
		api.InitSourcesRouter(s)
	}
	if cfg.EnableDebugRoute {
		api.InitTestRouter(s)
	}

	if cfg.CertFile != "" {
		// Load certificates
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err == nil {
			s.tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
		} else {
			s.logger.Warn().Err(err)
		}
	}

	s.HTTP = &http.Server{
		Addr:         cfg.Address,
		Handler:      s.router,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
	}
}

func (s *Server) startHTTPServer() error {
	if s.tlsConfig == nil {
		return s.HTTP.ListenAndServe()
	} else {
		s.HTTP.TLSConfig = s.tlsConfig
		return s.HTTP.ListenAndServeTLS("", "")
	}
}

func (s *Server) GetRouter() *heligo.Router {
	return s.router
}
