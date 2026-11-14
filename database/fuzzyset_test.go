package database

import (
	"strings"
	"testing"
)

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
	// a value longer than any hint can be is not compared at all
	long := strings.Repeat("a", 1<<20)
	if m := newFuzzySet([]string{"(a, b)", strings.Repeat("a", 200)}).get(long, 0.33); m != nil {
		t.Errorf("1 MiB value: want no match, got %d", len(m))
	}
	// an exact match, ignoring case, scores 1
	if m := newFuzzySet([]string{"Actors"}).get("actors", 0.33); len(m) != 1 || m[0].score != 1 || m[0].entry != "Actors" {
		t.Errorf("exact match: got %v", m)
	}
}

// The hint candidates are the objects of PostgREST's schema cache: functions
// only (prokind 'f'), no trigger function, and no partition among the tables.
func TestHintCandidatesMatchPostgREST(t *testing.T) {
	arg := Argument{Name: "a", Type: "integer", TypeId: 23}
	idx := indexFunctions([]Function{
		{Name: "f", Schema: "s", Kind: "f", Arguments: []Argument{arg}},
		{Name: "p", Schema: "s", Kind: "p", Arguments: []Argument{arg}},
		{Name: "agg", Schema: "s", Kind: "a", Arguments: []Argument{arg}},
		{Name: "trg", Schema: "s", Kind: "f", Returns: "trigger"},
	})
	if got := idx.callable["s"]; len(got) != 1 || got[0] != "f" {
		t.Errorf("callable functions: want [f], got %v", got)
	}
	names := indexTables(map[string]Table{
		"s.t":    {Name: "t", Schema: "s"},
		"s.t_p1": {Name: "t_p1", Schema: "s", IsPartition: true},
		"o.u":    {Name: "u", Schema: "o"},
	})
	if got := names["s"]; len(got) != 1 || got[0] != "t" {
		t.Errorf("tables of s: want [t], got %v", got)
	}
}
