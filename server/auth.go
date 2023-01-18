package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/golang-jwt/jwt/v4/request"
)

type Auth struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func extractAuthHeader(req *http.Request) string {
	tokenString, _ := request.AuthorizationHeaderExtractor.ExtractToken(req)
	return tokenString
}

func parseAuthHeader(tokenString string, secret string) (*Auth, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Auth{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Auth); ok && token.Valid {
		return claims, nil
	} else {
		return nil, err
	}
}

func GenerateToken(role, secret string) (string, error) {
	auth := &Auth{Role: role}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, auth)
	return token.SignedString([]byte(secret))
}

func (server *Server) authenticate(c *gin.Context, tokenString string) *Session {
	var auth *Auth
	var session *Session
	var err error

	if tokenString != "" {
		// normal authentication

		auth, err = parseAuthHeader(tokenString, server.Config.JWTSecret)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return nil
		}
		session = server.sessionManager.newSession(auth)
		session.Token = tokenString
	} else {
		// no jwt, check if we allow anonymous connections

		if server.Config.AllowAnon {
			auth = &Auth{Role: server.Config.Database.AnonRole}
			session = server.sessionManager.newSession(auth)
		} else {
			c.AbortWithError(http.StatusUnauthorized, errors.New("anonymous users not permitted"))
			return nil
		}
	}
	c.SetCookie("session_id", session.Id, 60, "", "", false, true)

	return session
}
