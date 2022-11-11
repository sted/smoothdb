package server

import (
	"green/green-ds/database"
	"net/http"
)

type Server struct {
	Config         *Config
	DBE            *database.DBEngine
	HTTP           *http.Server
	sessionManager SessionManager
}

func NewServer() (*Server, error) {
	config := GetConfig("./config.json")

	// DB Engine
	dbe, err := database.InitDBEngine(config.DatabaseURL)
	if err != nil {
		return nil, err
	}

	// Main Server
	server := &Server{Config: config, DBE: dbe}

	// Init HTTP Server
	server.initHTTPServer(dbe)

	return server, nil
}

func (s *Server) Start() error {
	return s.HTTP.ListenAndServe()
}
