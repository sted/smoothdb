package common

import (
	"github.com/sted/heligo"
	"github.com/sted/smoothdb/authn"
)

type APIHelper interface {
	Router() *heligo.Router
	MiddlewareStd() heligo.Middleware
	MiddlewareDBE() heligo.Middleware
	MiddlewareWithDbName(string) heligo.Middleware

	BaseAdminURL() string
	BaseAPIURL() string
	HasShortAPIURL() bool

	SessionStatistics() authn.SessionStatistics
}
