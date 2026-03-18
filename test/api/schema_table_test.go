package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// Tests for issue #18: Table creation in non-default schema returns 404

func TestCreateTableInSchema(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	// create the schema first
	commands := []test.Command{
		{
			Method: "POST",
			Query:  "/dbtest/schemas",
			Body:   `{"name": "testschema"}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	tests := []test.Test{
		// create table in non-default schema should return 201 with the table
		{
			Description: "create table in non-default schema",
			Method:      "POST",
			Query:       "/dbtest/tables",
			Body: `{
				"name": "schema_test_table",
				"schema": "testschema",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true},
					{"name": "val", "type": "text"}
				]
			}`,
			Status: 201,
		},
	}

	test.Execute(t, cmdConfig, tests)
}
