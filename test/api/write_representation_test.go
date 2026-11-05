package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestWriteRepresentationRollsBack: with TransactionMode "none" (the suite's
// mode) a write that asks for a representation runs in a transaction of its
// own, ended after the representation is serialized. A representation that
// fails (a single object asked of several rows, a content type the rows do
// not fit) answers its error and leaves the rows as they were, as in
// PostgREST, where the representation is part of the transaction
// (MainTx.hs, failNotSingular: SQL.condemn). Writes without a representation
// autocommit as before.
func TestWriteRepresentationRollsBack(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		{
			Method: "DELETE",
			Query:  "/databases/dbtest/tables/wrep",
		},
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables",
			Body: `{
				"name": "wrep",
				"columns": [
					{"name": "id", "type": "integer"},
					{"name": "counter", "type": "integer"}
				]}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	object := test.Headers{
		"Accept": {"application/vnd.pgrst.object+json"},
		"Prefer": {"return=representation"},
	}
	reset := []test.Test{
		{
			Description: "remove the rows",
			Method:      "DELETE",
			Query:       "/wrep?id=gte.0",
			Status:      204,
		},
		{
			Description: "insert records",
			Method:      "POST",
			Query:       "/wrep",
			Body:        `[{"id": 1, "counter": 1}, {"id": 2, "counter": 2}]`,
			Status:      201,
		},
	}
	unchanged := test.Test{
		Description: "no row changed",
		Method:      "GET",
		Query:       "/wrep?select=id,counter&order=id",
		Expected:    `[{"id": 1, "counter": 1}, {"id": 2, "counter": 2}]`,
		Status:      200,
	}
	cases := []struct {
		name  string
		tests []test.Test
	}{
		// it "raises an error for multiple rows with return=rep" (SingularSpec.hs)
		{"patch two rows as an object", []test.Test{{
			Description: "PATCH of two rows as an object: refused",
			Method:      "PATCH",
			Query:       "/wrep",
			Body:        `{"counter": -1}`,
			Headers:     object,
			Status:      406,
		}, unchanged}},
		// it "raises an error when attempting to delete multiple entities with return=rep"
		{"delete two rows as an object", []test.Test{{
			Description: "DELETE of two rows as an object: refused",
			Method:      "DELETE",
			Query:       "/wrep",
			Headers:     object,
			Status:      406,
		}, unchanged}},
		// it "raises an error when attempting to create multiple entities with return=rep"
		{"bulk post as an object", []test.Test{{
			Description: "bulk POST of two rows as an object: refused",
			Method:      "POST",
			Query:       "/wrep",
			Body:        `[{"id": 3, "counter": 3}, {"id": 4, "counter": 4}]`,
			Headers:     object,
			Status:      406,
		}, unchanged}},
		// it "raises an error for zero rows with return=rep"
		{"patch no row as an object", []test.Test{{
			Description: "PATCH of no row as an object: refused",
			Method:      "PATCH",
			Query:       "/wrep?id=eq.99",
			Body:        `{"counter": -1}`,
			Headers:     object,
			Status:      406,
		}, unchanged}},
		{"representation that does not fit", []test.Test{{
			Description: "a representation that does not fit its content type: refused",
			Method:      "POST",
			Query:       "/wrep",
			Body:        `{"id": 5, "counter": 5}`,
			Headers: test.Headers{
				"Accept": {"application/octet-stream"},
				"Prefer": {"return=representation"},
			},
			Status: 406,
		}, unchanged}},
		{"writes that succeed persist", []test.Test{
			{
				Description: "PATCH of one row as an object: the object",
				Method:      "PATCH",
				Query:       "/wrep?id=eq.1",
				Body:        `{"counter": 10}`,
				Headers:     object,
				Expected:    `{"id": 1, "counter": 10}`,
				Status:      200,
			},
			{
				Description: "POST of one row as an object: the object",
				Method:      "POST",
				Query:       "/wrep",
				Body:        `{"id": 6, "counter": 6}`,
				Headers:     object,
				Expected:    `{"id": 6, "counter": 6}`,
				Status:      201,
			},
			{
				Description: "PATCH with return=representation as an array",
				Method:      "PATCH",
				Query:       "/wrep?id=eq.6",
				Body:        `{"counter": 60}`,
				Headers:     test.Headers{"Prefer": {"return=representation"}},
				Expected:    `[{"id": 6, "counter": 60}]`,
				Status:      200,
			},
			{
				Description: "PATCH without a representation",
				Method:      "PATCH",
				Query:       "/wrep?id=eq.2",
				Body:        `{"counter": 20}`,
				Status:      204,
			},
			{
				Description: "DELETE of one row as an object: the object",
				Method:      "DELETE",
				Query:       "/wrep?id=eq.1",
				Headers:     object,
				Expected:    `{"id": 1, "counter": 10}`,
				Status:      200,
			},
			{
				Description: "the writes persisted",
				Method:      "GET",
				Query:       "/wrep?select=id,counter&order=id",
				Expected:    `[{"id": 2, "counter": 20}, {"id": 6, "counter": 60}]`,
				Status:      200,
			},
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			test.Execute(t, testConfig, append(append([]test.Test{}, reset...), c.tests...))
		})
	}
}
