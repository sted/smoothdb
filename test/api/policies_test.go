package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func TestPolicies(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		// drop table table_policies
		{
			Method: "DELETE",
			Query:  "/databases/dbtest/tables/table_policies",
		},
		// create table table_policies
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables",
			Body: `{
				"name": "table_policies", 
				"columns": [
					{"name": "name", "type": "text", "notnull": true},
					{"name": "creator", "type": "text", "default": "current_user"}
				]}`,
		},
		// enable row level security
		{
			Method: "PATCH",
			Query:  "/databases/dbtest/tables/table_policies",
			Body: `{
				"rowsecurity": true
				}`,
		},
		// grant public access to table_policies
		{
			Method: "POST",
			Query:  "/grants/dbtest/table/table_policies",
			Body:   `{"types": ["all"], "grantee": "public"}`,
		},
		// create policy for select
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables/table_policies/policies",
			Body:   `{"name": "perm_read", "command": "select", "roles": ["public"], "using": "creator = current_user"}`,
		},
		// create policy for insert
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables/table_policies/policies",
			Body:   `{"name": "perm_write", "command": "insert",  "check": "name = current_user"}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: nil,
	}

	tests := []test.Test{
		{
			Description: "create a record",
			Method:      "POST",
			Query:       "/table_policies",
			Body:        `{"name": "user1"}`,
			Headers:     test.Headers{"Authorization": {user1Token}},
			Status:      201,
		},
		{
			Description: "create a record",
			Method:      "POST",
			Query:       "/table_policies",
			Body:        `{"name": "user2"}`,
			Headers:     test.Headers{"Authorization": {user2Token}},
			Status:      201,
		},
		{
			Description: "select",
			Method:      "GET",
			Query:       "/table_policies?select=name",
			Expected:    `[{"name": "user1"}]`,
			Headers:     test.Headers{"Authorization": {user1Token}},
			Status:      200,
		},
		{
			Description: "select",
			Method:      "GET",
			Query:       "/table_policies?select=name",
			Expected:    `[{"name": "user2"}]`,
			Headers:     test.Headers{"Authorization": {user2Token}},
			Status:      200,
		},
		{
			Description: "create a record with wrong name",
			Method:      "POST",
			Query:       "/table_policies",
			Body:        `{"name": "user3"}`,
			Headers:     test.Headers{"Authorization": {user1Token}},
			Status:      401,
		},
	}

	test.Execute(t, testConfig, tests)
}
