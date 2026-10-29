package authn

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
	"github.com/sted/smoothdb/logging"
	"github.com/sted/smoothdb/version"
)

type MiddlewareConfig interface {
	GetDatabase(context.Context, string) (*database.Database, error)
	JWTVerifier() *JWTVerifier
	AllowAnon() bool
	AnonRole() string
	RequestMaxBytes() int64
	SessionManager() *SessionManager
	GetLogger() *logging.Logger
	// AuthErrorWriter writes the refusals of the middleware the way the API
	// writes its errors; nil writes a bare {"error": ...}
	AuthErrorWriter() ErrorWriter
}

type GetDatabaseNameFn func(ctx context.Context, r heligo.Request) string

type middleware struct {
	MiddlewareConfig
}

// admit applies the first anonymous-access rule: without a token a request
// is refused outright when anonymous access is off.
func (m middleware) admit(tokenString string) (int, error) {
	if tokenString == "" && !m.AllowAnon() {
		return http.StatusUnauthorized, fmt.Errorf("unauthorized access")
	}
	return http.StatusOK, nil
}

// resolveClaims verifies the token, or builds the anonymous claims for a
// request without one, applying the anonymous-access rules. It is the whole
// authentication of a request: the database middleware runs it when it builds
// a session, the auth-only middleware on every request.
func (m middleware) resolveClaims(tokenString string) (*Claims, int, error) {
	var claims *Claims
	var err error
	if tokenString != "" {
		claims, err = authenticate(tokenString, m.JWTVerifier())
		if err != nil {
			return nil, http.StatusUnauthorized, err
		}
		if !claims.hasRole {
			// A token without a role claim is anonymous to PostgREST: it
			// runs as the anon role, and is refused like a request without
			// a token when anonymous access is off. Left empty, the role
			// would skip the SET ROLE in PrepareConnection and the request
			// would run as the authenticator.
			if !m.AllowAnon() || m.AnonRole() == "" {
				return nil, http.StatusUnauthorized, fmt.Errorf("Anonymous access is disabled")
			}
			claims.setAnonRole(m.AnonRole())
		} else if claims.Role == "" {
			// An empty role claim is a role PostgREST switches to, and
			// PostgreSQL refuses it with this message (22023, a 401).
			return nil, http.StatusUnauthorized, fmt.Errorf(`role "" does not exist`)
		}
	} else {
		// Anonymous request. Like PostgREST (db-anon-role unset →
		// PGRST302 "Anonymous access is disabled"), refuse it when no
		// anonymous role is configured: with an empty role PrepareConnection
		// skips the SET ROLE, so the request would run as the connecting
		// (authenticator) role with the pool's own privileges.
		anonRole := m.AnonRole()
		if anonRole == "" {
			return nil, http.StatusUnauthorized, fmt.Errorf("Anonymous access is disabled")
		}
		// The claims GUC must be set for every request, anonymous included,
		// so policies keying off request.jwt.claims evaluate in context.
		claims = &Claims{}
		claims.setAnonRole(anonRole)
	}
	return claims, http.StatusOK, nil
}

// acquireSession authenticates the request and prepares its session and
// connection. authenticated says whether the request runs as a role other than
// the anonymous one, as PostgREST decides it (App.hs: Just authRole /=
// configDbAnonRole): a token naming the anon role, or without a role claim,
// is anonymous.
func (m middleware) acquireSession(ctx context.Context, r heligo.Request,
	forceDBE bool, getDBName GetDatabaseNameFn) (_ context.Context, _ *Session, authenticated bool, _ error) {
	var claims *Claims
	var err error
	var db *database.Database
	var dbconn *database.DbPoolConn
	var newAcquire bool
	var claimsString string

	tokenString := extractAuthHeader(r.Request)
	if status, err := m.admit(tokenString); err != nil {
		return nil, nil, false, refusal(tokenString, status, err)
	}
	dbname := getDBName(ctx, r)
	keyInput := tokenString + "; "
	if !forceDBE {
		keyInput += dbname
	}
	h := sha256.Sum256([]byte(keyInput))
	key := hex.EncodeToString(h[:])
	session, isNewSession := m.SessionManager().getSession(key)
	// Every failure from here on returns before Middleware can call releaseSession,
	// so this is the only place that can hand back what the call took. A connection
	// left behind is a pool slot lost for good, and a session left in use is never
	// revisited by the session watcher. The way in is ordinary: a client that hangs
	// up during SET ROLE fails PrepareConnection with a canceled context.
	acquired := false
	defer func() {
		if acquired {
			return
		}
		if newAcquire && dbconn != nil {
			dbconn.Release()
			session.DbConn = nil
		}
		if isNewSession {
			// A half-built session (no claims, or no database) must not be found by
			// the next request with the same token: it would skip verification and
			// run with nil claims, or against the main database.
			m.SessionManager().dropSession(session)
		} else {
			m.SessionManager().leaveSession(session)
		}
	}()
	if isNewSession {
		var status int
		claims, status, err = m.resolveClaims(tokenString)
		if err != nil {
			return nil, nil, false, refusal(tokenString, status, err)
		}
		session.Claims = claims
		if dbname != "" && !forceDBE {
			db, err = m.GetDatabase(ctx, dbname)
			if errors.Is(err, database.ErrAcquisitionTimeout) {
				// activating the database read it from a saturated main pool
				return nil, nil, false, &Error{Status: http.StatusGatewayTimeout, Code: "PGRST003",
					Message: err.Error()}
			} else if err != nil {
				return nil, nil, false, &Error{Status: http.StatusNotFound, Message: err.Error(), Err: err}
			}
			session.Db = db
		}
	} else {
		// A hit skips token verification; the expiry is the one thing that
		// verification would have caught since the session was built.
		if claimsExpired(session.Claims, time.Now()) {
			m.SessionManager().dropSession(session)
			return nil, nil, false, refusal(tokenString, http.StatusUnauthorized, ErrJWTExpired)
		}
		db = session.Db
	}
	if session.DbConn == nil || session.DbConn.Conn().PgConn().IsClosed() {
		if session.DbConn != nil {
			// Connection went stale while held by the session — release it
			session.DbConn.Release()
			session.DbConn = nil
		}
		dbconn, err = database.AcquireConnection(ctx, db)
		if errors.Is(err, database.ErrAcquisitionTimeout) {
			// PostgREST's AcquisitionTimeoutUsageError
			return nil, nil, false, &Error{Status: http.StatusGatewayTimeout, Code: "PGRST003",
				Message: err.Error()}
		} else if err != nil {
			// PostgREST's ConnectionUsageError: the database is unreachable
			return nil, nil, false, &Error{Status: http.StatusServiceUnavailable, Code: "PGRST000",
				Message: "Database connection error.", Err: err}
		}
		session.DbConn = dbconn
		newAcquire = true
	} else {
		dbconn = session.DbConn
	}
	claimsString = session.Claims.RawClaims
	anonRole := m.AnonRole()
	authenticated = anonRole == "" || session.Claims.Role != anonRole
	// GET and HEAD are safe methods: like PostgREST, run them READ ONLY, so a
	// function or view that writes fails (25006 -> 405) instead of mutating on
	// a request that proxies, prefetchers and caches replay at will.
	readOnly := r.Method == http.MethodGet || r.Method == http.MethodHead
	err = database.PrepareConnection(ctx, dbconn, session.Claims.Role, claimsString, newAcquire, readOnly)
	if err != nil {
		// a refused role switch (42501) or a role that does not exist (22023)
		// is reported like any database error, by the status of its code
		return nil, nil, authenticated, err
	}
	ctx = database.FillContext(ctx, r.Request, db, dbconn.Conn(), session.Claims.Role)
	acquired = true
	return ctx, session, authenticated, nil
}

func (m middleware) releaseSession(ctx context.Context, status int, session *Session) {
	var err error
	// An error status rolls the transaction back, except a 416: like
	// PostgREST (Response.hs, actionResponse), the 416 of an offset past the
	// total is the answer of a call that ran, and its writes commit. The 416
	// of a Range header refused before execution has nothing to commit.
	httpErr := status >= http.StatusBadRequest && status != http.StatusRequestedRangeNotSatisfiable
	// Release the connection only if:
	// - the sessionmanager is not enabled
	// - the connection has an open transaction
	// Otherwise it will be released in the sessionmanager after a cer
	sm := m.SessionManager()
	if !sm.enabled || session.transient || !sm.retainConnections {
		// no sessions, a transient session (over the cap) or "claims" mode: the
		// connection goes back to the pool at the end of every request
		if session.DbConn != nil {
			err = database.ReleaseConnection(ctx, session.DbConn, httpErr, true)
			session.DbConn = nil
		}
	} else if session.DbConn != nil && database.HasTX(session.DbConn) {
		err = database.ReleaseConnection(ctx, session.DbConn, httpErr, false)
		session.DbConn = nil
	}
	if err != nil {
		m.GetLogger().Err(err).Msg("error releasing database connection")
	}
	m.SessionManager().leaveSession(session)
}

func Middleware(cfg MiddlewareConfig, forceDBE bool, getDBName GetDatabaseNameFn) heligo.Middleware {
	m := middleware{cfg}
	return func(next heligo.Handler) heligo.Handler {
		return func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
			r.Body = LimitBody(w, r.Body, cfg.RequestMaxBytes())
			defer r.Body.Close()
			setServerHeader(w)
			ctx, session, authenticated, err := m.acquireSession(c, r, forceDBE, getDBName)
			if err != nil {
				return m.writeError(w, err, authenticated)
			}
			// the API's error writers see only w: an insufficient privilege
			// is 403 for an authenticated request, 401 otherwise
			w = WithAuthenticated(w, authenticated)
			// deferred so that a panic, recovered outside this middleware, still
			// returns the connection; status stays 500 then, so a transaction
			// left open is rolled back as on any error status
			status := http.StatusInternalServerError
			defer func() { m.releaseSession(ctx, status, session) }()
			//w.(http.Flusher).Flush() // to enable Transfer-Encoding: chunked
			status, err = next(ctx, w, r)
			return status, err
		}
	}
}

// AuthMiddleware authenticates a request like Middleware — the same token
// verification and anonymous-access rules — but takes no session and no
// database connection. It is for routes that never touch the database, such
// as POST /jq: under Middleware each call would hold a pool connection, with
// its SET ROLE, for a handler that does not use it.
func AuthMiddleware(cfg MiddlewareConfig) heligo.Middleware {
	m := middleware{cfg}
	return func(next heligo.Handler) heligo.Handler {
		return func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
			r.Body = LimitBody(w, r.Body, cfg.RequestMaxBytes())
			defer r.Body.Close()
			setServerHeader(w)
			tokenString := extractAuthHeader(r.Request)
			status, err := m.admit(tokenString)
			if err == nil {
				_, status, err = m.resolveClaims(tokenString)
			}
			if err != nil {
				return m.writeError(w, refusal(tokenString, status, err), false)
			}
			return next(c, w, r)
		}
	}
}

func setServerHeader(w http.ResponseWriter) {
	if version.Version != "" {
		w.Header().Set("Server", "smoothdb/"+version.Version)
	} else {
		w.Header().Set("Server", "smoothdb")
	}
}

// LimitBody caps the request body at max bytes. A non-positive max means
// unlimited, as RequestMaxBytes documents: MaxBytesReader(0) would instead
// fail on the first byte and block every request with a body.
func LimitBody(w http.ResponseWriter, body io.ReadCloser, max int64) io.ReadCloser {
	if max <= 0 {
		return body
	}
	return http.MaxBytesReader(w, body, max)
}
