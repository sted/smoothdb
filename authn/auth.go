package authn

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-jwt/jwt/v5/request"
)

type Claims struct {
	Role string `json:"role"`
	Id   string `json:"id"`
	jwt.RegisteredClaims
	RawClaims string `json:"-"`
	// hasRole says whether the role claim key found a value at all: Role
	// alone cannot tell a missing claim, which falls back to the anon role,
	// from an empty one, which is refused.
	hasRole bool
}

// setAnonRole gives claims without a role of their own the anonymous role, as
// PostgREST does (Auth/Jwt.hs, parseClaims: the role claim <|> db-anon-role).
// The request switches to it, and request.jwt.claims carries it inserted into
// the claims (Query/PreQuery.hs): {"role":"<anon>"} for a request without a
// token, the token's claims plus "role" for a token without a role claim. The
// claims are updated in place, so a token's expiry stays checked on the
// session hits.
func (c *Claims) setAnonRole(anonRole string) {
	c.Role = anonRole
	c.RawClaims = insertRole(c.RawClaims, anonRole)
}

// insertRole returns the claims JSON with "role" set to role.
func insertRole(rawClaims string, role string) string {
	var claims map[string]json.RawMessage
	if rawClaims != "" {
		// the payload the token was verified with: a JSON object, or null
		json.Unmarshal([]byte(rawClaims), &claims)
	}
	if claims == nil {
		claims = map[string]json.RawMessage{}
	}
	claims["role"], _ = json.Marshal(role)
	raw, _ := json.Marshal(claims)
	return string(raw)
}

func extractAuthHeader(req *http.Request) string {
	tokenString, _ := request.AuthorizationHeaderExtractor.ExtractToken(req)
	return tokenString
}

// JWTError is a token that failed verification: PostgREST's PGRST301 (the
// token cannot be decoded or no key verifies it) and PGRST303 (its claims are
// invalid), with the same messages (Error.hs, ErrorBody JwtError). Both are a
// 401 for an invalid token, the case for which PostgREST answers with
// WWW-Authenticate: Bearer error="invalid_token", error_description=<Message>.
// Other authentication errors, such as a missing anonymous role, are not
// JWTErrors.
type JWTError struct {
	Code    string
	Message string
	Details string
}

func (e *JWTError) Error() string { return e.Message }

func decodeError(message string, details ...string) *JWTError {
	e := &JWTError{Code: "PGRST301", Message: message}
	if len(details) > 0 {
		e.Details = details[0]
	}
	return e
}

func claimsError(message string) *JWTError {
	return &JWTError{Code: "PGRST303", Message: message}
}

// ErrJWTExpired is the claims error of an expired token.
var ErrJWTExpired = claimsError("JWT expired")

// ErrJWTSecretMissing refuses a bearer token when no key is configured: with
// no secret (possible when LoginMode is "none") a token would be verified
// against the empty HMAC key, which anyone can sign with.
var ErrJWTSecretMissing = errors.New("bearer token refused: no JWTSecret is configured")

// allowedSkew is PostgREST's fixed clock-skew allowance, in seconds, on the
// exp, nbf and iat claims (Auth/Jwt.hs, checkForErrors: allowedSkewSeconds).
const allowedSkew = 30

// JWTConfig is the JWT part of the server configuration.
type JWTConfig struct {
	Secret         string // PostgREST's jwt-secret
	SecretIsBase64 bool   // jwt-secret-is-base64
	Aud            string // jwt-aud, "" for any audience
	RoleClaimKey   string // jwt-role-claim-key, "" for "$.role"
}

// JWTVerifier verifies bearer tokens and maps their claims, as PostgREST's
// Auth/Jwt.hs does.
type JWTVerifier struct {
	secret  []byte
	aud     string
	roleKey *RoleClaimKey
}

// NewJWTVerifier checks the JWT configuration and builds its verifier.
func NewJWTVerifier(cfg JWTConfig) (*JWTVerifier, error) {
	v := &JWTVerifier{aud: cfg.Aud}
	var err error
	if v.roleKey, err = ParseRoleClaimKey(cfg.RoleClaimKey); err != nil {
		return nil, err
	}
	if err = checkAudience(cfg.Aud); err != nil {
		return nil, err
	}
	v.secret = []byte(cfg.Secret)
	if cfg.SecretIsBase64 && cfg.Secret != "" {
		if v.secret, err = decodeBase64Secret(cfg.Secret); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// RoleClaimKey is the parsed role claim key.
func (v *JWTVerifier) RoleClaimKey() *RoleClaimKey { return v.roleKey }

// SigningSecret is the HMAC key that GenerateToken signs with: the secret,
// base64-decoded when JWTSecretIsBase64 is set.
func (v *JWTVerifier) SigningSecret() string { return string(v.secret) }

// decodeBase64Secret decodes a base64 secret as PostgREST does (Config.hs,
// decodeSecret): the URL-safe characters are mapped to the standard ones, "."
// stands for the padding "=", the surrounding blanks are stripped, and the
// rest must be strict base64, padded and on one line.
func decodeBase64Secret(s string) ([]byte, error) {
	s = strings.NewReplacer("_", "/", "-", "+", ".", "=").Replace(s)
	s = strings.TrimSpace(s)
	// the decoder skips line breaks, which PostgREST refuses
	if strings.ContainsAny(s, "\r\n") {
		return nil, fmt.Errorf("invalid base64 JWTSecret: line breaks are not allowed")
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 JWTSecret: %v", err)
	}
	return b, nil
}

// checkAudience applies PostgREST's rule for jwt-aud: a value with a ":" must
// be a URI (Config.hs, optStringOrURI).
func checkAudience(aud string) error {
	if !strings.Contains(aud, ":") {
		return nil
	}
	if !isURI(aud) {
		return fmt.Errorf("invalid 'JWTAud' %q: it should be a string or a valid URI", aud)
	}
	return nil
}

// isURI checks the RFC 3986 URI syntax: a scheme, then characters a URI may
// hold, percent-escapes well formed.
func isURI(s string) bool {
	colon := strings.IndexByte(s, ':')
	if colon < 1 {
		return false
	}
	for i := 0; i < colon; i++ {
		c := s[i]
		alpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		if !(alpha || (i > 0 && ((c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.'))) {
			return false
		}
	}
	rest := s[colon+1:]
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		switch {
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'):
		case strings.IndexByte("-._~:/?#[]@!$&'()*+,;=", c) >= 0:
		case c == '%':
			if i+2 >= len(rest) || !isHex(rest[i+1]) || !isHex(rest[i+2]) {
				return false
			}
			i += 2
		default:
			return false
		}
	}
	return true
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
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

func authenticate(tokenString string, v *JWTVerifier) (*Claims, error) {
	if v == nil || len(v.secret) == 0 {
		return nil, ErrJWTSecretMissing
	}
	return v.verify(tokenString, time.Now())
}

// verify checks the token's signature and claims and maps its role.
func (v *JWTVerifier) verify(tokenString string, now time.Time) (*Claims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, decodeError(fmt.Sprintf("Expected 3 parts in JWT; got %d", len(parts)))
	}
	var header struct {
		Alg string `json:"alg"`
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(headerJSON, &header) != nil {
		return nil, decodeError("JWT cryptographic operation failed")
	}
	method := jwt.GetSigningMethod(header.Alg)
	if method == nil || header.Alg == "none" {
		return nil, decodeError("Wrong or unsupported encoding algorithm", "Unsupported algorithm "+header.Alg)
	}
	if header.Alg != "HS256" {
		return nil, decodeError("No suitable key or wrong key type", "No suitable key was found to decode the JWT")
	}
	// every part decodes before any key is tried, as in PostgREST: a payload
	// that is not base64 fails the decoding, not the signature
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, decodeError("JWT cryptographic operation failed")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, decodeError("JWT cryptographic operation failed")
	}
	if method.Verify(parts[0]+"."+parts[1], sig, v.secret) != nil {
		return nil, decodeError("No suitable key or wrong key type", "None of the keys was able to decode the JWT")
	}
	claims, ok := decodeClaims(payload)
	if !ok {
		return nil, claimsError("Parsing claims failed")
	}
	if err := v.validateClaims(claims, now); err != nil {
		return nil, err
	}
	return v.mapClaims(claims, payload), nil
}

// decodeClaims decodes the payload, which must be a JSON object.
func decodeClaims(payload []byte) (map[string]any, bool) {
	d := json.NewDecoder(strings.NewReader(string(payload)))
	d.UseNumber()
	var claims map[string]any
	if err := d.Decode(&claims); err != nil || claims == nil {
		return nil, false
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, false
	}
	return claims, true
}

// validateClaims checks exp, nbf, iat and aud in this order and reports the
// first failure, as PostgREST's checkForErrors. A claim that is absent or
// null is not checked; the times are allowed a 30 s skew.
func (v *JWTVerifier) validateClaims(claims map[string]any, now time.Time) error {
	t := now.Unix()
	checks := []struct {
		claim, notNumber string
		invalid          func(int64) bool
		err              *JWTError
	}{
		{"exp", "The JWT 'exp' claim must be a number", func(exp int64) bool { return t-allowedSkew > exp }, ErrJWTExpired},
		{"nbf", "The JWT 'nbf' claim must be a number", func(nbf int64) bool { return t+allowedSkew < nbf }, claimsError("JWT not yet valid")},
		{"iat", "The JWT 'iat' claim must be a number", func(iat int64) bool { return t+allowedSkew < iat }, claimsError("JWT issued at future")},
	}
	for _, c := range checks {
		raw := claims[c.claim]
		if raw == nil {
			continue
		}
		n, ok := raw.(json.Number)
		if !ok {
			return claimsError(c.notNumber)
		}
		if c.invalid(claimSeconds(n)) {
			return c.err
		}
	}
	if raw := claims["aud"]; raw != nil {
		notStrings := claimsError("The JWT 'aud' claim must be a string or an array of strings")
		matches := func(aud string) bool { return v.aud == "" || aud == v.aud }
		switch aud := raw.(type) {
		case string:
			if !matches(aud) {
				return claimsError("JWT not in audience")
			}
		case []any:
			found := len(aud) == 0
			for _, a := range aud {
				s, ok := a.(string)
				if !ok {
					return notStrings
				}
				found = found || matches(s)
			}
			if !found {
				return claimsError("JWT not in audience")
			}
		default:
			return notStrings
		}
	}
	return nil
}

// claimSeconds reads a time claim as PostgREST does (Sci.toBoundedInteger,
// defaulting to 0): an integral value that fits an int64, else 0. So a
// fractional or huge exp counts as expired, as it does in PostgREST.
func claimSeconds(n json.Number) int64 {
	if i, err := n.Int64(); err == nil {
		return i
	}
	s := string(n)
	if e := strings.IndexAny(s, "eE"); e >= 0 {
		// bound the exponent before building the exact value
		if exp, err := strconv.Atoi(s[e+1:]); err != nil || exp > 400 || exp < -400 {
			return 0
		}
	}
	if f, err := n.Float64(); err != nil || math.Abs(f) > math.MaxInt64 {
		return 0
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok || !r.IsInt() || !r.Num().IsInt64() {
		return 0
	}
	return r.Num().Int64()
}

// mapClaims builds the request claims: the role found by the role claim key,
// and request.jwt.claims with that role in "role", as PostgREST's PreQuery
// inserts it.
func (v *JWTVerifier) mapClaims(claims map[string]any, payload []byte) *Claims {
	c := &Claims{RawClaims: string(payload)}
	if role, ok := v.roleKey.evaluate(claims); ok {
		c.hasRole = true
		c.Role = roleText(role)
		if s, ok := claims["role"].(string); !ok || s != c.Role {
			c.RawClaims = insertRole(c.RawClaims, c.Role)
		}
	}
	if id, ok := claims["id"].(string); ok {
		c.Id = id
	}
	if exp, ok := claims["exp"].(json.Number); ok {
		c.ExpiresAt = jwt.NewNumericDate(time.Unix(claimSeconds(exp), 0))
	}
	return c
}

// roleText is the role a claim value names: a string as it is, any other
// value as its JSON text (Auth/Jwt.hs, parseClaims: unquoted).
func roleText(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return string(t)
	}
	b, _ := json.Marshal(v)
	return string(b)
}
