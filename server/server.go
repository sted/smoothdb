package server

import (
	"green/green-ds/database"
	"net/http"
)

var MainServer *Server

type Server struct {
	DBE  *database.DBEngine
	HTTP *http.Server

	CurrentID uint
	Sessions  map[string]*Session
}

func NewServer(addr string, dburl string) (*Server, error) {
	// DB Engine
	dbe, err := database.InitDBEngine(dburl)
	if err != nil {
		return nil, err
	}

	// HTTP Server
	http := InitHTTPServer(addr, dbe)

	MainServer = &Server{dbe, http, 0, map[string]*Session{}}
	return MainServer, nil
}

func (s *Server) Start() error {
	return s.HTTP.ListenAndServe()
}
