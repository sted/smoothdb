package server

import (
	"errors"
	"green/green-ds/database"

	"github.com/gin-gonic/gin"
)

func (server *Server) authenticate(c *gin.Context, tokenString string) *Session {
	var session *Session
	if tokenString != "" {
		// normal authentication

		auth, err := parseAuthHeader(tokenString, server.Config.JWTSecret)
		if err != nil {
			c.AbortWithError(401, err)
		}
		session = server.sessionManager.NewSession(auth)
		c.SetCookie("session_id", session.Id, 600, "", "", false, true)
	} else {
		// no jwt, check if we allow anonymous connections

		if server.Config.AllowAnon {
			session = &Session{}
		} else {
			c.AbortWithError(401, errors.New("anonymous users not permitted"))
		}
	}
	return session
}

func (server *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var session *Session

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
			}
		}
		if session == nil {
			return
		}
		err := database.AcquireContext(c, session.Role)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}

		c.Next()

		database.ReleaseContext(c)
	}
}
