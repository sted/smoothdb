package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func TestWeirdNames(t *testing.T) {

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
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
		{
			Description: "create a table with a column with a dot",
			Method:      "POST",
			Query:       "/databases/dbtest/tables",
			Body:        `{"name": "table1", "columns": [{"name": "a.a", "type": "int"}]}`,
			Status:      201,
		},
		{
			Description: "get the table",
			Method:      "GET",
			Query:       "/databases/dbtest/tables/table1",
			Body:        ``,
			Expected:    `{"name":"table1","schema":"public","owner":"admin","rowsecurity":false,"hasindexes":false,"hastriggers":false,"ispartition":false,"constraints":null,"columns":[{"name":"a.a","type":"int4","notnull":false,"default":null,"constraints":null,"table":"table1","schema":"public"}]}`,
			Status:      200,
		},
		{
			Description: "insert in the table",
			Method:      "POST",
			Query:       "http://localhost:8082/api/dbtest/table1",
			Body: `[
				{"a.a":1},{"a.a":2},{"a.a":3}
			]`,
			Status: 201,
		},
		{
			Description: "select the table",
			Method:      "GET",
			Query:       "http://localhost:8082/api/dbtest/table1",
			Body:        ``,
			Expected:    `[{"a.a":1},{"a.a":2},{"a.a":3}]`,
			Status:      200,
		},
		{
			Description: "select the table with a single column (\")",
			Method:      "GET",
			Query:       "http://localhost:8082/api/dbtest/table1?select=\"a.a\"",
			Body:        ``,
			Expected:    `[{"a.a":1},{"a.a":2},{"a.a":3}]`,
			Status:      200,
		},
		{
			Description: "select the table with a single column (')",
			Method:      "GET",
			Query:       "http://localhost:8082/api/dbtest/table1?select='a.a'",
			Body:        ``,
			Expected:    `[{"a.a":1},{"a.a":2},{"a.a":3}]`,
			Status:      200,
		},
		{
			Description: "select the table with a single column (%22)",
			Method:      "GET",
			Query:       "http://localhost:8082/api/dbtest/table1?select=%22a.a%22",
			Body:        ``,
			Expected:    `[{"a.a":1},{"a.a":2},{"a.a":3}]`,
			Status:      200,
		},
		{
			Description: "delete a table",
			Method:      "DELETE",
			Query:       "/databases/dbtest/tables/table1",
			Status:      204,
		},
		{
			Description: "create a table with a column with a double quote",
			Method:      "POST",
			Query:       "/databases/dbtest/tables",
			Body:        `{"name": "table2", "columns": [{"name": "a\"a", "type": "int"}]}`,
			Status:      201,
		},
		{
			Description: "get the table",
			Method:      "GET",
			Query:       "/databases/dbtest/tables/table2",
			Body:        ``,
			Expected:    `{"name":"table2","schema":"public","owner":"admin","rowsecurity":false,"hasindexes":false,"hastriggers":false,"ispartition":false,"constraints":null,"columns":[{"name":"a\"a","type":"int4","notnull":false,"default":null,"constraints":null,"table":"table2","schema":"public"}]}`,
			Status:      200,
		},
		{
			Description: "insert in the table",
			Method:      "POST",
			Query:       "http://localhost:8082/api/dbtest/table2",
			Body: `[
				{"a\"a":1},{"a\"a":2},{"a\"a":3}
			]`,
			Status: 201,
		},
		{
			Description: "select the table",
			Method:      "GET",
			Query:       "http://localhost:8082/api/dbtest/table2",
			Body:        ``,
			Expected:    `[{"a\"a":1},{"a\"a":2},{"a\"a":3}]`,
			Status:      200,
		},
		{
			Description: "delete a table",
			Method:      "DELETE",
			Query:       "/databases/dbtest/tables/table2",
			Status:      204,
		},
	}

	test.Execute(t, testConfig, tests)
}
