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

	activated           atomic.Bool
	pool                *pgxpool.Pool
	cachedTables        map[string]Table
	cachedConstraints   map[string][]Constraint
	cachedRelationships map[string][]Relationship
	exec                *QueryExecutor
}

func (db *Database) GetRelationships(table string) []Relationship {
	return db.cachedRelationships[table]
}

func (db *Database) Activate(ctx context.Context) error {
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
	config.ConnConfig.Tracer = DBE.logger

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
	db.cachedTables = map[string]Table{}
	db.cachedConstraints = map[string][]Constraint{}
	db.cachedRelationships = map[string][]Relationship{}
	db.exec = DBE.exec

	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		return err
	}
	constraints, err := getConstraints(ctx, conn, nil)
	if err != nil {
		return err
	}
	for _, c := range constraints {
		db.cachedConstraints[c.Table] = append(db.cachedConstraints[c.Table], c)
		if c.Type == 'f' {
			rels := constraintToRelationships(&c)
			db.cachedRelationships[rels[0].Table] = append(db.cachedRelationships[rels[0].Table], rels[0])
			db.cachedRelationships[rels[1].Table] = append(db.cachedRelationships[rels[1].Table], rels[1])
		}
	}
	db.activated.Store(true)
	return nil
}

func (db *Database) refreshTable(ctx context.Context, name string) {
	// table := Table{}
	// table.Columns, _ = db.GetColumns(ctx, name)
	// table.Struct = fieldsToStruct(table.Columns)
	// db.cachedTables[name] = table
}

func (db *Database) AcquireConnection(ctx context.Context) *pgxpool.Conn {
	conn, _ := db.pool.Acquire(ctx)
	return conn
}
