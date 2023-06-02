package test_api

import (
	"testing"

	"github.com/smoothdb/smoothdb/test"
)

func TestFunctions(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Test{
		// drop schema
		{
			Method: "DELETE",
			Query:  "/dbtest/schemas/functions",
		},
		// create schema
		{
			Method: "POST",
			Query:  "/dbtest/schemas",
			Body: `{ 
				"name": "functions"
			}`,
		},
	}
	test.Execute(t, cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl: "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken},
			"Content-Profile": {"functions"}},
		NoCookies: true,
	}

	tests := []test.Test{
		{
			Description: "create a function",
			Method:      "POST",
			Query:       "http://localhost:8082/admin/databases/dbtest/functions",
			Body: `{
				"name": "functions.f1",
				"arguments": [
					{"name": "a", "type": "integer"},
					{"name": "b", "type": "text"}
				],
				"returns": "table(a int, b text)",
				"definition": "select $1, $2"
			}`,
			Status: 201,
		},
		{
			Description: "exec a function",
			Method:      "POST",
			Query:       "/rpc/f1",
			Body: `{
				"a": 42,
				"b": "Wow"
			}`,
			Expected: `[{"a":42,"b":"Wow"}]`,
			Status:   200,
		},
	}

	test.Execute(t, testConfig, tests)
}
