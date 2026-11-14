package database

import "testing"

// TestFuzzySetMatchesPostgREST runs the doctests of noRpcHint and
// noRelBetweenHint (PostgREST 16.4, Error.hs) on the port of fuzzyset.
func TestFuzzySetMatchesPostgREST(t *testing.T) {
	cases := []struct {
		entries  []string
		value    string
		minScore float64
		want     string // "" for no match
	}{
		// noRpcHint, by function name (minimum score 0.75)
		{[]string{"test", "another"}, "testt", 0.75, "test"},
		{[]string{"test", "another"}, "other", 0.75, ""},
		{[]string{"test", "another"}, "noclosealternative", 0.75, ""},
		// noRpcHint, by argument names (minimum score 0.33)
		{[]string{"(name, param, val)", "(attr, id)"}, "(nam, pqaram, vall)", 0.33, "(name, param, val)"},
		{[]string{"(name, param, val)", "(attr, id)"}, "(param, val)", 0.33, "(name, param, val)"},
		{[]string{"(name, param, val)", "(attr, id)"}, "(attrs, id)", 0.33, "(attr, id)"},
		{[]string{"(name, param, val)", "(attr, id)"}, "(id)", 0.33, "(attr, id)"},
		{[]string{"(name, param, val)", "(attr, id)"}, "(noclosealternative)", 0.33, ""},
		// noRelBetweenHint
		{[]string{"films"}, "film", 0.33, "films"},
		{[]string{"directors", "roles", "actors"}, "role", 0.33, "roles"},
		{[]string{"films"}, "noclosealternative", 0.33, ""},
		{[]string{"directors", "roles", "actors"}, "noclosealternative", 0.33, ""},
	}
	for _, c := range cases {
		got, _ := newFuzzySet(c.entries).getOne(c.value, c.minScore)
		if got != c.want {
			t.Errorf("%v / %q: want %q, got %q", c.entries, c.value, c.want, got)
		}
	}
	// an exact match, ignoring case, scores 1
	if m := newFuzzySet([]string{"Actors"}).get("actors", 0.33); len(m) != 1 || m[0].score != 1 || m[0].entry != "Actors" {
		t.Errorf("exact match: got %v", m)
	}
}
