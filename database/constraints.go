package database

import (
	"context"
)

type Constraint struct {
	Name           string   `json:"name"`
	Type           string   `json:"type"` // check, unique, primary, foreign
	Table          string   `json:"table"`
	Schema         string   `json:"schema"`
	Columns        []string `json:"columns"`
	RelatedTable   *string  `json:"reltable"`
	RelatedSchema  *string  `json:"relschema"`
	RelatedColumns []string `json:"relcolumns"`
	Definition     string   `json:"definition"`
}

type ForeignKey struct {
	Name           string
	Table          string
	Schema         string
	Columns        []string
	RelatedTable   string
	RelatedSchema  string
	RelatedColumns []string
}

// Query to retrieve all the Constraints in a database
const constraintsQuery = `
SELECT
    c.conname name,
    CASE c.contype
		WHEN 'p' THEN 'primary'
		WHEN 'u' THEN 'unique'
		WHEN 'c' THEN 'check'
		WHEN 'f' THEN 'foreign'
		ELSE ''
	END type,
    cls1.relname table,
	ns1.nspname schema,
    columns.cols,
    cls2.relname ftable,
	ns2.nspname fschema,
    columns.fcols,
    pg_get_constraintdef(c.oid, true) def
FROM pg_constraint c
JOIN LATERAL (
    SELECT
    	array_agg(cols.attname order by ord) cols,
    	coalesce(array_agg(fcols.attname order by ord) filter (where fcols.attname is not null), '{}') fcols
    FROM unnest(c.conkey, c.confkey) WITH ORDINALITY AS _(col, fcol, ord)
    JOIN pg_attribute cols ON cols.attrelid = c.conrelid AND cols.attnum = col
    LEFT JOIN pg_attribute fcols ON fcols.attrelid = c.confrelid AND fcols.attnum = fcol
) AS columns ON TRUE
JOIN pg_namespace ns1 ON ns1.oid = c.connamespace
JOIN pg_class cls1 ON cls1.oid = c.conrelid
LEFT JOIN pg_class cls2 ON cls2.oid = c.confrelid
LEFT JOIN pg_namespace ns2 ON ns2.oid = cls2.relnamespace`

func fillTableConstraints(table *Table, constraints []Constraint) {
	table.Constraints = nil
	for _, c := range constraints {
		if c.Table == table.Name && c.Schema == table.Schema && (len(c.Columns) > 1 || c.Type == "primary") {
			table.Constraints = append(table.Constraints, c.Definition)
		}
	}
}

func fillColumnConstraints(column *Column, constraints []Constraint) {
	column.Constraints = nil
	for _, c := range constraints {
		if c.Table == column.Table && c.Schema == column.Schema && len(c.Columns) == 1 && c.Columns[0] == column.Name {
			column.Constraints = append(column.Constraints, c.Definition)
		}
	}
}

func GetConstraints(ctx context.Context, tablename string) ([]Constraint, error) {
	conn, schemaname := GetConnAndSchema(ctx)

	constraints := []Constraint{}
	query := constraintsQuery
	var args []any
	if tablename != "" {
		query += " WHERE cls1.relname = $1 AND ns1.nspname = $2"
		args = append(args, tablename, schemaname)
	} else {
		query += " WHERE ns1.nspname NOT IN ('pg_catalog', 'information_schema')"
	}
	query += " ORDER BY cls1.relname"

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	constraint := Constraint{}
	for rows.Next() {
		err := rows.Scan(&constraint.Name, &constraint.Type,
			&constraint.Table, &constraint.Schema, &constraint.Columns,
			&constraint.RelatedTable, &constraint.RelatedSchema, &constraint.RelatedColumns,
			&constraint.Definition)
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

func CreateConstraint(ctx context.Context, constraint *Constraint) (*Constraint, error) {
	conn := GetConn(ctx)
	ftablename := composeName(ctx, constraint.Schema, constraint.Name)
	create := "ALTER TABLE " + ftablename + " ADD "
	create += constraint.Definition

	_, err := conn.Exec(ctx, create)
	if err != nil {
		return nil, err
	}
	//db.refreshTable(ctx, constraint.Table)
	return constraint, nil
}

func DeleteConstraint(ctx context.Context, tablename string, name string) error {
	conn, schemaname := GetConnAndSchema(ctx)
	ftablename := _sq(tablename, schemaname)

	_, err := conn.Exec(ctx, "ALTER TABLE "+ftablename+" DROP CONSTRAINT "+quote(name))
	if err != nil {
		return err
	}
	//db.refreshTable(ctx, table)
	return nil
}

func constraintToForeignKey(c *Constraint) *ForeignKey {
	return &ForeignKey{
		Name:           c.Name,
		Table:          c.Table,
		Schema:         c.Schema,
		Columns:        c.Columns,
		RelatedTable:   *c.RelatedTable,
		RelatedSchema:  *c.RelatedSchema,
		RelatedColumns: c.RelatedColumns,
	}
}

// func getForeignKeys(ctx context.Context, tables []string) (fkeys []ForeignKey, err error) {
// 	conn := GetConn(ctx)
// 	query := constraintsQuery
// 	query += " WHERE c.contype = 'f' AND ("
// 	for i, table := range tables {
// 		if i > 0 {
// 			query += " OR"
// 		}
// 		schemaname, tablename := splitTableName(table)
// 		query += " cls.relname = '" + tablename + "' AND n.nspname = '" + schemaname + "'"
// 	}
// 	query += ")"
// 	rows, err := conn.Query(ctx, query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	constraint := Constraint{}
// 	for rows.Next() {
// 		err := rows.Scan(&constraint.Name, &constraint.Type,
// 			&constraint.Table, &constraint.Columns, &constraint.ColSig, &constraint.Definition)
// 		if err != nil {
// 			return nil, err
// 		}
// 		fkeys = append(fkeys, *constraintToForeignKey(&constraint))
// 	}
// 	if err := rows.Err(); err != nil {
// 		return nil, err
// 	}
// 	return fkeys, nil
// }
