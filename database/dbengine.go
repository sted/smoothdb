package database

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/sted/smoothdb/logging"
)

const (
	SMOOTHDB = "smoothdb"
)

var dbe *DbEngine

// DbEngine represents the database engine (the PostgreSQL instance or "cluster")
type DbEngine struct {
	config           *Config
	dbtracer         pgx.QueryTracer
	activeDatabases  sync.Map
	dbCreationLock   sync.Mutex
	allowedDatabases map[string]struct{}
	defaultSchema    string
	authRole         string
	mainDb           *Database
}

// InitDbEngine creates a connection pool, connects to the engine and initializes it
func InitDbEngine(dbConfig *Config, logger *logging.Logger) (*DbEngine, error) {

	dbe = &DbEngine{config: dbConfig}

	config, err := pgxpool.ParseConfig(dbConfig.URL)
	if err != nil {
		return nil, err
	}
	// Main db name
	configDbName := config.ConnConfig.Config.Database
	if configDbName == "" {
		configDbName = SMOOTHDB
	}

	// Authenticator role
	dbe.authRole = config.ConnConfig.User
	// Db logging
	if logger != nil {
		dbe.dbtracer = &tracelog.TraceLog{
			Logger:   NewDbLogger(logger.Logger),
			LogLevel: tracelog.LogLevelDebug - tracelog.LogLevel(logger.GetLevel()),
		}
	}

	ctx := context.Background()
	conn, err := pgx.ConnectConfig(ctx, config.ConnConfig)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	// Test connection
	err = conn.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot connect with %q (%w)", dbConfig.URL, err)
	}

	// Activate main db
	dbi, err := dbe.getDatabase(ctx, conn, configDbName)
	if err != nil {
		return nil, err
	}
	db := &Database{}
	db.Name = dbi.Name
	db.Owner = dbi.Owner
	db.activation = make(chan struct{})
	err = db.activate(ctx)
	if err != nil {
		return nil, err
	}
	dbe.activeDatabases.Store(configDbName, db)
	dbe.mainDb = db

	if len(dbConfig.AllowedDatabases) != 0 {
		dbe.allowedDatabases = map[string]struct{}{}
		for _, dbname := range dbConfig.AllowedDatabases {
			dbe.allowedDatabases[dbname] = struct{}{}
		}
	}
	if len(dbConfig.SchemaSearchPath) != 0 {
		dbe.defaultSchema = dbConfig.SchemaSearchPath[0]
	} else {
		dbe.defaultSchema = "public"
	}

	return dbe, nil
}

// Close closes the database engine and all of its active databases
func (dbe *DbEngine) Close() {
	dbe.activeDatabases.Range(func(k, v any) bool {
		v.(*Database).Close()
		dbe.activeDatabases.Delete(k)
		return true
	})
}

// AcquireConnection acquires a connection in the database engine
func (dbe *DbEngine) AcquireConnection(ctx context.Context) (*pgxpool.Conn, error) {
	return dbe.mainDb.AcquireConnection(ctx)
}

// IsDatabaseAllowed checks if a database can be managed by smoothdb
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

// GetDatabases lists the available databases, filtering the not-managed ones
func (dbe *DbEngine) GetDatabases(ctx context.Context) ([]DatabaseInfo, error) {
	conn := GetConn(ctx)
	databases := []DatabaseInfo{}
	rows, err := conn.Query(ctx, databaseQuery+" WHERE d.datistemplate = false ORDER BY 1;")
	if err != nil {
		return databases, err
	}
	defer rows.Close()

	dbi := DatabaseInfo{}
	for rows.Next() {
		err := rows.Scan(&dbi.Name, &dbi.Owner)
		if err != nil {
			return databases, err
		}
		if dbe.IsDatabaseAllowed(dbi.Name) {
			databases = append(databases, dbi)
		}
	}

	if err := rows.Err(); err != nil {
		return databases, err
	}
	return databases, nil
}

func (dbe *DbEngine) getDatabase(ctx context.Context, conn *pgx.Conn, name string) (*DatabaseInfo, error) {
	if !dbe.IsDatabaseAllowed(name) {
		return nil, fmt.Errorf("database %q not found or not allowed", name)
	}
	dbi := &DatabaseInfo{}
	err := conn.QueryRow(ctx, databaseQuery+
		" WHERE d.datname = $1", name).Scan(&dbi.Name, &dbi.Owner)
	if err != nil {
		return nil, fmt.Errorf("database %q not found (%w)", name, err)
	}
	return dbi, nil
}

// GetDatabase gets information about a database.
// Use GetActiveDatabase to get an active instance of a database
func (dbe *DbEngine) GetDatabase(ctx context.Context, name string) (*DatabaseInfo, error) {
	conn, err := dbe.mainDb.pool.Acquire(ctx)
	defer conn.Release()
	if err != nil {
		return nil, err
	}
	return dbe.getDatabase(ctx, conn.Conn(), name)
}

// CreateDatabase creates a new database
// Use CreateActiveDatabase to create an active instance of a database
func (dbe *DbEngine) CreateDatabase(ctx context.Context, name string, owner string, getIfExists bool) (*DatabaseInfo, error) {
	if !dbe.IsDatabaseAllowed(name) {
		return nil, fmt.Errorf("database %q not allowed", name)
	}
	conn := GetConn(ctx)
	create := "CREATE DATABASE \"" + name + "\""
	if owner != "" {
		create += " OWNER " + owner
	}
	_, err := conn.Exec(ctx, create)
	if err != nil {
		if e, ok := err.(*pgconn.PgError); !ok || !getIfExists || e.Code != "42P04" {
			return nil, err
		}
	}
	return dbe.GetDatabase(ctx, name)
}

var terminateQuery = `SELECT pg_terminate_backend(pid)
	FROM pg_stat_activity WHERE datname = $1
	AND pid <> pg_backend_pid();`

func enableConnections(ctx context.Context, name string, enable bool) error {
	conn := GetConn(ctx)
	enableStr := "false"
	if enable {
		enableStr = "true"
	}
	query := "ALTER DATABASE \"" + name + "\" WITH ALLOW_CONNECTIONS " + enableStr
	_, err := conn.Exec(ctx, query)
	if err != nil {
		return err
	}
	return nil
}

func terminateSessions(ctx context.Context, name string) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, terminateQuery, name)
	if err != nil {
		return err
	}
	return nil
}

// CloneDatabase creates a new database cloning a source db.
// The force parameter can be used to stop all the connections to the source db, which is a prerequisite.
func (dbe *DbEngine) CloneDatabase(ctx context.Context, name string, source string, force bool) (*DatabaseInfo, error) {
	if !dbe.IsDatabaseAllowed(name) {
		return nil, fmt.Errorf("database %q not allowed", name)
	}
	if force {
		err := enableConnections(ctx, source, false)
		if err != nil {
			return nil, err
		}
		defer enableConnections(ctx, source, true)
		err = terminateSessions(ctx, source)
		if err != nil {
			return nil, err
		}
	}
	conn := GetConn(ctx)
	create := "CREATE DATABASE " + quote(name) + " WITH TEMPLATE " + quote(source)
	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	return dbe.GetDatabase(ctx, name)
}

// UpdateDatabase updates a database.
func (dbe *DbEngine) UpdateDatabase(ctx context.Context, name string, update *DatabaseUpdate) error {
	conn := GetConn(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if update.Owner != nil {
		_, err = conn.Exec(ctx, "ALTER DATABASE "+quote(name)+" OWNER to "+*update.Owner)
		if err != nil {
			return err
		}
	}
	// NAME as the last update
	if update.Name != nil && *update.Name != name {
		_, err = conn.Exec(ctx, "ALTER DATABASE "+quote(name)+" RENAME TO "+*update.Name)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// DeleteDatabase deletes a database.
// It blocks until all connections are returned to pool.
func (dbe *DbEngine) DeleteDatabase(ctx context.Context, name string) error {
	db_, exists := dbe.activeDatabases.Load(name)
	if exists {
		db := db_.(*Database)
		db.pool.Close()
		dbe.activeDatabases.Delete(name)
	}
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "DROP DATABASE "+quote(name)+" (FORCE)")
	return err
}

// GetActiveDatabase allows to get a database and ensures it is activated (eg it has a connection pool
// and its cached information)
func (dbe *DbEngine) GetActiveDatabase(ctx context.Context, name string) (db *Database, err error) {
	db_, exists := dbe.activeDatabases.LoadOrStore(name, &Database{
		activation: make(chan struct{}),
	})
	db = db_.(*Database)
	if exists {
		select {
		case <-db.activation:
			if db.activationErr == nil {
				return db, nil
			} else {
				return nil, db.activationErr
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	} else {
		defer func() {
			if err != nil {
				dbe.activeDatabases.Delete(name)
			}
		}()

		dbi, err := dbe.GetDatabase(ctx, name)
		if err != nil {
			close(db.activation)
			return nil, err
		}
		db.Name = dbi.Name
		db.Owner = dbi.Owner
		err = db.activate(ctx)
		if err != nil {
			return nil, fmt.Errorf("cannot activate database %q (%w)", name, err)
		}
	}
	return db, nil
}

func (dbe *DbEngine) createActiveDatabase(ctx context.Context, name string, owner string) (*Database, error) {
	_, err := dbe.CreateDatabase(ctx, name, owner, false)
	if err != nil {
		return nil, err
	}
	return dbe.GetActiveDatabase(ctx, name)
}

// CreateActiveDatabase allows to create a database and ensures it is activated
func (dbe *DbEngine) CreateActiveDatabase(ctx context.Context, name string, owner string) (*Database, error) {
	dbe.dbCreationLock.Lock()
	defer dbe.dbCreationLock.Unlock()
	return dbe.createActiveDatabase(ctx, name, owner)
}

// GetOrCreateActiveDatabase allows to get or create a database and ensures it is activated
func (dbe *DbEngine) GetOrCreateActiveDatabase(ctx context.Context, name string) (*Database, error) {
	db, err := dbe.GetActiveDatabase(ctx, name)
	if err == nil {
		return db, err
	}
	dbe.dbCreationLock.Lock()
	defer dbe.dbCreationLock.Unlock()
	db, err = dbe.GetActiveDatabase(ctx, name)
	if err == nil {
		return db, err
	}
	return dbe.createActiveDatabase(ctx, name, "")
}

// GetMainDatabase is used to get the main db
func (dbe *DbEngine) GetMainDatabase(ctx context.Context) (*Database, error) {
	return dbe.mainDb, nil
}
