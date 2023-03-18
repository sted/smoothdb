package database

import (
	"strconv"
	"strings"

	"github.com/samber/lo"
)

type QueryBuilder interface {
	BuildInsert(table string, records []Record, parts *QueryParts, options *QueryOptions, info *DbInfo) (string, []any, error)
	BuildUpdate(table string, record Record, parts *QueryParts, options *QueryOptions, info *DbInfo) (string, []any, error)
	BuildDelete(table string, parts *QueryParts, options *QueryOptions, info *DbInfo) (string, error)
	BuildSelect(table string, parts *QueryParts, options *QueryOptions, info *DbInfo) (string, error)

	preferredSerializer() ResultSerializer
}

type Join struct {
	fields   string
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

func selectForJoinClause(join Join, parts *QueryParts, afterWithClause bool) (sel string) {
	rel := join.rel
	sel = " SELECT " + join.fields
	sel += " FROM " + quoteParts(rel.RelatedTable)
	if rel.JunctionTable != "" {
		sel += ", " + quoteParts(rel.JunctionTable)
	}
	sel += " WHERE "
	if rel.JunctionTable == "" {
		for i := range rel.Columns {
			if i != 0 {
				sel += " AND "
			}
			sel += quoteParts(rel.RelatedTable) + "." + quote(rel.RelatedColumns[i])
			sel += " = "
			if !afterWithClause {
				sel += quoteParts(rel.Table)
			} else {
				sel += quote("_source")
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
			sel += quoteParts(rel.Table) + "." + quote(rel.Columns[i])
		}

		for i := range rel.JRelatedColumns {
			sel += " AND "
			sel += quoteParts(rel.JunctionTable) + "." + quote(rel.JRelatedColumns[i])
			sel += " = "
			sel += quoteParts(rel.RelatedTable) + "." + quote(rel.RelatedColumns[i])
		}
	}
	// where and order clause for the internal select: the expressions related to
	// the external query are skipped inside the functions.
	// If the internal table is equal to the external one we avoid repeating
	// the expressions.
	if rel.Table != rel.RelatedTable {
		schema, table := splitTableName(rel.RelatedTable)
		whereClause := whereClause(table, schema, parts.whereConditionsTree, join.relLabel)
		if whereClause != "" {
			sel += " AND " + whereClause
		}
		orderClause := orderClause(table, schema, parts.orderFields)
		if orderClause != "" {
			sel += " ORDER BY " + orderClause
		}
	}
	return
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

func selectClause(table, schema string, parts *QueryParts, info *DbInfo, afterWithClause bool) (
	selectClause string, joins string, keys []string, err error) {

	joinMap := map[string]Join{}

	for i, sfield := range parts.selectFields {
		if i != 0 {
			selectClause += ", "
		}
		if sfield.relation != nil {
			relation := sfield.relation.name
			frel, err := findRelationship(table, relation, schema, info)
			if err != nil {
				return "", "", nil, err
			}
			_, relatedTable := splitTableName(frel.RelatedTable)
			fieldPart := prepareField(relatedTable, schema, sfield)
			var labelRelName string
			if sfield.relation.label == "" {
				labelRelName = relatedTable
			} else {
				labelRelName = sfield.relation.label
			}
			relName := table + "_" + labelRelName
			if join, exists := joinMap[relName]; exists {
				join.fields += ", " + fieldPart
				joinMap[relName] = join
				// no comma required, take it back (hack but perhaps more readable)
				selectClause = strings.TrimSuffix(selectClause, ", ")
			} else {
				joinMap[relName] = Join{fieldPart, frel, sfield.relation.label}
				switch frel.Type {
				case M2O, O2O:
					selectClause += " row_to_json(\"" + relName + "\".*) AS " + quote(labelRelName)
				case O2M, M2M:
					selectClause += " COALESCE(\"" + relName + "\".\"_" + relName + "\", '[]') AS " + quote(labelRelName)
				}
			}
		} else {
			var fieldPart string
			if !afterWithClause {
				fieldPart = prepareField(table, schema, sfield)
			} else {
				fieldPart = prepareField("_source", "", sfield)
			}
			selectClause += fieldPart
		}
	}
	if selectClause == "" {
		selectClause = "*"
	}
	if len(joinMap) > 0 {
		for relName, join := range joinMap {
			selectForJoin := selectForJoinClause(join, parts, afterWithClause)
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
	return
}

func orderClause(table, schema string, orderFields []OrderField) string {
	var order string
	for _, o := range orderFields {
		if o.field.tablename != table {
			// skip where filters for other tables
			continue
		}
		if order != "" {
			order += ", "
		}
		order += _stq(o.field.name, schema, table) + o.field.jsonPath
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

func appendValue(where, value string) string {
	if value == "null" ||
		value == "true" ||
		value == "false" ||
		value == "unknown" {
		where += value
	} else {
		where += "'" + value + "'"
	}
	return where
}

func whereClause(table, schema string, node *WhereConditionNode, label string) string {
	var where string
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
		var children string
		for _, n := range node.children {
			if n.field.tablename != table &&
				n.field.tablename != label {
				// skip where filters for other tables
				continue
			}
			if children != "" {
				children += bool_op
			}
			children += whereClause(table, schema, n, label)
		}
		where += children
		if node.not || node.operator == "OR" {
			where += ")"
		}
	} else {
		if node.not {
			where += "NOT "
		}
		where += _stq(node.field.name, schema, table)
		where += node.field.jsonPath
		where += " " + node.operator + " "
		if node.operator == "IN" {
			where += "("
			for i, value := range node.values {
				if i != 0 {
					where += ", "
				}
				where = appendValue(where, value)
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
			where = appendValue(where, node.values[0])
			where += ")"

		} else {
			where = appendValue(where, node.values[0])
		}
	}
	return where
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
			sc, joins, keys, _ := selectClause(table, schema, parts, info, true)
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
	whereClause := whereClause(table, schema, parts.whereConditionsTree, "")
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

func (CommonBuilder) BuildDelete(table string, parts *QueryParts, options *QueryOptions, info *DbInfo) (string, error) {
	schema := options.Schema
	whereClause := whereClause(table, schema, parts.whereConditionsTree, "")
	delete := "DELETE FROM " + _sq(table, schema)
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
	return delete, nil
}

type DirectQueryBuilder struct {
	CommonBuilder
}

func (DirectQueryBuilder) BuildSelect(table string, parts *QueryParts, options *QueryOptions, info *DbInfo) (string, error) {
	schema := options.Schema
	selectClause, joins, _, err := selectClause(table, schema, parts, info, false)
	if err != nil {
		return "", err
	}
	whereClause := whereClause(table, schema, parts.whereConditionsTree, "")
	orderClause := orderClause(table, schema, parts.orderFields)
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
	return query, nil
}

func (DirectQueryBuilder) preferredSerializer() ResultSerializer {
	return DirectJSONSerializer{}
}

type QueryWithJSON struct {
	CommonBuilder
}

func (QueryWithJSON) BuildSelect(table string, parts *QueryParts, options *QueryOptions, info *DbInfo) (string, error) {
	schema := options.Schema
	selectClause, joins, _, err := selectClause(table, schema, parts, info, false)
	if err != nil {
		return "", err
	}
	whereClause := whereClause(table, schema, parts.whereConditionsTree, "")
	orderClause := orderClause(table, schema, parts.orderFields)
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
	return query, nil
}

func (QueryWithJSON) preferredSerializer() ResultSerializer {
	return DatabaseJSONSerializer{}
}
