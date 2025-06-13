package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func TestRecords(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		// create table table_records with numeric column
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
					{"name": "float8", "type": "float8"},
					{"name": "numeric_val", "type": "numeric(15,4)"},
					{"name": "interval", "type": "interval"}
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
			Description: "insert records with comprehensive numeric values",
			Method:      "POST",
			Query:       "/table_records",
			Body: `[
				{"name": "basic_positive", "int4": 1, "int8": 2, "float4": 1.2E3, "float8": 2.3, "numeric_val": "123.4567", "interval": "P1Y2M6DT7M"},
				{"name": "basic_negative", "int4": -1, "int8": -50000000, "float4": -1.2, "float8": -2.34554, "numeric_val": "-456.7890", "interval": "3 mons 05:06:07"},
				{"name": "zero_values", "int4": 0, "int8": 0, "float4": 0.0, "float8": 0.0, "numeric_val": "0.0000", "interval": "00:00:00"},
				{"name": "high_precision", "int4": 2147483647, "int8": 9223372036854775807, "float4": 3.4028235E38, "float8": 1.7976931348623157E308, "numeric_val": "99999999999.9999", "interval": "999 days 23:59:59"},
				{"name": "small_decimals", "int4": 42, "int8": 12345, "float4": 0.001, "float8": 0.000001, "numeric_val": "0.0001", "interval": "1 second"},
				{"name": "financial_test", "int4": 1000, "int8": 1000000, "float4": 999.99, "float8": 1234567.89, "numeric_val": "12345.6789", "interval": "30 days"}
			]`,
			Status: 201,
		},
		{
			Description: "select records and verify numeric serialization",
			Method:      "GET",
			Query:       "/table_records",
			Expected: `[
				{"name": "basic_positive", "int4": 1, "int8": 2, "float4": 1.2E3, "float8": 2.3, "numeric_val": 123.4567, "interval": "1 year 2 mons 6 days 00:07:00"},
				{"name": "basic_negative", "int4": -1, "int8": -50000000, "float4": -1.2, "float8": -2.34554, "numeric_val": -456.789, "interval": "3 mons 05:06:07"},
				{"name": "zero_values", "int4": 0, "int8": 0, "float4": 0, "float8": 0, "numeric_val": 0, "interval": ""},
				{"name": "high_precision", "int4": 2147483647, "int8": 9223372036854776000, "float4": 3.4028235e+38, "float8": 1.7976931348623157e+308, "numeric_val": 99999999999.9999, "interval": "999 days 23:59:59"},
				{"name": "small_decimals", "int4": 42, "int8": 12345, "float4": 0.001, "float8": 0.000001, "numeric_val": 0.0001, "interval": "00:00:01"},
				{"name": "financial_test", "int4": 1000, "int8": 1000000, "float4": 999.99, "float8": 1234567.89, "numeric_val": 12345.6789, "interval": "30 days"}
			]`,
			Status: 200,
		},
	}

	test.Execute(t, testConfig, tests)
}
