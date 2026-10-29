package database

import (
	"context"
	"strings"
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
		{"no settings, batch", nil, true, withTxAccessMode, prepareLocal},
		{"no settings, autocommit", nil, false, withAccessMode, prepareAuto},
		{"no settings, kept", nil, false, withTxAccessMode, prepareKeep},
		{"transaction", two, true, withSwitch,
			"SELECT set_config($1, $2, true), set_config($3, $4, true), set_config('role', $5, true), set_config('request.jwt.claims', $6, true)"},
		{"batch", two, true, withTxAccessMode,
			"SELECT set_config($1, $2, true), set_config($3, $4, true), set_config('role', $5, true), set_config('request.jwt.claims', $6, true), set_config('transaction_read_only', $7, true)"},
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

// Database.SchemaSearchPath is set with SQL on every new connection, never
// as a startup parameter, which a pooler such as PgBouncer refuses by default
// (ignore_startup_parameters). The release of an autocommit connection
// restores it explicitly where it RESETs the other role settings: a RESET
// would go back to the server's default, not to the configured path.
func TestSchemaSearchPathIsSetWithSQL(t *testing.T) {
	saved := dbe.config.SchemaSearchPath
	dbe.config.SchemaSearchPath = []string{"public", "information_schema"}
	defer func() { dbe.config.SchemaSearchPath = saved }()

	ctx, conn, err := ContextWithDb(context.Background(), nil, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_search_path")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_search_path")
	ReleaseConn(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}
	pc, err := db.AcquireConnection(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var path, source string
	err = pc.QueryRow(context.Background(),
		"SELECT setting, source FROM pg_settings WHERE name = 'search_path'").Scan(&path, &source)
	pc.Release()
	if err != nil {
		t.Fatal(err)
	}
	if path != "public, information_schema" || source != "session" {
		t.Errorf("search_path of a pool connection: got %q from %q, want %q from \"session\"",
			path, source, "public, information_schema")
	}

	// the release puts the configured path back instead of RESETting it
	c, err := dbe.AcquireConnection(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Release()
	drop := "DROP ROLE IF EXISTS rs_path"
	for _, q := range []string{drop,
		"CREATE ROLE rs_path NOLOGIN",
		"GRANT rs_path TO CURRENT_USER",
		"ALTER ROLE rs_path SET search_path = pg_catalog",
	} {
		if _, err := c.Exec(context.Background(), q); err != nil {
			t.Fatalf("%q: %v", q, err)
		}
	}
	defer c.Exec(context.Background(), drop)
	rs, err := loadRoleSettings(context.Background(), c.Conn(), dbe.mainDb.info.Load().ServerVersion, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "; SELECT set_config('search_path', E'public, information_schema', false)"
	if strings.Contains(rs.reset, `RESET "search_path"`) || !strings.Contains(rs.reset, want) {
		t.Errorf("reset: got %q, want %q in place of RESET search_path", rs.reset, want)
	}
}

// A schema name with an apostrophe or a backslash, quoted as an identifier,
// is set as it is, whatever standard_conforming_strings says: the path is a
// SQL literal in set_config and in the release reset.
func TestSchemaSearchPathWithAnApostrophe(t *testing.T) {
	saved := dbe.config.SchemaSearchPath
	dbe.config.SchemaSearchPath = []string{"public", `"O'Reilly"`, `"a\b"`}
	defer func() { dbe.config.SchemaSearchPath = saved }()

	c, err := dbe.AcquireConnection(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Release()
	for _, conforming := range []string{"on", "off"} {
		// in a transaction rolled back, so the connection keeps its own path
		tx, err := c.Begin(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec(context.Background(), "SET LOCAL standard_conforming_strings = "+conforming); err != nil {
			t.Fatal(err)
		}
		var path string
		// a text of its own for each setting: a statement prepared under the
		// other one would be reused as it was parsed
		err = tx.QueryRow(context.Background(), "SELECT "+searchPathSet()+
			", current_setting('search_path') /* "+conforming+" */").Scan(nil, &path)
		tx.Rollback(context.Background())
		if err != nil {
			t.Fatalf("%s: %v", searchPathSet(), err)
		}
		if want := `public, "O'Reilly", "a\b"`; path != want {
			t.Errorf("standard_conforming_strings %s: got %q, want %q", conforming, path, want)
		}
	}
}
