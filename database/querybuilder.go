package database

import (
	"strconv"
	"strings"
)

type QueryBuilder interface {
	BuildInsert(table string, records []Record, parts *QueryParts, options *QueryOptions) (string, []any, error)
	BuildUpdate(table string, record Record, parts *QueryParts, options *QueryOptions) (string, []any, error)
	BuildDelete(table string, parts *QueryParts, options *QueryOptions) (string, error)
	BuildSelect(table string, parts *QueryParts, options *QueryOptions, rels []Relationship) (string, error)
	preferredSerializer() ResultSerializer
}

type Join struct {
	fields string
	rel    Relationship
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

func selectForJoinClause(join Join, parts *QueryParts) (sel string) {
	rel := join.rel
	sel = " SELECT " + join.fields
	sel += " FROM " + rel.RelatedTable
	sel += " WHERE "
	for i := range rel.Columns {
		if i != 0 {
			sel += " AND "
		}
		sel += quoteParts(rel.RelatedTable) + "." + quoteParts(rel.RelatedColumns[i])
		sel += " = "
		sel += quoteParts(rel.Table) + "." + quoteParts(rel.Columns[i])
	}
	schema, table := splitTableName(rel.RelatedTable)
	whereClause := whereClause(table, schema, parts.whereConditionsTree)
	if whereClause != "" {
		sel += " AND " + whereClause
	}
	orderClause := orderClause(table, schema, parts.orderFields)
	if orderClause != "" {
		sel += " ORDER BY " + orderClause
	}
	return
}

func selectClause(table, schema string, parts *QueryParts, rels []Relationship) (selectClause string, joins string, err error) {
	joinMap := map[string]Join{}

	for i, sfield := range parts.selectFields {
		if i != 0 {
			selectClause += ", "
		}
		if sfield.table != nil {
			relatedTable := sfield.table.name
			fieldPart := prepareField(relatedTable, schema, sfield)
			frels := filterRelationships(rels, _s(relatedTable, schema))
			if len(frels) != 1 {
				return "", "", &BuildError{"cannot find relationship for table " + relatedTable}
			}
			frel := frels[0]
			relName := table + "_" + relatedTable
			if join, exists := joinMap[relName]; exists {
				join.fields += ", " + fieldPart
				joinMap[relName] = join
				// no comma required, take it back (hack but perhaps more readable)
				selectClause = strings.TrimSuffix(selectClause, ", ")
			} else {
				joinMap[relName] = Join{fieldPart, frel}
				if sfield.table.label != "" {
					relatedTable = sfield.table.label
				}
				switch frel.Type {
				case M2O:
					selectClause += " row_to_json(\"" + relName + "\".*) AS " + quote(relatedTable)
				case O2M:
					selectClause += " COALESCE(\"" + relName + "\".\"_" + relName + "\", '[]') AS " + quote(relatedTable)
				}
			}
		} else {
			fieldPart := prepareField(table, schema, sfield)
			selectClause += fieldPart
		}
	}
	if selectClause == "" {
		selectClause = "*"
	}
	if len(joinMap) > 0 {
		for relName, join := range joinMap {
			selectForJoin := selectForJoinClause(join, parts)
			joins += " LEFT JOIN LATERAL ("
			switch join.rel.Type {
			case M2O:
				joins += selectForJoin
			case O2M:
				joins += " SELECT json_agg(\"_" + relName + "\") AS \"_" + relName + "\""
				joins += " FROM ("
				joins += selectForJoin
				joins += " ) AS \"_" + relName + "\""
			}
			joins += ") AS \"" + relName + "\" ON TRUE"
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

func whereClause(table, schema string, node *WhereConditionNode) string {
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
			if n.field.tablename != table {
				// skip where filters for other tables
				continue
			}
			if children != "" {
				children += bool_op
			}
			children += whereClause(table, schema, n)
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

func returningClause(table, schema string, selectFields []SelectField) (ret string) {
	ret += " RETURNING "
	if len(selectFields) == 0 {
		ret += "*"
	} else {
		for i, sfield := range selectFields {
			if i != 0 {
				ret += ", "
			}
			ret += prepareField(table, schema, sfield)
		}
	}
	return
}

type CommonBuilder struct{}

func (CommonBuilder) BuildInsert(table string, records []Record, parts *QueryParts, options *QueryOptions) (insert string, valueList []any, err error) {
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
		fields += key
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
	if options.ReturnRepresentation {
		insert += returningClause(table, schema, parts.selectFields)
	}
	return insert, valueList, nil
}

func (CommonBuilder) BuildUpdate(table string, record Record, parts *QueryParts, options *QueryOptions) (update string, valueList []any, err error) {
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
		pairs += key
		i++
		pairs += " = $" + strconv.Itoa(i)
		valueList = append(valueList, record[key])
	}
	schema := options.Schema
	whereClause := whereClause(table, schema, parts.whereConditionsTree)
	update = "UPDATE " + _sq(table, schema) + " SET " + pairs
	if whereClause != "" {
		update += " WHERE " + whereClause
	}
	if options.ReturnRepresentation {
		update += returningClause(table, schema, parts.selectFields)
	}
	return update, valueList, nil
}

func (CommonBuilder) BuildDelete(table string, parts *QueryParts, options *QueryOptions) (string, error) {
	schema := options.Schema
	whereClause := whereClause(table, schema, parts.whereConditionsTree)
	delete := "DELETE FROM " + _sq(table, schema)
	if whereClause != "" {
		delete += " WHERE " + whereClause
	}
	if options.ReturnRepresentation {
		delete += " RETURNING *"
	}
	return delete, nil
}

type DirectQueryBuilder struct {
	CommonBuilder
}

func (DirectQueryBuilder) BuildSelect(table string, parts *QueryParts, options *QueryOptions, rels []Relationship) (string, error) {
	schema := options.Schema
	selectClause, joins, err := selectClause(table, schema, parts, rels)
	if err != nil {
		return "", err
	}
	whereClause := whereClause(table, schema, parts.whereConditionsTree)
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

func (QueryWithJSON) BuildSelect(table string, parts *QueryParts, options *QueryOptions, rels []Relationship) (string, error) {
	schema := options.Schema
	selectClause, joins, err := selectClause(table, schema, parts, rels)
	if err != nil {
		return "", err
	}
	whereClause := whereClause(table, schema, parts.whereConditionsTree)
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
