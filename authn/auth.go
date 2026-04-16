package authn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-jwt/jwt/v5/request"
)

type Claims struct {
	Role string `json:"role"`
	Id   string `json:"id"`
	jwt.RegisteredClaims
	RawClaims string `json:"-"`
}

func (c *Claims) UnmarshalJSON(data []byte) error {
	type Alias Claims
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	c.RawClaims = string(data)
	return nil
}

func extractAuthHeader(req *http.Request) string {
	tokenString, _ := request.AuthorizationHeaderExtractor.ExtractToken(req)
	return tokenString
}

func parseAuthHeader(tokenString string, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != "HS256" {
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

// GenerateToken creates a signed JWT for the given role.
// If expiry > 0, the token will include exp and iat claims.
// If expiry == 0, the token has no expiration (for testing or long-lived tokens).
func GenerateToken(role, secret string, expiry ...time.Duration) (string, error) {
	claims := &Claims{Role: role}
	if len(expiry) > 0 && expiry[0] > 0 {
		now := time.Now()
		claims.RegisteredClaims = jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry[0])),
			IssuedAt:  jwt.NewNumericDate(now),
		}
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func authenticate(tokenString string, jwtSecret string) (*Claims, error) {
	claims, err := parseAuthHeader(tokenString, jwtSecret)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
