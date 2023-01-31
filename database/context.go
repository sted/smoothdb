package database

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

var greenTag = "gctx"

//type greenCtxKey struct{}

type GreenContext struct {
	GinContext    *gin.Context
	Db            *Database
	Conn          *DbConn
	Role          string
	RequestParser RequestParser
	QueryBuilder  QueryBuilder
	QueryOptions  *QueryOptions
}

func FillContext(gctx *gin.Context, role string, oldconn *DbConn) (*DbConn, error) {
	var db *Database
	var err error
	dbname := gctx.Param("dbname")
	if dbname != "" {
		db, err = DBE.GetDatabase(gctx, dbname)
		if err != nil {
			return nil, err
		}
	}
	conn, err := AcquireConnection(gctx, db, role, oldconn)
	if err != nil {
		return nil, err
	}

	defaultParser := PostgRestParser{}
	defaultBuilder := DirectQueryBuilder{}
	queryOptions := defaultParser.getOptions(gctx.Request)
	if queryOptions.Schema == "" {
		queryOptions.Schema = DBE.defaultSchema
	}

	gctx.Set(greenTag, &GreenContext{
		gctx,
		db, conn, role,
		defaultParser, defaultBuilder, queryOptions,
	})

	return conn, nil
}

// It is used for testing
func ReleaseContext(ctx context.Context) {
	gi := GetGreenContext(ctx)
	ReleaseConnection(ctx, gi.Conn, true)
}

// func WithGreenContext(parent context.Context, gctx *gin.Context) context.Context {
// 	gi, ok := gctx.Get(greenTag)
// 	if !ok {
// 		panic("")
// 	}
// 	return context.WithValue(parent, greenTag, gi)
// }

func GetGreenContext(ctx context.Context) *GreenContext {
	v := ctx.Value(greenTag)
	if v == nil {
		return nil
	}
	return v.(*GreenContext)
}

func WithDb(parent context.Context, db *Database) context.Context {
	var conn *DbConn
	if db != nil {
		conn = db.AcquireConnection(parent)
	} else {
		conn = DBE.AcquireConnection(parent)
	}

	defaultParser := PostgRestParser{}
	queryOptions := &QueryOptions{}
	defaltBuilder := DirectQueryBuilder{}

	//lint:ignore SA1029 should not use built-in type string as key for value; define your own type to avoid collisions
	return context.WithValue(parent, greenTag,
		&GreenContext{nil, db, conn, "", defaultParser, defaltBuilder, queryOptions})
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
