package database

import (
	"context"
	"strings"
	"testing"
)

// The schema cache carries the PostgreSQL version so that version-gated
// features can fall back or refuse instead of letting the server raise a
// syntax error (card 32705 server-version). PostgREST reads it the same way,
// once per (re)connection, before the schema cache (Config/Database.hs
// queryPgVersion, AppState/Reload.hs qPgVersion).
func TestServerVersion(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)

	// Positive control: the version as the server reports it on this very
	// connection, read through the same query the cache is expected to use.
	var expected int
	err = GetConn(ctx).QueryRow(ctx, "SELECT current_setting('server_version_num')::int").Scan(&expected)
	if err != nil {
		t.Fatal(err)
	}
	if expected < 100000 {
		t.Fatalf("server_version_num %d is not a supported server", expected)
	}

	db := GetDb(ctx)

	t.Run("read at activation", func(t *testing.T) {
		info := db.info.Load()
		if info.ServerVersion != expected {
			t.Fatalf("ServerVersion = %d, want %d", info.ServerVersion, expected)
		}
	})

	t.Run("refreshed on schema reload", func(t *testing.T) {
		old := db.info.Load()
		if err := db.ReloadSchemaCache(ctx); err != nil {
			t.Fatal(err)
		}
		info := db.info.Load()
		if info == old {
			t.Fatal("ReloadSchemaCache did not replace the schema cache")
		}
		if info.ServerVersion != expected {
			t.Fatalf("ServerVersion after reload = %d, want %d", info.ServerVersion, expected)
		}
	})

	t.Run("ServerAtLeast against the live server", func(t *testing.T) {
		info := db.info.Load()
		major := expected / 10000
		if !info.ServerAtLeast(major) {
			t.Errorf("ServerAtLeast(%d) = false on server %d", major, expected)
		}
		if !info.ServerAtLeast(major - 1) {
			t.Errorf("ServerAtLeast(%d) = false on server %d", major-1, expected)
		}
		if info.ServerAtLeast(major + 1) {
			t.Errorf("ServerAtLeast(%d) = true on server %d", major+1, expected)
		}
	})

	t.Run("GetServerVersion", func(t *testing.T) {
		got, err := GetServerVersion(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got != expected {
			t.Fatalf("GetServerVersion = %d, want %d", got, expected)
		}
	})
}

func TestServerAtLeastBoundary(t *testing.T) {
	cases := []struct {
		version int
		major   int
		want    bool
	}{
		{160015, 16, true},  // same major, any minor
		{160015, 14, true},  // older major
		{160015, 17, false}, // next major
		{170000, 17, true},  // exact boundary: 17.0 is "at least 17"
		{169999, 17, false}, // one below the boundary
		{0, 1, false},       // unset (hand-built SchemaInfo): never "at least"
	}
	for _, c := range cases {
		si := &SchemaInfo{ServerVersion: c.version}
		if got := si.ServerAtLeast(c.major); got != c.want {
			t.Errorf("ServerVersion %d: ServerAtLeast(%d) = %v, want %v", c.version, c.major, got, c.want)
		}
	}
}

func TestFormatServerVersion(t *testing.T) {
	cases := map[int]string{
		160015: "16.15",
		140000: "14.0",
		90624:  "9.6.24", // pre-10 numbering: major.minor.patch
	}
	for num, want := range cases {
		if got := formatServerVersion(num); got != want {
			t.Errorf("formatServerVersion(%d) = %q, want %q", num, got, want)
		}
	}
}

// The startup refusal mirrors PostgREST's minimumPgVersion (Config/PgVersion.hs):
// an unsupported server is a clear error naming both versions, not a syntax
// error later or a silent exit.
func TestCheckServerVersion(t *testing.T) {
	if err := checkServerVersion(160015); err != nil {
		t.Errorf("16.15 refused: %v", err)
	}
	if err := checkServerVersion(MinServerVersion); err != nil {
		t.Errorf("the minimum itself refused: %v", err)
	}
	err := checkServerVersion(130016)
	if err == nil {
		t.Fatal("13.16 accepted, want a refusal")
	}
	for _, want := range []string{"13.16", formatServerVersion(MinServerVersion)} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal %q does not name %q", err, want)
		}
	}
}
