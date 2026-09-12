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
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, "postgresql://postgres:postgres@localhost:5432/dbtest")
	if err != nil {
		t.Fatalf("cannot connect to dbtest: %v", err)
	}
	defer conn.Close(ctx)
	// created through SQL because the /admin functions API has no volatility option
	_, err = conn.Exec(ctx, `
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
	if err != nil {
		t.Fatalf("cannot create the rpc_methods fixtures: %v", err)
	}
	// the functions were created behind smoothdb's back: refresh its schema cache
	db, err := srv.GetDatabase(ctx, "dbtest")
	if err != nil {
		t.Fatalf("cannot get dbtest: %v", err)
	}
	if err := db.ReloadSchemaCache(ctx); err != nil {
		t.Fatalf("cannot reload the schema cache: %v", err)
	}

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
