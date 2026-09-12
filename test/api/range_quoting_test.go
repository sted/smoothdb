package test_api

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sted/smoothdb/test"
)

// A range is returned as the string PostgreSQL's to_json prints for it, which
// is what PostgREST returns (its body is json_agg of the row): range_out's
// text, with a bound quoted when it contains a comma, a quote, whitespace or
// a bracket, and those quotes escaped in the JSON string. int4range and
// numrange print their bounds bare and are the positive controls; daterange
// prints unquoted dates, tsrange and tstzrange quoted timestamps (in
// PostgreSQL's text form, a space between date and time, the session time
// zone for tstzrange). Every body must be valid JSON before it is compared.
func TestRangeQuoting(t *testing.T) {
	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	commands := []test.Command{
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "range_quoting",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true},
					{"name": "r4", "type": "int4range"},
					{"name": "rn", "type": "numrange"},
					{"name": "rd", "type": "daterange"},
					{"name": "rts", "type": "tsrange"},
					{"name": "rtz", "type": "tstzrange"},
					{"name": "rd_arr", "type": "daterange[]"},
					{"name": "rts_arr", "type": "tsrange[]"}
				],
				"ifnotexists": true
			}`,
		},
		// a function returning ranges, for the rpc path (as a table, whose
		// shape does not depend on the function being in the schema cache)
		{
			Method: "POST",
			Query:  "/dbtest/functions",
			Body: `{
				"name": "range_quoting_rows",
				"returns": "table(id int, days daterange, span tsrange)",
				"definition": "select 1, '[2024-01-01,2024-06-01)'::daterange, '[2024-01-01 10:00:00,2024-06-01 12:00:00)'::tsrange"
			}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	// The text of a tstzrange depends on the server's session time zone:
	// ask PostgreSQL what it prints, as PostgREST would return it.
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, testDatabaseURL)
	if err != nil {
		t.Fatalf("cannot connect to the test database: %v", err)
	}
	var rtz string
	err = conn.QueryRow(ctx, `select to_json('[2024-01-01 10:00:00+00,)'::tstzrange)::text`).Scan(&rtz)
	conn.Close(ctx)
	if err != nil {
		t.Fatal(err)
	}

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	test.Execute(t, testConfig, []test.Test{
		{
			Description: "insert one full row, one of nulls and one of shapes with nothing to quote",
			Method:      "POST",
			Query:       "/range_quoting",
			Body: `[
				{"id": 1, "r4": "[1,10)", "rn": "[1.5,2.5]", "rd": "[2024-01-01,2024-06-01)",
				 "rts": "[2024-01-01 10:00:00,2024-06-01 12:00:00)", "rtz": "[2024-01-01 10:00:00+00,)",
				 "rd_arr": "{\"[2024-01-01,2024-06-01)\",\"[2024-02-01,)\"}",
				 "rts_arr": "{\"[\\\"2024-01-01 10:00:00\\\",\\\"2024-06-01 12:00:00\\\")\",NULL}"},
				{"id": 2},
				{"id": 3, "rd": "empty", "rts": "(,)", "rtz": "empty"}
			]`,
			Status: 201,
		},
	})

	// One Execute per case: Execute stops at the first body mismatch, and a
	// body that is not JSON decodes to null there, so each body is checked
	// for validity first, with the raw bytes in the failure.
	tests := []test.Test{
		{
			Description: "bare bounds (control)",
			Query:       "/range_quoting?id=eq.1&select=r4,rn",
			Expected:    `[{"r4":"[1,10)","rn":"[1.5,2.5]"}]`,
			Status:      200,
		},
		{
			Description: "daterange: unquoted bounds",
			Query:       "/range_quoting?id=eq.1&select=rd",
			Expected:    `[{"rd":"[2024-01-01,2024-06-01)"}]`,
			Status:      200,
		},
		{
			Description: "tsrange: quoted bounds, escaped",
			Query:       "/range_quoting?id=eq.1&select=rts",
			Expected:    `[{"rts":"[\"2024-01-01 10:00:00\",\"2024-06-01 12:00:00\")"}]`,
			Status:      200,
		},
		{
			Description: "tstzrange: quoted bound in the session time zone",
			Query:       "/range_quoting?id=eq.1&select=rtz",
			Expected:    `[{"rtz":` + rtz + `}]`,
			Status:      200,
		},
		{
			Description: "arrays of ranges",
			Query:       "/range_quoting?id=eq.1&select=rd_arr,rts_arr",
			Expected:    `[{"rd_arr":["[2024-01-01,2024-06-01)","[2024-02-01,)"],"rts_arr":["[\"2024-01-01 10:00:00\",\"2024-06-01 12:00:00\")",null]}]`,
			Status:      200,
		},
		{
			Description: "nulls stay null",
			Query:       "/range_quoting?id=eq.2&select=rd,rts,rtz,rd_arr",
			Expected:    `[{"rd":null,"rts":null,"rtz":null,"rd_arr":null}]`,
			Status:      200,
		},
		{
			Description: "empty and unbounded shapes",
			Query:       "/range_quoting?id=eq.3&select=rd,rts,rtz",
			Expected:    `[{"rd":"empty","rts":"(,)","rtz":"empty"}]`,
			Status:      200,
		},
		{
			Description: "every family in one row",
			Query:       "/range_quoting?id=eq.1&select=id,r4,rd,rts,rn",
			Expected:    `[{"id":1,"r4":"[1,10)","rd":"[2024-01-01,2024-06-01)","rts":"[\"2024-01-01 10:00:00\",\"2024-06-01 12:00:00\")","rn":"[1.5,2.5]"}]`,
			Status:      200,
		},
		{
			Description: "return=representation after an update",
			Method:      "PATCH",
			Query:       "/range_quoting?id=eq.2&select=id,rd,rts",
			Body:        `{"rd": "[2024-03-01,2024-04-01]", "rts": "(2024-03-01 00:00:00,2024-04-01 00:00:00]"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":2,"rd":"[2024-03-01,2024-04-02)","rts":"(\"2024-03-01 00:00:00\",\"2024-04-01 00:00:00\"]"}]`,
			Status:      200,
		},
		{
			Description: "a function returning ranges",
			Method:      "POST",
			Query:       "/rpc/range_quoting_rows",
			Body:        `{}`,
			Expected:    `[{"id":1,"days":"[2024-01-01,2024-06-01)","span":"[\"2024-01-01 10:00:00\",\"2024-06-01 12:00:00\")"}]`,
			Status:      200,
		},
		{
			Description: "a function returning ranges, with select",
			Query:       "/rpc/range_quoting_rows?select=span",
			Expected:    `[{"span":"[\"2024-01-01 10:00:00\",\"2024-06-01 12:00:00\")"}]`,
			Status:      200,
		},
		// CSV: PostgreSQL's text for the range, quoted for its commas and
		// quotes as PostgREST's CSV (the record text) quotes it.
		{
			Description: "csv",
			Query:       "/range_quoting?id=eq.1&select=r4,rd,rts,rn",
			Headers:     test.Headers{"Accept": {"text/csv"}},
			Expected: "r4,rd,rts,rn\n" +
				`"[1,10)","[2024-01-01,2024-06-01)","[""2024-01-01 10:00:00"",""2024-06-01 12:00:00"")","[1.5,2.5]"`,
			Status: 200,
		},
	}
	client := test.InitClient()
	for _, tc := range tests {
		t.Run(tc.Description, func(t *testing.T) {
			if _, ok := tc.Headers["Accept"]; !ok {
				body, _, _, err := test.Exec(client, testConfig, &test.Command{Method: tc.Method, Query: tc.Query, Body: tc.Body, Headers: tc.Headers})
				if err != nil {
					t.Fatal(err)
				}
				if !json.Valid(body) {
					t.Errorf("the body is not JSON: %s", body)
				}
			}
			test.Execute(t, testConfig, []test.Test{tc})
		})
	}
}
