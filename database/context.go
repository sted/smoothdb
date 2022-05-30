package database

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
)

type greenCtx struct {
	context.Context
	conn *pgxpool.Conn
	db   *Database
}

func NewContext(c *gin.Context) context.Context {
	ctx := context.Background()
	connvalue := c.MustGet("conn")
	conn := connvalue.(*pgxpool.Conn)
	dbvalue := c.MustGet("db")
	db := dbvalue.(*Database)
	return greenCtx{ctx, conn, db}
}

func NewContextFromDb(db *Database) context.Context {
	ctx := context.Background()
	conn := db.AcquireConnection(ctx)
	return greenCtx{ctx, conn, db}
}

func GetConn(ctx context.Context) *pgxpool.Conn {
	return ctx.(greenCtx).conn
}

func GetDb(ctx context.Context) *Database {
	return ctx.(greenCtx).db
}
