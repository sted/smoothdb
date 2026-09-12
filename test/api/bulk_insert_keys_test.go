package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// A bulk insert must not take its column list from the first object: a key
// present only in later objects was silently dropped and a key missing there
// became NULL, with a 201. As in PostgREST (Payload.hs, payloadAttributes) an
// array whose objects do not share one key set is refused with 400 "All object
// keys must match", unless ?columns= names the column set explicitly: then the
// listed columns are inserted for every row, an absent key as NULL, and the
// keys not listed are ignored (Prefer: missing=default is not supported).
func TestBulkInsertKeys(t *testing.T) {
	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	commands := []test.Command{
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "bulk_keys",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true, "constraints": ["PRIMARY KEY"]},
					{"name": "body", "type": "text"},
					{"name": "note", "type": "text", "default": "'dflt'"}
				],
				"ifnotexists": true
			}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	const mismatch = `{"subsystem":"network","message":"All object keys must match","code":"","hint":"","details":null,"position":0}`

	tests := []test.Test{
		// positive control: one key set, in a different order per object, is uniform
		{
			Description: "uniform array with keys in a different order is inserted",
			Method:      "POST",
			Query:       "/bulk_keys",
			Body:        `[{"id": 1, "body": "a"}, {"body": "b", "id": 2}]`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":1,"body":"a","note":"dflt"},{"id":2,"body":"b","note":"dflt"}]`,
			Status:      201,
		},
		// the card's example: the second object carries a key the first one lacks
		{
			Description: "key present only in a later object is refused",
			Method:      "POST",
			Query:       "/bulk_keys",
			Body:        `[{"id": 3}, {"id": 4, "body": "y"}]`,
			Expected:    mismatch,
			Status:      400,
		},
		{
			Description: "key missing from a later object is refused",
			Method:      "POST",
			Query:       "/bulk_keys",
			Body:        `[{"id": 3, "body": "x"}, {"id": 4}]`,
			Expected:    mismatch,
			Status:      400,
		},
		{
			Description: "a null-valued key is still a key",
			Method:      "POST",
			Query:       "/bulk_keys",
			Body:        `[{"id": 3, "body": null}, {"id": 4}]`,
			Expected:    mismatch,
			Status:      400,
		},
		{
			Description: "the same check applies to a bulk upsert",
			Method:      "POST",
			Query:       "/bulk_keys?on_conflict=id",
			Body:        `[{"id": 1, "body": "a2"}, {"id": 4, "body": "y", "note": "n"}]`,
			Headers:     test.Headers{"Prefer": {"resolution=merge-duplicates"}},
			Expected:    mismatch,
			Status:      400,
		},
		{
			Description: "a refused array inserts nothing",
			Query:       "/bulk_keys?select=id,body&order=id",
			Expected:    `[{"id":1,"body":"a"},{"id":2,"body":"b"}]`,
			Status:      200,
		},
		// ?columns= is the explicit opt-in: the listed columns are inserted for
		// every row, an absent key is NULL, keys not listed are ignored
		{
			Description: "?columns= inserts the listed columns for every row",
			Method:      "POST",
			Query:       "/bulk_keys?columns=id,body",
			Body:        `[{"id": 5}, {"id": 6, "body": "y", "bogus": true}]`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":5,"body":null,"note":"dflt"},{"id":6,"body":"y","note":"dflt"}]`,
			Status:      201,
		},
		{
			Description: "?columns= ignores the keys not listed",
			Method:      "POST",
			Query:       "/bulk_keys?columns=id",
			Body:        `[{"id": 7, "body": "dropped"}, {"id": 8}]`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":7,"body":null,"note":"dflt"},{"id":8,"body":null,"note":"dflt"}]`,
			Status:      201,
		},
		// PostgREST inserts a listed column absent from the payload as NULL, not
		// as its default (docs/references/api/preferences.rst, "Missing")
		{
			Description: "?columns= inserts a listed column absent from the payload as NULL",
			Method:      "POST",
			Query:       "/bulk_keys?columns=id,note",
			Body:        `{"id": 9}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":9,"body":null,"note":null}]`,
			Status:      201,
		},
	}
	test.Execute(t, testConfig, tests)
}
