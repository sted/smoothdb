package postgrest

import (
	"log"
	"os"
	"testing"

	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/server"
	"github.com/sted/smoothdb/test"
)

var testConfig test.Config

func TestMain(m *testing.M) {
	c := map[string]any{
		"Address":                   "localhost:8082",
		"JWTSecret":                 "reallyreallyreallyreallyverysafe",
		"AllowAnon":                 false,
		"EnableAdminRoute":          true,
		"Logging.Level":             "info",
		"Logging.FileLogging":       true,
		"Logging.FilePath":          "../../smoothdb.log",
		"Logging.StdOut":            false,
		"Database.URL":              "postgresql://postgres:postgres@localhost:5432/postgres",
		"Database.SchemaSearchPath": []string{"test"},
		"Database.TransactionMode":  "rollback-allow-override",
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
		BaseUrl: "http://localhost:8082/api/pgrest",
		CommonHeaders: test.Headers{
			"Authorization":   {anonymousToken},
			"Accept-Profile":  {"test"},
			"Content-Profile": {"test"},
		},
	}

	go s.Start()

	// Tear-up

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin",
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

	os.Exit(code)
}
