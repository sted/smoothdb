package server

import (
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang-jwt/jwt/v4/request"
)

type Claims struct {
	Role string `json:"role"`
	Id   string `json:"id"`
	jwt.RegisteredClaims
}

func extractAuthHeader(req *http.Request) string {
	tokenString, _ := request.AuthorizationHeaderExtractor.ExtractToken(req)
	return tokenString
}

func parseAuthHeader(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, err
	}
}

func GenerateToken(role, secret string) (string, error) {
	claims := &Claims{Role: role}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func (server *Server) authenticate(tokenString string) (*Claims, error) {
	claims, err := parseAuthHeader(tokenString, server.Config.JWTSecret)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
