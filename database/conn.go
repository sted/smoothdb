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
// i.e. if TransactionEnd is not equal to 'none'.
// If the role parameter is not empty, it binds the role to the connection.
func PrepareConnection(ctx context.Context, conn *DbPoolConn, role string) error {
	// transaction
	needTX := DBE.config.TransactionEnd != "none"
	if needTX {
		_, err := conn.Exec(ctx, "BEGIN")
		if err != nil {
			return err
		}
	}

	// set role only on the first acquire
	if role != "" {
		_, err := conn.Exec(ctx, "SET ROLE "+role)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReleaseConnection releases a connection to the proper pool.
// It resets its role if requested and closes the transaction, based on the configuration.
func ReleaseConnection(ctx context.Context, conn *DbPoolConn, resetRole bool) error {
	hasTX := DBE.config.TransactionEnd != "none"

	if resetRole && !hasTX {
		_, err := conn.Exec(ctx, "SET ROLE NONE")
		if err != nil {
			return err
		}
	}

	if hasTX {
		var end string
		switch DBE.config.TransactionEnd {
		case "commit":
			end = "COMMIT"
		case "commit-allow-override":
			sctx := GetSmoothContext(ctx)
			if sctx.QueryOptions.TxRollback {
				end = "ROLLBACK"
			} else {
				end = "COMMIT"
			}
		case "rollback":
			end = "ROLLBACK"
		case "rollback-allow-override":
			sctx := GetSmoothContext(ctx)
			if sctx.QueryOptions.TxCommit {
				end = "COMMIT"
			} else {
				end = "ROLLBACK"
			}
		}
		_, err := conn.Exec(ctx, end)
		if err != nil {
			return err
		}
	}

	conn.Release()
	return nil
}
