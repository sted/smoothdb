package database

import (
	"strings"
	"testing"
)

// A quoted empty name ("") names no column. The parser took it as a field
// name, and the builder took a node with an empty field name for a boolean
// node: the filter vanished, and DELETE /t?""=eq.1 deleted every row. The
// parser refuses the name, as it refused an empty field before.
func TestParserRefusesAnEmptyFieldName(t *testing.T) {
	for _, q := range []string{
		`?%22%22=eq.1`,
		`?%22%22.x=eq.1`,
		`?or=(%22%22.eq.1)`,
		`?and=(a.eq.1,%22%22.eq.2)`,
		`?order=%22%22`,
	} {
		if _, err := parseQuery(t, q); err == nil {
			t.Errorf("%s: expected a parse error", q)
		}
	}
	// positive control: a quoted name that is not empty is a column
	if _, err := parseQuery(t, `?%22a.b%22=eq.1`); err != nil {
		t.Errorf(`"a.b": unexpected error: %v`, err)
	}
}

// The builder decides that a node holds conditions by its operator, never by
// an empty field name: a filter whose field name is empty is still a
// condition and keeps its WHERE.
func TestBuilderKeepsAFilterWithAnEmptyFieldName(t *testing.T) {
	parts, err := parseQuery(t, "?a=eq.1")
	if err != nil {
		t.Fatal(err)
	}
	parts.whereConditionsTree.children[0].field.name = ""
	sql, _, err := CommonBuilder{}.BuildDelete("t", parts, &QueryOptions{}, nil)
	if err == nil && !strings.Contains(sql, "WHERE") {
		t.Errorf("a filter with an empty field name lost its WHERE: %s", sql)
	}
}

// PostgreSQL's array parser ignores only ASCII whitespace around an element
// (arrayfuncs.c): a no-break space before null keeps the element a string.
func TestAnyListNullWithUnicodeSpaceIsAString(t *testing.T) {
	parts, err := parseQuery(t, "?name=eq(any).{a,%C2%A0null}")
	if err != nil {
		t.Fatal(err)
	}
	_, values, err := DirectQueryBuilder{}.BuildSelect("t", parts, &QueryOptions{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[1] != "\u00a0null" {
		t.Errorf("values %#v, want [a \"\\u00a0null\"]", values)
	}
}

// A write took only the filters of its own table into the WHERE: a filter on
// an embed, which a write without a representation never has, was dropped,
// and DELETE /t?x.id=eq.1 deleted every row. The recursive filters (start.,
// recurse.) were dropped the same way. Both are errors now, as PGRST108 in
// PostgREST for the first.
func TestWritesRefuseFiltersTheyDoNotApply(t *testing.T) {
	for _, q := range []string{
		`?x.id=eq.1`,
		`?x.%22%22=eq.1`,
		`?%22%22.or=(id.eq.1)`,
		`?x.or=(id.eq.1)`,
		`?id=eq.1&x.id=eq.2`,
		`?id=start.1&parent_id=recurse.all`,
	} {
		parts, err := parseQuery(t, q)
		if err != nil {
			continue // refused by the parser: safe
		}
		if sql, _, err := (CommonBuilder{}).BuildDelete("t", parts, &QueryOptions{}, nil); err == nil {
			t.Errorf("%s: DELETE built %s, want an error", q, sql)
		}
		parts, _ = parseQuery(t, q)
		if sql, _, err := (CommonBuilder{}).BuildUpdate("t", Record{"a": 1}, parts, &QueryOptions{}, nil); err == nil {
			t.Errorf("%s: UPDATE built %s, want an error", q, sql)
		}
	}
	// positive control: a filter on the table itself
	parts, _ := parseQuery(t, "?id=eq.1")
	if sql, _, err := (CommonBuilder{}).BuildDelete("t", parts, &QueryOptions{}, nil); err != nil || !strings.Contains(sql, "WHERE") {
		t.Errorf("?id=eq.1: %s, %v", sql, err)
	}
}

// select="" was read as the end of the list, so a PATCH with
// return=representation and select="" returned every column; rel("") was an
// empty embed.
func TestSelectRefusesAnEmptyName(t *testing.T) {
	for _, q := range []string{`?select=%22%22`, `?select=id,%22%22`, `?select=rel(%22%22)`, `?select=%22%22:id`} {
		if _, err := parseQuery(t, q); err == nil {
			t.Errorf("%s: expected a parse error", q)
		}
	}
	if _, err := parseQuery(t, `?select=%22a%20b%22`); err != nil {
		t.Errorf(`select="a b": unexpected error: %v`, err)
	}
}

// A call took the recursive filters (start., recurse.) out of its filters and
// ignored them: it returned every row and ran a VOLATILE function anyway.
func TestCallRefusesRecursiveFilters(t *testing.T) {
	parts, err := parseQuery(t, "?id=start.1&parent_id=recurse.all")
	if err != nil {
		t.Fatal(err)
	}
	if sql, _, err := (CommonBuilder{}).BuildExecute("f", nil, Record{}, parts, &QueryOptions{}, nil); err == nil {
		t.Errorf("BuildExecute built %s, want an error", sql)
	}
}
