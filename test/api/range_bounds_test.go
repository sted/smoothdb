package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// A range column is returned as the text PostgreSQL's to_json prints for it,
// as PostgREST does (its body is json_agg of the row). 'empty', '[10,)',
// '(,10)' and '(,)' used to crash a binary decoder that read both bounds
// unconditionally; every shape must serialize as PostgreSQL prints it. The
// bounded rows are the positive controls. Integer and numeric subtypes keep
// the text valid JSON independently of how the bounds are quoted.
func TestRangeBounds(t *testing.T) {
	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	commands := []test.Command{
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "range_bounds",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true},
					{"name": "r4", "type": "int4range"},
					{"name": "r8", "type": "int8range"},
					{"name": "rn", "type": "numrange"}
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
	test.Execute(t, testConfig, []test.Test{
		{
			Description: "insert one row per range shape",
			Method:      "POST",
			Query:       "/range_bounds",
			Body: `[
				{"id": 1, "r4": "[1,10)", "r8": "[1,10)", "rn": "[1,10)"},
				{"id": 2, "r4": "[1,10]", "r8": "[1,10]", "rn": "[1,10]"},
				{"id": 3, "r4": "(1,5]", "r8": "(1,5]", "rn": "(1,5]"},
				{"id": 4, "r4": "[10,)", "r8": "[10,)", "rn": "[10,)"},
				{"id": 5, "r4": "(,10)", "r8": "(,10)", "rn": "(,10)"},
				{"id": 6, "r4": "(,)", "r8": "(,)", "rn": "(,)"},
				{"id": 7, "r4": "empty", "r8": "empty", "rn": "empty"}
			]`,
			Status: 201,
		},
	})

	// One Execute per shape: Execute stops at the first body mismatch, and
	// each shape must be reported on its own. Expected bodies are what
	// `select to_json(col)` returns (int4range/int8range are canonicalized
	// to the [) form by the server).
	tests := []test.Test{
		{
			Description: "bounded [1,10) (control)",
			Query:       "/range_bounds?id=eq.1&select=r4,r8,rn",
			Expected:    `[{"r4":"[1,10)","r8":"[1,10)","rn":"[1,10)"}]`,
			Status:      200,
		},
		{
			Description: "bounded [1,10], upper inclusive (control)",
			Query:       "/range_bounds?id=eq.2&select=r4,r8,rn",
			Expected:    `[{"r4":"[1,11)","r8":"[1,11)","rn":"[1,10]"}]`,
			Status:      200,
		},
		{
			Description: "bounded (1,5], lower exclusive (control)",
			Query:       "/range_bounds?id=eq.3&select=r4,r8,rn",
			Expected:    `[{"r4":"[2,6)","r8":"[2,6)","rn":"(1,5]"}]`,
			Status:      200,
		},
		{
			Description: "upper unbounded [10,)",
			Query:       "/range_bounds?id=eq.4&select=r4,r8,rn",
			Expected:    `[{"r4":"[10,)","r8":"[10,)","rn":"[10,)"}]`,
			Status:      200,
		},
		{
			Description: "lower unbounded (,10)",
			Query:       "/range_bounds?id=eq.5&select=r4,r8,rn",
			Expected:    `[{"r4":"(,10)","r8":"(,10)","rn":"(,10)"}]`,
			Status:      200,
		},
		{
			Description: "unbounded (,)",
			Query:       "/range_bounds?id=eq.6&select=r4,r8,rn",
			Expected:    `[{"r4":"(,)","r8":"(,)","rn":"(,)"}]`,
			Status:      200,
		},
		{
			Description: "empty",
			Query:       "/range_bounds?id=eq.7&select=r4,r8,rn",
			Expected:    `[{"r4":"empty","r8":"empty","rn":"empty"}]`,
			Status:      200,
		},
		{
			Description: "all shapes in one response",
			Query:       "/range_bounds?select=id,r4&order=id",
			Expected: `[{"id":1,"r4":"[1,10)"},{"id":2,"r4":"[1,11)"},{"id":3,"r4":"[2,6)"},
				{"id":4,"r4":"[10,)"},{"id":5,"r4":"(,10)"},{"id":6,"r4":"(,)"},{"id":7,"r4":"empty"}]`,
			Status: 200,
		},
		// The CSV value is PostgreSQL's text for the range, quoted for the
		// comma in it as PostgREST's CSV (the record text) quotes it.
		{
			Description: "csv: unbounded shapes",
			Query:       "/range_bounds?id=in.(1,4,5,6)&select=r4,r8,rn&order=id",
			Headers:     test.Headers{"Accept": {"text/csv"}},
			Expected: "r4,r8,rn\n" +
				"\"[1,10)\",\"[1,10)\",\"[1,10)\"\n" +
				"\"[10,)\",\"[10,)\",\"[10,)\"\n" +
				"\"(,10)\",\"(,10)\",\"(,10)\"\n" +
				"\"(,)\",\"(,)\",\"(,)\"",
			Status: 200,
		},
	}
	for _, tc := range tests {
		t.Run(tc.Description, func(t *testing.T) {
			test.Execute(t, testConfig, []test.Test{tc})
		})
	}
}
