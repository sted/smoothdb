package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// A top-level filter value runs to the end of the query parameter, as in
// PostgREST (pSingleVal): dots, commas, colons, quotes and spaces inside it are
// part of the value. Inside in.(), any/all lists and logic trees a value ends
// at the next separator unless it is quoted (pListElement, pLogicSingleVal).
// Until this was fixed the parser kept one extra dot at most (so that float
// literals survived) and silently dropped the rest, answering 200 with the
// wrong rows: ?ver=eq.1.2.3 filtered on "1.2", ?name=eq.a,b on "a".
func TestFilterValues(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		{
			Method: "POST",
			Query:  "/dbtest/tables",
			Body: `{
				"name": "filter_values",
				"columns": [
					{"name": "id", "type": "int4", "notnull": true},
					{"name": "name", "type": "text"},
					{"name": "ver", "type": "text"}
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
			Description: "insert records",
			Method:      "POST",
			Query:       "/filter_values",
			Body: `[
				{"id": 1, "name": "a,b", "ver": "1.2.3"},
				{"id": 2, "name": "a.b@c.com", "ver": "1.2"},
				{"id": 3, "name": "autoexec.bat", "ver": "1"},
				{"id": 4, "name": "O'Brien", "ver": "2.0.1"},
				{"id": 5, "name": "Sidney K. Meier", "ver": "c.d"},
				{"id": 6, "name": "[draft] v2", "ver": "a.b"},
				{"id": 7, "name": "", "ver": "10:30"},
				{"id": 8, "name": "http://x.y/z?q=1", "ver": "0.5"},
				{"id": 9, "name": "a", "ver": "5\" display"}
			]`,
			Status: 201,
		},
		// --- top-level filters: the value is the whole remainder of the parameter ---
		{
			Description: "semver: more than one dot",
			Query:       "/filter_values?ver=eq.1.2.3&select=id",
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
		{
			Description: "email: dots and an at sign",
			Query:       "/filter_values?name=eq.a.b@c.com&select=id",
			Expected:    `[{"id":2}]`,
			Status:      200,
		},
		{
			Description: "comma in a top-level value",
			Query:       "/filter_values?name=eq.a,b&select=id",
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
		{
			Description: "file name with one dot (worked before: positive control)",
			Query:       "/filter_values?name=eq.autoexec.bat&select=id",
			Expected:    `[{"id":3}]`,
			Status:      200,
		},
		{
			Description: "apostrophe in a value",
			Query:       "/filter_values?name=eq.O'Brien&select=id",
			Expected:    `[{"id":4}]`,
			Status:      200,
		},
		{
			Description: "space after a dot",
			Query:       "/filter_values?name=eq.Sidney K. Meier&select=id",
			Expected:    `[{"id":5}]`,
			Status:      200,
		},
		{
			Description: "bracket that does not span the whole value",
			Query:       "/filter_values?name=eq.[draft] v2&select=id",
			Expected:    `[{"id":6}]`,
			Status:      200,
		},
		{
			Description: "colon in a value",
			Query:       "/filter_values?ver=eq.10:30&select=id",
			Expected:    `[{"id":7}]`,
			Status:      200,
		},
		{
			Description: "url in a value",
			Query:       "/filter_values?name=eq.http://x.y/z?q=1&select=id",
			Expected:    `[{"id":8}]`,
			Status:      200,
		},
		{
			Description: "unbalanced double quote in a value",
			Query:       `/filter_values?ver=eq.5" display&select=id`,
			Expected:    `[{"id":9}]`,
			Status:      200,
		},
		{
			Description: "float literal (worked before: positive control)",
			Query:       "/filter_values?ver=eq.0.5&select=id",
			Expected:    `[{"id":8}]`,
			Status:      200,
		},
		{
			Description: "empty value matches the empty string",
			Query:       "/filter_values?name=eq.&select=id",
			Expected:    `[{"id":7}]`,
			Status:      200,
		},
		{
			Description: "negated operator",
			Query:       "/filter_values?ver=not.eq.1.2.3&select=id&order=id",
			Expected:    `[{"id":2},{"id":3},{"id":4},{"id":5},{"id":6},{"id":7},{"id":8},{"id":9}]`,
			Status:      200,
		},
		{
			Description: "like pattern with a dot",
			Query:       "/filter_values?name=like.*a.b*&select=id",
			Expected:    `[{"id":2}]`,
			Status:      200,
		},
		// --- explicit quoting keeps working (a smoothdb leniency: PostgREST
		//     takes the quotes literally at the top level) ---
		{
			Description: "quoted value with a comma",
			Query:       `/filter_values?name=eq."a,b"&select=id`,
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
		{
			Description: "quoted value with dots",
			Query:       `/filter_values?ver=eq."1.2.3"&select=id`,
			Expected:    `[{"id":1}]`,
			Status:      200,
		},
		// --- lists and logic trees: a value ends at the next separator ---
		{
			Description: "in: one dot per element (worked before: positive control)",
			Query:       "/filter_values?ver=in.(a.b,c.d)&select=id&order=id",
			Expected:    `[{"id":5},{"id":6}]`,
			Status:      200,
		},
		{
			Description: "in: more than one dot in an element",
			Query:       "/filter_values?ver=in.(1.2.3,a.b)&select=id&order=id",
			Expected:    `[{"id":1},{"id":6}]`,
			Status:      200,
		},
		{
			Description: "in: apostrophe in an element",
			Query:       "/filter_values?name=in.(O'Brien,a.b@c.com)&select=id&order=id",
			Expected:    `[{"id":2},{"id":4}]`,
			Status:      200,
		},
		{
			Description: "in: quoted element with a comma next to a dotted one",
			Query:       `/filter_values?name=in.("a,b",a.b@c.com)&select=id&order=id`,
			Expected:    `[{"id":1},{"id":2}]`,
			Status:      200,
		},
		{
			Description: "any list: dotted elements",
			Query:       "/filter_values?name=eq(any).{a.b@c.com,autoexec.bat}&select=id&order=id",
			Expected:    `[{"id":2},{"id":3}]`,
			Status:      200,
		},
		{
			Description: "logic tree: dotted values",
			Query:       "/filter_values?or=(ver.eq.1.2.3,name.eq.a.b@c.com)&select=id&order=id",
			Expected:    `[{"id":1},{"id":2}]`,
			Status:      200,
		},
		{
			Description: "logic tree: a quoted value may contain a comma",
			Query:       `/filter_values?or=(name.eq."a,b",ver.eq.10:30)&select=id&order=id`,
			Expected:    `[{"id":1},{"id":7}]`,
			Status:      200,
		},
		{
			Description: "logic tree: an unquoted comma separates filters, as in PostgREST",
			Query:       "/filter_values?or=(name.eq.a,b)&select=id",
			Status:      400,
		},
		{
			Description: "operator without the delimiter",
			Query:       "/filter_values?name=eq&select=id",
			Expected:    `{"subsystem":"network","message":"'.' expected","code":"","hint":"","details":null,"position":0}`,
			Status:      400,
		},
	}
	test.Execute(t, testConfig, tests)
}
