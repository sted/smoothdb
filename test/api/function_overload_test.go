package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// createOverloadFixtures builds the rpc_overload schema: scalar functions,
// one name overloaded with a table and a jsonb return type, one overloaded
// with the same argument name and different types, and one overloaded with a
// STABLE and a VOLATILE signature, both writing through a VOLATILE function.
func createOverloadFixtures(t *testing.T) {
	t.Helper()
	execSQLAndReload(t, `
		DROP SCHEMA IF EXISTS rpc_overload CASCADE;
		CREATE SCHEMA rpc_overload;
		CREATE FUNCTION rpc_overload.ret_int() RETURNS int
			LANGUAGE sql AS $$ SELECT 3 $$;
		CREATE FUNCTION rpc_overload.ret_jsonb() RETURNS jsonb
			LANGUAGE sql AS $$ SELECT '{"inserted": 2, "deleted": 0}'::jsonb $$;
		CREATE FUNCTION rpc_overload.ret_setof_int() RETURNS SETOF int
			LANGUAGE sql AS $$ VALUES (1), (2) $$;
		CREATE FUNCTION rpc_overload.ov(p jsonb) RETURNS TABLE (inserted int, deleted int)
			LANGUAGE sql AS $$ SELECT jsonb_array_length(p), 0 $$;
		CREATE FUNCTION rpc_overload.ov(p jsonb, mode text) RETURNS jsonb
			LANGUAGE sql AS $$
			SELECT jsonb_build_object('inserted', jsonb_array_length(p), 'deleted', 0, 'mode', mode)
		$$;
		CREATE FUNCTION rpc_overload.f_over(a int) RETURNS int
			LANGUAGE sql AS $$ SELECT a $$;
		CREATE FUNCTION rpc_overload.f_over(a text) RETURNS text
			LANGUAGE sql AS $$ SELECT a $$;
		CREATE TABLE rpc_overload.audit (note text);
		CREATE FUNCTION rpc_overload.write_audit(note text) RETURNS text
			LANGUAGE sql VOLATILE AS $$
			INSERT INTO rpc_overload.audit (note) VALUES ($1) RETURNING note
		$$;
		CREATE FUNCTION rpc_overload.log_note(note text) RETURNS text
			LANGUAGE sql STABLE AS $$ SELECT rpc_overload.write_audit(note) $$;
		CREATE FUNCTION rpc_overload.log_note(note text, n int) RETURNS text
			LANGUAGE sql VOLATILE AS $$ SELECT rpc_overload.write_audit(note || n) $$;
	`)
}

// An overloaded function answers with the shape of the overload the call
// invokes, chosen as PostgREST chooses it (Plan.hs findProc): by the argument
// names, an argument with a default being optional. When more than one
// overload accepts the names the call is refused with 300 and PostgREST's
// message and hint (PGRST203, its code left empty for now). The schema cache
// kept one overload per name, which then decided the shape of every call:
// a table overload answered "2,0", not JSON, and a jsonb overload [{"t": …}]
// (card 33869 rpc-scalar), and a POST ran READ ONLY or not by the volatility
// of the cached overload.
func TestFunctionOverloads(t *testing.T) {
	createOverloadFixtures(t)

	testConfig := test.Config{
		BaseUrl: "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{
			"Authorization":   {adminToken},
			"Accept-Profile":  {"rpc_overload"},
			"Content-Profile": {"rpc_overload"},
		},
	}

	const ambiguous = `{"subsystem":"network","message":"Could not choose the best candidate function between: rpc_overload.f_over(a => integer), rpc_overload.f_over(a => text)","code":"","hint":"Try renaming the parameters or the function itself in the database so function overloading can be resolved","details":null,"position":0}`

	// test.Execute stops a list at its first wrong body: one list per
	// behaviour, so that each reports
	scalars := []test.Test{
		{
			Description: "a scalar int function answers the value",
			Method:      "POST",
			Query:       "/rpc/ret_int",
			Body:        `{}`,
			Expected:    `3`,
			Status:      200,
		},
		{
			Description: "a scalar jsonb function answers the object",
			Method:      "POST",
			Query:       "/rpc/ret_jsonb",
			Body:        `{}`,
			Expected:    `{"inserted": 2, "deleted": 0}`,
			Status:      200,
		},
		{
			Description: "a setof int function answers an array of values",
			Method:      "GET",
			Query:       "/rpc/ret_setof_int",
			Expected:    `[1,2]`,
			Status:      200,
		},
	}
	shapes := []test.Test{
		{
			Description: "the table overload answers rows",
			Method:      "POST",
			Query:       "/rpc/ov",
			Body:        `{"p": [1, 2]}`,
			Expected:    `[{"inserted": 2, "deleted": 0}]`,
			Status:      200,
		},
		{
			Description: "the jsonb overload answers the object",
			Method:      "POST",
			Query:       "/rpc/ov",
			Body:        `{"p": [1, 2], "mode": "x"}`,
			Expected:    `{"inserted": 2, "deleted": 0, "mode": "x"}`,
			Status:      200,
		},
		{
			Description: "the table overload answers rows to GET",
			Method:      "GET",
			Query:       "/rpc/ov?p=[1,2,3]",
			Expected:    `[{"inserted": 3, "deleted": 0}]`,
			Status:      200,
		},
		{
			Description: "the jsonb overload answers the object to GET",
			Method:      "GET",
			Query:       "/rpc/ov?p=[1,2,3]&mode=y",
			Expected:    `{"inserted": 3, "deleted": 0, "mode": "y"}`,
			Status:      200,
		},
	}
	ambiguity := []test.Test{
		{
			Description:     "two overloads accepting the same names answer 300",
			Method:          "POST",
			Query:           "/rpc/f_over",
			Body:            `{"a": 5}`,
			Expected:        ambiguous,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          300,
		},
		{
			Description: "and to GET",
			Method:      "GET",
			Query:       "/rpc/f_over?a=5",
			Expected:    ambiguous,
			Status:      300,
		},
	}
	// the transaction is READ ONLY by the volatility of the invoked overload
	volatility := []test.Test{
		{
			Description:     "POST on the STABLE overload that writes is refused",
			Method:          "POST",
			Query:           "/rpc/log_note",
			Body:            `{"note": "stable"}`,
			ExpectedHeaders: map[string]string{"Allow": "POST"},
			Status:          405,
		},
		{
			Description: "POST on the VOLATILE overload writes",
			Method:      "POST",
			Query:       "/rpc/log_note",
			Body:        `{"note": "volatile", "n": 1}`,
			Expected:    `"volatile1"`,
			Status:      200,
		},
		{
			Description: "only the VOLATILE overload wrote",
			Method:      "GET",
			Query:       "/audit?select=note",
			Expected:    `[{"note": "volatile1"}]`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, scalars)
	test.Execute(t, testConfig, shapes)
	test.Execute(t, testConfig, ambiguity)
	test.Execute(t, testConfig, volatility)
}
