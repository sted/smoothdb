package api

import (
	"green/green-ds/logging"
	"green/green-ds/server"
	"green/green-ds/test"
	"log"
	"os"
	"testing"
)

var (
	adminToken string
	user1Token string
	user2Token string
)

func TestMain(m *testing.M) {
	c := &server.Config{
		AllowAnon:        false,
		EnableAdminRoute: true,
		Logging: logging.Config{
			FileLogging: false,
			StdOut:      true,
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

	go s.Start()

	// Tear-up

	postgresToken, _ := server.GenerateToken("postgres", s.Config.JWTSecret)
	adminToken, _ = server.GenerateToken("admin", s.Config.JWTSecret)
	user1Token, _ = server.GenerateToken("user1", s.Config.JWTSecret)
	user2Token, _ = server.GenerateToken("user2", s.Config.JWTSecret)

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8081/admin",
		CommonHeaders: map[string]string{"Authorization": postgresToken},
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
		// grant role admin to anon
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
		// create role user2
		{
			Method: "POST",
			Query:  "/users",
			Body: `{
				"name": "user2"
			}`,
		},
		// create database dbtest
		{
			Method:  "POST",
			Query:   "/databases",
			Body:    `{"name": "dbtest"}`,
			Headers: map[string]string{"Authorization": adminToken},
		},
	}
	test.Prepare(cmdConfig, commands)

	code := m.Run()

	// Tear-down

	os.Exit(code)
}
