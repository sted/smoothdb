package postgrest

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"

	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/server"
	"github.com/sted/smoothdb/test"
)

var testConfig test.Config

// PostgRESTVersion is the PostgREST release this port targets: its spec
// files (test/spec in the PostgREST repository, at the tag v<version>) are the
// source of the cases here, and its behaviour, run live, is the reference
// where a card and the spec disagree. See README.md in this directory.
const PostgRESTVersion = "16.4"

// suiteConfig is the configuration of the suite server. A variant server
// (see withVariant) starts from it and changes some keys. The servers ignore
// the SMOOTHDB_ environment variables (see startServer), except
// SMOOTHDB_DATABASE_URL, which sets Database.URL here.
func suiteConfig() map[string]any {
	dbURL := os.Getenv("SMOOTHDB_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres:postgres@localhost:5432/postgres"
	}
	return map[string]any{
		"Address":                   "localhost:8084",
		"VerboseErrors":             true, // the suites assert hint/details in error bodies
		"JWTSecret":                 "reallyreallyreallyreallyverysafe",
		"AllowAnon":                 false,
		"EnableAdminRoute":          true,
		"Logging.Level":             "info",
		"Logging.FileLogging":       true,
		"Logging.FilePath":          "../../smoothdb.log",
		"Logging.StdOut":            false,
		"Database.URL":              dbURL,
		"Database.SchemaSearchPath": []string{"test"},
		// the schemas a Profile header may select (PostgREST db-schemas); تست
		// is exposed but not on the search path, as UnicodeSpec runs upstream
		"Database.ExposedSchemas":  []string{"test", "تست"},
		"Database.TransactionMode": "rollback-allow-override",
		// the anonymous role the suite's token names, as db-anon-role in
		// PostgREST's test config: 42501 is 401 for it and 403 for the others
		"Database.AnonRole": "postgrest_test_anonymous",
	}
}

// suiteServer is the running suite server. withVariant stops it and starts
// it again.
var suiteServer *server.Server

// startServer starts a server with the configuration c and waits until it
// answers. The environment does not change c: a variant gets exactly its
// overrides. The test binds the address itself and the server serves on
// that listener (Start does the same with ListenAndServe), so no other
// process can answer in place of the new server.
func startServer(c map[string]any) (*server.Server, error) {
	addr := c["Address"].(string)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("the address of the server is not free: %w", err)
	}
	s, err := server.NewServerWithConfig(c,
		&server.ConfigOptions{
			ConfigFilePath: "../../config.jsonc",

			SkipFlags: true,
			SkipEnv:   true,
		})
	if err != nil {
		l.Close()
		return nil, err
	}
	go s.HTTP.Serve(l)
	if err := test.WaitForServer("http://" + addr); err != nil {
		s.Shutdown(context.Background())
		return nil, err
	}
	return s, nil
}

func TestMain(m *testing.M) {
	s, err := startServer(suiteConfig())
	if err != nil {
		log.Fatal(err)
	}
	suiteServer = s

	postgresToken, _ := authn.GenerateToken("postgres", s.JWTSecret())
	anonymousToken, _ := authn.GenerateToken("postgrest_test_anonymous", s.JWTSecret())

	testConfig = test.Config{
		BaseUrl: "http://localhost:8084/api/pgrest",
		CommonHeaders: test.Headers{
			"Authorization":   {anonymousToken},
			"Accept-Profile":  {"test"},
			"Content-Profile": {"test"},
		},
	}

	// Tear-up

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8084/admin",
		CommonHeaders: test.Headers{"Authorization": {postgresToken}},
	}

	commands := []test.Command{
		// create role postgrest_test_anonymous
		{
			Method: "POST",
			Query:  "/users",
			Body: `{
				"name": "postgrest_test_anonymous"
			}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	// Run
	code := m.Run()

	// The compatibility number: the ported PostgREST cases smoothdb passes,
	// against those skipped with a reason. With -run it covers the cases run.
	run, skip := test.Counts()
	if run+skip > 0 {
		fmt.Printf("PostgREST %s spec cases: %d run, %d skipped (%.1f%% run)\n",
			PostgRESTVersion, run, skip, 100*float64(run)/float64(run+skip))
	}

	os.Exit(code)
}
