package database

import (
	"strconv"

	"github.com/samber/lo"
)

type QueryBuilder interface {
	BuildSelect(table string, parts *QueryParts, options *QueryOptions, info *DbInfo) (string, []any, error)
	BuildInsert(table string, records []Record, parts *QueryParts, options *QueryOptions, info *DbInfo) (string, []any, error)
	BuildUpdate(table string, record Record, parts *QueryParts, options *QueryOptions, info *DbInfo) (string, []any, error)
	BuildDelete(table string, parts *QueryParts, options *QueryOptions, info *DbInfo) (string, []any, error)
	BuildExecute(table string, record Record, parts *QueryParts, options *QueryOptions, info *DbInfo) (string, []any, error)

	preferredSerializer() ResultSerializer
}

type Join struct {
	name     string
	fields   string
	inner    string
	rel      *Relationship
	relLabel string
}

type BuildError struct {
	msg string // description of error
}

func (e *BuildError) Error() string { return e.msg }

func prepareField(table, schema string, sfield SelectField) string {
	var fieldPart string
	fieldname := _sq(table, schema) + "." + quoteIf(sfield.field.name, !isStar(sfield.field.name))
	if sfield.field.jsonPath != "" {
		fieldPart += "(" + fieldname + sfield.field.jsonPath + ")"
	} else {
		fieldPart += fieldname
	}
	if sfield.cast != "" {
		fieldPart += "::" + sfield.cast
	}
	if sfield.label != "" {
		fieldPart += " AS \"" + sfield.label + "\""
	}
	return fieldPart
}

func labelWithNumber(table string, num int) string {
	return table + "_" + strconv.Itoa(num)
}

func selectForJoinClause(join Join, parts *QueryParts, level int, afterWithClause bool) (sel string) {
	rel := join.rel
	sel = " SELECT " + join.fields
	sel += " FROM " + quoteParts(rel.RelatedTable)
	_, t1 := splitTableName(rel.RelatedTable)
	label1 := quote(labelWithNumber(t1, level))
	sel += " AS " + label1
	var label2 string
	if level > 1 {
		_, t2 := splitTableName(rel.Table)
		label2 = quote(labelWithNumber(t2, level-1))
	} else {
		label2 = quoteParts(rel.Table)
	}
	if rel.JunctionTable != "" {
		sel += ", " + quoteParts(rel.JunctionTable)
	}
	if join.inner != "" {
		sel += " " + join.inner
	}
	sel += " WHERE "
	if rel.JunctionTable == "" {
		for i := range rel.Columns {
			if i != 0 {
				sel += " AND "
			}
			sel += label1 + "." + quote(rel.RelatedColumns[i])
			sel += " = "
			if afterWithClause {
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
			if afterWithClause {
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
		whereClause, _ := whereClause(table, schema, label1, parts.whereConditionsTree, join.relLabel, -1)
		if whereClause != "" {
			sel += " AND " + whereClause
		}
		orderClause := orderClause(table, schema, label1, parts.orderFields)
		if orderClause != "" {
			sel += " ORDER BY " + orderClause
		}
	}
	return sel
}

func findRelationship(table, relation, schema string, info *DbInfo) (rel *Relationship, err error) {
	rels := info.GetRelationships(_s(table, schema))
	frels := filterRelationships(rels, _s(relation, schema))
	nrels := len(frels)
	switch {
	case nrels == 0:
		// search self rel by column (try to see if relation is an fk column)
		rel = info.FindRelationshipByCol(_s(table, schema), relation)

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

func selectClause(table, schema string, parts *QueryParts, info *DbInfo, level int, afterWithClause bool) (
	selClause string, joins string, keys []string, err error) {

	var parentTable string
	var relatedTable string
	var labelRelName string
	var joinName string

	joinSeq := []Join{}

	for i, sfield := range parts.selectFields {
		if i != 0 {
			selClause += ", "
		}
		if sfield.relation != nil {
			if sfield.relation.parent != "" {
				parentTable = sfield.relation.parent
			} else {
				parentTable = table
			}
			frel, err := findRelationship(parentTable, sfield.relation.name, schema, info)
			if err != nil {
				return "", "", nil, err
			}
			_, relatedTable = splitTableName(frel.RelatedTable)
			if sfield.label == "" {
				labelRelName = relatedTable
			} else {
				labelRelName = sfield.label
			}
			joinName = labelWithNumber(parentTable+"_"+labelRelName, level+1)

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
			}
			internalParts := &QueryParts{selectFields: sfield.relation.fields}
			sc, j, _, err := selectClause(relatedTable, schema, internalParts, info, level+1, false)
			if err != nil {
				return "", "", nil, err
			}
			joinSeq = append(joinSeq, Join{joinName, sc, j, frel, sfield.label})
		} else {
			var fieldPart string
			if !afterWithClause {
				if level == 0 {
					fieldPart = prepareField(table, schema, sfield)
				} else {
					fieldPart = prepareField(labelWithNumber(table, level), "", sfield)
				}
			} else {
				fieldPart = prepareField("_source", "", sfield)
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
			selectForJoin := selectForJoinClause(join, parts, level+1, afterWithClause)
			joins += " LEFT JOIN LATERAL ("
			switch join.rel.Type {
			case M2O, O2O:
				joins += selectForJoin
			case O2M, M2M:
				joins += " SELECT json_agg(\"_" + relName + "\") AS \"_" + relName + "\""
				joins += " FROM ("
				joins += selectForJoin
				joins += " ) AS \"_" + relName + "\""
			}
			joins += ") AS \"" + relName + "\" ON TRUE"
			for i := range join.rel.Columns {
				keys = append(keys, quoteParts(join.rel.Table)+"."+quote(join.rel.Columns[i]))
			}
		}
	}
	return selClause, joins, keys, nil
}

func orderClause(table, schema, label string, orderFields []OrderField) string {
	var order string
	for _, o := range orderFields {
		if o.field.tablename != table {
			// skip where filters for other tables
			continue
		}
		if order != "" {
			order += ", "
		}
		if label == "" {
			order += _stq(o.field.name, schema, table)
		} else {
			order += label + "." + quote(o.field.name)
		}
		order += o.field.jsonPath
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
	return order
}

func appendValue(where, value string, valueList []any, nmarker int) (string, []any, int) {
	if value == "null" ||
		value == "true" ||
		value == "false" ||
		value == "unknown" {
		where += value
	} else if nmarker >= 0 {
		nmarker++
		where += "$" + strconv.Itoa(nmarker)
		valueList = append(valueList, value)
	} else {
		where += "'" + value + "'"
	}
	return where, valueList, nmarker
}

// whereClause builds a WHERE condition string starting from the table name, the schema, and a root node
// of a condition tree. The other string is used as an additional value together with table to skip internal
// conditions: all the conditions nodes on a field with a table name different from table or other are not processed.
// The nmarker integer is used to indicate the last marker inserted: the first one will be $(nmarker + 1).
// To skip the usage of markers and to embed the condition values in the string pass -1.
// It returns the condition string and the condition values.
func whereClause(table, schema, label string, node *WhereConditionNode, other string, nmarker int) (where string, valueList []any) {
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
		for _, n := range node.children {
			if n.field.tablename != table &&
				n.field.tablename != other {
				// skip where filters for other tables
				continue
			}
			if children_where != "" {
				children_where += bool_op
			}
			w, list := whereClause(table, schema, label, n, other, nmarker)
			children_where += w
			valueList = append(valueList, list...)
			nmarker += len(list)
		}
		where += children_where
		if node.not || node.operator == "OR" {
			where += ")"
		}
	} else {
		if node.not {
			where += "NOT "
		}
		if label == "" {
			where += _stq(node.field.name, schema, table)
		} else {
			where += label + "." + quote(node.field.name)
		}
		where += node.field.jsonPath
		where += " " + node.operator + " "
		if node.operator == "IN" {
			where += "("
			for i, value := range node.values {
				if i != 0 {
					where += ", "
				}
				where, valueList, nmarker = appendValue(where, value, valueList, nmarker)
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
				where += "'" + arg + "'"
				where += ", "
			}
			where, valueList, _ = appendValue(where, node.values[0], valueList, nmarker)
			where += ")"

		} else {
			where, valueList, _ = appendValue(where, node.values[0], valueList, nmarker)
		}
	}
	return where, valueList
}

// returningClause
func returningClause(table, schema string, parts *QueryParts, info *DbInfo) (ret, sel string) {
	ret += " RETURNING "
	if len(parts.selectFields) == 0 {
		ret += "*"
	} else {
		var f, fields string
		var fieldMap = make(map[string]struct{})
		var hasResourceEmbed bool
		for _, sfield := range parts.selectFields {
			if sfield.relation != nil {
				hasResourceEmbed = true
			} else {
				if fields != "" {
					fields += ", "
				}
				f = prepareField(table, schema, sfield)
				fields += f
				fieldMap[f] = struct{}{}
			}
		}
		ret += fields
		if hasResourceEmbed {
			sc, joins, keys, _ := selectClause(table, schema, parts, info, 0, true)
			// add foreign keys to Returning clause if they are not already present
			for _, k := range keys {
				if _, exists := fieldMap[k]; !exists {
					if ret != "" {
						ret += ", "
					}
					ret += k
				}
			}
			sel = "SELECT " + sc + " FROM _source"
			if joins != "" {
				sel += " " + joins
			}
		}
	}
	return
}

func onConflictClause(table, schema string, fields []string,
	conflictFields []string, options *QueryOptions, info *DbInfo) string {

	var cFields []string
	hasConflictFields := len(conflictFields) > 0
	if hasConflictFields {
		// @@ should we check if these fields are UNIQUE fields?
		cFields = conflictFields
	} else {
		pk, ok := info.GetPrimaryKey(_s(table, schema))
		if !ok {
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

type CommonBuilder struct{}

func (CommonBuilder) BuildInsert(table string, records []Record, parts *QueryParts, options *QueryOptions, info *DbInfo) (
	insert string, valueList []any, err error) {

	var fields string
	var fieldList []string
	var values string

	// if len(records) == 0 {
	// 	return "", nil, fmt.Errorf("no records to insert")
	// }
	var n int
	for key := range records[0] {
		// check if there are specified columns
		if len(parts.columnFields) > 0 {
			if _, ok := parts.columnFields[key]; !ok {
				continue
			}
		}
		n += 1
		if fields != "" {
			fields += ", "
		}
		fields += quote(key)
		fieldList = append(fieldList, key)
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
		if err != nil {
			return "", nil, err
		}
		insert += onConflict
	}
	if options.ReturnRepresentation {
		ret, sel := returningClause(table, schema, parts, info)
		insert += ret
		if sel != "" {
			insert = "WITH _source AS (" + insert + ") " + sel
		}
	}
	return insert, valueList, nil
}

func (CommonBuilder) BuildUpdate(table string, record Record, parts *QueryParts, options *QueryOptions, info *DbInfo) (
	update string, valueList []any, err error) {

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
	whereClause, whereValueList := whereClause(table, schema, "", parts.whereConditionsTree, "", i)
	valueList = append(valueList, whereValueList...)
	update = "UPDATE " + _sq(table, schema) + " SET " + pairs
	if whereClause != "" {
		update += " WHERE " + whereClause
	}
	if options.ReturnRepresentation {
		ret, sel := returningClause(table, schema, parts, info)
		update += ret
		if sel != "" {
			update = "WITH _source AS (" + update + ") " + sel
		}
	}
	return update, valueList, nil
}

func (CommonBuilder) BuildDelete(table string, parts *QueryParts, options *QueryOptions, info *DbInfo) (
	delete string, valueList []any, err error) {

	schema := options.Schema
	whereClause, valueList := whereClause(table, schema, "", parts.whereConditionsTree, "", 0)
	delete = "DELETE FROM " + _sq(table, schema)
	if whereClause != "" {
		delete += " WHERE " + whereClause
	}
	if options.ReturnRepresentation {
		ret, sel := returningClause(table, schema, parts, info)
		delete += ret
		if sel != "" {
			delete = "WITH _source AS (" + delete + ") " + sel
		}
	}
	return delete, valueList, nil
}

func (CommonBuilder) BuildExecute(name string, record Record, parts *QueryParts, options *QueryOptions, info *DbInfo) (
	exec string, valueList []any, err error) {

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
		pairs += " := $" + strconv.Itoa(i)
		valueList = append(valueList, record[key])
	}
	schema := options.Schema
	selectClause, _, _, err := selectClause("t", "", parts, info, 0, false)
	if err != nil {
		return "", nil, err
	}
	whereClause, whereValueList := whereClause("t", "", "", parts.whereConditionsTree, name, i)
	valueList = append(valueList, whereValueList...)
	exec = "SELECT " + selectClause + " FROM " + _sq(name, schema) + "(" + pairs + ") t "
	if whereClause != "" {
		exec += " WHERE " + whereClause
	}
	return exec, valueList, nil
}

type DirectQueryBuilder struct {
	CommonBuilder
}

func (DirectQueryBuilder) BuildSelect(table string, parts *QueryParts, options *QueryOptions, info *DbInfo) (string, []any, error) {
	schema := options.Schema
	selectClause, joins, _, err := selectClause(table, schema, parts, info, 0, false)
	if err != nil {
		return "", nil, err
	}
	whereClause, valueList := whereClause(table, schema, "", parts.whereConditionsTree, "", 0)
	orderClause := orderClause(table, schema, "", parts.orderFields)
	query := "SELECT " + selectClause + " FROM " + _sq(table, schema)
	if joins != "" {
		query += " " + joins
	}
	if whereClause != "" {
		query += " WHERE " + whereClause
	}
	if orderClause != "" {
		query += " ORDER BY " + orderClause
	}
	if parts.limit != "" {
		query += " LIMIT " + parts.limit
	}
	if parts.offset != "" {
		query += " OFFSET " + parts.offset
	}
	return query, valueList, nil
}

func (DirectQueryBuilder) preferredSerializer() ResultSerializer {
	return DirectJSONSerializer{}
}

type QueryWithJSON struct {
	CommonBuilder
}

func (QueryWithJSON) BuildSelect(table string, parts *QueryParts, options *QueryOptions, info *DbInfo) (string, []any, error) {
	schema := options.Schema
	selectClause, joins, _, err := selectClause(table, schema, parts, info, 0, false)
	if err != nil {
		return "", nil, err
	}
	whereClause, valueList := whereClause(table, schema, "", parts.whereConditionsTree, "", 0)
	orderClause := orderClause(table, schema, "", parts.orderFields)
	query := "SELECT "
	if selectClause == "*" {
		query += "json_agg(" + table + ")" + " FROM " + table
	} else {
		query += "SELECT " + selectClause + " FROM " + table
	}
	if joins != "" {
		query += " " + joins
	}
	if whereClause != "" {
		query += " WHERE " + whereClause
	}
	if orderClause != "" {
		query += " ORDER BY " + orderClause
	}
	if parts.limit != "" {
		query += " LIMIT " + parts.limit
	}
	if parts.offset != "" {
		query += " OFFSET " + parts.offset
	}
	return query, valueList, nil
}

func (QueryWithJSON) preferredSerializer() ResultSerializer {
	return DatabaseJSONSerializer{}
}
