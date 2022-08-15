package database

import (
	"fmt"
	"strconv"
)

type QueryBuilder interface {
	BuildInsert(table string, records []Record, options QueryOptions) (string, []any, error)
	BuildUpdate(table string, record Record, parts QueryParts, options QueryOptions) (string, []any, error)
	BuildDelete(table string, parts QueryParts, options QueryOptions) (string, error)
	BuildSelect(table string, parts QueryParts, options QueryOptions) (string, error)
	preferredSerializer() ResultSerializer
}

func selectClause(queryFields []QueryField) string {
	selectClause := ""
	for i, field := range queryFields {
		if i != 0 {
			selectClause += ", "
		}
		selectClause += "\"" + field.name + "\""
		if field.cast != "" {
			selectClause += "::" + field.cast
		}
		if field.label != "" {
			selectClause += " AS \"" + field.label + "\""
		}
	}
	if selectClause == "" {
		selectClause = "*"
	}
	return selectClause
}

func orderClause(orderFields []OrderField) string {
	order := ""
	for i, field := range orderFields {
		if i != 0 {
			order += ", "
		}
		order += field.name
		if field.descending {
			order += " DESC"
		}
		if field.invertNulls {
			if field.descending {
				order += " NULLS LAST"
			} else {
				order += " NULLS FIRST"
			}
		}
	}
	return order
}

func whereClause(node *WhereConditionNode) string {
	where := ""
	if node.operator == "" || node.field == "" {
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
		where += node.field + " " + node.operator + " "
		if node.operator == "IN" ||
			node.value == "null" ||
			node.value == "true" ||
			node.value == "false" ||
			node.value == "unknown" {
			where += node.value
		} else {
			where += "'" + node.value + "'"
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
	query := "SELECT " + selectClause + " FROM " + table
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
