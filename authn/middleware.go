package authn

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	JWTSecret() string
	AllowAnon() bool
	AnonRole() string
	RequestMaxBytes() int64
	SessionManager() *SessionManager
	GetLogger() *logging.Logger
}

type GetDatabaseNameFn func(ctx context.Context, r heligo.Request) string

type middleware struct {
	MiddlewareConfig
}

func (m middleware) acquireSession(ctx context.Context, r heligo.Request,
	forceDBE bool, getDBName GetDatabaseNameFn) (context.Context, *Session, int, error) {
	var claims *Claims
	var err error
	var db *database.Database
	var dbconn *database.DbPoolConn
	var newAcquire bool
	var claimsString string

	tokenString := extractAuthHeader(r.Request)
	if tokenString == "" && !m.AllowAnon() {
		return nil, nil, http.StatusUnauthorized, fmt.Errorf("unauthorized access")
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
		if tokenString != "" {
			claims, err = authenticate(tokenString, m.JWTSecret())
			if err != nil {
				return nil, nil, http.StatusUnauthorized, err
			}
		} else {
			// Anonymous request. Like PostgREST (db-anon-role unset →
			// PGRST302 "Anonymous access is disabled"), refuse it when no
			// anonymous role is configured: with an empty role PrepareConnection
			// skips the SET ROLE, so the request would run as the connecting
			// (authenticator) role with the pool's own privileges.
			anonRole := m.AnonRole()
			if anonRole == "" {
				return nil, nil, http.StatusUnauthorized, fmt.Errorf("Anonymous access is disabled")
			}
			// The claims GUC must be set for every request, anonymous included,
			// so policies keying off request.jwt.claims evaluate in context.
			// PostgREST sets it to the claims with "role" inserted: {"role":"<anon>"}.
			rawClaims, _ := json.Marshal(map[string]string{"role": anonRole})
			claims = &Claims{Role: anonRole, RawClaims: string(rawClaims)}
		}
		session.Claims = claims
		if dbname != "" && !forceDBE {
			db, err = m.GetDatabase(ctx, dbname)
			if err != nil {
				return nil, nil, http.StatusNotFound, err
			}
			session.Db = db
		}
	} else {
		// A hit skips token verification; the expiry is the one thing that
		// verification would have caught since the session was built.
		if claimsExpired(session.Claims, time.Now()) {
			m.SessionManager().dropSession(session)
			return nil, nil, http.StatusUnauthorized, fmt.Errorf("token is expired")
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
		if err != nil {
			return nil, nil, http.StatusInternalServerError, err
		}
		session.DbConn = dbconn
		newAcquire = true
	} else {
		dbconn = session.DbConn
	}
	claimsString = session.Claims.RawClaims
	// GET and HEAD are safe methods: like PostgREST, run them READ ONLY, so a
	// function or view that writes fails (25006 -> 405) instead of mutating on
	// a request that proxies, prefetchers and caches replay at will.
	readOnly := r.Method == http.MethodGet || r.Method == http.MethodHead
	err = database.PrepareConnection(ctx, dbconn, session.Claims.Role, claimsString, newAcquire, readOnly)
	if err != nil {
		return nil, nil, http.StatusInternalServerError, err
	}
	ctx = database.FillContext(ctx, r.Request, db, dbconn.Conn(), session.Claims.Role)
	acquired = true
	return ctx, session, http.StatusOK, nil
}

func (m middleware) releaseSession(ctx context.Context, status int, session *Session) {
	var err error
	httpErr := status >= http.StatusBadRequest
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
			if version.Version != "" {
				w.Header().Set("Server", "smoothdb/"+version.Version)
			} else {
				w.Header().Set("Server", "smoothdb")
			}
			ctx, session, status, err := m.acquireSession(c, r, forceDBE, getDBName)
			if err != nil {
				heligo.WriteJSON(w, status, map[string]string{"error": err.Error()})
				return status, err
			}
			//w.(http.Flusher).Flush() // to enable Transfer-Encoding: chunked
			status, err = next(ctx, w, r)
			m.releaseSession(ctx, status, session)
			return status, err
		}
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
