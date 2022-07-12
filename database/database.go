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

type Constraint struct {
	Name       string
	Type       byte // c: check, u: unique, p: primary, f: foreign
	Table      string
	Column     string
	NumCols    int
	Definition string
}

type Table struct {
	Name    string   `json:"name"`
	Owner   string   `json:"owner"`
	Check   []string `json:"check"`
	Unique  []string `json:"unique"`
	Primary string   `json:"primary"`
	Foreign []string `json:"foreign"`
	Columns []Column `json:"columns,omitempty"`

	Struct reflect.Type `json:"-"`
}

type Column struct {
	Name    string  `json:"name"`
	Type    string  `json:"type"`
	NotNull bool    `json:"notnull"`
	Default *string `json:"default"`
	Check   string  `json:"check"`
	Unique  bool    `json:"unique"`
	Primary bool    `json:"primary"`
	Foreign string  `json:"foreign"`
	Table   string  `json:"table,omitempty"`
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

func fieldsToStruct(fields []Column) reflect.Type {
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

const constraintsQuery = `
	SELECT
		c.conrelid::regclass tablename,
		att.attname colname,
		c.conname  name,
		c.contype  type,
		cardinality(c.conkey) cols,
		pg_get_constraintdef(c.oid, true) def
	FROM pg_constraint c
		JOIN pg_namespace ns ON c.connamespace = ns.oid
		JOIN pg_attribute att ON c.conrelid = att.attrelid and c.conkey[1] = att.attnum
	WHERE ns.nspname = 'public' 
	ORDER BY tablename, type`

func getConstraints(ctx context.Context) ([]Constraint, error) {
	conn := GetConn(ctx)
	constraints := []Constraint{}
	rows, err := conn.Query(ctx, constraintsQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	constraint := Constraint{}
	for rows.Next() {
		err := rows.Scan(&constraint.Table, &constraint.Column, &constraint.Name,
			&constraint.Type, &constraint.NumCols, &constraint.Definition)
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, constraint)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return constraints, nil
}

func fillTableConstraints(table *Table, constraints []Constraint) {
	table.Check = nil
	table.Unique = nil
	table.Primary = ""
	table.Foreign = nil
	for _, c := range constraints {
		if c.Table == table.Name && (c.NumCols > 1 || c.Type == 'p') {
			switch c.Type {
			case 'c':
				table.Check = append(table.Check, c.Definition)
			case 'u':
				table.Unique = append(table.Unique, c.Definition)
			case 'p':
				table.Primary = c.Definition
			case 'f':
				table.Foreign = append(table.Foreign, c.Definition)
			}
		}
	}
}

func fillColumnsConstraints(tableName string, column *Column, constraints []Constraint) {
	column.Check = ""
	column.Unique = false
	column.Primary = false
	column.Foreign = ""
	for _, c := range constraints {
		if c.Table == tableName && c.Column == column.Name && c.NumCols == 1 {
			switch c.Type {
			case 'c':
				column.Check = c.Definition
			case 'u':
				column.Unique = true
			case 'p':
				column.Primary = true
			case 'f':
				column.Foreign = c.Definition
			}
		}
	}
}

func (db *Database) GetTables(ctx context.Context) ([]Table, error) {
	conn := GetConn(ctx)
	constraints, err := getConstraints(ctx)
	if err != nil {
		return nil, err
	}
	tables := []Table{}
	rows, err := conn.Query(ctx, `
		SELECT tablename, tableowner 
		FROM pg_tables
		WHERE schemaname = 'public'`)
	if err != nil {
		return tables, err
	}
	defer rows.Close()

	table := Table{}
	for rows.Next() {
		err := rows.Scan(&table.Name, &table.Owner)
		if err != nil {
			return tables, err
		}
		fillTableConstraints(&table, constraints)
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return tables, err
	}

	return tables, nil
}

func (db *Database) GetTable(ctx context.Context, name string) (*Table, error) {
	conn := GetConn(ctx)
	constraints, err := getConstraints(ctx)
	if err != nil {
		return nil, err
	}
	table := Table{}
	err = conn.QueryRow(ctx,
		"SELECT tablename, tableowner FROM pg_tables WHERE tablename = $1", name).
		Scan(&table.Name, &table.Owner)
	if err != nil {
		return nil, err
	}
	fillTableConstraints(&table, constraints)
	return &table, nil
}

func composeColumnSQL(sql *string, column *Column) {
	*sql += column.Name + " " + column.Type
	if column.NotNull {
		*sql += " NOT NULL"
	}
	if column.Default != nil {
		*sql += " DEFAULT '" + *column.Default + "'"
	}
	if column.Check != "" {
		*sql += " CHECK (" + column.Check + ")"
	}
	if column.Unique {
		*sql += " UNIQUE"
	}
	if column.Primary {
		*sql += " PRIMARY KEY"
	}
	if column.Foreign != "" {
		*sql += " REFERENCES " + column.Foreign
	}
}

func (db *Database) CreateTable(ctx context.Context, table *Table) (*Table, error) {
	conn := GetConn(ctx)

	var columnList string
	for _, col := range table.Columns {
		if columnList != "" {
			columnList += ", "
		}
		composeColumnSQL(&columnList, &col)
	}

	create := "CREATE TABLE " + table.Name + " (" + columnList
	if table.Check != nil {
		for _, check := range table.Check {
			create += ", " + check
		}
	}
	if table.Unique != nil {
		for _, unique := range table.Unique {
			create += ", " + unique
		}
	}
	if table.Primary != "" {
		create += ", " + table.Primary
	}
	if table.Foreign != nil {
		for _, foreign := range table.Foreign {
			create += ", " + foreign
		}
	}
	create += ")"

	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	db.refreshTable(ctx, table.Name)
	return table, nil
}

func (db *Database) DeleteTable(ctx context.Context, name string) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "DROP TABLE "+name)
	if err != nil {
		return err
	}
	delete(db.cachedTables, name)
	return nil
}

func (db *Database) GetColumns(ctx context.Context, table string) ([]Column, error) {
	conn := GetConn(ctx)
	constraints, err := getConstraints(ctx)
	if err != nil {
		return nil, err
	}
	columns := []Column{}
	rows, err := conn.Query(ctx, `
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1`, table)
	if err != nil {
		return columns, err
	}
	defer rows.Close()

	var nullable string
	column := Column{}
	for rows.Next() {
		err := rows.Scan(&column.Name, &column.Type, &nullable, &column.Default)
		if err != nil {
			return columns, err
		}
		if nullable == "NO" {
			column.NotNull = true
		} else {
			column.NotNull = false
		}
		fillColumnsConstraints(table, &column, constraints)
		columns = append(columns, column)
	}

	if rows.Err() != nil {
		return columns, err
	}

	return columns, nil
}

func (db *Database) CreateColumn(ctx context.Context, column *Column) (*Column, error) {
	conn := GetConn(ctx)
	create := "ALTER TABLE " + column.Table + " ADD COLUMN "
	composeColumnSQL(&create, column)
	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	db.refreshTable(ctx, column.Table)
	return column, nil
}

func (db *Database) DeleteColumn(ctx context.Context, table string, name string) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "ALTER TABLE "+table+" DROP COLUMN "+name)
	if err != nil {
		return err
	}
	db.refreshTable(ctx, table)
	return nil
}
