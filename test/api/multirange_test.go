package test_api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sted/smoothdb/test"
)

// A multirange is returned as the string PostgreSQL's to_json prints for it,
// which is what PostgREST returns (its body is json_agg of the row): the
// ranges between braces, each with range_out's quoting of its bounds, "{}" for
// the empty one. pgx requests the builtin multiranges in binary and the
// serializer has no decoder for them, so any such column failed; a custom
// multirange (textmultirange, created with textrange) is unknown to pgx and
// arrives in text, the positive control with the int4range column. The range
// operators apply to a multirange column as PostgreSQL resolves them for an
// untyped parameter, which is how PostgREST binds a filter value: against a
// multirange literal (cs.{[1,2)}, ov.{[6,9)}), and a bare element is a 22P02.
// Every body must be valid JSON before it is compared.
func TestMultirange(t *testing.T) {
	// textrange (and with it textmultirange) is created directly: the admin
	// API creates no types.
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, strings.TrimSuffix(testDatabaseURL, "/postgres")+"/dbtest")
	if err != nil {
		t.Fatalf("cannot connect to the test database: %v", err)
	}
	_, err = conn.Exec(ctx, `do $$ begin
		if not exists (select 1 from pg_type where typname = 'textrange') then
			create type textrange as range (subtype = text);
		end if;
	end $$`)
	conn.Close(ctx)
	if err != nil {
		t.Fatal(err)
	}

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	commands := []test.Command{
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "multiranges",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true},
					{"name": "r4", "type": "int4range"},
					{"name": "m4", "type": "int4multirange"},
					{"name": "mn", "type": "nummultirange"},
					{"name": "md", "type": "datemultirange"},
					{"name": "mts", "type": "tsmultirange"},
					{"name": "mt", "type": "textmultirange"},
					{"name": "m4_arr", "type": "int4multirange[]"}
				],
				"ifnotexists": true
			}`,
		},
		// a function returning multiranges, for the rpc path (as a table,
		// whose shape does not depend on the function being in the schema
		// cache)
		{
			Method: "POST",
			Query:  "/dbtest/functions",
			Body: `{
				"name": "multirange_rows",
				"returns": "table(id int, slots int4multirange, days datemultirange)",
				"definition": "select 1, '{[1,3),[5,7)}'::int4multirange, '{[2024-01-01,2024-02-01)}'::datemultirange"
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
			Description: "insert one full row, one of nulls and one of empty multiranges",
			Method:      "POST",
			Query:       "/multiranges",
			Body: `[
				{"id": 1, "r4": "[1,10)", "m4": "{[1,3),[5,7)}", "mn": "{[1.5,2.5],[3,)}",
				 "md": "{[2024-01-01,2024-02-01),[2024-03-01,)}", "mts": "{[2024-01-01 10:00:00,2024-06-01 12:00:00)}",
				 "mt": "{[\"a,b\",\"c\\\"d\"],[x,y)}",
				 "m4_arr": "{\"{[1,3)}\",\"{}\",NULL}"},
				{"id": 2, "r4": null, "m4": null, "mn": null, "md": null, "mts": null, "mt": null, "m4_arr": null},
				{"id": 3, "r4": "empty", "m4": "{}", "mn": null, "md": "{}", "mts": null, "mt": "{}", "m4_arr": "{}"}
			]`,
			Status: 201,
		},
	})

	// One Execute per case: Execute stops at the first body mismatch, and a
	// body that is not JSON decodes to null there, so each body is checked
	// for validity first, with the raw bytes in the failure.
	tests := []test.Test{
		{
			Description: "range and custom multirange (controls)",
			Query:       "/multiranges?id=eq.1&select=id,r4,mt",
			Expected:    `[{"id":1,"r4":"[1,10)","mt":"{[\"a,b\",\"c\"\"d\"],[x,y)}"}]`,
			Status:      200,
		},
		{
			Description: "builtin multiranges with bare bounds",
			Query:       "/multiranges?id=eq.1&select=m4,mn,md",
			Expected:    `[{"m4":"{[1,3),[5,7)}","mn":"{[1.5,2.5],[3,)}","md":"{[2024-01-01,2024-02-01),[2024-03-01,)}"}]`,
			Status:      200,
		},
		{
			Description: "tsmultirange: quoted bounds, escaped",
			Query:       "/multiranges?id=eq.1&select=mts",
			Expected:    `[{"mts":"{[\"2024-01-01 10:00:00\",\"2024-06-01 12:00:00\")}"}]`,
			Status:      200,
		},
		{
			Description: "array of multiranges",
			Query:       "/multiranges?id=eq.1&select=m4_arr",
			Expected:    `[{"m4_arr":["{[1,3)}","{}",null]}]`,
			Status:      200,
		},
		{
			Description: "nulls stay null",
			Query:       "/multiranges?id=eq.2&select=m4,mn,md,mts,mt,m4_arr",
			Expected:    `[{"m4":null,"mn":null,"md":null,"mts":null,"mt":null,"m4_arr":null}]`,
			Status:      200,
		},
		{
			Description: "the empty multirange",
			Query:       "/multiranges?id=eq.3&select=r4,m4,md,mt,m4_arr",
			Expected:    `[{"r4":"empty","m4":"{}","md":"{}","mt":"{}","m4_arr":[]}]`,
			Status:      200,
		},
		{
			Description: "every column in one row",
			Query:       "/multiranges?id=eq.1",
			Expected:    `[{"id":1,"r4":"[1,10)","m4":"{[1,3),[5,7)}","mn":"{[1.5,2.5],[3,)}","md":"{[2024-01-01,2024-02-01),[2024-03-01,)}","mts":"{[\"2024-01-01 10:00:00\",\"2024-06-01 12:00:00\")}","mt":"{[\"a,b\",\"c\"\"d\"],[x,y)}","m4_arr":["{[1,3)}","{}",null]}]`,
			Status:      200,
		},
		{
			Description: "return=representation after an update",
			Method:      "PATCH",
			Query:       "/multiranges?id=eq.2&select=id,m4,md",
			Body:        `{"m4": "{[1,2],[2,4)}", "md": "{[2024-03-01,2024-04-01]}"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":2,"m4":"{[1,4)}","md":"{[2024-03-01,2024-04-02)}"}]`,
			Status:      200,
		},
		{
			Description: "a function returning multiranges",
			Method:      "POST",
			Query:       "/rpc/multirange_rows",
			Body:        `{}`,
			Expected:    `[{"id":1,"slots":"{[1,3),[5,7)}","days":"{[2024-01-01,2024-02-01)}"}]`,
			Status:      200,
		},
		{
			Description: "a function returning multiranges, with select",
			Query:       "/rpc/multirange_rows?select=slots",
			Expected:    `[{"slots":"{[1,3),[5,7)}"}]`,
			Status:      200,
		},
		// The range operators, with a multirange literal: PostgreSQL resolves
		// `m4 @> $1` for an untyped $1 to the multirange variant, so the
		// value is a multirange, as with PostgREST.
		{
			Description: "cs with a multirange literal",
			Query:       "/multiranges?m4=cs.{[1,2)}&select=id&order=id",
			Expected:    `[{"id":1},{"id":2}]`,
			Status:      200,
		},
		{
			Description: "ov with a multirange literal",
			Query:       "/multiranges?m4=ov.{[6,9)}&select=id",
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
		{
			Description: "cd with a multirange literal",
			Query:       "/multiranges?m4=cd.{[0,10)}&select=id&order=id",
			Expected:    `[{"id":1},{"id":2},{"id":3}]`,
			Status:      200,
		},
		{
			Description: "adj with a multirange literal",
			Query:       "/multiranges?m4=adj.{[7,9)}&select=id",
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
		{
			Description: "eq with the empty multirange",
			Query:       "/multiranges?m4=eq.{}&select=id",
			Expected:    `[{"id":3}]`,
			Status:      200,
		},
		{
			Description: "cs with a bare element is a 22P02, as with PostgREST",
			Query:       "/multiranges?m4=cs.5&select=id",
			Status:      400,
		},
		// CSV: PostgreSQL's text for the multirange, quoted for its commas
		// and quotes as PostgREST's CSV (the record text) quotes it.
		{
			Description: "csv",
			Query:       "/multiranges?id=eq.1&select=r4,m4,mts,mt",
			Headers:     test.Headers{"Accept": {"text/csv"}},
			Expected: "r4,m4,mts,mt\n" +
				`"[1,10)","{[1,3),[5,7)}","{[""2024-01-01 10:00:00"",""2024-06-01 12:00:00"")}","{[""a,b"",""c""""d""],[x,y)}"`,
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
				if tc.Status == 400 && !strings.Contains(string(body), `"code":"22P02"`) {
					t.Errorf("expected a 22P02 in the body, got %s", body)
				}
			}
			test.Execute(t, testConfig, []test.Test{tc})
		})
	}

	// $info reports the column types by name, the array as PostgreSQL's
	// udt_name.
	t.Run("$info", func(t *testing.T) {
		body, _, status, err := test.Exec(client, testConfig, &test.Command{Query: "/$info/multiranges"})
		if err != nil {
			t.Fatal(err)
		}
		if status != 200 {
			t.Fatalf("expected 200, got %d: %s", status, body)
		}
		var table struct {
			Columns []struct{ Name, Type string } `json:"columns"`
		}
		if err := json.Unmarshal(body, &table); err != nil {
			t.Fatalf("%v: %s", err, body)
		}
		types := map[string]string{}
		for _, c := range table.Columns {
			types[c.Name] = c.Type
		}
		for name, want := range map[string]string{"r4": "int4range", "m4": "int4multirange", "mt": "textmultirange", "m4_arr": "_int4multirange"} {
			if types[name] != want {
				t.Errorf("column %s: expected type %q, got %q (columns: %v)", name, want, types[name], types)
			}
		}
	})
}
