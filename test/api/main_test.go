package test_api

import (
	"log"
	"os"
	"testing"

	"github.com/smoothdb/smoothdb/logging"
	"github.com/smoothdb/smoothdb/server"
	"github.com/smoothdb/smoothdb/test"
)

var (
	srv        *server.Server
	adminToken string
	user1Token string
	user2Token string
)

func TestMain(m *testing.M) {
	c := &server.Config{
		Address:          "localhost:8082",
		AllowAnon:        false,
		EnableAdminRoute: true,
		Logging: logging.Config{
			Level:       "trace",
			FileLogging: false,
			StdOut:      true,
		},
	}
	s, err := server.NewServerWithConfig(c,
		&server.ConfigOptions{
			ConfigFilePath: "../../config.jsonc",
			SkipFlags:      true,
		})
	if err != nil {
		log.Fatal(err)
	}
	srv = s

	go s.Start()

	// Tear-up

	postgresToken, _ := server.GenerateToken("postgres", s.Config.JWTSecret)
	adminToken, _ = server.GenerateToken("admin", s.Config.JWTSecret)
	user1Token, _ = server.GenerateToken("user1", s.Config.JWTSecret)
	user2Token, _ = server.GenerateToken("user2", s.Config.JWTSecret)

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin",
		CommonHeaders: test.Headers{"Authorization": {postgresToken}},
		NoCookies:     true,
	}

	commands := []test.Command{
		// create role admin
		{
			Method: "POST",
			Query:  "/roles",
			Body: `{
				"name": "admin",
				"issuperuser": true,
				"cancreatedatabases": true
			}`,
		},
		// grant role admin to auth
		{
			Method: "POST",
			Query:  "/grants",
			Body: `{
				"targetname": "admin",
				"grantee": "auth"
			}`,
		},
		// create role user1
		{
			Method: "POST",
			Query:  "/users",
			Body: `{
				"name": "user1"
			}`,
		},
		// grant role user1 to auth
		{
			Method: "POST",
			Query:  "/grants",
			Body: `{
				"targetname": "user1",
				"grantee": "auth"
			}`,
		},
		// create role user2
		{
			Method: "POST",
			Query:  "/users",
			Body: `{
				"name": "user2"
			}`,
		},
		// grant role user2 to auth
		{
			Method: "POST",
			Query:  "/grants",
			Body: `{
				"targetname": "user2",
				"grantee": "auth"
			}`,
		},
		// delete database dbtest
		{
			Method:  "DELETE",
			Query:   "/databases/dbtest",
			Headers: test.Headers{"Authorization": {adminToken}},
		},
		// create database dbtest
		{
			Method:  "POST",
			Query:   "/databases",
			Body:    `{"name": "dbtest"}`,
			Headers: test.Headers{"Authorization": {adminToken}},
		},
	}
	test.Prepare(cmdConfig, commands)

	code := m.Run()

	// Tear-down

	os.Exit(code)
}
