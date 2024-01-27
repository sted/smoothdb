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

// AcquireConnection takes a connection from a database pool.
// If the db parameter is nil, it uses the global engine pool.
func AcquireConnection(ctx context.Context, db *Database) (conn *DbPoolConn, err error) {
	if db != nil {
		conn, err = db.AcquireConnection(ctx)
	} else {
		conn, err = DBE.AcquireConnection(ctx)
	}
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// PrepareConnection starts a transaction if the configuration requires it,
// i.e. if TransactionMode is not equal to 'none'.
// If the role parameter is not empty, it binds the role to the connection.
func PrepareConnection(ctx context.Context, conn *DbPoolConn, role string, claims string, newAcquire bool) error {
	// transaction
	needTX := DBE.config.TransactionMode != "none"
	if needTX {
		_, err := conn.Exec(ctx, "BEGIN")
		if err != nil {
			return err
		}
	}

	// set role and other configurations only on the first acquire
	if newAcquire && role != "" {
		_, err := conn.Exec(ctx, "SET ROLE "+role)
		if err != nil {
			return err
		}

		set := "SELECT set_config($1, $2, false)"
		_, err = conn.Exec(ctx, set, "request.jwt.claims", claims)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReleaseConnection releases a connection to the proper pool.
// It resets its role if requested and closes the transaction, based on the configuration.
func ReleaseConnection(ctx context.Context, conn *DbPoolConn, httpErr bool, resetRole bool) error {
	hasTX := DBE.config.TransactionMode != "none"

	if resetRole && !hasTX {
		_, err := conn.Exec(ctx, "SET ROLE NONE")
		if err != nil {
			return err
		}
	}

	if hasTX {
		var end string

		if !httpErr {
			switch DBE.config.TransactionMode {
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

	conn.Release()
	return nil
}

// ReleaseConn releases a connection to the proper pool.
// It is a simplified version of ReleaseConnection
func ReleaseConn(ctx context.Context, conn *DbPoolConn) error {
	return ReleaseConnection(ctx, conn, false, true)
}
