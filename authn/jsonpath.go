package authn

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// RoleClaimKey locates the role in the JWT claims, PostgREST's
// jwt-role-claim-key. PostgREST 16 reads it as a JSONPath query (RFC 9535,
// Config/JSPath.hs, with the aeson-jsonpath library); smoothdb reads the
// subset that the identity providers need and PostgREST documents and tests:
// $, .name, ['name'] and ["name"], [index], and one filter on the current
// element, [?@ == "text"], [?@ != "text"] or [?search(@, "regex")]. A key
// with any other JSONPath construct is refused at startup, naming it, so that
// the divergence from PostgREST is never a different role at runtime. A key
// that does not start with "$" is read with the JSPath DSL of PostgREST 15
// and earlier (Config/DeprecatedJSPath.hs), which 16.4 still accepts with a
// warning.
type RoleClaimKey struct {
	query      jpQuery
	deprecated []jspExp
	src        string
}

// DefaultRoleClaimKey is PostgREST's default jwt-role-claim-key.
const DefaultRoleClaimKey = "$.role"

// ParseRoleClaimKey parses a jwt-role-claim-key value; an empty one is the
// default.
func ParseRoleClaimKey(s string) (*RoleClaimKey, error) {
	if s == "" {
		s = DefaultRoleClaimKey
	}
	if strings.HasPrefix(s, "$") {
		q, err := parseJSONPath(s)
		if u, ok := err.(*unsupportedError); ok {
			return nil, fmt.Errorf("'JWTRoleClaimKey' %q: %s is not supported; smoothdb reads a subset of "+
				`PostgREST's JSONPath: $, .name, ['name'], [index], [?@ == "text"], [?@ != "text"], [?search(@, "regex")]`,
				s, u.construct)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse role-claim-key value (%s): %v", s, err)
		}
		return &RoleClaimKey{query: q, src: s}, nil
	}
	d, err := parseDeprecatedJSPath(s)
	if err != nil {
		return nil, fmt.Errorf("failed to parse role-claim-key value (%s): %v", s, err)
	}
	return &RoleClaimKey{deprecated: d, src: s}, nil
}

// Deprecated says whether the key uses the JSPath DSL of PostgREST 15.
func (k *RoleClaimKey) Deprecated() bool { return k.deprecated != nil }

func (k *RoleClaimKey) String() string { return k.src }

// evaluate returns the first value the key selects in the claims.
func (k *RoleClaimKey) evaluate(claims any) (any, bool) {
	if k.deprecated != nil {
		return evaluateDeprecatedJSPath(claims, k.deprecated)
	}
	nodes := k.query.eval(claims)
	if len(nodes) == 0 {
		return nil, false
	}
	return nodes[0], true
}

// The JSON values below are those of encoding/json with UseNumber: nil, bool,
// json.Number, string, []any and map[string]any. Object members are visited
// in key order, as aeson's KeyMap does, so that "the first value" a filter
// selects in an object is the same as PostgREST's.

// --- the JSONPath subset ---------------------------------------------------------

type jpQuery []jpStep

// jpStep is a segment with a single selector: a member name, an array index,
// or a filter on the current element.
type jpStep struct {
	kind  byte // 'n' name, 'i' index, 'f' filter
	name  string
	index int64
	op    string // the filter: "==", "!=" or "search"
	text  string
	re    *regexp.Regexp // search(): nil for an invalid pattern, which never matches
}

func (q jpQuery) eval(root any) []any {
	nodes := []any{root}
	for _, s := range q {
		var out []any
		for _, v := range nodes {
			out = s.sel(v, out)
		}
		if nodes = out; len(nodes) == 0 {
			break
		}
	}
	return nodes
}

func (s jpStep) sel(v any, out []any) []any {
	switch s.kind {
	case 'n':
		if m, ok := v.(map[string]any); ok {
			if c, ok := m[s.name]; ok {
				out = append(out, c)
			}
		}
	case 'i':
		if a, ok := v.([]any); ok {
			i := s.index
			if i < 0 {
				i += int64(len(a))
			}
			if i >= 0 && i < int64(len(a)) {
				out = append(out, a[i])
			}
		}
	case 'f':
		for _, c := range children(v) {
			if s.test(c) {
				out = append(out, c)
			}
		}
	}
	return out
}

// test applies the filter to an element: == holds for the same string only,
// != for anything else (RFC 9535 §2.3.5.2.2), search() for a string with a
// match (§2.4.7).
func (s jpStep) test(v any) bool {
	str, isString := v.(string)
	switch s.op {
	case "==":
		return isString && str == s.text
	case "!=":
		return !isString || str != s.text
	}
	return isString && s.re != nil && s.re.MatchString(str)
}

func children(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]any, len(keys))
		for i, k := range keys {
			out[i] = t[k]
		}
		return out
	}
	return nil
}

// compileIRegexp compiles an I-Regexp (RFC 9485) as a Go regular expression:
// the two dialects differ in "." only, which in I-Regexp matches neither \n
// nor \r. It returns nil for a pattern that is not valid.
func compileIRegexp(p string) *regexp.Regexp {
	var b strings.Builder
	inClass := false
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch {
		case c == '\\' && i+1 < len(p):
			b.WriteByte(c)
			i++
			b.WriteByte(p[i])
			continue
		case c == '[' && !inClass:
			inClass = true
		case c == ']' && inClass:
			inClass = false
		case c == '.' && !inClass:
			b.WriteString(`[^\n\r]`)
			continue
		}
		b.WriteByte(c)
	}
	re, err := regexp.Compile(b.String())
	if err != nil {
		return nil
	}
	return re
}

// --- parser -------------------------------------------------------------------

// unsupportedError is a JSONPath construct that PostgREST 16.4 accepts and
// the subset does not.
type unsupportedError struct{ construct string }

func (e *unsupportedError) Error() string { return e.construct + " is not supported" }

func unsupported(construct string) error { return &unsupportedError{construct} }

type jpParser struct {
	s   string
	pos int
}

func parseJSONPath(s string) (jpQuery, error) {
	p := &jpParser{s: s}
	if !p.eat('$') {
		return nil, p.errorf("expected '$'")
	}
	var q jpQuery
	for {
		// blanks may come before a segment, never at the end
		save := p.pos
		p.blanks()
		switch {
		case p.eatStr(".."):
			return nil, unsupported("the descendant segment ..")
		case p.eat('.'):
			if p.peek() == '*' {
				return nil, unsupported("the wildcard *")
			}
			name, err := p.memberName()
			if err != nil {
				return nil, err
			}
			q = append(q, jpStep{kind: 'n', name: name})
		case p.peek() == '[':
			step, err := p.bracketed()
			if err != nil {
				return nil, err
			}
			q = append(q, step)
		default:
			p.pos = save
			if p.pos != len(p.s) {
				return nil, p.errorf("unexpected %q", p.s[p.pos:])
			}
			return q, nil
		}
	}
}

func (p *jpParser) errorf(format string, args ...any) error {
	return fmt.Errorf("JSONPath, column %d: %s", p.pos+1, fmt.Sprintf(format, args...))
}

func (p *jpParser) peek() byte {
	if p.pos < len(p.s) {
		return p.s[p.pos]
	}
	return 0
}

func (p *jpParser) eat(c byte) bool {
	if p.peek() == c && p.pos < len(p.s) {
		p.pos++
		return true
	}
	return false
}

func (p *jpParser) eatStr(t string) bool {
	if strings.HasPrefix(p.s[p.pos:], t) {
		p.pos += len(t)
		return true
	}
	return false
}

// blanks skips RFC 9535 blank space (S).
func (p *jpParser) blanks() {
	for p.pos < len(p.s) {
		switch p.s[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

// memberName parses a member-name-shorthand: ALPHA / "_" / non-ASCII, then
// those or DIGIT.
func (p *jpParser) memberName() (string, error) {
	start := p.pos
	for p.pos < len(p.s) {
		r, size := utf8.DecodeRuneInString(p.s[p.pos:])
		first := r == '_' || r >= 0x80 || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
		if !(first || (p.pos > start && r >= '0' && r <= '9')) {
			break
		}
		if r == utf8.RuneError && size == 1 {
			return "", p.errorf("invalid UTF-8")
		}
		p.pos += size
	}
	if p.pos == start {
		return "", p.errorf("expected a member name")
	}
	return p.s[start:p.pos], nil
}

// bracketed parses a bracketed selection holding one selector: a name, an
// index or a filter.
func (p *jpParser) bracketed() (jpStep, error) {
	p.eat('[')
	p.blanks()
	var step jpStep
	switch c := p.peek(); {
	case c == '\'' || c == '"':
		name, err := p.stringLiteral()
		if err != nil {
			return step, err
		}
		step = jpStep{kind: 'n', name: name}
	case c == '*':
		return step, unsupported("the wildcard *")
	case c == ':':
		return step, unsupported("a slice [start:end:step]")
	case c == '-' || (c >= '0' && c <= '9'):
		i, err := p.integer()
		if err != nil {
			return step, err
		}
		p.blanks()
		if p.peek() == ':' {
			return step, unsupported("a slice [start:end:step]")
		}
		step = jpStep{kind: 'i', index: i}
	case c == '?':
		p.pos++
		var err error
		if step, err = p.filter(); err != nil {
			return step, err
		}
	default:
		return step, p.errorf("expected a selector")
	}
	p.blanks()
	if p.peek() == ',' {
		return step, unsupported("several selectors in one bracket")
	}
	if !p.eat(']') {
		return step, p.errorf("expected ']'")
	}
	return step, nil
}

// filter parses the filter of the subset, in any parentheses: @ == "text",
// @ != "text" or search(@, "regex").
func (p *jpParser) filter() (jpStep, error) {
	step := jpStep{kind: 'f'}
	parens := 0
	for {
		p.blanks()
		if p.eat('(') {
			parens++
			continue
		}
		if p.peek() == '!' {
			return step, unsupported("the negation !")
		}
		break
	}
	switch c := p.peek(); {
	case c == '@':
		p.pos++
		if err := p.currentOnly(); err != nil {
			return step, err
		}
		op := p.comparisonOp()
		switch op {
		case "":
			if c := p.peek(); c == ']' || c == ')' || c == '&' || c == '|' {
				return step, unsupported("an existence test [?@]")
			}
			return step, p.errorf("expected a comparison")
		case "<", "<=", ">", ">=":
			return step, unsupported("the comparison " + op)
		}
		text, err := p.stringOperand("a literal that is not a string", "a query inside a filter (@.name, @[...], $...)")
		if err != nil {
			return step, err
		}
		step.op, step.text = op, text
	case c == '$':
		return step, unsupported("a query inside a filter (@.name, @[...], $...)")
	case c == '\'' || c == '"' || c == '-' || (c >= '0' && c <= '9') || p.isLiteralKeyword():
		return step, unsupported("a literal on the left of a comparison")
	case c >= 'a' && c <= 'z':
		if err := p.search(&step); err != nil {
			return step, err
		}
	default:
		return step, p.errorf("expected a filter")
	}
	p.blanks()
	switch {
	case strings.HasPrefix(p.s[p.pos:], "&&"):
		return step, unsupported("the logical operator &&")
	case strings.HasPrefix(p.s[p.pos:], "||"):
		return step, unsupported("the logical operator ||")
	}
	for ; parens > 0; parens-- {
		p.blanks()
		if !p.eat(')') {
			return step, p.errorf("expected ')'")
		}
	}
	return step, nil
}

// currentOnly checks that "@" stands alone, the current element itself.
func (p *jpParser) currentOnly() error {
	save := p.pos
	p.blanks()
	if c := p.peek(); c == '.' || c == '[' {
		return unsupported("a query inside a filter (@.name, @[...], $...)")
	}
	p.pos = save
	return nil
}

// search parses search(@, "regex"), the only function PostgREST provides.
func (p *jpParser) search(step *jpStep) error {
	start := p.pos
	for c := p.peek(); (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'; c = p.peek() {
		p.pos++
	}
	if name := p.s[start:p.pos]; name != "search" || !p.eat('(') {
		p.pos = start
		return p.errorf("unknown function: only search() is available")
	}
	p.blanks()
	if !p.eat('@') {
		if c := p.peek(); c == '$' {
			return unsupported("a query inside a filter (@.name, @[...], $...)")
		}
		return unsupported("a search() value other than @")
	}
	if err := p.currentOnly(); err != nil {
		return err
	}
	p.blanks()
	if !p.eat(',') {
		return p.errorf("search() takes two arguments")
	}
	p.blanks()
	pattern, err := p.stringOperand("a search() pattern that is not a string", "a query as the search() pattern")
	if err != nil {
		return err
	}
	p.blanks()
	if !p.eat(')') {
		return p.errorf("expected ')'")
	}
	if p.comparisonOp() != "" {
		return p.errorf("search() is not comparable")
	}
	step.op, step.text, step.re = "search", pattern, compileIRegexp(pattern)
	return nil
}

// stringOperand parses the string literal a filter compares with; another
// literal, or a query, is valid JSONPath beyond the subset.
func (p *jpParser) stringOperand(otherLiteral, query string) (string, error) {
	switch c := p.peek(); {
	case c == '\'' || c == '"':
		return p.stringLiteral()
	case c == '@' || c == '$':
		return "", unsupported(query)
	case c == '-' || (c >= '0' && c <= '9') || p.isLiteralKeyword():
		return "", unsupported(otherLiteral)
	}
	return "", p.errorf("expected a string")
}

// comparisonOp consumes S and an operator, or nothing.
func (p *jpParser) comparisonOp() string {
	save := p.pos
	p.blanks()
	for _, op := range []string{"==", "!=", "<=", ">=", "<", ">"} {
		if p.eatStr(op) {
			p.blanks()
			return op
		}
	}
	p.pos = save
	return ""
}

func (p *jpParser) isLiteralKeyword() bool {
	rest := p.s[p.pos:]
	for _, k := range []string{"true", "false", "null"} {
		if strings.HasPrefix(rest, k) {
			after := rest[len(k):]
			if after == "" {
				return true
			}
			r, _ := utf8.DecodeRuneInString(after)
			if !(r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)) {
				return true
			}
		}
	}
	return false
}

// integer parses an RFC 9535 int: "0", or an optional "-" and digits without
// a leading zero, within the I-JSON range.
func (p *jpParser) integer() (int64, error) {
	start := p.pos
	p.eat('-')
	if p.eat('0') {
		if p.pos-start == 2 {
			return 0, p.errorf("-0 is not an integer")
		}
		if c := p.peek(); c >= '0' && c <= '9' {
			return 0, p.errorf("leading zero")
		}
		return 0, nil
	}
	digits := p.pos
	for c := p.peek(); c >= '0' && c <= '9'; c = p.peek() {
		p.pos++
	}
	if p.pos == digits {
		return 0, p.errorf("expected an integer")
	}
	i, err := strconv.ParseInt(p.s[start:p.pos], 10, 64)
	const maxSafe = 1<<53 - 1
	if err != nil || i > maxSafe || i < -maxSafe {
		return 0, p.errorf("integer out of range")
	}
	return i, nil
}

func (p *jpParser) stringLiteral() (string, error) {
	quote := p.s[p.pos]
	p.pos++
	var b strings.Builder
	for {
		if p.pos >= len(p.s) {
			return "", p.errorf("unterminated string")
		}
		c := p.s[p.pos]
		switch {
		case c == quote:
			p.pos++
			return b.String(), nil
		case c < 0x20:
			return "", p.errorf("control character in string")
		case c == '\\':
			p.pos++
			if p.pos >= len(p.s) {
				return "", p.errorf("unterminated string")
			}
			e := p.s[p.pos]
			p.pos++
			switch e {
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '/', '\\':
				b.WriteByte(e)
			case '\'', '"':
				if e != quote {
					return "", p.errorf("invalid escape")
				}
				b.WriteByte(e)
			case 'u':
				r, err := p.unicodeEscape()
				if err != nil {
					return "", err
				}
				b.WriteRune(r)
			default:
				return "", p.errorf("invalid escape")
			}
		default:
			r, size := utf8.DecodeRuneInString(p.s[p.pos:])
			if r == utf8.RuneError && size == 1 {
				return "", p.errorf("invalid UTF-8")
			}
			b.WriteString(p.s[p.pos : p.pos+size])
			p.pos += size
		}
	}
}

func (p *jpParser) hex4() (rune, error) {
	if p.pos+4 > len(p.s) {
		return 0, p.errorf("invalid \\u escape")
	}
	v, err := strconv.ParseUint(p.s[p.pos:p.pos+4], 16, 32)
	if err != nil {
		return 0, p.errorf("invalid \\u escape")
	}
	p.pos += 4
	return rune(v), nil
}

func (p *jpParser) unicodeEscape() (rune, error) {
	r, err := p.hex4()
	if err != nil {
		return 0, err
	}
	switch {
	case r >= 0xD800 && r <= 0xDBFF:
		if !p.eatStr(`\u`) {
			return 0, p.errorf("lone surrogate")
		}
		lo, err := p.hex4()
		if err != nil {
			return 0, err
		}
		if lo < 0xDC00 || lo > 0xDFFF {
			return 0, p.errorf("lone surrogate")
		}
		return 0x10000 + (r-0xD800)<<10 + (lo - 0xDC00), nil
	case r >= 0xDC00 && r <= 0xDFFF:
		return 0, p.errorf("lone surrogate")
	}
	return r, nil
}

// --- deprecated JSPath ----------------------------------------------------------

// The JSPath DSL of PostgREST 15 and earlier: .key, ."quoted key", [index],
// and one filter on an array at the end, [?(@ op "text")] with op among ==,
// !=, ^== (prefix), ==^ (suffix) and *== (substring).

type jspExp struct {
	kind  byte // 'k' key, 'i' index, 'f' filter
	key   string
	index int
	op    string
	text  string
}

func parseDeprecatedJSPath(s string) ([]jspExp, error) {
	var exps []jspExp
	pos := 0
	quoted := func() (string, bool) {
		if pos >= len(s) || s[pos] != '"' {
			return "", false
		}
		end := strings.IndexByte(s[pos+1:], '"')
		if end < 0 {
			return "", false
		}
		v := s[pos+1 : pos+1+end]
		pos += end + 2
		return v, true
	}
	for pos < len(s) {
		switch {
		case s[pos] == '.':
			pos++
			start := pos
			for pos < len(s) {
				r, size := utf8.DecodeRuneInString(s[pos:])
				if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$' || r == '@') {
					break
				}
				pos += size
			}
			if pos > start {
				exps = append(exps, jspExp{kind: 'k', key: s[start:pos]})
				continue
			}
			v, ok := quoted()
			if !ok {
				return nil, fmt.Errorf("column %d: expected a key", pos+1)
			}
			exps = append(exps, jspExp{kind: 'k', key: v})
		case strings.HasPrefix(s[pos:], "[?("):
			pos += 3
			if pos >= len(s) || s[pos] != '@' {
				return nil, fmt.Errorf("column %d: expected '@'", pos+1)
			}
			pos++
			for pos < len(s) && unicode.IsSpace(rune(s[pos])) {
				pos++
			}
			var op string
			for _, o := range []string{"==^", "==", "!=", "^==", "*=="} {
				if strings.HasPrefix(s[pos:], o) {
					op = o
					break
				}
			}
			if op == "" {
				return nil, fmt.Errorf("column %d: expected an operator", pos+1)
			}
			pos += len(op)
			for pos < len(s) && unicode.IsSpace(rune(s[pos])) {
				pos++
			}
			v, ok := quoted()
			if !ok || !strings.HasPrefix(s[pos:], ")]") {
				return nil, fmt.Errorf("column %d: expected a quoted value and ')]'", pos+1)
			}
			pos += 2
			if pos != len(s) {
				return nil, fmt.Errorf("column %d: a filter must come last", pos+1)
			}
			exps = append(exps, jspExp{kind: 'f', op: op, text: v})
		case s[pos] == '[':
			pos++
			start := pos
			for pos < len(s) && s[pos] >= '0' && s[pos] <= '9' {
				pos++
			}
			if pos == start || pos >= len(s) || s[pos] != ']' {
				return nil, fmt.Errorf("column %d: expected an index", pos+1)
			}
			i, err := strconv.Atoi(s[start:pos])
			if err != nil {
				return nil, fmt.Errorf("column %d: index out of range", start+1)
			}
			pos++
			exps = append(exps, jspExp{kind: 'i', index: i})
		default:
			return nil, fmt.Errorf(`column %d: unexpected %q, expecting ".", "[?(" or "["`, pos+1, s[pos:pos+1])
		}
	}
	if len(exps) == 0 {
		return nil, fmt.Errorf(`column 1: expecting ".", "[?(" or "["`)
	}
	return exps, nil
}

func evaluateDeprecatedJSPath(v any, exps []jspExp) (any, bool) {
	for _, e := range exps {
		switch e.kind {
		case 'k':
			m, ok := v.(map[string]any)
			if !ok {
				return nil, false
			}
			if v, ok = m[e.key]; !ok {
				return nil, false
			}
		case 'i':
			a, ok := v.([]any)
			if !ok || e.index >= len(a) {
				return nil, false
			}
			v = a[e.index]
		case 'f':
			a, ok := v.([]any)
			if !ok {
				return nil, false
			}
			for _, x := range a {
				s, ok := x.(string)
				if ok && jspMatch(e.op, e.text, s) {
					return s, true
				}
			}
			return nil, false
		}
	}
	return v, true
}

func jspMatch(op, pattern, s string) bool {
	switch op {
	case "==":
		return s == pattern
	case "!=":
		return s != pattern
	case "^==":
		return strings.HasPrefix(s, pattern)
	case "==^":
		return strings.HasSuffix(s, pattern)
	case "*==":
		return strings.Contains(s, pattern)
	}
	return false
}
