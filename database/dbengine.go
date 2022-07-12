package database

import (
	"context"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4/pgxpool"
)

var DBE *DBEngine

// DBEngine represents a database instance (a "cluster")
type DBEngine struct {
	connString      string
	pool            *pgxpool.Pool
	activeDatabases map[string]*Database
	exec            *QueryExecutor
}

// InitDBEngine creates a connection pool, connects to the engine and initializes it
func InitDBEngine(connString string) (*DBEngine, error) {
	context := context.Background()
	pool, err := pgxpool.Connect(context, connString)
	if err != nil {
		return nil, err
	}
	DBE = &DBEngine{connString, pool, map[string]*Database{}, &QueryExecutor{}}
	return DBE, nil
}

func (dbe *DBEngine) AcquireConnection(ctx context.Context) *pgxpool.Conn {
	conn, _ := dbe.pool.Acquire(ctx)
	return conn
}

func (dbe *DBEngine) ActivateDatabase(ctx context.Context, db *Database) error {
	connString := dbe.connString
	if !strings.HasSuffix(connString, "/") {
		connString += "/"
	}
	connString += db.Name
	pool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		return err
	}
	db.pool = pool
	db.cachedTables = map[string]Table{}
	db.exec = dbe.exec
	dbe.activeDatabases[db.Name] = db
	return nil
}

func (dbe *DBEngine) GetActiveDatabases(ctx context.Context) []Database {
	var databases []Database
	for _, v := range dbe.activeDatabases {
		databases = append(databases, *v)
	}
	return databases
}

const databaseQuery = `
	SELECT d.datname, a.rolname 
	FROM pg_database d
	LEFT JOIN pg_authid a ON d.datdba = a.oid 
	WHERE datistemplate = false`

func (dbe *DBEngine) GetDatabases(ctx context.Context) ([]Database, error) {
	databases := []Database{}
	rows, err := dbe.pool.Query(ctx, databaseQuery)
	if err != nil {
		return databases, err
	}
	defer rows.Close()

	for rows.Next() {
		database := &Database{}
		err := rows.Scan(&database.Name, &database.Owner)
		if err != nil {
			return databases, err
		}
		databases = append(databases, *database)
	}

	if err := rows.Err(); err != nil {
		return databases, err
	}
	return databases, nil
}

func (dbe *DBEngine) GetDatabase(ctx context.Context, name string) (*Database, error) {
	if db, ok := dbe.activeDatabases[name]; ok {
		return db, nil
	}
	db := &Database{}
	err := dbe.pool.QueryRow(ctx, databaseQuery+
		" AND d.datname = $1", name).Scan(&db.Name, &db.Owner)
	if err != nil {
		return nil, err
	}
	err = dbe.ActivateDatabase(ctx, db)
	if err != nil {
		return nil, err
	}
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
	// SELECT pg_terminate_backend(pg_stat_activity.pid)
	// FROM pg_stat_activity
	// WHERE pg_stat_activity.datname = 'target_db'
	// AND pid <> pg_backend_pid();
	_, err = dbe.pool.Exec(ctx, "DROP DATABASE "+name)
	return err
}
