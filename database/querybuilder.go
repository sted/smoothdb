package database

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/samber/lo"
)

type QueryBuilder interface {
	BuildSelect(table string, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (string, []any, error)
	BuildInsert(table string, records []Record, body []byte, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (string, []any, error)
	BuildUpdate(table string, record Record, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (string, []any, error)
	BuildDelete(table string, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (string, []any, error)
	// BuildExecute calls the function name, whose overload f (nil when unknown)
	// is the one SchemaInfo.FindFunction resolves for the record's keys
	BuildExecute(name string, f *Function, record Record, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (string, []any, error)

	preferredSerializer() TextSerializer
}

// Join containts information to create a join relationship.
// It is composed while building the select clause.
type Join struct {
	name         string        // join name, crafted to be unique
	fields       string        // fields to be selected in the related table (eg. a, b::text, c AS C)
	nested       string        // nested joins
	inner        bool          // is a inner join
	rel          *Relationship // base relationship
	relLabel     string        // label for the relationship, taken from the select clause
	relName      string        // relation/function name as it appears in the query
	selectFields []SelectField // inner select fields (for related order validation)
	innerConds   []string      // an EXISTS per !inner embed of this one, for the count (see selectForJoinClause)
}

type BuildError struct {
	msg string // description of error
}

func (e BuildError) Error() string { return e.msg }

// BuildStack represents the context when navigating the AST produced by the parser
type BuildStack struct {
	info            *SchemaInfo       // database information (tables, contraints, etc)
	level           int               // depth
	relPath         []string          // sequence of nested tables
	labelPath       []string          // sequence of nested labels for the correspondent tables (can contain empty strings)
	colPath         []string          // sequence of FK column names that point to the correspondent tables (can contain empty strings)
	namePath        []string          // sequence of the names the select embeds each level with (a table, an FK column, a constraint, a function)
	siblingPath     []map[string]bool // per level, the names and aliases of every embed of the select that embeds it
	siblings        map[string]bool   // the names and aliases of the embeds of the select being built, for the next level
	afterWithClause bool              //
	maxRows         int64             // a read's cap on the rows of each to-many embed (see maxRows), 0 for none
	st              *buildState       // what the build owns: its values and what it knows of the filters
	countChain      bool              // the total of a ranged count reads this level: the root, or an embed reached through !inner embeds only
	rootAlias       string            // the name the statement reads the root columns through (the CTE of a recursive select), "" for the table
	pseudoAlias     string            // the same for __depth and __path only (the __dedup of a via recursion)
}

// buildState is what one build of a request owns: the values it binds and
// what it knows of each filter. The parsed request (QueryParts) stays
// unchanged, so a second build of it gives the same SQL and values.
type buildState struct {
	page      binder    // the statement the build returns
	count     *binder   // the total of a ranged count, when it is a statement of its own (countApart)
	countMode countMode // how the total reads the !inner embeds
	levels    int       // the levels whose WHERE the page has built (see pageWhere)
	rootRow   bool      // a computed embed reads the root row from the CTE of a recursive select
	filterBuf [8]filterState
}

// countMode tells how the total of a ranged count reads the filters of the
// !inner embeds (see buildAfterSelect).
type countMode uint8

const (
	countNone   countMode = iota // no total of a ranged count
	countInPage                  // in the page statement, with the page's markers
	countApart                   // in a statement of its own, with its own markers
)

// binder collects the values one statement binds: the marker $n is
// values[n-base-1].
type binder struct {
	values  []any
	base    int
	filters []filterState // the page's, per filter node (see WhereConditionNode.id)
	level   int           // the level whose WHERE the page builds
	// the count of a level takes the filters that the page took at that level
	replay      *binder
	replayLevel int
}

// filterState is what the page knows of one filter node.
type filterState struct {
	level   int // the level that took it, 0 when none did
	matched int // how many leading elements of field.relPath a level matched (see checkFiltersApplied)
}

// newBuildState starts a build whose statement binds values first.
func newBuildState(parts *QueryParts, values []any) *buildState {
	st := &buildState{}
	st.page.values = values
	if n := parts.nodeCount + 1; n <= len(st.filterBuf) {
		st.page.filters = st.filterBuf[:n]
	} else {
		st.page.filters = make([]filterState, n)
	}
	return st
}

// filter returns the state of a filter node.
func (b *binder) filter(n *WhereConditionNode) *filterState {
	if n.id >= len(b.filters) { // a tree built without the parser
		b.filters = append(b.filters, make([]filterState, n.id+1-len(b.filters))...)
	}
	return &b.filters[n.id]
}

// bind appends the marker of a new value to where.
func (b *binder) bind(where string, v any) string {
	b.values = append(b.values, v)
	return where + "$" + strconv.Itoa(b.base+len(b.values))
}

// pageWhere builds the WHERE of one level of the page and returns its level.
func (st *buildState) pageWhere(table, schema, label string, node *WhereConditionNode, stack BuildStack) (string, int) {
	st.levels++
	st.page.level = st.levels
	return whereClause(table, schema, label, node, &st.page, stack), st.levels
}

// countWhere builds the WHERE of a level for a total of its own (countApart):
// the filters the page took at that level, with the count's markers.
func (st *buildState) countWhere(table, schema, label string, node *WhereConditionNode, stack BuildStack, level int) string {
	st.count.replay, st.count.replayLevel = &st.page, level
	return whereClause(table, schema, label, node, st.count, stack)
}

// rootAliasFor returns the alias the statement reads a root column through,
// "" for the table itself.
func (stack BuildStack) rootAliasFor(name string) string {
	if stack.pseudoAlias != "" && (name == "__depth" || name == "__path") {
		return stack.pseudoAlias
	}
	return stack.rootAlias
}

// rootColumn is a column of the root relation, as the statement reads it. A
// dotted name ("a.b" in the URL) is one column, as in PostgREST.
func (stack BuildStack) rootColumn(table, schema, name string) string {
	if alias := stack.rootAliasFor(name); alias != "" {
		return quote(alias) + "." + quote(name)
	}
	return _sq(table, schema) + "." + quote(name)
}

// nextBuildStack creates a stack with a new level: the embed of the table rel
// (its label, the FK column pointing to it, the name the select uses for it),
// among the siblings the current select embeds. The paths are clipped, so that
// sibling levels never share, and overwrite, a backing array.
func nextBuildStack(stack BuildStack, rel string, label string, col string, name string) BuildStack {
	return BuildStack{
		info:        stack.info,
		level:       stack.level + 1,
		relPath:     append(slices.Clip(stack.relPath), rel),
		labelPath:   append(slices.Clip(stack.labelPath), label),
		colPath:     append(slices.Clip(stack.colPath), col),
		namePath:    append(slices.Clip(stack.namePath), name),
		siblingPath: append(slices.Clip(stack.siblingPath), stack.siblings),
		maxRows:     stack.maxRows,
		st:          stack.st,
	}
}

// maxRows is Database.MaxRows, PostgREST's db-max-rows: the most rows a read
// takes, at its top level and in each to-many embed (Plan.hs
// treeRestrictRange restricts every node of a read plan), from its offset and
// whatever limit it asks. The total of a count and the representation of a
// write are not capped. 0 is no cap.
func maxRows() int64 {
	if dbe == nil || dbe.config.MaxRows < 0 {
		return 0
	}
	return dbe.config.MaxRows
}

// matchPath returns how many leading elements of an embed path (the x.y of a
// filter x.y.col or of an order x.y.order) the levels of the stack match.
func (stack BuildStack) matchPath(path []string) int {
	matched := 0
	for i := range path {
		if i >= len(stack.relPath) || !stack.levelMatches(i, path[i]) {
			break
		}
		matched++
	}
	return matched
}

// levelMatches reports whether an element of an embed path names the level i
// of the stack. As in PostgREST (Plan.hs, the legacy target matching) it names
// the level by the name the select embeds it with or by its alias. smoothdb
// also accepts the embedded table or the FK column, but only when no embed of
// that select is named so: the parent and the children of a self-reference
// both embed the same table, and web_content.name must reach the children
// (web_content), not the parent (parent:p_web_id).
func (stack BuildStack) levelMatches(i int, name string) bool {
	if (i < len(stack.namePath) && name == stack.namePath[i]) ||
		(stack.labelPath[i] != "" && name == stack.labelPath[i]) {
		return true
	}
	if name != stack.relPath[i] && (len(stack.colPath) <= i || name != stack.colPath[i]) {
		return false
	}
	return i >= len(stack.siblingPath) || !stack.siblingPath[i][name]
}

// toJson wraps a field into a to_jsonb operator if its type is array or composite,
// otherwise it returns it unchanged
func toJson(table, schema, field, quotedField string, info *SchemaInfo) string {
	if info == nil {
		return quotedField // to enable building queries without schema info
	}
	ftablename := _s(table, schema)
	typ := info.GetColumnType(ftablename, field)
	if typ != nil && (typ.IsArray || typ.IsComposite) {
		quotedField = "to_jsonb(" + quotedField + ")"
	}
	return quotedField
}

func prepareField(table, schema string, sfield SelectField, info *SchemaInfo) string {
	return prepareFieldAs(table, schema, table, schema, sfield, info)
}

// prepareFieldAs builds the select-list item of a field read through an alias
// (a label, a nesting level, the _source CTE of a mutation) while the column
// type is still looked up on the real table: the json path of an array or
// composite column needs its to_jsonb wrapper whatever the prefix.
func prepareFieldAs(alias, aliasSchema, table, schema string, sfield SelectField, info *SchemaInfo) string {
	var fieldPart string

	if sfield.aggregate == "" {
		// Regular field without aggregate
		fieldname := _sq(alias, aliasSchema) + "." + quoteIf(sfield.field.name, !isStar(sfield.field.name))
		if sfield.field.jsonPath != "" {
			fieldname = toJson(table, schema, sfield.field.name, fieldname, info)
			fieldname = "(" + fieldname + sfield.field.jsonPath + ")"
		}
		// Apply field cast for regular fields (no extra parentheses needed)
		if sfield.cast != "" {
			fieldname = fieldname + "::" + sfield.cast
		}
		fieldPart = fieldname
	} else {
		// For COUNT() without a field (standalone count)
		if sfield.field.name == "*" && sfield.aggregate == "COUNT" {
			fieldPart = "COUNT(*)"
		} else {
			// For aggregates with specific fields
			fieldname := _sq(alias, aliasSchema) + "." + quoteIf(sfield.field.name, !isStar(sfield.field.name))
			if sfield.field.jsonPath != "" {
				fieldname = toJson(table, schema, sfield.field.name, fieldname, info)
				fieldname = "(" + fieldname + sfield.field.jsonPath + ")"
			}
			// Apply field cast before aggregation (need parentheses for complex expressions)
			if sfield.cast != "" {
				// Only add parentheses if the fieldname doesn't already have them (JSON paths already wrapped)
				if strings.HasPrefix(fieldname, "(") {
					fieldname = fieldname + "::" + sfield.cast
				} else {
					fieldname = "(" + fieldname + ")::" + sfield.cast
				}
			}
			fieldPart = sfield.aggregate + "(" + fieldname + ")"
		}
		// Apply aggregate cast after aggregation
		if sfield.aggCast != "" {
			fieldPart = fieldPart + "::" + sfield.aggCast
		}
	}

	if sfield.label != "" {
		fieldPart += " AS " + quote(sfield.label)
	}
	return fieldPart
}

func labelWithNumber(table string, num int) string {
	return table + "_" + strconv.Itoa(num)
}

// selectForJoinClause builds the subquery of an embed, sel, and the same rows
// for the total of a ranged count, count: PostgREST's readPlanToCountQuery,
// SELECT 1 with the embed's join condition and filters, and an EXISTS per
// !inner embed of its own instead of the nested joins, at every level. Neither
// the select list nor the order reach count: an aggregate there would always
// yield its one row. Both come from one pass, since whereClause takes each
// filter once.
func selectForJoinClause(join Join, label string, parts *QueryParts, stack BuildStack) (sel, count string, err error) {
	rel := join.rel
	st := stack.st
	// the count of this embed is read only along !inner embeds from the root
	withCount := join.inner && stack.countChain
	_, t1 := splitTableName(rel.RelatedTable)
	label1 := quote(labelWithNumber(t1, stack.level+1))
	var label2 string
	if stack.level > 0 {
		_, t2 := splitTableName(rel.Table)
		label2 = quote(labelWithNumber(t2, stack.level))
	} else if stack.rootAlias != "" {
		label2 = quote(stack.rootAlias)
	} else if label != "" {
		label2 = label
	} else {
		label2 = quoteParts(rel.Table)
	}

	var from, oc string
	var conds, countConds []string
	// where adds the filters of the embed to the page and, when it is
	// counted, to the count
	where := func(table, schema string, embedStack BuildStack) {
		wc, level := st.pageWhere(table, schema, join.relLabel, parts.whereConditionsTree, embedStack)
		if wc != "" {
			conds = append(conds, wc)
		}
		if withCount && st.countMode == countApart {
			wc = st.countWhere(table, schema, join.relLabel, parts.whereConditionsTree, embedStack, level)
		}
		if withCount && wc != "" {
			countConds = append(countConds, wc)
		}
	}
	if rel.Type == Computed {
		// Computed relationship: call the function with parent row reference
		// Use the unqualified table name as the row reference (the implicit alias from outer FROM),
		// with a qualified cast for the type. This is needed because in nested subqueries
		// (inside json_agg wrapper), the schema-qualified reference doesn't resolve as a row ref.
		var parentRef string
		parentType := "::" + quoteParts(rel.Table)
		if stack.afterWithClause {
			parentRef = quote("_source")
		} else if stack.level == 0 && stack.rootAlias != "" {
			// the CTE of a recursive select carries the root row as __row,
			// already of the table's type
			parentRef, parentType = quote(stack.rootAlias)+"."+quote("__row"), ""
			st.rootRow = true
		} else if stack.level > 0 {
			_, t2 := splitTableName(rel.Table)
			parentRef = quote(labelWithNumber(t2, stack.level))
		} else if label != "" {
			parentRef = label
		} else {
			_, t2 := splitTableName(rel.Table)
			parentRef = quote(t2)
		}
		funcRef := _sq(rel.FunctionName, rel.FunctionSchema)
		from = " FROM " + funcRef + "(" + parentRef + parentType + ")"
		from += " AS " + label1
		// Apply embedded resource filters and orders (use function name for relPath matching)
		schema, table := splitTableName(rel.RelatedTable)
		stackRelName := join.relName
		embedStack := nextBuildStack(stack, stackRelName, join.relLabel, "", join.relName)
		where(table, schema, embedStack)
		oc, err = levelOrderClause(table, schema, label1, parts.orderFields, join.selectFields, embedStack)
		if err != nil {
			return "", "", err
		}
	} else {
		// FK-based relationship
		from = " FROM " + quoteParts(rel.RelatedTable)
		from += " AS " + label1
		if rel.JunctionTable != "" {
			from += ", " + quoteParts(rel.JunctionTable)
		}
		var on string
		if rel.JunctionTable == "" {
			for i := range rel.Columns {
				if i != 0 {
					on += " AND "
				}
				on += label1 + "." + quote(rel.RelatedColumns[i])
				on += " = "
				if stack.afterWithClause {
					on += quote("_source")
				} else {
					on += label2
				}
				on += "." + quote(rel.Columns[i])
			}
		} else {
			// M2M Join
			for i := range rel.JColumns {
				if i != 0 {
					on += " AND "
				}
				on += quoteParts(rel.JunctionTable) + "." + quote(rel.JColumns[i])
				on += " = "
				if stack.afterWithClause {
					on += quote("_source")
				} else {
					on += label2
				}
				on += "." + quote(rel.Columns[i])
			}
			for i := range rel.JRelatedColumns {
				on += " AND "
				on += quoteParts(rel.JunctionTable) + "." + quote(rel.JRelatedColumns[i])
				on += " = "
				on += label1 + "." + quote(rel.RelatedColumns[i])
			}
		}
		conds = append(conds, on)
		if withCount {
			countConds = append(countConds, on)
		}
		// where and order clause for the internal select: the expressions related to
		// the external query are skipped inside the functions.
		// A self-reference is filtered and ordered like any other embed: the
		// filters and the orders are matched by embed path, and the related
		// side has its own alias.
		schema, table := splitTableName(rel.RelatedTable)
		col := ""
		if len(rel.Columns) == 1 {
			col = rel.Columns[0]
		}
		embedStack := nextBuildStack(stack, table, join.relLabel, col, join.relName)
		where(table, schema, embedStack)
		oc, err = levelOrderClause(table, schema, label1, parts.orderFields, join.selectFields, embedStack)
		if err != nil {
			return "", "", err
		}
	}

	sel = " SELECT " + join.fields + from
	if join.nested != "" {
		sel += " " + join.nested
	}
	if len(conds) > 0 {
		sel += " WHERE " + strings.Join(conds, " AND ")
	}
	if oc != "" {
		sel += " ORDER BY " + oc
	}
	if withCount {
		count = " SELECT 1" + from
		if countConds = append(countConds, join.innerConds...); len(countConds) > 0 {
			count += " WHERE " + strings.Join(countConds, " AND ")
		}
	}
	return sel, count, nil
}

func findRelationship(table, relation, fk, schema string, info *SchemaInfo) (rel *Relationship, err error) {
	tableWithSchema := _s(table, schema)
	relationWithSchema := _s(relation, schema)
	rels := info.GetRelationships(tableWithSchema)
	frels := filterRelationships(rels, relationWithSchema, fk)
	nrels := len(frels)
	switch {
	case nrels == 0:
		// Try computed relationships by function name
		for i := range rels {
			if rels[i].Type == Computed && rels[i].FunctionName == relation &&
				(schema == "" || rels[i].FunctionSchema == schema) {
				return &rels[i], nil
			}
		}
		if fk != "" {
			// the hint names no constraint: it can name a column of a
			// relationship to the relation named, and only to that one
			hrels := info.FindRelationshipsByHintCol(tableWithSchema, relationWithSchema, fk)
			if len(hrels) > 1 {
				return nil, &BuildError{"more than one possible relationship for table " + table + " with table " + relation}
			}
			if len(hrels) == 1 {
				rel = &hrels[0]
			}
		} else {
			// search rel by column (try to see if relation is instead an fk column)
			rel = info.FindRelationshipByCol(tableWithSchema, relation)
			if rel == nil {
				// search rel by fk constraint (try to see if relation is instead an fk constraint)
				rel = info.FindRelationshipByFK(tableWithSchema, relation)
			}
		}
		if rel == nil {
			return nil, &BuildError{"cannot find relationship for table " + table + " with table " + relation}
		}
	case nrels == 1:
		// ok, found a single relationship
		rel = &frels[0]
	case nrels == 2 && table == relation:
		// a self relationship, we prioritize the O2M one
		rel = &frels[1]
	default:
		return nil, &BuildError{"more than one possible relationship for table " + table + " with table " + relation}
	}
	return rel, nil
}

// selectClause prepares the select clause of a query and builds the discovered joins.
// The label parameter is used to construct a query for the execution of functions.
// innerConds is an EXISTS per !inner embed of this level that the total of a
// ranged count reads (see selectForJoinClause, buildAfterSelect).
func selectClause(table, schema, label string, parts *QueryParts, stack BuildStack) (
	selClause string, joins string, keys []string, innerConds []string, err error) {

	var parentTable string
	var relatedTable string
	var labelRelName string
	var joinName string

	joinSeq := []Join{}

	// the names and aliases of this select's embeds, which an embed path
	// prefers to the embedded tables (see levelMatches)
	stack.siblings = map[string]bool{}
	for _, sfield := range parts.selectFields {
		if sfield.relation != nil {
			stack.siblings[sfield.relation.name] = true
			if sfield.label != "" {
				stack.siblings[sfield.label] = true
			}
		}
	}

	for _, sfield := range parts.selectFields {
		if sfield.relation != nil {
			if sfield.relation.parent != "" {
				parentTable = sfield.relation.parent
			} else {
				parentTable = table
			}
			frel, err := findRelationship(parentTable, sfield.relation.name, sfield.relation.fk, schema, stack.info)
			if err != nil {
				return "", "", nil, nil, err
			}
			if sfield.label == "" {
				labelRelName = sfield.relation.name
			} else {
				labelRelName = sfield.label
			}
			joinName = labelWithNumber(parentTable+"_"+labelRelName, stack.level+1)

			if len(sfield.relation.fields) != 0 {
				// an empty embed adds no column, so the separator follows the
				// columns written so far
				if selClause != "" {
					selClause += ", "
				}
				switch frel.Type {
				case M2O, O2O:
					if sfield.relation.spread {
						selClause += quote(joinName) + ".*"
					} else {
						selClause += " row_to_json(" + quote(joinName) + ".*) AS " + quote(labelRelName)
					}
				case O2M, M2M:
					if sfield.relation.spread {
						err = &BuildError{"A spread operation on " + sfield.relation.name + " is not possible"}
						return "", "", nil, nil, err
					}
					selClause += " COALESCE(" + quote(joinName) + "." + quote("_"+joinName) + ", '[]') AS " + quote(labelRelName)
				case Computed:
					if frel.ReturnIsSet {
						if sfield.relation.spread {
							err = &BuildError{"A spread operation on " + sfield.relation.name + " is not possible"}
							return "", "", nil, nil, err
						}
						selClause += " COALESCE(" + quote(joinName) + "." + quote("_"+joinName) + ", '[]') AS " + quote(labelRelName)
					} else {
						if sfield.relation.spread {
							selClause += quote(joinName) + ".*"
						} else {
							selClause += " row_to_json(" + quote(joinName) + ".*) AS " + quote(labelRelName)
						}
					}
				}
			}
			_, relatedTable = splitTableName(frel.RelatedTable)
			// the filters and the orders of every level travel down: each level
			// takes those whose embed path names it
			internalParts := &QueryParts{selectFields: sfield.relation.fields, whereConditionsTree: parts.whereConditionsTree, orderFields: parts.orderFields}
			// For computed relationships, use the function name in the relPath so WHERE filters match
			stackRelName := relatedTable
			if frel.Type == Computed {
				stackRelName = sfield.relation.name
			}
			col := ""
			if frel.Type != Computed && len(frel.Columns) == 1 {
				col = frel.Columns[0]
			}
			childStack := nextBuildStack(stack, stackRelName, sfield.label, col, sfield.relation.name)
			childStack.countChain = stack.countChain && sfield.relation.inner
			sc, j, _, childInner, err := selectClause(relatedTable, schema, "", internalParts, childStack)
			if err != nil {
				return "", "", nil, nil, err
			}
			joinSeq = append(joinSeq, Join{joinName, sc, j, sfield.relation.inner, frel, sfield.label, sfield.relation.name, sfield.relation.fields, childInner})
		} else {
			if selClause != "" {
				selClause += ", "
			}
			var fieldPart string
			if !stack.afterWithClause {
				if stack.level == 0 {
					if alias := stack.rootAliasFor(sfield.field.name); alias != "" {
						fieldPart = prepareFieldAs(alias, "", table, schema, sfield, stack.info)
					} else if label == "" {
						fieldPart = prepareField(table, schema, sfield, stack.info)
					} else {
						fieldPart = prepareFieldAs(label, "", table, schema, sfield, stack.info)
					}
				} else {
					fieldPart = prepareFieldAs(labelWithNumber(table, stack.level), "", table, schema, sfield, stack.info)
				}
			} else {
				fieldPart = prepareFieldAs("_source", "", table, schema, sfield, stack.info)
			}
			selClause += fieldPart
		}
	}
	if selClause == "" {
		selClause = "*"
	}
	if len(joinSeq) > 0 {
		for _, join := range joinSeq {
			// the join name carries the user's label: quote it, never wrap it
			qName, qAgg := quote(join.name), quote("_"+join.name)
			selectForJoin, count, err := selectForJoinClause(join, label, parts, stack)
			if err != nil {
				return "", "", nil, nil, err
			}
			if count != "" {
				// one EXISTS per !inner embed, for the total of a ranged count:
				// the root's restrict it, a nested level's go into its
				// parent's (see selectForJoinClause)
				innerConds = append(innerConds, "EXISTS ("+count+")")
			}
			if join.inner {
				joins += " INNER"
			} else {
				joins += " LEFT"
			}
			joins += " JOIN LATERAL ("
			switch join.rel.Type {
			case M2O, O2O:
				joins += selectForJoin
			case O2M, M2M:
				joins += " SELECT json_agg(" + qAgg + ") AS " + qAgg
				joins += " FROM ("
				joins += selectForJoin + embedLimit(stack)
				joins += " ) AS " + qAgg
			case Computed:
				if join.rel.ReturnIsSet {
					joins += " SELECT json_agg(" + qAgg + ") AS " + qAgg
					joins += " FROM ("
					joins += selectForJoin + embedLimit(stack)
					joins += " ) AS " + qAgg
				} else {
					joins += selectForJoin
				}
			}
			joins += ") AS " + qName + " ON"
			if join.inner && (join.rel.Type == O2M || join.rel.Type == M2M || (join.rel.Type == Computed && join.rel.ReturnIsSet)) {
				joins += " " + qName + " IS NOT NULL"
			} else {
				joins += " TRUE"
			}
			// No FK columns needed for computed relationships.
			// Keys are bare column names: the caller qualifies them as needed.
			if join.rel.Type != Computed {
				keys = append(keys, join.rel.Columns...)
			}
		}
	}
	return selClause, joins, keys, innerConds, nil
}

// embedLimit is the LIMIT of a to-many embed of a read under maxRows
func embedLimit(stack BuildStack) string {
	if stack.maxRows <= 0 {
		return ""
	}
	return " LIMIT " + strconv.FormatInt(stack.maxRows, 10)
}

// groupByClause creates a GROUP BY clause when aggregate functions are present
func groupByClause(table, schema string, parts *QueryParts, info *SchemaInfo) string {
	return groupByClauseAs(table, schema, table, schema, parts, info)
}

// groupByClauseAs is groupByClause for fields read through an alias (the t of
// a function call), the column types still looked up on the real relation,
// as prepareFieldAs does for the select list.
func groupByClauseAs(alias, aliasSchema, table, schema string, parts *QueryParts, info *SchemaInfo) string {
	// If no select fields are specified, no GROUP BY needed
	if len(parts.selectFields) == 0 {
		return ""
	}

	var hasAggregates bool
	var nonAggregateFields []string

	for _, sfield := range parts.selectFields {
		if sfield.aggregate != "" {
			hasAggregates = true
		} else if sfield.relation == nil && sfield.field.name != "*" && sfield.field.name != "" && sfield.field.name != "," {
			// Skip empty field names and comma separators
			fieldname := _sq(alias, aliasSchema) + "." + quote(sfield.field.name)
			if sfield.field.jsonPath != "" {
				fieldname = "(" + toJson(table, schema, sfield.field.name, fieldname, info) +
					sfield.field.jsonPath + ")"
			}
			// Apply field cast in GROUP BY if present
			if sfield.cast != "" {
				fieldname = fieldname + "::" + sfield.cast
			}
			nonAggregateFields = append(nonAggregateFields, fieldname)
		}
	}

	// Only add GROUP BY if we have both aggregates AND non-aggregate fields
	if hasAggregates && len(nonAggregateFields) > 0 {
		return strings.Join(nonAggregateFields, ", ")
	}

	return ""
}

// orderClause builds the ORDER BY of the top level of a query: the orders
// with no embed path (order=...).
func orderClause(table, schema, label string, level int, orderFields []OrderField,
	selectFields []SelectField, info *SchemaInfo) (string, error) {
	return levelOrderClause(table, schema, label, orderFields, selectFields, BuildStack{info: info, level: level})
}

// levelOrderClause builds the ORDER BY of the level the stack describes: the
// orders whose embed path names it (x.order on the embed x, x.y.order on y
// inside x), matched as the filters are. A self-referencing embed is then
// told from its parent though both read the same table.
func levelOrderClause(table, schema, label string, orderFields []OrderField,
	selectFields []SelectField, stack BuildStack) (string, error) {
	level, info := stack.level, stack.info
	var order string
	for _, o := range orderFields {
		if len(o.field.relPath) != len(stack.relPath) || stack.matchPath(o.field.relPath) < len(o.field.relPath) {
			// skip the orders of other levels
			continue
		}
		if order != "" {
			order += ", "
		}
		var fieldname string
		if o.relation != "" {
			// Related order: order by a column in a to-one related table
			matchedLabel, err := validateRelatedOrder(table, schema, o, selectFields, info)
			if err != nil {
				return "", err
			}
			joinAlias := quote(labelWithNumber(table+"_"+matchedLabel, level+1))
			fieldname = joinAlias + "." + quote(o.field.name)
			if o.field.jsonPath != "" {
				fieldname = "(" + fieldname + o.field.jsonPath + ")"
			}
		} else if label == "" {
			fieldname = stack.rootColumn(table, schema, o.field.name)
			if o.field.jsonPath != "" {
				fieldname = "(" + toJson(table, schema, o.field.name, fieldname, info) +
					o.field.jsonPath + ")"
			}
		} else {
			fieldname = label + "." + quote(o.field.name)
			if o.field.jsonPath != "" {
				fieldname = "(" + toJson(table, schema, o.field.name, fieldname, info) +
					o.field.jsonPath + ")"
			}
		}
		order += fieldname
		if o.descending {
			order += " DESC"
		}
		if o.invertNulls {
			if o.descending {
				order += " NULLS LAST"
			} else {
				order += " NULLS FIRST"
			}
		}
	}
	return order, nil
}

// validateRelatedOrder checks that a related order references an embedded to-one relationship.
// Returns the matched label (used for join alias construction) or an error.
func validateRelatedOrder(table, schema string, o OrderField, selectFields []SelectField, info *SchemaInfo) (string, error) {
	for _, sf := range selectFields {
		if sf.relation == nil {
			continue
		}
		sfLabel := sf.label
		if sfLabel == "" {
			sfLabel = sf.relation.name
		}
		if sfLabel == o.relation || sf.relation.name == o.relation {
			// Found matching embedded resource - check relationship type
			parentTable := sf.relation.parent
			if parentTable == "" {
				parentTable = table
			}
			rel, err := findRelationship(parentTable, sf.relation.name, sf.relation.fk, schema, info)
			if err != nil {
				return "", err
			}
			switch rel.Type {
			case M2O, O2O:
				// OK
			case O2M, M2M:
				return "", &BuildError{"A related order on '" + o.relation + "' is not possible"}
			case Computed:
				if rel.ReturnIsSet {
					return "", &BuildError{"A related order on '" + o.relation + "' is not possible"}
				}
			}
			return sfLabel, nil
		}
	}
	return "", &BuildError{"'" + o.relation + "' is not an embedded resource in this request"}
}

// appendNull appends a NULL operand: a NULL parameter.
func appendNull(where string, b *binder) string {
	return b.bind(where, nil)
}

// appendValue appends an operand: the keyword of IS verbatim (forceParam
// false), anything else as a parameter.
func appendValue(where, value string, b *binder, forceParam bool) string {
	if !forceParam {
		if value == "not_null" {
			return where + "NOT NULL"
		}
		if value == "null" || value == "true" || value == "false" || value == "unknown" {
			return where + value
		}
	}
	return b.bind(where, value)
}

// checkFiltersApplied refuses a filter that no level of the built query took,
// as PostgREST does (PGRST108): its relation path names a relation that is
// not embedded in the request, or does not exist. Without it the filter was
// silently dropped and the request answered as if it were not there.
// Filters on embeds are checked only when the representation was built
// (withEmbeds), since only then the embed levels were visited; root filters
// are skipped when skipRoot, as an insert ignores them. b is the binder of
// the page, which knows what each level took.
// checkWriteFilters checks that a write takes every filter it was given,
// embeds included: a filter that neither the WHERE of the write nor an embed
// of its select takes must not be dropped, or the write reaches rows the
// request excluded (PGRST108 in PostgREST, which validates the select
// whatever the write returns). With a representation, st already built the
// embeds. Without one, they are built here on a state of their own, only to
// see which filters they take, and only when a filter names an embed. The
// write's WHERE takes the root filters; skipRoot (an insert) has none.
func checkWriteFilters(table, schema string, parts *QueryParts, options *QueryOptions, info *SchemaInfo,
	st *buildState, skipRoot bool) error {
	if options.ReturnRepresentation || !hasPathFilter(parts.whereConditionsTree) {
		return checkFiltersApplied(parts.whereConditionsTree, skipRoot, true, &st.page)
	}
	dry := newBuildState(parts, nil)
	if !skipRoot {
		dry.pageWhere(table, schema, "", parts.whereConditionsTree, BuildStack{info: info})
	}
	if _, _, err := returningClause(table, schema, parts, info, dry); err != nil {
		return err
	}
	return checkFiltersApplied(parts.whereConditionsTree, skipRoot, true, &dry.page)
}

// hasPathFilter reports whether a node of the tree names a relation path: an
// embed filter, or a boolean node with its table.
func hasPathFilter(node *WhereConditionNode) bool {
	if node == nil {
		return false
	}
	if len(node.field.relPath) > 0 {
		return true
	}
	for _, n := range node.children {
		if hasPathFilter(n) {
			return true
		}
	}
	return false
}

// errRecursiveWrite refuses the recursive filters (start., recurse.) on a
// write: the write builders do not walk a hierarchy, and dropping the two
// filters would widen the write to every row.
var errRecursiveWrite = &BuildError{"recursive filters are only supported on reads"}

func checkFiltersApplied(node *WhereConditionNode, skipRoot, withEmbeds bool, b *binder) error {
	if node == nil {
		return nil
	}
	onEmbed := len(node.field.relPath) > 0
	container := node.isContainer()
	if (onEmbed && !withEmbeds) || (!onEmbed && skipRoot && !container) {
		return nil
	}
	if onEmbed {
		if fs := b.filter(node); fs.level == 0 && fs.matched < len(node.field.relPath) {
			return &BuildError{"'" + node.field.relPath[fs.matched] + "' is not an embedded resource in this request"}
		}
	}
	if container {
		// its path matched (or it is the root): check the children
		for _, n := range node.children {
			if err := checkFiltersApplied(n, skipRoot, withEmbeds, b); err != nil {
				return err
			}
		}
	}
	return nil
}

// whereClause builds a WHERE condition string starting a root node of a condition tree.
// Only nodes with fields related to passed table or label are processed: the
// binder records that its level took them. Their values are bound by b: the
// first new marker is $(b.base + len(b.values) + 1). A count binder (replay
// set) takes the nodes the page took at the level it replays instead.
func whereClause(table, schema, label string, node *WhereConditionNode, b *binder, stack BuildStack) (where string) {
	if node == nil {
		return
	}
	if node.isContainer() {
		// It is a root or a boolean operator

		var bool_op string
		if node.operator == "" {
			bool_op = " AND "
		} else {
			bool_op = " " + node.operator + " "
		}
		if node.not {
			where += "NOT "
		}
		if node.not || node.operator == "OR" {
			where += "("
		}
		var children_where string
		for _, n := range node.children {
			if b.replay != nil {
				if b.replay.filter(n).level != b.replayLevel {
					continue
				}
			} else {
				// check if the field is part of this where clause, by comparing
				// each item in the field relPath with each item in the stack, table
				// or label or FK column name; the longest prefix any level matches
				// is kept, to name the first unmatched relation (see checkFiltersApplied)
				fs := b.filter(n)
				matched := stack.matchPath(n.field.relPath)
				fs.matched = max(fs.matched, matched)
				if matched < len(n.field.relPath) || len(n.field.relPath) != len(stack.relPath) {
					continue
				}
				if fs.level != 0 {
					// already taken by an earlier embed its path also names, as
					// siblings embedding the same table (a parent and the children
					// of a self-reference): it applies once, without a dangling AND
					continue
				}
			}
			if children_where != "" {
				children_where += bool_op
			}
			children_where += whereClause(table, schema, label, n, b, stack)
		}
		where += children_where
		if node.not || node.operator == "OR" {
			where += ")"
		}
		if node.operator != "" && b.replay == nil {
			// a logic tree is taken as a whole, like a single filter
			b.filter(node).level = b.level
		}
	} else {
		if b.replay == nil {
			// skip nodes already inserted
			fs := b.filter(node)
			if fs.level != 0 {
				return
			}
			fs.level = b.level
		}
		if node.not {
			where += "NOT "
		}
		var fieldname string
		if stack.level == 0 {
			fieldname = stack.rootColumn(table, schema, node.field.name)
		} else {
			fieldname = quote(labelWithNumber(table, stack.level)) + "." + quote(node.field.name)
		}
		if node.field.jsonPath != "" {
			fieldname = "(" + toJson(table, schema, node.field.name, fieldname, stack.info) +
				node.field.jsonPath + ")"
		}
		// Every operand is bound, as in PostgREST (pgFmtFilter): eq.null and
		// eq.true compare against the strings 'null' and 'true', and PostgreSQL
		// casts them to the column type. The IS operator is the exception: its
		// operand is one of the keywords null/not_null/true/false/unknown
		// (enforced by the parser) and `x IS $1` is a syntax error, so the
		// keyword is written verbatim.
		forceParam := node.operator != "IS"
		if node.opModifier != "" {
			// any/all modifier: expand to (field OP v1 OR/AND field OP v2 ...)
			var boolOp string
			if node.opModifier == "any" {
				boolOp = " OR "
			} else {
				boolOp = " AND "
			}
			where += "("
			for i, value := range node.values {
				if i != 0 {
					where += boolOp
				}
				where += fieldname + " " + node.operator + " "
				if len(node.nullAt) != 0 && slices.Contains(node.nullAt, i) {
					// an unquoted null element: a NULL parameter
					where = appendNull(where, b)
					continue
				}
				where = appendValue(where, value, b, forceParam)
			}
			where += ")"
		} else {
			where += fieldname
			if node.operator == "IN" && len(node.values) == 0 {
				where += " = ANY('{}')"
				return where
			}
			if node.operator == "@@" {
				// If the column is not a tsvector, wrap with to_tsvector()
				if stack.info != nil {
					ftablename := _s(table, schema)
					ct := stack.info.GetColumnType(ftablename, node.field.name)
					if ct != nil && ct.Type != "tsvector" {
						lang := "'english'"
						if len(node.opArgs) > 0 {
							lang = quoteLit(node.opArgs[0])
						}
						where = where[:len(where)-len(fieldname)] + "to_tsvector(" + lang + ", " + fieldname + ")"
					}
				}
			}
			where += " " + node.operator + " "
			if node.operator == "IN" {
				where += "("
				for i, value := range node.values {
					if i != 0 {
						where += ", "
					}
					where = appendValue(where, value, b, forceParam)
				}
				where += ")"
			} else if node.operator == "@@" {
				switch node.opSource {
				case "fts":
					where += "to_tsquery("
				case "plfts":
					where += "plainto_tsquery("
				case "phfts":
					where += "phraseto_tsquery("
				case "wfts":
					where += "websearch_to_tsquery("
				}
				for _, arg := range node.opArgs {
					where += quoteLit(arg)
					where += ", "
				}
				where = appendValue(where, node.values[0], b, true)
				where += ")"

			} else {
				where = appendValue(where, node.values[0], b, forceParam)
			}
		}
	}
	return where
}

// returningClause builds the RETURNING clause of a mutation and, when the
// representation needs an outer select (embeds, or an order to apply), the
// select over the _source CTE the callers wrap the mutation in.
func returningClause(table, schema string, parts *QueryParts, info *SchemaInfo, st *buildState) (ret, sel string, err error) {
	ret += " RETURNING "
	// order= applies to the returned representation (PostgREST 13.0.0, #3013):
	// it is built against the _source CTE the outer select reads. limit and
	// offset are ignored on mutations, as in PostgREST since the same release
	// dropped limited updates/deletes: every matching row is written.
	order, err := orderClause(table, schema, quote("_source"), 0, parts.orderFields, parts.selectFields, info)
	if err != nil {
		return "", "", err
	}
	if len(parts.selectFields) == 0 {
		ret += "*"
		if order != "" {
			sel = "SELECT * FROM _source ORDER BY " + order
		}
		return
	}
	var hasResourceEmbed bool
	for _, sfield := range parts.selectFields {
		if sfield.relation != nil {
			hasResourceEmbed = true
			break
		}
	}
	if !hasResourceEmbed && order == "" {
		// The RETURNING clause is the final response: use the formatted
		// fields, with casts and aliases.
		var fields string
		for _, sfield := range parts.selectFields {
			if fields != "" {
				fields += ", "
			}
			fields += prepareField(table, schema, sfield, info)
		}
		ret += fields
		return
	}
	// With embeds (or an order) the RETURNING clause only feeds the _source
	// CTE, which the outer select and its lateral joins read by column name
	// (casts, aliases and json paths are applied there). So it must expose
	// each base column once, raw and deduplicated by name: a formatted or
	// duplicated fk column would make _source references ambiguous (42702).
	sc, joins, keys, _, err := selectClause(table, schema, "", parts, BuildStack{info: info, afterWithClause: true, st: st})
	if err != nil {
		return "", "", err
	}
	var fields string
	var hasStar bool
	var colMap = make(map[string]struct{})
	addColumn := func(name string) {
		if _, exists := colMap[name]; exists {
			return
		}
		colMap[name] = struct{}{}
		if fields != "" {
			fields += ", "
		}
		fields += _sq(table, schema) + "." + quote(name)
	}
	for _, sfield := range parts.selectFields {
		if sfield.relation != nil {
			continue
		}
		if isStar(sfield.field.name) {
			hasStar = true
			break
		}
	}
	if hasStar {
		// a star subsumes every column, including the fk keys for the joins
		fields = _sq(table, schema) + ".*"
	} else {
		for _, sfield := range parts.selectFields {
			if sfield.relation != nil {
				continue
			}
			addColumn(sfield.field.name)
		}
		// add the foreign keys needed by the embed joins
		for _, k := range keys {
			addColumn(k)
		}
		// and the columns of the order, which the outer ORDER BY must see even
		// when they are not selected (a related order reads the join instead)
		for _, o := range parts.orderFields {
			if o.relation == "" && len(o.field.relPath) == 0 {
				addColumn(o.field.name)
			}
		}
	}
	ret += fields
	sel = "SELECT " + sc + " FROM _source"
	if joins != "" {
		sel += " " + joins
	}
	if order != "" {
		sel += " ORDER BY " + order
	}
	return
}

func onConflictClause(table, schema string, fields []string,
	conflictFields []string, options *QueryOptions, info *SchemaInfo) string {

	var cFields []string
	hasConflictFields := len(conflictFields) > 0
	if hasConflictFields {
		// @@ should we check if these fields are UNIQUE fields?
		cFields = conflictFields
	} else {
		pk := info.GetPrimaryKey(_s(table, schema))
		if pk == nil {
			// no pk, we ignore the resolution header
			return ""
		}
		cFields = pk.Columns
	}
	s := " ON CONFLICT ("
	for i, col := range cFields {
		if i != 0 {
			s += ", "
		}
		s += quote(col)
	}
	s += ") "
	if options.IgnoreDuplicates || len(fields) == 0 {
		// with no column to set a merge has nothing to update (PostgREST,
		// QueryBuilder.hs: MergeDuplicates with null iCols is DO NOTHING)
		s += "DO NOTHING"
	} else if options.MergeDuplicates {
		s += "DO UPDATE SET "
		for i, f := range fields {
			if i != 0 {
				s += ", "
			}
			f = quote(f)
			s += f + " = EXCLUDED." + f
		}
	}
	return s
}

// orderedRecordKeys returns record's keys in a deterministic order for use
// when building RPC CALL sites. When a function signature is available and
// has no unnamed parameters, IN/INOUT/VARIADIC argument names are emitted in
// declared order; any extra keys (or the whole record when no signature is
// known) follow in alphabetical order. columnFields, when non-empty,
// restricts which keys are included.
func orderedRecordKeys(record Record, columnFields map[string]struct{}, f *Function) []string {
	included := func(key string) bool {
		// the body of a single unnamed parameter (key "") is no column
		if len(columnFields) == 0 || key == "" {
			return true
		}
		_, ok := columnFields[key]
		return ok
	}

	var keys []string
	seen := map[string]bool{}
	if f != nil && !f.HasUnnamed {
		for _, arg := range f.Arguments {
			// Only IN-like modes are passed as input params.
			// Mode 0 is the default (IN) when proargmodes is NULL.
			if arg.Mode != 0 && arg.Mode != 'i' && arg.Mode != 'b' && arg.Mode != 'v' {
				continue
			}
			if _, ok := record[arg.Name]; !ok {
				continue
			}
			if !included(arg.Name) {
				continue
			}
			keys = append(keys, arg.Name)
			seen[arg.Name] = true
		}
	}
	var extra []string
	for k := range record {
		if seen[k] {
			continue
		}
		if !included(k) {
			continue
		}
		extra = append(extra, k)
	}
	sort.Strings(extra)
	return append(keys, extra...)
}

// sameKeys reports whether two records have the same set of keys.
func sameKeys(a, b Record) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

type CommonBuilder struct{}

// checkColumns refuses a ?columns= naming a column the table does not have,
// as PostgREST does (PGRST204), instead of building a statement PostgreSQL
// refuses in its own words. A table the cache does not know is left to
// PostgreSQL, which answers 42P01 (404) whatever the columns.
func checkColumns(table, schema string, parts *QueryParts, info *SchemaInfo) error {
	if info == nil || len(parts.columnFields) == 0 || info.GetTable(_s(table, schema)) == nil {
		return nil
	}
	cols := lo.Keys(parts.columnFields)
	sort.Strings(cols)
	for _, c := range cols {
		if info.GetColumnType(_s(table, schema), c) == nil {
			return &BuildError{"Could not find the '" + c + "' column of '" + table + "' in the schema cache"}
		}
	}
	return nil
}

// appendJSONValue encodes a record value as the JSON the column's input
// function reads, keeping the meaning pgx gave the Go value when it bound it:
// a nil []byte or an invalid pgtype is null, a []byte is bytea's hex form (not
// json's base64, which bytea would store as text), a driver.Valuer gives its
// value (a pgtype.Interval its text) unless it marshals itself to JSON, and a
// float NaN or infinity, which JSON has no number for, the string the float
// input reads. A json.Marshaler comes first, so that a JSON value type keeps
// its JSON; one that fails (a pgtype.Float8 NaN) falls back to its Valuer.
func appendJSONValue(buf *bytes.Buffer, enc *json.Encoder, v any) error {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
		return nil
	case []byte:
		if x == nil {
			buf.WriteString("null")
		} else {
			buf.WriteString(`"\\x` + hex.EncodeToString(x) + `"`)
		}
		return nil
	case float64:
		if s, ok := nonFiniteFloat(x); ok {
			buf.WriteString(s)
			return nil
		}
	case float32:
		if s, ok := nonFiniteFloat(float64(x)); ok {
			buf.WriteString(s)
			return nil
		}
	case json.Marshaler:
		err := enc.Encode(v)
		if err == nil {
			buf.Truncate(buf.Len() - 1) // the newline Encode ends each value with
			return nil
		}
		valuer, ok := v.(driver.Valuer)
		if !ok {
			return err
		}
		return appendValuer(buf, enc, valuer)
	case driver.Valuer:
		return appendValuer(buf, enc, x)
	}
	if err := enc.Encode(v); err != nil {
		return err
	}
	buf.Truncate(buf.Len() - 1)
	return nil
}

func appendValuer(buf *bytes.Buffer, enc *json.Encoder, v driver.Valuer) error {
	dv, err := callValuer(v)
	if err != nil {
		return err
	}
	return appendJSONValue(buf, enc, dv)
}

var valuerType = reflect.TypeFor[driver.Valuer]()

// callValuer calls v.Value as database/sql and pgx do: a nil pointer whose
// Value its element's value receiver gives is NULL without the call, while a
// pointer receiver is called on a nil pointer too.
func callValuer(v driver.Valuer) (driver.Value, error) {
	if rv := reflect.ValueOf(v); rv.Kind() == reflect.Pointer && rv.IsNil() && rv.Type().Elem().Implements(valuerType) {
		return nil, nil
	}
	return v.Value()
}

func nonFiniteFloat(f float64) (string, bool) {
	switch {
	case math.IsNaN(f):
		return `"NaN"`, true
	case math.IsInf(f, 1):
		return `"Infinity"`, true
	case math.IsInf(f, -1):
		return `"-Infinity"`, true
	}
	return "", false
}

// isJSONNull reports whether body is the JSON null, with JSON's whitespace
// around it and nothing else.
func isJSONNull(body []byte) bool {
	return string(bytes.Trim(body, " \t\r\n")) == "null"
}

// goKind is what a Go value means for its column, as pgx binds it there
// (see appendGoValue). The zero value is a column whose type is not known.
type goKind uint8

const (
	goUnknown   goKind = iota
	goJSON             // json, jsonb
	goJSONArray        // json[], jsonb[]
	goBytea            // bytea
	goOther            // any other known type
)

// goValueKind is the kind of a column of the schema cache: only pg_catalog's
// json types are JSON, a domain counting as its base type.
func goValueKind(ct *ColumnType) goKind {
	if ct.Builtin {
		switch ct.Type {
		case "json", "jsonb":
			return goJSON
		case "_json", "_jsonb":
			return goJSONArray
		case "bytea":
			return goBytea
		}
	}
	return goOther
}

// appendGoValue encodes a Go value with the meaning pgx gives it for the
// column's kind: JSON for a json column (appendRawJSON) and for the elements
// of a json[] column, a []byte as text for a column of another known type;
// everything else as appendJSONValue does.
func appendGoValue(buf *bytes.Buffer, enc *json.Encoder, field string, kind goKind, v any) error {
	switch kind {
	case goJSON:
		return appendRawJSON(buf, enc, field, v)
	case goJSONArray:
		return appendJSONArray(buf, enc, field, v)
	case goOther:
		return appendOtherValue(buf, enc, field, v)
	}
	return appendJSONValue(buf, enc, v)
}

// appendOtherValue encodes a Go value for a column of a known type other than
// json and bytea: a []byte is its text, as pgx's text codec sends it, also a
// named one (a json.RawMessage keeps its quotes). As pgx dereferences and
// plans again at each level, the value is resolved step by step: a
// driver.Valuer gives its value, a pointer or an interface its element, until
// a byte slice, or a value that marshals itself and is encoded as it is, so
// that a pointer keeps its own MarshalJSON.
func appendOtherValue(buf *bytes.Buffer, enc *json.Encoder, field string, v any) error {
	for range 64 { // a guard against a Valuer that keeps answering a Valuer
		if v == nil {
			buf.WriteString("null")
			return nil
		}
		if valuer, ok := v.(driver.Valuer); ok {
			dv, err := callValuer(valuer)
			if err != nil {
				return err
			}
			v = dv
			continue
		}
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
			if rv.IsNil() {
				buf.WriteString("null")
				return nil
			}
			if !utf8.Valid(rv.Bytes()) {
				return &BuildError{"Invalid UTF-8 for the column '" + field + "'"}
			}
			return appendJSONValue(buf, enc, string(rv.Bytes()))
		}
		if rv.Kind() != reflect.Pointer && rv.Kind() != reflect.Interface {
			return appendJSONValue(buf, enc, v)
		}
		if rv.IsNil() {
			buf.WriteString("null")
			return nil
		}
		// a byte slice comes before a MarshalJSON the pointer inherits from
		// it (a *json.RawMessage is its bytes)
		el := rv.Elem()
		if _, ok := v.(json.Marshaler); ok && !(el.Kind() == reflect.Slice && el.Type().Elem().Kind() == reflect.Uint8) {
			return appendJSONValue(buf, enc, v)
		}
		v = el.Interface()
	}
	return &BuildError{"Cannot resolve the value for the column '" + field + "'"}
}

// appendRawJSON encodes a Go value for a json or jsonb column in the order
// of pgx's JSON codec (pgtype/json.go): a string, a []byte or a
// json.RawMessage is the JSON itself and must be valid; a driver.Valuer gives
// its value before a json.Marshaler is asked; a pointer is followed and a
// named string or []byte type is one; anything else is encoded as JSON.
func appendRawJSON(buf *bytes.Buffer, enc *json.Encoder, field string, v any) error {
	var raw []byte
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
		return nil
	case string:
		raw = []byte(x)
	case []byte:
		raw = x
	case json.RawMessage:
		raw = x
	case driver.Valuer:
		dv, err := callValuer(x)
		if err != nil {
			return err
		}
		return appendRawJSON(buf, enc, field, dv)
	case json.Marshaler:
		return appendJSONValue(buf, enc, v)
	default:
		rv := reflect.ValueOf(v)
		switch {
		case rv.Kind() == reflect.Pointer:
			if rv.IsNil() {
				buf.WriteString("null")
				return nil
			}
			return appendRawJSON(buf, enc, field, rv.Elem().Interface())
		case rv.Kind() == reflect.String:
			raw = []byte(rv.String())
		case rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8:
			if rv.IsNil() {
				buf.WriteString("null")
				return nil
			}
			raw = rv.Bytes()
		default:
			return appendJSONValue(buf, enc, v)
		}
	}
	if raw == nil {
		buf.WriteString("null")
		return nil
	}
	if !json.Valid(raw) {
		return &BuildError{"Invalid JSON for the column '" + field + "'"}
	}
	buf.Write(raw)
	return nil
}

// appendJSONArray encodes a Go slice or array for a json[] column as pgx
// does: a one-dimensional array literal whose elements are JSON
// (appendRawJSON), a nested slice included, since the json codec takes any
// value; sent as a string for the array's input function.
// A JSON array cannot carry it: json_to_recordset reads a nested JSON array as
// a further dimension, not as a json element. A driver.Valuer, or any other
// value, goes through appendJSONValue.
func appendJSONArray(buf *bytes.Buffer, enc *json.Encoder, field string, v any) error {
	if v == nil {
		buf.WriteString("null")
		return nil
	}
	if _, ok := v.(driver.Valuer); ok {
		return appendJSONValue(buf, enc, v)
	}
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			buf.WriteString("null")
			return nil
		}
		rv = rv.Elem()
	}
	if !isGoArray(rv) {
		return appendJSONValue(buf, enc, v)
	}
	if rv.Kind() == reflect.Slice && rv.IsNil() {
		buf.WriteString("null")
		return nil
	}
	var lit strings.Builder
	if err := jsonArrayLiteral(&lit, field, rv); err != nil {
		return err
	}
	return appendJSONValue(buf, enc, lit.String())
}

// isGoArray reports whether v is a slice or an array, but not of bytes.
func isGoArray(v reflect.Value) bool {
	return (v.Kind() == reflect.Slice || v.Kind() == reflect.Array) && v.Type().Elem().Kind() != reflect.Uint8
}

// jsonArrayLiteral writes the array literal of a json[] value: each element
// its JSON text quoted, NULL for a nil one or a Valuer giving nil. pgx makes
// it one-dimensional whatever the Go type: its slice wrapper is tried first,
// and the json codec takes an inner slice as a JSON array, ragged or not.
func jsonArrayLiteral(lit *strings.Builder, field string, rv reflect.Value) error {
	lit.WriteByte('{')
	for i := 0; i < rv.Len(); i++ {
		if i > 0 {
			lit.WriteByte(',')
		}
		e := rv.Index(i)
		var v any
		if e.Kind() != reflect.Interface || !e.IsNil() {
			v = e.Interface()
		}
		if valuer, ok := v.(driver.Valuer); ok {
			dv, err := callValuer(valuer)
			if err != nil {
				return err
			}
			v = dv
		}
		if v == nil {
			lit.WriteString("NULL")
			continue
		}
		if rv := reflect.ValueOf(v); (rv.Kind() == reflect.Pointer || rv.Kind() == reflect.Map || rv.Kind() == reflect.Slice) && rv.IsNil() {
			lit.WriteString("NULL")
			continue
		}
		var raw bytes.Buffer
		enc := json.NewEncoder(&raw)
		enc.SetEscapeHTML(false)
		if err := appendRawJSON(&raw, enc, field, v); err != nil {
			return err
		}
		lit.WriteByte('"')
		lit.WriteString(strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(raw.String()))
		lit.WriteByte('"')
	}
	lit.WriteByte('}')
	return nil
}

// insertRows is the JSON array of rows an insert reads: the request body as
// it came, an object wrapped into an array, or else the Go records encoded
// with the insert's columns, each value with the meaning pgx gives it for its
// column (kinds, when the types are known; see appendGoValue).
func insertRows(records []Record, fields []string, kinds []goKind, body []byte) ([]byte, error) {
	if body != nil {
		if trimmed := bytes.TrimLeft(body, " \t\r\n"); len(trimmed) > 0 && trimmed[0] == '{' {
			rows := make([]byte, 0, len(body)+2)
			rows = append(rows, '[')
			rows = append(rows, body...)
			return append(rows, ']'), nil
		}
		return body, nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	keys := make([]string, len(fields))
	for i, f := range fields {
		k, _ := json.Marshal(f)
		keys[i] = string(k) + ":"
	}
	buf.WriteByte('[')
	for i, record := range records {
		if i > 0 {
			buf.WriteByte(',')
		}
		if record == nil {
			// the null element a request body would carry: refused before
			// without ?columns=, left to PostgreSQL with it
			buf.WriteString("null")
			continue
		}
		buf.WriteByte('{')
		first := true
		for k, f := range fields {
			v, ok := record[f]
			if !ok {
				continue
			}
			if !first {
				buf.WriteByte(',')
			}
			first = false
			buf.WriteString(keys[k])
			kind := goUnknown
			if kinds != nil {
				kind = kinds[k]
			}
			if err := appendGoValue(&buf, enc, f, kind, v); err != nil {
				return nil, err
			}
		}
		buf.WriteByte('}')
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

// BuildInsert inserts the rows as PostgREST does (SqlFragment.hs,
// fromJsonBodyF): the rows travel as the statement's only parameter, a json
// array read with json_to_recordset, so no bulk insert runs into the 65 535
// parameters of the extended protocol, and the database's input functions
// convert the values (an array, a composite, a json column keeping its text).
// body is the request's JSON, sent as it is: the JSON the records were decoded
// from, or the JSON a CSV or form body converts to, as in PostgREST. Without one
// (the Go API) the records are Go values, encoded with the meaning pgx gave
// them (see insertRows).
func (CommonBuilder) BuildInsert(table string, records []Record, body []byte, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (
	insert string, valueList []any, err error) {

	schema := options.Schema
	// A body that is neither an array nor an object is no row, as PostgREST
	// reads it (Payload.hs, payloadAttributes); null is the one such body
	// that decodes into records. With ?columns= PostgREST passes the body as
	// it is (RawJSON), and PostgreSQL refuses a null one (22023).
	if body != nil && len(parts.columnFields) == 0 && isJSONNull(body) {
		records, body = nil, []byte("[]")
	}

	// The column list. With ?columns= it is exactly the listed set, for every
	// row: a key absent from an object is inserted as NULL, a key not listed is
	// ignored (json_to_recordset reads only the declared columns). Otherwise it
	// is the key set of the first object, which every other object must share:
	// taking it from the first object alone silently dropped the keys the others
	// added, so a non-uniform array is refused as PostgREST does.
	var fieldList []string
	if len(parts.columnFields) > 0 {
		fieldList = lo.Keys(parts.columnFields)
	} else if len(records) > 0 {
		// a null element is not an object, even beside an empty one
		for i := range records {
			if records[i] == nil || !sameKeys(records[0], records[i]) {
				return "", nil, &BuildError{"All object keys must match"}
			}
		}
		fieldList = lo.Keys(records[0])
	}
	// alphabetical, so the SQL text is stable across runs (see orderedRecordKeys)
	sort.Strings(fieldList)

	// The column definition list of json_to_recordset takes each column's type
	// from the schema cache, as written by format_type (typmod and domain
	// included, as PostgREST's nominal type), and a key that is not a column,
	// from the body or ?columns=, is refused as PostgREST does (resolveOrError,
	// the checkColumns message). Any relation the cache has the columns of is
	// typed, a foreign table too. One it does not know is read with its own
	// row type instead, json_populate_recordset, and left to PostgreSQL, which
	// answers 42P01 when it is missing; that function runs every column the
	// body omits through its input function, so there an omitted column of a
	// NOT NULL domain fails even with a default.
	ftable := _s(table, schema)
	var typed bool
	if info != nil {
		_, typed = info.cachedColumnTypes[ftable]
	}
	var fields, sourceFields, defs string
	var kinds []goKind
	if typed {
		kinds = make([]goKind, len(fieldList))
	}
	for i, f := range fieldList {
		if i > 0 {
			fields += ", "
			sourceFields += ", "
			defs += ", "
		}
		fields += quote(f)
		sourceFields += "_." + quote(f)
		if typed {
			ct := info.GetColumnType(ftable, f)
			if ct == nil {
				return "", nil, &BuildError{"Could not find the '" + f + "' column of '" + table + "' in the schema cache"}
			}
			defs += quote(f) + " " + ct.DataType
			kinds[i] = goValueKind(ct)
		}
	}
	rows, err := insertRows(records, fieldList, kinds, body)
	if err != nil {
		return "", nil, err
	}
	st := newBuildState(parts, []any{rows})

	insert = "INSERT INTO " + _sq(table, schema)
	switch {
	case len(fieldList) == 0:
		// no columns: one row of defaults per object, as PostgREST does
		// ([{}, {}] inserts two rows, {} one)
		insert += " SELECT FROM json_array_elements($1::json) AS _"
	case typed:
		insert += " (" + fields + ") SELECT " + sourceFields + " FROM json_to_recordset($1::json) AS _(" + defs + ")"
	default:
		insert += " (" + fields + ") SELECT " + sourceFields + " FROM json_populate_recordset(NULL::" + _sq(table, schema) + ", $1::json) AS _"
	}
	// ON CONFLICT only for a resolution, as PostgREST (Plan.hs, mutatePlan):
	// on_conflict alone is a plain insert, and a conflict its 409
	if options.MergeDuplicates || options.IgnoreDuplicates {
		conflictFields := lo.Keys(parts.conflictFields)
		sort.Strings(conflictFields) // one SQL text, see BuildUpdate
		onConflict := onConflictClause(table, schema, fieldList, conflictFields, options, info)
		insert += onConflict
	}
	if options.ReturnRepresentation {
		ret, sel, err := returningClause(table, schema, parts, info, st)
		if err != nil {
			return "", nil, err
		}
		insert += ret
		if sel != "" {
			insert = "WITH _source AS (" + insert + ") " + sel
		}
	}
	if err := checkWriteFilters(table, schema, parts, options, info, st, true); err != nil {
		return "", nil, err
	}
	return insert, st.page.values, nil
}

func (CommonBuilder) BuildUpdate(table string, record Record, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (
	update string, valueList []any, err error) {
	if parts.recursive != nil {
		return "", nil, errRecursiveWrite
	}

	stack := BuildStack{info: info}
	schema := options.Schema
	if err := checkColumns(table, schema, parts, info); err != nil {
		return "", nil, err
	}
	var pairs string
	var i int
	// in a fixed order, as INSERT: one SQL text per set of columns, so one
	// prepared statement in pgx's cache (the filters' markers follow)
	for _, key := range orderedRecordKeys(record, parts.columnFields, nil) {
		if pairs != "" {
			pairs += ", "
		}
		pairs += quote(key)
		i++
		pairs += " = $" + strconv.Itoa(i)
		valueList = append(valueList, record[key])
	}
	if pairs == "" {
		// ?columns= left nothing of the body to set: an UPDATE with an empty
		// SET is a syntax error, so PostgREST selects nothing from the table
		// instead (mutatePlanToQuery, null uCols) and the table name still
		// reaches PostgreSQL, so a missing one answers 42P01.
		return "SELECT * FROM " + _sq(table, schema) + " WHERE false", nil, nil
	}
	st := newBuildState(parts, valueList)
	whereClause, _ := st.pageWhere(table, schema, "", parts.whereConditionsTree, stack)
	update = "UPDATE " + _sq(table, schema) + " SET " + pairs
	if whereClause != "" {
		update += " WHERE " + whereClause
	}
	if options.ReturnRepresentation {
		ret, sel, err := returningClause(table, schema, parts, info, st)
		if err != nil {
			return "", nil, err
		}
		update += ret
		if sel != "" {
			update = "WITH _source AS (" + update + ") " + sel
		}
	}
	if err := checkWriteFilters(table, schema, parts, options, info, st, false); err != nil {
		return "", nil, err
	}
	return update, st.page.values, nil
}

func (CommonBuilder) BuildDelete(table string, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (
	delete string, valueList []any, err error) {
	if parts.recursive != nil {
		return "", nil, errRecursiveWrite
	}

	stack := BuildStack{info: info}
	schema := options.Schema
	st := newBuildState(parts, nil)
	whereClause, _ := st.pageWhere(table, schema, "", parts.whereConditionsTree, stack)
	delete = "DELETE FROM " + _sq(table, schema)
	if whereClause != "" {
		delete += " WHERE " + whereClause
	}
	if options.ReturnRepresentation {
		ret, sel, err := returningClause(table, schema, parts, info, st)
		if err != nil {
			return "", nil, err
		}
		delete += ret
		if sel != "" {
			delete = "WITH _source AS (" + delete + ") " + sel
		}
	}
	if err := checkWriteFilters(table, schema, parts, options, info, st, false); err != nil {
		return "", nil, err
	}
	return delete, st.page.values, nil
}

// totalSetting is the transaction-local setting through which the page of a
// ranged count on a function hands its total to the statement that follows
// it (see buildAfterSelect)
const totalSetting = "smoothdb.total"

// buildAfterSelect completes a read: the source (from), its joins, the WHERE,
// GROUP BY, ORDER BY, LIMIT and OFFSET. once says that the source is a call
// to a function that must run once, FROM fn(...) t: a VOLATILE function, or
// one missing from the cache.
//
// The total of a ranged count (count=exact with a limit, an offset or a Range
// header) is PostgREST's readPlanToCountQuery: the root rows that pass the
// filters and have a row in each !inner embed. The left embeds do not
// restrict them, and the GROUP BY is left out: a grouped aggregate totals the
// rows it groups. Without a range the total is the number of rows read, as
// PostgREST's page count.
//
// The total travels outside the rows, in a second statement that the
// executor sends in the same batch (options.totalQuery): the page is the
// query a request without a count runs, so its rows and their order are the
// same, and an empty page is empty. On a table the second statement counts
// the table again, as PostgREST's count CTE does, and so on a STABLE or
// IMMUTABLE function, which PostgreSQL may inline into both statements. A
// VOLATILE function must run once (PostgREST reads both from its
// pgrst_source CTE): the page reads it through a materialized CTE and stores
// the count of that CTE in totalSetting, from a scalar subquery that
// PostgreSQL evaluates once, as a one-time filter, before the first row; the
// second statement reads the setting back. With limit=0 PostgreSQL reads
// nothing for the page, and the second statement counts the function itself.
//
// innerConds is an EXISTS per !inner embed of the root (see selectClause);
// count binds the values of their filters when the total is a statement of
// its own (countApart), after the values that the page binds before its
// embeds (the arguments of a function, the root filters).
func buildAfterSelect(query, from string, once bool, joins, whereClause, groupByClause, orderClause string, valueList []any, innerConds []string, count *binder, parts *QueryParts, options *QueryOptions) (string, []any, error) {
	options.totalQuery, options.totalValues, options.totalPrelude = "", nil, ""
	limit, hasLimit, offset, hasOffset := pageBounds(parts, options)
	if !options.HasRange {
		if parts.limit != "" {
			l, _ := strconv.ParseInt(parts.limit, 10, 64)
			options.RangeMax = l - 1
		}
		if hasOffset {
			options.RangeMin = offset
			if options.RangeMax == -1 {
				options.RangeMax = 0
			}
			options.RangeMax += options.RangeMin
		}
	}

	if countModeOf(options, once, limit, offset) != countNone {
		conds := innerConds
		if whereClause != "" {
			conds = append([]string{whereClause}, conds...)
		}
		var countWhere string
		if len(conds) > 0 {
			countWhere = " WHERE " + strings.Join(conds, " AND ")
		}
		if !once || limit == 0 {
			options.totalQuery = "SELECT COUNT(*) " + strings.TrimSpace(from) + countWhere
			options.totalValues = valueList
			if count != nil {
				options.totalValues = valueList[:count.base:count.base]
				if len(count.values) != 0 {
					options.totalValues = slices.Concat(options.totalValues, count.values)
				}
			}
		} else {
			query = "WITH __source AS MATERIALIZED (SELECT * " + strings.TrimSpace(from) + ") " + query
			from = "FROM __source t"
			gate := "(SELECT set_config('" + totalSetting + "', COUNT(*)::text, true) FROM __source t" + countWhere + ") IS NOT NULL"
			if whereClause != "" {
				whereClause = gate + " AND (" + whereClause + ")"
			} else {
				whereClause = gate
			}
			// a filter that PostgreSQL folds to a constant false drops the
			// one-time filter with it, and the total stays unset (or '', the
			// value of a setting defined earlier in the session): the count
			// over a false filter is 0
			options.totalQuery = "SELECT coalesce(nullif(current_setting('" + totalSetting + "', true), ''), '0')::bigint"
			// the setting lasts as long as the transaction: cleared before
			// the page, a count of an earlier call in it is not read back
			options.totalPrelude = "SELECT set_config('" + totalSetting + "', '0', true)"
		}
	}

	nmarker := len(valueList)
	query += " " + from
	if joins != "" {
		query += " " + joins
	}
	if whereClause != "" {
		query += " WHERE " + whereClause
	}
	if groupByClause != "" {
		query += " GROUP BY " + groupByClause
	}
	if orderClause != "" {
		query += " ORDER BY " + orderClause
	}
	if hasLimit {
		nmarker += 1
		query += " LIMIT $" + strconv.Itoa(nmarker)
		valueList = append(valueList, limit)
	}
	if hasOffset {
		nmarker += 1
		query += " OFFSET $" + strconv.Itoa(nmarker)
		valueList = append(valueList, offset)
	}
	return query, valueList, nil
}

// pageBounds returns the LIMIT and the OFFSET of a read: from a Range or
// from limit= and offset=, the limit capped by maxRows. -1 is none.
func pageBounds(parts *QueryParts, options *QueryOptions) (limit int64, hasLimit bool, offset int64, hasOffset bool) {
	limit = -1
	hasLimit = parts.limit != "" || options.HasRange && options.RangeMax != -1
	if hasLimit {
		if options.HasRange {
			limit = options.RangeMax - options.RangeMin + 1
		} else {
			limit, _ = strconv.ParseInt(parts.limit, 10, 64)
		}
	}
	if capRows := maxRows(); capRows > 0 && (limit == -1 || limit > capRows) {
		limit, hasLimit = capRows, true
	}
	offset = -1
	hasOffset = parts.offset != "" || options.HasRange
	if hasOffset {
		if options.HasRange {
			offset = options.RangeMin
		} else {
			offset, _ = strconv.ParseInt(parts.offset, 10, 64)
		}
	}
	return
}

// countModeOf tells how the total of a ranged count is built (see
// buildAfterSelect): a count with a limit or an offset has one, counted in a
// statement of its own, or in the page for a function that must run once.
func countModeOf(options *QueryOptions, once bool, limit, offset int64) countMode {
	switch {
	case options.Count == "" || limit == -1 && offset <= 0:
		return countNone
	case !once || limit == 0:
		return countApart
	}
	return countInPage
}

// startCount prepares the build of the total of a ranged count, after the
// page has bound the values that the total shares: the arguments of a
// function and the root filters.
func (st *buildState) startCount(parts *QueryParts, options *QueryOptions, once bool) {
	limit, _, offset, _ := pageBounds(parts, options)
	st.countMode = countModeOf(options, once, limit, offset)
	if st.countMode == countApart {
		st.count = &binder{base: len(st.page.values)}
	}
}

const defaultMaxRecursiveDepth = 100

// referencesField reports whether a where-condition tree filters on the named
// field, at any depth of its and/or/not nesting.
func referencesField(node *WhereConditionNode, name string) bool {
	if node == nil {
		return false
	}
	if node.field.name == name {
		return true
	}
	for _, child := range node.children {
		if referencesField(child, name) {
			return true
		}
	}
	return false
}

// recursiveAliases sets the names through which the outer query of a
// recursive select reads the root columns (see buildRecursiveSelect).
// Single-table mode selects from the CTE, which carries the rows, so every
// root column is read from "__recursive". Via mode joins the table back
// onto "__dedup", so its columns are read from the table itself and only the
// pseudo-columns __depth and __path from "__dedup". The select list, the
// filters, the order and the embeds are built with these names in the first
// place: values and identifiers are never rewritten in the finished SQL.
func recursiveAliases(stack *BuildStack, rec *RecursiveInfo) {
	if rec.ViaTable == "" {
		stack.rootAlias = "__recursive"
	} else {
		stack.pseudoAlias = "__dedup"
	}
}

func buildRecursiveSelect(table, schema string, parts *QueryParts, options *QueryOptions,
	selectClause, mainWhere, orderClause, joins string,
	st *buildState, info *SchemaInfo) (string, []any, error) {

	rec := parts.recursive
	qtable := _sq(table, schema)
	startField := quote(rec.StartField)
	recurseField := quote(rec.RecurseField)
	cteName := quote("__recursive")
	b := &st.page

	// Computed-relationship embeds call a function with the parent ROW. The base
	// table is not in scope in the recursive outer query (rows live in the CTE), so
	// when such an embed is present the CTE carries the source row as a composite
	// `__row` column and the embed reads it (see selectForJoinClause). `via`+embed is
	// rejected below, so this only applies single-table.
	rowCol := ""
	if st.rootRow {
		rowCol = ", " + quote(table) + " AS " + quote("__row")
	}

	// Determine effective max depth.
	// MaxRecursiveDepth == 0 means recursive queries are disabled.
	if dbe != nil && dbe.config.MaxRecursiveDepth == 0 {
		return "", nil, &ParseError{"recursive queries are disabled"}
	}

	// Embedding (LEFT JOIN LATERAL) composes with single-table recursion only.
	// Via-mode embedding would fight the min-depth dedup, so reject it cleanly.
	if joins != "" && rec.ViaTable != "" {
		return "", nil, &ParseError{"embedding is not supported with via() recursion"}
	}

	// __depth and __path are selectable (and orderable) pseudo-columns. Single-table
	// mode always carries the path array — one path per node on an FK tree, it costs
	// nothing. Via mode carries it only when __path is asked for: it is what makes the
	// CTE enumerate paths instead of nodes (see the via branch below).
	withPath := false
	for _, sf := range parts.selectFields {
		if sf.field.name == "__path" {
			withPath = true
		}
	}
	for _, of := range parts.orderFields {
		if of.field.name == "__path" {
			withPath = true
		}
	}
	// a result filter on __path (`__path=cs.{1,2}`) reads it from the CTE too
	if referencesField(parts.whereConditionsTree, "__path") {
		withPath = true
	}
	maxDepth := rec.MaxDepth
	serverMax := defaultMaxRecursiveDepth
	if dbe != nil && dbe.config.MaxRecursiveDepth > 0 {
		serverMax = dbe.config.MaxRecursiveDepth
	}
	if maxDepth < 0 || maxDepth > serverMax {
		maxDepth = serverMax
	}

	// Parameter marker allocation order, after the result filters and the
	// embeds:
	// 1. Via edge conditions (if any)
	// 2. Walk conditions (if any)
	// 3. Start value
	// 4. Max depth
	// The binder assigns markers in the order they are built.

	// Build via edge table WHERE clause using the standard whereClause builder
	var viaWhere string
	if rec.ViaConditions != nil {
		viaWhere, _ = st.pageWhere(rec.ViaTable, schema, "", rec.ViaConditions, BuildStack{})
	}

	// Build the walk-prune WHERE (walk.* filters) against the main table.
	// Marker order stays via -> walk -> start -> depth.
	var walkWhere string
	if rec.WalkConditions != nil {
		walkWhere, _ = st.pageWhere(table, schema, "", rec.WalkConditions, BuildStack{info: info})
	}

	var q strings.Builder

	// --- CTE ---
	q.WriteString("WITH RECURSIVE " + cteName + " AS (")

	if rec.ViaTable == "" {
		// --- Single-table mode ---

		// Base case — when ExcludeStart is true (after operator), skip walk filters on
		// the seed row so it remains a traversal anchor even if it doesn't match them.
		q.WriteString("SELECT " + qtable + ".*" + rowCol + ", 0 AS __depth, ARRAY[" + qtable + "." + startField + "] AS __path")
		q.WriteString(" FROM " + qtable)
		q.WriteString(b.bind(" WHERE "+qtable+"."+startField+" = ", rec.StartValue))
		if walkWhere != "" && !rec.ExcludeStart {
			q.WriteString(" AND " + walkWhere)
		}

		q.WriteString(" UNION ALL ")

		// Recursive step. Default (down): join new.recurseField = prev.startField - follow the
		// FK backwards to descendants (rows whose FK points at a known node). recurse!up reverses
		// the same two fields to new.startField = prev.recurseField — follow the FK forwards to
		// ancestors (the row the known node's FK points at). Base case and the __path cycle guard
		// (keyed on startField) are identical for both directions.
		q.WriteString("SELECT " + qtable + ".*" + rowCol + ", " + cteName + ".__depth + 1, " + cteName + ".__path || " + qtable + "." + startField)
		q.WriteString(" FROM " + qtable)
		// down: new.recurseField = prev.startField; up swaps the two fields.
		newField, prevField := recurseField, startField
		if rec.RecurseUp {
			newField, prevField = startField, recurseField
		}
		q.WriteString(" INNER JOIN " + cteName + " ON " + qtable + "." + newField + " = " + cteName + "." + prevField)
		q.WriteString(b.bind(" WHERE "+cteName+".__depth < ", maxDepth))
		q.WriteString(" AND NOT " + qtable + "." + startField + " = ANY(" + cteName + ".__path)")
		if walkWhere != "" {
			q.WriteString(" AND " + walkWhere)
		}
	} else {
		// --- Multi-table (via) mode ---
		// doc?id=start.1&id=recurse.all&doc_rel=via(src_id,dst_id)
		// Base case is the start node itself (depth 0), same as single-table.
		// The edge table join happens in the recursive step.
		//
		// The CTE carries only (__node, __depth): a whole-row CTE with a per-path cycle
		// guard (NOT key = ANY(__path)) enumerates every simple path of the graph, which
		// is exponential in depth — a 650-node DAG of depth 12 materialised 797,161 rows
		// for 490 nodes. With UNION on the narrow row the set dedup bounds the work to one
		// row per (node, depth): a node reached again through a cycle only re-enters at a
		// larger depth, so a cyclic graph is walked up to the depth cap, at most one row
		// per node per level. The seed is never re-entered — a walk back into it only
		// repeats a shorter walk — which is also what lets `after` drop it as the only
		// node at depth 0. The set of nodes and the shallowest depth per node are the same
		// as with the per-path guard: every walk to a node contains a simple path to it of
		// no greater length. The table is joined back in the outer query.
		//
		// When __path is asked for the array is carried again (UNION ALL, per-path guard):
		// it is the only way to have a path, and it costs the enumeration of every simple
		// path — the README documents it as exponential.
		qvia := _sq(rec.ViaTable, schema)
		viaFrom := quote(rec.ViaFromCol)
		viaTo := quote(rec.ViaToCol)
		edgeName := quote("__edge")

		// Base case: the start node itself — when ExcludeStart is true (after operator),
		// skip walk filters so the seed remains a traversal anchor.
		q.WriteString("SELECT " + qtable + "." + startField + " AS __node, 0 AS __depth")
		if withPath {
			q.WriteString(", ARRAY[" + qtable + "." + startField + "] AS __path")
		}
		q.WriteString(" FROM " + qtable)
		startMarker := b.bind("", rec.StartValue)
		q.WriteString(" WHERE " + qtable + "." + startField + " = " + startMarker)
		if walkWhere != "" && !rec.ExcludeStart {
			q.WriteString(" AND " + walkWhere)
		}

		if withPath {
			q.WriteString(" UNION ALL ")
		} else {
			q.WriteString(" UNION ")
		}

		// Recursive step: follow edges from previously found nodes. The edge table is
		// wrapped in a derived table of (__from, __to) pairs — for via!both both
		// orientations, UNION ALL — so the planner drives each arm with an index on the
		// known node rather than the OR join it could not index (which scanned every edge
		// per working-table row). Edge filters go inside the arms, where the edge table
		// is in scope; their markers are simply reused by the second arm.
		edges := "SELECT " + qvia + "." + viaFrom + " AS __from, " + qvia + "." + viaTo + " AS __to FROM " + qvia
		if viaWhere != "" {
			edges += " WHERE " + viaWhere
		}
		if rec.ViaBidirectional {
			edges += " UNION ALL SELECT " + qvia + "." + viaTo + ", " + qvia + "." + viaFrom + " FROM " + qvia
			if viaWhere != "" {
				edges += " WHERE " + viaWhere
			}
		}
		q.WriteString("SELECT " + qtable + "." + startField + ", " + cteName + ".__depth + 1")
		if withPath {
			q.WriteString(", " + cteName + ".__path || " + qtable + "." + startField)
		}
		q.WriteString(" FROM " + cteName)
		q.WriteString(" INNER JOIN (" + edges + ") " + edgeName + " ON " + edgeName + ".__from = " + cteName + ".__node")
		q.WriteString(" INNER JOIN " + qtable + " ON " + qtable + "." + startField + " = " + edgeName + ".__to")
		q.WriteString(b.bind(" WHERE "+cteName+".__depth < ", maxDepth))
		if withPath {
			q.WriteString(" AND NOT " + qtable + "." + startField + " = ANY(" + cteName + ".__path)")
		} else {
			q.WriteString(" AND " + qtable + "." + startField + " <> " + startMarker)
		}
		if walkWhere != "" {
			q.WriteString(" AND " + walkWhere)
		}
	}

	q.WriteString(")")

	// via node dedup: one row per node reachable by multiple paths (or at several
	// depths), at its shallowest depth — with __path, the shortest path, ties broken by
	// the path itself. Keying on the node needs an equality operator on the start field
	// only; DISTINCT over the whole row would fail on json, xml or point columns, which
	// have none. Single-table trees skip this — they can't reach a node at two depths
	// (the __path guard keeps edges unique), so a plain SELECT below is fine.
	dedupName := quote("__dedup")
	if rec.ViaTable != "" {
		q.WriteString(", " + dedupName + " AS (SELECT DISTINCT ON (__node) __node, __depth")
		if withPath {
			q.WriteString(", __path")
		}
		q.WriteString(" FROM " + cteName + " ORDER BY __node, __depth")
		if withPath {
			q.WriteString(", __path")
		}
		q.WriteString(")")
	}
	q.WriteString(" ")

	// --- Outer query ---
	// The select list, the result filters, the order and the embeds read the
	// root columns through the names of recursiveAliases.

	// Build the SELECT list (without keyword).
	var sel string
	if selectClause != "*" {
		sel = selectClause
	} else if rec.ViaTable != "" {
		sel = qtable + ".*"
	} else {
		// Enumerate actual table columns to exclude internal __depth/__path
		ftable := _s(table, schema)
		if info != nil {
			if colTypes, ok := info.cachedColumnTypes[ftable]; ok {
				cols := make([]string, 0, len(colTypes))
				for colName := range colTypes {
					cols = append(cols, cteName+"."+quote(colName))
				}
				sort.Strings(cols)
				sel = strings.Join(cols, ", ")
			} else {
				sel = cteName + ".*"
			}
		} else {
			sel = cteName + ".*"
		}
	}

	// Embed joins (LEFT JOIN LATERAL) correlate to "__recursive". via+embed is
	// rejected by the guard above, so joinsClause is only ever set in single-table mode.
	var joinsClause string
	if joins != "" {
		joinsClause = " " + joins
	}

	// Plain filters become a result WHERE on the outer query (composed with the
	// ExcludeStart seed-skip); walk.* filters already pruned the CTE arms above.
	var outerWhere string
	if rec.ExcludeStart {
		if rec.ViaTable != "" {
			outerWhere = dedupName + ".__depth > 0"
		} else {
			outerWhere = "__depth > 0"
		}
	}
	if mainWhere != "" {
		if outerWhere != "" {
			outerWhere += " AND "
		}
		outerWhere += mainWhere
	}

	if rec.ViaTable != "" {
		// The table joined back onto the deduplicated nodes. A plain SELECT, so a user
		// ORDER BY can name a column that is not selected, __depth included.
		q.WriteString("SELECT " + sel + " FROM " + dedupName + " INNER JOIN " + qtable + " ON " + qtable + "." + startField + " = " + dedupName + ".__node")
	} else {
		q.WriteString("SELECT " + sel + " FROM " + cteName + joinsClause)
	}
	if outerWhere != "" {
		q.WriteString(" WHERE " + outerWhere)
	}

	if orderClause != "" {
		q.WriteString(" ORDER BY " + orderClause)
	}

	// Limit
	var limit int64 = -1
	hasLimit := parts.limit != "" || options.HasRange && options.RangeMax != -1
	if hasLimit {
		if options.HasRange {
			limit = options.RangeMax - options.RangeMin + 1
		} else {
			limit, _ = strconv.ParseInt(parts.limit, 10, 64)
			options.RangeMax = limit - 1
		}
	}
	if capRows := maxRows(); capRows > 0 && (limit == -1 || limit > capRows) {
		limit, hasLimit = capRows, true
	}
	if hasLimit {
		q.WriteString(b.bind(" LIMIT ", limit))
	}

	// Offset
	if parts.offset != "" || options.HasRange {
		var offset int64
		if options.HasRange {
			offset = options.RangeMin
		} else {
			offset, _ = strconv.ParseInt(parts.offset, 10, 64)
			options.RangeMin = offset
			if options.RangeMax == -1 {
				options.RangeMax = 0
			}
			options.RangeMax += options.RangeMin
		}
		q.WriteString(b.bind(" OFFSET ", offset))
	}

	return q.String(), b.values, nil
}

// OIDs of the json types
const (
	jsonOID  = 114
	jsonbOID = 3802
)

// argumentValue returns the value that the argument key of a call to f
// (nil when unknown) binds, whether the argument is VARIADIC, and the
// cast of its placeholder, as PostgREST's callPlanToQuery:
//   - QueryValues fill a VARIADIC parameter as an array; any other
//     parameter takes the last of them.
//   - A JSON string for a json or jsonb parameter stays a JSON string. The
//     driver sends a Go string to a json parameter as JSON text.
//   - The body for a single unnamed parameter (a *RawBody) takes the type
//     of the parameter, so that PostgreSQL can choose the overload. With
//     no function found, it takes the type of the body: the call fails
//     there (42883, 404), as in PostgREST, and does not reach a function
//     of another type.
func argumentValue(f *Function, key string, value any) (any, bool, string) {
	if raw, ok := value.(*RawBody); ok {
		typ := raw.Type
		if f != nil {
			if u := f.singleUnnamed(); u != nil {
				typ = u.Type
			}
		}
		switch typ {
		case "text", "xml":
			return string(raw.Data), false, "::" + typ
		}
		return raw.Data, false, "::" + typ
	}
	var arg *Argument
	if f != nil {
		for i := range f.Arguments {
			a := &f.Arguments[i]
			if a.Name == key && a.TypeId != 0 && a.Mode != 'o' && a.Mode != 't' {
				arg = a
				break
			}
		}
	}
	if arg == nil {
		if vv, ok := value.(QueryValues); ok {
			return vv[len(vv)-1], false, ""
		}
		return value, false, ""
	}
	variadic := arg.Mode == 'v'
	switch v := value.(type) {
	case QueryValues:
		if variadic {
			return pgArrayLiteral(v), true, ""
		}
		return v[len(v)-1], false, ""
	case string:
		if arg.TypeId == jsonOID || arg.TypeId == jsonbOID {
			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			enc.SetEscapeHTML(false)
			if enc.Encode(v) == nil {
				return bytes.TrimRight(buf.Bytes(), "\n"), variadic, ""
			}
		}
	}
	return value, variadic, ""
}

// pgArrayLiteral writes values as a PostgreSQL array literal, each element
// quoted, as PostgREST's pgBuildArrayLiteral. PostgreSQL reads it as an
// array of the type of the parameter.
func pgArrayLiteral(values []string) string {
	var b strings.Builder
	b.WriteByte('{')
	for i, v := range values {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('"')
		for _, c := range strings.ReplaceAll(v, "\x00", "") {
			if c == '"' || c == '\\' {
				b.WriteByte('\\')
			}
			b.WriteRune(c)
		}
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String()
}

func (CommonBuilder) BuildExecute(name string, f *Function, record Record, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (
	query string, valueList []any, err error) {
	if parts.recursive != nil {
		// a call does not walk a hierarchy: dropping the recursive filters
		// would return every row, and run a VOLATILE function anyway
		return "", nil, &BuildError{"recursive filters are not supported on a function call"}
	}

	stack := BuildStack{info: info, maxRows: maxRows()}
	schema := options.Schema

	// Determine a deterministic key order so identical RPC calls generate
	// identical SQL (pg_stat_statements hashes by normalized query). Prefer
	// the declared function-signature order; fall back to alphabetical.
	keys := orderedRecordKeys(record, parts.columnFields, f)

	var pairs string
	var i int
	for _, key := range keys {
		if pairs != "" {
			pairs += ", "
		}
		i++
		value, variadic, cast := argumentValue(f, key, record[key])
		if variadic {
			pairs += "VARIADIC "
		}
		if key != "" {
			pairs += quote(key)
			pairs += " := "
		}
		pairs += "$" + strconv.Itoa(i) + cast
		valueList = append(valueList, value)
	}

	// Extract the return type and discover if it is a table.
	// In that case, we will use its name and schema to compose the select clause
	var t, s string
	from := "FROM " + _sq(name, schema) + "(" + pairs + ") t "
	isRecord := f != nil && f.returnsRecord()
	if isRecord {
		// A record has no column list to select from (42601). As PostgREST
		// (callPlanToQuery), call the function in a select list: each row is
		// one value, the column pgrst_scalar.
		from = "FROM (SELECT " + _sq(name, schema) + "(" + pairs + ") \"pgrst_scalar\") t "
	} else if f != nil {
		rettype := info.GetTypeById(f.ReturnTypeId)
		if rettype == nil {
			// the cache knows the function but not its return type: a schema
			// cache that changed under the request
			return "", nil, errors.New("the return type of function " + strconv.Quote(name) +
				" is not in the schema cache; reload the schema cache")
		}
		if rettype.IsTable {
			if rettype.IsDomain {
				t = rettype.DomainSubType
			} else {
				t = rettype.Name
			}
			s = rettype.Schema
		}
	}
	// the JSON paths on the array and composite output columns
	info = info.withCallColumns(f, t, s)
	stack.info = info
	// the root filters bind their values before the embeds (see buildAfterSelect)
	st := newBuildState(parts, valueList)
	stack.st = st
	whereClause, _ := st.pageWhere("t", "", name, parts.whereConditionsTree, stack)
	once := f == nil || f.Volatility == "v"
	st.startCount(parts, options, once)
	stack.countChain = st.countMode != countNone
	selectClause, joins, _, innerConds, err := selectClause(t, s, "t", parts, stack)
	if err != nil {
		return "", nil, err
	}
	// When SELECT is *, check for composite/table-typed output columns
	// and wrap them with to_json() so they serialize as nested JSON objects
	if selectClause == "*" && f != nil && f.HasOut {
		var cols []string
		hasComposite := false
		for _, arg := range f.Arguments {
			if arg.Mode != 't' && arg.Mode != 'o' && arg.Mode != 'b' {
				continue
			}
			st := info.GetTypeById(arg.TypeId)
			if st != nil && (st.IsComposite || st.IsTable) {
				cols = append(cols, "to_json(\"t\"."+quote(arg.Name)+") AS "+quote(arg.Name))
				hasComposite = true
			} else {
				cols = append(cols, "\"t\"."+quote(arg.Name))
			}
		}
		if hasComposite {
			selectClause = strings.Join(cols, ", ")
		}
	}
	if isRecord && (selectClause == "*" || selectClause == "\"t\".\"pgrst_scalar\"") {
		// The fields of a record have no names on the wire: JSON takes
		// them from to_json, as PostgREST's json_agg does. CSV writes the
		// record as PostgreSQL's text, as PostgREST does.
		if options.ContentType == "text/csv" {
			selectClause = "\"t\".\"pgrst_scalar\""
		} else {
			selectClause = "to_json(\"t\".\"pgrst_scalar\") AS \"pgrst_scalar\""
		}
	}
	if err := checkFiltersApplied(parts.whereConditionsTree, false, true, &st.page); err != nil {
		return "", nil, err
	}
	orderClause, err := orderClause("t", "", "", 0, parts.orderFields, parts.selectFields, info)
	if err != nil {
		return "", nil, err
	}
	// an aggregate beside plain columns groups by them, as on a table
	groupByClause := groupByClauseAs("t", "", t, s, parts, info)
	query = "SELECT " + selectClause

	return buildAfterSelect(query, from, once, joins, whereClause, groupByClause, orderClause, st.page.values, innerConds, st.count, parts, options)
}

type DirectQueryBuilder struct {
	CommonBuilder
}

func (DirectQueryBuilder) BuildSelect(table string, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (string, []any, error) {
	st := newBuildState(parts, nil)
	stack := BuildStack{info: info, maxRows: maxRows(), st: st}
	schema := options.Schema
	if parts.recursive != nil {
		recursiveAliases(&stack, parts.recursive)
	}
	// the root filters bind their values before the embeds, so that the total
	// of a ranged count shares them (see buildAfterSelect)
	whereClause, _ := st.pageWhere(table, schema, "", parts.whereConditionsTree, stack)
	if parts.recursive == nil {
		st.startCount(parts, options, false)
		stack.countChain = st.countMode != countNone
	}
	selectClause, joins, _, innerConds, err := selectClause(table, schema, "", parts, stack)
	if err != nil {
		return "", nil, err
	}
	if err := checkFiltersApplied(parts.whereConditionsTree, false, true, &st.page); err != nil {
		return "", nil, err
	}
	groupByClause := groupByClause(table, schema, parts, info)
	orderClause, err := levelOrderClause(table, schema, "", parts.orderFields, parts.selectFields, stack)
	if err != nil {
		return "", nil, err
	}

	if parts.recursive != nil {
		if groupByClause != "" {
			return "", nil, &ParseError{"aggregate functions cannot be used with recursive queries"}
		}
		return buildRecursiveSelect(table, schema, parts, options,
			selectClause, whereClause, orderClause, joins, st, info)
	}

	query := "SELECT " + selectClause
	from := "FROM " + _sq(table, schema)

	return buildAfterSelect(query, from, false, joins, whereClause, groupByClause, orderClause, st.page.values, innerConds, st.count, parts, options)
}

// preferredSerializer: the response is written in Go from the raw row values
// by the hand-written serializers, and PostgreSQL builds only the nested
// embeds as json, passed through untouched. Building the whole response in
// the backend (one json_agg over the select) was tried and rejected (card
// 32703 select-select): it is slower and costs more CPU, moves the escaping
// and the number formatting into the backend, materializes the whole result
// as one datum before the first byte, breaks Content-Range, and cannot serve
// CSV or binary output.
func (DirectQueryBuilder) preferredSerializer() TextSerializer {
	return &JSONSerializer{}
}
