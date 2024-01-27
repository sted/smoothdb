package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
	"github.com/sted/smoothdb/logging"
)

type Server struct {
	Config            *Config
	Logger            *logging.Logger
	DBE               *database.DbEngine
	HTTP              *http.Server
	router            *heligo.Router
	sessionManager    *SessionManager
	shutdown          chan struct{}
	shutdownCompleted chan struct{}
	OnBeforeStart     func(*Server)
	OnBeforeShutdown  func(*Server)
}

func NewServer() (*Server, error) {
	return NewServerWithConfig(nil, nil)
}

func NewServerWithConfig(config map[string]any, configOpts *ConfigOptions) (*Server, error) {
	cfg := getConfig(config, configOpts)
	err := checkConfig(cfg)
	if err != nil {
		return nil, err
	}

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
	s.sessionManager = newSessionManager(logger, s.Config.SessionMode != "none", s.shutdown)

	// Initialize HTTP Server
	s.initHTTPServer()

	return s, nil
}

func (s *Server) Start() error {
	if s.OnBeforeStart != nil {
		s.OnBeforeStart(s)
	}
	err := s.HTTP.ListenAndServe()
	if err == http.ErrServerClosed {
		// wait for graceful shutdown
		<-s.shutdownCompleted
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) {
	if s.OnBeforeShutdown != nil {
		s.OnBeforeShutdown(s)
	}
	// HTTP server shutdown
	s.HTTP.Shutdown(ctx)
	// Close goroutines (for now just the checker in the service manager)
	close(s.shutdown)
	// Close database pools - ok, not so graceful
	// s.DBE.Close() @@ to be fixed, now blocks
	close(s.shutdownCompleted)
}

func (s *Server) stopHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	ctx, cancel := context.WithTimeout(context.Background(), 1000*time.Millisecond)
	defer cancel()
	fmt.Println("\nStarting shutdown...")
	s.Shutdown(ctx)
}

func (s *Server) Run() {
	go s.stopHandler()
	err := s.Start()
	if err != nil {
		if err == http.ErrServerClosed {
			fmt.Println("Stopped.")
		} else {
			fmt.Println(err.Error())
		}
		os.Exit(1)
	}
}
