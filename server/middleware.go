package server

import (
	"green/green-ds/database"

	"github.com/gin-gonic/gin"
)

func (server *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var session *Session
		var tokenString string
		var auth *Auth
		var err error

		tokenString = extractAuthHeader(c.Request)

		sessionId, _ := c.Cookie("session_id")
		if sessionId == "" {

			if tokenString == "" {
				if !server.Config.AllowAnon {
					c.AbortWithError(401, err)
					return
				}
			} else {
				auth, err = parseAuthHeader(tokenString, server.Config.JWTSecret)
				if err != nil {
					c.AbortWithError(401, err)
					return
				}
				session = server.sessionManager.NewSession(auth)
				c.SetCookie("session_id", session.Id, 600, "", "", false, true)
			}
		} else {
			session = server.sessionManager.getSession(sessionId)
			if session == nil {
				c.AbortWithError(401, err)
				return
			}
		}

		database.AcquireContext(c)
		c.Next()

		database.ReleaseContext(c)
	}
}
