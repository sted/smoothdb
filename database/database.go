package database

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Database struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`

	pool                *pgxpool.Pool
	cachedTables        map[string]Table
	cachedConstraints   map[string][]Constraint
	cachedRelationships map[string][]Relationship
	exec                *QueryExecutor
}

// var typeMap = map[string]any{
// 	"integer":   int32(0),
// 	"serial":    int32(0),
// 	"text":      string(""),
// 	"boolean":   bool(false),
// 	"timestamp": time.Now(),
// }

func (db *Database) GetRelationships(table string) []Relationship {
	return db.cachedRelationships[table]
}

func (db *Database) Activate(ctx context.Context) error {
	connString := DBE.config.URL
	if !strings.HasSuffix(connString, "/") {
		connString += "/"
	}
	connString += db.Name + "?pool_max_conns=" + strconv.Itoa(int(DBE.config.MaxPoolConnections))
	pool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		return err
	}
	db.pool = pool
	db.cachedTables = map[string]Table{}
	db.cachedConstraints = map[string][]Constraint{}
	db.cachedRelationships = map[string][]Relationship{}
	db.exec = DBE.exec
	DBE.activeDatabases[db.Name] = db

	tempCtx := WithDb(ctx, db)
	defer ReleaseContext(tempCtx)
	constraints, err := getConstraints(tempCtx, nil)
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
	return nil
}

// func fieldsToStruct(columns []Column) reflect.Type {
// 	var structFields []reflect.StructField
// 	for _, field := range columns {
// 		name := strings.TrimPrefix(field.Name, "_")
// 		newField := reflect.StructField{
// 			Name: strings.Title(name),
// 			Type: reflect.TypeOf(typeMap[strings.ToLower(field.Type)]),
// 			Tag:  reflect.StructTag("`json:\"" + strings.ToLower(field.Name+"\"`")),
// 		}
// 		if !field.NotNull {
// 			newField.Type = reflect.PtrTo(newField.Type)
// 		}
// 		structFields = append(structFields, newField)
// 	}
// 	return reflect.StructOf(structFields)
// }

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
