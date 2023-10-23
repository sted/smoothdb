package database

import (
	"context"

	"github.com/samber/lo"
)

type RelType int

const (
	O2M RelType = iota
	M2O
	O2O
	M2M
)

type Relationship struct {
	Type            RelType
	Table           string
	Columns         []string
	RelatedTable    string
	RelatedColumns  []string
	JunctionTable   string
	JColumns        []string
	JRelatedColumns []string
}

type SchemaInfo struct {
	cachedTypes             map[uint32]Type
	cachedComposites        []Type
	cachedTables            map[string]Table
	cachedColumnTypes       map[string]map[string]ColumnType
	cachedPrimaryKeys       map[string]Constraint
	cachedForeignKeys       map[string][]ForeignKey
	cachedUniqueConstraints map[string][]Constraint
	cachedCheckConstraints  map[string][]Constraint
	cachedRelationships     map[string][]Relationship
	cachedFunctions         map[string]Function
}

func NewSchemaInfo(ctx context.Context, db *Database) (*SchemaInfo, error) {
	dbi := &SchemaInfo{}
	dbi.cachedTypes = map[uint32]Type{}
	dbi.cachedTables = map[string]Table{}
	dbi.cachedColumnTypes = map[string]map[string]ColumnType{}
	dbi.cachedPrimaryKeys = map[string]Constraint{}
	dbi.cachedForeignKeys = map[string][]ForeignKey{}
	dbi.cachedUniqueConstraints = map[string][]Constraint{}
	dbi.cachedCheckConstraints = map[string][]Constraint{}
	dbi.cachedRelationships = map[string][]Relationship{}
	dbi.cachedFunctions = map[string]Function{}

	// Types
	types, err := GetTypes(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range types {
		dbi.cachedTypes[t.Id] = t
		if t.IsComposite {
			dbi.cachedComposites = append(dbi.cachedComposites, t)
		}
	}
	// Tables
	tables, err := db.GetTables(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range tables {
		//t.Columns, err = db.GetColumns(ctx, t.Name)
		dbi.cachedTables[t.Name] = t
	}
	// Column types
	colTypes, err := db.GetColumnTypes(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range colTypes {
		if _, ok := dbi.cachedColumnTypes[t.Table]; !ok {
			dbi.cachedColumnTypes[t.Table] = map[string]ColumnType{}
		}
		dbi.cachedColumnTypes[t.Table][t.Name] = t
	}
	// Constraints
	constraints, err := db.GetConstraints(ctx, "")
	if err != nil {
		return nil, err
	}
	for _, c := range constraints {
		switch c.Type {
		case 'p':
			dbi.cachedPrimaryKeys[c.Table] = c
		case 'f':
			// Here we skip foreign keys from or to partitions
			if !dbi.cachedTables[c.Table].IsPartition && !dbi.cachedTables[*c.RelatedTable].IsPartition {
				dbi.cachedForeignKeys[c.Table] = append(dbi.cachedForeignKeys[c.Table], *constraintToForeignKey(&c))
			}
		case 'u':
			dbi.cachedUniqueConstraints[c.Table] = append(dbi.cachedUniqueConstraints[c.Table], c)
		case 'c':
			dbi.cachedCheckConstraints[c.Table] = append(dbi.cachedCheckConstraints[c.Table], c)
		}
	}
	var pk *Constraint
	for t, fkeys := range dbi.cachedForeignKeys {

		c, ok := dbi.cachedPrimaryKeys[t]
		if ok {
			pk = &c
		} else {
			pk = nil
		}
		for _, fk := range fkeys {
			dbi.addRelationships(&fk, pk)
		}

		// check junction tables: for now when they have exactly two foreign keys @@
		if len(fkeys) == 2 && pk != nil && lo.Every(pk.Columns, fkeys[0].Columns) && lo.Every(pk.Columns, fkeys[1].Columns) {
			dbi.addM2MRelationships(fkeys)
		}
	}
	// Functions
	functions, err := db.GetFunctions(ctx)
	if err != nil {
		return nil, err
	}
	for _, f := range functions {
		if f.HasUnnamed {
			continue
		}
		dbi.cachedFunctions[f.Name] = f
	}
	return dbi, nil
}

func (si *SchemaInfo) GetTypeById(id uint32) *Type {
	t, ok := si.cachedTypes[id]
	if !ok {
		return nil
	}
	return &t
}

func (si *SchemaInfo) GetTable(table string) *Table {
	t, ok := si.cachedTables[table]
	if !ok {
		return nil
	}
	return &t
}

func (si *SchemaInfo) GetColumnType(table string, column string) *ColumnType {
	t, ok := si.cachedColumnTypes[table]
	if !ok {
		return nil
	}
	ct, ok := t[column]
	if !ok {
		return nil
	}
	return &ct
}

func (si *SchemaInfo) GetPrimaryKey(table string) *Constraint {
	c, ok := si.cachedPrimaryKeys[table]
	if !ok {
		return nil
	}
	return &c
}

func (si *SchemaInfo) GetForeignKeys(table string) []ForeignKey {
	return si.cachedForeignKeys[table]
}

func (si *SchemaInfo) GetRelationships(table string) []Relationship {
	return si.cachedRelationships[table]
}

func (si *SchemaInfo) FindRelationshipByCol(table, col string) *Relationship {
	rels := si.GetRelationships(table)
	for _, rel := range rels {
		if len(rel.Columns) == 1 && rel.Columns[0] == col {
			return &rel
		}
	}
	return nil
}

func (si *SchemaInfo) addRelationships(fk *ForeignKey, pk *Constraint) {
	table := fk.Table

	var uniqueSource bool
	if pk != nil {
		if arrayEquals(fk.Columns, pk.Columns) {
			uniqueSource = true
		}
	}
	uc := si.cachedUniqueConstraints[table]
	for _, u := range uc {
		if arrayEquals(fk.Columns, u.Columns) {
			uniqueSource = true
			break
		}
	}
	var type1, type2 RelType
	if uniqueSource {
		type1, type2 = O2O, O2O
	} else {
		type1, type2 = M2O, O2M
	}
	rels := [2]Relationship{
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
	si.cachedRelationships[table] = append(si.cachedRelationships[table], rels[0])
	si.cachedRelationships[fk.RelatedTable] = append(si.cachedRelationships[fk.RelatedTable], rels[1])
}

func (si *SchemaInfo) addM2MRelationships(fkeys []ForeignKey) {
	table := fkeys[0].RelatedTable
	relTable := fkeys[1].RelatedTable

	rels := [2]Relationship{
		{
			Type:            M2M,
			Table:           table,
			Columns:         fkeys[0].RelatedColumns,
			RelatedTable:    relTable,
			RelatedColumns:  fkeys[1].RelatedColumns,
			JunctionTable:   fkeys[0].Table,
			JColumns:        fkeys[0].Columns,
			JRelatedColumns: fkeys[1].Columns,
		},
		{
			Type:            M2M,
			Table:           relTable,
			Columns:         fkeys[1].RelatedColumns,
			RelatedTable:    table,
			RelatedColumns:  fkeys[0].RelatedColumns,
			JunctionTable:   fkeys[0].Table,
			JColumns:        fkeys[1].Columns,
			JRelatedColumns: fkeys[0].Columns,
		},
	}
	si.cachedRelationships[table] = append(si.cachedRelationships[table], rels[0])
	si.cachedRelationships[relTable] = append(si.cachedRelationships[relTable], rels[1])
}

func filterRelationships(rels []Relationship, relatedTable string) []Relationship {
	return lo.Filter(rels, func(rel Relationship, _ int) bool {
		return rel.RelatedTable == relatedTable
	})
}

func (si *SchemaInfo) GetFunction(name string) *Function {
	f, ok := si.cachedFunctions[name]
	if !ok {
		return nil
	}
	return &f
}
