package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// Tests for issue #19: JSONB filter behavior differs from PostgREST.
// Covers JSON-path filters with quoted string values, containment with
// JSON arrays of strings, and whole-column JSONB equality/inequality.

func TestJSONBFilters(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "jsonb_filters",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true},
					{"name": "payload", "type": "jsonb"}
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

	tests := []test.Test{
		{
			Description: "insert jsonb records",
			Method:      "POST",
			Query:       "/jsonb_filters",
			Body: `[
				{"id": 1, "payload": {"status":"red","nums":[1,2,3],"tags":["red","blue"],"mixed":["red",1,true]}},
				{"id": 2, "payload": {"status":"blue","flag":"true"}},
				{"id": 3, "payload": {"status":"green","flag":true}}
			]`,
			Status: 201,
		},
		// issue #19 repro 1
		{
			Description: "json path scalar string equality with quoted value",
			Query:       `/jsonb_filters?payload->status=eq."red"&select=id`,
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
		// issue #19 repro 2
		{
			Description: "json path string array containment",
			Query:       `/jsonb_filters?payload->tags=cs.["red"]&select=id`,
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
		// issue #19 repro 3
		{
			Description: "whole-column jsonb equality",
			Query:       `/jsonb_filters?payload=eq.{"status":"red","nums":[1,2,3],"tags":["red","blue"],"mixed":["red",1,true]}&select=id`,
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
		// issue #19 repro 4
		{
			Description: "whole-column jsonb inequality",
			Query:       `/jsonb_filters?payload=neq.{"status":"red","nums":[1,2,3],"tags":["red","blue"],"mixed":["red",1,true]}&select=id&order=id`,
			Expected:    `[{"id":2},{"id":3}]`,
			Status:      200,
		},
		// regressions: these work today and must keep working
		{
			Description: "json path numeric array containment",
			Query:       `/jsonb_filters?payload->nums=cs.[1]&select=id`,
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
		{
			Description: "json path text equality with double arrow",
			Query:       `/jsonb_filters?payload->>status=eq.red&select=id`,
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
		{
			Description: "json path boolean equality keeps matching booleans",
			Query:       `/jsonb_filters?payload->flag=eq.true&select=id`,
			Expected:    `[{"id":3}]`,
			Status:      200,
		},
		// quoted "true" must mean the JSON string, not the boolean
		{
			Description: "json path quoted true matches the string not the boolean",
			Query:       `/jsonb_filters?payload->flag=eq."true"&select=id`,
			Expected:    `[{"id":2}]`,
			Status:      200,
		},
		// same code paths, extra coverage
		{
			Description: "json path IN with quoted values",
			Query:       `/jsonb_filters?payload->status=in.("red","blue")&select=id&order=id`,
			Expected:    `[{"id":1},{"id":2}]`,
			Status:      200,
		},
		{
			Description: "whole-column jsonb containment",
			Query:       `/jsonb_filters?payload=cs.{"status":"red"}&select=id`,
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}
