package database

import "context"

type Column struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	NotNull     bool     `json:"notnull"`
	Default     *string  `json:"default"`
	Generated   string   `json:"generated,omitempty"` // "stored" or "virtual" (PostgreSQL 18) for a GENERATED ALWAYS AS column
	ReadOnly    bool     `json:"readonly"`            // a generated column has no default and cannot be written
	Comment     *string  `json:"comment"`
	Constraints []string `json:"constraints"`
	Table       string   `json:"table,omitempty"`
	Schema      string   `json:"schema,omitempty"`
}

type ColumnUpdate struct {
	Name    *string `json:"name"`
	Type    *string `json:"type"`
	NotNull *bool   `json:"notnull"`
	Default *string `json:"default"`
	Comment *string `json:"comment"`
}

// The columns are read from pg_catalog, not information_schema, which shows a
// role only the columns of the tables it has a privilege on: the schema cache
// is loaded by the connecting role, an authenticator with no privilege at all
// on the tables it serves in a PostgREST-style deployment. A domain reports its
// base type and a generated column has no default, as information_schema has it.
const columnsQuery = `
	SELECT a.attname,
		CASE WHEN t.typtype = 'd' THEN bt.typname ELSE t.typname END,
		CASE WHEN a.attnotnull THEN 'NO' ELSE 'YES' END,
		CASE WHEN a.attgenerated = '' THEN pg_get_expr(d.adbin, d.adrelid) END,
		CASE a.attgenerated WHEN 's' THEN 'stored' WHEN 'v' THEN 'virtual' ELSE '' END,
		col_description(c.oid, a.attnum),
		c.relname, n.nspname
	FROM pg_attribute a
	JOIN pg_class c ON c.oid = a.attrelid
	JOIN pg_namespace n ON n.oid = c.relnamespace
	JOIN pg_type t ON t.oid = a.atttypid
	LEFT JOIN pg_type bt ON bt.oid = t.typbasetype
	LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
	WHERE c.relname = $1 AND n.nspname = $2
		AND c.relkind IN ('r', 'v', 'm', 'f', 'p')
		AND a.attnum > 0 AND NOT a.attisdropped
	ORDER BY a.attnum`

func GetColumns(ctx context.Context, tablename string) ([]Column, error) {
	conn, schemaname := GetConnAndSchema(ctx)

	constraints, err := GetConstraints(ctx, tablename)
	if err != nil {
		return nil, err
	}
	columns := []Column{}
	rows, err := conn.Query(ctx, columnsQuery, tablename, schemaname)
	if err != nil {
		return columns, err
	}
	defer rows.Close()

	var nullable string
	column := Column{}
	for rows.Next() {
		err := rows.Scan(&column.Name, &column.Type, &nullable, &column.Default, &column.Generated, &column.Comment, &column.Table, &column.Schema)
		if err != nil {
			return columns, err
		}
		column.NotNull = nullable == "NO"
		column.ReadOnly = column.Generated != ""
		fillColumnConstraints(&column, constraints)
		columns = append(columns, column)
	}
	if rows.Err() != nil {
		return columns, err
	}
	return columns, nil
}

func GetColumn(ctx context.Context, tablename string, name string) (*Column, error) {
	conn, schemaname := GetConnAndSchema(ctx)

	constraints, err := GetConstraints(ctx, tablename)
	if err != nil {
		return nil, err
	}
	column := &Column{}
	var nullable string
	err = conn.QueryRow(ctx, columnsQuery, tablename, schemaname).
		Scan(&column.Name, &column.Type, &nullable, &column.Default, &column.Generated, &column.Comment, &column.Table, &column.Schema)
	if err != nil {
		return nil, err
	}
	column.NotNull = nullable == "NO"
	column.ReadOnly = column.Generated != ""
	fillColumnConstraints(column, constraints)
	return column, nil
}

func CreateColumn(ctx context.Context, column *Column) (*Column, error) {
	conn := GetConn(ctx)
	ftablename := composeName(ctx, column.Schema, column.Table)

	create := "ALTER TABLE " + ftablename + " ADD COLUMN "
	composeColumnSQL(&create, column)
	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	// COMMENT
	if column.Comment != nil {
		comment := "COMMENT ON COLUMN " + ftablename + "." + quote(column.Name) + " IS " + quoteLit(*column.Comment)
		_, err = conn.Exec(ctx, comment)
		if err != nil {
			return nil, err
		}
	}
	//db.refreshTable(ctx, column.Table)
	return column, nil
}

func UpdateColumn(ctx context.Context, tablename string, name string, column *ColumnUpdate) error {
	conn, schemaname := GetConnAndSchema(ctx)
	ftablename := _sq(tablename, schemaname)

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var alter string

	prefix := "ALTER TABLE " + ftablename + " ALTER COLUMN "
	// TYPE
	if column.Type != nil {
		alter = prefix + quote(name) + " TYPE " + *column.Type
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return err
		}
	}
	// NOT NULL
	if column.NotNull != nil {
		var set_drop string
		if *column.NotNull {
			set_drop = "SET"
		} else {
			set_drop = "DROP"
		}
		alter = prefix + name + " " + set_drop + " NOT NULL"
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return err
		}
	}
	// DEFAULT
	if column.Default != nil {
		if *column.Default != "" {
			alter = prefix + name + " SET DEFAULT " + *column.Default
		} else {
			alter = prefix + name + " DROP DEFAULT"
		}
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return err
		}
	}
	// COMMENT
	if column.Comment != nil {
		comment := "COMMENT ON COLUMN " + ftablename + "." + quote(name) + " IS " + quoteLit(*column.Comment)
		_, err = tx.Exec(ctx, comment)
		if err != nil {
			return err
		}
	}
	// NAME as the last update
	if column.Name != nil && *column.Name != name {
		alter = "ALTER TABLE " + ftablename + " RENAME " + name + " TO " + *column.Name
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return err
		}
	}

	//db.refreshTable(ctx, ftablename)
	return tx.Commit(ctx)
}

func DeleteColumn(ctx context.Context, tablename string, name string, cascade bool) error {
	conn, schemaname := GetConnAndSchema(ctx)
	ftablename := _sq(tablename, schemaname)

	delete := "ALTER TABLE " + ftablename + " DROP COLUMN " + quote(name)
	if cascade {
		delete += " CASCADE"
	}
	_, err := conn.Exec(ctx, delete)
	if err != nil {
		return err
	}
	//db.refreshTable(ctx, table)
	return nil
}

type ColumnType struct {
	Table       string `json:"table"`
	Schema      string `json:"schema"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	DataType    string `json:"datatype"`
	IsArray     bool   `json:"isarray"`
	IsComposite bool   `json:"iscomposite"`
}

// Every column of every relation, from pg_catalog for the reason columnsQuery
// gives; a domain counts as its base type.
const columnTypesQuery = `
	SELECT
		c.relname tablename,
		n.nspname schema,
		a.attname name,
		ut.typname type,
		format_type(a.atttypid, a.atttypmod) datatype,
		(ut.typcategory = 'A') AS isarray,
		(ut.typcategory = 'C') AS iscomposite
	FROM pg_attribute a
	JOIN pg_class c ON c.oid = a.attrelid
	JOIN pg_namespace n ON n.oid = c.relnamespace
	JOIN pg_type t ON t.oid = a.atttypid
	JOIN pg_type ut ON ut.oid = CASE WHEN t.typtype = 'd' THEN t.typbasetype ELSE t.oid END
	WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'
		AND c.relkind IN ('r', 'v', 'm', 'f', 'p')
		AND a.attnum > 0 AND NOT a.attisdropped
	ORDER BY c.relname, n.nspname, a.attnum;
`

func GetColumnTypes(ctx context.Context) ([]ColumnType, error) {
	conn := GetConn(ctx)
	types := []ColumnType{}
	rows, err := conn.Query(ctx, columnTypesQuery)
	if err != nil {
		return types, err
	}
	defer rows.Close()

	typ := ColumnType{}
	for rows.Next() {
		err := rows.Scan(&typ.Table, &typ.Schema, &typ.Name, &typ.Type, &typ.DataType, &typ.IsArray, &typ.IsComposite)
		if err != nil {
			return types, err
		}
		types = append(types, typ)
	}
	if rows.Err() != nil {
		return types, err
	}
	return types, nil
}
