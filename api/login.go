package api

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/database"
)

type Credentials struct {
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type User struct {
	Aud   string `json:"aud"`
	Role  string `json:"role"`
	Email string `json:"email"`
	// to be extended
}

type WeakPasswordError struct {
	Message string   `json:"message,omitempty"`
	Reasons []string `json:"reasons,omitempty"`
}

func (e *WeakPasswordError) Error() string {
	return e.Message
}

type AccessTokenResponse struct {
	Token                string             `json:"access_token"`
	TokenType            string             `json:"token_type"` // Bearer
	ExpiresIn            int                `json:"expires_in"`
	ExpiresAt            int64              `json:"expires_at"`
	RefreshToken         string             `json:"refresh_token"`
	User                 *User              `json:"user"`
	ProviderAccessToken  string             `json:"provider_token,omitempty"`
	ProviderRefreshToken string             `json:"provider_refresh_token,omitempty"`
	WeakPassword         *WeakPasswordError `json:"weak_password,omitempty"`
}

// loginTimeout bounds a login attempt: the database dial of db mode, the
// call to the auth server of gotrue mode.
var loginTimeout = 10 * time.Second

func InitLoginRoute(apiHelper Helper, loginMode string, authURL string, jwtSecret string, tokenExpiry int64) {
	api := apiHelper.GetRouter()
	// /token is reachable without credentials and sits outside MiddlewareStd, so
	// it gets its own per-client rate limit and the same body cap as every
	// other route.
	limiter := newIPLimiter(apiHelper.LoginRateLimit(), time.Now)

	api.Handle("POST", "/token", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var credentials Credentials
		var resp AccessTokenResponse

		if ok, wait := limiter.Allow(clientIP(r.RemoteAddr)); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
			return heligo.WriteJSON(w, http.StatusTooManyRequests,
				SmoothError{Subsystem: "auth", Message: "too many login attempts, retry later"})
		}
		r.Body = authn.LimitBody(w, r.Body, apiHelper.RequestMaxBytes())
		err := r.ReadJSON(&credentials)
		if err != nil {
			return WriteBadRequest(w, err)
		}
		switch loginMode {
		case "db":
			ctx, cancel := context.WithTimeout(c, loginTimeout)
			defer cancel()
			err = database.VerifyAuthN(ctx, credentials.Email, credentials.Password)
			if err == nil {
				var expiry time.Duration
				if tokenExpiry > 0 {
					expiry = time.Duration(tokenExpiry) * time.Second
				}
				token, _ := authn.GenerateToken(credentials.Email, jwtSecret, expiry)
				resp = AccessTokenResponse{Token: token, ExpiresIn: int(tokenExpiry)}
				return heligo.WriteJSON(w, http.StatusOK, resp)
			} else {
				// The error is a pgx ConnectError: it names the database user,
				// the database and the dial address, and carries the server's
				// FATAL text. The caller is unauthenticated: it gets a fixed
				// message, the request log gets the cause.
				heligo.WriteJSON(w, http.StatusBadRequest, SmoothError{Subsystem: "auth", Message: "invalid credentials"})
				return http.StatusBadRequest, err
			}
		case "gotrue":
			var errorMessage any

			url := authURL
			if !strings.HasSuffix(url, "/") {
				url += "/"
			}
			url += "token?grant_type=password"

			ctx, cancel := context.WithTimeout(c, loginTimeout)
			defer cancel()
			status, err := postJSON(ctx, url, credentials, &resp, &errorMessage)
			if err == nil && status < 400 {
				return heligo.WriteJSON(w, http.StatusOK, resp)
			} else {
				var message string
				if status == 0 {
					status = http.StatusBadRequest
				}
				if err != nil {
					// a transport or decoding failure: its text names the
					// auth server, so it goes to the request log only
					message = "authentication service error"
				} else {
					message = "See details for the Auth / GoTrue error"
				}
				heligo.WriteJSON(w, status, SmoothError{
					Subsystem: "auth",
					Message:   message,
					Details:   errorMessage})
				return status, err
			}
		default:
			return WriteServerError(w, fmt.Errorf("invalid login mode"))
		}

	})
}
