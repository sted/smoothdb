package database

import (
	"context"
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

type OrderField struct {
	name        string
	descending  bool
	invertNulls bool
}

type WhereConditionNode struct {
	field    string
	operator string
	not      bool
	value    string
	children []*WhereConditionNode
}

type QueryParts struct {
	selectFields        []QueryField
	orderFields         []OrderField
	limit               string
	offset              string
	whereConditionsTree WhereConditionNode
}

type QueryOptions struct {
	ReturnRepresentation bool
	AcceptProfile        string
}

// RequestParser is the interface used to parse the query string and
// extract the significant headers.
// Initially we will support the PostgREST mode and later the Django mode.
type RequestParser interface {
	parseQuery(filters Filters) (QueryParts, error)
	getOptions(ctx context.Context) (QueryOptions, error)
}

type PostgRestParser struct {
	tokens []string
	cur    int
}

type ParseError struct {
	msg string // description of error
}

func (e *ParseError) Error() string { return e.msg }

// var postgRestReservedWords = []string{
// 	"select", "order", "limit", "offset", "not", "and", "or",
// }

// From https://github.com/PostgREST/postgrest/blob/v9.0.0/src/PostgREST/Query/SqlFragment.hs
var postgRestParserOperators = map[string]string{
	"eq":     "=",
	"gte":    ">=",
	"gt":     ">",
	"lte":    "<=",
	"lt":     "<",
	"neq":    "<>",
	"like":   "LIKE",
	"ilike":  "ILIKE",
	"match":  "~",
	"imatch": "~*",
	"in":     "IN",
	"is":     "IS",
	"cs":     "@>",
	"cd":     "<@",
	"ov":     "&&",
	"sl":     "<<",
	"sr":     ">>",
	"nxr":    "&<",
	"nxl":    "&>",
	"adj":    "-|-",
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
//
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
	//pos := 0

	// for {
	// 	n := strings.IndexAny(v[pos:], ".,():")
	// 	if n == -1 {
	// 		p.tokens = append(p.tokens, v[pos:])
	// 		break
	// 	}
	// 	if n != 0 {
	// 		p.tokens = append(p.tokens, v[pos:pos+n])
	// 	}
	// 	p.tokens = append(p.tokens, v[pos+n:pos+n+1]) // valid because our delims are ascii
	// 	pos += n + 1
	// }

	s := 0 // state 0: normal, 1: quoted 2: escaped (backslash in quotes)
	var normal []byte
	var quoted []byte
	var cur byte
	for i := 0; i < len(v); i++ {
		cur = v[i]
		if s == 0 { // normal
			if cur == '"' {
				s = 1
				quoted = nil
			} else if cur == '.' || cur == ',' || cur == '(' || cur == ')' || cur == ':' {
				if len(normal) != 0 {
					p.tokens = append(p.tokens, string(normal))
					normal = nil
				}
				p.tokens = append(p.tokens, string(cur))
			} else if cur == '*' {
				normal = append(normal, '%')
			} else {
				normal = append(normal, cur)
			}
		} else if s == 1 { // quoted
			if cur == '"' {
				s = 0
				p.tokens = append(p.tokens, string(quoted))
			} else if cur == '\\' {
				s = 2
			} else {
				quoted = append(quoted, cur)
			}
		} else if s == 2 { // escaped
			s = 1
			quoted = append(quoted, cur)
		}
	}
	if len(normal) != 0 {
		p.tokens = append(p.tokens, string(normal))
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

// (1,2,3) values
// ("aaa","hi,there")
// {1,2,3} arrays
// [ts1,ts2) ranges [],[),(],()
//
// NOT SUPPORTED FOR NOW: {{1,2},{3,4}} multi-dimensional arrays
func (p *PostgRestParser) getValueString() string {
	var value strings.Builder
	var end func(string) bool

	t := p.next()
	if t == "(" || strings.HasPrefix(t, "[") {
		end = func(t string) bool {
			return t == ")" || strings.HasSuffix(t, "]")
		}
	} else if strings.HasPrefix(t, "{") {
		end = func(t string) bool { return strings.HasSuffix(t, "}") }
	} else {
		return t
	}
	value.WriteString(t)
	if !end(t) {
		for {
			t = p.next()
			if t == "" {
				return "" // error
			}
			value.WriteString(t)
			if end(t) {
				break
			}
		}
	}
	return value.String()
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
				return &ParseError{"'.' expected"}
			}
			token = p.next()
		}
		if token != "and" && token != "or" {
			return &ParseError{"boolean operator expected"}
		}
		node.operator = strings.ToUpper(token)
		if p.next() != "(" {
			return &ParseError{"'(' expected"}
		}
		err = p.cond(node)
		if err != nil {
			return err
		}
		if p.next() != "," {
			return &ParseError{"',' expected"}
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
			return &ParseError{"')' expected"}
		}
	} else {
		node.field = token
		if p.next() != "." {
			return &ParseError{"'.' expected"}
		}
		token = p.next()
		if token == "not" {
			node.not = true
			if p.next() != "." {
				return &ParseError{"'.' expected"}
			}
			token = p.next()
		}
		op, ok := postgRestParserOperators[token]
		if !ok {
			return &ParseError{"valid sql operator expected"}
		}
		node.operator = op
		token = p.next()
		if token == "." { // value
			token = p.getValueString()
			if token == "" {
				return &ParseError{"value expected"}
			}
			ltoken := strings.ToLower(token)
			if ltoken == "null" ||
				ltoken == "true" ||
				ltoken == "false" ||
				ltoken == "unknown" {
				token = ltoken
			} else if node.operator == "IS" {
				return &ParseError{"IS operator requires null, true, false or unknown"}
			}
			node.value = token
		}
	}
	parent.children = append(parent.children, node)
	return nil
}

func (p PostgRestParser) parseQuery(filters Filters) (parts QueryParts, err error) {
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
				labelName, cast, _ := strings.Cut(field, "::")
				label, name, found := strings.Cut(labelName, ":")
				if !found {
					label, name = name, label
				}

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
			err = p.parseWhereCondition(k, v, &parts.whereConditionsTree)
			if err != nil {
				return QueryParts{}, err
			}
		}
	}
	return parts, nil
}

func (p PostgRestParser) getOptions(ctx context.Context) (QueryOptions, error) {
	header := GetHeader(ctx)
	options := QueryOptions{}
	if header.Get("Prefer") == "return=representation" {
		options.ReturnRepresentation = true
	}
	if ap := header.Get("Accept-Profile"); ap != "" {
		options.AcceptProfile = ap
	}
	return options, nil
}
