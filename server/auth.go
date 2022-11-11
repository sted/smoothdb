package server

import (
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
