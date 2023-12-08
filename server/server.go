package server

import (
	"context"
	"net/http"

	"github.com/sted/smoothdb/database"
	"github.com/sted/smoothdb/logging"
)

type Server struct {
	Config            *Config
	Logger            *logging.Logger
	DBE               *database.DbEngine
	HTTP              *http.Server
	sessionManager    *SessionManager
	shutdown          chan struct{}
	shutdownCompleted chan struct{}
}

func NewServer() (*Server, error) {
	return NewServerWithConfig(nil, nil)
}

func NewServerWithConfig(config map[string]any, configOpts *ConfigOptions) (*Server, error) {
	cfg := getConfig(config, configOpts)

	// Logger
	logger := logging.InitLogger(&cfg.Logging)

	// DB Engine
	dbe, err := database.InitDbEngine(&cfg.Database, logger)
	if err != nil {
		return nil, err
	}

	// Main Server
	s := &Server{
		Config:            cfg,
		Logger:            logger,
		DBE:               dbe,
		shutdown:          make(chan struct{}),
		shutdownCompleted: make(chan struct{}),
	}

	// Initialize session manager
	s.initSessionManager()

	// Initialize HTTP Server
	s.initHTTPServer()

	return s, nil
}

func (s *Server) Start() error {
	err := s.HTTP.ListenAndServe()
	if err == http.ErrServerClosed {
		// wait for graceful shutdown
		<-s.shutdownCompleted
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) {
	// HTTP server shutdown
	s.HTTP.Shutdown(ctx)
	// Close goroutines (for now just the checker in the service manager)
	close(s.shutdown)
	// Close database pools - ok, not so graceful
	// s.DBE.Close() @@ to be fixed, now blocks
	close(s.shutdownCompleted)
}
