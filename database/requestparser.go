package database

import (
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type Filters = url.Values
type Request = http.Request

// Field represents a column and its attributes in the different parts of a query: select, where, order, etc clauses
type Field struct {
	name      string   // field name
	tablename string   // table name
	relPath   []string // sequence of relations, tables or labels, including the field's own table
	jsonPath  string   // json expression like "->a->b->>c"
	last      string   // the last item of the field: field name or json item
}

// SelectField can contain a selected field or a relationship with another table.
// For example, select=id,name,other(name) will produce:
//  1. a SelectField with field.name = "id"
//  2. a SelectField with field.name = "name"
//  3. a SelectField with an empty field and a SelectRelation with relation.name = "other",
//     itself with a SelectField in relation.fields with field.name = "name".
//
// label is used as an alias both for a field and a relation.
type SelectField struct {
	field    Field           // field info
	label    string          // label for the field or the nested relation
	cast     string          // type cast
	relation *SelectRelation // relation (can be null)
}

// SelectRelation stores information about a relationship, expressed in the select clause like:
// /table?select=id,name,other1(name),...other2(id)
type SelectRelation struct {
	name   string        // the (related) table name
	parent string        // the parent table
	spread bool          // has a spread operator?
	inner  bool          // is an inner join?
	fk     string        //
	fields []SelectField // requested fields
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
	inserted bool
	children []*WhereConditionNode
}

func (node *WhereConditionNode) isRootNode() bool {
	return node.operator == "" && node.field.name == ""
}

// QueryParts is the root of the AST produced by the request parser
type QueryParts struct {
	selectFields        []SelectField
	columnFields        map[string]struct{}
	conflictFields      map[string]struct{}
	orderFields         []OrderField
	limit               string
	offset              string
	whereConditionsTree *WhereConditionNode
}

type QueryOptions struct {
	Schema               string
	ContentType          string // json, csv
	ReturnRepresentation bool
	MergeDuplicates      bool
	IgnoreDuplicates     bool
	TxCommit             bool
	TxRollback           bool
	Singular             bool
	HasRange             bool
	RangeMin             int64
	RangeMax             int64
	Count                string // exact, planned, estimated
}

// RequestParser is the interface used to parse the query string in the request and
// extract the significant headers.
// Initially we will support PostgREST mode and later perhaps others (Django?).
type RequestParser interface {
	parse(mainTable string, filters Filters) (*QueryParts, error)
	getRequestOptions(req *Request) QueryOptions
	filterParameters(filters Filters) Filters
}

type PostgRestParser struct {
	tokens []string
	cur    int
}

type ParseError struct {
	msg string // description of error
}

func (e *ParseError) Error() string { return e.msg }

var postgRestReservedWords = map[string]struct{}{
	"select": {}, "column": {}, "order": {}, "limit": {}, "offset": {}, "not": {}, "and": {}, "or": {}, "on_conlict": {},
}

// From https://github.com/PostgREST/postgrest/blob/main/src/PostgREST/Query/SqlFragment.hs
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

// filterParameters checks the map of filters and skip the keys with this rationale:
// - they are not reserved words and
// - they do not have an operator prefixing their value
// It returns the skipped parameters.
func (p PostgRestParser) filterParameters(filters Filters) Filters {
	skipped := make(Filters)
	for k, vv := range filters {
		if _, exists := postgRestReservedWords[k]; !exists {
			var toRemove []int
			for i, v := range vv {
				prefix := strings.SplitN(v, ".", 2)[0]
				if _, exists := postgRestParserOperators[prefix]; !exists {
					skipped[k] = append(skipped[k], v)
					toRemove = append(toRemove, i)
				}
			}
			for i := len(toRemove) - 1; i >= 0; i-- {
				index := toRemove[i]
				vv = append(vv[:index], vv[index+1:]...)
			}
			if len(vv) == 0 {
				delete(filters, k)
			} else {
				filters[k] = vv
			}
		}
	}
	return skipped
}

// scan splits the string s using the separators and
// skipping double quoted strings.
// Returns a slice of substrings and separators.
// sep is the set of single char separators.
// longSep is the set of multi char separators (put longest first!)
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
			if cur == '"' || cur == '\'' {
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
			if cur == '"' || cur == '\'' {
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

// next returns the next token and advances the cursor.
// It returns an empty string if it is at the end.
func (p *PostgRestParser) next() string {
	if p.cur == len(p.tokens) {
		return ""
	}
	t := p.tokens[p.cur]
	p.cur++
	return t
}

// back takes one step back
func (p *PostgRestParser) back() {
	if p.cur != 0 {
		p.cur--
	}
}

// lookAhead returns the next token _without_ advancing the cursor.
// It returns an empty string if it is at the end.
func (p *PostgRestParser) lookAhead() string {
	if p.cur == len(p.tokens) {
		return ""
	}
	return p.tokens[p.cur]
}

// reset reinitializes the parser
func (p *PostgRestParser) reset() {
	p.tokens = nil
	p.cur = 0
}

// General grammar:
// (should see this, discovered later:
// https://github.com/PostgREST/postgrest-docs/issues/228#issuecomment-346981443)
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

func (p *PostgRestParser) field(mayHaveTable bool, mayBeEmpty bool) (f Field, err error) {
	token := p.next()
	if token == "" {
		if mayBeEmpty {
			token = "*"
		} else {
			return f, &ParseError{"field expected"}
		}
	}
	if mayHaveTable {
		// tables
		var lastTable string
		for p.lookAhead() == "." {
			lastTable = token
			f.relPath = append(f.relPath, lastTable)
			p.next()
			token = p.next()
		}
		if lastTable != "" {
			f.tablename = lastTable
		}
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
	p.scan(s, ".,():!", "->>", "->", "::", "...")
	return p.selectList(nil)
}

func (p *PostgRestParser) selectList(rel *SelectRelation) (selectFields []SelectField, err error) {
	selectFields, err = p.selectItem(rel)
	if err != nil {
		return nil, err
	}
	for p.next() == "," {
		fields, err := p.selectItem(rel)
		if err != nil {
			return nil, err
		}
		selectFields = append(selectFields, fields...)
	}
	return selectFields, nil
}

func (p *PostgRestParser) selectItem(rel *SelectRelation) (selectFields []SelectField, err error) {
	var label, cast, fk string
	var spread, inner bool
	token := p.next()
	if token == "..." {
		spread = true
		token = p.next()
	}
	if p.lookAhead() == ":" {
		label = token
		p.next()
	} else {
		p.back()
	}
	field, err := p.field(false, true)
	if err != nil {
		return nil, err
	}
	if label == "" {
		label = field.last
	}
	if p.lookAhead() == "::" {
		p.next()
		cast = p.next()
	}
	if p.lookAhead() == "!" {
		p.next()
		fk = p.next()
		if fk == "inner" {
			fk = ""
			inner = true
		}
	}
	if p.lookAhead() != "(" {
		// field

		if spread {
			return nil, &ParseError{"cannot use the spread operator on fields"}
		}
		if rel != nil {
			field.tablename = rel.name
		}
		if field.name != "," {
			selectFields = append(selectFields, SelectField{field, label, cast, nil})
		} else {
			p.back()
		}
	} else {
		// table

		p.next()
		if cast != "" {
			return nil, &ParseError{"table cannot have cast"}
		}
		var parent string
		if rel != nil {
			parent = rel.name
		}
		newrel := &SelectRelation{field.name, parent, spread, inner, fk, nil}
		if p.lookAhead() != ")" {
			newrel.fields, err = p.selectList(newrel)
			if err != nil {
				return nil, err
			}
		}
		selectFields = append(selectFields, SelectField{label: label, relation: newrel})
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
		field, err := p.field(false, false)
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
		if value1 != "" &&
			value1 != "asc" && value1 != "desc" && value1 != "nullsfirst" && value1 != "nullslast" {
			return nil, &ParseError{"asc, desc, nullsfirst or nullslast expected"}
		}
		if value2 != "" &&
			value2 != "asc" && value2 != "desc" && value2 != "nullsfirst" && value2 != "nullslast" {
			return nil, &ParseError{"asc, desc, nullsfirst or nullslast expected"}
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

// isBooleanOp returns true if op is one of "and", "or", "not.and", "not.or"
func isBooleanOp(op string) bool {
	return op == "and" || op == "or" || op == "not.and" || op == "not.or"
}

// isBooleanOpStrict returns true if op is one of "and", "or", "not"
func isBooleanOpStrict(op string) bool {
	return op == "and" || op == "or" || op == "not"
}

// hasBooleanOp returns true if op ends with "and", "or"
func hasBooleanOp(op string) bool {
	return op == "and" || op == "or" || strings.HasSuffix(op, ".and") || strings.HasSuffix(op, ".or")
}

func (p *PostgRestParser) scanWhereCondition(k, v string) {
	p.reset()
	if hasBooleanOp(k) {
		if isBooleanOp(k) {
			v = k + v
		} else {
			// a hack to recognize boolean operators more easily in case of "embedded filters"
			v = "__boolean_later__." + k + v
		}
	} else {
		v = k + "=" + v
	}
	p.scan(v, "=.,()[]{}:", "->>", "->")
}

func (p *PostgRestParser) parseWhereCondition(mainTable, key, value string, root *WhereConditionNode) error {
	p.scanWhereCondition(key, value)
	return p.cond(mainTable, root)
}

func (p *PostgRestParser) completeIfFloat() string {
	// @@ should test if the current token is a number
	if p.lookAhead() == "." {
		return p.next() + p.next()
	}
	return ""
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
			value += p.completeIfFloat()
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
			value += p.completeIfFloat()
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
		} else {
			value += p.completeIfFloat()
		}
	}
	value = strings.ReplaceAll(value, "*", "%")
	node.values = append(node.values, value)
	return nil
}

func (p *PostgRestParser) booleanOp(table string, node *WhereConditionNode) (err error) {
	token := p.next()
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
	err = p.cond(table, node)
	if err != nil {
		return err
	}
	token = p.next()
	for token == "," {
		err = p.cond(table, node)
		if err != nil {
			return err
		}
		token = p.next()
	}
	if token != ")" {
		return &ParseError{"')' expected"}
	}
	return nil
}

func (p *PostgRestParser) cond(mainTable string, parent *WhereConditionNode) (err error) {
	node := &WhereConditionNode{}
	token := p.lookAhead()
	if token == "__boolean_later__" {
		// here we know that the sequence is something like "__boolean_later__.(table.)*[not.](and|or)"
		p.next() // boolean_later
		p.next() // dot
		var boolOpTable []string
		for { // see if we have "embedded resources" (eg table.or etc)
			token = p.lookAhead()
			if isBooleanOpStrict(token) {
				break
			}
			boolOpTable = append(boolOpTable, p.next())
			if token = p.next(); token != "." {
				return &ParseError{"'.' expected"}
			}
		}
		if len(boolOpTable) == 0 {
			boolOpTable = append(boolOpTable, mainTable)
		}
		table := boolOpTable[len(boolOpTable)-1]
		node.field.tablename = table
		node.field.relPath = boolOpTable
		p.booleanOp(table, node)

	} else if isBooleanOpStrict(token) {
		node.field.tablename = mainTable
		p.booleanOp(mainTable, node)

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
		node.field, err = p.field(mayHaveTable, false)
		if err != nil {
			return err
		}
		if node.field.tablename == "" {
			if parent.field.tablename != "" {
				node.field.tablename = parent.field.tablename
				node.field.relPath = parent.field.relPath
			} else {
				node.field.tablename = mainTable
			}
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
func (p PostgRestParser) parse(mainTable string, filters Filters) (parts *QueryParts, err error) {
	parts = &QueryParts{}

	// SELECT
	var sel string
	if selectFilters, ok := filters["select"]; ok {
		for i, csFields := range selectFilters {
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

	// COLUMNS
	// columns=c1,c2,c3
	var columns string
	if columnsFilters, ok := filters["columns"]; ok {
		for i, cFields := range columnsFilters {
			if i != 0 {
				columns += ","
			}
			columns += cFields
		}
		parts.columnFields = make(map[string]struct{})
		//for _, c := range p.parseColumns(columns) {
		for _, c := range strings.Split(columns, ",") {
			parts.columnFields[c] = struct{}{}
		}
		delete(filters, "columns")
	}

	// ON CONFLICT
	// on_conflict=c1,c2,c3
	var onconflict string
	if columnsFilters, ok := filters["on_conflict"]; ok {
		for i, cFields := range columnsFilters {
			if i != 0 {
				onconflict += ","
			}
			onconflict += cFields
		}
		parts.conflictFields = make(map[string]struct{})
		for _, c := range strings.Split(onconflict, ",") {
			parts.conflictFields[c] = struct{}{}
		}
		delete(filters, "on_conflict")
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

var rangeRe = regexp.MustCompile(`^(\d+)?-(\d+)?$`)

func (p PostgRestParser) getRequestOptions(req *Request) QueryOptions {
	header := req.Header
	options := QueryOptions{}

	accept := header.Get("Accept")
	mediatype, _, _ := mime.ParseMediaType(accept)
	switch mediatype {
	case "application/vnd.pgrst.object+json":
		options.ContentType = "json"
		options.Singular = true
	case "application/json":
		options.ContentType = "json"
	case "text/csv":
		options.ContentType = "csv"
	}

	preferValues := header.Values("Prefer")
	for _, prefer := range preferValues {
		switch prefer {
		case "return=representation":
			options.ReturnRepresentation = true
		case "resolution=merge-duplicates":
			options.MergeDuplicates = true
		case "resolution=ignore-duplicates":
			options.IgnoreDuplicates = true
		case "tx=commit":
			options.TxCommit = true
		case "tx=rollback":
			options.TxRollback = true
		}
	}

	rangeValues := header.Values("Range")
	if l := len(rangeValues); l != 0 {
		r := rangeValues[l-1]
		matches := rangeRe.FindStringSubmatch(r)
		if matches != nil {
			options.HasRange = true
			options.RangeMin, _ = strconv.ParseInt(matches[1], 10, 64)
			options.RangeMax, _ = strconv.ParseInt(matches[2], 10, 64)
		}
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
