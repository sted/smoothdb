package server

import (
	"errors"
	"net/http"

	"github.com/smoothdb/smoothdb/database"

	"github.com/gin-gonic/gin"
)

func before(ctx *gin.Context, server *Server) *Session {
	var session *Session
	var err error

	tokenString := extractAuthHeader(ctx.Request)

	sessionId, _ := ctx.Cookie("session_id")
	if sessionId == "" {
		// no previous session

		session, err = server.authenticate(tokenString)
		if err != nil {
			ctx.AbortWithError(http.StatusUnauthorized, err)
			return nil
		}

		ctx.SetCookie("session_id", session.Id, 60, "", "", false, true)

	} else {
		// we have a previous session id

		session = server.sessionManager.getSession(sessionId)
		if session == nil {
			// session not found, try to reauthenticate

			session, err = server.authenticate(tokenString)
			if err != nil {
				ctx.AbortWithError(http.StatusUnauthorized, err)
				return nil
			}

			ctx.SetCookie("session_id", session.Id, 60, "", "", false, true)

		} else if session.Token != tokenString {
			ctx.AbortWithError(http.StatusUnauthorized, errors.New("jwt mismatch"))
			return nil
		}
	}

	var db *database.Database
	dbname := ctx.Param("dbname")
	if dbname != "" {
		db, err = database.DBE.GetDatabase(ctx, dbname)
		if err != nil {
			ctx.AbortWithError(http.StatusNotFound, err)
			return nil
		}
	}
	var role string
	if session.DbConn == nil {
		session.DbConn, err = database.AcquireConnection(ctx, db)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return nil
		}
		role = session.Role
	} else {
		// we set the empty string to avoid rebinding the role to the connection
		role = ""
	}
	err = database.PrepareConnection(ctx, session.DbConn, role)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return nil
	}
	database.FillContext(ctx, db, session.DbConn.Conn(), session.Role)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return nil
	}

	return session
}

func after(ctx *gin.Context, server *Server, session *Session) {
	if session == nil {
		return
	}
	if database.HasTX(session.DbConn) {
		database.ReleaseConnection(ctx, session.DbConn, false)
		session.DbConn = nil
	}
	server.sessionManager.leaveSession(session)
}

func (server *Server) middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		session := before(ctx, server)

		ctx.Next()

		after(ctx, server, session)

		// Error handling
		if len(ctx.Errors) > 0 {
			ctx.JSON(-1, gin.H{"error": ctx.Errors[0].Error()})
		}
	}
}
