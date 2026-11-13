package test_api

import (
	"encoding/json"
	"testing"

	"github.com/sted/smoothdb/test"
)

// The functions routes: argument JSON shape, a zero-argument function with
// arguments [], options on POST, a 400 on a bad option, and DELETE of one
// overload with ?args=.
func TestAdminFunctionsRoutes(t *testing.T) {
	execSQLAndReload(t, `
		DROP SCHEMA IF EXISTS admfn_s CASCADE;
		CREATE SCHEMA admfn_s;
		CREATE FUNCTION admfn_s.ov(int4) RETURNS int LANGUAGE sql AS 'SELECT 1';
		CREATE FUNCTION admfn_s.ov(int4, int4) RETURNS int LANGUAGE sql AS 'SELECT 2';
		CREATE PROCEDURE admfn_s.pr() LANGUAGE sql AS 'SELECT 1';`)

	testConfig := test.Config{
		BaseUrl: "http://localhost:8082/admin/databases/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken},
			"Accept-Profile": {"admfn_s"}, "Content-Profile": {"admfn_s"}},
	}
	tests := []test.Test{
		{
			Description: "create a function with options",
			Method:      "POST",
			Query:       "/functions",
			Body: `{"name": "fx", "returns": "int", "arguments": [{"name": "My Arg", "type": "int4", "mode": "in"}],
				"definition": "SELECT \"My Arg\" + length($$ab$$)", "volatility": "stable", "strict": true, "security": "definer"}`,
			Status: 201,
		},
		{
			Description: "replace it",
			Method:      "POST",
			Query:       "/functions",
			Body: `{"name": "fx", "returns": "int", "arguments": [{"name": "My Arg", "type": "int4"}],
				"definition": "SELECT 7", "replace": true, "volatility": "immutable"}`,
			Status: 201,
		},
		{
			Description: "a bad volatility is a 400",
			Method:      "POST",
			Query:       "/functions",
			Body:        `{"name": "fy", "definition": "SELECT 1", "volatility": "sometimes"}`,
			Status:      400,
		},
		{
			Description: "a bad mode is a 400",
			Method:      "POST",
			Query:       "/functions",
			Body:        `{"name": "fy", "arguments": [{"name": "a", "type": "int4", "mode": "sideways"}], "definition": "SELECT 1"}`,
			Status:      400,
		},
		{
			Description: "create a zero-argument function",
			Method:      "POST",
			Query:       "/functions",
			Body:        `{"name": "fz", "returns": "int", "definition": "SELECT 1"}`,
			Status:      201,
		},
		{
			Description: "drop one overload",
			Method:      "DELETE",
			Query:       "/functions/ov?args=int4,int4",
			Status:      200,
		},
		{
			Description: "drop the zero-argument function",
			Method:      "DELETE",
			Query:       "/functions/fz?args=",
			Status:      200,
		},
		{
			Description: "an unknown argument type is a 400",
			Method:      "DELETE",
			Query:       "/functions/ov?args=nosuchtype",
			Status:      400,
		},
	}
	test.Execute(t, testConfig, tests)

	// the listing covers every schema: check the entries of admfn_s
	listConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	body, _, status, err := test.Exec(test.InitClient(), listConfig, &test.Command{Method: "GET", Query: "/functions"})
	if err != nil || status != 200 {
		t.Fatalf("GET /functions: %d %v", status, err)
	}
	var functions []map[string]any
	if err := json.Unmarshal(body, &functions); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range functions {
		if f["schema"] != "admfn_s" {
			continue
		}
		delete(f, "definition")
		b, _ := json.Marshal(map[string]any{"arguments": f["arguments"], "volatility": f["volatility"],
			"strict": f["strict"], "security": f["security"]})
		got[f["name"].(string)] += string(b)
	}
	want := map[string]string{
		"fx": `{"arguments":[{"mode":"in","name":"My Arg","type":"integer","typeid":23}],"security":"invoker","strict":false,"volatility":"i"}`,
		"ov": `{"arguments":[{"mode":"in","name":"","type":"integer","typeid":23}],"security":"invoker","strict":false,"volatility":"v"}`,
	}
	if len(got) != len(want) || got["fx"] != want["fx"] || got["ov"] != want["ov"] {
		t.Errorf("functions of admfn_s:\n got %v\nwant %v", got, want)
	}
	execSQLAndReload(t, `DROP SCHEMA admfn_s CASCADE`)
}
