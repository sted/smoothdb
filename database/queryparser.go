package database

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

type Filters = url.Values

type QueryField struct {
	name  string
	label string
	cast  string
}

type WhereConditionNode struct {
	field    string
	operator string
	not      bool
	value    string
	children []*WhereConditionNode
}

type OrderField struct {
	name        string
	descending  bool
	invertNulls bool
}

type QueryParts struct {
	selectFields        []QueryField
	orderFields         []OrderField
	limit               string
	offset              string
	whereConditionsTree WhereConditionNode
}

// QueryStringParser is the interface used to parse the query string and
// extract the query parts, like the WHERE clause.
// Initially we will support the PostgREST mode and later the Django mode.
type QueryStringParser interface {
	Parse(table string, filters Filters) (QueryParts, error)
}

type PostgRestParser struct {
	tokens []string
	cur    int
}

var postgRestReservedWords = []string{
	"select", "order", "limit", "offset", "not", "and", "or",
}

// From https://github.com/PostgREST/postgrest/blob/v9.0.0/src/PostgREST/Query/SqlFragment.hs
var postgRestParserOperators = map[string]string{
	"eq":    "=",
	"gte":   ">=",
	"gt":    ">",
	"lte":   "<=",
	"lt":    "<",
	"neq":   "<>",
	"like":  "LIKE",
	"ilike": "ILIKE",
	"in":    "IN",
	"is":    "IS",
	"cs":    "@>",
	"cd":    "<@",
	"ov":    "&&",
	"sl":    "<<",
	"sr":    ">>",
	"nxr":   "&<",
	"nxl":   "&>",
}

func isBooleanOp(op string) bool {
	if op == "and" || op == "or" || op == "not.and" || op == "not.or" {
		return true
	} else {
		return false
	}
}

// Filter := Select | Order | Limit | Offset | Where
// Select := SelectItem (',' SelectItem)*
// SelectItem := <name> [':' <label>] ["::" <cast>]
// Order := OrderItem (',' OrderItem)*
// OrderItem := <name> ['.' ("asc" | "desc")] ['.' ("nullsfirst" | "nullslast")]
// Cond := CondName | CondBool
// CondName := <name> '.' OpValue
// OpValue :=  ["not" ‘.’] <op> ‘.’ <value>
// CondBool := BoolOp CondList
// BoolOp := ["not" '.'] ("and" | "or")
// CondList := ’(‘ Cond (‘,’ Cond)+ ‘)’

func (p *PostgRestParser) lexWhereCondition(k, v string) {
	p.tokens = nil
	p.cur = 0
	if isBooleanOp(k) {
		ops := strings.Split(k, ".")
		p.tokens = append(p.tokens, ops[0])
		if len(ops) == 2 {
			p.tokens = append(p.tokens, ".", ops[1])
		}
	} else {
		p.tokens = append(p.tokens, k, ".")
	}
	pos := 0

	for {
		n := strings.IndexAny(v[pos:], ".,():")
		if n == -1 {
			p.tokens = append(p.tokens, v[pos:])
			break
		}
		if n != 0 {
			p.tokens = append(p.tokens, v[pos:pos+n])
		}
		p.tokens = append(p.tokens, v[pos+n:pos+n+1]) // valid because our delims are ascii
		pos += n + 1
	}
}

func (p *PostgRestParser) next() string {
	if p.cur == len(p.tokens) {
		return ""
	}
	t := p.tokens[p.cur]
	p.cur++
	return t
}

func (p *PostgRestParser) parseWhereCondition(key, value string, root *WhereConditionNode) error {
	p.lexWhereCondition(key, value)
	return p.cond(root)
}

func (p *PostgRestParser) cond(parent *WhereConditionNode) (err error) {
	node := &WhereConditionNode{}
	token := p.next()
	if token == "not" || token == "and" || token == "or" {
		if token == "not" {
			node.not = true
			if p.next() != "." {
				return fmt.Errorf("'.' expected")
			}
			token = p.next()
		}
		if token != "and" && token != "or" {
			return fmt.Errorf("boolean operator expected")
		}
		node.operator = strings.ToUpper(token)
		if p.next() != "(" {
			return fmt.Errorf("'(' expected")
		}
		err = p.cond(node)
		if err != nil {
			return err
		}
		if p.next() != "," {
			return fmt.Errorf("',' expected")
		}
		for {
			err = p.cond(node)
			if err != nil {
				return err
			}
			token = p.next()
			if token != "," {
				break
			}
		}
		if token != ")" {
			return fmt.Errorf("')' expected")
		}

	} else {
		node.field = token
		if p.next() != "." {
			return fmt.Errorf("'.' expected")
		}
		token = p.next()
		if token == "not" {
			node.not = true
			if p.next() != "." {
				return fmt.Errorf("'.' expected")
			}
			token = p.next()
		}
		op, ok := postgRestParserOperators[token]
		if !ok {
			return fmt.Errorf("valid sql operator expected")
		}
		node.operator = op
		if p.next() != "." {
			return fmt.Errorf("'.' expected")
		}
		token = p.next()
		if token == "" {
			return fmt.Errorf("value expected")
		}
		node.value = token
	}
	parent.children = append(parent.children, node)
	return nil
}

func (p PostgRestParser) Parse(table string, filters Filters) (QueryParts, error) {
	parts := QueryParts{}

	// SELECT
	// select=f1,f2,f3:field3
	if selectFilter, ok := filters["select"]; ok {
		for _, csFields := range selectFilter {
			fields := strings.Split(csFields, ",")
			for _, field := range fields {
				if field == "" {
					// log warning
					continue
				}
				nameLabel, cast, _ := strings.Cut(field, "::")
				name, label, _ := strings.Cut(nameLabel, ":")

				parts.selectFields = append(parts.selectFields,
					QueryField{name: name, label: label, cast: cast})
			}
		}
		delete(filters, "select")
	}
	// ORDER
	// order=f1,f2.asc,f3.desc.nullslast
	if orderFilter, ok := filters["order"]; ok {
		for _, csFields := range orderFilter {
			fields := strings.Split(csFields, ",")
			for _, field := range fields {
				values := strings.SplitN(field, ".", 3)
				if values[0] == "" {
					// log warning
					continue
				}
				// fill to 3 to simplify the next phase
				more := 3 - len(values)
				for i := 0; i < more; i++ {
					values = append(values, "")
				}
				name := values[0]
				descending := false
				invertNulls := false
				if values[1] == "desc" || values[2] == "desc" {
					descending = true
				}
				if descending && (values[1] == "nullslast" || values[2] == "nullslast") {
					invertNulls = true
				} else if !descending && (values[1] == "nullsfirst" || values[2] == "nullsfirst") {
					invertNulls = true
				}
				parts.orderFields = append(parts.orderFields,
					OrderField{name: name, descending: descending, invertNulls: invertNulls})
			}
			delete(filters, "order")
		}
	}
	// LIMIT
	// limit=100
	if limitFilter, ok := filters["limit"]; ok {
		parts.limit = limitFilter[0]
		delete(filters, "limit")
	}
	// OFFSET
	// offset=50
	if offsetFilter, ok := filters["offset"]; ok {
		parts.offset = offsetFilter[0]
		delete(filters, "offset")
	}
	// WHERE
	keys := []string{}
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys) // canonical order -> sorted alphabetically
	for _, k := range keys {
		for _, v := range filters[k] {
			p.parseWhereCondition(k, v, &parts.whereConditionsTree)
		}
	}
	return parts, nil
}
