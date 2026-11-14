package postgrest

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sted/smoothdb/test"
)

// variantAddress is where a variant server listens.
const variantAddress = "localhost:8180"

// withVariant runs fn against a server with the suite configuration, changed
// by overrides. fn gets testConfig, pointed at the variant server.
//
// The database package keeps its engine in a package variable: a second
// server in the same process replaces the engine of the first. So the helper
// never runs two servers at a time. It stops the suite server, starts the
// variant, runs fn, stops the variant and starts the suite server again.
//
// With copyDB, the variant serves pgrest_variant, a copy of the fixture
// database pgrest, and the helper drops the copy at the end. Use it when the
// variant commits writes: the other tests read pgrest and expect the
// fixtures as loaded. CREATE DATABASE ... TEMPLATE needs a template without
// connections, and the stopped suite server holds none.
func withVariant(t *testing.T, overrides map[string]any, copyDB bool, fn func(cfg test.Config)) {
	t.Helper()
	ctx := context.Background()
	suiteServer.Shutdown(ctx)
	defer func() {
		s, err := startServer(suiteConfig())
		if err != nil {
			t.Fatalf("restart the suite server: %v", err)
		}
		suiteServer = s
	}()

	db := "pgrest"
	if copyDB {
		db = "pgrest_variant"
		adminExec(t, "DROP DATABASE IF EXISTS pgrest_variant WITH (FORCE)")
		adminExec(t, "CREATE DATABASE pgrest_variant TEMPLATE pgrest")
		defer adminExec(t, "DROP DATABASE pgrest_variant WITH (FORCE)")
	}

	c := suiteConfig()
	c["Address"] = variantAddress
	for k, v := range overrides {
		c[k] = v
	}
	s, err := startServer(c)
	if err != nil {
		t.Fatalf("start the variant server: %v", err)
	}
	defer s.Shutdown(ctx)

	cfg := testConfig
	cfg.BaseUrl = "http://" + variantAddress + "/api/" + db
	fn(cfg)
}

// adminExec runs sql on the main database of the suite, as postgres.
func adminExec(t *testing.T, sql string) {
	t.Helper()
	url := os.Getenv("SMOOTHDB_DATABASE_URL")
	if url == "" {
		url = suiteConfig()["Database.URL"].(string)
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect to %s: %v", url, err)
	}
	defer conn.Close(ctx)
	if _, err := conn.Exec(ctx, sql); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}
