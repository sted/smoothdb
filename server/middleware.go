package server

import (
	"context"
	"encoding/json"
	"heligo"
	"net/http"

	"github.com/smoothdb/smoothdb/database"
)

func AcquireSession(ctx context.Context, r heligo.Request, server *Server, forceDBE bool) (context.Context, *Session, int, error) {
	var claims *Claims
	var err error
	var db *database.Database
	var dbconn *database.DbPoolConn
	var newAcquire bool
	var claimsString string

	tokenString := extractAuthHeader(r.Request)
	if tokenString == "" && !server.Config.AllowAnon {
		return nil, nil, http.StatusUnauthorized, err
	}
	dbname := r.Param("dbname")
	key := tokenString + "; "
	if !forceDBE {
		key += dbname
	}
	session, created := server.sessionManager.getSession(key)
	if created {
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
	return ctx, session, 200, nil
}

func ReleaseSession(ctx context.Context, status int, server *Server, session *Session) {
	err := status >= http.StatusBadRequest
	if !server.sessionManager.enabled {
		database.ReleaseConnection(ctx, session.DbConn, err, true)
	} else if database.HasTX(session.DbConn) {
		database.ReleaseConnection(ctx, session.DbConn, err, false)
		session.DbConn = nil
	}
	server.sessionManager.leaveSession(session)
}

func DatabaseMiddleware(server *Server, useDBE bool) heligo.Middleware {
	return func(next heligo.Handler) heligo.Handler {
		return func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
			ctx, session, status, err := AcquireSession(c, r, server, useDBE)
			if err != nil {
				WriteJSON(w, status, Data{"error": err})
				return status, err
			}
			status, err = next(ctx, w, r)
			ReleaseSession(ctx, status, server, session)
			return status, err
		}
	}
}
