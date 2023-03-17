package database

import "github.com/samber/lo"

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

type DbInfo struct {
	cachedTables            map[string]Table
	cachedPrimaryKeys       map[string]Constraint
	cachedForeignKeys       map[string][]ForeignKey
	cachedUniqueConstraints map[string][]Constraint
	cachedCheckConstraints  map[string][]Constraint
	cachedRelationships     map[string][]Relationship
}

func (db *DbInfo) initDbInfo(tables []Table, constraints []Constraint) {
	db.cachedTables = map[string]Table{}
	db.cachedPrimaryKeys = map[string]Constraint{}
	db.cachedForeignKeys = map[string][]ForeignKey{}
	db.cachedUniqueConstraints = map[string][]Constraint{}
	db.cachedCheckConstraints = map[string][]Constraint{}
	db.cachedRelationships = map[string][]Relationship{}

	for _, t := range tables {
		db.cachedTables[t.Name] = t
	}

	for _, c := range constraints {
		switch c.Type {
		case 'p':
			db.cachedPrimaryKeys[c.Table] = c
		case 'f':
			// Here we skip foreign keys from or to partitions
			if !db.cachedTables[c.Table].IsPartition && !db.cachedTables[*c.RelatedTable].IsPartition {
				db.cachedForeignKeys[c.Table] = append(db.cachedForeignKeys[c.Table], *constraintToForeignKey(&c))
			}
		case 'u':
			db.cachedUniqueConstraints[c.Table] = append(db.cachedUniqueConstraints[c.Table], c)
		case 'c':
			db.cachedCheckConstraints[c.Table] = append(db.cachedCheckConstraints[c.Table], c)
		}
	}
	var pk *Constraint
	for t, fkeys := range db.cachedForeignKeys {

		c, ok := db.cachedPrimaryKeys[t]
		if ok {
			pk = &c
		} else {
			pk = nil
		}
		for _, fk := range fkeys {
			db.addRelationships(&fk, pk)
		}

		// check junction tables: for now when they have exactly two foreign keys @@
		if len(fkeys) == 2 && pk != nil && lo.Every(pk.Columns, fkeys[0].Columns) && lo.Every(pk.Columns, fkeys[1].Columns) {
			db.addM2MRelationships(fkeys)
		}
	}
}

func (db *DbInfo) GetPrimaryKey(table string) (Constraint, bool) {
	c, ok := db.cachedPrimaryKeys[table]
	return c, ok
}

func (db *DbInfo) GetForeignKeys(table string) []ForeignKey {
	return db.cachedForeignKeys[table]
}

func (db *DbInfo) GetRelationships(table string) []Relationship {
	return db.cachedRelationships[table]
}

func (db *DbInfo) FindRelationshipByCol(table, col string) *Relationship {
	rels := db.GetRelationships(table)
	for _, rel := range rels {
		if len(rel.Columns) == 1 && rel.Columns[0] == col {
			return &rel
		}
	}
	return nil
}

func (db *DbInfo) addRelationships(fk *ForeignKey, pk *Constraint) {
	table := fk.Table

	var uniqueSource bool
	if pk != nil {
		if arrayEquals(fk.Columns, pk.Columns) {
			uniqueSource = true
		}
	}
	uc := db.cachedUniqueConstraints[table]
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
	db.cachedRelationships[table] = append(db.cachedRelationships[table], rels[0])
	db.cachedRelationships[fk.RelatedTable] = append(db.cachedRelationships[fk.RelatedTable], rels[1])
}

func (db *DbInfo) addM2MRelationships(fkeys []ForeignKey) {
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
	db.cachedRelationships[table] = append(db.cachedRelationships[table], rels[0])
	db.cachedRelationships[relTable] = append(db.cachedRelationships[relTable], rels[1])
}

func filterRelationships(rels []Relationship, relatedTable string) []Relationship {
	return lo.Filter(rels, func(rel Relationship, _ int) bool {
		return rel.RelatedTable == relatedTable
	})
}
