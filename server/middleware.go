package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
)

type GetDatabaseNameFn func(ctx context.Context, r heligo.Request, server *Server) string

func AcquireSession(ctx context.Context, r heligo.Request, server *Server, forceDBE bool, getDBName GetDatabaseNameFn) (context.Context, *Session, int, error) {
	var claims *Claims
	var err error
	var db *database.Database
	var dbconn *database.DbPoolConn
	var newAcquire bool
	var claimsString string

	tokenString := extractAuthHeader(r.Request)
	if tokenString == "" && !server.Config.AllowAnon {
		return nil, nil, http.StatusUnauthorized, fmt.Errorf("unauthorized access")
	}
	dbname := getDBName(ctx, r, server)
	key := tokenString + "; "
	if !forceDBE {
		key += dbname
	}
	session, isNewSession := server.sessionManager.getSession(key)
	if isNewSession {
		if tokenString != "" {
			claims, err = server.authenticate(tokenString)
			if err != nil {
				return nil, nil, http.StatusUnauthorized, err
			}
		} else {
			claims = &Claims{Role: server.Config.Database.AnonRole}
		}
		session.Claims = claims
		if dbname != "" && !forceDBE {
			db, err = database.DBE.GetActiveDatabase(ctx, dbname)
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

		b, _ := json.Marshal(session.Claims)
		claimsString = string(b)
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

func ReleaseSession(ctx context.Context, status int, server *Server, session *Session) {
	var err error
	httpErr := status >= http.StatusBadRequest
	// Release the connection only if:
	// - the sessionmanager is not enabled
	// - the connection has an open transaction
	// Otherwise it will be released in the sessionmanager after a cer
	if !server.sessionManager.enabled {
		err = database.ReleaseConnection(ctx, session.DbConn, httpErr, true)
	} else if database.HasTX(session.DbConn) {
		err = database.ReleaseConnection(ctx, session.DbConn, httpErr, false)
		session.DbConn = nil
	}
	server.Logger.Err(err).Msg("error releasing database connection")
	server.sessionManager.leaveSession(session)
}

func getDatabaseName(ctx context.Context, r heligo.Request, server *Server) string {
	if server.Config.ShortAPIURL {
		return server.Config.Database.AllowedDatabases[0]
	} else {
		return r.Param("dbname")
	}
}

func DatabaseMiddleware(server *Server, forceDBE bool, getDBName GetDatabaseNameFn) heligo.Middleware {
	return func(next heligo.Handler) heligo.Handler {
		return func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
			w.Header().Set("Server", "smoothdb")
			ctx, session, status, err := AcquireSession(c, r, server, forceDBE, getDBName)
			if err != nil {
				WriteJSON(w, status, Data{"error": err.Error()})
				return status, err
			}
			//w.(http.Flusher).Flush() // to enable Transfer-Encoding: chunked
			status, err = next(ctx, w, r)
			ReleaseSession(ctx, status, server, session)
			return status, err
		}
	}
}

func DatabaseMiddlewareStd(server *Server, forceDBE bool) heligo.Middleware {
	return DatabaseMiddleware(server, forceDBE, getDatabaseName)
}

func DatabaseMiddlewareWithName(server *Server, dbname string) heligo.Middleware {
	return DatabaseMiddleware(server, false, func(ctx context.Context, r heligo.Request, server *Server) string {
		return dbname
	})
}
