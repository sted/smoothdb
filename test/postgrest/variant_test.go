package postgrest

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sted/smoothdb/test"
)

// variantAddress is where a variant server listens. It names 127.0.0.1, not
// localhost, so that the bind, the probe and the client use the same address.
const variantAddress = "127.0.0.1:8180"

// variantDBs counts the copies of pgrest that withVariant made.
var variantDBs int

// withVariant runs fn against a server with the suite configuration, changed
// by overrides. fn gets testConfig, pointed at the variant server.
//
// The database package keeps its engine in a package variable: a second
// server in the same process replaces the engine of the first. So the helper
// never runs two servers at a time. It stops the suite server, starts the
// variant, runs fn, stops the variant and starts the suite server again.
//
// With copyDB, the variant serves a copy of the fixture database pgrest,
// with a name unique to the process and the call. The helper drops only that
// copy at the end. Use it when the
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
		variantDBs++
		db = fmt.Sprintf("pgrest_variant_%d_%d", os.Getpid(), variantDBs)
		adminExec(t, "CREATE DATABASE "+db+" TEMPLATE pgrest")
		defer adminExec(t, "DROP DATABASE "+db+" WITH (FORCE)")
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
	url := suiteConfig()["Database.URL"].(string)
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
