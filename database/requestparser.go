package database

import (
	"context"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type Filters = url.Values

type Field struct {
	name     string
	jsonPath string
	last     string
}

type SelectField struct {
	field Field
	label string
	cast  string
	table QueryTable
}

type QueryTable struct {
	table       string
	tableCast   string
	parentTable string
}

type OrderField struct {
	field       Field
	descending  bool
	invertNulls bool
}

type WhereConditionNode struct {
	field    Field
	operator string
	opSource string
	opArgs   []string
	not      bool
	values   []string
	children []*WhereConditionNode
}

type QueryParts struct {
	selectFields        []SelectField
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
	"fts":    "@@",
	"plfts":  "@@",
	"phfts":  "@@",
	"wfts":   "@@",
}

// scan splits the string s using the separators and
// skipping double quoted strings.
// Returns a slice of substrings and separators.
// sep is the set of single char separators.
// longSep is the set of multi char separators (longest first)
func (p *PostgRestParser) scan(s string, sep string, longSep ...string) {
	state := 0 // state 0: normal, 1: quoted 2: escaped (backslash in quotes)
	var normal []byte
	var quoted []byte
	var cur byte
outer:
	for i := 0; i < len(s); i++ {
		cur = s[i]
		if state == 0 { // normal
			// Manage long separators
			for _, lsep := range longSep {
				l := len(lsep)
				if i+l > len(s) {
					continue
				}
				if strings.Compare(lsep, s[i:i+l]) == 0 {
					if len(normal) != 0 {
						p.tokens = append(p.tokens, string(normal))
						normal = nil
					}
					p.tokens = append(p.tokens, lsep)
					i += l - 1
					continue outer
				}
			}
			if cur == '"' {
				if len(normal) != 0 {
					p.tokens = append(p.tokens, string(normal))
					normal = nil
				}
				state = 1
				quoted = nil
			} else if strings.Contains(sep, string(cur)) {
				if len(normal) != 0 {
					p.tokens = append(p.tokens, string(normal))
					normal = nil
				}
				p.tokens = append(p.tokens, string(cur))
			} else {
				normal = append(normal, cur)
			}
		} else if state == 1 { // quoted
			if cur == '"' {
				state = 0
				p.tokens = append(p.tokens, string(quoted))
			} else if cur == '\\' {
				state = 2
			} else {
				quoted = append(quoted, cur)
			}
		} else if state == 2 { // escaped
			state = 1
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

func (p *PostgRestParser) back() {
	if p.cur != 0 {
		p.cur--
	}
}

func (p *PostgRestParser) lookAhead() string {
	if p.cur == len(p.tokens) {
		return ""
	}
	return p.tokens[p.cur]
}

func (p *PostgRestParser) reset() {
	p.tokens = nil
	p.cur = 0
}

// General grammar:
//
// Filter := Select | Order | Limit | Offset | Where
//
// Field := <name> (( '->' | '->>' ) <member>)*
//
// Select := SelectList
// SelectList := SelectItem (',' SelectItem)*
// SelectItem := SelectField | SelectTable '(' SelectList ')'
// SelectField := [<label> ':'] Field ["::" <cast>]
// SelectTable := [<label> ':'] Field
//
// Order := OrderItem (',' OrderItem)*
// OrderItem := Field ['.' ("asc" | "desc")] ['.' ("nullsfirst" | "nullslast")]
//
// Cond := CondName | CondBool
// CondName := Field '.' OpValue
// OpValue :=  ["not" ‘.’] <op> ‘.’ Values
// CondBool := BoolOp CondList
// BoolOp := ["not" '.'] ("and" | "or")
// CondList := ’(‘ Cond (‘,’ Cond)+ ‘)’
// Values := Value | ValueList
// ValueList := '(' Value (',' Value)* ')'

func (p *PostgRestParser) field() (f Field, err error) {
	token := p.next()
	if token == "" {
		return f, &ParseError{"field expected"}
	}
	f.name = token
	token = p.lookAhead()
	for token == "->>" || token == "->" {
		f.jsonPath += token
		p.next()
		token = p.next()
		if token == "" {
			return f, &ParseError{"json path member expected"}
		}
		if _, err = strconv.Atoi(token); err == nil {
			f.jsonPath += token
		} else {
			f.jsonPath += "'" + token + "'"
			f.last = token
		}
		token = p.lookAhead()
	}
	if f.jsonPath != "" && f.last == "" {
		f.last = f.name
	}
	return f, nil
}

// SELECT
func (p *PostgRestParser) parseSelect(s string) ([]SelectField, error) {
	p.scan(s, ".,():", "->>", "->", "::")
	return p.selectList(&QueryTable{})
}

func (p *PostgRestParser) selectList(table *QueryTable) (selectFields []SelectField, err error) {
	selectFields, err = p.selectItem(table)
	if err != nil {
		return nil, err
	}
	for p.next() == "," {
		fields, err := p.selectItem(table)
		if err != nil {
			return nil, err
		}
		selectFields = append(selectFields, fields...)
	}
	return selectFields, nil
}

func (p *PostgRestParser) selectItem(table *QueryTable) (selectFields []SelectField, err error) {
	var label, cast string
	token := p.next()
	if p.lookAhead() == ":" {
		label = token
		p.next()
	} else {
		p.back()
	}
	field, err := p.field()
	if err != nil {
		return nil, err
	}
	if label == "" {
		label = field.last
	}
	token = p.lookAhead()
	if token == "::" {
		p.next()
		cast = p.next()
		token = p.lookAhead()
	}
	if token != "(" {
		// field
		if field.name != "," {
			selectFields = append(selectFields, SelectField{field, label, cast, *table})
		} else {
			p.back()
		}
	} else {
		// table
		if cast != "" {
			return nil, &ParseError{"table cannot have cast"}
		}
		fields, err := p.selectList(table)
		if err != nil {
			return nil, err
		}
		selectFields = append(selectFields, fields...)
	}
	return selectFields, nil
}

// ORDER
func (p *PostgRestParser) parseOrderCondition(o string) (fields []OrderField, err error) {
	var value1, value2 string
	p.reset()
	p.scan(o, ".,", "->>", "->")

	token := p.lookAhead()
	if token == "" {
		return nil, &ParseError{"order fields expected"}
	}
	for {
		field, err := p.field()
		if err != nil {
			return nil, err
		}
		if p.lookAhead() == "." {
			p.next()
			value1 = p.next()
		}
		if p.lookAhead() == "." {
			p.next()
			value2 = p.next()
		}
		descending := false
		invertNulls := false
		if value1 == "desc" || value2 == "desc" {
			descending = true
		}
		if descending && (value1 == "nullslast" || value2 == "nullslast") ||
			!descending && (value1 == "nullsfirst" || value2 == "nullsfirst") {
			invertNulls = true
		}
		fields = append(fields,
			OrderField{field: field, descending: descending, invertNulls: invertNulls})
		if p.lookAhead() != "," {
			break
		}
		p.next()

	}
	return fields, nil
}

// WHERE
func isBooleanOp(op string) bool {
	if op == "and" || op == "or" || op == "not.and" || op == "not.or" {
		return true
	} else {
		return false
	}
}

func (p *PostgRestParser) scanWhereCondition(k, v string) {
	p.reset()
	if isBooleanOp(k) {
		v = k + v
	} else {
		v = k + "." + v
	}
	p.scan(v, ".,()[]{}:", "->>", "->")
}

func (p *PostgRestParser) parseWhereCondition(key, value string, root *WhereConditionNode) error {
	p.scanWhereCondition(key, value)
	return p.cond(root)
}

func (p *PostgRestParser) value(node *WhereConditionNode) error {
	token := p.next()
	if token == "" {
		return &ParseError{"value expected"}
	}
	value := token
	level := 0
	if token == "(" || token == "[" { // Range or Composite
		level = 1
		for level > 0 {
			token := p.next()
			if token == "(" || token == "[" {
				level++
			} else if token == ")" || token == "]" {
				level--
			} else if token == "" {
				return &ParseError{"')' or ']' expected"}
			}
			value += token
		}
	} else if token == "{" { // Arrays or JSON Object
		level = 1
		for level > 0 {
			token := p.next()
			if token == "{" {
				level++
			} else if token == "}" {
				level--
			} else if token == "" {
				return &ParseError{"'}' expected"}
			} else if unicode.IsLetter(rune(token[0])) {
				token = "\"" + token + "\""
			}
			value += token
		}
	} else {
		lvalue := strings.ToLower(value)
		if lvalue == "null" ||
			lvalue == "true" ||
			lvalue == "false" ||
			lvalue == "unknown" {
			value = lvalue
		} else if node.operator == "IS" {
			return &ParseError{"IS operator requires null, true, false or unknown"}
		}
	}
	value = strings.ReplaceAll(value, "*", "%")
	node.values = append(node.values, value)
	return nil
}

func (p *PostgRestParser) cond(parent *WhereConditionNode) (err error) {
	node := &WhereConditionNode{}
	token := p.lookAhead()
	if token == "not" || token == "and" || token == "or" {
		p.next()
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
		token = p.next()
		for token == "," {
			err = p.cond(node)
			if err != nil {
				return err
			}
			token = p.next()
		}
		if token != ")" {
			return &ParseError{"')' expected"}
		}
	} else {
		node.field, err = p.field()
		if err != nil {
			return err
		}
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
		node.opSource = token
		token = p.next()
		if op == "@@" {
			if token == "(" {
				for {
					node.opArgs = append(node.opArgs, p.next())
					token = p.next()
					if token != "," {
						break
					}
				}
				if token != ")" {
					return &ParseError{"')' expected"}
				}
				token = p.next()
			}
		}
		if token == "." { // value
			if node.operator == "IN" {
				if p.next() != "(" {
					return &ParseError{"'(' expected"}
				}
				for {
					err = p.value(node)
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
				err = p.value(node)
				if err != nil {
					return err
				}
			}
		}
	}
	parent.children = append(parent.children, node)
	return nil
}

// QUERY
func (p PostgRestParser) parseQuery(filters Filters) (parts QueryParts, err error) {
	// SELECT
	var sel string
	if selectFilter, ok := filters["select"]; ok {
		for i, csFields := range selectFilter {
			if i != 0 {
				sel += ", "
			}
			sel += csFields
		}
		delete(filters, "select")
		parts.selectFields, err = p.parseSelect(sel)
		if err != nil {
			return QueryParts{}, err
		}
	}

	// ORDER
	// order=f1,f2.asc,f3.desc.nullslast
	var order string
	if orderFilter, ok := filters["order"]; ok {
		for i, oFields := range orderFilter {
			if i != 0 {
				order += ", "
			}
			order += oFields
		}
		parts.orderFields, err = p.parseOrderCondition(order)
		if err != nil {
			return QueryParts{}, err
		}
		delete(filters, "order")
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
