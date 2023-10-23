package database

import (
	"context"
	"net/http"
)

type smoothCtxKey struct{}

var smoothTag smoothCtxKey

type SmoothContext struct {
	Db            *Database
	Conn          *DbConn
	Role          string
	RequestParser RequestParser
	QueryBuilder  QueryBuilder
	QueryOptions  *QueryOptions
}

func FillContext(ctx context.Context, r *http.Request, db *Database, conn *DbConn, role string) context.Context {
	defaultParser := PostgRestParser{}
	defaultBuilder := DirectQueryBuilder{}
	queryOptions := defaultParser.getRequestOptions(r)
	if queryOptions.Schema == "" {
		queryOptions.Schema = DBE.defaultSchema
	}
	return context.WithValue(ctx, smoothTag,
		&SmoothContext{db, conn, role, defaultParser, defaultBuilder, queryOptions})
}

func GetSmoothContext(ctx context.Context) *SmoothContext {
	v := ctx.Value(smoothTag)
	if v == nil {
		return nil
	}
	return v.(*SmoothContext)
}

func ContextWithDb(parent context.Context, db *Database, role string) (context.Context, *DbPoolConn, error) {
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
	PrepareConnection(parent, conn, role, "", true)
	return ContextWithDbConn(parent, db, conn.Conn()), conn, nil
}

func ContextWithDbConn(parent context.Context, db *Database, conn *DbConn) context.Context {

	defaultParser := PostgRestParser{}
	queryOptions := &QueryOptions{}
	defaultBuilder := DirectQueryBuilder{}

	return context.WithValue(parent, smoothTag,
		&SmoothContext{db, conn, "", defaultParser, defaultBuilder, queryOptions})
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
