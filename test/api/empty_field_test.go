package test_api

import (
	"net/url"
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestEmptyFieldNameIsRefused: a filter on a quoted empty name ("") was
// parsed as a field with no name, and the builder dropped it as if it were a
// boolean node, so DELETE /t?""=eq.1 deleted every row. It is a 400 on every
// write and read, and the rows stay. A filter on an embed that a write does
// not select, and the recursive filters on a write, were dropped the same
// way; they are refused too.
func TestEmptyFieldNameIsRefused(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	test.Prepare(cmdConfig, []test.Command{
		{Method: "DELETE", Query: "/databases/dbtest/tables/emptyfield"},
		{Method: "POST", Query: "/databases/dbtest/tables", Body: `{
			"name": "emptyfield",
			"columns": [{"name": "id", "type": "integer"}, {"name": "n", "type": "integer"}]}`},
	})

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	rows := test.Test{
		Description: "the rows are unchanged",
		Method:      "GET",
		Query:       "/emptyfield?select=id,n&order=id",
		Expected:    `[{"id": 1, "n": 1}, {"id": 2, "n": 2}]`,
		Status:      200,
	}
	test.Execute(t, testConfig, []test.Test{
		{
			Description: "insert records",
			Method:      "POST",
			Query:       "/emptyfield",
			Body:        `[{"id": 1, "n": 1}, {"id": 2, "n": 2}]`,
			Status:      201,
		},
		{
			Description: `DELETE with a filter on "": refused`,
			Method:      "DELETE",
			Query:       `/emptyfield?%22%22=eq.1`,
			Status:      400,
		},
		rows,
		{
			Description: `PATCH with a filter on "" inside or: refused`,
			Method:      "PATCH",
			Query:       `/emptyfield?or=(%22%22.eq.1)`,
			Body:        `{"n": 0}`,
			Status:      400,
		},
		rows,
		{
			Description: `GET with a filter on "": refused`,
			Method:      "GET",
			Query:       `/emptyfield?%22%22=eq.1`,
			Status:      400,
		},
		{
			Description: "DELETE with a filter on an embed that is not selected: refused",
			Method:      "DELETE",
			Query:       `/emptyfield?x.id=eq.1`,
			Status:      400,
		},
		rows,
		{
			Description: "PATCH with a filter on an embed that is not selected: refused",
			Method:      "PATCH",
			Query:       `/emptyfield?x.id=eq.1`,
			Body:        `{"n": 0}`,
			Status:      400,
		},
		rows,
		{
			Description: `DELETE with a boolean filter on the embed "": refused`,
			Method:      "DELETE",
			Query:       `/emptyfield?%22%22.or=(id.eq.1)`,
			Status:      400,
		},
		rows,
		{
			Description: "DELETE with recursive filters: refused",
			Method:      "DELETE",
			Query:       `/emptyfield?id=start.1&n=recurse.all`,
			Status:      400,
		},
		rows,
		{
			Description: "PATCH with recursive filters: refused",
			Method:      "PATCH",
			Query:       `/emptyfield?id=start.1&n=recurse.all`,
			Body:        `{"n": 0}`,
			Status:      400,
		},
		rows,
		{
			Description: "jq PATCH with a filter on an embed: refused",
			Method:      "PATCH",
			Query:       "/emptyfield?x.id=eq.1&jq=" + url.QueryEscape(".n = 0"),
			Status:      400,
		},
		rows,
		{
			Description: `PATCH with select="" and a representation: refused`,
			Method:      "PATCH",
			Query:       `/emptyfield?id=eq.1&select=%22%22`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Body:        `{"n": 0}`,
			Status:      400,
		},
		rows,
		{
			Description: "positive control: DELETE with a real filter deletes one row",
			Method:      "DELETE",
			Query:       `/emptyfield?id=eq.1`,
			Status:      204,
		},
	})
}
