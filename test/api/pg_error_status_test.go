package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// A database error that the caller caused must be reported as a client error, the
// way PostgREST reports it. The shapes here all used to answer 500, which left a
// caller unable to tell a malformed filter from a broken server.
func TestDatabaseErrorsAreClientErrors(t *testing.T) {
	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	test.Prepare(cmdConfig, []test.Command{
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "error_status",
				"columns": [
					{"name": "name", "type": "text", "notnull": true},
					{"name": "flag", "type": "boolean"},
					{"name": "n", "type": "int4"}
				],
				"ifnotexists": true
			}`,
		},
	})

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	tests := []test.Test{
		{
			Description: "seed one row",
			Method:      "POST",
			Query:       "/error_status",
			Body:        `[{"name": "a", "flag": true, "n": 1}]`,
			Status:      201,
		},
		// The positive controls: IS TRUE on a boolean and IS NULL on text are legal
		// for Postgres, so the refusals below are about the operand's type and not
		// about `is` as such.
		{
			Description: "IS TRUE on a boolean column is legal",
			Query:       "/error_status?flag=is.true&select=name",
			Expected:    `[{"name": "a"}]`,
			Status:      200,
		},
		{
			Description: "IS NULL on a text column is legal",
			Query:       "/error_status?name=is.null&select=name",
			Expected:    `[]`,
			Status:      200,
		},
		// 42804 datatype_mismatch: Postgres takes IS TRUE / IS FALSE / IS UNKNOWN on
		// a boolean operand only.
		{
			Description: "IS TRUE on a text column is refused, not a server error",
			Query:       "/error_status?name=is.true",
			Status:      400,
		},
		{
			Description: "IS FALSE on a text column is refused",
			Query:       "/error_status?name=is.false",
			Status:      400,
		},
		{
			Description: "IS UNKNOWN on a text column is refused",
			Query:       "/error_status?name=is.unknown",
			Status:      400,
		},
		{
			Description: "IS TRUE inside a logical group is refused the same way",
			Query:       "/error_status?or=(name.is.true,n.eq.1)",
			Status:      400,
		},
		// 42704 undefined_object: the text search configuration is resolved by Postgres.
		{
			Description: "an unknown text search configuration is refused",
			Query:       "/error_status?name=fts(nonesuch).a",
			Status:      400,
		},
		{
			Description: "an unknown text search configuration inside a group is refused",
			Query:       "/error_status?or=(name.fts(nonesuch).a,n.eq.1)",
			Status:      400,
		},
		// 23502 not_null_violation.
		{
			Description: "a null in a not-null column is refused",
			Method:      "POST",
			Query:       "/error_status",
			Body:        `[{"name": null, "n": 2}]`,
			Status:      400,
		},
		// Already client errors before this change, and they must not move.
		{
			Description: "an unparseable number stays a bad request",
			Query:       "/error_status?n=eq.notanumber",
			Status:      400,
		},
		{
			Description: "an unknown column stays a bad request",
			Query:       "/error_status?nosuchcolumn=eq.1",
			Status:      400,
		},
		{
			Description: "an unknown table stays a not found",
			Query:       "/nosuchtable?select=name",
			Status:      404,
		},
	}
	test.Execute(t, testConfig, tests)
}
