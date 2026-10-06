package test_api

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sted/smoothdb/test"
)

// bulkBody is a JSON array of n objects {"a": i, "b": i*mult}.
func bulkBody(n, mult int) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for i := 0; i < n; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(`{"a":` + strconv.Itoa(i) + `,"b":` + strconv.Itoa(i*mult) + `}`)
	}
	sb.WriteByte(']')
	return sb.String()
}

// A bulk insert bound one parameter per value, and the PostgreSQL extended
// protocol carries at most 65 535 of them: 33 000 rows of two columns (66 000
// values) answered 500 "extended protocol limited to 65535 parameters". As in
// PostgREST (SqlFragment.hs, fromJsonBodyF) the body is now the statement's
// only parameter, read with json_to_recordset, whatever its size; the upsert
// and the representation paths go through the same statement.
func TestBulkInsertOverParameterLimit(t *testing.T) {
	execSQLAndReload(t, `
		DROP TABLE IF EXISTS bulk_limit;
		CREATE TABLE bulk_limit (a int PRIMARY KEY, b int);
	`)
	const rows = 33000 // × 2 columns = 66 000 values, over 65 535
	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	test.Execute(t, testConfig, []test.Test{
		{
			Description:   "66 000 values in one POST are inserted",
			Method:        "POST",
			Query:         "/bulk_limit",
			Body:          bulkBody(rows, 2),
			ExpectedEmpty: true,
			Status:        201,
		},
		{
			Description:     "every row is there",
			Query:           "/bulk_limit?select=a&order=a&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"a":0}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/33000"},
			Status:          200,
		},
		{
			Description: "with its values",
			Query:       "/bulk_limit?a=eq.32999",
			Expected:    `[{"a":32999,"b":65998}]`,
			Status:      200,
		},
		{
			Description:   "a bulk upsert over the limit merges every row",
			Method:        "POST",
			Query:         "/bulk_limit",
			Body:          bulkBody(rows, 3),
			Headers:       test.Headers{"Prefer": {"resolution=merge-duplicates"}},
			ExpectedEmpty: true,
			Status:        201,
		},
		{
			Description:     "the merge updated the rows and added none",
			Query:           "/bulk_limit?a=eq.32999",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"a":32999,"b":98997}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
			Status:          200,
		},
		{
			Description: "ignore-duplicates with a representation over the limit returns no row",
			Method:      "POST",
			Query:       "/bulk_limit",
			Body:        bulkBody(rows, 4),
			Headers:     test.Headers{"Prefer": {"resolution=ignore-duplicates", "return=representation"}},
			Expected:    `[]`,
			Status:      201,
		},
	})
}

// PostgREST applies ?columns= to a JSON body only (Payload.hs, getPayload):
// a CSV header or a form's fields are the arguments of an RPC, as they are
// the columns of an insert. PostgREST 14.15 on the same calls: 3, 3, 1.
func TestPayloadColumnsOnRPC(t *testing.T) {
	execSQLAndReload(t, `
		CREATE OR REPLACE FUNCTION bulk_probe_sum(a int, b int DEFAULT 0) RETURNS int
			LANGUAGE sql AS $$ SELECT a + b $$;
	`)
	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	test.Execute(t, testConfig, []test.Test{
		{
			Description: "?columns= does not apply to a form body on an RPC",
			Method:      "POST",
			Query:       "/rpc/bulk_probe_sum?columns=a",
			Body:        "a=1&b=2",
			Headers:     test.Headers{"Content-Type": {"application/x-www-form-urlencoded"}},
			Expected:    `3`,
			Status:      200,
		},
		{
			Description: "?columns= does not apply to a CSV body on an RPC",
			Method:      "POST",
			Query:       "/rpc/bulk_probe_sum?columns=a",
			Body:        "a,b\n1,2\n",
			Headers:     test.Headers{"Content-Type": {"text/csv"}},
			Expected:    `3`,
			Status:      200,
		},
		{
			Description: "?columns= applies to a JSON body on an RPC",
			Method:      "POST",
			Query:       "/rpc/bulk_probe_sum?columns=a",
			Body:        `{"a": 1, "b": 2}`,
			Expected:    `1`,
			Status:      200,
		},
	})
}

// With the body as one json parameter the values reach the columns through
// the database's input functions, as in PostgREST, instead of pgx's encoding
// of the decoded Go values: json_to_recordset gets the column definition list
// from the schema cache (format_type, so typmods and domains apply). Each case
// is what PostgREST answers for the same request.
func TestBulkInsertValueConversion(t *testing.T) {
	execSQLAndReload(t, `
		DROP TABLE IF EXISTS bulk_types;
		DROP TABLE IF EXISTS bulk_defaults;
		DROP TYPE IF EXISTS bulk_mood;
		DROP TYPE IF EXISTS bulk_pair;
		DROP DOMAIN IF EXISTS bulk_nn;
		CREATE TYPE bulk_mood AS ENUM ('sad', 'ok', 'happy');
		CREATE TYPE bulk_pair AS (x int, y text);
		CREATE DOMAIN bulk_nn AS text NOT NULL;
		CREATE TABLE bulk_types (
			id int PRIMARY KEY,
			arr int[],
			tags text[],
			jb jsonb,
			j json,
			pair bulk_pair,
			mood bulk_mood,
			price numeric(10,2),
			big numeric,
			ts timestamptz,
			raw bytea,
			note text DEFAULT 'dflt',
			nn bulk_nn DEFAULT 'nn-dflt',
			doubled int GENERATED ALWAYS AS (id * 2) STORED
		);
		CREATE TABLE bulk_defaults (id serial PRIMARY KEY, v text DEFAULT 'v');
	`)
	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	test.Execute(t, testConfig, []test.Test{
		{
			Description: "values of every kind reach their columns",
			Method:      "POST",
			Query:       "/bulk_types?select=id,arr,tags,jb,pair,mood,price,big,note,nn,doubled",
			Body: `[{"id": 1, "arr": [1, 2], "tags": "{a,b}", "jb": {"z": 1, "a": [true, null]},
				"j": {"z": 1,  "a": 2}, "pair": {"x": 1, "y": "one"}, "mood": "happy",
				"price": 1.239, "big": 123456789012345678901234567890.123456789,
				"ts": "2026-10-06T12:00:00+02:00", "raw": "\\x0102"}]`,
			Headers:  test.Headers{"Prefer": {"return=representation"}},
			Expected: `[{"id":1,"arr":[1,2],"tags":["a","b"],"jb":{"a":[true,null],"z":1},"pair":{"x":1,"y":"one"},"mood":"happy","price":1.24,"big":123456789012345678901234567890.123456789,"note":"dflt","nn":"nn-dflt","doubled":2}]`,
			Status:   201,
		},
		{
			// json keeps the text it is given: the body's own, as in PostgREST,
			// not a re-encoding of the decoded value (sorted keys, no spaces)
			Description: "a json column keeps the text of the body",
			Query:       "/bulk_types?select=j::text&id=eq.1",
			Expected:    `[{"j":"{\"z\": 1,  \"a\": 2}"}]`,
			Status:      200,
		},
		{
			Description: "a timestamptz is read by its input function",
			Query:       "/bulk_types?select=id&ts=eq.2026-10-06T10:00:00Z",
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
		{
			Description: "a bytea is read in its hex form",
			Query:       "/bulk_types?select=id&raw=eq.\\x0102",
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
		{
			Description: "a value its column cannot read is the database's 22P02",
			Method:      "POST",
			Query:       "/bulk_types",
			Body:        `[{"id": 2, "mood": "angry"}]`,
			Expected:    `{"subsystem":"database","message":"invalid input value for enum bulk_mood: \"angry\"","code":"22P02","hint":"","details":"","position":0}`,
			Status:      400,
		},
		{
			// PostgREST resolves every payload key against the schema cache
			// (Plan.hs, resolveOrError), as it does the ?columns= list
			Description: "a key that is not a column is refused",
			Method:      "POST",
			Query:       "/bulk_types",
			Body:        `[{"id": 2, "nope": 1}]`,
			Expected:    `{"subsystem":"network","message":"Could not find the 'nope' column of 'bulk_types' in the schema cache","code":"","hint":"","details":null,"position":0}`,
			Status:      400,
		},
		{
			// PostgREST builds ON CONFLICT only for a Prefer: resolution: with
			// on_conflict alone the insert meets the conflict
			Description: "on_conflict without a resolution is a plain insert",
			Method:      "POST",
			Query:       "/bulk_types?on_conflict=id",
			Body:        `{"id": 1}`,
			Expected:    `{"subsystem":"database","message":"duplicate key value violates unique constraint \"bulk_types_pkey\"","code":"23505","hint":"","details":"Key (id)=(1) already exists.","position":0}`,
			Status:      409,
		},
		{
			Description: "a generated column in the body is the database's 428C9",
			Method:      "POST",
			Query:       "/bulk_types",
			Body:        `[{"id": 2, "doubled": 5}]`,
			Expected:    `{"subsystem":"database","message":"cannot insert a non-DEFAULT value into column \"doubled\"","code":"428C9","hint":"","details":"Column \"doubled\" is a generated column.","position":0}`,
			Status:      400,
		},
		{
			// one default row per object (json_array_elements), where
			// INSERT ... DEFAULT VALUES inserted one row for the whole array
			Description: "an array of empty objects inserts one default row each",
			Method:      "POST",
			Query:       "/bulk_defaults",
			Body:        `[{}, {}, {}]`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":1,"v":"v"},{"id":2,"v":"v"},{"id":3,"v":"v"}]`,
			Status:      201,
		},
		{
			Description: "an empty object inserts one default row",
			Method:      "POST",
			Query:       "/bulk_defaults",
			Body:        `{}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":4,"v":"v"}]`,
			Status:      201,
		},
		{
			// a null element is not an object (Payload.hs, payloadAttributes)
			Description: "a null element is refused beside an empty object",
			Method:      "POST",
			Query:       "/bulk_defaults",
			Body:        `[null, {}]`,
			Expected:    `{"subsystem":"network","message":"All object keys must match","code":"","hint":"","details":null,"position":0}`,
			Status:      400,
		},
		{
			// and a body that is neither an array nor an object is no row
			Description: "a null body inserts nothing",
			Method:      "POST",
			Query:       "/bulk_defaults",
			Body:        `null`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[]`,
			Status:      201,
		},
		{
			// with ?columns= the body goes to PostgreSQL as it is (PostgREST's
			// RawJSON), which refuses a scalar
			Description: "a null body with ?columns= is PostgreSQL's 22023",
			Method:      "POST",
			Query:       "/bulk_defaults?columns=v",
			Body:        `null`,
			Expected:    `{"subsystem":"database","message":"cannot call json_to_recordset on a scalar","code":"22023","hint":"","details":"","position":0}`,
			Status:      400,
		},
		{
			Description: "a merge-duplicates upsert of empty objects inserts them",
			Method:      "POST",
			Query:       "/bulk_defaults",
			Body:        `[{}]`,
			Headers:     test.Headers{"Prefer": {"resolution=merge-duplicates", "return=representation"}},
			Expected:    `[{"id":5,"v":"v"}]`,
			Status:      201,
		},
		{
			// PostgREST turns a CSV or form body into JSON whose values are
			// strings (Payload.hs, csvToJson): a json column gets a JSON string,
			// never the JSON the text spells (PostgREST 14.15 on the same rows)
			Description: "a CSV value is a JSON string in json and jsonb columns",
			Method:      "POST",
			Query:       "/bulk_types",
			Body:        "id,j,jb\n12,[123],[123]\n13,abc,\"{\"\"a\"\": 1}\"\n",
			Headers:     test.Headers{"Content-Type": {"text/csv"}},
			Status:      201,
		},
		{
			Description: "the CSV values stored as JSON strings",
			Query:       "/bulk_types?select=id,j::text,jb::text&id=in.(12,13)&order=id",
			Expected:    `[{"id":12,"j":"\"[123]\"","jb":"\"[123]\""},{"id":13,"j":"\"abc\"","jb":"\"{\\\"a\\\": 1}\""}]`,
			Status:      200,
		},
		{
			Description: "a form value is a JSON string in a json column",
			Method:      "POST",
			Query:       "/bulk_types",
			Body:        "id=14&j=%5B1%5D",
			Headers:     test.Headers{"Content-Type": {"application/x-www-form-urlencoded"}},
			Status:      201,
		},
		{
			Description: "the form value stored as a JSON string",
			Query:       "/bulk_types?select=id,j::text&id=eq.14",
			Expected:    `[{"id":14,"j":"\"[1]\""}]`,
			Status:      200,
		},
		// A CSV or form body as PostgREST reads it (Payload.hs, PostgREST 14.15
		// on the same requests): an empty CSV is a 400, ?columns= does not
		// apply to a CSV (its header is the column list), a repeated form key
		// keeps its last value, and the URL's parameters are not form fields.
		{
			Description: "an empty CSV body is a 400",
			Method:      "POST",
			Query:       "/bulk_defaults",
			Body:        "",
			Headers:     test.Headers{"Content-Type": {"text/csv"}},
			Status:      400,
		},
		{
			Description: "?columns= does not apply to a CSV body",
			Method:      "POST",
			Query:       "/bulk_types?columns=id&select=id,note",
			Body:        "id,note\n15,from-csv\n",
			Headers:     test.Headers{"Content-Type": {"text/csv"}, "Prefer": {"return=representation"}},
			Expected:    `[{"id":15,"note":"from-csv"}]`,
			Status:      201,
		},
		{
			Description: "a repeated form key keeps its last value",
			Method:      "POST",
			Query:       "/bulk_types?select=id",
			Body:        "id=16&id=17",
			Headers:     test.Headers{"Content-Type": {"application/x-www-form-urlencoded"}, "Prefer": {"return=representation"}},
			Expected:    `[{"id":17}]`,
			Status:      201,
		},
		{
			// the select above is a URL parameter, not a form field
			Description: "the URL's parameters are not form fields",
			Method:      "POST",
			Query:       "/bulk_types?select=id,note",
			Body:        "id=18&note=n",
			Headers:     test.Headers{"Content-Type": {"application/x-www-form-urlencoded"}, "Prefer": {"return=representation"}},
			Expected:    `[{"id":18,"note":"n"}]`,
			Status:      201,
		},
		{
			// CSV values are strings (NULL is null), read by the same statement
			Description: "a CSV body is inserted",
			Method:      "POST",
			Query:       "/bulk_types?select=id,mood,note",
			Body:        "id,mood,note\n10,ok,NULL\n11,sad,x\n",
			Headers:     test.Headers{"Content-Type": {"text/csv"}, "Prefer": {"return=representation"}},
			Expected:    `[{"id":10,"mood":"ok","note":null},{"id":11,"mood":"sad","note":"x"}]`,
			Status:      201,
		},
	})

	// A table the schema cache does not know yet has no column types to
	// declare: the row type of the table reads the body instead, and a
	// missing table is still PostgreSQL's 42P01.
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, "postgresql://postgres:postgres@localhost:5432/dbtest")
	if err != nil {
		t.Fatalf("cannot connect to dbtest: %v", err)
	}
	_, err = conn.Exec(ctx, `DROP TABLE IF EXISTS bulk_uncached; CREATE TABLE bulk_uncached (id int, v text)`)
	conn.Close(ctx)
	if err != nil {
		t.Fatal(err)
	}
	test.Execute(t, testConfig, []test.Test{
		{
			Description: "a table missing from the schema cache is inserted into",
			Method:      "POST",
			Query:       "/bulk_uncached",
			Body:        `[{"id": 1, "v": "a"}, {"id": 2, "v": null}]`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":1,"v":"a"},{"id":2,"v":null}]`,
			Status:      201,
		},
		{
			Description: "a missing table is 404",
			Method:      "POST",
			Query:       "/bulk_nosuch",
			Body:        `[{"id": 1}]`,
			Status:      404,
		},
	})
}
