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
	ForeignKey      string
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
	tables, err := GetTables(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range tables {
		ftable := _s(t.Name, t.Schema)
		t.Columns, _ = GetColumns(ctx, ftable)
		dbi.cachedTables[ftable] = t
	}
	// Column types
	colTypes, err := GetColumnTypes(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range colTypes {
		ftable := _s(t.Table, t.Schema)
		if _, ok := dbi.cachedColumnTypes[ftable]; !ok {
			dbi.cachedColumnTypes[ftable] = map[string]ColumnType{}
		}
		dbi.cachedColumnTypes[ftable][t.Name] = t
	}
	// Constraints
	constraints, err := GetConstraints(ctx, "")
	if err != nil {
		return nil, err
	}
	for _, c := range constraints {
		ftable := _s(c.Table, c.Schema)
		switch c.Type {
		case "primary":
			dbi.cachedPrimaryKeys[ftable] = c
		case "foreign":
			freltable := _s(*c.RelatedTable, *c.RelatedSchema)
			// Here we skip foreign keys from or to partitions
			if !dbi.cachedTables[ftable].IsPartition && !dbi.cachedTables[freltable].IsPartition {
				dbi.cachedForeignKeys[ftable] = append(dbi.cachedForeignKeys[ftable], *constraintToForeignKey(&c))
			}
		case "unique":
			dbi.cachedUniqueConstraints[ftable] = append(dbi.cachedUniqueConstraints[ftable], c)
		case "check":
			dbi.cachedCheckConstraints[ftable] = append(dbi.cachedCheckConstraints[ftable], c)
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
	functions, err := GetFunctions(ctx)
	if err != nil {
		return nil, err
	}
	for _, f := range functions {
		fname := _s(f.Name, f.Schema)
		if f.HasUnnamed {
			continue
		}
		dbi.cachedFunctions[fname] = f
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

func (si *SchemaInfo) GetTable(ftable string) *Table {
	t, ok := si.cachedTables[ftable]
	if !ok {
		return nil
	}
	return &t
}

func (si *SchemaInfo) GetColumnType(ftable string, column string) *ColumnType {
	t, ok := si.cachedColumnTypes[ftable]
	if !ok {
		return nil
	}
	ct, ok := t[column]
	if !ok {
		return nil
	}
	return &ct
}

func (si *SchemaInfo) GetPrimaryKey(ftable string) *Constraint {
	c, ok := si.cachedPrimaryKeys[ftable]
	if !ok {
		return nil
	}
	return &c
}

func (si *SchemaInfo) GetForeignKeys(ftable string) []ForeignKey {
	return si.cachedForeignKeys[ftable]
}

func (si *SchemaInfo) GetRelationships(ftable string) []Relationship {
	return si.cachedRelationships[ftable]
}

func (si *SchemaInfo) FindRelationshipByCol(ftable, col string) *Relationship {
	rels := si.GetRelationships(ftable)
	for _, rel := range rels {
		if len(rel.Columns) == 1 && rel.Columns[0] == col {
			return &rel
		}
	}
	for _, rel := range rels {
		if len(rel.RelatedColumns) == 1 && rel.RelatedColumns[0] == col {
			return &rel
		}
	}
	return nil
}

func (si *SchemaInfo) FindRelationshipByFK(ftable, fk string) *Relationship {
	rels := si.GetRelationships(ftable)
	for _, rel := range rels {
		if rel.ForeignKey == fk {
			return &rel
		}
	}
	return nil
}

func (si *SchemaInfo) addRelationships(fk *ForeignKey, pk *Constraint) {
	ftable := _s(fk.Table, fk.Schema)
	freltable := _s(fk.RelatedTable, fk.RelatedSchema)

	var uniqueSource bool
	if pk != nil {
		if arrayEquals(fk.Columns, pk.Columns) {
			uniqueSource = true
		}
	}
	uc := si.cachedUniqueConstraints[ftable]
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
			Table:          ftable,
			Columns:        fk.Columns,
			RelatedTable:   freltable,
			RelatedColumns: fk.RelatedColumns,
			ForeignKey:     fk.Name,
		},
		{
			Type:           type2,
			Table:          freltable,
			Columns:        fk.RelatedColumns,
			RelatedTable:   ftable,
			RelatedColumns: fk.Columns,
			ForeignKey:     fk.Name,
		},
	}
	si.cachedRelationships[ftable] = append(si.cachedRelationships[ftable], rels[0])
	si.cachedRelationships[freltable] = append(si.cachedRelationships[freltable], rels[1])
}

func (si *SchemaInfo) addM2MRelationships(fkeys []ForeignKey) {
	ftable := _s(fkeys[0].RelatedTable, fkeys[0].RelatedSchema)
	freltable := _s(fkeys[1].RelatedTable, fkeys[1].RelatedSchema)
	fjtable := _s(fkeys[0].Table, fkeys[0].Schema)

	rels := [2]Relationship{
		{
			Type:            M2M,
			Table:           ftable,
			Columns:         fkeys[0].RelatedColumns,
			RelatedTable:    freltable,
			RelatedColumns:  fkeys[1].RelatedColumns,
			JunctionTable:   fjtable,
			JColumns:        fkeys[0].Columns,
			JRelatedColumns: fkeys[1].Columns,
			ForeignKey:      fkeys[0].Name,
		},
		{
			Type:            M2M,
			Table:           freltable,
			Columns:         fkeys[1].RelatedColumns,
			RelatedTable:    ftable,
			RelatedColumns:  fkeys[0].RelatedColumns,
			JunctionTable:   fjtable,
			JColumns:        fkeys[1].Columns,
			JRelatedColumns: fkeys[0].Columns,
			ForeignKey:      fkeys[1].Name,
		},
	}

	si.cachedRelationships[ftable] = append(si.cachedRelationships[ftable], rels[0])
	si.cachedRelationships[freltable] = append(si.cachedRelationships[freltable], rels[1])
}

func filterRelationships(rels []Relationship, relatedTable, fk string) []Relationship {
	return lo.Filter(rels, func(rel Relationship, _ int) bool {
		return rel.RelatedTable == relatedTable && (fk == "" || rel.ForeignKey == fk)
	})
}

func (si *SchemaInfo) GetFunction(name string) *Function {
	f, ok := si.cachedFunctions[name]
	if !ok {
		return nil
	}
	return &f
}
