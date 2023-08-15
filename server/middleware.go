package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/smoothdb/smoothdb/database"
)

func AcquireSession(w ResponseWriter, r *Request, server *Server, useDBE bool) (context.Context, *Session, error) {

	var claims *Claims
	var err error

	tokenString := extractAuthHeader(r.Request)

	if tokenString != "" {
		claims, err = server.authenticate(tokenString)
		if err != nil {
			w.Status(http.StatusUnauthorized)
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
			db, err = database.DBE.GetActiveDatabase(r.Context(), dbname)
			if err != nil {
				w.Status(http.StatusNotFound)
				return nil, nil, err
			}
			session.Db = db
		}
	} else {
		db = session.Db
	}
	if session.DbConn == nil {
		dbconn, err = database.AcquireConnection(r.Context(), db)
		if err != nil {
			w.Status(http.StatusInternalServerError)
			return nil, nil, err
		}
		session.DbConn = dbconn
		newAcquire = true

		b, _ := json.Marshal(session.Claims)
		claimsString = string(b)
	} else {
		dbconn = session.DbConn
	}
	err = database.PrepareConnection(r.Context(), dbconn, claims.Role, claimsString, newAcquire)
	if err != nil {
		w.Status(http.StatusInternalServerError)
		return nil, nil, err
	}
	ctx := database.FillContext(r.Request, db, dbconn.Conn(), claims.Role)
	if err != nil {
		w.Status(http.StatusInternalServerError)
		return nil, nil, err
	}
	return ctx, session, nil
}

func ReleaseSession(ctx context.Context, server *Server, session *Session) {
	if !server.sessionManager.enabled {
		database.ReleaseConnection(ctx, session.DbConn, true)
	} else if database.HasTX(session.DbConn) {
		database.ReleaseConnection(ctx, session.DbConn, false)
		session.DbConn = nil
	}
	server.sessionManager.leaveSession(session)
}

func DatabaseAccess(w ResponseWriter, r *Request, server *Server, useDBE bool, f Handler) {
	ctx, session, err := AcquireSession(w, r, server, useDBE)
	if err != nil {
		w.JSON(http.StatusInternalServerError, Data{"error": err})
		return
	}
	f(ctx, w, r)
	ReleaseSession(ctx, server, session)
}

// func DatabaseMiddleware(next httprouter.Handle, server *Server, useDBE bool) httprouter.Handle {
// 	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
// 		DatabaseAccess(w, r, params, server, useDBE, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) { next(w, r, params) })
// 	}
// }
