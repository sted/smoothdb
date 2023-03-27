package server

import (
	"errors"
	"fmt"
	"net/http"

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

func (server *Server) authenticate(tokenString string) (session *Session, err error) {
	var auth *Auth

	if tokenString != "" {
		// normal authentication

		auth, err = parseAuthHeader(tokenString, server.Config.JWTSecret)
		if err != nil {
			return nil, err
		}
		session = server.sessionManager.newSession(auth.Role)
		session.Token = tokenString
	} else {
		// no jwt, check if we allow anonymous connections

		if server.Config.AllowAnon {
			auth = &Auth{Role: server.Config.Database.AnonRole}
			session = server.sessionManager.newSession(auth.Role)
		} else {
			return nil, errors.New("anonymous users not permitted")
		}
	}
	return
}
