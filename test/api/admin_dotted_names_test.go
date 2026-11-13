package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// The admin routes qualified a dotted path segment twice (public.s1.st, 400
// "cross-database references"). The segment is the literal object name and
// the schema comes from the profile header only: a table named "a.b" can be
// created, altered and dropped.
func TestAdminDottedNames(t *testing.T) {
	execSQLAndReload(t, `
		DROP SCHEMA IF EXISTS admdot_s CASCADE;
		CREATE SCHEMA admdot_s;`)

	testConfig := test.Config{
		BaseUrl: "http://localhost:8082/admin/databases/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken},
			"Accept-Profile": {"admdot_s"}, "Content-Profile": {"admdot_s"}},
	}
	tests := []test.Test{
		{
			Description: "create a table named a.b",
			Method:      "POST",
			Query:       "/tables",
			Body:        `{"name": "a.b", "columns": [{"name": "id", "type": "int"}, {"name": "n", "type": "int"}]}`,
			Status:      201,
		},
		{
			Description: "get it",
			Method:      "GET",
			Query:       "/tables/a.b",
			Status:      200,
		},
		{
			Description: "comment it",
			Method:      "PATCH",
			Query:       "/tables/a.b",
			Body:        `{"comment": "dotted"}`,
		},
		{
			Description: "add a column",
			Method:      "POST",
			Query:       "/tables/a.b/columns",
			Body:        `{"name": "x", "type": "text"}`,
			Status:      201,
		},
		{
			Description: "alter a column",
			Method:      "PATCH",
			Query:       "/tables/a.b/columns/x",
			Body:        `{"notnull": true}`,
			Status:      200,
		},
		{
			Description: "drop a column",
			Method:      "DELETE",
			Query:       "/tables/a.b/columns/x",
			Status:      200,
		},
		{
			Description: "add a constraint",
			Method:      "POST",
			Query:       "/tables/a.b/constraints",
			Body:        `{"name": "n_pos", "definition": "CHECK (n > 0)"}`,
			Status:      201,
		},
		{
			Description: "drop the constraint",
			Method:      "DELETE",
			Query:       "/tables/a.b/constraints/n_pos",
			Status:      200,
		},
		{
			Description: "create a view named v.1",
			Method:      "POST",
			Query:       "/views",
			Body:        `{"name": "v.1", "definition": "SELECT 1 AS one"}`,
			Status:      201,
		},
		{
			Description: "drop the view",
			Method:      "DELETE",
			Query:       "/views/v.1",
			Status:      200,
		},
		{
			Description: "rename the table to c.d",
			Method:      "PATCH",
			Query:       "/tables/a.b",
			Body:        `{"name": "c.d"}`,
		},
		{
			Description: "drop it",
			Method:      "DELETE",
			Query:       "/tables/c.d",
			Status:      204,
		},
		{
			Description: "it is gone",
			Method:      "GET",
			Query:       "/tables/c.d",
			Status:      404,
		},
	}
	// the PATCH status of a table is the subject of another change: check the
	// two PATCH cases for success only
	for i := range tests {
		if tests[i].Method == "PATCH" && tests[i].Status == 0 {
			tests[i].Status = tablePatchStatus
		}
	}
	test.Execute(t, testConfig, tests)
	execSQLAndReload(t, `DROP SCHEMA admdot_s CASCADE`)
}

// tablePatchStatus is the status of a successful PATCH /tables/:table.
const tablePatchStatus = 201
