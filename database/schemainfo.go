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
	Computed
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
	// Computed relationship fields
	FunctionName   string // function name (e.g., "read_principals")
	FunctionSchema string // function schema (e.g., "public")
	ReturnIsSet    bool   // whether the function returns SETOF
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
	// Views (added to cachedTables so they can participate in embedding)
	views, err := GetViews(ctx)
	if err != nil {
		return nil, err
	}
	for _, v := range views {
		fview := _s(v.Name, v.Schema)
		if _, exists := dbi.cachedTables[fview]; !exists {
			columns, _ := GetColumns(ctx, fview)
			dbi.cachedTables[fview] = Table{
				Name:    v.Name,
				Schema:  v.Schema,
				Owner:   v.Owner,
				Columns: columns,
			}
		}
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
	// View relationships
	viewColDeps, err := GetViewColDeps(ctx)
	if err != nil {
		return nil, err
	}
	dbi.addViewRelationships(viewColDeps)

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
	// Computed relationships: detect functions with a single IN argument of table type
	for _, f := range functions {
		// Count IN arguments (mode 0 means default=IN, 'i' means explicit IN)
		var inArgs []Argument
		for _, arg := range f.Arguments {
			if arg.Mode == 0 || arg.Mode == 'i' {
				inArgs = append(inArgs, arg)
			}
		}
		if len(inArgs) != 1 {
			continue
		}
		// The IN argument's type must be a table type
		argType, ok := dbi.cachedTypes[inArgs[0].TypeId]
		if !ok || !argType.IsTable {
			continue
		}
		// The return type must be a table or composite type
		retType, ok := dbi.cachedTypes[f.ReturnTypeId]
		if !ok || (!retType.IsTable && !retType.IsComposite) {
			continue
		}
		sourceTable := _s(argType.Name, argType.Schema)
		relatedTable := _s(retType.Name, retType.Schema)
		// ROWS 1 hint means the function effectively returns a single row (to-one)
		returnIsSet := f.ReturnIsSet && f.ReturnRows != 1
		rel := Relationship{
			Type:           Computed,
			Table:          sourceTable,
			RelatedTable:   relatedTable,
			FunctionName:   f.Name,
			FunctionSchema: f.Schema,
			ReturnIsSet:    returnIsSet,
		}
		dbi.cachedRelationships[sourceTable] = append(dbi.cachedRelationships[sourceTable], rel)
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
		if arrayEqualUnordered(fk.Columns, pk.Columns) {
			uniqueSource = true
		}
	}
	uc := si.cachedUniqueConstraints[ftable]
	for _, u := range uc {
		if arrayEqualUnordered(fk.Columns, u.Columns) {
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

// addViewRelationships detects FK-based relationships for views.
// For each FK (T1.cols → T2.refcols), a "surface" is any table or view that exposes
// those columns. This creates relationships between all surface pairs,
// skipping table↔table pairs (already handled by addRelationships).
func (si *SchemaInfo) addViewRelationships(deps []ViewColDep) {
	type baseCol struct{ table, col string }

	// surfaceMap: (baseTable, baseColumn) → list of surfaces exposing that column
	type surface struct {
		name, schema, colName string
		isView                bool
	}
	surfaceMap := map[baseCol][]surface{}

	// Add views as surfaces
	for _, d := range deps {
		key := baseCol{_s(d.BaseTable, d.BaseSchema), d.BaseColumn}
		surfaceMap[key] = append(surfaceMap[key], surface{d.ViewName, d.ViewSchema, d.ViewColumn, true})
	}
	// Add base tables themselves as surfaces
	for _, fkeys := range si.cachedForeignKeys {
		for _, fk := range fkeys {
			for _, col := range fk.Columns {
				key := baseCol{_s(fk.Table, fk.Schema), col}
				surfaceMap[key] = append(surfaceMap[key], surface{fk.Table, fk.Schema, col, false})
			}
			for _, col := range fk.RelatedColumns {
				key := baseCol{_s(fk.RelatedTable, fk.RelatedSchema), col}
				surfaceMap[key] = append(surfaceMap[key], surface{fk.RelatedTable, fk.RelatedSchema, col, false})
			}
		}
	}
	// Add M2M endpoint columns as table surfaces
	for _, rels := range si.cachedRelationships {
		for _, rel := range rels {
			if rel.Type != M2M {
				continue
			}
			tSchema, tName := splitTableName(rel.Table)
			for _, col := range rel.Columns {
				key := baseCol{rel.Table, col}
				surfaceMap[key] = append(surfaceMap[key], surface{tName, tSchema, col, false})
			}
			// Also add junction table columns
			jSchema, jTable := splitTableName(rel.JunctionTable)
			for _, col := range rel.JColumns {
				key := baseCol{rel.JunctionTable, col}
				surfaceMap[key] = append(surfaceMap[key], surface{jTable, jSchema, col, false})
			}
			for _, col := range rel.JRelatedColumns {
				key := baseCol{rel.JunctionTable, col}
				surfaceMap[key] = append(surfaceMap[key], surface{jTable, jSchema, col, false})
			}
		}
	}
	// Deduplicate table surfaces
	for key, surfaces := range surfaceMap {
		seen := map[string]bool{}
		deduped := surfaces[:0]
		for _, s := range surfaces {
			id := _s(s.name, s.schema) + "." + s.colName
			if !seen[id] {
				seen[id] = true
				deduped = append(deduped, s)
			}
		}
		surfaceMap[key] = deduped
	}

	// collectSurfaces returns surfaces that expose ALL given columns of a base table
	type surfaceEntry struct {
		name, schema string
		cols         []string
		isView       bool
	}
	collectSurfaces := func(table, schema string, cols []string) []surfaceEntry {
		fbase := _s(table, schema)
		type skey struct{ name, schema string }
		candidates := map[skey]*surfaceEntry{}
		for _, col := range cols {
			for _, s := range surfaceMap[baseCol{fbase, col}] {
				sk := skey{s.name, s.schema}
				if _, ok := candidates[sk]; !ok {
					candidates[sk] = &surfaceEntry{name: s.name, schema: s.schema, isView: s.isView}
				}
				candidates[sk].cols = append(candidates[sk].cols, s.colName)
			}
		}
		var result []surfaceEntry
		for _, se := range candidates {
			if len(se.cols) == len(cols) {
				result = append(result, *se)
			}
		}
		return result
	}

	// For each FK, create relationships between all surface pairs (skipping table↔table)
	for _, fkeys := range si.cachedForeignKeys {
		for _, fk := range fkeys {
			fbaseTable := _s(fk.Table, fk.Schema)
			var uniqueSource bool
			if pk, ok := si.cachedPrimaryKeys[fbaseTable]; ok {
				if arrayEqualUnordered(fk.Columns, pk.Columns) {
					uniqueSource = true
				}
			}
			for _, uc := range si.cachedUniqueConstraints[fbaseTable] {
				if arrayEqualUnordered(fk.Columns, uc.Columns) {
					uniqueSource = true
					break
				}
			}

			srcSurfaces := collectSurfaces(fk.Table, fk.Schema, fk.Columns)
			tgtSurfaces := collectSurfaces(fk.RelatedTable, fk.RelatedSchema, fk.RelatedColumns)

			for _, src := range srcSurfaces {
				for _, tgt := range tgtSurfaces {
					if !src.isView && !tgt.isView {
						continue // table↔table already handled
					}
					fsrc := _s(src.name, src.schema)
					ftgt := _s(tgt.name, tgt.schema)
					var type1, type2 RelType
					if uniqueSource {
						type1, type2 = O2O, O2O
					} else {
						type1, type2 = M2O, O2M
					}
					si.cachedRelationships[fsrc] = append(si.cachedRelationships[fsrc], Relationship{
						Type: type1, Table: fsrc, Columns: src.cols,
						RelatedTable: ftgt, RelatedColumns: tgt.cols, ForeignKey: fk.Name,
					})
					si.cachedRelationships[ftgt] = append(si.cachedRelationships[ftgt], Relationship{
						Type: type2, Table: ftgt, Columns: tgt.cols,
						RelatedTable: fsrc, RelatedColumns: src.cols, ForeignKey: fk.Name,
					})
				}
			}
		}
	}

	// M2M view relationships: for each existing M2M between base tables,
	// create M2M rels for view surfaces on both sides and the junction.
	// Only process each M2M pair once (deduplicate by junction + sorted table pair).
	seen := map[string]bool{}
	for _, rels := range si.cachedRelationships {
		for _, rel := range rels {
			if rel.Type != M2M {
				continue
			}
			// Deduplicate: use sorted table pair + junction to identify unique M2M
			pairKey := rel.Table + "|" + rel.RelatedTable + "|" + rel.JunctionTable
			if rel.Table > rel.RelatedTable {
				pairKey = rel.RelatedTable + "|" + rel.Table + "|" + rel.JunctionTable
			}
			if seen[pairKey] {
				continue
			}
			seen[pairKey] = true

			jSchema, jTable := splitTableName(rel.JunctionTable)
			tSchema, tName := splitTableName(rel.Table)
			rSchema, rName := splitTableName(rel.RelatedTable)

			tableSurfaces := collectSurfaces(tName, tSchema, rel.Columns)
			relSurfaces := collectSurfaces(rName, rSchema, rel.RelatedColumns)
			juncSurfaces := collectSurfaces(jTable, jSchema, append(rel.JColumns, rel.JRelatedColumns...))

			for _, ts := range tableSurfaces {
				for _, rs := range relSurfaces {
					for _, js := range juncSurfaces {
						if !ts.isView && !rs.isView && !js.isView {
							continue
						}
						fts := _s(ts.name, ts.schema)
						frs := _s(rs.name, rs.schema)
						fjs := _s(js.name, js.schema)
						// Skip if a M2M between these two endpoints already exists
						endpointKey := fts + "|" + frs
						if seen[endpointKey] {
							continue
						}
						seen[endpointKey] = true

						jCols := js.cols[:len(rel.JColumns)]
						jRelCols := js.cols[len(rel.JColumns):]

						si.cachedRelationships[fts] = append(si.cachedRelationships[fts], Relationship{
							Type: M2M, Table: fts, Columns: ts.cols,
							RelatedTable: frs, RelatedColumns: rs.cols,
							JunctionTable: fjs, JColumns: jCols, JRelatedColumns: jRelCols,
							ForeignKey: rel.ForeignKey,
						})
						si.cachedRelationships[frs] = append(si.cachedRelationships[frs], Relationship{
							Type: M2M, Table: frs, Columns: rs.cols,
							RelatedTable: fts, RelatedColumns: ts.cols,
							JunctionTable: fjs, JColumns: jRelCols, JRelatedColumns: jCols,
							ForeignKey: rel.ForeignKey,
						})
					}
				}
			}
		}
	}
}

func filterRelationships(rels []Relationship, relatedTable, fk string) []Relationship {
	return lo.Filter(rels, func(rel Relationship, _ int) bool {
		// Computed relationships are matched by function name, not by RelatedTable
		if rel.Type == Computed {
			return false
		}
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
