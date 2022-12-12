package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DbConn = pgxpool.Conn

func AcquireConnection(ctx context.Context, db *Database, role string, oldconn *DbConn) (*DbConn, error) {
	conn := oldconn
	if db != nil {
		if conn == nil {
			conn = db.AcquireConnection(ctx)
		}
	} else {
		conn = DBE.AcquireConnection(ctx)
	}
	if conn == nil {
		return nil, errors.New("no available connections")
	}

	if oldconn == nil && role != "" {
		_, err := conn.Exec(ctx, "SET ROLE "+role)
		if err != nil {
			return nil, err
		}
	}

	return conn, nil
}

func ReleaseConnection(ctx context.Context, conn *DbConn) error {
	_, err := conn.Exec(ctx, "SET ROLE NONE")
	if err != nil {
		return err
	}
	conn.Release()
	return nil
}
