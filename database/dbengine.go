package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"green/green-ds/logging"

	zerologadapter "github.com/jackc/pgx-zerolog"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
)

var DBE *DbEngine

// DbEngine represents a database instance (a "cluster")
type DbEngine struct {
	config           *Config
	pool             *pgxpool.Pool
	logger           pgx.QueryTracer
	activeDatabases  sync.Map
	allowedDatabases map[string]struct{}
	exec             *QueryExecutor
	defaultSchema    string
}

// InitDbEngine creates a connection pool, connects to the engine and initializes it
func InitDbEngine(dbConfig *Config, logger *logging.Logger) (*DbEngine, error) {
	context := context.Background()

	DBE = &DbEngine{config: dbConfig, exec: &QueryExecutor{}}

	poolConfig, err := pgxpool.ParseConfig(dbConfig.URL)
	if err != nil {
		return nil, err
	}
	if logger != nil {
		DBE.logger = &tracelog.TraceLog{
			Logger:   zerologadapter.NewLogger(*logger.Logger),
			LogLevel: tracelog.LogLevelDebug - tracelog.LogLevel(logger.GetLevel()),
		}
		poolConfig.ConnConfig.Tracer = DBE.logger
	}

	// Create DBE connection pool
	pool, err := pgxpool.NewWithConfig(context, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot connect with %q (%w)", dbConfig.URL, err)
	}
	DBE.pool = pool

	if len(dbConfig.AllowedDatabases) != 0 {
		DBE.allowedDatabases = map[string]struct{}{}
		for _, db := range dbConfig.AllowedDatabases {
			DBE.allowedDatabases[db] = struct{}{}
		}
	}
	if len(dbConfig.SchemaSearchPath) != 0 {
		DBE.defaultSchema = dbConfig.SchemaSearchPath[0]
	}
	// Auth role
	if dbConfig.AuthRole != "" {
		_, err = pool.Exec(context, "CREATE ROLE "+dbConfig.AuthRole+" NOINHERIT;")
		if err != nil && err.(*pgconn.PgError).Code != "42710" {
			return nil, err
		}
	}
	// Anon role
	if dbConfig.AnonRole != "" {
		_, err = pool.Exec(context, "CREATE ROLE "+dbConfig.AnonRole+" NOLOGIN")
		if err != nil && err.(*pgconn.PgError).Code != "42710" {
			return nil, err
		}
	}
	return DBE, nil
}

func (dbe *DbEngine) AcquireConnection(ctx context.Context) *pgxpool.Conn {
	conn, _ := dbe.pool.Acquire(ctx)
	return conn
}

// func (dbe *DBEngine) GetActiveDatabases(ctx context.Context) []Database {
// 	var databases []Database
// 	dbe.activeDatabases.Range(func(k, v any) bool {
// 		databases = append(databases, v.(Database))
// 		return true
// 	})
// 	return databases
// }

func (dbe *DbEngine) IsDatabaseAllowed(name string) bool {
	if dbe.allowedDatabases == nil {
		return true
	}
	_, ok := dbe.allowedDatabases[name]
	return ok
}

const databaseQuery = `
	SELECT d.datname, a.rolname 
	FROM pg_database d
	LEFT JOIN pg_authid a ON d.datdba = a.oid 
	WHERE datistemplate = false`

func (dbe *DbEngine) GetDatabases(ctx context.Context) ([]Database, error) {
	conn := GetConn(ctx)
	databases := []Database{}
	rows, err := conn.Query(ctx, databaseQuery)
	if err != nil {
		return databases, err
	}
	defer rows.Close()

	database := Database{}
	for rows.Next() {
		err := rows.Scan(&database.Name, &database.Owner)
		if err != nil {
			return databases, err
		}
		databases = append(databases, database)
	}

	if err := rows.Err(); err != nil {
		return databases, err
	}
	return databases, nil
}

func (dbe *DbEngine) GetDatabase(ctx context.Context, name string) (*Database, error) {
	if !dbe.IsDatabaseAllowed(name) {
		return nil, fmt.Errorf("database %q not found or not allowed", name)
	}
	db_, loaded := dbe.activeDatabases.LoadOrStore(name, &Database{})
	db := db_.(*Database)
	if loaded {
		if !db.activated.Load() {
			for {
				time.Sleep(10 * time.Millisecond)
				if db.activated.Load() {
					break
				}
			}
		}
	} else {
		err := dbe.pool.QueryRow(ctx, databaseQuery+
			" AND d.datname = $1", name).Scan(&db.Name, &db.Owner)
		if err != nil {
			return nil, fmt.Errorf("database %q not found or not allowed (%w)", name, err)
		}
		err = db.Activate(ctx)
		if err != nil {
			return nil, fmt.Errorf("cannot activate database %q (%w)", name, err)
		}
	}
	return db, nil
}

func (dbe *DbEngine) CreateDatabase(ctx context.Context, name string) (*Database, error) {
	_, err := dbe.pool.Exec(ctx, "CREATE DATABASE "+name)
	if err != nil && err.(*pgconn.PgError).Code != "42P04" {
		return nil, err
	}
	return dbe.GetDatabase(ctx, name)
}

func (dbe *DbEngine) DeleteDatabase(ctx context.Context, name string) error {
	// db, err := dbe.GetDatabase(ctx, name)
	// if err != nil {
	// 	return err
	// }
	//db.pool.Close()
	// SELECT pg_terminate_backend(pg_stat_activity.pid)
	// FROM pg_stat_activity
	// WHERE pg_stat_activity.datname = 'target_db'
	// AND pid <> pg_backend_pid();
	_, err := dbe.pool.Exec(ctx, "DROP DATABASE "+name+" (FORCE)")
	return err
}
