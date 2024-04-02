package server

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/database"
	"github.com/sted/smoothdb/logging"
)

type Server struct {
	Config            *Config
	logger            *logging.Logger
	DBE               *database.DbEngine
	HTTP              *http.Server
	tlsConfig         *tls.Config
	router            *heligo.Router
	sessionManager    *authn.SessionManager
	shutdown          chan struct{}
	shutdownCompleted chan struct{}
	OnBeforeStart     func(*Server)
	OnBeforeShutdown  func(*Server)
}

func NewServer() (*Server, error) {
	return NewServerWithConfig(nil, nil)
}

func NewServerWithConfig(config map[string]any, configOpts *ConfigOptions) (*Server, error) {
	cfg, err := getConfig(config, configOpts)
	if err != nil {
		return nil, err
	}
	err = checkConfig(cfg)
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
		logger:            logger,
		DBE:               dbe,
		shutdown:          make(chan struct{}),
		shutdownCompleted: make(chan struct{}),
	}

	// Initialize session manager
	s.sessionManager = authn.NewSessionManager(logger, s.Config.SessionMode != "none", s.shutdown)

	// Initialize HTTP Server
	s.initHTTPServer()

	return s, nil
}

func (s *Server) Start() error {
	if s.OnBeforeStart != nil {
		s.OnBeforeStart(s)
	}
	err := s.startHTTPServer()
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
		} else if errors.Is(err, syscall.EADDRINUSE) {
			fmt.Printf("EndPoint address already in use. Is there another smoothdb running? (%s)\n", err)
		} else {
			fmt.Println(err.Error())
		}
		os.Exit(1)
	}
}

func (s *Server) GetDBE() *database.DbEngine {
	return s.DBE
}

func (s *Server) GetDatabase(ctx context.Context, name string) (*database.Database, error) {
	return s.DBE.GetActiveDatabase(ctx, name)
}

func (s *Server) GetMainDatabase(ctx context.Context) (*database.Database, error) {
	return s.DBE.GetMainDatabase(ctx)
}

func (s *Server) JWTSecret() string {
	return s.Config.JWTSecret
}

func (s *Server) AllowAnon() bool {
	return s.Config.AllowAnon
}

func (s *Server) AnonRole() string {
	return s.Config.Database.AnonRole
}

func (s *Server) BaseAdminURL() string {
	return s.Config.BaseAdminURL
}

func (s *Server) BaseAPIURL() string {
	return s.Config.BaseAPIURL
}

func (s *Server) HasShortAPIURL() bool {
	return s.Config.ShortAPIURL
}

func (s *Server) RequestMaxBytes() int64 {
	return s.Config.RequestMaxBytes
}

func (s *Server) SessionManager() *authn.SessionManager {
	return s.sessionManager
}

func (s *Server) Logger() *logging.Logger {
	return s.logger
}
