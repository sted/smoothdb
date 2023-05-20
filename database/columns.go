package database

import "context"

type Column struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	NotNull     bool     `json:"notnull"`
	Default     *string  `json:"default"`
	Constraints []string `json:"constraints"`
	Table       string   `json:"table,omitempty"`
}

type ColumnUpdate struct {
	Name    string  `json:"name"`
	NewName *string `json:"newname"`
	Type    *string `json:"type"`
	NotNull *bool   `json:"notnull"`
	Default *string `json:"default"`
	Table   string  `json:"-"`
}

const columnsQuery = `
	SELECT column_name, data_type, is_nullable, column_default, table_schema || '.' || table_name
	FROM information_schema.columns
	WHERE table_name = $1 AND table_schema = $2`

func (db *Database) GetColumns(ctx context.Context, ftablename string) ([]Column, error) {
	conn := GetConn(ctx)
	constraints, err := db.GetConstraints(ctx, ftablename)
	if err != nil {
		return nil, err
	}
	schemaname, tablename := splitTableName(ftablename)
	columns := []Column{}
	rows, err := conn.Query(ctx, columnsQuery, tablename, schemaname)
	if err != nil {
		return columns, err
	}
	defer rows.Close()

	var nullable string
	column := Column{}
	for rows.Next() {
		err := rows.Scan(&column.Name, &column.Type, &nullable, &column.Default, &column.Table)
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

func (db *Database) GetColumn(ctx context.Context, ftablename string, name string) (*Column, error) {
	conn := GetConn(ctx)
	constraints, err := db.GetConstraints(ctx, ftablename)
	if err != nil {
		return nil, err
	}
	schemaname, tablename := splitTableName(ftablename)
	column := &Column{}
	var nullable string
	err = conn.QueryRow(ctx, columnsQuery, tablename, schemaname).
		Scan(&column.Name, &column.Type, &nullable, &column.Default, &column.Table)
	if err != nil {
		return nil, err
	}
	column.NotNull = nullable == "NO"
	fillColumnConstraints(column, constraints)
	return column, nil
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

func (db *Database) UpdateColumn(ctx context.Context, column *ColumnUpdate) (*Column, error) {
	conn := GetConn(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var alter string

	prefix := "ALTER TABLE " + column.Table + " ALTER COLUMN "
	// TYPE
	if column.Type != nil {
		alter = prefix + column.Name + " TYPE " + *column.Type
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return nil, err
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
		alter = prefix + column.Name + " " + set_drop + " NOT NULL"
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return nil, err
		}
	}
	// DEFAULT
	if column.Default != nil {
		if *column.Default != "" {
			alter = prefix + column.Name + " SET DEFAULT " + *column.Default
		} else {
			alter = prefix + column.Name + " DROP DEFAULT"
		}
		_, err = tx.Exec(ctx, alter)
		if err != nil {
			return nil, err
		}
	}
	// NAME
	if column.NewName != nil {
		alter = "ALTER TABLE " + column.Table + " RENAME " + column.Name + " TO " + *column.NewName
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
	return &Column{}, nil
}

func (db *Database) DeleteColumn(ctx context.Context, table string, name string, cascade bool) error {
	conn := GetConn(ctx)
	delete := "ALTER TABLE " + table + " DROP COLUMN " + name
	if cascade {
		delete += " CASCADE"
	}
	_, err := conn.Exec(ctx, delete)
	if err != nil {
		return err
	}
	db.refreshTable(ctx, table)
	return nil
}

type ColumnType struct {
	Table       string `json:"table"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	DataType    string `json:"datatype"`
	IsArray     bool   `json:"isarray"`
	IsComposite bool   `json:"iscomposite"`
}

const columnTypesQuery = `
	SELECT
		c.table_schema || '.' || c.table_name tablename,
		c.column_name name,
		c.udt_name type,		
		c.data_type datatype,
		(t.typcategory = 'A') AS isarray,
		(t.typcategory = 'C') AS iscomposite
	FROM
		information_schema.columns c
		JOIN pg_type t ON c.udt_name = t.typname and c.udt_schema::regnamespace = t.typnamespace
	WHERE
		c.table_schema NOT IN ('pg_catalog', 'information_schema')
	ORDER BY
		table_schema, table_name, ordinal_position;
`

func (db *Database) GetColumnTypes(ctx context.Context) ([]ColumnType, error) {
	conn := GetConn(ctx)
	types := []ColumnType{}
	rows, err := conn.Query(ctx, columnTypesQuery)
	if err != nil {
		return types, err
	}
	defer rows.Close()

	typ := ColumnType{}
	for rows.Next() {
		err := rows.Scan(&typ.Table, &typ.Name, &typ.Type, &typ.DataType, &typ.IsArray, &typ.IsComposite)
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
