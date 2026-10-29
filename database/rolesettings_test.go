package database

import (
	"context"
	"testing"
)

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

// A role's DateStyle keeps its field order but not its output style: the
// text of a timestamp the serializers convert (a field of a composite that
// arrives in text) is taken for ISO, the style the connections are opened
// with, and PostgREST's JSON is ISO whatever the role's DateStyle.
func TestRoleDateStyleKeepsISOOutput(t *testing.T) {
	ctx := context.Background()
	conn, err := dbe.AcquireConnection(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	drop := "DROP ROLE IF EXISTS rs_dmy"
	for _, q := range []string{drop,
		"CREATE ROLE rs_dmy NOLOGIN",
		"GRANT rs_dmy TO CURRENT_USER",
		"ALTER ROLE rs_dmy SET DateStyle = 'SQL, DMY'",
		"ALTER ROLE rs_dmy SET work_mem = '7MB'",
	} {
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("%q: %v", q, err)
		}
	}
	defer conn.Exec(ctx, drop)
	rs, err := loadRoleSettings(ctx, conn.Conn(), dbe.mainDb.info.Load().ServerVersion, nil)
	if err != nil {
		t.Fatal(err)
	}
	rc := rs.roles["rs_dmy"]
	if rc == nil {
		t.Fatal("no settings loaded for rs_dmy")
	}
	want := []any{"DateStyle", "ISO, DMY", "work_mem", "7MB"}
	for name, args := range map[string][]any{"transaction": rc.txArgs, "autocommit": rc.autoArgs} {
		if len(args) != len(want) {
			t.Errorf("%s: got %v, want %v", name, args, want)
			continue
		}
		for i := range want {
			if args[i] != want[i] {
				t.Errorf("%s: got %v, want %v", name, args, want)
				break
			}
		}
	}
}
