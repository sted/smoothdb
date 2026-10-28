package database

import (
	"net/url"
	"strings"
	"testing"
	"time"
)

// limit/offset are parameterized but their value was parsed with the error
// discarded, so a non-integer silently became LIMIT 0 (an empty result) instead
// of a 400. They must be rejected at parse time.
func TestParserRejectsInvalidLimitOffset(t *testing.T) {
	bad := []struct{ query, errSubstr string }{
		{"?limit=abc", "limit"},
		{"?limit=-1", "limit"},
		{"?limit=", "limit"},
		{"?offset=xyz", "offset"},
		{"?offset=-5", "offset"},
	}
	for _, c := range bad {
		u, _ := url.Parse(c.query)
		_, err := PostgRestParser{}.parse("t", u.Query())
		if err == nil {
			t.Errorf("%s: expected a parse error, got nil", c.query)
			continue
		}
		if !strings.Contains(err.Error(), c.errSubstr) {
			t.Errorf("%s: expected an error mentioning %q, got %q", c.query, c.errSubstr, err.Error())
		}
	}
	valid := []string{"?limit=10", "?offset=0", "?limit=0&offset=100"}
	for _, q := range valid {
		u, _ := url.Parse(q)
		if _, err := (PostgRestParser{}).parse("t", u.Query()); err != nil {
			t.Errorf("%s: unexpected error: %v", q, err)
		}
	}
}

// A deeply nested boolean filter recurses cond -> booleanOp -> cond; without a
// cap a crafted query string can overflow the goroutine stack (a cheap DoS). The
// parser must refuse excessive nesting rather than crash.
func TestParserRejectsDeeplyNestedFilters(t *testing.T) {
	body := "id.eq.1"
	for i := 0; i < 300; i++ {
		body = "and(" + body + ")"
	}
	q := "?and=" + body[len("and"):] // key "and" re-prepends the leading "and"
	u, _ := url.Parse(q)
	_, err := PostgRestParser{}.parse("t", u.Query())
	if err == nil {
		t.Fatal("expected a parse error for deeply nested filters, got nil")
	}
	if !strings.Contains(err.Error(), "nest") {
		t.Errorf("expected a nesting-depth error, got %q", err.Error())
	}
}

// '*' is the URL-friendly stand-in for the LIKE/ILIKE wildcard '%'. It must be
// rewritten only for those operators and stay literal everywhere else.
func TestLikeWildcardScoping(t *testing.T) {
	if _, vals := buildFromQuery(t, "?name=like.a*b"); len(vals) != 1 || vals[0] != "a%b" {
		t.Errorf("like: expected [a%%b], got %v", vals)
	}
	if _, vals := buildFromQuery(t, "?name=ilike.a*b"); len(vals) != 1 || vals[0] != "a%b" {
		t.Errorf("ilike: expected [a%%b], got %v", vals)
	}
	if _, vals := buildFromQuery(t, "?name=eq.a*b"); len(vals) != 1 || vals[0] != "a*b" {
		t.Errorf("eq: expected [a*b] (literal), got %v", vals)
	}
}

// The source segment of the URL is quoted by splitting on dots (the schema is
// prepended and each part quoted), so a dotted name such as
// doc.derived.member_choices reached Postgres as
// "sources"."doc"."derived"."member_choices" and failed with 42601 (too many
// dotted names) as a 500 — after a prepare, a query and a rollback. A name
// that is not a single identifier must be refused by the parser, before SQL.
func TestParserRejectsDottedSourceName(t *testing.T) {
	bad := []string{"doc.derived.member_choices", "schema.table", "a.", ".a", "."}
	for _, name := range bad {
		_, err := PostgRestParser{}.parse(name, url.Values{})
		if err == nil {
			t.Errorf("%q: expected a parse error, got nil", name)
			continue
		}
		if _, ok := err.(*ParseError); !ok {
			t.Errorf("%q: expected a *ParseError (400), got %T: %v", name, err, err)
		}
		if !strings.Contains(err.Error(), name) {
			t.Errorf("%q: expected the error to name the source, got %q", name, err.Error())
		}
	}
	for _, name := range []string{"member_choices", "MixedCase", "with space", "tab-le"} {
		if _, err := (PostgRestParser{}).parse(name, url.Values{}); err != nil {
			t.Errorf("%q: unexpected error: %v", name, err)
		}
	}
}

// PostgREST parses the operand of `is` as a grammar token, not as a value: only
// null, not_null, true, false and unknown are accepted, case-insensitively, and
// never in quotes. A quoted operand used to skip the keyword check and reach
// Postgres as `IS foo` (42601 → 500); it must be a 400 like the bare form.
func TestParserRejectsNonKeywordIsOperand(t *testing.T) {
	bad := []string{
		`?a=is.foo`,
		`?a=is."foo"`,
		`?a=is."null"`,
		`?a=not.is."true"`,
		`?data->>k=is."null"`,
		`?a=is.`,
	}
	for _, q := range bad {
		u, _ := url.Parse(q)
		_, err := PostgRestParser{}.parse("t", u.Query())
		if err == nil {
			t.Errorf("%s: expected a parse error, got nil", q)
			continue
		}
		if _, ok := err.(*ParseError); !ok {
			t.Errorf("%s: expected a *ParseError, got %T: %v", q, err, err)
		}
	}
	valid := []string{`?a=is.null`, `?a=is.NULL`, `?a=is.Not_Null`, `?a=not.is.true`, `?a=is.unknown`, `?data->>k=is.false`}
	for _, q := range valid {
		u, _ := url.Parse(q)
		if _, err := (PostgRestParser{}).parse("t", u.Query()); err != nil {
			t.Errorf("%s: unexpected error: %v", q, err)
		}
	}
}

// A via! filter value without the opening parenthesis sliced the value at
// IndexByte(v, '(') == -1 and panicked; it must be a parse error.
func TestParserViaWithoutParenthesis(t *testing.T) {
	for _, q := range []string{"?x=via!)", "?x=via!x)", "?x=via!both)"} {
		u, _ := url.Parse(q)
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s: parser panicked: %v", q, r)
				}
			}()
			_, err = PostgRestParser{}.parse("t", u.Query())
		}()
		if err == nil {
			t.Errorf("%s: expected a parse error, got nil", q)
		}
	}
}

// A list value that ends inside a quoted token makes the parser rescan the
// rest of the source; done for every element, that rescan was quadratic in the
// length of the list (16k elements took seconds of CPU). The parse must stay
// linear: 20k elements well under a second, even under -race.
func TestParserListRescanIsLinear(t *testing.T) {
	const n = 20000
	q := "?a=in.(" + strings.Repeat(`x.",`, n) + "x)"
	u, err := url.Parse(q)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	parts, err := PostgRestParser{}.parse("t", u.Query())
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	node := parts.whereConditionsTree.children[0]
	if len(node.values) != n+1 || node.values[0] != `x."` || node.values[n] != "x" {
		t.Fatalf("expected %d values x.\" ... x, got %d: first %q, last %q",
			n+1, len(node.values), node.values[0], node.values[len(node.values)-1])
	}
	if elapsed > time.Second {
		t.Errorf("parsing %d list elements took %v", n, elapsed)
	}
}

// not, start, after, recurse and via are in the operator table with an empty
// SQL operator, to be recognized as keywords. Reaching a condition as its
// operator (inside a logic tree, or after not.) they made a node with no
// operator, which the WHERE builder takes for a container: the condition
// vanished (or=(id.start.1,b.eq.2) became WHERE b = $1, and a DELETE deleted
// more) or left an empty NOT (). They must be parse errors.
func TestParserRejectsKeywordsAsOperators(t *testing.T) {
	bad := []string{
		"?or=(id.start.1,b.eq.2)",
		"?or=(id.after.1,b.eq.2)",
		"?and=(id.recurse.1,b.eq.2)",
		"?or=(id.via.1,b.eq.2)",
		"?id=not.not.5",
		"?id=not.after.3",
		"?or=(id.not.start.1,b.eq.2)",
	}
	for _, q := range bad {
		u, _ := url.Parse(q)
		_, err := PostgRestParser{}.parse("t", u.Query())
		if err == nil {
			t.Errorf("%s: expected a parse error, got nil", q)
		} else if !strings.Contains(err.Error(), "operator") {
			t.Errorf("%s: expected an operator error, got %q", q, err.Error())
		}
	}
	// controls: real operators, negated or in a logic tree
	for _, q := range []string{"?or=(id.eq.1,b.eq.2)", "?id=not.eq.5", "?or=(id.not.in.(1,2),b.eq.2)"} {
		u, _ := url.Parse(q)
		if _, err := (PostgRestParser{}).parse("t", u.Query()); err != nil {
			t.Errorf("%s: unexpected error: %v", q, err)
		}
	}
}

// On GET /rpc every parameter that is neither reserved nor an operator filter
// becomes a function argument. Only the bare keys were reserved, and two of them
// were misspelled ("column", "on_conlict"), so an embedded order such as
// steps.order=id.asc reached the call as an argument named "steps.order": no
// overload matched and the embed failed with an empty table name. PostgREST
// reserves select, columns and on_conflict, and every key whose last dotted
// word is order, limit, offset, and or or (QueryParams.hs, isFilter).
func TestParserRPCParametersSkipReservedKeys(t *testing.T) {
	q := "?a=1&b=x.y" +
		"&select=*,steps(*)&columns=a,b&on_conflict=a" +
		"&order=id&limit=1&offset=2&or=(a.eq.1)&and=(a.eq.1)&not.or=(a.eq.1)&not.and=(a.eq.1)" +
		"&steps.order=id.asc&steps.limit=5&steps.offset=1" +
		"&steps.or=(id.eq.1)&steps.and=(id.eq.1)&steps.not.or=(id.eq.1)&steps.not.and=(id.eq.1)" +
		"&steps.sub.order=id.desc&c=eq.3"
	u, err := url.Parse(q)
	if err != nil {
		t.Fatal(err)
	}
	filters := Filters(u.Query())
	params := PostgRestParser{}.filterParameters(filters)
	// positive control: plain arguments stay arguments, whatever their value
	for _, k := range []string{"a", "b"} {
		if _, ok := params[k]; !ok {
			t.Errorf("%q should be a function argument, got arguments %v", k, params)
		}
	}
	for k := range params {
		if k != "a" && k != "b" {
			t.Errorf("%q is a reserved parameter, not a function argument", k)
		}
	}
	// the reserved keys and the operator filter stay for the parser
	for _, k := range []string{"select", "columns", "on_conflict", "steps.order", "steps.not.or", "c"} {
		if _, ok := filters[k]; !ok {
			t.Errorf("%q should remain in the filters", k)
		}
	}
}

// An embed takes a hint and a join modifier together (rel!hint!inner), as in
// PostgREST. The parser consumed one token after '!', so the second '!' ended
// the item, clients became a plain field and, since nothing checked that the
// select was consumed, the rest of it was dropped without an error.
func TestParserSelectHintAndModifier(t *testing.T) {
	type embed struct {
		name, fk string
		inner    bool
	}
	good := []struct {
		sel    string
		fields int // top-level items
		embed  embed
	}{
		{"id,clients!client!inner(id),name", 3, embed{"clients", "client", true}},
		{"id,clients!client!left(id)", 2, embed{"clients", "client", false}},
		// PostgREST's pEmbedParams reads up to two parameters in either
		// order; the hint is the first hint, the join the first modifier, and
		// a second hint or a second modifier is ignored
		{"id,clients!inner!client(id),name", 3, embed{"clients", "client", true}},
		{"id,clients!left!client(id)", 2, embed{"clients", "client", false}},
		{"id,clients!client!bogus(id)", 2, embed{"clients", "client", false}},
		{"id,clients!bogus!client(id)", 2, embed{"clients", "bogus", false}},
		{"id,clients!inner!left(id)", 2, embed{"clients", "", true}},
		{"id,clients!left!inner(id)", 2, embed{"clients", "", false}},
		{"id,clients!inner!inner(id)", 2, embed{"clients", "", true}},
		{`id,clients!"inner"(id)`, 2, embed{"clients", "inner", false}}, // a quoted name is a hint
		// controls: one embed parameter, as before
		{"id,clients!inner(id)", 2, embed{"clients", "", true}},
		{"id,clients!left(id)", 2, embed{"clients", "", false}},
		{"id,clients!client(id)", 2, embed{"clients", "client", false}},
		{"id,clients(id)", 2, embed{"clients", "", false}},
	}
	for _, c := range good {
		p := PostgRestParser{}
		fields, err := p.parseSelect(c.sel)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.sel, err)
			continue
		}
		if len(fields) != c.fields {
			t.Errorf("%s: %d select items, want %d", c.sel, len(fields), c.fields)
			continue
		}
		rel := fields[1].relation
		if rel == nil {
			t.Errorf("%s: item 1 is not an embed: %+v", c.sel, fields[1])
			continue
		}
		if got := (embed{rel.name, rel.fk, rel.inner}); got != c.embed {
			t.Errorf("%s: embed %+v, want %+v", c.sel, got, c.embed)
		}
	}
	// trailing input and malformed embed parameters are parse errors, not a
	// truncated select (card 35452 parser-trailing for the first two); a
	// third embed parameter fails in PostgREST too
	bad := []string{
		"a)b",
		"a!(b)",
		"id,clients!client!inner!x(id)",
		"id,clients!inner!client!left(id)",
		"id,clients!client!(id)",
		"id,clients!(id)",
		"id,clients!(id),name",
		"a,b(c))",
	}
	for _, sel := range bad {
		p := PostgRestParser{}
		if fields, err := p.parseSelect(sel); err == nil {
			t.Errorf("%s: expected a parse error, got %d items", sel, len(fields))
		}
	}
}

// A schema-qualified cast arrives as three tokens (the dot separates in a
// select): only the schema was read as the type and the rest was dropped
// silently, so t::public.mytype cast to a type named public. The cast must
// carry the whole name, and a cast followed by an aggregate (t::int4.sum())
// must keep working.
func TestParserSchemaQualifiedCast(t *testing.T) {
	cases := map[string]string{
		"?select=t::public.mytype":   `SELECT "table"."t"::public.mytype FROM "table"`,
		"?select=x:t::public.mytype": `SELECT "table"."t"::public.mytype AS "x" FROM "table"`,
		"?select=t::int4":            `SELECT "table"."t"::int4 FROM "table"`,
	}
	for q, want := range cases {
		u, _ := url.Parse(q)
		parts, err := PostgRestParser{}.parse("table", u.Query())
		if err != nil {
			t.Errorf("%s: unexpected parse error: %v", q, err)
			continue
		}
		sql, _, err := DirectQueryBuilder{}.BuildSelect("table", parts, &QueryOptions{}, nil)
		if err != nil {
			t.Errorf("%s: unexpected build error: %v", q, err)
			continue
		}
		if sql != want {
			t.Errorf("%s\n want: %s\n  got: %s", q, want, sql)
		}
	}
}
