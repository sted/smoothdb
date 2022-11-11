package database

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
)

var greenTag = "gctx"

//type greenCtxKey struct{}

type GreenContext struct {
	GinContext    *gin.Context
	Conn          *pgxpool.Conn
	Db            *Database
	RequestParser RequestParser
	QueryOptions  *QueryOptions
	QueryBuilder  QueryBuilder
}

func AcquireContext(gctx *gin.Context) {
	var db *Database
	var err error
	var conn *pgxpool.Conn
	request := gctx.Request

	dbname := gctx.Param("dbname")
	if dbname != "" {
		db, err = DBE.GetDatabase(gctx, dbname)
		if err != nil {
			panic("")
		}
		conn = db.AcquireConnection(request.Context())
	} else {
		conn = DBE.AcquireConnection(request.Context())
	}

	defaultParser := PostgRestParser{}
	queryOptions := defaultParser.getOptions(request)
	defaltBuilder := DirectQueryBuilder{}

	gctx.Set(greenTag, &GreenContext{gctx, conn, db, defaultParser, queryOptions, defaltBuilder})
}

// func WithGreenContext(parent context.Context, gctx *gin.Context) context.Context {
// 	gi, ok := gctx.Get(greenTag)
// 	if !ok {
// 		panic("")
// 	}
// 	return context.WithValue(parent, greenTag, gi)
// }

func GetGreenContext(ctx context.Context) *GreenContext {
	return ctx.Value(greenTag).(*GreenContext)
}

func WithDb(parent context.Context, db *Database) context.Context {
	conn := db.AcquireConnection(parent)

	defaultParser := PostgRestParser{}
	queryOptions := &QueryOptions{}
	defaltBuilder := DirectQueryBuilder{}

	//lint:ignore SA1029 should not use built-in type string as key for value; define your own type to avoid collisions
	return context.WithValue(parent, greenTag, &GreenContext{nil, conn, db, defaultParser, queryOptions, defaltBuilder})
}

func ReleaseContext(ctx context.Context) {
	gi := GetGreenContext(ctx)
	gi.Conn.Release()
}

func GetConn(ctx context.Context) *pgxpool.Conn {
	gi := GetGreenContext(ctx)
	return gi.Conn
}

func GetDb(ctx context.Context) *Database {
	gi := GetGreenContext(ctx)
	return gi.Db
}

func SetRequestParser(ctx context.Context, rp RequestParser) {
	gi := GetGreenContext(ctx)
	gi.RequestParser = rp
}

func SetQueryBuilder(ctx context.Context, qb QueryBuilder) {
	gi := GetGreenContext(ctx)
	gi.QueryBuilder = qb
}
