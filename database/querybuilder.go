package database

import (
	"bytes"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
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
	}
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
	_, t1 := splitTableName(rel.RelatedTable)
	label1 := quote(labelWithNumber(t1, stack.level+1))
	var label2 string
	if stack.level > 0 {
		_, t2 := splitTableName(rel.Table)
		label2 = quote(labelWithNumber(t2, stack.level))
	} else if label != "" {
		label2 = label
	} else {
		label2 = quoteParts(rel.Table)
	}

	var from, oc string
	var conds []string
	if rel.Type == Computed {
		// Computed relationship: call the function with parent row reference
		// Use the unqualified table name as the row reference (the implicit alias from outer FROM),
		// with a qualified cast for the type. This is needed because in nested subqueries
		// (inside json_agg wrapper), the schema-qualified reference doesn't resolve as a row ref.
		var parentRef string
		if stack.afterWithClause {
			parentRef = quote("_source")
		} else if stack.level > 0 {
			_, t2 := splitTableName(rel.Table)
			parentRef = quote(labelWithNumber(t2, stack.level))
		} else if label != "" {
			parentRef = label
		} else {
			_, t2 := splitTableName(rel.Table)
			parentRef = quote(t2)
		}
		parentType := quoteParts(rel.Table)
		funcRef := _sq(rel.FunctionName, rel.FunctionSchema)
		from = " FROM " + funcRef + "(" + parentRef + "::" + parentType + ")"
		from += " AS " + label1
		// Apply embedded resource filters and orders (use function name for relPath matching)
		schema, table := splitTableName(rel.RelatedTable)
		stackRelName := join.relName
		embedStack := nextBuildStack(stack, stackRelName, join.relLabel, "", join.relName)
		wc, _ := whereClause(table, schema, join.relLabel, parts.whereConditionsTree, -1, embedStack)
		if wc != "" {
			conds = append(conds, wc)
		}
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
		wc, _ := whereClause(table, schema, join.relLabel, parts.whereConditionsTree, -1, embedStack)
		if wc != "" {
			conds = append(conds, wc)
		}
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
	count = " SELECT 1" + from
	if countConds := append(slices.Clip(conds), join.innerConds...); len(countConds) > 0 {
		count += " WHERE " + strings.Join(countConds, " AND ")
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
			hint := fk // the hint is not an fk but a col? we try
			rel = info.FindRelationshipByCol(tableWithSchema, hint)
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
// It
func selectClause(table, schema, label string, parts *QueryParts, stack BuildStack) (
	selClause string, joins string, keys []string, err error) {

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

	for i, sfield := range parts.selectFields {
		if sfield.relation != nil {
			if sfield.relation.parent != "" {
				parentTable = sfield.relation.parent
			} else {
				parentTable = table
			}
			frel, err := findRelationship(parentTable, sfield.relation.name, sfield.relation.fk, schema, stack.info)
			if err != nil {
				return "", "", nil, err
			}
			if sfield.label == "" {
				labelRelName = sfield.relation.name
			} else {
				labelRelName = sfield.label
			}
			joinName = labelWithNumber(parentTable+"_"+labelRelName, stack.level+1)

			if len(sfield.relation.fields) != 0 {
				if i != 0 {
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
						err = &BuildError{"A spread operation on " + relatedTable + " is not possible"}
						return "", "", nil, err
					}
					selClause += " COALESCE(" + quote(joinName) + ".\"_" + joinName + "\", '[]') AS " + quote(labelRelName)
				case Computed:
					if frel.ReturnIsSet {
						if sfield.relation.spread {
							err = &BuildError{"A spread operation on " + sfield.relation.name + " is not possible"}
							return "", "", nil, err
						}
						selClause += " COALESCE(" + quote(joinName) + ".\"_" + joinName + "\", '[]') AS " + quote(labelRelName)
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
			sc, j, _, err := selectClause(relatedTable, schema, "", internalParts, nextBuildStack(stack, stackRelName, sfield.label, col, sfield.relation.name))
			if err != nil {
				return "", "", nil, err
			}
			joinSeq = append(joinSeq, Join{joinName, sc, j, sfield.relation.inner, frel, sfield.label, sfield.relation.name, sfield.relation.fields, internalParts.innerEmbedConds})
		} else {
			if i != 0 {
				selClause += ", "
			}
			var fieldPart string
			if !stack.afterWithClause {
				if stack.level == 0 {
					if label == "" {
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
	root := stack.level == 0 && !stack.afterWithClause
	if root {
		parts.innerEmbedConds = nil
	}
	if len(joinSeq) > 0 {
		for _, join := range joinSeq {
			relName := join.name
			selectForJoin, count, err := selectForJoinClause(join, label, parts, stack)
			if err != nil {
				return "", "", nil, err
			}
			if join.inner && !stack.afterWithClause {
				// one EXISTS per !inner embed, for the total of a ranged count:
				// the root's restrict it, a nested level's go into its
				// parent's (see selectForJoinClause)
				parts.innerEmbedConds = append(parts.innerEmbedConds, "EXISTS ("+count+")")
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
				joins += " SELECT json_agg(\"_" + relName + "\") AS \"_" + relName + "\""
				joins += " FROM ("
				joins += selectForJoin
				joins += " ) AS \"_" + relName + "\""
			case Computed:
				if join.rel.ReturnIsSet {
					joins += " SELECT json_agg(\"_" + relName + "\") AS \"_" + relName + "\""
					joins += " FROM ("
					joins += selectForJoin
					joins += " ) AS \"_" + relName + "\""
				} else {
					joins += selectForJoin
				}
			}
			joins += ") AS \"" + relName + "\" ON"
			if join.inner && (join.rel.Type == O2M || join.rel.Type == M2M || (join.rel.Type == Computed && join.rel.ReturnIsSet)) {
				joins += " \"" + relName + "\" IS NOT NULL"
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
	return selClause, joins, keys, nil
}

// groupByClause creates a GROUP BY clause when aggregate functions are present
func groupByClause(table, schema string, parts *QueryParts, info *SchemaInfo) string {
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
			fieldname := _sq(table, schema) + "." + quote(sfield.field.name)
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
			fieldname = _stq(o.field.name, schema, table)
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

func appendValue(where, value string, valueList []any, nmarker int, forceParam bool) (string, []any, int) {
	if !forceParam && value == "not_null" {
		where += "NOT NULL"
	} else if !forceParam && (value == "null" ||
		value == "true" ||
		value == "false" ||
		value == "unknown") {
		where += value
	} else if nmarker >= 0 {
		nmarker++
		where += "$" + strconv.Itoa(nmarker)
		valueList = append(valueList, value)
	} else {
		where += quoteLit(value)
	}
	return where, valueList, nmarker
}

// checkFiltersApplied refuses a filter that no level of the built query took,
// as PostgREST does (PGRST108): its relation path names a relation that is
// not embedded in the request, or does not exist. Without it the filter was
// silently dropped and the request answered as if it were not there.
// Filters on embeds are checked only when the representation was built
// (withEmbeds), since only then the embed levels were visited; root filters
// are skipped when skipRoot, as an insert ignores them.
func checkFiltersApplied(node *WhereConditionNode, skipRoot, withEmbeds bool) error {
	if node == nil {
		return nil
	}
	onEmbed := len(node.field.relPath) > 0
	container := node.operator == "" || node.field.name == "" // the root or a boolean operator
	if (onEmbed && !withEmbeds) || (!onEmbed && skipRoot && !container) {
		return nil
	}
	if onEmbed && !node.inserted && node.matched < len(node.field.relPath) {
		return &BuildError{"'" + node.field.relPath[node.matched] + "' is not an embedded resource in this request"}
	}
	if container {
		// its path matched (or it is the root): check the children
		for _, n := range node.children {
			if err := checkFiltersApplied(n, skipRoot, withEmbeds); err != nil {
				return err
			}
		}
	}
	return nil
}

// whereClause builds a WHERE condition string starting a root node of a condition tree.
// Only nodes with fields related to passed table or label are processed (node.inserted is set to true).
// The nmarker integer is used to keep track of the last marker inserted: the first one will be $(nmarker + 1).
// To skip the usage of markers and to embed the condition values in the string pass -1.
// It returns the condition string and the condition values.
func whereClause(table, schema, label string, node *WhereConditionNode, nmarker int, stack BuildStack) (where string, valueList []any) {
	if node == nil {
		return
	}
	if node.operator == "" || node.field.name == "" {
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
	outer:
		for _, n := range node.children {
			// check if the field is part of this where clause, by comparing
			// each item in the field relPath with each item in the stack, table
			// or label or FK column name; the longest prefix any level matches
			// is kept, to name the first unmatched relation (see checkFiltersApplied)
			matched := stack.matchPath(n.field.relPath)
			n.matched = max(n.matched, matched)
			if matched < len(n.field.relPath) || len(n.field.relPath) != len(stack.relPath) {
				continue outer
			}
			if n.inserted {
				// already taken by an earlier embed its path also names, as
				// siblings embedding the same table (a parent and the children
				// of a self-reference): it applies once, without a dangling AND
				continue outer
			}
			if children_where != "" {
				children_where += bool_op
			}
			w, list := whereClause(table, schema, label, n, nmarker, stack)
			children_where += w
			valueList = append(valueList, list...)
			nmarker += len(list)
		}
		where += children_where
		if node.not || node.operator == "OR" {
			where += ")"
		}
		if node.operator != "" {
			// a logic tree is taken as a whole, like a single filter
			node.inserted = true
		}
	} else {
		// skip nodes already inserted
		if node.inserted {
			return
		}
		node.inserted = true
		if node.not {
			where += "NOT "
		}
		var fieldname string
		if stack.level == 0 {
			fieldname = _stq(node.field.name, schema, table)
		} else {
			fieldname = quote(labelWithNumber(table, stack.level)) + "." + quote(node.field.name)
		}
		if node.field.jsonPath != "" {
			fieldname = "(" + toJson(table, schema, node.field.name, fieldname, stack.info) +
				node.field.jsonPath + ")"
		}
		// On a JSON path every value is bound, so `->>k=eq.null` compares against
		// the string 'null' as in PostgREST. The IS operator is the exception:
		// its operand is one of the keywords null/not_null/true/false/unknown
		// (enforced by the parser) and `x IS $1` is a syntax error, so the
		// keyword is written verbatim whatever the field.
		forceParam := node.field.jsonPath != "" && node.operator != "IS"
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
				where, valueList, nmarker = appendValue(where, value, valueList, nmarker, forceParam)
			}
			where += ")"
		} else {
			where += fieldname
			if node.operator == "IN" && len(node.values) == 0 {
				where += " = ANY('{}')"
				return where, valueList
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
					where, valueList, nmarker = appendValue(where, value, valueList, nmarker, forceParam)
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
				where, valueList, _ = appendValue(where, node.values[0], valueList, nmarker, false)
				where += ")"

			} else {
				where, valueList, _ = appendValue(where, node.values[0], valueList, nmarker, forceParam)
			}
		}
	}
	return where, valueList
}

// returningClause builds the RETURNING clause of a mutation and, when the
// representation needs an outer select (embeds, or an order to apply), the
// select over the _source CTE the callers wrap the mutation in.
func returningClause(table, schema string, parts *QueryParts, info *SchemaInfo) (ret, sel string, err error) {
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
	sc, joins, keys, err := selectClause(table, schema, "", parts, BuildStack{info: info, afterWithClause: true})
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
		if len(columnFields) == 0 {
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
	valueList = []any{rows}

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
		onConflict := onConflictClause(table, schema, fieldList, conflictFields, options, info)
		insert += onConflict
	}
	if options.ReturnRepresentation {
		ret, sel, err := returningClause(table, schema, parts, info)
		if err != nil {
			return "", nil, err
		}
		insert += ret
		if sel != "" {
			insert = "WITH _source AS (" + insert + ") " + sel
		}
	}
	if err := checkFiltersApplied(parts.whereConditionsTree, true, options.ReturnRepresentation); err != nil {
		return "", nil, err
	}
	return insert, valueList, nil
}

func (CommonBuilder) BuildUpdate(table string, record Record, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (
	update string, valueList []any, err error) {

	stack := BuildStack{info: info}
	schema := options.Schema
	if err := checkColumns(table, schema, parts, info); err != nil {
		return "", nil, err
	}
	var pairs string
	var i int
	for key := range record {
		// check if there are specified columns
		if len(parts.columnFields) > 0 {
			if _, ok := parts.columnFields[key]; !ok {
				continue
			}
		}
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
	whereClause, whereValueList := whereClause(table, schema, "", parts.whereConditionsTree, i, stack)
	valueList = append(valueList, whereValueList...)
	update = "UPDATE " + _sq(table, schema) + " SET " + pairs
	if whereClause != "" {
		update += " WHERE " + whereClause
	}
	if options.ReturnRepresentation {
		ret, sel, err := returningClause(table, schema, parts, info)
		if err != nil {
			return "", nil, err
		}
		update += ret
		if sel != "" {
			update = "WITH _source AS (" + update + ") " + sel
		}
	}
	if err := checkFiltersApplied(parts.whereConditionsTree, false, options.ReturnRepresentation); err != nil {
		return "", nil, err
	}
	return update, valueList, nil
}

func (CommonBuilder) BuildDelete(table string, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (
	delete string, valueList []any, err error) {

	stack := BuildStack{info: info}
	schema := options.Schema
	whereClause, valueList := whereClause(table, schema, "", parts.whereConditionsTree, 0, stack)
	delete = "DELETE FROM " + _sq(table, schema)
	if whereClause != "" {
		delete += " WHERE " + whereClause
	}
	if options.ReturnRepresentation {
		ret, sel, err := returningClause(table, schema, parts, info)
		if err != nil {
			return "", nil, err
		}
		delete += ret
		if sel != "" {
			delete = "WITH _source AS (" + delete + ") " + sel
		}
	}
	if err := checkFiltersApplied(parts.whereConditionsTree, false, options.ReturnRepresentation); err != nil {
		return "", nil, err
	}
	return delete, valueList, nil
}

func buildAfterSelect(query, from, joins, whereClause, groupByClause, orderClause string, valueList []any, parts *QueryParts, options *QueryOptions) (string, []any, error) {
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
	var limit int64 = -1
	if parts.limit != "" || options.HasRange && options.RangeMax != -1 {
		nmarker += 1
		query += " LIMIT $" + strconv.Itoa(nmarker)
		if options.HasRange {
			limit = options.RangeMax - options.RangeMin + 1
		} else {
			limit, _ = strconv.ParseInt(parts.limit, 10, 64)
			options.RangeMax = limit - 1
		}
		valueList = append(valueList, limit)
	}
	var offset int64 = -1
	if parts.offset != "" || options.HasRange {
		nmarker += 1
		query += " OFFSET $" + strconv.Itoa(nmarker)
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
		valueList = append(valueList, offset)
	}
	if options.Count != "" && (limit != -1 || offset > 0) {
		// The total of a ranged request is PostgREST's (readPlanToCountQuery):
		// the root rows that pass the filters and have a row in each !inner
		// embed. The left embeds do not restrict them, and the GROUP BY is
		// left out: a grouped aggregate totals the rows it groups. Without a
		// range the total is the number of rows read, as PostgREST's page count.
		conds := parts.innerEmbedConds
		if whereClause != "" {
			conds = append([]string{whereClause}, conds...)
		}
		countQuery := "WITH Total AS (SELECT COUNT(*) AS __count " + from
		if len(conds) > 0 {
			countQuery += " WHERE " + strings.Join(conds, " AND ")
		}
		query = countQuery +
			"), Data AS (" + query +
			`),
			PseudoRow AS (
				SELECT 1 AS Dummy
			)
			SELECT 
				__t.__count,
				d.*
			FROM PseudoRow
			CROSS JOIN Total __t
			LEFT JOIN Data d ON true;`
	}
	return query, valueList, nil
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

func buildRecursiveSelect(table, schema string, parts *QueryParts, options *QueryOptions,
	selectClause, mainWhere, orderClause, joins string,
	valueList []any, info *SchemaInfo) (string, []any, error) {

	rec := parts.recursive
	qtable := _sq(table, schema)
	startField := quote(rec.StartField)
	recurseField := quote(rec.RecurseField)
	cteName := quote("__recursive")

	// Computed-relationship embeds call a function with the parent ROW, emitted as
	// `"table"::schema.table` (an unqualified row ref + cast). The tablePrefix->cte
	// rewrite below only repoints qualified column refs (`"schema"."table".col`), so
	// that bare row ref is left dangling — the base-table alias doesn't exist in the
	// recursive outer query (rows live in the CTE). When such an embed is present we
	// carry the source row as a composite `__row` column on the CTE and repoint the
	// embed at it. `via`+embed is rejected below, so this only applies single-table.
	rowRef := quote(table) + "::" + qtable
	needsRow := rec.ViaTable == "" && joins != "" && strings.Contains(joins, rowRef)
	rowCol := ""
	if needsRow {
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

	// Parameter marker allocation order:
	// 1. Via edge conditions (if any) — built first so they get the lowest markers
	// 2. Start value ($N)
	// 3. Max depth ($N+1)
	// This ordering matters because whereClause() assigns markers sequentially from nmarker.
	nmarker := len(valueList)

	// Build via edge table WHERE clause using the standard whereClause builder
	var viaWhere string
	if rec.ViaConditions != nil {
		var viaValues []any
		viaWhere, viaValues = whereClause(rec.ViaTable, schema, "", rec.ViaConditions, nmarker, BuildStack{})
		valueList = append(valueList, viaValues...)
		nmarker = len(valueList)
	}

	// Build the walk-prune WHERE (walk.* filters) against the main table.
	// Marker order stays via -> walk -> start -> depth.
	var walkWhere string
	if rec.WalkConditions != nil {
		var walkValues []any
		walkWhere, walkValues = whereClause(table, schema, "", rec.WalkConditions, nmarker, BuildStack{info: info})
		valueList = append(valueList, walkValues...)
		nmarker = len(valueList)
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
		nmarker++
		q.WriteString(" WHERE " + qtable + "." + startField + " = $" + strconv.Itoa(nmarker))
		valueList = append(valueList, rec.StartValue)
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
		nmarker++
		q.WriteString(" WHERE " + cteName + ".__depth < $" + strconv.Itoa(nmarker))
		valueList = append(valueList, maxDepth)
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
		nmarker++
		startMarker := "$" + strconv.Itoa(nmarker)
		q.WriteString(" WHERE " + qtable + "." + startField + " = " + startMarker)
		valueList = append(valueList, rec.StartValue)
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
		nmarker++
		q.WriteString(" WHERE " + cteName + ".__depth < $" + strconv.Itoa(nmarker))
		valueList = append(valueList, maxDepth)
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
	tablePrefix := qtable + "."
	ctePrefix := cteName + "."
	// outer rewrites a clause built against the table (select list, result filters,
	// ORDER BY) for the outer query. Single-table mode selects from the CTE, which
	// carries the rows, so every table reference moves to the CTE name. Via mode joins
	// the table back onto "__dedup", so its columns are read from the table itself and
	// only the pseudo-columns move, "table"."__depth" to "__dedup"."__depth".
	outer := func(clause string) string {
		if rec.ViaTable == "" {
			return strings.ReplaceAll(clause, tablePrefix, ctePrefix)
		}
		for _, col := range []string{"__depth", "__path"} {
			clause = strings.ReplaceAll(clause, tablePrefix+quote(col), dedupName+"."+quote(col))
		}
		return clause
	}

	// Build the SELECT list (without keyword).
	var sel string
	if selectClause != "*" {
		sel = outer(selectClause)
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

	// Embed joins (LEFT JOIN LATERAL) correlate to the base-table alias; rewrite
	// the top-level "table". -> "__recursive". correlation. The lateral's own numbered
	// alias (relationship_1.) is untouched, so the rewrite is surgical. via+embed is
	// rejected by the guard above, so joinsClause is only ever set in single-table mode.
	var joinsClause string
	if joins != "" {
		joinsClause = " " + strings.ReplaceAll(joins, tablePrefix, ctePrefix)
		if needsRow {
			// Repoint computed-relationship embeds from the (now absent) base-table row
			// ref to the CTE's `__row` composite column. `__row` is already the table's
			// row type, so the original `::schema.table` cast is dropped with the ref.
			joinsClause = strings.ReplaceAll(joinsClause, rowRef, ctePrefix+quote("__row"))
		}
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
		outerWhere += outer(mainWhere)
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
		q.WriteString(" ORDER BY " + outer(orderClause))
	}

	// Limit
	var limit int64 = -1
	if parts.limit != "" || options.HasRange && options.RangeMax != -1 {
		nmarker++
		q.WriteString(" LIMIT $" + strconv.Itoa(nmarker))
		if options.HasRange {
			limit = options.RangeMax - options.RangeMin + 1
		} else {
			limit, _ = strconv.ParseInt(parts.limit, 10, 64)
			options.RangeMax = limit - 1
		}
		valueList = append(valueList, limit)
	}

	// Offset
	if parts.offset != "" || options.HasRange {
		nmarker++
		q.WriteString(" OFFSET $" + strconv.Itoa(nmarker))
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
		valueList = append(valueList, offset)
	}

	return q.String(), valueList, nil
}

func (CommonBuilder) BuildExecute(name string, f *Function, record Record, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (
	query string, valueList []any, err error) {

	stack := BuildStack{info: info}
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
		if key != "" { // unnamed parameter @@ to be continued
			pairs += quote(key)
			pairs += " := "
		}
		pairs += "$" + strconv.Itoa(i)
		valueList = append(valueList, record[key])
	}

	// Extract the return type and discover if it is a table.
	// In that case, we will use its name and schema to compose the select clause
	var t, s string
	if f != nil {
		rettype := info.GetTypeById(f.ReturnTypeId)
		if rettype.IsTable {
			if rettype.IsDomain {
				t = rettype.DomainSubType
			} else {
				t = rettype.Name
			}
			s = rettype.Schema
		}
	}
	selectClause, joins, _, err := selectClause(t, s, "t", parts, stack)
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
	whereClause, whereValueList := whereClause("t", "", name, parts.whereConditionsTree, i, stack)
	valueList = append(valueList, whereValueList...)
	if err := checkFiltersApplied(parts.whereConditionsTree, false, true); err != nil {
		return "", nil, err
	}
	// patch order fields
	for i := range parts.orderFields {
		parts.orderFields[i].field.tablename = "t"
	}
	orderClause, err := orderClause("t", "", "", 0, parts.orderFields, parts.selectFields, info)
	if err != nil {
		return "", nil, err
	}
	query = "SELECT " + selectClause
	from := "FROM " + _sq(name, schema) + "(" + pairs + ") t "

	return buildAfterSelect(query, from, joins, whereClause, "", orderClause, valueList, parts, options)
}

type DirectQueryBuilder struct {
	CommonBuilder
}

func (DirectQueryBuilder) BuildSelect(table string, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (string, []any, error) {
	stack := BuildStack{info: info}
	schema := options.Schema
	selectClause, joins, _, err := selectClause(table, schema, "", parts, stack)
	if err != nil {
		return "", nil, err
	}
	whereClause, valueList := whereClause(table, schema, "", parts.whereConditionsTree, 0, stack)
	if err := checkFiltersApplied(parts.whereConditionsTree, false, true); err != nil {
		return "", nil, err
	}
	groupByClause := groupByClause(table, schema, parts, info)
	orderClause, err := orderClause(table, schema, "", 0, parts.orderFields, parts.selectFields, info)
	if err != nil {
		return "", nil, err
	}

	if parts.recursive != nil {
		if groupByClause != "" {
			return "", nil, &ParseError{"aggregate functions cannot be used with recursive queries"}
		}
		return buildRecursiveSelect(table, schema, parts, options,
			selectClause, whereClause, orderClause, joins, valueList, info)
	}

	query := "SELECT " + selectClause
	from := "FROM " + _sq(table, schema)

	return buildAfterSelect(query, from, joins, whereClause, groupByClause, orderClause, valueList, parts, options)
}

func (DirectQueryBuilder) preferredSerializer() TextSerializer {
	return &JSONSerializer{}
}

type QueryWithJSON struct {
	CommonBuilder
}

func (QueryWithJSON) BuildSelect(table string, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (string, []any, error) {
	stack := BuildStack{info: info}
	schema := options.Schema
	selectClause, joins, _, err := selectClause(table, schema, "", parts, stack)
	if err != nil {
		return "", nil, err
	}
	whereClause, valueList := whereClause(table, schema, "", parts.whereConditionsTree, 0, stack)
	if err := checkFiltersApplied(parts.whereConditionsTree, false, true); err != nil {
		return "", nil, err
	}
	groupByClause := groupByClause(table, schema, parts, info)
	orderClause, err := orderClause(table, schema, "", 0, parts.orderFields, parts.selectFields, info)
	if err != nil {
		return "", nil, err
	}

	if parts.recursive != nil {
		if groupByClause != "" {
			return "", nil, &ParseError{"aggregate functions cannot be used with recursive queries"}
		}
		return buildRecursiveSelect(table, schema, parts, options,
			selectClause, whereClause, orderClause, joins, valueList, info)
	}

	query := "SELECT "
	if selectClause == "*" {
		query += "json_agg(" + table + ")"
	} else {
		query += "SELECT " + selectClause
	}
	from := "FROM " + table
	return buildAfterSelect(query, from, joins, whereClause, groupByClause, orderClause, valueList, parts, options)
}

func (QueryWithJSON) preferredSerializer() TextSerializer {
	return DatabaseJSONSerializer{}
}
