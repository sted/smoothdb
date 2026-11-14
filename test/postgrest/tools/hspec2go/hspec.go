package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Block is a describe, context or it block of a spec file.
type Block struct {
	Kind   string // describe, context, it
	Title  string
	Line   int // 1-based line of the block header
	Indent int
}

// Case is one request of an it block and the response it expects. An it
// with several requests gives several cases (Index counts them from 0).
type Case struct {
	Path  []Block // the enclosing describe and context blocks
	It    Block
	Index int
	// Lines is the hspec text of the request, from FirstLine to LastLine.
	FirstLine, LastLine int

	Method      string
	Query       string
	Headers     [][2]string
	Body        string
	BodyIsJSON  bool
	Status      int
	Expected    string // the expected body; "" with HasExpected false = no check
	HasExpected bool
	ExpEmpty    bool
	ExpHeaders  [][2]string

	// TODO is set when the request or the expectation uses a form that the
	// tool does not convert. The case is then emitted as a comment.
	TODO string
}

// Leftover is hspec text inside an it block that is not a request with
// shouldRespondWith (liftIO, let, shouldBe, ...). The tool reports it as a
// TODO, so that no part of the spec is dropped without a trace.
type Leftover struct {
	It                  Block
	FirstLine, LastLine int
}

// SpecFile is the result of reading one *Spec.hs file.
type SpecFile struct {
	Name      string
	Lines     []string
	BaseInd   int
	Its       []ItBlock
	Cases     []Case
	Leftovers []Leftover
	Bindings  []Binding
	bindLine  map[int]bool
}

// Binding is a simple let or where binding (name = expression). The tool
// puts its expression in place of the name in requests and expectations.
type Binding struct {
	Name         string
	Line, Indent int
	Last         int
	Toks         []token
}

// ItBlock is an it block with its enclosing blocks and its source span.
type ItBlock struct {
	Path            []Block
	It              Block
	First, Last     int // 1-based, inclusive
	Cases, TODOs    int
	LeftoverStrings []string
}

// a block header; it may follow a definition (spec actualPgVersion = describe "...")
// or the in of a let (in  it "...")
var blockRE = regexp.MustCompile(`^(\s*)(?:[a-z][\w' ]*=\s*|in\s+)?(describe|context|it|specify)\s+"((?:[^"\\]|\\.)*)"`)

// ParseSpec reads the text of a spec file and returns its it blocks and
// cases.
func ParseSpec(name, src string) *SpecFile {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	f := &SpecFile{Name: name, Lines: lines, BaseInd: -1}
	var stack []Block
	type open struct {
		path []Block
		it   Block
	}
	var cur *open
	curFirst := 0
	closeIt := func(last int) {
		if cur == nil {
			return
		}
		// drop trailing blank lines and comment lines
		for last > curFirst && strings.TrimSpace(lines[last-1]) == "" {
			last--
		}
		f.Its = append(f.Its, ItBlock{Path: cur.path, It: cur.it, First: curFirst, Last: last})
		cur = nil
	}
	for i, l := range lines {
		ln := i + 1
		trim := strings.TrimSpace(l)
		if trim == "" || strings.HasPrefix(trim, "--") {
			continue
		}
		ind := len(l) - len(strings.TrimLeft(l, " "))
		if cur != nil && ind <= cur.it.Indent {
			closeIt(ln - 1)
		}
		m := blockRE.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		for len(stack) > 0 && stack[len(stack)-1].Indent >= ind {
			stack = stack[:len(stack)-1]
		}
		title, err := strconv.Unquote(`"` + m[3] + `"`)
		if err != nil {
			title = m[3]
		}
		b := Block{Kind: m[2], Title: title, Line: ln, Indent: ind}
		if f.BaseInd < 0 || ind < f.BaseInd {
			f.BaseInd = ind
		}
		if b.Kind == "it" || b.Kind == "specify" {
			b.Kind = "it"
			cur = &open{path: append([]Block(nil), stack...), it: b}
			curFirst = ln
			continue
		}
		stack = append(stack, b)
	}
	closeIt(len(lines))
	f.findBindings()
	for i := range f.Its {
		f.convertIt(&f.Its[i])
	}
	return f
}

// --- tokens ---

type tokKind int

const (
	tIdent tokKind = iota
	tStr
	tQQ
	tNum
	tOp
	tPunct
)

type token struct {
	kind tokKind
	text string // the decoded value for tStr and tQQ, the text otherwise
	qq   string // the quasiquoter name for tQQ (json, str, ...)
	line int
	// bound is true for the tokens of a binding: they are not a leftover
	bound bool
}

func (t token) is(kind tokKind, text string) bool { return t.kind == kind && t.text == text }

// tokenize splits the hspec text of lines [first, last] into tokens. It
// knows strings, quasiquotes, numbers, identifiers, operators and brackets,
// and drops -- comments.
func tokenize(lines []string, first, last int) ([]token, error) {
	var toks []token
	src := strings.Join(lines[first-1:last], "\n")
	line := first
	i := 0
	for i < len(src) {
		c := src[i]
		switch {
		case c == '\n':
			line++
			i++
		case c == ' ' || c == '\t' || c == '\r':
			i++
		case c == '-' && i+1 < len(src) && src[i+1] == '-' && (i+2 >= len(src) || !strings.ContainsRune("!#$%&*+./<=>?@\\^|~:", rune(src[i+2]))):
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '"':
			j := i + 1
			var sb strings.Builder
			for j < len(src) && src[j] != '"' {
				if src[j] == '\\' && j+1 < len(src) {
					j++
					switch src[j] {
					case 'n':
						sb.WriteByte('\n')
					case 't':
						sb.WriteByte('\t')
					case 'r':
						sb.WriteByte('\r')
					case '&':
					case '\n', ' ', '\t':
						// a string gap: \ whitespace \
						for j < len(src) && src[j] != '\\' {
							if src[j] == '\n' {
								line++
							}
							j++
						}
					default:
						if src[j] >= '0' && src[j] <= '9' {
							k := j
							for k < len(src) && src[k] >= '0' && src[k] <= '9' {
								k++
							}
							n, _ := strconv.Atoi(src[j:k])
							sb.WriteRune(rune(n))
							j = k - 1
						} else {
							sb.WriteByte(src[j])
						}
					}
					j++
					continue
				}
				if src[j] == '\n' {
					return nil, fmt.Errorf("line %d: unterminated string", line)
				}
				sb.WriteByte(src[j])
				j++
			}
			toks = append(toks, token{kind: tStr, text: sb.String(), line: line})
			i = j + 1
		case c == '[' && qqStart(src[i:]) != "":
			name := qqStart(src[i:])
			j := strings.Index(src[i:], "|]")
			if j < 0 {
				return nil, fmt.Errorf("line %d: unterminated quasiquote", line)
			}
			body := src[i+len(name)+2 : i+j]
			toks = append(toks, token{kind: tQQ, text: body, qq: name, line: line})
			line += strings.Count(src[i:i+j], "\n")
			i += j + 2
		case c >= '0' && c <= '9':
			j := i
			for j < len(src) && (src[j] >= '0' && src[j] <= '9') {
				j++
			}
			toks = append(toks, token{kind: tNum, text: src[i:j], line: line})
			i = j
		case isIdentStart(c):
			j := i
			for j < len(src) && (isIdentStart(src[j]) || (src[j] >= '0' && src[j] <= '9') || src[j] == '\'' || src[j] == '.') {
				j++
			}
			toks = append(toks, token{kind: tIdent, text: strings.TrimRight(src[i:j], "."), line: line})
			i = j
		case c == '`':
			j := strings.IndexByte(src[i+1:], '`')
			if j < 0 {
				return nil, fmt.Errorf("line %d: unterminated backtick", line)
			}
			toks = append(toks, token{kind: tOp, text: src[i+1 : i+1+j], line: line})
			i += j + 2
		case strings.ContainsRune("()[]{},", rune(c)):
			toks = append(toks, token{kind: tPunct, text: string(c), line: line})
			i++
		default:
			j := i
			for j < len(src) && strings.ContainsRune("!#$%&*+./<=>?@\\^|~:-", rune(src[j])) {
				j++
			}
			if j == i {
				j = i + 1
			}
			toks = append(toks, token{kind: tOp, text: src[i:j], line: line})
			i = j
		}
	}
	return toks, nil
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

var qqRE = regexp.MustCompile(`^\[([a-zA-Z][a-zA-Z0-9]*)\|`)

func qqStart(s string) string {
	m := qqRE.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return m[1]
}

// --- conversion of an it block ---

var requestWords = map[string]string{
	"get": "GET", "post": "POST", "patch": "PATCH", "put": "PUT", "delete": "DELETE",
	"options": "OPTIONS", "request": "",
}

// trivial tokens between statements: they carry no meaning for a case
func trivial(t token) bool {
	return t.is(tOp, "$") || t.is(tIdent, "do") || t.is(tIdent, "in") || t.is(tIdent, "where") || t.is(tOp, "<$>") ||
		t.is(tIdent, "it") || t.is(tIdent, "specify") || (t.kind == tStr && false)
}

func (f *SpecFile) convertIt(it *ItBlock) {
	toks, err := tokenize(f.Lines, it.First, it.Last)
	if err != nil {
		f.addLeftover(it, it.First, it.Last, "cannot tokenize: "+err.Error())
		return
	}
	toks = f.substitute(toks, it)
	// skip the header: it "title"
	pos := 0
	if pos < len(toks) && toks[pos].kind == tIdent && (toks[pos].text == "it" || toks[pos].text == "specify") {
		pos++
	}
	if pos < len(toks) && toks[pos].kind == tStr {
		pos++
	}
	idx := 0
	for {
		op := -1
		for k := pos; k < len(toks); k++ {
			if toks[k].is(tOp, "shouldRespondWith") {
				op = k
				break
			}
		}
		if op < 0 {
			break
		}
		// the request starts at the last request word before op
		start := -1
		for k := op - 1; k >= pos; k-- {
			if toks[k].kind == tIdent {
				if _, ok := requestWords[toks[k].text]; ok {
					start = k
					break
				}
			}
		}
		end := expectationEnd(toks, op+1)
		c := Case{Path: it.Path, It: it.It, Index: idx}
		if start < 0 {
			c.FirstLine, c.LastLine = toks[pos].line, toks[end-1].line
			c.TODO = "no request form found before shouldRespondWith"
		} else {
			// text before the request that is not trivial is a leftover
			if lo, hi, ok := nonTrivial(toks, pos, start); ok {
				f.addLeftover(it, lo, hi, "")
			}
			c.FirstLine, c.LastLine = toks[start].line, toks[end-1].line
			if err := parseRequest(&c, toks[start:op]); err != nil {
				c.TODO = err.Error()
			} else if err := parseExpectation(&c, toks[op+1:end]); err != nil {
				c.TODO = err.Error()
			}
		}
		f.Cases = append(f.Cases, c)
		it.Cases++
		if c.TODO != "" {
			it.TODOs++
		}
		idx++
		pos = end
	}
	if lo, hi, ok := nonTrivial(toks, pos, len(toks)); ok {
		f.addLeftover(it, lo, hi, "")
	}
}

func nonTrivial(toks []token, from, to int) (int, int, bool) {
	lo, hi := -1, -1
	for k := from; k < to; k++ {
		if trivial(toks[k]) || toks[k].bound {
			continue
		}
		if lo < 0 {
			lo = toks[k].line
		}
		hi = toks[k].line
	}
	return lo, hi, lo >= 0
}

func (f *SpecFile) addLeftover(it *ItBlock, lo, hi int, why string) {
	f.Leftovers = append(f.Leftovers, Leftover{It: it.It, FirstLine: lo, LastLine: hi})
	s := fmt.Sprintf("lines %d-%d", lo, hi)
	if why != "" {
		s += ": " + why
	}
	it.LeftoverStrings = append(it.LeftoverStrings, s)
}

// expectationEnd returns the index after the expectation that starts at
// from: a value (json, string, number) and an optional { ... } record.
func expectationEnd(toks []token, from int) int {
	k := from
	if k < len(toks) && toks[k].is(tPunct, "(") {
		k = matchClose(toks, k) + 1
	} else if k < len(toks) {
		k++
	}
	if k < len(toks) && toks[k].is(tPunct, "{") {
		k = matchClose(toks, k) + 1
	}
	if k > len(toks) {
		k = len(toks)
	}
	return k
}

func matchClose(toks []token, open int) int {
	depth := 0
	for k := open; k < len(toks); k++ {
		if toks[k].kind == tPunct {
			switch toks[k].text {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
				if depth == 0 {
					return k
				}
			}
		}
	}
	return len(toks) - 1
}

// stripParens removes parentheses that enclose the whole token list.
func stripParens(t []token) []token {
	for len(t) >= 2 && t[0].is(tPunct, "(") && matchClose(t, 0) == len(t)-1 {
		t = t[1 : len(t)-1]
	}
	return t
}

// splitArgs splits a token list into arguments: a token, or a bracketed
// group.
func splitArgs(t []token) [][]token {
	var args [][]token
	for k := 0; k < len(t); {
		if t[k].kind == tPunct && (t[k].text == "(" || t[k].text == "[") {
			e := matchClose(t, k)
			args = append(args, t[k:e+1])
			k = e + 1
			continue
		}
		args = append(args, t[k:k+1])
		k++
	}
	return args
}

func text(t []token) string {
	var parts []string
	for _, x := range t {
		switch x.kind {
		case tStr:
			parts = append(parts, strconv.Quote(x.text))
		case tQQ:
			parts = append(parts, "["+x.qq+"|"+x.text+"|]")
		default:
			parts = append(parts, x.text)
		}
	}
	return strings.Join(parts, " ")
}

func parseRequest(c *Case, t []token) error {
	word := t[0].text
	args := splitArgs(t[1:])
	method := requestWords[word]
	if word == "request" {
		if len(args) != 4 {
			return fmt.Errorf("request with %d arguments: %s", len(args), text(t))
		}
		m := stripParens(args[0])
		if len(m) != 1 || m[0].kind != tIdent || !strings.HasPrefix(m[0].text, "method") {
			return fmt.Errorf("unknown method %s", text(args[0]))
		}
		method = strings.ToUpper(strings.TrimPrefix(m[0].text, "method"))
		args = args[1:]
		h, err := parseHeaders(args[1])
		if err != nil {
			return err
		}
		c.Headers = h
		args = []([]token){args[0], args[2]}
	} else if len(args) != 1 && len(args) != 2 {
		return fmt.Errorf("%s with %d arguments: %s", word, len(args), text(t))
	}
	c.Method = method
	p := stripParens(args[0])
	if len(p) != 1 || p[0].kind != tStr {
		return fmt.Errorf("the path is not a string: %s", text(args[0]))
	}
	c.Query = p[0].text
	if len(args) == 2 {
		b := stripParens(args[1])
		switch {
		case len(b) == 1 && b[0].is(tIdent, "mempty"):
		case len(b) == 1 && b[0].kind == tStr:
			c.Body = b[0].text
		case len(b) == 1 && b[0].kind == tQQ && b[0].qq == "json":
			s, err := jsonQQ(b[0].text)
			if err != nil {
				return fmt.Errorf("request body: %v", err)
			}
			c.Body, c.BodyIsJSON = s, true
		case len(b) == 1 && b[0].kind == tQQ && b[0].qq == "str":
			c.Body = b[0].text
		default:
			return fmt.Errorf("request body not converted: %s", text(args[1]))
		}
	}
	return nil
}

func parseHeaders(t []token) ([][2]string, error) {
	t = stripParens(t)
	var out [][2]string
	// split on ++ and :
	var parts [][]token
	depth, from := 0, 0
	for k, x := range t {
		if x.kind == tPunct {
			switch x.text {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			}
		}
		if depth == 0 && x.kind == tOp && (x.text == "++" || x.text == "<>" || x.text == ":") {
			parts = append(parts, t[from:k])
			from = k + 1
		}
	}
	parts = append(parts, t[from:])
	for _, p := range parts {
		p = stripParens(p)
		switch {
		case len(p) == 0:
			return nil, fmt.Errorf("empty header expression")
		case len(p) == 1 && (p[0].is(tIdent, "mempty")):
		case p[0].is(tPunct, "["):
			items, err := splitList(p)
			if err != nil {
				return nil, err
			}
			for _, it := range items {
				h, err := parseHeader(it)
				if err != nil {
					return nil, err
				}
				out = append(out, h...)
			}
		default:
			h, err := parseHeader(p)
			if err != nil {
				return nil, err
			}
			out = append(out, h...)
		}
	}
	return out, nil
}

// splitList splits [a, b, c] into its items.
func splitList(t []token) ([][]token, error) {
	if !t[0].is(tPunct, "[") || matchClose(t, 0) != len(t)-1 {
		return nil, fmt.Errorf("not a list: %s", text(t))
	}
	inner := t[1 : len(t)-1]
	var items [][]token
	depth, from := 0, 0
	for k, x := range inner {
		if x.kind == tPunct {
			switch x.text {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				depth--
			case ",":
				if depth == 0 {
					items = append(items, inner[from:k])
					from = k + 1
				}
			}
		}
	}
	if from < len(inner) {
		items = append(items, inner[from:])
	}
	return items, nil
}

var hNames = map[string]string{
	"hAccept": "Accept", "hContentType": "Content-Type", "hContentLength": "Content-Length",
	"hLocation": "Location", "hAuthorization": "Authorization", "hRange": "Range",
	"hContentRange": "Content-Range", "hOrigin": "Origin", "hVary": "Vary",
	"hContentEncoding": "Content-Encoding", "hCacheControl": "Cache-Control",
	"hAllow": "Allow", "hPrefer": "Prefer", "hContentDisposition": "Content-Disposition",
}

// parseHeader converts one request header expression to name/value pairs.
func parseHeader(t []token) ([][2]string, error) {
	t = collapseParens(stripParens(t))
	// a tuple ("K", "V")
	if len(t) == 0 {
		return nil, fmt.Errorf("empty header")
	}
	if t[0].is(tPunct, "(") || (len(t) == 3 && t[1].is(tPunct, ",")) {
		var inner []token
		if t[0].is(tPunct, "(") {
			inner = t[1 : len(t)-1]
		} else {
			inner = t
		}
		if len(inner) == 3 && inner[1].is(tPunct, ",") && inner[2].kind == tStr {
			name := inner[0].text
			if inner[0].kind == tIdent {
				n, ok := hNames[inner[0].text]
				if !ok {
					return nil, fmt.Errorf("unknown header name %s", inner[0].text)
				}
				name = n
			} else if inner[0].kind != tStr {
				return nil, fmt.Errorf("header name not converted: %s", text(t))
			}
			return [][2]string{{name, inner[2].text}}, nil
		}
		return nil, fmt.Errorf("header not converted: %s", text(t))
	}
	switch {
	case t[0].is(tIdent, "acceptHdrs") && len(t) == 2 && t[1].kind == tStr:
		return [][2]string{{"Accept", t[1].text}}, nil
	case t[0].is(tIdent, "authHeaderJWT") && len(t) == 2 && t[1].kind == tStr:
		return [][2]string{{"Authorization", "Bearer " + t[1].text}}, nil
	case t[0].is(tIdent, "authHeader") && len(t) == 3 && t[1].kind == tStr && t[2].kind == tStr:
		return [][2]string{{"Authorization", t[1].text + " " + t[2].text}}, nil
	case t[0].is(tIdent, "planHdr") && len(t) == 1:
		return [][2]string{{"Accept", "application/vnd.pgrst.plan+json"}}, nil
	case (t[0].is(tIdent, "rangeHdrs") || t[0].is(tIdent, "rangeHdrsWithCount")) && len(t) >= 2:
		arg := t[1:]
		if arg[0].is(tOp, "$") {
			arg = arg[1:]
		}
		r, err := byteRange(stripParens(arg))
		if err != nil {
			return nil, err
		}
		// Range-Unit: items is left out, as in the rest of the port
		var h [][2]string
		if t[0].text == "rangeHdrsWithCount" {
			h = append(h, [2]string{"Prefer", "count=exact"})
		}
		return append(h, [2]string{"Range", r}), nil
	}
	return nil, fmt.Errorf("header not converted: %s", text(t))
}

func byteRange(t []token) (string, error) {
	switch {
	case len(t) == 3 && t[0].is(tIdent, "ByteRangeFromTo") && t[1].kind == tNum && t[2].kind == tNum:
		return t[1].text + "-" + t[2].text, nil
	case len(t) == 2 && t[0].is(tIdent, "ByteRangeFrom") && t[1].kind == tNum:
		return t[1].text + "-", nil
	case len(t) == 2 && t[0].is(tIdent, "ByteRangeSuffix") && t[1].kind == tNum:
		return "-" + t[1].text, nil
	}
	return "", fmt.Errorf("byte range not converted: %s", text(t))
}

func parseExpectation(c *Case, t []token) error {
	if len(t) == 0 {
		return fmt.Errorf("no expectation")
	}
	c.Status = 200
	v := t[0:1]
	rest := t[1:]
	if t[0].is(tPunct, "(") {
		e := matchClose(t, 0)
		v, rest = stripParens(t[:e+1]), t[e+1:]
	}
	switch {
	case len(v) == 1 && v[0].kind == tNum:
		c.Status, _ = strconv.Atoi(v[0].text)
	case len(v) == 1 && v[0].kind == tStr:
		if v[0].text == "" {
			c.ExpEmpty = true
		} else {
			c.Expected, c.HasExpected = v[0].text, true
		}
	case len(v) == 1 && v[0].kind == tQQ && v[0].qq == "json":
		s, err := jsonQQ(v[0].text)
		if err != nil {
			return fmt.Errorf("expected body: %v", err)
		}
		c.Expected, c.HasExpected = s, true
	case len(v) == 1 && v[0].kind == tQQ && v[0].qq == "str":
		c.Expected, c.HasExpected = v[0].text, true
	default:
		return fmt.Errorf("expectation not converted: %s", text(v))
	}
	if len(rest) == 0 {
		return nil
	}
	if !rest[0].is(tPunct, "{") {
		return fmt.Errorf("expectation not converted: %s", text(rest))
	}
	fields, err := splitList(append(append([]token{{kind: tPunct, text: "["}}, rest[1:len(rest)-1]...), token{kind: tPunct, text: "]"}))
	if err != nil {
		return err
	}
	for _, fl := range fields {
		if len(fl) < 3 || fl[0].kind != tIdent || !fl[1].is(tOp, "=") {
			return fmt.Errorf("record field not converted: %s", text(fl))
		}
		val := fl[2:]
		switch fl[0].text {
		case "matchStatus":
			if len(val) != 1 || val[0].kind != tNum {
				return fmt.Errorf("matchStatus not converted: %s", text(val))
			}
			c.Status, _ = strconv.Atoi(val[0].text)
		case "matchHeaders":
			items, err := splitList(val)
			if err != nil {
				return err
			}
			for _, it := range items {
				h, err := matchHeader(it)
				if err != nil {
					return err
				}
				c.ExpHeaders = append(c.ExpHeaders, h)
			}
		default:
			return fmt.Errorf("record field %s not converted", fl[0].text)
		}
	}
	return nil
}

func matchHeader(t []token) ([2]string, error) {
	switch {
	case len(t) == 3 && t[0].kind == tStr && t[1].is(tOp, "<:>") && t[2].kind == tStr:
		return [2]string{t[0].text, t[2].text}, nil
	case len(t) == 1 && t[0].is(tIdent, "matchContentTypeJson"):
		return [2]string{"Content-Type", "application/json; charset=utf-8"}, nil
	case len(t) == 1 && t[0].is(tIdent, "matchContentTypeSingular"):
		return [2]string{"Content-Type", "application/vnd.pgrst.object+json; charset=utf-8"}, nil
	case len(t) == 1 && t[0].is(tIdent, "matchCTArrayStrip"):
		return [2]string{"Content-Type", "application/vnd.pgrst.array+json;nulls=stripped; charset=utf-8"}, nil
	case len(t) == 1 && t[0].is(tIdent, "matchCTSingularStrip"):
		return [2]string{"Content-Type", "application/vnd.pgrst.object+json;nulls=stripped; charset=utf-8"}, nil
	case len(t) == 2 && t[0].is(tIdent, "matchHeaderAbsent"):
		name := t[1].text
		if t[1].kind == tIdent {
			n, ok := hNames[name]
			if !ok {
				return [2]string{}, fmt.Errorf("unknown header name %s", name)
			}
			name = n
		}
		return [2]string{name, ""}, nil
	}
	return [2]string{}, fmt.Errorf("header match not converted: %s", text(t))
}

var bareKeyRE = regexp.MustCompile(`([{,]\s*)([A-Za-z_][A-Za-z0-9_]*)\s*:`)

// jsonQQ returns the text of a [json|...|] block, trimmed. The aeson-qq
// quasiquoter accepts bare object keys: they are quoted here. An
// interpolation (#{...}) is not converted.
func jsonQQ(s string) (string, error) {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "#{") {
		return "", fmt.Errorf("json interpolation not converted: %s", s)
	}
	if json.Valid([]byte(s)) {
		return s, nil
	}
	q := bareKeyRE.ReplaceAllString(s, `$1"$2":`)
	if json.Valid([]byte(q)) {
		return q, nil
	}
	return "", fmt.Errorf("json block is not valid JSON: %s", s)
}

var bindRE = regexp.MustCompile(`^(\s*)(?:let\s+)?([a-z][A-Za-z0-9_']*)\s*=\s*(\S.*)$`)

var notBindable = map[string]bool{"it": true, "describe": true, "context": true, "spec": true}

// findBindings records the lines name = expression of the file. The
// expression goes on while the next lines are more indented and do not
// start with "in".
func (f *SpecFile) findBindings() {
	f.bindLine = map[int]bool{}
	for i, l := range f.Lines {
		m := bindRE.FindStringSubmatch(l)
		// a record field (matchStatus = 201) is not a binding
		if m == nil || notBindable[m[2]] || strings.HasPrefix(m[3], "=") || strings.HasPrefix(m[2], "match") {
			continue
		}
		ind := len(m[1])
		if strings.HasPrefix(strings.TrimSpace(l), "let ") {
			ind += 4
		}
		last := i + 1
		for last < len(f.Lines) {
			n := f.Lines[last]
			t := strings.TrimSpace(n)
			nind := len(n) - len(strings.TrimLeft(n, " "))
			if t == "" || nind <= ind || strings.HasPrefix(t, "in ") || t == "in" {
				break
			}
			last++
		}
		// the expression starts after "="
		eq := strings.Index(l, "=")
		lines := append([]string{strings.Repeat(" ", eq+1) + l[eq+1:]}, f.Lines[i+1:last]...)
		toks, err := tokenize(lines, 1, len(lines))
		if err != nil {
			continue
		}
		for k := range toks {
			toks[k].line += i
		}
		f.Bindings = append(f.Bindings, Binding{Name: m[2], Line: i + 1, Indent: ind, Last: last, Toks: toks})
		for n := i + 1; n <= last; n++ {
			f.bindLine[n] = true
		}
	}
}

// lookup returns the binding of name seen from line: the nearest one before
// it, or else a where binding after the it block.
func (f *SpecFile) lookup(name string, line int, it *ItBlock) *Binding {
	var best *Binding
	for i := range f.Bindings {
		b := &f.Bindings[i]
		if b.Name == name && b.Line < line {
			best = b
		}
	}
	if best != nil {
		return best
	}
	for i := range f.Bindings {
		b := &f.Bindings[i]
		if b.Name == name && b.Line > it.Last {
			return b
		}
	}
	return nil
}

// substitute marks the tokens of binding lines and puts the expression of a
// binding in place of its name.
func (f *SpecFile) substitute(toks []token, it *ItBlock) []token {
	var out []token
	for k, t := range toks {
		if f.bindLine[t.line] && t.line != it.It.Line {
			t.bound = true
			out = append(out, t)
			continue
		}
		if t.kind == tIdent && !t.bound && k > 0 && !toks[k-1].is(tOp, "=") && !(k+1 < len(toks) && toks[k+1].is(tOp, "=")) {
			if _, isReq := requestWords[t.text]; !isReq {
				if b := f.lookup(t.text, t.line, it); b != nil && len(b.Toks) > 0 {
					out = append(out, token{kind: tPunct, text: "(", line: t.line})
					for _, x := range b.Toks {
						x.line = t.line
						out = append(out, x)
					}
					out = append(out, token{kind: tPunct, text: ")", line: t.line})
					continue
				}
			}
		}
		out = append(out, t)
	}
	return out
}

// collapseParens replaces ( x ) by x when x is one token, as a substituted
// binding gives.
func collapseParens(t []token) []token {
	var out []token
	for k := 0; k < len(t); k++ {
		if k+2 < len(t) && t[k].is(tPunct, "(") && t[k+2].is(tPunct, ")") && t[k+1].kind != tPunct {
			out = append(out, t[k+1])
			k += 2
			continue
		}
		out = append(out, t[k])
	}
	return out
}
