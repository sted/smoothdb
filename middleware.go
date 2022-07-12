package main

import (
	"green/green-ds/database"

	"github.com/gin-gonic/gin"
)

func internalMiddleware(admin bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var s *Session = nil
		sessionId, _ := c.Cookie("session_id")
		if sessionId == "" {
			s = NewSession()
			c.SetCookie("session_id", s.Id, 0, "", "", true, false)
		} else {
			s = ThisServer.Sessions[sessionId]
		}
		if s == nil {
			return
		}
		database.FillContext(c)
		c.Next()

		database.ReleaseContext(c)
	}
}

func Authenticated() gin.HandlerFunc {
	return internalMiddleware(false)
}

func AdminOnly() gin.HandlerFunc {
	return internalMiddleware(true)
}
