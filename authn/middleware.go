package authn

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
	"github.com/sted/smoothdb/logging"
)

type MiddlewareConfig interface {
	GetDatabase(context.Context, string) (*database.Database, error)
	JWTSecret() string
	AllowAnon() bool
	AnonRole() string
	RequestMaxBytes() int64
	SessionManager() *SessionManager
	GetLogger() *logging.Logger
}

type GetDatabaseNameFn func(ctx context.Context, r heligo.Request) string

type middleware struct {
	MiddlewareConfig
}

func (m middleware) acquireSession(ctx context.Context, r heligo.Request,
	forceDBE bool, getDBName GetDatabaseNameFn) (context.Context, *Session, int, error) {
	var claims *Claims
	var err error
	var db *database.Database
	var dbconn *database.DbPoolConn
	var newAcquire bool
	var claimsString string

	tokenString := extractAuthHeader(r.Request)
	if tokenString == "" && !m.AllowAnon() {
		return nil, nil, http.StatusUnauthorized, fmt.Errorf("unauthorized access")
	}
	dbname := getDBName(ctx, r)
	key := tokenString + "; "
	if !forceDBE {
		key += dbname
	}
	session, isNewSession := m.SessionManager().getSession(key)
	if isNewSession {
		if tokenString != "" {
			claims, err = authenticate(tokenString, m.JWTSecret())
			if err != nil {
				return nil, nil, http.StatusUnauthorized, err
			}
		} else {
			claims = &Claims{Role: m.AnonRole()}
		}
		session.Claims = claims
		if dbname != "" && !forceDBE {
			db, err = m.GetDatabase(ctx, dbname)
			if err != nil {
				return nil, nil, http.StatusNotFound, err
			}
			session.Db = db
		}
	} else {
		db = session.Db
	}
	if session.DbConn == nil {
		dbconn, err = database.AcquireConnection(ctx, db)
		if err != nil {
			return nil, nil, http.StatusInternalServerError, err
		}
		session.DbConn = dbconn
		newAcquire = true

		claimsString = session.Claims.RawClaims
	} else {
		dbconn = session.DbConn
	}
	err = database.PrepareConnection(ctx, dbconn, session.Claims.Role, claimsString, newAcquire)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, err
	}
	ctx = database.FillContext(ctx, r.Request, db, dbconn.Conn(), session.Claims.Role)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, err
	}
	return ctx, session, http.StatusOK, nil
}

func (m middleware) releaseSession(ctx context.Context, status int, session *Session) {
	var err error
	httpErr := status >= http.StatusBadRequest
	// Release the connection only if:
	// - the sessionmanager is not enabled
	// - the connection has an open transaction
	// Otherwise it will be released in the sessionmanager after a cer
	if !m.SessionManager().enabled {
		err = database.ReleaseConnection(ctx, session.DbConn, httpErr, true)
	} else if database.HasTX(session.DbConn) {
		err = database.ReleaseConnection(ctx, session.DbConn, httpErr, false)
		session.DbConn = nil
	}
	if err != nil {
		m.GetLogger().Err(err).Msg("error releasing database connection")
	}
	m.SessionManager().leaveSession(session)
}

func Middleware(cfg MiddlewareConfig, forceDBE bool, getDBName GetDatabaseNameFn) heligo.Middleware {
	m := middleware{cfg}
	return func(next heligo.Handler) heligo.Handler {
		return func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
			r.Body = http.MaxBytesReader(w, r.Body, cfg.RequestMaxBytes())
			defer r.Body.Close()
			w.Header().Set("Server", "smoothdb")
			ctx, session, status, err := m.acquireSession(c, r, forceDBE, getDBName)
			if err != nil {
				heligo.WriteJSON(w, status, map[string]string{"error": err.Error()})
				return status, err
			}
			//w.(http.Flusher).Flush() // to enable Transfer-Encoding: chunked
			status, err = next(ctx, w, r)
			m.releaseSession(ctx, status, session)
			return status, err
		}
	}
}
