package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DbConn = pgxpool.Conn

func HasTX(c *DbConn) bool {
	return c.Conn().PgConn().TxStatus() != 'I'
}

func AcquireConnection(ctx context.Context, db *Database, role string, oldconn *DbConn) (*DbConn, error) {
	conn := oldconn
	if conn == nil {
		if db != nil {
			conn = db.AcquireConnection(ctx)

		} else {
			conn = DBE.AcquireConnection(ctx)
		}
	}
	if conn == nil {
		return nil, errors.New("no available connections")
	}

	// transaction
	needTX := DBE.config.TransactionEnd != "none"
	if needTX {
		_, err := conn.Exec(ctx, "BEGIN")
		if err != nil {
			return nil, err
		}
	}

	// check for oldconn == nil : set role only on the first acquire
	if oldconn == nil && role != "" {
		_, err := conn.Exec(ctx, "SET ROLE "+role)
		if err != nil {
			return nil, err
		}
	}

	return conn, nil
}

func ReleaseConnection(ctx context.Context, conn *DbConn, resetRole bool) error {
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
			gctx := GetSmoothContext(ctx)
			if gctx.QueryOptions.TxRollback {
				end = "ROLLBACK"
			}
		case "rollback":
			end = "ROLLBACK"
		case "rollback-allow-override":
			gctx := GetSmoothContext(ctx)
			if gctx.QueryOptions.TxCommit {
				end = "COMMIT"
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
