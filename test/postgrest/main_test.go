package postgrest

import (
	"fmt"
	"log"
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

func TestMain(m *testing.M) {
	c := map[string]any{
		"Address":                   "localhost:8084",
		"VerboseErrors":             true, // the suites assert hint/details in error bodies
		"JWTSecret":                 "reallyreallyreallyreallyverysafe",
		"AllowAnon":                 false,
		"EnableAdminRoute":          true,
		"Logging.Level":             "info",
		"Logging.FileLogging":       true,
		"Logging.FilePath":          "../../smoothdb.log",
		"Logging.StdOut":            false,
		"Database.URL":              "postgresql://postgres:postgres@localhost:5432/postgres",
		"Database.SchemaSearchPath": []string{"test"},
		// the schemas a Profile header may select (PostgREST db-schemas); تست
		// is exposed but not on the search path, as UnicodeSpec runs upstream
		"Database.ExposedSchemas":  []string{"test", "تست"},
		"Database.TransactionMode": "rollback-allow-override",
		// the anonymous role the suite's token names, as db-anon-role in
		// PostgREST's test config: 42501 is 401 for it and 403 for the others
		"Database.AnonRole": "postgrest_test_anonymous",
	}
	s, err := server.NewServerWithConfig(c,
		&server.ConfigOptions{
			ConfigFilePath: "../../config.jsonc",

			SkipFlags: true,
		})
	if err != nil {
		log.Fatal(err)
	}

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

	go s.Start()
	if err := test.WaitForServer("http://localhost:8084"); err != nil {
		log.Fatal(err)
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
