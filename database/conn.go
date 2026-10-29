package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DbPoolConn = pgxpool.Conn
type DbConn = pgx.Conn

func HasTX(c *DbPoolConn) bool {
	return c.Conn().PgConn().TxStatus() != 'I'
}

// readOnlyGUC makes every implicit (autocommit) transaction of a session READ
// ONLY: it is how a request runs read-only with TransactionMode "none", where
// there is no BEGIN to qualify. PostgreSQL reports the setting to the client
// whenever it changes (GUC_REPORT since PostgreSQL 14), so the value a
// connection currently has is read from pgconn's parameter statuses: it is the
// server's own word, tracked per physical connection for free, and it cannot
// go stale the way a flag of ours could.
const readOnlyGUC = "default_transaction_read_only"

// txReadOnlyGUC is the access mode of the current transaction alone: set
// transaction-local, it makes a batch's implicit transaction READ ONLY and
// ends with it (PostgreSQL refuses only the way back to read-write once the
// transaction has run a query).
const txReadOnlyGUC = "transaction_read_only"

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// setSessionReadOnly flips the session setting only when the server reports a
// different value (or none: a server before 14 does not report it, so there the
// statement is always sent).
func setSessionReadOnly(ctx context.Context, conn *DbConn, readOnly bool) error {
	want := onOff(readOnly)
	if conn.PgConn().ParameterStatus(readOnlyGUC) == want {
		return nil
	}
	_, err := conn.Exec(ctx, "SET "+readOnlyGUC+" = "+want)
	return err
}

// SetReadOnly makes the rest of the request run READ ONLY: the open transaction
// when there is one, the session's implicit transactions otherwise. It is for a
// request prepared as a write that turns out to be a read (a POST calling a
// STABLE or IMMUTABLE function); the next request on the connection sets the
// mode it needs again through PrepareConnection.
func SetReadOnly(ctx context.Context) error {
	if v := GetSmoothContext(ctx).pending(); v != nil {
		// not on the connection yet: the variables carry it
		v.readOnly = true
		return nil
	}
	conn := GetConn(ctx)
	if conn.PgConn().TxStatus() != 'I' {
		_, err := conn.Exec(ctx, "SET TRANSACTION READ ONLY")
		return err
	}
	return setSessionReadOnly(ctx, conn, true)
}

// ErrAcquisitionTimeout is the error of an acquire that ran out of
// PoolAcquisitionTimeout, with PostgREST's message (PGRST003).
var ErrAcquisitionTimeout = errors.New("Timed out acquiring connection from connection pool.")

// AcquireConnection takes a connection from a database pool.
// If the db parameter is nil, it uses the main db pool.
// The wait is bounded by PoolAcquisitionTimeout (see acquireBounded).
func AcquireConnection(ctx context.Context, db *Database) (conn *DbPoolConn, err error) {
	if db == nil {
		db = dbe.mainDb
	}
	return acquireBounded(ctx, db)
}

// acquireBounded takes a connection from the pool of db, waiting at most
// PoolAcquisitionTimeout, as PostgREST bounds it by
// db-pool-acquisition-timeout: a saturated pool answers instead of holding the
// request until the client gives up. Every acquire on the path of a request
// goes through it.
func acquireBounded(ctx context.Context, db *Database) (*DbPoolConn, error) {
	actx := ctx
	if timeout := dbe.config.PoolAcquisitionTimeout; timeout > 0 {
		var cancel context.CancelFunc
		actx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}
	conn, err := db.AcquireConnection(actx)
	if err != nil {
		if ctx.Err() == nil && errors.Is(actx.Err(), context.DeadlineExceeded) {
			return nil, ErrAcquisitionTimeout
		}
		return nil, err
	}
	return conn, nil
}

// PrepareConnection starts a transaction if the configuration requires it,
// i.e. if TransactionMode is not equal to 'none'.
// If the role parameter is not empty, it binds the role to the connection.
// A read-only request (GET, HEAD) runs READ ONLY as in PostgREST, so that
// whatever it reaches cannot write: the transaction is begun READ ONLY, or,
// with TransactionMode "none", the session is set read-only for it and back
// for the next write on the same connection.
//
// The role's own settings (ALTER ROLE ... SET) apply to the request, as in
// PostgREST, in the statement that switches the role: no extra round trip.
// In a transaction they are transaction-local, so they end with it whatever
// becomes of the connection, and the transaction takes the role's isolation.
// With autocommit there is no transaction to hold them: they are session
// settings, which the connection keeps while its session (same token, so same
// role) retains it, and which ReleaseConnection resets before the pool hands
// it to anyone else.
func PrepareConnection(ctx context.Context, conn *DbPoolConn, role string, claims string, newAcquire bool, readOnly bool) error {
	return prepareConnection(ctx, conn.Conn(), role, claims, newAcquire, readOnly)
}

func prepareConnection(ctx context.Context, conn *DbConn, role string, claims string, newAcquire bool, readOnly bool) error {
	rc := roleConfigOf(role)
	// transaction
	needTX := dbe.config.TransactionMode != "none"
	if needTX {
		begin := "BEGIN"
		if rc != nil && rc.isolation != "" {
			begin += " ISOLATION LEVEL " + rc.isolation
		}
		if readOnly {
			begin += " READ ONLY"
		}
		_, err := conn.Exec(ctx, begin)
		if err != nil {
			return err
		}
	}

	// set role and other configurations only on the first acquire
	if newAcquire && role != "" {
		// One statement: the role's settings, the switch (set_config takes the
		// role from the JWT claim as a value, nothing in it is parsed as SQL),
		// the claims and, with autocommit, the access mode, so that a
		// connection taken from the pool always starts in the requested one.
		// With autocommit the statement is its own transaction: a refused
		// switch rolls back the settings it follows.
		var err error
		switch {
		case needTX && rc != nil:
			_, err = conn.Exec(ctx, rc.sqlTX, append(rc.txArgs[:len(rc.txArgs):len(rc.txArgs)], role, claims)...)
		case needTX:
			_, err = conn.Exec(ctx, prepareTX, role, claims)
		case rc != nil:
			_, err = conn.Exec(ctx, rc.sqlAuto, append(rc.autoArgs[:len(rc.autoArgs):len(rc.autoArgs)], role, claims, onOff(readOnly))...)
		default:
			_, err = conn.Exec(ctx, prepareAuto, role, claims, onOff(readOnly))
		}
		if err != nil {
			return err
		}
		if !needTX {
			// session-level now: reset when the connection goes back to the
			// pool
			markSession(conn, "")
		}
	} else if needTX && rc != nil && rc.sqlTXSettings != "" {
		// a connection kept by its session, already switched: the settings
		// last only as long as each transaction
		_, err := conn.Exec(ctx, rc.sqlTXSettings, rc.txArgs...)
		if err != nil {
			return err
		}
	} else if !needTX {
		// a connection kept by its session between requests (or one prepared
		// with no role): flip the setting only if it was left the other way
		err := setSessionReadOnly(ctx, conn, readOnly)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReleaseConnection releases a connection to the proper pool.
// It resets the session-level role and claims a request left on it, if
// requested, and closes the transaction, based on the configuration.
func ReleaseConnection(ctx context.Context, conn *DbPoolConn, httpErr bool, resetRole bool) error {
	defer conn.Release()
	hasTX := dbe.config.TransactionMode != "none"

	if resetRole {
		if err := resetSession(ctx, conn.Conn()); err != nil {
			return err
		}
	}

	if hasTX && conn.Conn().PgConn().TxStatus() != 'I' {
		var end string

		if !httpErr {
			switch dbe.config.TransactionMode {
			case "commit":
				end = "COMMIT"
			case "commit-allow-override":
				sctx := GetSmoothContext(ctx)
				if sctx != nil && sctx.QueryOptions.TxRollback {
					end = "ROLLBACK"
				} else {
					end = "COMMIT"
				}
			case "rollback":
				end = "ROLLBACK"
			case "rollback-allow-override":
				sctx := GetSmoothContext(ctx)
				if sctx != nil && sctx.QueryOptions.TxCommit {
					end = "COMMIT"
				} else {
					end = "ROLLBACK"
				}
			}
		} else {
			end = "ROLLBACK"
		}

		_, err := conn.Exec(ctx, end)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReleaseConn releases a connection to the proper pool.
// It is a simplified version of ReleaseConnection
func ReleaseConn(ctx context.Context, conn *DbPoolConn) error {
	return ReleaseConnection(ctx, conn, false, true)
}

// sessionMarkKey names, in the custom data of a connection, the mark of a
// role and claims set on it session-level (see markSession).
const sessionMarkKey = "smoothdb.session"

// markSession records that conn carries, session-level, the role, the claims
// and the role settings of a request: key is the session that keeps the
// connection between its requests ("role" mode), "" a request's own, to reset
// when the connection goes back to the pool.
func markSession(conn *DbConn, key string) {
	conn.PgConn().CustomData()[sessionMarkKey] = key
}

// SessionMark says whether conn carries a role and claims session-level, and
// for which session (see markSession).
func SessionMark(conn *DbPoolConn) (key string, marked bool) {
	key, marked = conn.Conn().PgConn().CustomData()[sessionMarkKey].(string)
	return
}

// resetSession takes off conn what a request set on it session-level: the
// role, the claims, the access mode and the role settings, in one round trip
// and only when there is something to reset. The reset runs even when ctx is
// cancelled (a client that hangs up), and a connection it fails on is closed,
// so the pool destroys it instead of handing out a client's role.
func resetSession(ctx context.Context, conn *DbConn) error {
	if _, marked := conn.PgConn().CustomData()[sessionMarkKey]; !marked {
		return nil
	}
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	_, err := conn.Exec(rctx, "SET ROLE NONE; RESET request.jwt.claims; RESET "+readOnlyGUC+roleSettingsReset())
	if err != nil {
		conn.Close(rctx)
		return err
	}
	delete(conn.PgConn().CustomData(), sessionMarkKey)
	return nil
}
