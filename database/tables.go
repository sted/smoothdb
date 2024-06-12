package database

import (
	"context"
	"strings"
)

type Table struct {
	Name        string   `json:"name"`
	Schema      string   `json:"schema"`
	Owner       string   `json:"owner"`
	RowSecurity bool     `json:"rowsecurity"`
	Columns     []Column `json:"columns,omitempty"`
	Constraints []string `json:"constraints"`
	Inherits    string   `json:"inherit,omitempty"`
	IfNotExists bool     `json:"ifnotexists,omitempty"`
	HasIndexes  bool     `json:"hasindexes"`
	HasTriggers bool     `json:"hastriggers"`
	IsPartition bool     `json:"ispartition"`
}

type TableUpdate struct {
	Name        *string `json:"name"`
	Schema      *string `json:"schema"`
	Owner       *string `json:"owner"`
	RowSecurity *bool   `json:"rowsecurity"`
}

func splitTableName(name string) (schemaname, tablename string) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		schemaname = "public" //@@ should depend on the config
		tablename = parts[0]
	} else {
		schemaname = parts[0]
		tablename = parts[1]
	}
	return
}

const tablesQuery = `
	SELECT c.relname tablename,
		n.nspname schema,
		pg_get_userbyid(c.relowner) tableowner,
		c.relrowsecurity rowsecurity,
		c.relhasindex hasindexes,
		c.relhastriggers hastriggers,
		c.relispartition ispartition
	FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
  	WHERE c.relkind = ANY (ARRAY['r'::"char", 'p'::"char"])`

func GetTables(ctx context.Context) ([]Table, error) {
	conn := GetConn(ctx)
	constraints, err := GetConstraints(ctx, "")
	if err != nil {
		return nil, err
	}
	tables := []Table{}
	rows, err := conn.Query(ctx, tablesQuery+
		" AND n.nspname !~ '^pg_' AND n.nspname <> 'information_schema' ORDER BY 1")
	if err != nil {
		return tables, err
	}
	defer rows.Close()

	table := Table{}
	for rows.Next() {
		err := rows.Scan(&table.Name, &table.Schema, &table.Owner, &table.RowSecurity,
			&table.HasIndexes, &table.HasTriggers, &table.IsPartition)
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

func GetTable(ctx context.Context, fname string) (*Table, error) {
	conn := GetConn(ctx)
	constraints, err := GetConstraints(ctx, fname)
	if err != nil {
		return nil, err
	}
	schemaname, tablename := splitTableName(fname)
	table := Table{}
	err = conn.QueryRow(ctx,
		tablesQuery+" AND c.relname = $1 AND n.nspname = $2", tablename, schemaname).
		Scan(&table.Name, &table.Schema, &table.Owner, &table.RowSecurity, &table.HasIndexes, &table.HasTriggers, &table.IsPartition)
	if err != nil {
		return nil, err
	}
	fillTableConstraints(&table, constraints)
	table.Columns, err = GetColumns(ctx, fname)
	if err != nil {
		return nil, err
	}
	return &table, nil
}

func composeColumnSQL(sql *string, column *Column) {
	*sql += quote(column.Name) + " " + column.Type
	if column.NotNull {
		*sql += " NOT NULL"
	}
	if column.Default != nil {
		*sql += " DEFAULT " + *column.Default
	}
	for _, constraint := range column.Constraints {
		*sql += " " + constraint
	}
}

func CreateTable(ctx context.Context, table *Table) (*Table, error) {
	conn := GetConn(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	create := "CREATE TABLE "
	if table.IfNotExists {
		create += "IF NOT EXISTS "
	}
	var ftablename string
	if table.Schema != "" {
		ftablename = _sq(table.Name, table.Schema)
	} else {
		ftablename = quote(table.Name)
	}
	create += ftablename
	var columnList string
	for _, col := range table.Columns {
		if columnList != "" {
			columnList += ", "
		}
		composeColumnSQL(&columnList, &col)
	}
	create += " (" + columnList
	for _, constraint := range table.Constraints {
		create += ", " + constraint
	}
	create += ")"
	if table.Inherits != "" {
		create += "INHERITS (" + table.Inherits + ")"
	}
	_, err = conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}

	var alter string
	prefix := "ALTER TABLE " + ftablename

	// SCHEMA
	if table.Schema != "" {
		alter = prefix + " SET SCHEMA " + quote(table.Schema)
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return nil, err
		}
	}
	// OWNER
	if table.Owner != "" {
		alter = prefix + " OWNER TO " + quote(table.Owner)
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return nil, err
		}
	}
	// RLS
	if table.RowSecurity {
		alter = prefix + " ENABLE ROW LEVEL SECURITY"
	} else {
		alter = prefix + " DISABLE ROW LEVEL SECURITY"
	}
	_, err = tx.Exec(ctx, alter)
	if err != nil {
		return nil, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	//db.refreshTable(ctx, table.Name)

	return GetTable(ctx, table.Name)
}

func UpdateTable(ctx context.Context, fname string, table *TableUpdate) error {
	gi := GetSmoothContext(ctx)
	conn := gi.Conn

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var alter string
	prefix := "ALTER TABLE " + quoteParts(fname)

	// SCHEMA
	if table.Schema != nil {
		alter = prefix + " SET SCHEMA " + quote(*table.Schema)
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return err
		}
	}
	// OWNER
	if table.Owner != nil {
		alter = prefix + " OWNER TO " + quote(*table.Owner)
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return err
		}
	}
	// RLS
	if table.RowSecurity != nil {
		if *table.RowSecurity {
			alter = prefix + " ENABLE ROW LEVEL SECURITY"
		} else {
			alter = prefix + " DISABLE ROW LEVEL SECURITY"
		}
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return err
		}
	}
	// NAME as the last update
	if table.Name != nil {
		alter = prefix + " RENAME TO " + quote(*table.Name)
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return err
		}
	}
	//db.refreshTable(ctx, column.Table)
	return tx.Commit(ctx)
}

func DeleteTable(ctx context.Context, fname string, ifExists bool) error {
	conn := GetConn(ctx)
	delete := "DROP TABLE"
	if ifExists {
		delete += " IF EXISTS "
	}
	delete += quoteParts(fname)
	_, err := conn.Exec(ctx, delete)
	if err != nil {
		return err
	}
	//delete(db.info.cachedTables, name)
	return nil
}
