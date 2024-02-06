package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/sted/smoothdb/logging"
)

var DBE *DbEngine

// DbEngine represents the database engine (a "cluster")
type DbEngine struct {
	config           *Config
	pool             *pgxpool.Pool
	dbtracer         pgx.QueryTracer
	activeDatabases  sync.Map
	allowedDatabases map[string]struct{}
	defaultSchema    string
	authRole         string
}

// InitDbEngine creates a connection pool, connects to the engine and initializes it
func InitDbEngine(dbConfig *Config, logger *logging.Logger) (*DbEngine, error) {
	context := context.Background()

	DBE = &DbEngine{config: dbConfig}

	poolConfig, err := pgxpool.ParseConfig(dbConfig.URL)
	if err != nil {
		return nil, err
	}
	configDbName := poolConfig.ConnConfig.Config.Database
	if configDbName != "" {
		if len(dbConfig.AllowedDatabases) == 0 {
			dbConfig.AllowedDatabases = append(dbConfig.AllowedDatabases, configDbName)
		} else if len(dbConfig.AllowedDatabases) > 1 || dbConfig.AllowedDatabases[0] != configDbName {
			return nil, fmt.Errorf("invalid configuration: Database.URL and Database.AllowedDatabases conflicting")
		}
	}
	DBE.authRole = poolConfig.ConnConfig.User
	if logger != nil {
		DBE.dbtracer = &tracelog.TraceLog{
			Logger:   NewDbLogger(logger.Logger),
			LogLevel: tracelog.LogLevelDebug - tracelog.LogLevel(logger.GetLevel()),
		}
		poolConfig.ConnConfig.Tracer = DBE.dbtracer
	}

	// Create DBE connection pool
	pool, err := pgxpool.NewWithConfig(context, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot create pool with %q (%w)", dbConfig.URL, err)
	}
	DBE.pool = pool

	// Test connection
	err = pool.Ping(context)
	if err != nil {
		return nil, fmt.Errorf("cannot connect with %q (%w)", dbConfig.URL, err)
	}

	if len(dbConfig.AllowedDatabases) != 0 {
		DBE.allowedDatabases = map[string]struct{}{}
		for _, dbname := range dbConfig.AllowedDatabases {
			DBE.allowedDatabases[dbname] = struct{}{}
		}
	}
	if len(dbConfig.SchemaSearchPath) != 0 {
		DBE.defaultSchema = dbConfig.SchemaSearchPath[0]
	} else {
		DBE.defaultSchema = "public"
	}
	// Anon role
	if dbConfig.AnonRole != "" {
		_, err = pool.Exec(context, "CREATE ROLE "+dbConfig.AnonRole+" NOLOGIN")
		if err != nil && err.(*pgconn.PgError).Code != "42710" {
			return nil, err
		}
		// Grant anon to auth
		_, err = pool.Exec(context, "GRANT "+dbConfig.AnonRole+" TO "+DBE.authRole)
		if err != nil {
			return nil, err
		}
	}

	return DBE, nil
}

// Close closes the database engine and all of its active databases
func (dbe *DbEngine) Close() {
	dbe.pool.Close()
	dbe.activeDatabases.Range(func(k, v any) bool {
		v.(*Database).Close()
		dbe.activeDatabases.Delete(k)
		return true
	})
}

// AcquireConnection acquires a connection in the database engine
func (dbe *DbEngine) AcquireConnection(ctx context.Context) (*pgxpool.Conn, error) {
	return dbe.pool.Acquire(ctx)
}

// func (dbe *DBEngine) GetActiveDatabases(ctx context.Context) []Database {
// 	var databases []Database
// 	dbe.activeDatabases.Range(func(k, v any) bool {
// 		databases = append(databases, v.(Database))
// 		return true
// 	})
// 	return databases
// }

// IsDatabaseAllowed chacks if a database is usable
func (dbe *DbEngine) IsDatabaseAllowed(name string) bool {
	if len(dbe.allowedDatabases) == 0 {
		return true
	}
	_, ok := dbe.allowedDatabases[name]
	return ok
}

const databaseQuery = `
	SELECT d.datname, pg_catalog.pg_get_userbyid(d.datdba)
	FROM pg_catalog.pg_database d`

func (dbe *DbEngine) GetDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	conn := GetConn(ctx)
	databases := []DatabaseInfo{}
	rows, err := conn.Query(ctx, databaseQuery+" WHERE d.datistemplate = false ORDER BY 1;")
	if err != nil {
		return databases, err
	}
	defer rows.Close()

	database := DatabaseInfo{}
	for rows.Next() {
		err := rows.Scan(&database.Name, &database.Owner)
		if err != nil {
			return databases, err
		}
		if dbe.IsDatabaseAllowed(database.Name) {
			databases = append(databases, database)
		}
	}

	if err := rows.Err(); err != nil {
		return databases, err
	}
	return databases, nil
}

func (dbe *DbEngine) GetDatabase(ctx context.Context, name string) (*DatabaseInfo, error) {
	if !dbe.IsDatabaseAllowed(name) {
		return nil, fmt.Errorf("database %q not found or not allowed", name)
	}

	db := &DatabaseInfo{}
	err := dbe.pool.QueryRow(ctx, databaseQuery+
		" WHERE d.datname = $1", name).Scan(&db.Name, &db.Owner)
	if err != nil {
		return nil, fmt.Errorf("database %q not found (%w)", name, err)
	}

	return db, nil
}

func (dbe *DbEngine) CreateDatabase(ctx context.Context, name string, noErrIfExists bool) (*DatabaseInfo, error) {
	conn := GetConn(ctx)
	create := "CREATE DATABASE \"" + name + "\""
	_, err := conn.Exec(ctx, create)
	if err != nil {
		if e, ok := err.(*pgconn.PgError); !ok || !noErrIfExists || e.Code != "42P04" {
			return nil, err
		}
	}
	return dbe.GetDatabase(ctx, name)
}

func (dbe *DbEngine) DeleteDatabase(ctx context.Context, name string) error {
	db, err := dbe.GetActiveDatabase(ctx, name)
	if err != nil {
		return err
	}
	db.pool.Close()
	dbe.activeDatabases.Delete(name)

	// SELECT pg_terminate_backend(pg_stat_activity.pid)
	// FROM pg_stat_activity
	// WHERE pg_stat_activity.datname = 'target_db'
	// AND pid <> pg_backend_pid();
	conn := GetConn(ctx)
	_, err = conn.Exec(ctx, "DROP DATABASE \""+name+"\" (FORCE)")
	return err
}

func (dbe *DbEngine) GetActiveDatabase(ctx context.Context, name string) (*Database, error) {

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
		dbi, err := dbe.GetDatabase(ctx, name)
		if err != nil {
			dbe.activeDatabases.Delete(name)
			return nil, err
		}
		db.Name = dbi.Name
		db.Owner = dbi.Owner
		err = db.Activate(ctx)
		if err != nil {
			dbe.activeDatabases.Delete(name)
			return nil, fmt.Errorf("cannot activate database %q (%w)", name, err)
		}
	}
	return db, nil
}

func (dbe *DbEngine) CreateActiveDatabase(ctx context.Context, name string, noErrIfExists bool) (*Database, error) {
	_, err := dbe.CreateDatabase(ctx, name, noErrIfExists)
	if err != nil {
		return nil, err
	}
	return dbe.GetActiveDatabase(ctx, name)
}
