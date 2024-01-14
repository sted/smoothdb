package database

import (
	"context"
	"net/http"
)

type smoothCtxKey struct{}

var smoothTag = smoothCtxKey{}

// SmoothContext contains all the information needed to perform database commands
// as well as configured parser and query builder
type SmoothContext struct {
	Db            *Database
	Conn          *DbConn
	Role          string
	RequestParser RequestParser
	QueryBuilder  QueryBuilder
	QueryOptions  QueryOptions
}

// FillContext compiles and inserts the information related to the database, cresting a new derived context
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

// GetSmoothContext gets SmoothContext from the standard context
func GetSmoothContext(ctx context.Context) *SmoothContext {
	v := ctx.Value(smoothTag)
	if v == nil {
		return nil
	}
	return v.(*SmoothContext)
}

// ContextWithDb creates a context and gets a connection to the requested database,
// with a specific role. The db parameter can be nil, to indicate the database engine (DBE).
// It is used to start a communication with a database, both internally and from code using
// smoothdb as a library.
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

// ContextWithDb creates a context using a specific connection
func ContextWithDbConn(parent context.Context, db *Database, conn *DbConn) context.Context {
	defaultParser := PostgRestParser{}
	queryOptions := QueryOptions{}
	defaultBuilder := DirectQueryBuilder{}

	return context.WithValue(parent, smoothTag,
		&SmoothContext{db, conn, "", defaultParser, defaultBuilder, queryOptions})
}

// GetConn gets the database connection from the current context
func GetConn(ctx context.Context) *DbConn {
	gi := GetSmoothContext(ctx)
	return gi.Conn
}

// GetDb gets the database from the current context
func GetDb(ctx context.Context) *Database {
	gi := GetSmoothContext(ctx)
	return gi.Db
}

// SetRequestParser allows changing the request parser in the current context
func SetRequestParser(ctx context.Context, rp RequestParser) {
	gi := GetSmoothContext(ctx)
	gi.RequestParser = rp
}

// SetQueryBuilder allows changing the query builder in the current context
func SetQueryBuilder(ctx context.Context, qb QueryBuilder) {
	gi := GetSmoothContext(ctx)
	gi.QueryBuilder = qb
}
