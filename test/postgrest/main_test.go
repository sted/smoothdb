package postgrest

import (
	"log"
	"os"
	"testing"

	"github.com/sted/smoothdb/server"
	"github.com/sted/smoothdb/test"
)

var testConfig test.Config

func TestMain(m *testing.M) {
	c := map[string]any{
		"Address":                   "localhost:8082",
		"AllowAnon":                 false,
		"EnableAdminRoute":          true,
		"Logging.Level":             "info",
		"Logging.FileLogging":       false,
		"Logging.StdOut":            false,
		"Database.SchemaSearchPath": []string{"test"},
		"Database.TransactionEnd":   "rollback-allow-override",
		"JWTSecret":                 "reallyreallyreallyreallyverysafe",
	}
	s, err := server.NewServerWithConfig(c,
		&server.ConfigOptions{
			SkipFlags: true,
		})
	if err != nil {
		log.Fatal(err)
	}

	postgresToken, _ := server.GenerateToken("postgres", s.Config.JWTSecret)
	anonymousToken, _ := server.GenerateToken("postgrest_test_anonymous", s.Config.JWTSecret)

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
