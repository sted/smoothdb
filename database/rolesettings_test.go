package database

import "testing"

// The preparation statement puts the role's settings first, then the switch
// and the claims, with one placeholder per argument; without settings it is
// the constant one.
func TestPrepareStatement(t *testing.T) {
	two := []any{"statement_timeout", "1s", "work_mem", "8MB"}
	tests := []struct {
		name  string
		args  []any
		local bool
		end   int
		want  string
	}{
		{"no settings, transaction", nil, true, withSwitch, prepareTX},
		{"no settings, autocommit", nil, false, withAccessMode, prepareAuto},
		{"transaction", two, true, withSwitch,
			"SELECT set_config($1, $2, true), set_config($3, $4, true), set_config('role', $5, false), set_config('request.jwt.claims', $6, false)"},
		{"autocommit", two, false, withAccessMode,
			"SELECT set_config($1, $2, false), set_config($3, $4, false), set_config('role', $5, false), set_config('request.jwt.claims', $6, false), set_config('default_transaction_read_only', $7, false)"},
		{"settings only", two, true, settingsOnly,
			"SELECT set_config($1, $2, true), set_config($3, $4, true)"},
	}
	for _, tt := range tests {
		if got := prepareStatement(tt.args, tt.local, tt.end); got != tt.want {
			t.Errorf("%s:\n\tgot  %s\n\twant %s", tt.name, got, tt.want)
		}
	}
}

func TestIsolationLevel(t *testing.T) {
	for value, want := range map[string]string{
		"serializable": "SERIALIZABLE", "Repeatable Read": "REPEATABLE READ",
		"read committed": "", "read uncommitted": "",
	} {
		if got := isolationLevel(value); got != want {
			t.Errorf("%q: got %q, want %q", value, got, want)
		}
	}
}
