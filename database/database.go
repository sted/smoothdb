package database

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type Database struct {
	name          string
	pool          *pgxpool.Pool
	cachedSources map[string]SourceDesc
	exec          *QueryExecutor
}

type Source struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Check string `json:"check"`
}

type SourceDesc struct {
	Source
	Fields []Field
	Struct reflect.Type
}

type Field struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Source      string `json:"source"`
	Unique      bool   `json:"unique"`
	NotNull     bool   `json:"notnull"`
	Default     string `json:"default"`
	Check       string `json:"check"`
}

type Record map[string]interface{}

var DBE *DBEngine

var typeMap = map[string]interface{}{
	"integer":   int32(0),
	"serial":    int32(0),
	"text":      string(""),
	"boolean":   bool(false),
	"timestamp": time.Now(),
}

func fieldsToStruct(fields []Field) reflect.Type {
	var structFields []reflect.StructField
	for _, field := range fields {
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

func (db *Database) refreshSource(ctx context.Context, source string) {
	sourceDesc := SourceDesc{}
	sourceDesc.Fields, _ = db.GetFields(ctx, source)
	sourceDesc.Struct = fieldsToStruct(sourceDesc.Fields)
	db.cachedSources[source] = sourceDesc
}

func (db *Database) AcquireConnection(ctx context.Context) *pgxpool.Conn {
	conn, _ := db.pool.Acquire(ctx)
	return conn
}

func (db *Database) GetSources(ctx context.Context) ([]Source, error) {
	conn := GetConn(ctx)
	sources := []Source{}
	rows, err := conn.Query(ctx, "SELECT * FROM _sources")
	if err != nil {
		return sources, err
	}
	defer rows.Close()

	for rows.Next() {
		source := &Source{}
		err := rows.Scan(&source.Id, &source.Name, &source.Check)
		if err != nil {
			return sources, err
		}
		sources = append(sources, *source)
	}

	if err := rows.Err(); err != nil {
		return sources, err
	}

	return sources, nil
}

func (db *Database) GetSource(ctx context.Context, name string) (*Source, error) {
	conn := GetConn(ctx)
	source := Source{}
	err := conn.QueryRow(ctx, "SELECT * FROM _sources WHERE _name = $1", name).Scan(&source.Id, &source.Name, &source.Check)
	if err != nil {
		return nil, err
	}

	return &source, nil
}

func (db *Database) CreateSource(ctx context.Context, name string) (*Source, error) {
	conn := GetConn(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Create source item in the _sources table
	source := &Source{}
	err = tx.QueryRow(ctx, "INSERT INTO _sources (_name) VALUES ($1) RETURNING *", name).
		Scan(&source.Id, &source.Name, &source.Check)
	if err != nil {
		return nil, err
	}

	// Create the related data source
	_, err = tx.Exec(ctx, "CREATE TABLE "+name+" ()")
	if err != nil {
		return nil, err
	}
	_, err = db.CreateField(ctx, &Field{Name: "_id", Type: "SERIAL", Source: name, Check: "PRIMARY KEY"})
	db.refreshSource(ctx, name)

	tx.Commit(ctx)
	return source, nil
}

func (db *Database) DeleteSource(ctx context.Context, name string) error {
	conn := GetConn(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete the source item from the _sources table
	_, err = tx.Exec(ctx, "DELETE FROM _sources WHERE _name = $1", name)
	if err != nil {
		return err
	}

	// Delete the related data source
	_, err = tx.Exec(ctx, "DROP TABLE "+name)
	if err != nil {
		return err
	}

	tx.Commit(ctx)
	return nil
}

func (db *Database) GetFields(ctx context.Context, source string) ([]Field, error) {
	conn := GetConn(ctx)
	fields := []Field{}
	rows, err := conn.Query(ctx, "SELECT * FROM _fields WHERE _source = $1", source)
	if err != nil {
		return fields, err
	}
	defer rows.Close()

	for rows.Next() {
		field := &Field{}
		err := rows.Scan(&field.Id, &field.Name, &field.Type, &field.Source, &field.Check)
		if err != nil {
			return fields, err
		}
		fields = append(fields, *field)
	}

	if rows.Err() != nil {
		return fields, err
	}

	return fields, nil
}

func (db *Database) CreateField(ctx context.Context, field *Field) (*Field, error) {
	conn := GetConn(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Create the field item in the _fields table
	err = tx.QueryRow(ctx, `INSERT INTO _fields 
		(_name, _type, _source, _check) VALUES ($1, $2, $3, $4) 
		RETURNING *`,
		field.Name, field.Type, field.Source, field.Check).
		Scan(&field.Id, &field.Name, &field.Type, &field.Source, &field.Check)
	if err != nil {
		return nil, err
	}

	// Create the related column
	_, err = tx.Exec(ctx, "ALTER TABLE "+field.Source+" ADD COLUMN "+field.Name+" "+field.Type)
	if err != nil {
		return nil, err
	}
	db.refreshSource(ctx, field.Source)

	tx.Commit(ctx)
	return field, nil
}

func (db *Database) DeleteField(ctx context.Context, source string, name string) error {
	conn := GetConn(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete the field item from the fields table
	_, err = tx.Exec(ctx, "DELETE FROM _fields WHERE _source = $1 AND _name = $2", source, name)
	if err != nil {
		return err
	}

	// Delete the related column
	_, err = tx.Exec(ctx, "ALTER TABLE "+source+" DROP COLUMN "+name)
	if err != nil {
		return err
	}

	tx.Commit(ctx)
	return nil
}
