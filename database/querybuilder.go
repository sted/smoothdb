package database

import (
	"sort"
	"strconv"
	"strings"

	"github.com/samber/lo"
)

type QueryBuilder interface {
	BuildSelect(table string, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (string, []any, error)
	BuildInsert(table string, records []Record, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (string, []any, error)
	BuildUpdate(table string, record Record, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (string, []any, error)
	BuildDelete(table string, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (string, []any, error)
	BuildExecute(table string, record Record, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (string, []any, error)

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
}

type BuildError struct {
	msg string // description of error
}

func (e BuildError) Error() string { return e.msg }

// BuildStack represents the context when navigating the AST produced by the parser
type BuildStack struct {
	info            *SchemaInfo // database information (tables, contraints, etc)
	level           int         // depth
	relPath         []string    // sequence of nested tables
	labelPath       []string    // sequence of nested labels for the correspondent tables (can contain empty strings)
	colPath         []string    // sequence of FK column names that point to the correspondent tables (can contain empty strings)
	afterWithClause bool        //
}

// nextBuildStack creates a stack with a new level
func nextBuildStack(stack BuildStack, rel string, label string, col string) BuildStack {
	relPath := append(stack.relPath, rel)
	labelPath := append(stack.labelPath, label)
	colPath := append(stack.colPath, col)
	return BuildStack{stack.info, stack.level + 1, relPath, labelPath, colPath, false}
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
	var fieldPart string

	if sfield.aggregate == "" {
		// Regular field without aggregate
		fieldname := _sq(table, schema) + "." + quoteIf(sfield.field.name, !isStar(sfield.field.name))
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
			fieldname := _sq(table, schema) + "." + quoteIf(sfield.field.name, !isStar(sfield.field.name))
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

func selectForJoinClause(join Join, label string, parts *QueryParts, stack BuildStack) (sel string, err error) {
	rel := join.rel
	sel = " SELECT " + join.fields
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
		sel += " FROM " + funcRef + "(" + parentRef + "::" + parentType + ")"
		sel += " AS " + label1
		if join.nested != "" {
			sel += " " + join.nested
		}
		// Apply embedded resource filters (use function name for relPath matching)
		schema, table := splitTableName(rel.RelatedTable)
		stackRelName := join.relName
		wc, _ := whereClause(table, schema, join.relLabel, parts.whereConditionsTree, -1, nextBuildStack(stack, stackRelName, join.relLabel, ""))
		if wc != "" {
			sel += " WHERE " + wc
		}
		oc, err := orderClause(table, schema, label1, stack.level+1, parts.orderFields, join.selectFields, stack.info)
		if err != nil {
			return "", err
		}
		if oc != "" {
			sel += " ORDER BY " + oc
		}
	} else {
		// FK-based relationship
		sel += " FROM " + quoteParts(rel.RelatedTable)
		sel += " AS " + label1
		if rel.JunctionTable != "" {
			sel += ", " + quoteParts(rel.JunctionTable)
		}
		if join.nested != "" {
			sel += " " + join.nested
		}
		sel += " WHERE "
		if rel.JunctionTable == "" {
			for i := range rel.Columns {
				if i != 0 {
					sel += " AND "
				}
				sel += label1 + "." + quote(rel.RelatedColumns[i])
				sel += " = "
				if stack.afterWithClause {
					sel += quote("_source")
				} else {
					sel += label2
				}
				sel += "." + quote(rel.Columns[i])
			}
		} else {
			// M2M Join
			for i := range rel.JColumns {
				if i != 0 {
					sel += " AND "
				}
				sel += quoteParts(rel.JunctionTable) + "." + quote(rel.JColumns[i])
				sel += " = "
				if stack.afterWithClause {
					sel += quote("_source")
				} else {
					sel += label2
				}
				sel += "." + quote(rel.Columns[i])
			}
			for i := range rel.JRelatedColumns {
				sel += " AND "
				sel += quoteParts(rel.JunctionTable) + "." + quote(rel.JRelatedColumns[i])
				sel += " = "
				sel += label1 + "." + quote(rel.RelatedColumns[i])
			}
		}
		// where and order clause for the internal select: the expressions related to
		// the external query are skipped inside the functions.
		// If the internal table is equal to the external one we avoid repeating
		// the expressions.
		if rel.Table != rel.RelatedTable {
			schema, table := splitTableName(rel.RelatedTable)
			col := ""
			if len(rel.Columns) == 1 {
				col = rel.Columns[0]
			}
			whereClause, _ := whereClause(table, schema, join.relLabel, parts.whereConditionsTree, -1, nextBuildStack(stack, table, join.relLabel, col))
			if whereClause != "" {
				sel += " AND " + whereClause
			}
			oc, err := orderClause(table, schema, label1, stack.level+1, parts.orderFields, join.selectFields, stack.info)
			if err != nil {
				return "", err
			}
			if oc != "" {
				sel += " ORDER BY " + oc
			}
		}
	}
	return sel, nil
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
			internalParts := &QueryParts{selectFields: sfield.relation.fields, whereConditionsTree: parts.whereConditionsTree}
			// For computed relationships, use the function name in the relPath so WHERE filters match
			stackRelName := relatedTable
			if frel.Type == Computed {
				stackRelName = sfield.relation.name
			}
			col := ""
			if frel.Type != Computed && len(frel.Columns) == 1 {
				col = frel.Columns[0]
			}
			sc, j, _, err := selectClause(relatedTable, schema, "", internalParts, nextBuildStack(stack, stackRelName, sfield.label, col))
			if err != nil {
				return "", "", nil, err
			}
			joinSeq = append(joinSeq, Join{joinName, sc, j, sfield.relation.inner, frel, sfield.label, sfield.relation.name, sfield.relation.fields})
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
						fieldPart = prepareField(label, "", sfield, stack.info)
					}
				} else {
					fieldPart = prepareField(labelWithNumber(table, stack.level), "", sfield, stack.info)
				}
			} else {
				fieldPart = prepareField("_source", "", sfield, stack.info)
			}
			selClause += fieldPart
		}
	}
	if selClause == "" {
		selClause = "*"
	}
	if len(joinSeq) > 0 {
		for _, join := range joinSeq {
			relName := join.name
			selectForJoin, err := selectForJoinClause(join, label, parts, stack)
			if err != nil {
				return "", "", nil, err
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

func orderClause(table, schema, label string, level int, orderFields []OrderField,
	selectFields []SelectField, info *SchemaInfo) (string, error) {
	var order string
	for _, o := range orderFields {
		if o.field.tablename != table {
			// skip order fields for other tables
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
			// check if the field is part of this where clause:

			// 1. first by comparing the stack depth
			if len(n.field.relPath) != len(stack.relPath) {
				continue
			}
			// 2. then by comparing each item in the field relPath
			//    with each item in the stack, table or label or FK column name
			for i := range n.field.relPath {
				if n.field.relPath[i] != stack.relPath[i] &&
					n.field.relPath[i] != stack.labelPath[i] &&
					(len(stack.colPath) <= i || n.field.relPath[i] != stack.colPath[i]) {
					continue outer
				}
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
	sc, joins, keys, _ := selectClause(table, schema, "", parts, BuildStack{info: info, afterWithClause: true})
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
			if o.relation == "" && o.field.tablename == table {
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
	if options.IgnoreDuplicates {
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

func (CommonBuilder) BuildInsert(table string, records []Record, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (
	insert string, valueList []any, err error) {

	var fields string
	var fieldList []string
	var values string

	// if len(records) == 0 {
	// 	return "", nil, fmt.Errorf("no records to insert")
	// }
	// The column list. With ?columns= it is exactly the listed set, for every
	// row: a key absent from an object is inserted as NULL, a key not listed is
	// ignored (PostgREST passes such a body to json_to_recordset untouched).
	// Otherwise it is the key set of the first object, which every other object
	// must share: taking it from the first object alone silently dropped the
	// keys the others added, so a non-uniform array is refused as PostgREST does.
	if len(parts.columnFields) > 0 {
		fieldList = lo.Keys(parts.columnFields)
	} else {
		for i := 1; i < len(records); i++ {
			if !sameKeys(records[0], records[i]) {
				return "", nil, &BuildError{"All object keys must match"}
			}
		}
		fieldList = lo.Keys(records[0])
	}
	// alphabetical, so the SQL text is stable across runs (see orderedRecordKeys)
	sort.Strings(fieldList)
	n := len(fieldList)
	for _, key := range fieldList {
		if fields != "" {
			fields += ", "
		}
		fields += quote(key)
	}
	var j int
	for i, record := range records {
		if i > 0 {
			values += "), ("
		}
		for _, f := range fieldList {
			if j > 0 {
				values += ", "
			}
			j += 1
			values += "$" + strconv.Itoa(i*n+j)
			valueList = append(valueList, record[f])
		}
		j = 0
	}
	schema := options.Schema
	if n > 0 {
		insert = "INSERT INTO " + _sq(table, schema) + " (" + fields + ") VALUES (" + values + ")"
	} else {
		insert = "INSERT INTO " + _sq(table, schema) + " DEFAULT VALUES"
	}
	if options.MergeDuplicates || options.IgnoreDuplicates || len(parts.conflictFields) > 0 {
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
	return insert, valueList, nil
}

func (CommonBuilder) BuildUpdate(table string, record Record, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (
	update string, valueList []any, err error) {

	stack := BuildStack{info: info}
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
	schema := options.Schema
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
		countQuery := "WITH Total AS (SELECT COUNT(*) AS __count " + from
		if whereClause != "" {
			countQuery += " WHERE " + whereClause
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

func (CommonBuilder) BuildExecute(name string, record Record, parts *QueryParts, options *QueryOptions, info *SchemaInfo) (
	query string, valueList []any, err error) {

	stack := BuildStack{info: info}
	schema := options.Schema

	var f *Function
	if info != nil {
		f = info.GetFunction(_s(name, schema))
	}

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
