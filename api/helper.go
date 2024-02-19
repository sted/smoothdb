package api

import (
	"context"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/database"
)

type Helper interface {
	GetDBE() *database.DbEngine
	GetDatabase(context.Context, string) (*database.Database, error)

	Router() *heligo.Router
	MiddlewareStd() heligo.Middleware
	MiddlewareDBE() heligo.Middleware
	MiddlewareWithDbName(string) heligo.Middleware

	BaseAdminURL() string
	BaseAPIURL() string
	HasShortAPIURL() bool

	SessionStatistics() authn.SessionStatistics
}
