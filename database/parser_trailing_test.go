package database

import (
	"net/url"
	"strings"
	"testing"
)

// parseQuery parses a raw query string for table "t".
func parseQuery(t *testing.T, query string) (*QueryParts, error) {
	t.Helper()
	u, err := url.Parse(query)
	if err != nil {
		t.Fatal(err)
	}
	return PostgRestParser{}.parse("t", u.Query())
}

// Card 35452 parser-trailing. The parser stopped at the first token that could
// not continue an expression and dropped the rest, so a malformed parameter
// answered 200 with another query than the one asked for. Trailing or
// malformed input is a 400 in select, order, logic trees, columns and
// on_conflict, also where PostgREST 16.4 is lenient (the logic tree, the
// top-level in and any/all lists).
func TestParserRejectsTrailingInput(t *testing.T) {
	bad := []struct {
		query string
		what  string // the parameter the error names
	}{
		// order: PostgREST pOrder ends every term with pEnd (',' or eof)
		{"?order=a)x", "order"},
		{"?order=a.asc)x", "order"},
		{"?order=a.desc.nullslast)x", "order"},
		{"?order=a.asc.nullsfirst.x", "order"},
		{"?order=a)", "order"},
		{"?order=a,,b", "order"},
		{"?order=a,", "order"},
		{"?order=,a", "order"},
		{"?order=rel(a)x", "order"},
		{"?emb.order=a)x", "order"},
		// logic trees, top level and nested
		{"?or=(a.eq.1)junk", "logic tree"},
		{"?and=(a.eq.1))", "logic tree"},
		{"?or=(a.eq.1,b.eq.2)x", "logic tree"},
		{"?not.or=(a.eq.1)x", "logic tree"},
		{"?emb.or=(a.eq.1)x", "logic tree"},
		{"?or=(and(a.eq.1,b.eq.2)x,c.eq.3)", "logic tree"},
		{"?or=(a.eq.1,and(b.eq.2,c.eq.3)))", "logic tree"},
		// top-level lists
		{"?a=in.(1,2)x", "filter"},
		{"?a=eq(any).{1,2}x", "filter"},
		// columns and on_conflict: PostgREST pColumns, pFieldName sepBy1 ','
		{"?columns=a,,b", "columns"},
		{"?columns=a,", "columns"},
		{"?columns=,a", "columns"},
		{"?columns=", "columns"},
		{"?columns=a,,b,c&on_conflict=x", "columns"},
		{"?on_conflict=a,,b", "on_conflict"},
		{"?on_conflict=a,", "on_conflict"},
		{"?on_conflict=", "on_conflict"},
		// select: a separator at the end of input, an empty item
		{"?select=a:", "select"},
		{"?select=a::", "select"},
		{"?select=a::,b", "select"},
		{"?select=x:a.sum()::", "select"},
		{"?select=a,", "select"},
		{"?select=a,,b", "select"}, // was accepted as "skipping a column"; PostgREST answers 400
		{"?select=,a", "select"},
		{"?select=a)b", "select"},
	}
	for _, c := range bad {
		parts, err := parseQuery(t, c.query)
		if err == nil {
			t.Errorf("%s: expected a parse error, got select %v, order %v, columns %v, on_conflict %v, where %v",
				c.query, parts.selectFields, parts.orderFields, parts.columnFields, parts.conflictFields,
				parts.whereConditionsTree)
			continue
		}
		if _, ok := err.(*ParseError); !ok {
			t.Errorf("%s: expected a *ParseError (400), got %T: %v", c.query, err, err)
			continue
		}
		if !strings.Contains(err.Error(), "failed to parse "+c.what) {
			t.Errorf("%s: expected %q in the error, got %q", c.query, "failed to parse "+c.what, err.Error())
		}
	}

	// positive controls: complete parameters still parse
	good := []string{
		"?order=a",
		"?order=a.desc.nullslast,b.asc,c.nullsfirst",
		"?order=rel(a).desc",
		"?order=data->>k.desc",
		"?emb.order=a.asc",
		"?or=(a.eq.1,b.eq.2)",
		"?not.and=(a.eq.1,or(b.eq.2,c.eq.3))",
		"?emb.or=(a.eq.1)",
		"?or=(a.in.(1,2),b.eq(any).{1,2})",
		"?a=in.(1,2)",
		"?a=in.()",
		"?a=eq(any).{1,2}",
		"?a=eq.x)y",
		"?columns=a,b",
		"?columns=a, b",
		`?columns="a,b",c`,
		"?on_conflict=a,b",
		"?select=a,b:c::text,d",
		"?select=*",
		"?select=",
		"?select=a, b",
		"?select=x:a.sum()::int",
		"?select=id,rel(*)",
	}
	for _, q := range good {
		if _, err := parseQuery(t, q); err != nil {
			t.Errorf("%s: unexpected error: %v", q, err)
		}
	}
}

// The scanner kept the tokens left unread by one parameter ahead of the next
// one, so columns=a,,b,c&on_conflict=x gave columns {a, ","} and on_conflict
// {}. Each parameter must start from its own source.
func TestParserParametersDoNotShareTokens(t *testing.T) {
	p := &PostgRestParser{}
	// leave tokens unread, as a malformed parameter does
	p.scan("x)y,z", selectSeparators, selectLongSeparators...)
	p.next()
	conflicts, err := p.parseConflicts("k")
	if err != nil {
		t.Fatalf("on_conflict: unexpected error: %v", err)
	}
	if len(conflicts) != 1 || conflicts[0] != "k" {
		t.Errorf("on_conflict: got %q, want [k]", conflicts)
	}
	p.scan("x)y,z", selectSeparators, selectLongSeparators...)
	p.next()
	columns, err := p.parseColumns("a,b")
	if err != nil {
		t.Fatalf("columns: unexpected error: %v", err)
	}
	if strings.Join(columns, "|") != "a|b" {
		t.Errorf("columns: got %q, want [a b]", columns)
	}
	p.scan("x)y,z", ".,()", "->>", "->")
	p.next()
	fields, err := p.parseSelect("a,b")
	if err != nil {
		t.Fatalf("select: unexpected error: %v", err)
	}
	if len(fields) != 2 || fields[0].field.name != "a" || fields[1].field.name != "b" {
		t.Errorf("select: got %+v, want a, b", fields)
	}

	parts, err := parseQuery(t, "?select=a,b&columns=x,y&on_conflict=z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := parts.columnFields["x"]; !ok || len(parts.columnFields) != 2 {
		t.Errorf("columns: got %v, want {x y}", parts.columnFields)
	}
	if _, ok := parts.conflictFields["z"]; !ok || len(parts.conflictFields) != 1 {
		t.Errorf("on_conflict: got %v, want {z}", parts.conflictFields)
	}
}

// back() stepped back even when next() had returned the end of input without
// moving, so select=a: read the ':' as the field: "t".":" AS "a".
func TestParserBackAtEndOfInput(t *testing.T) {
	p := &PostgRestParser{}
	p.scan("a", ",")
	if tok := p.next(); tok != "a" {
		t.Fatalf("next: got %q, want a", tok)
	}
	if tok := p.next(); tok != "" {
		t.Fatalf("next at the end: got %q, want empty", tok)
	}
	p.back() // undoes the read at the end: the cursor stays after "a"
	if tok := p.next(); tok != "" {
		t.Errorf("back after the end stepped onto %q", tok)
	}
}
