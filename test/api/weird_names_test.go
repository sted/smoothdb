package test_api

import (
	"testing"

	"github.com/smoothdb/smoothdb/test"
)

func TestWeirdNames(t *testing.T) {

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
		NoCookies:     true,
	}

	tests := []test.Test{
		{
			Description: "create a role",
			Method:      "POST",
			Query:       "/roles",
			Body:        `{"name": "john.gruber@catflow.it"}`,
			Status:      201,
		},
		{
			Description: "delete a role",
			Method:      "DELETE",
			Query:       "/roles/john.gruber@catflow.it",
			Status:      204,
		},
		{
			Description: "create a user",
			Method:      "POST",
			Query:       "/users",
			Body:        `{"name": "john.gruber@catflow.it"}`,
			Status:      201,
		},
		{
			Description: "delete a user",
			Method:      "DELETE",
			Query:       "/users/john.gruber@catflow.it",
			Status:      204,
		},
		{
			Description: "create a schema",
			Method:      "POST",
			Query:       "/databases/dbtest/schemas",
			Body:        `{"name": "''sS '"}`,
			Status:      201,
		},
		{
			Description: "delete a schema",
			Method:      "DELETE",
			Query:       "/databases/dbtest/schemas/''sS '",
			Status:      204,
		},
		{
			Description: "create a table",
			Method:      "POST",
			Query:       "/databases/dbtest/tables",
			Body:        `{"name": "' Wt"}`,
			Status:      201,
		},
		{
			Description: "delete a table",
			Method:      "DELETE",
			Query:       "/databases/dbtest/tables/' Wt",
			Status:      204,
		},
	}

	test.Execute(t, testConfig, tests)
}
