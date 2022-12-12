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
	constraints, err := getConstraints(ctx, conn.Conn(), &ftablename)
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
		if nullable == "NO" {
			column.NotNull = true
		} else {
			column.NotNull = false
		}
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
	constraints, err := getConstraints(ctx, conn.Conn(), &ftablename)
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
	if nullable == "NO" {
		column.NotNull = true
	} else {
		column.NotNull = false
	}
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
