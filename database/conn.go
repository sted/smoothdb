package database

import (
	"context"

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
	conn := GetConn(ctx)
	if conn.PgConn().TxStatus() != 'I' {
		_, err := conn.Exec(ctx, "SET TRANSACTION READ ONLY")
		return err
	}
	return setSessionReadOnly(ctx, conn, true)
}

// AcquireConnection takes a connection from a database pool.
// If the db parameter is nil, it uses the main db pool.
func AcquireConnection(ctx context.Context, db *Database) (conn *DbPoolConn, err error) {
	if db != nil {
		conn, err = db.AcquireConnection(ctx)
	} else {
		conn, err = dbe.AcquireConnection(ctx)
	}
	if err != nil {
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
func PrepareConnection(ctx context.Context, conn *DbPoolConn, role string, claims string, newAcquire bool, readOnly bool) error {
	// transaction
	needTX := dbe.config.TransactionMode != "none"
	if needTX {
		begin := "BEGIN"
		if readOnly {
			begin = "BEGIN READ ONLY"
		}
		_, err := conn.Exec(ctx, begin)
		if err != nil {
			return err
		}
	}

	// set role and other configurations only on the first acquire
	if newAcquire && role != "" {
		// role comes from the JWT claim; quote it as an identifier so it cannot
		// break out of the SET ROLE statement (it is otherwise unvalidated).
		_, err := conn.Exec(ctx, "SET ROLE "+quote(role))
		if err != nil {
			return err
		}

		if needTX {
			set := "SELECT set_config($1, $2, false)"
			_, err = conn.Exec(ctx, set, "request.jwt.claims", claims)
		} else {
			// the access mode travels with the claims, in the same round trip:
			// a connection taken from the pool always starts in the requested one
			set := "SELECT set_config($1, $2, false), set_config($3, $4, false)"
			_, err = conn.Exec(ctx, set, "request.jwt.claims", claims, readOnlyGUC, onOff(readOnly))
		}
		if err != nil {
			return err
		}
	} else if !needTX {
		// a connection kept by its session between requests (or one prepared
		// with no role): flip the setting only if it was left the other way
		err := setSessionReadOnly(ctx, conn.Conn(), readOnly)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReleaseConnection releases a connection to the proper pool.
// It resets its role if requested and closes the transaction, based on the configuration.
func ReleaseConnection(ctx context.Context, conn *DbPoolConn, httpErr bool, resetRole bool) error {
	defer conn.Release()
	hasTX := dbe.config.TransactionMode != "none"

	if resetRole && !hasTX {
		// one round trip: the role reset and, so that the pool never hands out
		// a read-only connection (the schema reload and library users take
		// connections without PrepareConnection), the access mode reset
		_, err := conn.Exec(ctx, "SET ROLE NONE; RESET "+readOnlyGUC)
		if err != nil {
			return err
		}
	}

	if hasTX {
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
