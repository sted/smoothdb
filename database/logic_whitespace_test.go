package database

import "testing"

// PostgREST 16.4 (QueryParams.hs pLogicTree) reads the parenthesis of a logic
// operator with lexeme, so spaces can come between "and" or "or" and its "(".
// AndOrParamsSpec "allows whitespace" sends such a tree. A tree with spaces
// must build the SQL of the same tree without them.
func TestLogicTreeWhitespace(t *testing.T) {
	tests := []struct {
		spaced  string
		compact string
	}{
		{"?and=(%20and%20(%20id.in.(%201,%202,%203%20)%20,%20id.eq.3%20)%20,%20or%20(%20id.eq.2%20,%20id.eq.3%20)%20)",
			"?and=(and(id.in.(1,2,3),id.eq.3),or(id.eq.2,id.eq.3))"},
		{"?or=(and%20(a.eq.1,b.eq.2),c.eq.3)", "?or=(and(a.eq.1,b.eq.2),c.eq.3)"},
		{"?or=(not.and%20%20(a.eq.1,b.eq.2),c.eq.3)", "?or=(not.and(a.eq.1,b.eq.2),c.eq.3)"},
		{"?not.or=(a.eq.1,and%20(b.eq.2,c.eq.3))", "?not.or=(a.eq.1,and(b.eq.2,c.eq.3))"},
	}
	for _, test := range tests {
		got, _ := buildFromQuery(t, test.spaced)
		want, _ := buildFromQuery(t, test.compact)
		if got != want {
			t.Errorf("%s\n want: %s\n  got: %s", test.spaced, want, got)
		}
	}
}
