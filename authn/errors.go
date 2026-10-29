package authn

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sted/heligo"
)

// Error is a refusal the middleware decides itself, before the database has a
// say: a missing or invalid token, an unknown database, a pool that hands out
// no connection. The errors the database raises while the request is prepared
// (the role switch) travel as they are, and get the status of every other
// database error.
type Error struct {
	Status int
	// Code is PostgREST's code for the refusal (PGRST301, ...), "" for none
	Code    string
	Message string
	// Challenge is the WWW-Authenticate value of a 401, "" for none
	Challenge string
	// Details are reported when errors are verbose, else the cause's message
	Details string
	// Err is the cause
	Err error
}

func (e *Error) Error() string { return e.Message }

func (e *Error) Unwrap() error { return e.Err }

// ErrorWriter writes the response of a request the middleware refuses.
// authenticated says whether the request runs as a role other than the
// anonymous one, which decides the status of an insufficient privilege.
type ErrorWriter func(w http.ResponseWriter, err error, authenticated bool) (int, error)

const anonDisabled = "Anonymous access is disabled"

// requiredChallenge and tokenChallenge are PostgREST's WWW-Authenticate
// values (Error.hs, requiredTokenHeader and invalidTokenHeader, where `show`
// quotes the message).
const requiredChallenge = "Bearer"

func tokenChallenge(message string) string {
	return `Bearer error="invalid_token", error_description=` + strconv.Quote(message)
}

// refusal gives an error of the token checks (admit, resolveClaims) the shape
// PostgREST reports it with:
//   - without a token, or with one that falls back to a disabled anonymous
//     role: PGRST302 and the bare challenge (JwtTokenRequired);
//   - an empty role claim: the 22023 PostgreSQL raises for that role, which
//     the database errors map to 401;
//   - a token that fails verification (a JWTError): its code, PGRST303 for
//     its claims (expired, not yet valid, ...), PGRST301 otherwise, with the
//     invalid_token challenge; any other token failure is a PGRST301 too.
func refusal(tokenString string, status int, err error) error {
	msg := err.Error()
	switch {
	case tokenString == "" || msg == anonDisabled:
		return &Error{Status: status, Code: "PGRST302", Message: anonDisabled,
			Challenge: requiredChallenge, Err: err}
	case strings.HasPrefix(msg, "role ") && strings.HasSuffix(msg, " does not exist"):
		return &pgconn.PgError{Severity: "ERROR", Code: "22023", Message: msg}
	}
	var jerr *JWTError
	if errors.As(err, &jerr) {
		return &Error{Status: status, Code: jerr.Code, Message: jerr.Message,
			Details: jerr.Details, Challenge: tokenChallenge(jerr.Message), Err: err}
	}
	return &Error{Status: status, Code: "PGRST301", Message: msg,
		Challenge: tokenChallenge(msg), Err: err}
}

// authenticatedWriter marks the response of an authenticated request, for the
// error writers that see only the ResponseWriter.
type authenticatedWriter struct {
	http.ResponseWriter
}

func (w authenticatedWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// WithAuthenticated marks w as the response of an authenticated request, or
// returns it as it is.
func WithAuthenticated(w http.ResponseWriter, authenticated bool) http.ResponseWriter {
	if !authenticated {
		return w
	}
	return authenticatedWriter{w}
}

// IsAuthenticated reports whether w was marked by WithAuthenticated, here or
// below the writers that wrap it.
func IsAuthenticated(w http.ResponseWriter) bool {
	for {
		switch rw := w.(type) {
		case authenticatedWriter:
			return true
		case interface{ Unwrap() http.ResponseWriter }:
			w = rw.Unwrap()
		default:
			return false
		}
	}
}

// writeError hands err to the configured writer. Without one (a
// MiddlewareConfig of a library user) it writes the status of an Error, 500
// otherwise, and the message.
func (m middleware) writeError(w http.ResponseWriter, err error, authenticated bool) (int, error) {
	if write := m.AuthErrorWriter(); write != nil {
		return write(w, err, authenticated)
	}
	status := http.StatusInternalServerError
	var aerr *Error
	if errors.As(err, &aerr) {
		status = aerr.Status
		if aerr.Challenge != "" {
			w.Header().Set("WWW-Authenticate", aerr.Challenge)
		}
	}
	heligo.WriteJSON(w, status, map[string]string{"error": err.Error()})
	return status, err
}
