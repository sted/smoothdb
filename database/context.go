package database

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

var smoothTag = "gctx"

//type smoothCtxKey struct{}

type SmoothContext struct {
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

	gctx.Set(smoothTag, &SmoothContext{
		gctx,
		db, conn, role,
		defaultParser, defaultBuilder, queryOptions,
	})

	return conn, nil
}

// It is used for testing
func ReleaseContext(ctx context.Context) {
	gi := GetSmoothContext(ctx)
	ReleaseConnection(ctx, gi.Conn, true)
}

func GetSmoothContext(ctx context.Context) *SmoothContext {
	v := ctx.Value(smoothTag)
	if v == nil {
		return nil
	}
	return v.(*SmoothContext)
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
	return context.WithValue(parent, smoothTag,
		&SmoothContext{nil, db, conn, "", defaultParser, defaltBuilder, queryOptions})
}

func GetConn(ctx context.Context) *pgxpool.Conn {
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
