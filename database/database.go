package database

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`

	activated atomic.Bool
	pool      *pgxpool.Pool
	exec      *QueryExecutor
	DbInfo
}

// Activate initializes a database and starts its connection pool.
// It signals a db as activated even if there is an error, which
// is managed in DBE
func (db *Database) Activate(ctx context.Context) error {
	defer db.activated.Store(true)

	connString := DBE.config.URL
	if !strings.HasSuffix(connString, "/") {
		connString += "/"
	}
	connString += db.Name

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return err
	}
	if DBE.config.AuthRole != "" {
		config.ConnConfig.User = DBE.config.AuthRole
	}
	config.MinConns = DBE.config.MinPoolConnections
	config.MaxConns = DBE.config.MaxPoolConnections
	config.ConnConfig.Tracer = DBE.dblogger

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		var set string
		var err error
		if len(DBE.config.SchemaSearchPath) != 0 {
			set = "set_config('search_path', '"
			for i, schema := range DBE.config.SchemaSearchPath {
				if i != 0 {
					set += ", "
				}
				set += schema
			}
			set += "', false)"
			_, err = conn.Exec(ctx, "SELECT "+set)
		}
		return err
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}
	db.pool = pool
	db.exec = DBE.exec

	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	c := WithDbConn(context.Background(), db, conn)
	tables, err := db.GetTables(c)
	if err != nil {
		return err
	}
	constraints, err := db.GetConstraints(c, "")
	if err != nil {
		return err
	}
	db.initDbInfo(tables, constraints)

	return nil
}

func (db *Database) Close() {
	db.pool.Close()
}

func (db *Database) refreshTable(ctx context.Context, name string) {
	// table := Table{}
	// table.Columns, _ = db.GetColumns(ctx, name)
	// table.Struct = fieldsToStruct(table.Columns)
	// db.cachedTables[name] = table
}

func (db *Database) AcquireConnection(ctx context.Context) (*pgxpool.Conn, error) {
	return db.pool.Acquire(ctx)
}
