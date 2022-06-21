package database

import (
	"context"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4/pgxpool"
)

// DBEngine represents a database instance (a "cluster")
type DBEngine struct {
	connString string
	pool       *pgxpool.Pool
	databases  map[string]*Database
	exec       *QueryExecutor
}

// InitDBEngine creates a connection pool, connects to the engine and initializes it
func InitDBEngine(connString string) (*DBEngine, error) {
	context := context.Background()
	pool, err := pgxpool.Connect(context, connString)
	if err != nil {
		return nil, err
	}
	dbe := &DBEngine{connString, pool, map[string]*Database{}, &QueryExecutor{}}
	return dbe, nil
}

func (dbe *DBEngine) AcquireConnection(ctx context.Context) *pgxpool.Conn {
	conn, _ := dbe.pool.Acquire(ctx)
	return conn
}

func checkDatabase(ctx context.Context) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS _sources (
		_id		SERIAL PRIMARY KEY, 
		_name	TEXT UNIQUE,
		_check	TEXT NOT NULL DEFAULT ''
		)`)
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS _fields (
		_id		SERIAL PRIMARY KEY, 
		_name 	TEXT, 
		_type	TEXT,
		_source	TEXT REFERENCES _sources(_name) ON DELETE CASCADE ON UPDATE CASCADE, 
		_check  TEXT NOT NULL DEFAULT '',
		UNIQUE(_name, _source)
		)`)
	if err != nil {
		return err
	}
	return nil
}

func (dbe *DBEngine) GetDatabases(ctx context.Context) []*Database {
	var databases []*Database
	for _, v := range dbe.databases {
		databases = append(databases, v)
	}
	return databases
}

func (dbe *DBEngine) GetDatabase(ctx context.Context, name string) (*Database, error) {
	if db, ok := dbe.databases[name]; ok {
		return db, nil
	}

	connString := dbe.connString
	if !strings.HasSuffix(connString, "/") {
		connString += "/"
	}
	connString += name
	pool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		return nil, err
	}
	db := &Database{name, pool, map[string]SourceDesc{}, dbe.exec}
	db_ctx := NewContextForDb(db)
	defer GetConn(db_ctx).Release()
	err = checkDatabase(db_ctx)
	if err != nil {
		return nil, err
	}
	sources, err := db.GetSources(db_ctx)
	if err != nil {
		return nil, err
	}
	for _, source := range sources {
		db.refreshSource(db_ctx, source.Name)
	}
	dbe.databases[name] = db
	return db, nil
}

func (dbe *DBEngine) CreateDatabase(ctx context.Context, name string) (*Database, error) {
	_, err := dbe.pool.Exec(ctx, "CREATE DATABASE "+name)
	if err != nil && err.(*pgconn.PgError).Code != "42P04" {
		return nil, err
	}
	return dbe.GetDatabase(ctx, name)
}

func (dbe *DBEngine) DeleteDatabase(ctx context.Context, name string) error {
	db, err := dbe.GetDatabase(ctx, name)
	if err != nil {
		return err
	}
	db.pool.Close()
	_, err = dbe.pool.Exec(ctx, "DROP DATABASE "+name)
	return err
}
