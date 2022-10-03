package database

import (
	"fmt"
	"strconv"
	"strings"
)

type QueryBuilder interface {
	BuildInsert(table string, records []Record, options QueryOptions) (string, []any, error)
	BuildUpdate(table string, record Record, parts QueryParts, options QueryOptions) (string, []any, error)
	BuildDelete(table string, parts QueryParts, options QueryOptions) (string, error)
	BuildSelect(table string, parts QueryParts, options QueryOptions) (string, error)
	preferredSerializer() ResultSerializer
}

func selectClause(selectFields []SelectField) string {
	selectClause := ""
	for i, sfield := range selectFields {
		if i != 0 {
			selectClause += ", "
		}
		if sfield.field.jsonPath != "" {
			selectClause += "(" + sfield.field.name + sfield.field.jsonPath + ")"
		} else {
			selectClause += "\"" + sfield.field.name + "\""
		}
		if sfield.cast != "" {
			selectClause += "::" + sfield.cast
		}
		if sfield.label != "" {
			selectClause += " AS \"" + sfield.label + "\""
		}
	}
	if selectClause == "" {
		selectClause = "*"
	}
	return selectClause
}

func orderClause(orderFields []OrderField) string {
	order := ""
	for i, ofield := range orderFields {
		if i != 0 {
			order += ", "
		}
		order += "\"" + ofield.field.name + "\"" + ofield.field.jsonPath
		if ofield.descending {
			order += " DESC"
		}
		if ofield.invertNulls {
			if ofield.descending {
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

func whereClause(node *WhereConditionNode) string {
	where := ""
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
		for i, n := range node.children {
			if i != 0 {
				where += bool_op
			}
			where += whereClause(n)
		}
		if node.not || node.operator == "OR" {
			where += ")"
		}
	} else {
		if node.not {
			where += "NOT "
		}
		if strings.HasPrefix(node.field.name, "\"") {
			where += node.field.name
		} else {
			where += "\"" + node.field.name + "\""
		}
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

type CommonBuilder struct{}

func (CommonBuilder) BuildInsert(table string, records []Record, options QueryOptions) (insert string, valueList []any, err error) {
	var fields string
	var fieldList []string
	var values string

	if len(records) == 0 {
		return "", nil, fmt.Errorf("no records to insert")
	}
	n := len(records[0])
	for key := range records[0] {
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
	insert = "INSERT INTO " + table + " (" + fields + ") VALUES (" + values + ")"
	if options.ReturnRepresentation {
		insert += " RETURNING *"
	}
	return insert, valueList, nil
}

func (CommonBuilder) BuildUpdate(table string, record Record, parts QueryParts, options QueryOptions) (update string, valueList []any, err error) {
	var pairs string

	var i int
	for key := range record {
		if pairs != "" {
			pairs += ", "
		}
		pairs += key
		i++
		pairs += " = $" + strconv.Itoa(i)
		valueList = append(valueList, record[key])
	}
	whereClause := whereClause(&parts.whereConditionsTree)
	update = "UPDATE " + table + " SET " + pairs
	if whereClause != "" {
		update += " WHERE " + whereClause
	}
	if options.ReturnRepresentation {
		update += " RETURNING *"
	}
	return update, valueList, nil
}

func (CommonBuilder) BuildDelete(table string, parts QueryParts, options QueryOptions) (string, error) {
	whereClause := whereClause(&parts.whereConditionsTree)
	delete := "DELETE FROM " + table
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

func (DirectQueryBuilder) BuildSelect(table string, parts QueryParts, options QueryOptions) (string, error) {
	selectClause := selectClause(parts.selectFields)
	orderClause := orderClause(parts.orderFields)
	whereClause := whereClause(&parts.whereConditionsTree)
	query := "SELECT " + selectClause + " FROM \"" + table + "\""
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

func (QueryWithJSON) BuildSelect(table string, parts QueryParts, options QueryOptions) (string, error) {
	selectClause := selectClause(parts.selectFields)
	orderClause := orderClause(parts.orderFields)
	whereClause := whereClause(&parts.whereConditionsTree)
	query := "SELECT "
	if selectClause == "*" {
		query += "json_agg(" + table + ")" + " FROM " + table
	} else {
		query += "SELECT " + selectClause + " FROM " + table
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
