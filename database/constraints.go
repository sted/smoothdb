package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/samber/lo"
)

type Constraint struct {
	Name           string   `json:"name"`
	Type           byte     `json:"type"` // c: check, u: unique, p: primary, f: foreign
	Table          string   `json:"table"`
	Columns        []string `json:"columns"`
	RelatedTable   *string  `json:"reltable"`
	RelatedColumns []string `json:"relcolumns"`
	Definition     string   `json:"definition"`
}

type ForeignKey struct {
	Name           string
	Table          string
	Columns        []string
	RelatedTable   string
	RelatedColumns []string
}

type RelType int

const (
	O2M RelType = iota
	M2O
	O2O
)

type Relationship struct {
	Type           RelType
	Table          string
	Columns        []string
	RelatedTable   string
	RelatedColumns []string
}

// Query to retrieve all the Constraints in a database
const constraintsQuery = `
SELECT
    c.conname name,
    c.contype type, 
    ns1.nspname||'.'||cls1.relname table,
    columns.cols,
    ns2.nspname||'.'||cls2.relname ftable,
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

func getConstraints(ctx context.Context, conn *pgx.Conn, ftablename *string) ([]Constraint, error) {
	constraints := []Constraint{}
	query := constraintsQuery
	var args []any
	if ftablename != nil {
		schemaname, tablename := splitTableName(*ftablename)
		query += " WHERE cls1.relname = $1 AND ns1.nspname = $2"
		args = append(args, tablename, schemaname)
	} else {
		query += " WHERE ns1.nspname !~ '^pg_'"
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
			&constraint.Table, &constraint.Columns,
			&constraint.RelatedTable, &constraint.RelatedColumns,
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

func fillTableConstraints(table *Table, constraints []Constraint) {
	table.Constraints = nil
	for _, c := range constraints {
		if c.Table == table.Name && (len(c.Columns) > 1 || c.Type == 'p') {
			table.Constraints = append(table.Constraints, c.Definition)
		}
	}
}

func fillColumnConstraints(column *Column, constraints []Constraint) {
	column.Constraints = nil
	for _, c := range constraints {
		if c.Table == column.Table && len(c.Columns) == 1 && c.Columns[0] == column.Name {
			column.Constraints = append(column.Constraints, c.Definition)
		}
	}
}

func (db *Database) GetConstraints(ctx context.Context, tablename string) ([]Constraint, error) {
	conn := GetConn(ctx)
	constraints, err := getConstraints(ctx, conn.Conn(), &tablename)
	if err != nil {
		return nil, err
	}
	return constraints, nil
}

func (db *Database) CreateConstraint(ctx context.Context, constraint *Constraint) (*Constraint, error) {
	conn := GetConn(ctx)
	create := "ALTER TABLE " + constraint.Table + " ADD "
	create += constraint.Definition

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

func constraintToForeignKey(c *Constraint) *ForeignKey {
	return &ForeignKey{
		Name:           c.Name,
		Table:          c.Table,
		Columns:        c.Columns,
		RelatedTable:   *c.RelatedTable,
		RelatedColumns: c.RelatedColumns,
	}
}

func equalColumns(cols []string, otherCols []string) bool {
	if len(cols) != len(otherCols) {
		return false
	}
	for i := range cols {
		if cols[i] != otherCols[i] {
			return false
		}
	}
	return true
}

func foreignKeyToRelationships(fk *ForeignKey, pk *Constraint, uc []Constraint) [2]Relationship {
	var type1, type2 RelType
	var uniqueSource bool
	if pk != nil {
		if equalColumns(fk.Columns, pk.Columns) {
			uniqueSource = true
		}
	}
	for _, u := range uc {
		if equalColumns(fk.Columns, u.Columns) {
			uniqueSource = true
			break
		}
	}
	if uniqueSource {
		type1, type2 = O2O, O2O
	} else {
		type1, type2 = M2O, O2M
	}
	return [2]Relationship{
		{
			Type:           type1,
			Table:          fk.Table,
			Columns:        fk.Columns,
			RelatedTable:   fk.RelatedTable,
			RelatedColumns: fk.RelatedColumns,
		},
		{
			Type:           type2,
			Table:          fk.RelatedTable,
			Columns:        fk.RelatedColumns,
			RelatedTable:   fk.Table,
			RelatedColumns: fk.Columns,
		},
	}
}

func filterRelationships(rels []Relationship, relatedTable string) []Relationship {
	return lo.Filter(rels, func(rel Relationship, _ int) bool {
		return rel.RelatedTable == relatedTable
	})
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
