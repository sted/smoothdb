package authn

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// RoleClaimKey locates the role in the JWT claims, PostgREST's
// jwt-role-claim-key. It is a JSONPath query (RFC 9535), as in PostgREST 16
// (Config/JSPath.hs); a key that does not parse as one is read with the JSPath
// DSL of PostgREST 15 and earlier (Config/DeprecatedJSPath.hs), which 16.4
// still accepts with a warning. Of the RFC 9535 function extensions only
// search() is available, as in PostgREST (docs/references/auth.rst).
type RoleClaimKey struct {
	query      *jpQuery
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
	q, err := parseJSONPath(s)
	if err == nil {
		return &RoleClaimKey{query: q, src: s}, nil
	}
	d, derr := parseDeprecatedJSPath(s)
	if derr != nil {
		// PostgREST reports the error of the deprecated parser, the last tried
		return nil, fmt.Errorf("failed to parse role-claim-key value (%s): %v", s, derr)
	}
	return &RoleClaimKey{deprecated: d, src: s}, nil
}

// Deprecated says whether the key uses the JSPath DSL of PostgREST 15.
func (k *RoleClaimKey) Deprecated() bool { return k.query == nil }

func (k *RoleClaimKey) String() string { return k.src }

// evaluate returns the first value the key selects in the claims.
func (k *RoleClaimKey) evaluate(claims any) (any, bool) {
	if k.query == nil {
		return evaluateDeprecatedJSPath(claims, k.deprecated)
	}
	nodes := k.query.eval(claims, claims)
	if len(nodes) == 0 {
		return nil, false
	}
	return nodes[0], true
}

// The JSON values below are those of encoding/json with UseNumber: nil, bool,
// json.Number, string, []any and map[string]any. Object members are visited
// in key order, as aeson's KeyMap does, so that "the first value" of a query
// over an object is the same as PostgREST's.

// --- RFC 9535 JSONPath --------------------------------------------------------

type jpQuery struct {
	segments []jpSegment
}

type jpSegment struct {
	descendant bool
	selectors  []jpSelector
}

type jpSelector interface {
	sel(v, root any, out []any) []any
}

type jpName struct{ name string }
type jpWildcard struct{}
type jpIndex struct{ index int64 }
type jpSlice struct {
	start, end *int64
	step       int64
}
type jpFilter struct{ expr jpLogical }

func (q *jpQuery) eval(cur, root any) []any {
	nodes := []any{cur}
	for _, seg := range q.segments {
		var out []any
		for _, n := range nodes {
			if seg.descendant {
				out = seg.descend(n, root, out)
			} else {
				for _, s := range seg.selectors {
					out = s.sel(n, root, out)
				}
			}
		}
		nodes = out
		if len(nodes) == 0 {
			break
		}
	}
	return nodes
}

// descend applies the selectors to v and to each of its descendants, a node
// before its children (RFC 9535 §2.5.2.2).
func (seg jpSegment) descend(v, root any, out []any) []any {
	for _, s := range seg.selectors {
		out = s.sel(v, root, out)
	}
	for _, c := range children(v) {
		out = seg.descend(c, root, out)
	}
	return out
}

func children(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case map[string]any:
		keys := sortedKeys(t)
		out := make([]any, len(keys))
		for i, k := range keys {
			out[i] = t[k]
		}
		return out
	}
	return nil
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (s jpName) sel(v, _ any, out []any) []any {
	if m, ok := v.(map[string]any); ok {
		if c, ok := m[s.name]; ok {
			out = append(out, c)
		}
	}
	return out
}

func (jpWildcard) sel(v, _ any, out []any) []any {
	return append(out, children(v)...)
}

func (s jpIndex) sel(v, _ any, out []any) []any {
	if a, ok := v.([]any); ok {
		i := s.index
		if i < 0 {
			i += int64(len(a))
		}
		if i >= 0 && i < int64(len(a)) {
			out = append(out, a[i])
		}
	}
	return out
}

func (s jpSlice) sel(v, _ any, out []any) []any {
	a, ok := v.([]any)
	if !ok || s.step == 0 {
		return out
	}
	n := int64(len(a))
	norm := func(i int64) int64 {
		if i >= 0 {
			return i
		}
		return n + i
	}
	clamp := func(i, lo, hi int64) int64 { return min(max(i, lo), hi) }
	if s.step > 0 {
		start, end := int64(0), n
		if s.start != nil {
			start = norm(*s.start)
		}
		if s.end != nil {
			end = norm(*s.end)
		}
		for i := clamp(start, 0, n); i < clamp(end, 0, n); i += s.step {
			out = append(out, a[i])
		}
		return out
	}
	start, end := n-1, -n-1
	if s.start != nil {
		start = norm(*s.start)
	}
	if s.end != nil {
		end = norm(*s.end)
	}
	lower := clamp(end, -1, n-1)
	for i := clamp(start, -1, n-1); lower < i; i += s.step {
		out = append(out, a[i])
	}
	return out
}

func (s jpFilter) sel(v, root any, out []any) []any {
	for _, c := range children(v) {
		if s.expr.test(c, root) {
			out = append(out, c)
		}
	}
	return out
}

// Filter expressions (RFC 9535 §2.3.5).

type jpLogical interface {
	test(cur, root any) bool
}

type jpOr []jpLogical
type jpAnd []jpLogical
type jpNot struct{ e jpLogical }

// jpExists is a test expression: the query selects at least one node.
type jpExists struct {
	relative bool
	q        *jpQuery
}

type jpCompare struct {
	op          string
	left, right jpComparable
}

// jpSearch is the search() function extension: the regular expression
// matches a substring of the value (RFC 9535 §2.4.7).
type jpSearch struct {
	value, pattern jpComparable
	re             *regexp.Regexp // compiled once for a literal pattern
}

func (e jpOr) test(cur, root any) bool {
	for _, x := range e {
		if x.test(cur, root) {
			return true
		}
	}
	return false
}

func (e jpAnd) test(cur, root any) bool {
	for _, x := range e {
		if !x.test(cur, root) {
			return false
		}
	}
	return true
}

func (e jpNot) test(cur, root any) bool { return !e.e.test(cur, root) }

func (e jpExists) test(cur, root any) bool {
	start := root
	if e.relative {
		start = cur
	}
	return len(e.q.eval(start, root)) > 0
}

func (e jpSearch) test(cur, root any) bool {
	v, ok := e.value.value(cur, root)
	if !ok {
		return false
	}
	s, ok := v.(string)
	if !ok {
		return false
	}
	re := e.re
	if re == nil {
		p, ok := e.pattern.value(cur, root)
		if !ok {
			return false
		}
		ps, ok := p.(string)
		if !ok {
			return false
		}
		if re = compileIRegexp(ps); re == nil {
			return false
		}
	}
	return re.MatchString(s)
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

func (e jpCompare) test(cur, root any) bool {
	l, lok := e.left.value(cur, root)
	r, rok := e.right.value(cur, root)
	switch e.op {
	case "==":
		return jpEqual(l, lok, r, rok)
	case "!=":
		return !jpEqual(l, lok, r, rok)
	case "<":
		return lok && rok && jpLess(l, r)
	case ">":
		return lok && rok && jpLess(r, l)
	case "<=":
		return lok && rok && jpLess(l, r) || jpEqual(l, lok, r, rok)
	case ">=":
		return lok && rok && jpLess(r, l) || jpEqual(l, lok, r, rok)
	}
	return false
}

// jpComparable is a literal or a singular query; a query that selects no node
// is "Nothing", which equals only Nothing.
type jpComparable interface {
	value(cur, root any) (any, bool)
}

type jpLiteral struct{ v any }

type jpSingular struct {
	relative bool
	segs     []jpSelector // jpName and jpIndex only
}

func (l jpLiteral) value(_, _ any) (any, bool) { return l.v, true }

func (q jpSingular) value(cur, root any) (any, bool) {
	v := root
	if q.relative {
		v = cur
	}
	for _, s := range q.segs {
		out := s.sel(v, nil, nil)
		if len(out) == 0 {
			return nil, false
		}
		v = out[0]
	}
	return v, true
}

func jpEqual(l any, lok bool, r any, rok bool) bool {
	if !lok || !rok {
		return lok == rok
	}
	return jsonEqual(l, r)
}

func jsonEqual(l, r any) bool {
	switch a := l.(type) {
	case nil:
		return r == nil
	case bool:
		b, ok := r.(bool)
		return ok && a == b
	case string:
		b, ok := r.(string)
		return ok && a == b
	case json.Number:
		b, ok := r.(json.Number)
		return ok && numCompare(a, b) == 0
	case []any:
		b, ok := r.([]any)
		if !ok || len(a) != len(b) {
			return false
		}
		for i := range a {
			if !jsonEqual(a[i], b[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		b, ok := r.(map[string]any)
		if !ok || len(a) != len(b) {
			return false
		}
		for k, v := range a {
			w, ok := b[k]
			if !ok || !jsonEqual(v, w) {
				return false
			}
		}
		return true
	}
	return false
}

func jpLess(l, r any) bool {
	switch a := l.(type) {
	case json.Number:
		b, ok := r.(json.Number)
		return ok && numCompare(a, b) < 0
	case string:
		// UTF-8 byte order is Unicode scalar value order
		b, ok := r.(string)
		return ok && a < b
	}
	return false
}

func numCompare(a, b json.Number) int {
	if x, err := a.Int64(); err == nil {
		if y, err := b.Int64(); err == nil {
			switch {
			case x < y:
				return -1
			case x > y:
				return 1
			}
			return 0
		}
	}
	x, _ := a.Float64()
	y, _ := b.Float64()
	switch {
	case x < y:
		return -1
	case x > y:
		return 1
	}
	return 0
}

// --- parser -------------------------------------------------------------------

type jpParser struct {
	s   string
	pos int
}

func parseJSONPath(s string) (*jpQuery, error) {
	p := &jpParser{s: s}
	if !p.eat('$') {
		return nil, p.errorf("expected '$'")
	}
	q, err := p.segments()
	if err != nil {
		return nil, err
	}
	if p.pos != len(p.s) {
		return nil, p.errorf("unexpected %q", p.s[p.pos:])
	}
	return q, nil
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

// segments parses *(S segment): the blanks before a segment belong to it, so
// they are given back when no segment follows.
func (p *jpParser) segments() (*jpQuery, error) {
	q := &jpQuery{}
	for {
		save := p.pos
		p.blanks()
		switch {
		case p.eatStr(".."):
			seg := jpSegment{descendant: true}
			switch {
			case p.peek() == '[':
				sels, err := p.bracketed()
				if err != nil {
					return nil, err
				}
				seg.selectors = sels
			case p.eat('*'):
				seg.selectors = []jpSelector{jpWildcard{}}
			default:
				name, err := p.memberName()
				if err != nil {
					return nil, err
				}
				seg.selectors = []jpSelector{jpName{name}}
			}
			q.segments = append(q.segments, seg)
		case p.eat('.'):
			if p.eat('*') {
				q.segments = append(q.segments, jpSegment{selectors: []jpSelector{jpWildcard{}}})
				continue
			}
			name, err := p.memberName()
			if err != nil {
				return nil, err
			}
			q.segments = append(q.segments, jpSegment{selectors: []jpSelector{jpName{name}}})
		case p.peek() == '[':
			sels, err := p.bracketed()
			if err != nil {
				return nil, err
			}
			q.segments = append(q.segments, jpSegment{selectors: sels})
		default:
			p.pos = save
			return q, nil
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
		if r >= 0xD800 && r <= 0xDFFF {
			break
		}
		p.pos += size
	}
	if p.pos == start {
		return "", p.errorf("expected a member name")
	}
	return p.s[start:p.pos], nil
}

func (p *jpParser) bracketed() ([]jpSelector, error) {
	p.eat('[')
	var sels []jpSelector
	for {
		p.blanks()
		s, err := p.selector()
		if err != nil {
			return nil, err
		}
		sels = append(sels, s)
		p.blanks()
		if p.eat(']') {
			return sels, nil
		}
		if !p.eat(',') {
			return nil, p.errorf("expected ',' or ']'")
		}
	}
}

func (p *jpParser) selector() (jpSelector, error) {
	switch c := p.peek(); {
	case c == '\'' || c == '"':
		s, err := p.stringLiteral()
		if err != nil {
			return nil, err
		}
		return jpName{s}, nil
	case c == '*':
		p.pos++
		return jpWildcard{}, nil
	case c == '?':
		p.pos++
		p.blanks()
		e, err := p.logicalOr()
		if err != nil {
			return nil, err
		}
		return jpFilter{e}, nil
	case c == ':' || c == '-' || (c >= '0' && c <= '9'):
		return p.indexOrSlice()
	}
	return nil, p.errorf("expected a selector")
}

func (p *jpParser) indexOrSlice() (jpSelector, error) {
	var start *int64
	if p.peek() != ':' {
		i, err := p.integer()
		if err != nil {
			return nil, err
		}
		save := p.pos
		p.blanks()
		if p.peek() != ':' {
			p.pos = save
			return jpIndex{i}, nil
		}
		start = &i
	}
	p.eat(':')
	p.blanks()
	sl := jpSlice{start: start, step: 1}
	if c := p.peek(); c == '-' || (c >= '0' && c <= '9') {
		e, err := p.integer()
		if err != nil {
			return nil, err
		}
		sl.end = &e
		p.blanks()
	}
	if p.eat(':') {
		p.blanks()
		if c := p.peek(); c == '-' || (c >= '0' && c <= '9') {
			st, err := p.integer()
			if err != nil {
				return nil, err
			}
			sl.step = st
		}
	}
	return sl, nil
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

func (p *jpParser) logicalOr() (jpLogical, error) {
	var or jpOr
	for {
		e, err := p.logicalAnd()
		if err != nil {
			return nil, err
		}
		or = append(or, e)
		save := p.pos
		p.blanks()
		if !p.eatStr("||") {
			p.pos = save
			break
		}
		p.blanks()
	}
	if len(or) == 1 {
		return or[0], nil
	}
	return or, nil
}

func (p *jpParser) logicalAnd() (jpLogical, error) {
	var and jpAnd
	for {
		e, err := p.basic()
		if err != nil {
			return nil, err
		}
		and = append(and, e)
		save := p.pos
		p.blanks()
		if !p.eatStr("&&") {
			p.pos = save
			break
		}
		p.blanks()
	}
	if len(and) == 1 {
		return and[0], nil
	}
	return and, nil
}

func (p *jpParser) basic() (jpLogical, error) {
	if p.eat('!') {
		p.blanks()
		if p.peek() == '(' {
			e, err := p.paren()
			if err != nil {
				return nil, err
			}
			return jpNot{e}, nil
		}
		e, err := p.testExpr()
		if err != nil {
			return nil, err
		}
		return jpNot{e}, nil
	}
	if p.peek() == '(' {
		return p.paren()
	}
	switch c := p.peek(); {
	case c == '@' || c == '$':
		q, relative, err := p.filterQuery()
		if err != nil {
			return nil, err
		}
		if op := p.comparisonOp(); op != "" {
			left, err := singular(q, relative)
			if err != nil {
				return nil, p.errorf("%v", err)
			}
			return p.comparison(left, op)
		}
		return jpExists{relative: relative, q: q}, nil
	case c >= 'a' && c <= 'z' && !p.isLiteralKeyword():
		e, err := p.function()
		if err != nil {
			return nil, err
		}
		if p.comparisonOp() != "" {
			return nil, p.errorf("search() is not comparable")
		}
		return e, nil
	}
	left, err := p.literal()
	if err != nil {
		return nil, err
	}
	op := p.comparisonOp()
	if op == "" {
		return nil, p.errorf("expected a comparison operator")
	}
	return p.comparison(left, op)
}

func (p *jpParser) testExpr() (jpLogical, error) {
	if c := p.peek(); c == '@' || c == '$' {
		q, relative, err := p.filterQuery()
		if err != nil {
			return nil, err
		}
		return jpExists{relative: relative, q: q}, nil
	}
	return p.function()
}

func (p *jpParser) paren() (jpLogical, error) {
	p.eat('(')
	p.blanks()
	e, err := p.logicalOr()
	if err != nil {
		return nil, err
	}
	p.blanks()
	if !p.eat(')') {
		return nil, p.errorf("expected ')'")
	}
	return e, nil
}

func (p *jpParser) filterQuery() (*jpQuery, bool, error) {
	relative := p.peek() == '@'
	p.pos++
	q, err := p.segments()
	return q, relative, err
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

func (p *jpParser) comparison(left jpComparable, op string) (jpLogical, error) {
	right, err := p.comparable()
	if err != nil {
		return nil, err
	}
	return jpCompare{op: op, left: left, right: right}, nil
}

// comparable parses a literal or a singular query.
func (p *jpParser) comparable() (jpComparable, error) {
	if c := p.peek(); c == '@' || c == '$' {
		q, relative, err := p.filterQuery()
		if err != nil {
			return nil, err
		}
		s, err := singular(q, relative)
		if err != nil {
			return nil, p.errorf("%v", err)
		}
		return s, nil
	}
	if c := p.peek(); c >= 'a' && c <= 'z' && !p.isLiteralKeyword() {
		return nil, p.errorf("only search() is available, and it is not comparable")
	}
	return p.literal()
}

func singular(q *jpQuery, relative bool) (jpSingular, error) {
	s := jpSingular{relative: relative}
	for _, seg := range q.segments {
		if seg.descendant || len(seg.selectors) != 1 {
			return s, fmt.Errorf("a comparison needs a singular query")
		}
		switch seg.selectors[0].(type) {
		case jpName, jpIndex:
			s.segs = append(s.segs, seg.selectors[0])
		default:
			return s, fmt.Errorf("a comparison needs a singular query")
		}
	}
	return s, nil
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

func (p *jpParser) literal() (jpComparable, error) {
	switch c := p.peek(); {
	case c == '\'' || c == '"':
		s, err := p.stringLiteral()
		if err != nil {
			return nil, err
		}
		return jpLiteral{s}, nil
	case p.eatStr("true"):
		return jpLiteral{true}, nil
	case p.eatStr("false"):
		return jpLiteral{false}, nil
	case p.eatStr("null"):
		return jpLiteral{nil}, nil
	case c == '-' || (c >= '0' && c <= '9'):
		return p.number()
	}
	return nil, p.errorf("expected a literal or a query")
}

// number parses an RFC 9535 number: (int / "-0") [frac] [exp].
func (p *jpParser) number() (jpComparable, error) {
	start := p.pos
	if p.eatStr("-0") {
		if c := p.peek(); c >= '0' && c <= '9' {
			return nil, p.errorf("leading zero")
		}
	} else if _, err := p.integer(); err != nil {
		// a number literal is not bound to the I-JSON range of an index
		if !strings.Contains(err.Error(), "range") {
			return nil, err
		}
	}
	if p.eat('.') {
		d := p.pos
		for c := p.peek(); c >= '0' && c <= '9'; c = p.peek() {
			p.pos++
		}
		if p.pos == d {
			return nil, p.errorf("expected a digit")
		}
	}
	if p.eat('e') || p.eat('E') {
		if !p.eat('-') {
			p.eat('+')
		}
		d := p.pos
		for c := p.peek(); c >= '0' && c <= '9'; c = p.peek() {
			p.pos++
		}
		if p.pos == d {
			return nil, p.errorf("expected a digit")
		}
	}
	n := json.Number(p.s[start:p.pos])
	if f, err := n.Float64(); err != nil || math.IsInf(f, 0) {
		return nil, p.errorf("invalid number")
	}
	return jpLiteral{n}, nil
}

// function parses a function expression; search() is the only one.
func (p *jpParser) function() (jpLogical, error) {
	start := p.pos
	for c := p.peek(); (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'; c = p.peek() {
		p.pos++
	}
	name := p.s[start:p.pos]
	if !p.eat('(') {
		return nil, p.errorf("expected a query, a literal or a function")
	}
	if name != "search" {
		p.pos = start
		return nil, p.errorf("unknown function %s(): only search() is available", name)
	}
	p.blanks()
	value, err := p.comparable()
	if err != nil {
		return nil, err
	}
	p.blanks()
	if !p.eat(',') {
		return nil, p.errorf("search() takes two arguments")
	}
	p.blanks()
	pattern, err := p.comparable()
	if err != nil {
		return nil, err
	}
	p.blanks()
	if !p.eat(')') {
		return nil, p.errorf("expected ')'")
	}
	e := jpSearch{value: value, pattern: pattern}
	if l, ok := pattern.(jpLiteral); ok {
		s, _ := l.v.(string)
		if e.re = compileIRegexp(s); e.re == nil {
			// a pattern that is not a valid I-Regexp, or not a string, never
			// matches (RFC 9535 §2.4.7)
			e.pattern = jpLiteral{nil}
		}
	}
	return e, nil
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
