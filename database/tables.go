package database

import (
	"context"
	"strings"
)

type Table struct {
	Name        string   `json:"name"`
	Owner       string   `json:"owner"`
	RowSecurity bool     `json:"rowsecurity"`
	HasIndexes  bool     `json:"hasindexes"`
	HasTriggers bool     `json:"hastriggers"`
	IsPartition bool     `json:"ispartition"`
	Constraints []string `json:"constraints"`
	Columns     []Column `json:"columns,omitempty"`
	Inherits    string   `json:"inherit,omitempty"`
	IfNotExists bool     `json:"ifnotexists,omitempty"`

	//Struct reflect.Type `json:"-"`
}

type TableUpdate struct {
	Name        string  `json:"name"`
	NewName     *string `json:"newname"`
	NewSchema   *string `json:"newschema"`
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
	SELECT n.nspname  || '.' || c.relname tablename,
		pg_get_userbyid(c.relowner) tableowner,
		c.relrowsecurity rowsecurity,
		c.relhasindex hasindexes,
		c.relhastriggers hastriggers,
		c.relispartition ispartition
	FROM pg_class c
    JOIN pg_namespace n ON n.oid = c.relnamespace
  	WHERE c.relkind = ANY (ARRAY['r'::"char", 'p'::"char"]) AND 
		n.nspname NOT IN ('pg_catalog', 'information_schema')`

func GetTables(ctx context.Context) ([]Table, error) {
	conn := GetConn(ctx)
	constraints, err := GetConstraints(ctx, "")
	if err != nil {
		return nil, err
	}
	tables := []Table{}
	rows, err := conn.Query(ctx, tablesQuery+" ORDER BY 1")
	if err != nil {
		return tables, err
	}
	defer rows.Close()

	table := Table{}
	for rows.Next() {
		err := rows.Scan(&table.Name, &table.Owner, &table.RowSecurity,
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

func GetTable(ctx context.Context, name string) (*Table, error) {
	conn := GetConn(ctx)
	constraints, err := GetConstraints(ctx, name)
	if err != nil {
		return nil, err
	}
	schemaname, tablename := splitTableName(name)
	table := Table{}
	err = conn.QueryRow(ctx,
		tablesQuery+" AND c.relname = $1 AND n.nspname = $2", tablename, schemaname).
		Scan(&table.Name, &table.Owner, &table.RowSecurity, &table.HasIndexes, &table.HasTriggers, &table.IsPartition)
	if err != nil {
		return nil, err
	}
	fillTableConstraints(&table, constraints)
	table.Columns, err = GetColumns(ctx, name)
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
		*sql += " DEFAULT " + *column.Default + ""
	}
	for _, constraint := range column.Constraints {
		*sql += " " + constraint
	}
}

func CreateTable(ctx context.Context, table *Table) (*Table, error) {
	gi := GetSmoothContext(ctx)
	conn := gi.Conn
	options := gi.QueryOptions

	var columnList string
	for _, col := range table.Columns {
		if columnList != "" {
			columnList += ", "
		}
		composeColumnSQL(&columnList, &col)
	}

	create := "CREATE "
	create += "TABLE "
	if table.IfNotExists {
		create += "IF NOT EXISTS "
	}
	create += quoteParts(table.Name)
	create += " (" + columnList
	for _, constraint := range table.Constraints {
		create += ", " + constraint
	}
	create += ")"
	if table.Inherits != "" {
		create += "INHERITS (" + table.Inherits + ")"
	}

	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}

	//db.refreshTable(ctx, table.Name)

	if options.ReturnRepresentation {
		table, _ = GetTable(ctx, table.Name)
		return table, nil
	} else {
		return nil, nil
	}
}

func UpdateTable(ctx context.Context, table *TableUpdate) (*Table, error) {
	gi := GetSmoothContext(ctx)
	conn := gi.Conn
	options := gi.QueryOptions

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var alter string
	prefix := "ALTER TABLE " + quoteParts(table.Name)

	// NAME
	if table.NewName != nil {
		alter = prefix + " RENAME TO " + quote(*table.NewName)
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return nil, err
		}
	}
	// SCHEMA
	if table.NewSchema != nil {
		alter = prefix + " SET SCHEMA " + quote(*table.NewSchema)
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return nil, err
		}
	}
	// OWNER
	if table.Owner != nil {
		alter = prefix + " OWNER TO " + quote(*table.Owner)
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return nil, err
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
			return nil, err
		}
	}
	//db.refreshTable(ctx, column.Table)
	err = tx.Commit(ctx)
	if err != nil {
		return nil, err
	}

	if options.ReturnRepresentation {
		table, _ := GetTable(ctx, table.Name)
		return table, nil
	} else {
		return nil, nil
	}
}

func DeleteTable(ctx context.Context, name string, ifExists bool) error {
	conn := GetConn(ctx)
	delete := "DROP TABLE"
	if ifExists {
		delete += " IF EXISTS"
	}
	delete += quote(name)
	_, err := conn.Exec(ctx, delete)
	if err != nil {
		return err
	}
	//delete(db.info.cachedTables, name)
	return nil
}
