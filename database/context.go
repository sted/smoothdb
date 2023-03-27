package database

import (
	"context"

	"github.com/gin-gonic/gin"
)

var smoothTag = "gctx"

//type smoothCtxKey struct{}

type SmoothContext struct {
	Db            *Database
	Conn          *DbConn
	Role          string
	RequestParser RequestParser
	QueryBuilder  QueryBuilder
	QueryOptions  *QueryOptions
}

func FillContext(gctx *gin.Context, db *Database, conn *DbConn, role string) {

	defaultParser := PostgRestParser{}
	defaultBuilder := DirectQueryBuilder{}
	queryOptions := defaultParser.getRequestOptions(gctx.Request)
	if queryOptions.Schema == "" {
		queryOptions.Schema = DBE.defaultSchema
	}

	gctx.Set(smoothTag, &SmoothContext{
		db, conn, role,
		defaultParser, defaultBuilder, queryOptions,
	})
}

func GetSmoothContext(ctx context.Context) *SmoothContext {
	v := ctx.Value(smoothTag)
	if v == nil {
		return nil
	}
	return v.(*SmoothContext)
}

func WithDb(parent context.Context, db *Database) (context.Context, *DbPoolConn, error) {
	var conn *DbPoolConn
	var err error
	if db != nil {
		conn, err = db.AcquireConnection(parent)
	} else {
		conn, err = DBE.AcquireConnection(parent)
	}
	if err != nil {
		return nil, nil, err
	}
	return WithDbConn(parent, db, conn.Conn()), conn, nil
}

func WithDbConn(parent context.Context, db *Database, conn *DbConn) context.Context {

	defaultParser := PostgRestParser{}
	queryOptions := &QueryOptions{}
	defaltBuilder := DirectQueryBuilder{}

	//lint:ignore SA1029 should not use built-in type string as key for value; define your own type to avoid collisions
	return context.WithValue(parent, smoothTag,
		&SmoothContext{db, conn, "", defaultParser, defaltBuilder, queryOptions})
}

func GetConn(ctx context.Context) *DbConn {
	gi := GetSmoothContext(ctx)
	return gi.Conn
}

func GetDb(ctx context.Context) *Database {
	gi := GetSmoothContext(ctx)
	return gi.Db
}

func SetRequestParser(ctx context.Context, rp RequestParser) {
	gi := GetSmoothContext(ctx)
	gi.RequestParser = rp
}

func SetQueryBuilder(ctx context.Context, qb QueryBuilder) {
	gi := GetSmoothContext(ctx)
	gi.QueryBuilder = qb
}
