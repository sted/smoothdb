package test_api

import (
	"strings"
	"testing"

	"github.com/sted/smoothdb/test"
)

// createRPCParityFixtures builds the functions of TestRPCParity in public:
// copies of the PostgREST fixtures it needs (test/postgrest/fixtures), and
// a VOLATILE set-returning function that writes.
func createRPCParityFixtures(t *testing.T) {
	execSQLAndReload(t, `
		DROP TABLE IF EXISTS rpcp_counters, rpcp_favs CASCADE;
		DROP TYPE IF EXISTS rpcp_complex CASCADE;
		DROP FUNCTION IF EXISTS rpcp_sayhello, rpcp_three_defaults, rpcp_noparams,
			rpcp_unnamed_json, rpcp_record, rpcp_setof_record, rpcp_void,
			rpcp_setof_ints, rpcp_bump, rpcp_bump_one, rpcp_favs;
		CREATE TABLE rpcp_counters (id int, counter int);
		INSERT INTO rpcp_counters VALUES (1, 1), (2, 2);
		CREATE TYPE rpcp_complex AS (r float8, i float8);
		CREATE TABLE rpcp_favs (id int, num rpcp_complex);
		INSERT INTO rpcp_favs VALUES (1, row(0.5, 1)), (2, row(0.6, 2));
		CREATE FUNCTION rpcp_favs() RETURNS SETOF rpcp_favs LANGUAGE sql STABLE AS 'SELECT * FROM rpcp_favs';
		CREATE FUNCTION rpcp_sayhello(name text) RETURNS text LANGUAGE sql IMMUTABLE AS $$ SELECT 'Hello, ' || name $$;
		CREATE FUNCTION rpcp_three_defaults(a int DEFAULT 1, b int DEFAULT 2, c int DEFAULT 3) RETURNS int LANGUAGE sql AS 'SELECT a + b + c';
		CREATE FUNCTION rpcp_noparams() RETURNS text LANGUAGE sql AS $$ SELECT 'none' $$;
		CREATE FUNCTION rpcp_unnamed_json(json) RETURNS json LANGUAGE sql AS 'SELECT $1';
		CREATE FUNCTION rpcp_record() RETURNS record LANGUAGE sql AS 'SELECT * FROM rpcp_counters ORDER BY id LIMIT 1';
		CREATE FUNCTION rpcp_setof_record() RETURNS SETOF record LANGUAGE sql AS 'SELECT * FROM rpcp_counters ORDER BY id';
		CREATE FUNCTION rpcp_void() RETURNS void LANGUAGE sql AS '';
		CREATE FUNCTION rpcp_setof_ints() RETURNS SETOF int LANGUAGE sql AS 'VALUES (1), (2), (3)';
		CREATE FUNCTION rpcp_bump() RETURNS SETOF rpcp_counters LANGUAGE sql VOLATILE AS
			'UPDATE rpcp_counters SET counter = counter + 10 RETURNING *';
		CREATE FUNCTION rpcp_bump_one(row_id int) RETURNS SETOF rpcp_counters LANGUAGE sql VOLATILE AS
			'UPDATE rpcp_counters SET counter = counter + 100 WHERE id = row_id RETURNING *';`)
}

// TestRPCParity checks the function calls of PostgREST 16.4 that its spec
// does not cover, against the behaviour of PostgREST 16.4 run live.
func TestRPCParity(t *testing.T) {
	createRPCParityFixtures(t)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	object := test.Headers{"Accept": {"application/vnd.pgrst.object+json"}}
	text := test.Headers{"Content-Type": {"text/plain"}}
	tests := []test.Test{
		// With TransactionMode "none" (the suite's mode), a VOLATILE call
		// refused as a single object leaves the rows as they were, as in
		// PostgREST (MainTx.hs: the transaction is condemned).
		{
			Description: "a writing call refused as an object",
			Method:      "POST",
			Query:       "/rpc/rpcp_bump",
			Body:        `{}`,
			Headers:     object,
			Status:      406,
		},
		{
			Description: "the refused call changed no row",
			Query:       "/rpcp_counters?select=id,counter&order=id",
			Expected:    `[{"id": 1, "counter": 1}, {"id": 2, "counter": 2}]`,
			Status:      200,
		},
		{
			Description: "a writing call accepted as an object",
			Method:      "POST",
			Query:       "/rpc/rpcp_bump_one",
			Body:        `{"row_id": 1}`,
			Headers:     object,
			Expected:    `{"id": 1, "counter": 101}`,
			Status:      200,
		},
		{
			Description: "the accepted call persisted",
			Query:       "/rpcp_counters?select=id,counter&order=id",
			Expected:    `[{"id": 1, "counter": 101}, {"id": 2, "counter": 2}]`,
			Status:      200,
		},
		// A raw body reaches only a single unnamed parameter (Plan.hs findProc).
		{
			Description: "a text body for a named parameter: no function",
			Method:      "POST",
			Query:       "/rpc/rpcp_sayhello",
			Body:        `world`,
			Headers:     text,
			Status:      404,
		},
		{
			Description: "a text body for parameters with defaults: the defaults",
			Method:      "POST",
			Query:       "/rpc/rpcp_three_defaults",
			Body:        `x`,
			Headers:     text,
			Expected:    `6`,
			Status:      200,
		},
		{
			Description: "a bytea body for parameters with defaults: the defaults",
			Method:      "POST",
			Query:       "/rpc/rpcp_three_defaults",
			Body:        `x`,
			Headers:     test.Headers{"Content-Type": {"application/octet-stream"}},
			Expected:    `6`,
			Status:      200,
		},
		// An empty array is a call with no argument (Payload.hs).
		{
			Description: "an empty array body calls the function",
			Method:      "POST",
			Query:       "/rpc/rpcp_noparams",
			Body:        `[]`,
			Expected:    `"none"`,
			Status:      200,
		},
		{
			Description: "an empty array body with return=representation calls the function",
			Method:      "POST",
			Query:       "/rpc/rpcp_noparams",
			Body:        `[]`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `"none"`,
			Status:      200,
		},
		{
			Description: "an empty array body goes to an unnamed json parameter",
			Method:      "POST",
			Query:       "/rpc/rpcp_unnamed_json",
			Body:        `[]`,
			Expected:    `[]`,
			Status:      200,
		},
		// columns= does not drop the body of an unnamed parameter
		{
			Description: "columns= with an unnamed json parameter",
			Method:      "POST",
			Query:       "/rpc/rpcp_unnamed_json?columns=x",
			Body:        `{"x": 1}`,
			Expected:    `{"x": 1}`,
			Status:      200,
		},
		// Only vnd.pgrst.object checks the number of rows (MainTx.hs).
		{
			Description: "a record function with limit=0",
			Query:       "/rpc/rpcp_record?limit=0",
			Expected:    `null`,
			Status:      200,
		},
		{
			Description: "a scalar function with limit=0",
			Query:       "/rpc/rpcp_sayhello?name=x&limit=0",
			Expected:    `null`,
			Status:      200,
		},
		{
			Description:   "a void function with limit=0",
			Query:         "/rpc/rpcp_void?limit=0",
			ExpectedEmpty: true,
			Status:        204,
		},
		// A set of scalars as an object keeps its column (SqlFragment.hs asJsonSingleF)
		{
			Description: "setof int as an object",
			Query:       "/rpc/rpcp_setof_ints?limit=1",
			Headers:     object,
			Expected:    `{"pgrst_scalar": 1}`,
			Status:      200,
		},
		// The value of a record function is the column pgrst_scalar
		{
			Description: "select=pgrst_scalar on a record function",
			Query:       "/rpc/rpcp_record?select=pgrst_scalar",
			Expected:    `{"id": 1, "counter": 101}`,
			Status:      200,
		},
		{
			Description: "a filter on pgrst_scalar of a setof record function",
			Query:       "/rpc/rpcp_setof_record?pgrst_scalar=not.is.null",
			Expected:    `[{"id": 1, "counter": 101}, {"id": 2, "counter": 2}]`,
			Status:      200,
		},
		{
			Description: "a column of a record function that does not exist",
			Query:       "/rpc/rpcp_record?select=id",
			Status:      400,
		},
		{
			Description: "a record function as CSV",
			Query:       "/rpc/rpcp_setof_record",
			Headers:     test.Headers{"Accept": {"text/csv"}},
			Expected:    "pgrst_scalar\n\"(1,101)\"\n\"(2,2)\"",
			Status:      200,
		},
		// A JSON path on a composite column of a function that returns a table
		{
			Description: "a filter on a composite path of setof a table",
			Query:       "/rpc/rpcp_favs?num->>r=eq.0.5&select=id",
			Expected:    `[{"id": 1}]`,
			Status:      200,
		},
		{
			Description: "select and order on a composite path of setof a table",
			Query:       "/rpc/rpcp_favs?select=num->r&order=num->>i.desc",
			Expected:    `[{"r": 0.6}, {"r": 0.5}]`,
			Status:      200,
		},
		// fts(lang) without the dot is no operator: an argument
		{
			Description: "an argument that looks like fts(lang)",
			Query:       "/rpc/rpcp_sayhello?name=fts(english)",
			Expected:    `"Hello, fts(english)"`,
			Status:      200,
		},
		// columns= leaves out every other key of the body, "" too (Payload.hs)
		{
			Description: "columns= leaves out an empty key",
			Method:      "POST",
			Query:       "/rpc/rpcp_sayhello?columns=name",
			Body:        `{"name": "world", "": 123}`,
			Expected:    `"Hello, world"`,
			Status:      200,
		},
		// a set of records as one object keeps its column, as a set of scalars
		{
			Description: "setof record as an object",
			Query:       "/rpc/rpcp_setof_record?limit=1",
			Headers:     object,
			Expected:    `{"pgrst_scalar": {"id": 1, "counter": 101}}`,
			Status:      200,
		},
		{
			Description: "a record as an object",
			Query:       "/rpc/rpcp_record",
			Headers:     object,
			Expected:    `{"id": 1, "counter": 101}`,
			Status:      200,
		},
		// a projection of a record result names its own columns (SqlFragment.hs)
		{
			Description: "setof record projected, as an object",
			Query:       "/rpc/rpcp_setof_record?select=id:pgrst_scalar->id,counter:pgrst_scalar->counter&limit=1",
			Headers:     object,
			Expected:    `{"id": 1, "counter": 101}`,
			Status:      200,
		},
		// a JSON path on pgrst_scalar reads the record as JSON (Plan.hs)
		{
			Description: "a filter on a JSON path of pgrst_scalar",
			Query:       "/rpc/rpcp_setof_record?pgrst_scalar->>id=eq.2",
			Expected:    `[{"id": 2, "counter": 2}]`,
			Status:      200,
		},
	}
	// one case at a time: Execute stops at the first failure
	for _, tt := range tests {
		test.Execute(t, testConfig, []test.Test{tt})
	}
}

// TestRPCRecordSelect checks the response of the functions that return
// record against a table of PostgREST 16.4, run live: each select, as JSON
// and as one object. The response is the column pgrst_scalar of each row
// (asJsonF, asJsonSingleF: the routine is a scalar), but for one object of
// a SETOF record, which is the whole row. A select without a column
// pgrst_scalar fails there (400, 42703).
func TestRPCRecordSelect(t *testing.T) {
	createRPCParityFixtures(t)
	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	object := test.Headers{"Accept": {"application/vnd.pgrst.object+json"}}
	row := `{"id": 1, "counter": 1}`
	row2 := `{"id": 2, "counter": 2}`
	cases := []struct {
		fn, sel      string
		json, single string // the expected body, "" for a 400
	}{
		{"rpcp_record", "", row, row},
		{"rpcp_record", "*", row, row},
		{"rpcp_record", "pgrst_scalar", row, row},
		{"rpcp_record", "rec:pgrst_scalar", "", ""},
		{"rpcp_record", "pgrst_scalar:pgrst_scalar->id", "1", "1"},
		{"rpcp_record", "pgrst_scalar->id", "", ""},
		{"rpcp_record", "id:pgrst_scalar->id,counter:pgrst_scalar->counter", "", ""},
		{"rpcp_setof_record", "", "[" + row + "]", `{"pgrst_scalar": ` + row + `}`},
		{"rpcp_setof_record", "*", "[" + row + "]", `{"pgrst_scalar": ` + row + `}`},
		{"rpcp_setof_record", "pgrst_scalar", "[" + row + "]", `{"pgrst_scalar": ` + row + `}`},
		{"rpcp_setof_record", "rec:pgrst_scalar", "", `{"rec": ` + row + `}`},
		{"rpcp_setof_record", "pgrst_scalar:pgrst_scalar->id", "[1]", `{"pgrst_scalar": 1}`},
		{"rpcp_setof_record", "pgrst_scalar->id", "", `{"id": 1}`},
		{"rpcp_setof_record", "id:pgrst_scalar->id,counter:pgrst_scalar->counter", "", row},
		// the record stays a record in the read: order, filters, casts,
		// unknown columns and aggregates act as in PostgreSQL ("=": the
		// parameters as they are)
		{"rpcp_record", "order=pgrst_scalar->id.desc", row, row},
		{"rpcp_record", "pgrst_scalar->id=eq.2", "null", "null"},
		{"rpcp_record", "pgrst_scalar::text", `"(1,1)"`, `"(1,1)"`},
		{"rpcp_record", "pgrst_scalar,missing", "", ""},
		{"rpcp_record", "pgrst_scalar,count()", row, row},
		{"rpcp_setof_record", "order=pgrst_scalar->id.desc", "[" + row2 + "]", `{"pgrst_scalar": ` + row2 + `}`},
		{"rpcp_setof_record", "pgrst_scalar->id=eq.2", "[" + row2 + "]", `{"pgrst_scalar": ` + row2 + `}`},
		{"rpcp_setof_record", "pgrst_scalar::text", `["(1,1)"]`, `{"pgrst_scalar": "(1,1)"}`},
		{"rpcp_setof_record", "pgrst_scalar,missing", "", ""},
		{"rpcp_setof_record", "pgrst_scalar,count()", "[" + row + "]", `{"pgrst_scalar": ` + row + `, "count": 1}`},
	}
	for _, c := range cases {
		query := "/rpc/" + c.fn + "?limit=1"
		if c.sel != "" {
			if strings.Contains(c.sel, "=") {
				query += "&" + c.sel
			} else {
				query += "&select=" + c.sel
			}
		}
		for _, single := range []bool{false, true} {
			tt := test.Test{Description: query, Query: query, Expected: c.json, Status: 200}
			if single {
				tt.Description += " as an object"
				tt.Headers = object
				tt.Expected = c.single
			}
			if tt.Expected == "" {
				tt.Status = 400
			}
			test.Execute(t, testConfig, []test.Test{tt})
		}
	}
}
