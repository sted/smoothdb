package database

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type Filters = url.Values
type Request = http.Request

type Field struct {
	name      string
	tablename string
	jsonPath  string
	last      string
}

type SelectField struct {
	field Field
	label string
	cast  string
	table *SelectedTable
}

type SelectedTable struct {
	name   string
	label  string
	parent string
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

func (node *WhereConditionNode) isRootNode() bool {
	if node.operator == "" && node.field.name == "" {
		return true
	} else {
		return false
	}
}

type QueryParts struct {
	selectFields        []SelectField
	orderFields         []OrderField
	limit               string
	offset              string
	whereConditionsTree *WhereConditionNode
}

type QueryOptions struct {
	ReturnRepresentation bool
	Schema               string
}

// RequestParser is the interface used to parse the query string and
// extract the significant headers.
// Initially we will support PostgREST mode and later (perhaps) Django mode.
type RequestParser interface {
	parseQuery(mainTable string, filters Filters) (*QueryParts, error)
	getOptions(req *Request) *QueryOptions
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
// longSep is the set of multi char separators (put longest first)
func (p *PostgRestParser) scan(s string, sep string, longSep ...string) {
	state := 0 // state 0: normal, 1: quoted 2: escaped (backslash in quotes)
	var normal []byte
	var quoted []byte
	var cur byte
	wasSep := true
outer:
	for i := 0; i < len(s); i++ {
		cur = s[i]
		if state == 0 { // normal
			if wasSep && cur == ' ' {
				continue
			}
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
					wasSep = true
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
				wasSep = false
			} else if strings.Contains(sep, string(cur)) {
				if len(normal) != 0 {
					p.tokens = append(p.tokens, string(normal))
					normal = nil
				}
				p.tokens = append(p.tokens, string(cur))
				wasSep = true
			} else {
				normal = append(normal, cur)
				wasSep = false
			}
		} else if state == 1 { // quoted
			if cur == '"' {
				state = 0
				p.tokens = append(p.tokens, string(quoted))
				wasSep = true
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

func (p *PostgRestParser) field(mayHaveTable bool) (f Field, err error) {
	token := p.next()
	if token == "" {
		return f, &ParseError{"field expected"}
	}
	if mayHaveTable && p.lookAhead() == "." {
		// table name
		f.tablename = token
		p.next()
		token = p.next()
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
	return p.selectList(nil)
}

func (p *PostgRestParser) selectList(table *SelectedTable) (selectFields []SelectField, err error) {
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

func (p *PostgRestParser) selectItem(table *SelectedTable) (selectFields []SelectField, err error) {
	var label, cast string
	token := p.next()
	if p.lookAhead() == ":" {
		label = token
		p.next()
	} else {
		p.back()
	}
	field, err := p.field(false)
	if err != nil {
		return nil, err
	}
	if table != nil && table.name != "" {
		// for uniformity but for selects is also in QueryTable
		field.tablename = table.name
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
			selectFields = append(selectFields, SelectField{field, label, cast, table})
		} else {
			p.back()
		}
	} else {
		// table
		p.next()
		if cast != "" {
			return nil, &ParseError{"table cannot have cast"}
		}
		table = &SelectedTable{name: field.name, label: label}
		fields, err := p.selectList(table)
		if err != nil {
			return nil, err
		}
		selectFields = append(selectFields, fields...)
	}
	return selectFields, nil
}

// ORDER
func (p *PostgRestParser) parseOrderCondition(table, o string) (fields []OrderField, err error) {
	var value1, value2 string
	p.reset()
	p.scan(o, ".,", "->>", "->")

	token := p.lookAhead()
	if token == "" {
		return nil, &ParseError{"order fields expected"}
	}
	for {
		field, err := p.field(false)
		if err != nil {
			return nil, err
		}
		field.tablename = table
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
		v = k + "=" + v
	}
	p.scan(v, "=.,()[]{}:", "->>", "->")
}

func (p *PostgRestParser) parseWhereCondition(mainTable, key, value string, root *WhereConditionNode) error {
	p.scanWhereCondition(key, value)
	return p.cond(mainTable, root)
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

func (p *PostgRestParser) cond(mainTable string, parent *WhereConditionNode) (err error) {
	node := &WhereConditionNode{}
	token := p.lookAhead()
	if token == "not" || token == "and" || token == "or" {
		node.field.tablename = mainTable // @@ temp: must manage field.or etc
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
		err = p.cond(mainTable, node)
		if err != nil {
			return err
		}
		token = p.next()
		for token == "," {
			err = p.cond(mainTable, node)
			if err != nil {
				return err
			}
			token = p.next()
		}
		if token != ")" {
			return &ParseError{"')' expected"}
		}
	} else {
		var mayHaveTable bool
		var nextSep string
		if parent.isRootNode() {
			mayHaveTable = true
			nextSep = "="
		} else {
			mayHaveTable = false
			nextSep = "."
		}
		node.field, err = p.field(mayHaveTable)
		if err != nil {
			return err
		}
		if node.field.tablename == "" {
			node.field.tablename = mainTable
		}
		if p.next() != nextSep {
			return &ParseError{"'=' expected"}
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
func (p PostgRestParser) parseQuery(mainTable string, filters Filters) (parts *QueryParts, err error) {
	parts = &QueryParts{}

	// SELECT
	var sel string
	if selectFilter, ok := filters["select"]; ok {
		for i, csFields := range selectFilter {
			if i != 0 {
				sel += ","
			}
			sel += csFields
		}
		delete(filters, "select")
		parts.selectFields, err = p.parseSelect(sel)
		if err != nil {
			return nil, err
		}
	}

	// ORDER
	// order=f1,f2.asc,f3.desc.nullslast
	for k, v := range filters {
		var order, table string
		if k == "order" {
			table = mainTable
		} else if strings.HasSuffix(k, ".order") {
			table = strings.TrimSuffix(k, ".order")
		} else {
			continue
		}
		for i, oFields := range v {
			if i != 0 {
				order += ","
			}
			order += oFields
		}
		fields, err := p.parseOrderCondition(table, order)
		if err != nil {
			return nil, err
		}
		parts.orderFields = append(parts.orderFields, fields...)
		delete(filters, k)

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
	parts.whereConditionsTree = &WhereConditionNode{}
	for _, k := range keys {
		for _, v := range filters[k] {
			err = p.parseWhereCondition(mainTable, k, v, parts.whereConditionsTree)
			if err != nil {
				return nil, err
			}
		}
	}
	return parts, nil
}

func (p PostgRestParser) getOptions(req *Request) *QueryOptions {
	header := req.Header
	options := &QueryOptions{}
	if header.Get("Prefer") == "return=representation" {
		options.ReturnRepresentation = true
	}
	var schemaProfile string
	switch req.Method {
	case "GET", "HEAD":
		schemaProfile = "Accept-Profile"
	case "POST", "PUT", "PATCH", "DELETE":
		schemaProfile = "Content-Profile"
	}
	if ap := header.Get(schemaProfile); ap != "" {
		options.Schema = ap
	}
	return options
}
