package test_api

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sted/smoothdb/test"
)

func TestFunctions(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		// drop schema
		{
			Method: "DELETE",
			Query:  "/dbtest/schemas/functions",
		},
		// create schema
		{
			Method: "POST",
			Query:  "/dbtest/schemas",
			Body: `{ 
				"name": "functions"
			}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl: "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken},
			"Content-Profile": {"functions"}},
	}

	tests := []test.Test{
		{
			Description: "create a function",
			Method:      "POST",
			Query:       "http://localhost:8082/admin/databases/dbtest/functions",
			Body: `{
				"name": "f1",
				"arguments": [
					{"name": "a", "type": "integer"},
					{"name": "b", "type": "text"}
				],
				"returns": "table(a int, b text)",
				"definition": "select $1, $2"
			}`,
			Status: 201,
		},
		{
			Description: "exec a function",
			Method:      "POST",
			Query:       "/rpc/f1",
			Body: `{
				"a": 42,
				"b": "Wow"
			}`,
			Expected: `[{"a":42,"b":"Wow"}]`,
			Status:   200,
		},
	}

	test.Execute(t, testConfig, tests)
}

// GET and HEAD are safe methods: a function that writes must not be callable
// through them, whatever the proxies and prefetchers between the client and
// smoothdb do with a GET. PostgREST answers 405 (RpcSpec.hs, context "only for
// GET rpc" / "should fail on mutating procs"); the STABLE and IMMUTABLE
// functions are the positive controls, GET being their proper method.
func TestFunctionsMutatingGet(t *testing.T) {
	// created through SQL because the /admin functions API has no volatility option
	execSQLAndReload(t, `
		DROP SCHEMA IF EXISTS rpc_methods CASCADE;
		CREATE SCHEMA rpc_methods;
		CREATE TABLE rpc_methods.audit (note text);
		CREATE FUNCTION rpc_methods.write_audit(note text) RETURNS text
			LANGUAGE sql VOLATILE AS $$
			INSERT INTO rpc_methods.audit (note) VALUES ($1) RETURNING note
		$$;
		CREATE FUNCTION rpc_methods.plus_one(x int) RETURNS int
			LANGUAGE sql STABLE AS $$ SELECT $1 + 1 $$;
		CREATE FUNCTION rpc_methods.twice(x int) RETURNS int
			LANGUAGE sql IMMUTABLE AS $$ SELECT $1 * 2 $$;
	`)

	testConfig := test.Config{
		BaseUrl: "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{
			"Authorization":   {adminToken},
			"Accept-Profile":  {"rpc_methods"},
			"Content-Profile": {"rpc_methods"},
		},
	}

	tests := []test.Test{
		{
			Description:     "GET on a function that writes is refused",
			Method:          "GET",
			Query:           "/rpc/write_audit?note=via-get",
			ExpectedHeaders: map[string]string{"Allow": "POST"},
			Status:          405,
		},
		{
			Description:     "HEAD on a function that writes is refused",
			Method:          "HEAD",
			Query:           "/rpc/write_audit?note=via-head",
			ExpectedHeaders: map[string]string{"Allow": "POST"},
			Status:          405,
		},
		{
			Description: "the refused requests did not write",
			Method:      "GET",
			Query:       "/audit?select=note",
			Expected:    `[]`,
			Status:      200,
		},
		{
			Description: "POST on the same function works",
			Method:      "POST",
			Query:       "/rpc/write_audit",
			Body:        `{"note": "via-post"}`,
			Expected:    `"via-post"`,
			Status:      200,
		},
		{
			Description: "and it did write",
			Method:      "GET",
			Query:       "/audit?select=note",
			Expected:    `[{"note":"via-post"}]`,
			Status:      200,
		},
		{
			Description: "GET on a STABLE function works",
			Method:      "GET",
			Query:       "/rpc/plus_one?x=41",
			Expected:    `42`,
			Status:      200,
		},
		{
			Description: "GET on an IMMUTABLE function works",
			Method:      "GET",
			Query:       "/rpc/twice?x=21",
			Expected:    `42`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}

// execSQLAndReload runs DDL on dbtest behind smoothdb's back (the /admin API
// has no volatility, sequence or view options) and refreshes the schema cache.
func execSQLAndReload(t *testing.T, sql string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, "postgresql://postgres:postgres@localhost:5432/dbtest")
	if err != nil {
		t.Fatalf("cannot connect to dbtest: %v", err)
	}
	defer conn.Close(ctx)
	if _, err = conn.Exec(ctx, sql); err != nil {
		t.Fatalf("cannot create the fixtures: %v", err)
	}
	db, err := srv.GetDatabase(ctx, "dbtest")
	if err != nil {
		t.Fatalf("cannot get dbtest: %v", err)
	}
	if err := db.ReloadSchemaCache(ctx); err != nil {
		t.Fatalf("cannot reload the schema cache: %v", err)
	}
}

// createReadOnlyFixtures builds the ro_semantics schema: a table, a VOLATILE
// writer, a STABLE function that writes through it (PostgreSQL accepts the
// marker: volatility is a promise, not a check), a VOLATILE function that only
// reads, and the view of the PostgREST transactions docs, whose expression
// calls nextval().
func createReadOnlyFixtures(t *testing.T) {
	t.Helper()
	execSQLAndReload(t, `
		DROP SCHEMA IF EXISTS ro_semantics CASCADE;
		CREATE SCHEMA ro_semantics;
		CREATE TABLE ro_semantics.audit (note text);
		CREATE FUNCTION ro_semantics.write_audit(note text) RETURNS text
			LANGUAGE sql VOLATILE AS $$
			INSERT INTO ro_semantics.audit (note) VALUES ($1) RETURNING note
		$$;
		CREATE FUNCTION ro_semantics.stable_liar(note text) RETURNS text
			LANGUAGE sql STABLE AS $$ SELECT ro_semantics.write_audit($1) $$;
		CREATE FUNCTION ro_semantics.stable_count() RETURNS bigint
			LANGUAGE sql STABLE AS $$ SELECT count(*) FROM ro_semantics.audit $$;
		CREATE SEQUENCE ro_semantics.callcounter_count START 1;
		CREATE VIEW ro_semantics.callcounter AS
			SELECT nextval('ro_semantics.callcounter_count') AS n;
		CREATE FUNCTION ro_semantics.seq_state() RETURNS TABLE(last_value bigint, is_called boolean)
			LANGUAGE sql VOLATILE AS $$
			SELECT last_value, is_called FROM ro_semantics.callcounter_count
		$$;
	`)
}

// GET and HEAD run in a READ ONLY transaction, as in PostgREST
// (docs/references/transactions.rst "Access Mode"; Plan.hs callReadPlan:
// InvRead -> SQL.Read whatever the volatility). Volatility is not the gate:
// a VOLATILE function that only reads answers 200 to GET, while anything that
// writes — a view calling nextval(), the docs' own example — fails inside the
// transaction with SQLSTATE 25006, mapped to 405 (Error.hs mapSQLtoHTTP).
// A POST also runs READ ONLY when the function is STABLE or IMMUTABLE
// (Plan.hs: Inv + Stable/Immutable -> SQL.Read), so a STABLE function that
// writes is refused whatever the method.
func TestReadOnlyRequests(t *testing.T) {
	createReadOnlyFixtures(t)

	testConfig := test.Config{
		BaseUrl: "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{
			"Authorization":   {adminToken},
			"Accept-Profile":  {"ro_semantics"},
			"Content-Profile": {"ro_semantics"},
		},
	}

	tests := []test.Test{
		{
			Description:     "GET on a view whose expression calls nextval() is refused",
			Method:          "GET",
			Query:           "/callcounter",
			ExpectedHeaders: map[string]string{"Allow": "POST"},
			Status:          405,
		},
		{
			Description:     "HEAD on the same view is refused",
			Method:          "HEAD",
			Query:           "/callcounter",
			ExpectedHeaders: map[string]string{"Allow": "POST"},
			Status:          405,
		},
		{
			// the sequence did not advance, and a VOLATILE function that only
			// reads is callable with GET: PostgREST gates on the transaction,
			// not on the volatility marker
			Description: "GET on a VOLATILE function that only reads works, and the sequence did not advance",
			Method:      "GET",
			Query:       "/rpc/seq_state",
			Expected:    `[{"last_value":1,"is_called":false}]`,
			Status:      200,
		},
		{
			Description:     "POST on a STABLE function that writes is refused",
			Method:          "POST",
			Query:           "/rpc/stable_liar",
			Body:            `{"note": "via-stable"}`,
			ExpectedHeaders: map[string]string{"Allow": "POST"},
			Status:          405,
		},
		{
			Description: "GET on a STABLE function that reads works",
			Method:      "GET",
			Query:       "/rpc/stable_count",
			Expected:    `0`,
			Status:      200,
		},
		{
			Description: "POST on a STABLE function that reads works",
			Method:      "POST",
			Query:       "/rpc/stable_count",
			Body:        `{}`,
			Expected:    `0`,
			Status:      200,
		},
		{
			Description: "the STABLE function did not write",
			Method:      "GET",
			Query:       "/audit?select=note",
			Expected:    `[]`,
			Status:      200,
		},
		{
			Description: "POST on the VOLATILE writer works after the refused requests",
			Method:      "POST",
			Query:       "/rpc/write_audit",
			Body:        `{"note": "via-post"}`,
			Expected:    `"via-post"`,
			Status:      200,
		},
		{
			Description: "GET on the table works after the write",
			Method:      "GET",
			Query:       "/audit?select=note",
			Expected:    `[{"note":"via-post"}]`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}

// PostgREST accepts only GET, HEAD, POST (and OPTIONS) on /rpc/: any other
// method is refused with 405 before anything runs (ApiRequest.hs getAction,
// InvalidRpcMethod, PGRST101; RpcSpec.hs "unsupported method"). smoothdb
// answered 404. The Allow header lists what smoothdb routes on /rpc/.
func TestRpcMethodNotAllowed(t *testing.T) {
	createReadOnlyFixtures(t)

	testConfig := test.Config{
		BaseUrl: "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{
			"Authorization":   {adminToken},
			"Accept-Profile":  {"ro_semantics"},
			"Content-Profile": {"ro_semantics"},
		},
	}

	tests := []test.Test{}
	for _, method := range []string{"DELETE", "PATCH", "PUT"} {
		tests = append(tests, test.Test{
			Description:     method + " on /rpc/ is refused",
			Method:          method,
			Query:           "/rpc/write_audit",
			Body:            `{"note": "via-` + method + `"}`,
			ExpectedHeaders: map[string]string{"Allow": "GET, HEAD, POST"},
			Status:          405,
		})
	}
	tests = append(tests,
		test.Test{
			Description: "nothing was written",
			Method:      "GET",
			Query:       "/audit?select=note",
			Expected:    `[]`,
			Status:      200,
		},
		// positive control: the same function answers to POST
		test.Test{
			Description: "POST on the same function works",
			Method:      "POST",
			Query:       "/rpc/write_audit",
			Body:        `{"note": "via-post"}`,
			Expected:    `"via-post"`,
			Status:      200,
		},
	)

	test.Execute(t, testConfig, tests)
}
