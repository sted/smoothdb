package database

import (
	"context"
	"reflect"
	"strings"
)

type Table struct {
	Name        string   `json:"name"`
	Owner       string   `json:"owner"`
	Check       []string `json:"check"`
	Unique      []string `json:"unique"`
	Primary     string   `json:"primary"`
	Foreign     []string `json:"foreign"`
	RowSecurity bool     `json:"rowsecurity"`
	Temporary   bool     `json:"temporary,omitempty"`
	Inherits    string   `json:"inherit,omitempty"`
	Columns     []Column `json:"columns,omitempty"`

	Struct reflect.Type `json:"-"`
}

type TableUpdate struct {
	Name        string  `json:"name"`
	NewName     *string `json:"newname"`
	Owner       *string `json:"owner"`
	RowSecurity *bool   `json:"rowsecurity"`
}

func splitTableName(name string) (schemaname, tablename string) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		schemaname = "public"
		tablename = parts[0]
	} else {
		schemaname = parts[0]
		tablename = parts[1]
	}
	return
}

const tablesQuery = "SELECT  schemaname || '.' || tablename, tableowner, rowsecurity FROM pg_tables"

func (db *Database) GetTables(ctx context.Context) ([]Table, error) {
	conn := GetConn(ctx)
	constraints, err := getConstraints(ctx, nil)
	if err != nil {
		return nil, err
	}
	tables := []Table{}
	rows, err := conn.Query(ctx, tablesQuery+
		" WHERE schemaname <> 'pg_catalog' AND schemaname <> 'information_schema'")
	if err != nil {
		return tables, err
	}
	defer rows.Close()

	table := Table{}
	for rows.Next() {
		err := rows.Scan(&table.Name, &table.Owner, &table.RowSecurity)
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
	constraints, err := getConstraints(ctx, &name)
	if err != nil {
		return nil, err
	}
	schemaname, tablename := splitTableName(name)
	table := Table{}
	err = conn.QueryRow(ctx,
		tablesQuery+" WHERE tablename = $1 AND schemaname = $2", tablename, schemaname).
		Scan(&table.Name, &table.Owner, &table.RowSecurity)
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

	create := "CREATE "
	if table.Temporary {
		create += "TEMP "
	}
	create += "TABLE " + table.Name + " (" + columnList
	if table.Check != nil {
		for _, check := range table.Check {
			create += ", CHECK (" + check + ")"
		}
	}
	if table.Unique != nil {
		for _, unique := range table.Unique {
			create += ", UNIQUE (" + unique + ")"
		}
	}
	if table.Primary != "" {
		create += ", PRIMARY KEY (" + table.Primary + ")"
	}
	if table.Foreign != nil {
		for _, foreign := range table.Foreign {
			create += ",  FOREIGN KEY (" + foreign + ")"
		}
	}
	create += ")"
	if table.Inherits != "" {
		create += "INHERITS (" + table.Inherits + ")"
	}

	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	constraints, err := getConstraints(ctx, &table.Name)
	if err != nil {
		return nil, err
	}
	fillTableConstraints(table, constraints)
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
