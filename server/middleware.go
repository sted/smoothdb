package server

import (
	"errors"
	"green/green-ds/database"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func (server *Server) middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var session *Session
		var db *database.Database
		var conn *pgxpool.Conn
		var err error

		tokenString := extractAuthHeader(c.Request)

		sessionId, _ := c.Cookie("session_id")
		if sessionId == "" {
			// no previous session

			session = server.authenticate(c, tokenString)
		} else {
			// we have a previous session id

			session = server.sessionManager.getSession(sessionId)
			if session == nil {
				// session not found, try to reauthenticate

				session = server.authenticate(c, tokenString)
			} else if session.Token != tokenString {
				c.AbortWithError(http.StatusUnauthorized, errors.New("jwt mismatch"))
				session = nil
			}
		}
		if session != nil {
			conn, err = database.FillContext(c, session.Role, session.DbConn)
			if err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
			} else {
				db = database.GetDb(c)
				if db != nil {
					// Cache the database connection (excluding dbe connections)
					session.DbConn = conn
				}
			}
		}

		c.Next()

		server.sessionManager.leaveSession(session)
		if db == nil && conn != nil {
			database.ReleaseConnection(c, conn, false)
		}

		// Error handling
		if len(c.Errors) > 0 {
			c.JSON(-1, gin.H{"error": c.Errors[0].Error()})
		}
	}
}
