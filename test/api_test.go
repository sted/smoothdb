package test

import (
	"green/green-ds/server"
	"log"
	"os"
	"testing"
)

var (
	pippoToken string
	plutoToken string
)

func TestMain(m *testing.M) {
	s, err := server.NewServer()
	if err != nil {
		log.Fatal(err)
	}
	pippoToken, _ = server.GenerateToken("pippo", s.Config.JWTSecret)
	plutoToken, _ = server.GenerateToken("pluto", s.Config.JWTSecret)

	go s.Start()
	code := m.Run()
	os.Exit(code)
}

func TestPolicies(t *testing.T) {

	cmdConfig := TestConfig{
		BaseUrl: "http://localhost:8081/admin/databases",
	}

	commands := []Test{
		// create database dbtest
		{
			Method: "POST",
			Query:  "/",
			Body:   `{"name": "dbtest"}`,
		},
		// drop table table1
		{
			Method: "DELETE",
			Query:  "/dbtest/tables/table1",
		},
		// create table table1
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "table1", 
				"columns": [
					{"name": "name", "type": "text", "notnull": true},
					{"name": "creator", "type": "text", "default": "current_user"}
				]}`,
		},
		// enable row level security
		{
			Method: "PATCH",
			Query:  "/dbtest/tables/table1",
			Body: `{
				"rowsecurity": true
				}`,
		},
		// create policy for select
		{
			Method: "POST",
			Query:  "/dbtest/tables/table1/policies",
			Body:   `{"name": "perm_read", "command": "select", "roles": ["public"], "using": "creator = current_user"}`,
		},
		// create policy for insert
		{
			Method: "POST",
			Query:  "/dbtest/tables/table1/policies",
			Body:   `{"name": "perm_write", "command": "insert", "check": "true"}`,
		},
	}
	Execute(t, cmdConfig, commands)

	testConfig := TestConfig{
		BaseUrl:       "http://localhost:8081/dbtest",
		CommonHeaders: nil,
		NoCookies:     true,
	}

	tests := []Test{
		{
			Description: "create a record",
			Method:      "POST",
			Query:       "/table1",
			Body:        `{"name": "pippo"}`,
			Headers:     map[string]string{"Authorization": pippoToken},
			Status:      201,
		},
		{
			Description: "create a record",
			Method:      "POST",
			Query:       "/table1",
			Body:        `{"name": "pluto"}`,
			Headers:     map[string]string{"Authorization": plutoToken},
			Status:      201,
		},
		{
			Description: "select",
			Method:      "GET",
			Query:       "/table1?select=name",
			Expected:    `[{"name": "pippo"}]`,
			Headers:     map[string]string{"Authorization": pippoToken},
			Status:      200,
		},
		{
			Description: "select",
			Method:      "GET",
			Query:       "/table1?select=name",
			Expected:    `[{"name": "pluto"}]`,
			Headers:     map[string]string{"Authorization": plutoToken},
			Status:      200,
		},
	}

	Execute(t, testConfig, tests)
}
