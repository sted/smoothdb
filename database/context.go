package database

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
)

type greenContextTag int

var green_tag greenContextTag = 0

type GreenContext struct {
	Conn              *pgxpool.Conn
	Db                *Database
	QueryStringParser QueryStringParser
	QueryBuilder      QueryBuilder
}

func initDefaultGreenContext(conn *pgxpool.Conn, db *Database) *GreenContext {
	return &GreenContext{conn, db, PostgRestParser{}, DirectQueryBuilder{}}
}

func NewContext(c *gin.Context) context.Context {
	ctx := context.Background()
	connvalue := c.MustGet("conn")
	conn := connvalue.(*pgxpool.Conn)
	dbvalue := c.MustGet("db")
	db := dbvalue.(*Database)
	return context.WithValue(ctx, green_tag, initDefaultGreenContext(conn, db))
}

func GetGreenContext(ctx context.Context) *GreenContext {
	return ctx.Value(green_tag).(*GreenContext)
}

func NewContextForDb(db *Database) context.Context {
	ctx := context.Background()
	conn := db.AcquireConnection(ctx)
	return context.WithValue(ctx, green_tag, initDefaultGreenContext(conn, db))
}

func ReleaseContext(ctx context.Context) {
	gctx := GetGreenContext(ctx)
	gctx.Conn.Release()
}

func GetConn(ctx context.Context) *pgxpool.Conn {
	gctx := GetGreenContext(ctx)
	return gctx.Conn
}

func GetDb(ctx context.Context) *Database {
	gctx := GetGreenContext(ctx)
	return gctx.Db
}

func SetQueryBuilder(ctx context.Context, qb QueryBuilder) {
	gctx := GetGreenContext(ctx)
	gctx.QueryBuilder = qb
}
