package database

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DbInfo struct {
	cachedTables            map[string]Table
	cachedPrimaryKeys       map[string]Constraint
	cachedForeignKeys       map[string][]ForeignKey
	cachedUniqueConstraints map[string][]Constraint
	cachedCheckConstraints  map[string][]Constraint
	cachedRelationships     map[string][]Relationship
}

func (db *DbInfo) GetPrimaryKey(table string) (Constraint, bool) {
	c, ok := db.cachedPrimaryKeys[table]
	return c, ok
}

func (db *DbInfo) GetRelationships(table string) []Relationship {
	return db.cachedRelationships[table]
}

type Database struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`

	activated atomic.Bool
	pool      *pgxpool.Pool
	exec      *QueryExecutor
	DbInfo
}

// Activate initializes a database and start its connection pool.
// It signal a db as activated even if there is an error, which
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
	db.cachedTables = map[string]Table{}
	db.cachedPrimaryKeys = map[string]Constraint{}
	db.cachedForeignKeys = map[string][]ForeignKey{}
	db.cachedUniqueConstraints = map[string][]Constraint{}
	db.cachedCheckConstraints = map[string][]Constraint{}
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
		switch c.Type {
		case 'p':
			db.cachedPrimaryKeys[c.Table] = c
		case 'f':
			db.cachedForeignKeys[c.Table] = append(db.cachedForeignKeys[c.Table], *constraintToForeignKey(&c))
		case 'u':
			db.cachedUniqueConstraints[c.Table] = append(db.cachedUniqueConstraints[c.Table], c)
		case 'c':
			db.cachedCheckConstraints[c.Table] = append(db.cachedCheckConstraints[c.Table], c)
		}
	}
	var pk *Constraint
	for _, fkeys := range db.cachedForeignKeys {
		for _, fk := range fkeys {
			c, ok := db.cachedPrimaryKeys[fk.Table]
			if ok {
				pk = &c
			}

			rels := foreignKeyToRelationships(&fk, pk, db.cachedUniqueConstraints[fk.Table])
			db.cachedRelationships[fk.Table] = append(db.cachedRelationships[fk.Table], rels[0])
			db.cachedRelationships[fk.RelatedTable] = append(db.cachedRelationships[fk.RelatedTable], rels[1])
		}
	}
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

func (db *Database) AcquireConnection(ctx context.Context) *pgxpool.Conn {
	conn, _ := db.pool.Acquire(ctx)
	return conn
}
