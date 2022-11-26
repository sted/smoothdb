package server

import (
	"green/green-ds/config"
	"green/green-ds/database"
	"net/http"
)

type Server struct {
	Config         *config.Config
	DBE            *database.DBEngine
	HTTP           *http.Server
	sessionManager SessionManager
}

func NewServer() (*Server, error) {
	config := config.GetConfig("./config.json")

	// DB Engine
	dbe, err := database.InitDBEngine(&config.Database)
	if err != nil {
		return nil, err
	}

	// Main Server
	server := &Server{Config: config, DBE: dbe}

	// Initialize session manager
	server.initSessionManager()

	// Initialize HTTP Server
	server.initHTTPServer(dbe)

	return server, nil
}

func (s *Server) Start() error {
	return s.HTTP.ListenAndServe()
}
