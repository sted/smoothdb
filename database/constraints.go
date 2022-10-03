package database

import (
	"context"
)

type Constraint struct {
	Name       string
	Type       byte // c: check, u: unique, p: primary, f: foreign
	Table      string
	Column     string
	NumCols    int
	Definition string
}

const constraintsQuery = `
	SELECT
		c.connamespace::regnamespace || '.' || cls.relname tablename,
		att.attname colname,
		c.conname name,
		c.contype type,
		cardinality(c.conkey) cols,
		pg_get_constraintdef(c.oid, true) def
	FROM pg_constraint c
		JOIN pg_class cls ON c.conrelid = cls.oid
		JOIN pg_attribute att ON c.conrelid = att.attrelid and c.conkey[1] = att.attnum`

func getConstraints(ctx context.Context, ftablename *string) ([]Constraint, error) {
	conn := GetConn(ctx)
	constraints := []Constraint{}
	query := constraintsQuery
	if ftablename != nil {
		schemaname, tablename := splitTableName(*ftablename)
		query += " WHERE cls.relname = '" + tablename + "' AND c.connamespace::regnamespace = '" + schemaname + "'::regnamespace"
	}
	query += " ORDER BY tablename, type"
	rows, err := conn.Query(ctx, query)
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

func fillColumnConstraints(column *Column, constraints []Constraint) {
	column.Check = ""
	column.Unique = false
	column.Primary = false
	column.Foreign = ""
	for _, c := range constraints {
		if c.Table == column.Table && c.Column == column.Name && c.NumCols == 1 {
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

func (db *Database) GetConstraints(ctx context.Context, tablename string) ([]Constraint, error) {
	constraints, err := getConstraints(ctx, &tablename)
	if err != nil {
		return nil, err
	}
	return constraints, nil
}

func (db *Database) CreateConstraint(ctx context.Context, constraint *Constraint) (*Constraint, error) {
	conn := GetConn(ctx)
	create := "ALTER TABLE " + constraint.Table + " ADD "
	if constraint.Name != "" {
		create += "CONSTRAINTS " + constraint.Name + " "
	}
	switch constraint.Type {
	case 'c':
		create += "CHECK "
	case 'u':
		create += "UNIQUE "
	case 'p':
		create += "PRIMARY "
	case 'f':
		create += "FOREIGN KEY "
	}
	create += "(" + constraint.Definition + ")"

	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	db.refreshTable(ctx, constraint.Table)
	return constraint, nil
}

func (db *Database) DeleteConstraint(ctx context.Context, table string, name string) error {
	conn := GetConn(ctx)
	_, err := conn.Exec(ctx, "ALTER TABLE "+table+" DROP CONSTRAINT "+name)
	if err != nil {
		return err
	}
	db.refreshTable(ctx, table)
	return nil
}
