package server

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/api"
)

func (s *Server) initHTTPServer() {
	s.router = heligo.New()
	s.router.TrailingSlash = true
	s.router.Use(heligo.Recover(func(v any) {
		s.logger.Error().Msgf("panic recovered: %v", v)
	}))
	s.router.Use(heligo.CleanPaths())
	s.router.Use(HTTPLogger(s.logger))

	cfg := s.Config
	api.SetVerboseErrors(cfg.VerboseErrors)

	// Security headers
	hasTLS := cfg.CertFile != ""
	s.router.Use(func(next heligo.Handler) heligo.Handler {
		return func(ctx context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			if hasTLS {
				w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}
			return next(ctx, w, r)
		}
	})

	if len(cfg.CORSAllowedOrigins) != 0 {
		s.initCORS()
	}
	if cfg.EnableAdminRoute {
		api.InitAdminRouter(s)

		if cfg.EnableAdminUI {
			api.InitAdminUI(s)
		}
	}
	if cfg.EnableAPIRoute {
		api.InitSourcesRouter(s)
	}
	if cfg.EnableDebugRoute {
		api.InitTestRouter(s)
	}

	if cfg.LoginMode != "none" {
		api.InitLoginRoute(s, cfg.LoginMode, cfg.AuthURL, cfg.JWTSecret, cfg.TokenExpiry)
	}

	if cfg.CertFile != "" {
		// Load certificates
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err == nil {
			s.tlsConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			}
		} else {
			s.logger.Warn().Err(err)
		}
	}

	api.InitHealthRoutes(s)

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
