package server

import (
	"context"
	"encoding/json"
	"heligo"
	"net/http"

	"github.com/smoothdb/smoothdb/database"
)

func AcquireSession(ctx context.Context, w http.ResponseWriter, r heligo.Request, server *Server, useDBE bool) (context.Context, *Session, error) {

	var claims *Claims
	var err error

	tokenString := extractAuthHeader(r.Request)

	if tokenString != "" {
		claims, err = server.authenticate(tokenString)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return nil, nil, err
		}
	} else {
		tokenString = "anon"
		claims = &Claims{Role: "anon"}
	}
	var db *database.Database
	var dbconn *database.DbPoolConn
	var newAcquire bool
	var claimsString string
	dbname := r.Param("dbname")
	key := tokenString + "; "
	if !useDBE {
		key += dbname
	}
	session, created := server.sessionManager.getSession(key, claims)
	if created {
		if dbname != "" && !useDBE {
			db, err = database.DBE.GetActiveDatabase(ctx, dbname)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return nil, nil, err
			}
			session.Db = db
		}
	} else {
		db = session.Db
	}
	if session.DbConn == nil {
		dbconn, err = database.AcquireConnection(ctx, db)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return nil, nil, err
		}
		session.DbConn = dbconn
		newAcquire = true

		b, _ := json.Marshal(session.Claims)
		claimsString = string(b)
	} else {
		dbconn = session.DbConn
	}
	err = database.PrepareConnection(ctx, dbconn, claims.Role, claimsString, newAcquire)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, nil, err
	}
	ctx = database.FillContext(ctx, r.Request, db, dbconn.Conn(), claims.Role)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return nil, nil, err
	}
	return ctx, session, nil
}

func ReleaseSession(ctx context.Context, status int, server *Server, session *Session) {
	err := status >= 400
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
			ctx, session, err := AcquireSession(c, w, r, server, useDBE)
			if err != nil {
				WriteJSON(w, http.StatusInternalServerError, Data{"error": err})
				return http.StatusInternalServerError, err
			}
			status, err := next(ctx, w, r)
			ReleaseSession(ctx, status, server, session)
			return status, err
		}
	}
}
