package database

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Database struct {
	Name  string `json:"name"`
	Owner string `json:"owner"`

	pool         *pgxpool.Pool
	cachedTables map[string]Table
	exec         *QueryExecutor
}

type Record = map[string]any

var typeMap = map[string]any{
	"integer":   int32(0),
	"serial":    int32(0),
	"text":      string(""),
	"boolean":   bool(false),
	"timestamp": time.Now(),
}

func fieldsToStruct(columns []Column) reflect.Type {
	var structFields []reflect.StructField
	for _, field := range columns {
		name := strings.TrimPrefix(field.Name, "_")
		newField := reflect.StructField{
			Name: strings.Title(name),
			Type: reflect.TypeOf(typeMap[strings.ToLower(field.Type)]),
			Tag:  reflect.StructTag("`json:\"" + strings.ToLower(field.Name+"\"`")),
		}
		if !field.NotNull {
			newField.Type = reflect.PtrTo(newField.Type)
		}
		structFields = append(structFields, newField)
	}
	return reflect.StructOf(structFields)
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
