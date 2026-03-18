package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// Tests for issue #15: Incorrect native type casting for boolean and null values
// when filtering JSONB via `->` operator.

func TestJSONBBoolNull(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "jsonb_bool_null",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true},
					{"name": "data", "type": "jsonb"}
				],
				"ifnotexists": true
			}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	tests := []test.Test{
		{
			Description: "insert jsonb records with boolean and null values",
			Method:      "POST",
			Query:       "/jsonb_bool_null",
			Body: `[
				{"id": 1, "data": {"f": true}},
				{"id": 2, "data": {"f": false}},
				{"id": 3, "data": {"g": null}}
			]`,
			Status: 201,
		},
		{
			Description: "can filter jsonb boolean true via arrow operator",
			Query:       "/jsonb_bool_null?data->f=eq.true",
			Expected:    `[{"id":1,"data":{"f": true}}]`,
			Status:      200,
		},
		{
			Description: "can filter jsonb boolean false via arrow operator",
			Query:       "/jsonb_bool_null?data->f=eq.false",
			Expected:    `[{"id":2,"data":{"f": false}}]`,
			Status:      200,
		},
		{
			Description: "can filter jsonb null via arrow operator",
			Query:       "/jsonb_bool_null?data->g=eq.null",
			Expected:    `[{"id":3,"data":{"g": null}}]`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}
