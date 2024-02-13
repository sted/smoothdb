package server

import (
	"context"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/authn"
)

func (s *Server) getDatabaseName(ctx context.Context, r heligo.Request) string {
	if s.Config.ShortAPIURL {
		return s.Config.Database.AllowedDatabases[0]
	} else {
		return r.Param("dbname")
	}
}

func (s *Server) MiddlewareStd() heligo.Middleware {
	return authn.Middleware(s, false, s.getDatabaseName)
}

func (s *Server) MiddlewareDBE() heligo.Middleware {
	return authn.Middleware(s, true, s.getDatabaseName)
}

func (s *Server) MiddlewareWithDbName(dbname string) heligo.Middleware {
	return authn.Middleware(s, false, func(ctx context.Context, r heligo.Request) string {
		return dbname
	})
}

func (s *Server) SessionStatistics() authn.SessionStatistics {
	return s.sessionManager.Statistics()
}
