package database

import "context"

type Column struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	NotNull     bool     `json:"notnull"`
	Default     *string  `json:"default"`
	Constraints []string `json:"constraints"`
	Table       string   `json:"table,omitempty"`
	Schema      string   `json:"schema,omitempty"`
}

type ColumnUpdate struct {
	Name    *string `json:"name"`
	Type    *string `json:"type"`
	NotNull *bool   `json:"notnull"`
	Default *string `json:"default"`
}

const columnsQuery = `
	SELECT column_name, udt_name, is_nullable, column_default, table_name, table_schema
	FROM information_schema.columns
	WHERE table_name = $1 AND table_schema = $2`

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
		err := rows.Scan(&column.Name, &column.Type, &nullable, &column.Default, &column.Table, &column.Schema)
		if err != nil {
			return columns, err
		}
		column.NotNull = nullable == "NO"
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
		Scan(&column.Name, &column.Type, &nullable, &column.Default, &column.Table, &column.Schema)
	if err != nil {
		return nil, err
	}
	column.NotNull = nullable == "NO"
	fillColumnConstraints(column, constraints)
	return column, nil
}

func CreateColumn(ctx context.Context, column *Column) (*Column, error) {
	conn := GetConn(ctx)
	ftablename := composeTableName(ctx, column.Schema, column.Table)

	create := "ALTER TABLE " + ftablename + " ADD COLUMN "
	composeColumnSQL(&create, column)
	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
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

const columnTypesQuery = `
	SELECT
		c.table_name tablename,
		c.table_schema schema,
		c.column_name name,
		c.udt_name type,		
		c.data_type datatype,
		(t.typcategory = 'A') AS isarray,
		(t.typcategory = 'C') AS iscomposite
	FROM
		information_schema.columns c
		JOIN pg_type t ON c.udt_name = t.typname and c.udt_schema::regnamespace = t.typnamespace
	WHERE
		c.table_schema !~ '^pg_' AND c.table_schema <> 'information_schema'
	ORDER BY
		table_name, table_schema, ordinal_position;
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
