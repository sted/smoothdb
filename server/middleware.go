package server

import (
	"errors"
	"green/green-ds/database"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (server *Server) middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var session *Session
		//var db *database.Database
		var conn *pgxpool.Conn
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
				//db = database.GetDb(ctx)
				//if db != nil {
				// Cache the database connection (excluding dbe connections)
				session.DbConn = conn
				//}
			}
		}

		ctx.Next()

		if session != nil {
			server.sessionManager.leaveSession(session)
			//if db == nil && conn != nil {
			// 	database.ReleaseConnection(ctx, conn, false)
			// }
		}

		// Error handling
		if len(ctx.Errors) > 0 {
			ctx.JSON(-1, gin.H{"error": ctx.Errors[0].Error()})
		}
	}
}
