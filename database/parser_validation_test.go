package database

import (
	"net/url"
	"strings"
	"testing"
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
