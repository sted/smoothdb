package server

import (
	"encoding/json"
	"net/http"

	"github.com/smoothdb/smoothdb/database"

	"github.com/gin-gonic/gin"
)

func AcquireSession(ctx *gin.Context, server *Server, useDBE bool) *Session {

	var claims *Claims
	var err error

	tokenString := extractAuthHeader(ctx.Request)

	if tokenString != "" {
		claims, err = server.authenticate(tokenString)
		if err != nil {
			ctx.AbortWithError(http.StatusUnauthorized, err)
			return nil
		}
	} else {
		tokenString = "anon"
		claims = &Claims{Role: "anon"}
	}
	var db *database.Database
	var dbconn *database.DbPoolConn
	var newAcquire bool
	var claimsString string
	dbname := ctx.Param("dbname")
	key := tokenString + "; "
	if !useDBE {
		key += dbname
	}
	session, created := server.sessionManager.getSession(key, claims)
	if created {
		if dbname != "" && !useDBE {
			db, err = database.DBE.GetActiveDatabase(ctx, dbname)
			if err != nil {
				ctx.AbortWithError(http.StatusNotFound, err)
				return nil
			}
			session.Db = db
		}
	} else {
		db = session.Db
	}
	if session.DbConn == nil {
		dbconn, err = database.AcquireConnection(ctx, db)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return nil
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
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return nil
	}
	database.FillContext(ctx, db, dbconn.Conn(), claims.Role)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return nil
	}
	return session
}

func ReleaseSession(ctx *gin.Context, server *Server, session *Session) {
	if !server.sessionManager.enabled {
		database.ReleaseConnection(ctx, session.DbConn, true)
	} else if database.HasTX(session.DbConn) {
		database.ReleaseConnection(ctx, session.DbConn, false)
		session.DbConn = nil
	}
	server.sessionManager.leaveSession(session)
}

func (server *Server) DatabaseAccess(ctx *gin.Context, useDBE bool, f func(ctx *gin.Context)) {
	session := AcquireSession(ctx, server, useDBE)
	f(ctx)
	ReleaseSession(ctx, server, session)
	// Error handling
	if len(ctx.Errors) > 0 {
		ctx.JSON(-1, gin.H{"error": ctx.Errors[0].Error()})
	}
}

func (server *Server) DatabaseMiddleware(useDBE bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		server.DatabaseAccess(ctx, useDBE, func(ctx *gin.Context) { ctx.Next() })
	}
}
