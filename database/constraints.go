package database

import (
	"context"
	"regexp"
	"strings"

	"github.com/samber/lo"
)

type Constraint struct {
	Name       string
	Type       byte // c: check, u: unique, p: primary, f: foreign
	Table      string
	Column     string
	NumCols    int
	Definition string
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
)

type Relationship struct {
	Type           RelType
	Table          string
	Columns        []string
	RelatedTable   string
	RelatedColumns []string
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
	} else {
		query += " WHERE c.connamespace::regnamespace <> 'pg_catalog'::regnamespace"
		query += " AND c.connamespace::regnamespace <> 'information_schema'::regnamespace"
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

var re *regexp.Regexp = regexp.MustCompile(`^FOREIGN KEY \(([^\)]+)\) REFERENCES ([^\(]+)\(([^\)]+)\)`)

func constraintToForeignKey(c *Constraint) *ForeignKey {
	groups := re.FindStringSubmatch(c.Definition)
	cols := strings.Split(groups[1], ", ")
	refTable := groups[2]
	refCols := strings.Split(groups[3], ", ")
	return &ForeignKey{
		Name:           c.Name,
		Table:          c.Table,
		Columns:        cols,
		RelatedTable:   refTable,
		RelatedColumns: refCols,
	}
}

func constraintToRelationships(c *Constraint) [2]Relationship {
	fk := constraintToForeignKey(c)
	return [2]Relationship{
		{
			Type:           M2O,
			Table:          fk.Table,
			Columns:        fk.Columns,
			RelatedTable:   fk.RelatedTable,
			RelatedColumns: fk.RelatedColumns,
		},
		{
			Type:           O2M,
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

func getForeignKeys(ctx context.Context, tables []string) (fkeys []ForeignKey, err error) {
	conn := GetConn(ctx)
	query := constraintsQuery
	query += " WHERE c.contype = 'f' AND ("
	for i, table := range tables {
		if i > 0 {
			query += " OR"
		}
		schemaname, tablename := splitTableName(table)
		query += " cls.relname = '" + tablename + "' AND c.connamespace::regnamespace = '" + schemaname + "'::regnamespace"
	}
	query += ")"
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
		fkeys = append(fkeys, *constraintToForeignKey(&constraint))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return fkeys, nil
}
