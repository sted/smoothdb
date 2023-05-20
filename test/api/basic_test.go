package test_api

import (
	"testing"

	"github.com/smoothdb/smoothdb/test"
)

func TestRecords(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
		//NoCookies:     true,
	}

	commands := []test.Test{
		// drop table table_records
		{
			Method: "DELETE",
			Query:  "/dbtest/tables/table_records",
		},
		// create table table_records
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "table_records",
				"columns": [
					{"name": "name", "type": "text", "notnull": true},
					{"name": "int4", "type": "int4"},
					{"name": "int8", "type": "int8"},
					{"name": "float4", "type": "float4"},
					{"name": "float8", "type": "float8"}
				]}`,
		},
	}
	test.Execute(t, cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
		NoCookies:     true,
	}

	tests := []test.Test{
		{
			Description: "insert records",
			Method:      "POST",
			Query:       "/table_records",
			Body: `[
				{"name": "one", "int4": 1, "int8": 2, "float4": 1.2E3, "float8": 2.3}
			]`,
			Status: 201,
		},
		{
			Description: "select records",
			Method:      "GET",
			Query:       "/table_records",
			Expected: `[
				{"name": "one", "int4": 1, "int8": 2, "float4": 1.2E3, "float8": 2.3}
			]`,
			Status: 200,
		},
	}

	test.Execute(t, testConfig, tests)
}
