package server

import (
	"flag"
	"green/green-ds/config"
	"green/green-ds/database"
	"green/green-ds/logging"
	"net/http"
	"os"
)

type Server struct {
	Config         *Config
	Logger         *logging.Logger
	DBE            *database.DbEngine
	HTTP           *http.Server
	sessionManager SessionManager
}

func getConfig(filepath string) *Config {
	// Defaults
	c := DefaultConfig()

	// Environment
	dburl := os.Getenv("DATABASE_URL")
	if dburl != "" {
		c.Database.URL = dburl
	}

	// Command line flags
	flag.StringVar(&c.Address, "addr", c.Address, "Address")
	flag.StringVar(&c.Database.URL, "dburl", c.Database.URL, "DatabaseURL")

	c = config.GetConfig(c, filepath)
	flag.Parse()

	return c
}

func NewServer() (*Server, error) {
	config := getConfig("./config.json")

	logger := logging.InitLogger(&config.Logging)

	// DB Engine
	dbe, err := database.InitDbEngine(&config.Database, logger)
	if err != nil {
		return nil, err
	}

	// Main Server
	s := &Server{Config: config, Logger: logger, DBE: dbe}

	// Initialize session manager
	s.initSessionManager()

	// Initialize HTTP Server
	s.initHTTPServer(dbe)

	return s, nil
}

func (s *Server) Start() error {
	return s.HTTP.ListenAndServe()
}
