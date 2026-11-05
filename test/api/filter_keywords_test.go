package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// Card 35450 filter-keywords. A bare null, true, false, unknown or not_null
// operand was written into the SQL for every operator, so a text column
// holding the strings 'true' or 'null' could not be matched: eq.true failed
// with 42883 and eq.null matched nothing. As in PostgREST 16.4, every operand
// is bound except the operand of is, which is a keyword in any case. The
// expected rows are the live answers of PostgREST 16.4 on the same rows.
func TestFilterKeywords(t *testing.T) {
	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	test.Prepare(cmdConfig, []test.Command{
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "filter_keywords",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true},
					{"name": "name", "type": "text"},
					{"name": "flag", "type": "bool"}
				],
				"ifnotexists": true
			}`,
		},
	})

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	tests := []test.Test{
		{
			Description: "insert records",
			Method:      "POST",
			Query:       "/filter_keywords",
			Body: `[
				{"id": 1, "name": "true", "flag": true},
				{"id": 2, "name": "null", "flag": null},
				{"id": 3, "name": "NULL", "flag": false},
				{"id": 4, "name": "x", "flag": true},
				{"id": 5, "name": null, "flag": null}
			]`,
			Status: 201,
		},
		{
			Description: "eq.true on a text column matches the string",
			Query:       "/filter_keywords?name=eq.true&select=id",
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
		{
			Description: "eq.null on a text column matches the string",
			Query:       "/filter_keywords?name=eq.null&select=id",
			Expected:    `[{"id":2}]`,
			Status:      200,
		},
		{
			Description: "eq.NULL keeps its case",
			Query:       "/filter_keywords?name=eq.NULL&select=id",
			Expected:    `[{"id":3}]`,
			Status:      200,
		},
		{
			Description: "in.(x,null) matches the string null",
			Query:       "/filter_keywords?name=in.(x,null)&select=id&order=id",
			Expected:    `[{"id":2},{"id":4}]`,
			Status:      200,
		},
		{
			Description: "eq.not_null is a string",
			Query:       "/filter_keywords?name=eq.not_null&select=id",
			Expected:    `[]`,
			Status:      200,
		},
		{
			Description: "eq.unknown is a string",
			Query:       "/filter_keywords?name=eq.unknown&select=id",
			Expected:    `[]`,
			Status:      200,
		},
		{
			Description: "fts.null searches the word null",
			Query:       "/filter_keywords?name=fts.null&select=id&order=id",
			Expected:    `[{"id":2},{"id":3}]`,
			Status:      200,
		},
		{
			Description: "logic tree: eq.null and eq.true match the strings",
			Query:       "/filter_keywords?or=(name.eq.null,name.eq.true)&select=id&order=id",
			Expected:    `[{"id":1},{"id":2}]`,
			Status:      200,
		},
		{
			Description: "any: an unquoted null is a NULL element",
			Query:       "/filter_keywords?name=eq(any).{x,null}&select=id&order=id",
			Expected:    `[{"id":4}]`,
			Status:      200,
		},
		{
			Description: "any: a quoted null is the string",
			Query:       `/filter_keywords?name=eq(any).{x,"null"}&select=id&order=id`,
			Expected:    `[{"id":2},{"id":4}]`,
			Status:      200,
		},
		{
			Description: "eq.TRUE on a boolean column",
			Query:       "/filter_keywords?flag=eq.TRUE&select=id&order=id",
			Expected:    `[{"id":1},{"id":4}]`,
			Status:      200,
		},
		{
			Description: "eq.null on a boolean column is invalid input (22P02)",
			Query:       "/filter_keywords?flag=eq.null&select=id",
			Status:      400,
		},
		{
			Description: "eq.null on an integer column is invalid input (22P02)",
			Query:       "/filter_keywords?id=eq.null&select=id",
			Status:      400,
		},
		// controls: the operand of is stays a keyword
		{
			Description: "is.NULL",
			Query:       "/filter_keywords?flag=is.NULL&select=id&order=id",
			Expected:    `[{"id":2},{"id":5}]`,
			Status:      200,
		},
		{
			Description: "is.TRUE",
			Query:       "/filter_keywords?flag=is.TRUE&select=id&order=id",
			Expected:    `[{"id":1},{"id":4}]`,
			Status:      200,
		},
		{
			Description: "is.not_null",
			Query:       "/filter_keywords?name=is.not_null&select=id&order=id",
			Expected:    `[{"id":1},{"id":2},{"id":3},{"id":4}]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig, tests)
}
