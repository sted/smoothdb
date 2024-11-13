package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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

func InitLoginRoute(apiHelper Helper, loginMode string, authURL string, jwtSecret string) {
	api := apiHelper.GetRouter()

	api.Handle("POST", "/token", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		var credentials Credentials
		var resp AccessTokenResponse

		err := r.ReadJSON(&credentials)
		if err != nil {
			return WriteBadRequest(w, err)
		}
		switch loginMode {
		case "db":
			err = database.VerifyAuthN(credentials.Email, credentials.Password)
			if err == nil {
				token, _ := authn.GenerateToken(credentials.Email, jwtSecret)
				resp = AccessTokenResponse{Token: token}
				return heligo.WriteJSON(w, http.StatusOK, resp)
			} else {
				return heligo.WriteJSON(w, http.StatusBadRequest, SmoothError{Subsystem: "auth", Message: err.Error()})
			}
		case "gotrue":
			var errorMessage any

			url := authURL
			if !strings.HasSuffix(url, "/") {
				url += "/"
			}
			url += "token?grant_type=password"

			status, err := postJSON(url, credentials, &resp, &errorMessage)
			if err == nil && status < 400 {
				return heligo.WriteJSON(w, http.StatusOK, resp)
			} else {
				var message string
				if status == 0 {
					status = http.StatusBadRequest
				}
				if err != nil {
					message = err.Error()
				} else {
					message = "See details for the Auth / GoTrue error"
				}
				return heligo.WriteJSON(w, status, SmoothError{
					Subsystem: "auth",
					Message:   message,
					Details:   errorMessage})
			}
		default:
			return WriteServerError(w, fmt.Errorf("invalid login mode"))
		}

	})
}
