package authn

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// refusal gives the failures of the token checks the code and challenge
// PostgREST answers with: a JWTError keeps its own code, the anonymous-access
// refusals are recognized by their messages, which a change of wording there
// must show up here.
func TestRefusalFollowsPostgREST(t *testing.T) {
	v := newVerifier(t, JWTConfig{Secret: "secret"})
	expired, _ := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{"role": "r", "exp": time.Now().Add(-time.Minute).Unix()}).SignedString([]byte("secret"))
	_, expiredErr := authenticate(expired, v)
	_, malformedErr := authenticate("abc.def", v)
	if expiredErr == nil || malformedErr == nil {
		t.Fatal("the token checks accepted a bad token")
	}
	tests := []struct {
		name, token string
		err         error
		code        string
		challenge   string
	}{
		{"no token", "", errors.New("unauthorized access"), "PGRST302", "Bearer"},
		{"no key", "t", ErrJWTSecretMissing, "PGRST301", tokenChallenge(ErrJWTSecretMissing.Error())},
		{"anonymous access disabled", "t", errors.New(anonDisabled), "PGRST302", "Bearer"},
		{"expired", "t", expiredErr, "PGRST303", tokenChallenge(expiredErr.Error())},
		{"malformed", "t", malformedErr, "PGRST301", tokenChallenge(malformedErr.Error())},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var aerr *Error
			if !errors.As(refusal(tt.token, http.StatusUnauthorized, tt.err), &aerr) {
				t.Fatalf("not an *Error")
			}
			if aerr.Status != http.StatusUnauthorized || aerr.Code != tt.code || aerr.Challenge != tt.challenge {
				t.Errorf("got %d %s %q, want 401 %s %q", aerr.Status, aerr.Code, aerr.Challenge, tt.code, tt.challenge)
			}
		})
	}

	// an empty role claim is the 22023 PostgreSQL raises for that role
	var pgerr *pgconn.PgError
	if !errors.As(refusal("t", http.StatusUnauthorized, errors.New(`role "" does not exist`)), &pgerr) || pgerr.Code != "22023" {
		t.Errorf("empty role: want a 22023 PgError")
	}
}

func TestIsAuthenticatedSeesThroughWrappers(t *testing.T) {
	rec := httptest.NewRecorder()
	if IsAuthenticated(rec) || IsAuthenticated(WithAuthenticated(rec, false)) {
		t.Fatal("an unmarked writer is authenticated")
	}
	if !IsAuthenticated(wrapped{WithAuthenticated(rec, true)}) {
		t.Fatal("a marked writer below a wrapper is not authenticated")
	}
}

type wrapped struct{ http.ResponseWriter }

func (w wrapped) Unwrap() http.ResponseWriter { return w.ResponseWriter }
