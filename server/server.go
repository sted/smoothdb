package server

import (
	"context"
	"green/green-ds/database"
	"green/green-ds/logging"
	"net/http"
)

type Server struct {
	Config            *Config
	Logger            *logging.Logger
	DBE               *database.DbEngine
	HTTP              *http.Server
	sessionManager    SessionManager
	shutdown          chan struct{}
	shutdownCompleted chan struct{}
}

func NewServer() (*Server, error) {
	return NewServerWithConfig(nil, "./config.json")
}

func NewServerWithConfig(c *Config, configPath string) (*Server, error) {
	config := getConfig(c, configPath)

	// Logger
	logger := logging.InitLogger(&config.Logging)

	// DB Engine
	dbe, err := database.InitDbEngine(&config.Database, logger)
	if err != nil {
		return nil, err
	}

	// Main Server
	s := &Server{
		Config:            config,
		Logger:            logger,
		DBE:               dbe,
		shutdown:          make(chan struct{}),
		shutdownCompleted: make(chan struct{}),
	}

	// Initialize session manager
	s.initSessionManager()

	// Initialize HTTP Server
	s.initHTTPServer(dbe)

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
	// Close goroutines (now just the checker in the service manager)
	close(s.shutdown)
	// Close database pools
	s.DBE.Close()
	close(s.shutdownCompleted)
}