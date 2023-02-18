package server

import (
	"errors"
	"net/http"

	"github.com/smoothdb/smoothdb/database"

	"github.com/gin-gonic/gin"
)

func (server *Server) middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var session *Session
		var conn *database.DbConn
		var err error

		tokenString := extractAuthHeader(ctx.Request)

		sessionId, _ := ctx.Cookie("session_id")
		if sessionId == "" {
			// no previous session

			session = server.authenticate(ctx, tokenString)
		} else {
			// we have a previous session id

			session = server.sessionManager.getSession(sessionId)
			if session == nil {
				// session not found, try to reauthenticate

				session = server.authenticate(ctx, tokenString)
			} else if session.Token != tokenString {
				ctx.AbortWithError(http.StatusUnauthorized, errors.New("jwt mismatch"))
				session = nil
			}
		}
		if session != nil {
			conn, err = database.FillContext(ctx, session.Role, session.DbConn)
			if err != nil {
				ctx.AbortWithError(http.StatusInternalServerError, err)
			} else {
				// Cache the database connection
				session.DbConn = conn
			}
		}

		ctx.Next()

		if session != nil {
			if database.HasTX(conn) {
				database.ReleaseConnection(ctx, conn, false)
				session.DbConn = nil
			}
			server.sessionManager.leaveSession(session)
		}

		// Error handling
		if len(ctx.Errors) > 0 {
			ctx.JSON(-1, gin.H{"error": ctx.Errors[0].Error()})
		}
	}
}
