package main

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
)

func internalMiddleware(admin bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		var s *Session = nil
		sessionId, _ := c.Cookie("session_id")
		if sessionId == "" {
			s = NewSession()
			c.SetCookie("session_id", s.Id, 0, "", "", true, false)
		} else {
			s = server.Sessions[sessionId]
		}
		if s == nil {
			return
		}
		var conn *pgxpool.Conn
		dbname := c.Param("dbname")
		db, err := server.DBE.GetDatabase(c, dbname)
		if err != nil {
			return
		}
		c.Set("db", db)
		conn = db.AcquireConnection(context.Background())
		c.Set("conn", conn)

		c.Next()

		conn.Release()
	}
}

func Authenticated() gin.HandlerFunc {
	return internalMiddleware(false)
}

func AdminOnly() gin.HandlerFunc {
	return internalMiddleware(true)
}
