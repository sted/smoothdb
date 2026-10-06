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
	// hasRole says whether the token carries a role claim at all: Role alone
	// cannot tell a missing claim, which falls back to the anon role, from an
	// empty one, which is refused.
	hasRole bool
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
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}
	_, c.hasRole = keys["role"]
	c.RawClaims = string(data)
	return nil
}

// setAnonRole gives claims without a role of their own the anonymous role, as
// PostgREST does (Auth/Jwt.hs, parseClaims: the role claim <|> db-anon-role).
// The request switches to it, and request.jwt.claims carries it inserted into
// the claims (Query/PreQuery.hs): {"role":"<anon>"} for a request without a
// token, the token's claims plus "role" for a token without a role claim. The
// claims are updated in place, so a token's expiry stays checked on the
// session hits.
func (c *Claims) setAnonRole(anonRole string) {
	var claims map[string]json.RawMessage
	if c.RawClaims != "" {
		// the payload the token was verified with: a JSON object, or null
		json.Unmarshal([]byte(c.RawClaims), &claims)
	}
	if claims == nil {
		claims = map[string]json.RawMessage{}
	}
	claims["role"], _ = json.Marshal(anonRole)
	raw, _ := json.Marshal(claims)
	c.Role = anonRole
	c.RawClaims = string(raw)
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
	// With no secret configured (possible when LoginMode is "none") a bearer
	// token would be verified against the empty HMAC key, which anyone can
	// sign with. Fail closed: no secret, no bearer authentication.
	if jwtSecret == "" {
		return nil, fmt.Errorf("bearer token refused: no JWTSecret is configured")
	}
	claims, err := parseAuthHeader(tokenString, jwtSecret)
	if err != nil {
		return nil, err
	}
	return claims, nil
}
