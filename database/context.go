package database

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
)

var greenTag = "gctx"

type greenCtxKey struct{}

type GreenInfo struct {
	GinContext    *gin.Context
	Conn          *pgxpool.Conn
	Db            *Database
	RequestParser RequestParser
	QueryBuilder  QueryBuilder
}

func defaultGreenInfo(gctx *gin.Context, conn *pgxpool.Conn, db *Database) *GreenInfo {
	return &GreenInfo{gctx, conn, db, PostgRestParser{}, DirectQueryBuilder{}}
}

func AcquireContext(gctx *gin.Context) {
	var db *Database
	var err error
	var conn *pgxpool.Conn
	dbname := gctx.Param("dbname")
	if dbname != "" {
		db, err = DBE.GetDatabase(gctx, dbname)
		if err != nil {
			panic("")
		}
		conn = db.AcquireConnection(gctx.Request.Context())
	} else {
		conn = DBE.AcquireConnection(gctx.Request.Context())
	}
	gctx.Set(greenTag, defaultGreenInfo(gctx, conn, db))
}

func WithGreenInfo(parent context.Context, gctx *gin.Context) context.Context {
	gi, ok := gctx.Get(greenTag)
	if !ok {
		panic("")
	}
	return context.WithValue(parent, greenTag, gi)
}

func GetGreenInfo(ctx context.Context) *GreenInfo {
	return ctx.Value(greenTag).(*GreenInfo)
}

func WithDb(parent context.Context, db *Database) context.Context {
	conn := db.AcquireConnection(parent)
	return context.WithValue(parent, greenTag, defaultGreenInfo(nil, conn, db))
}

func ReleaseContext(ctx context.Context) {
	gi := GetGreenInfo(ctx)
	gi.Conn.Release()
}

func GetConn(ctx context.Context) *pgxpool.Conn {
	gi := GetGreenInfo(ctx)
	return gi.Conn
}

func GetDb(ctx context.Context) *Database {
	gi := GetGreenInfo(ctx)
	return gi.Db
}

func GetRequest(ctx context.Context) *http.Request {
	gi := GetGreenInfo(ctx)
	if gi.GinContext == nil {
		return nil
	}
	return gi.GinContext.Request
}

func SetRequestParser(ctx context.Context, rp RequestParser) {
	gi := GetGreenInfo(ctx)
	gi.RequestParser = rp
}

func SetQueryBuilder(ctx context.Context, qb QueryBuilder) {
	gi := GetGreenInfo(ctx)
	gi.QueryBuilder = qb
}
