package main

import (
	"green/green-ds/database"
	"net/http"
)

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

	return &Server{dbe, http, 0, map[string]*Session{}}, nil
}

func (s *Server) Start() error {
	return s.HTTP.ListenAndServe()
}
