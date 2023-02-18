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
		AllowAnon:        true,
		EnableAdminRoute: true,
		Logging: logging.Config{
			FileLogging: false,
			StdOut:      false,
		},
		Database: database.Config{
			SchemaSearchPath: []string{"public", "test"},
			TransactionEnd:   "rollback-allow-override",
		},
	}
	s, err := server.NewServerWithConfig(c,
		&server.ConfigOptions{
			ConfigFilePath: "../../config.json",
			SkipFlags:      true,
		})
	if err != nil {
		log.Fatal(err)
	}

	postgresToken, _ := server.GenerateToken("postgres", s.Config.JWTSecret)
	anonymousToken, _ := server.GenerateToken("postgrest_test_anonymous", s.Config.JWTSecret)

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8081/admin",
		CommonHeaders: test.Headers{"Authorization": postgresToken},
	}

	testConfig = test.Config{
		BaseUrl: "http://localhost:8081/api/pgrest",
		CommonHeaders: test.Headers{
			"Authorization":   anonymousToken,
			"Accept-Profile":  "test",
			"Content-Profile": "test",
		},
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

	go s.Start()
	code := m.Run()
	os.Exit(code)
}
