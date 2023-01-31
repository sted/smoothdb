package api

import (
	"green/green-ds/test"
	"testing"
)

func TestPolicies(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8081/admin",
		CommonHeaders: map[string]string{"Authorization": adminToken},
	}

	commands := []test.Command{
		// drop table table1
		{
			Method: "DELETE",
			Query:  "/databases/dbtest/tables/table1",
		},
		// create table table1
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables",
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
			Query:  "/databases/dbtest/tables/table1",
			Body: `{
				"rowsecurity": true
				}`,
		},
		// grant public access to table1
		{
			Method: "POST",
			Query:  "/grants/dbtest/table/table1",
			Body:   `{"types": ["all"], "grantee": "public"}`,
		},
		// create policy for select
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables/table1/policies",
			Body:   `{"name": "perm_read", "command": "select", "roles": ["public"], "using": "creator = current_user"}`,
		},
		// create policy for insert
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables/table1/policies",
			Body:   `{"name": "perm_write", "command": "insert",  "check": "name = current_user"}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8081/api/dbtest",
		CommonHeaders: nil,
		NoCookies:     true,
	}

	tests := []test.Test{
		{
			Description: "create a record",
			Method:      "POST",
			Query:       "/table1",
			Body:        `{"name": "user1"}`,
			Headers:     map[string]string{"Authorization": user1Token},
			Status:      201,
		},
		{
			Description: "create a record",
			Method:      "POST",
			Query:       "/table1",
			Body:        `{"name": "user2"}`,
			Headers:     map[string]string{"Authorization": user2Token},
			Status:      201,
		},
		{
			Description: "select",
			Method:      "GET",
			Query:       "/table1?select=name",
			Expected:    `[{"name": "user1"}]`,
			Headers:     map[string]string{"Authorization": user1Token},
			Status:      200,
		},
		{
			Description: "select",
			Method:      "GET",
			Query:       "/table1?select=name",
			Expected:    `[{"name": "user2"}]`,
			Headers:     map[string]string{"Authorization": user2Token},
			Status:      200,
		},
		{
			Description: "create a record with wrong name",
			Method:      "POST",
			Query:       "/table1",
			Body:        `{"name": "user3"}`,
			Headers:     map[string]string{"Authorization": user1Token},
			Status:      401,
		},
	}

	test.Execute(t, testConfig, tests)
}
