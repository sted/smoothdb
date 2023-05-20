package postgrest

import (
	"log"
	"os"
	"testing"

	"github.com/smoothdb/smoothdb/database"
	"github.com/smoothdb/smoothdb/logging"
	"github.com/smoothdb/smoothdb/server"
	"github.com/smoothdb/smoothdb/test"
)

var testConfig test.Config

func TestMain(m *testing.M) {
	c := &server.Config{
		Address:          "localhost:8082",
		AllowAnon:        true,
		EnableAdminRoute: true,
		Logging: logging.Config{
			FileLogging: false,
			StdOut:      true,
		},
		Database: database.Config{
			SchemaSearchPath: []string{"test"},
			TransactionEnd:   "rollback-allow-override",
		},
		JWTSecret: "reallyreallyreallyreallyverysafe",
	}
	s, err := server.NewServerWithConfig(c,
		&server.ConfigOptions{
			ConfigFilePath: "../../config.jsonc",
			SkipFlags:      true,
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
