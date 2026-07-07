package test_api

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/sted/smoothdb/jqeval"
	"github.com/sted/smoothdb/test"
)

// jqQuery composes a query string with properly escaped jq/jq_args parameters
func jqQuery(path string, filters string, program string, args string) string {
	q := path + "?"
	if filters != "" {
		q += filters + "&"
	}
	q += "jq=" + url.QueryEscape(program)
	if args != "" {
		q += "&jq_args=" + url.QueryEscape(args)
	}
	return q
}

// TestJQEndpoint tests POST /jq: batch evaluation and parse_only validation
func TestJQEndpoint(t *testing.T) {

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082",
		CommonHeaders: test.Headers{"Authorization": {user1Token}},
	}

	tests := []test.Test{
		{
			Description: "simple eval",
			Method:      "POST",
			Query:       "/jq",
			Body:        `{"evals": [{"program": ".a + 1", "input": {"a": 41}}]}`,
			Expected:    `[{"output": 42}]`,
			Status:      200,
		},
		{
			Description: "eval with args",
			Method:      "POST",
			Query:       "/jq",
			Body:        `{"evals": [{"program": "{sum: (.n + $delta), tag: $tag}", "input": {"n": 40}, "args": {"delta": 2, "tag": "ok"}}]}`,
			Expected:    `[{"output": {"sum": 42, "tag": "ok"}}]`,
			Status:      200,
		},
		{
			Description: "multiple evals",
			Method:      "POST",
			Query:       "/jq",
			Body:        `{"evals": [{"program": ".", "input": 1}, {"program": "[.[] | . * 2]", "input": [1,2]}]}`,
			Expected:    `[{"output": 1}, {"output": [2,4]}]`,
			Status:      200,
		},
		{
			Description: "parse_only ok",
			Method:      "POST",
			Query:       "/jq",
			Body:        `{"parse_only": true, "evals": [{"program": ".a.b"}, {"program": "$x + 1", "args": {"x": null}}]}`,
			Expected:    `[{}, {}]`,
			Status:      200,
		},
		{
			Description: "empty evals",
			Method:      "POST",
			Query:       "/jq",
			Body:        `{"evals": []}`,
			Expected:    `[]`,
			Status:      200,
		},
		{
			Description: "malformed envelope",
			Method:      "POST",
			Query:       "/jq",
			Body:        `{"evals": "nope"}`,
			Status:      400,
		},
		{
			Description: "unauthorized without token",
			Method:      "POST",
			Query:       "/jq",
			Body:        `{"evals": []}`,
			Headers:     test.Headers{"Authorization": {""}},
			Status:      401,
		},
	}

	test.Execute(t, testConfig, tests)

	// per-item errors: check the error/output shape without depending on
	// exact error strings
	t.Run("PerItemErrors", func(t *testing.T) {
		client := test.InitClient()
		body, _, status, err := test.Exec(client, testConfig, &test.Command{
			Method: "POST",
			Query:  "/jq",
			Body: `{"evals": [
				{"program": ".a |"},
				{"program": "empty"},
				{"program": ".[]", "input": [1,2]},
				{"program": "def f: f; f"},
				{"program": ".ok", "input": {"ok": true}}
			]}`,
		})
		if err != nil || status != 200 {
			t.Fatalf("status %d, err %v", status, err)
		}
		var results []map[string]any
		if err := json.Unmarshal(body, &results); err != nil {
			t.Fatal(err)
		}
		if len(results) != 5 {
			t.Fatalf("expected 5 results, got %d", len(results))
		}
		for i := range 4 {
			if _, hasError := results[i]["error"]; !hasError {
				t.Errorf("item %d: expected an error, got %v", i, results[i])
			}
		}
		if out, hasOutput := results[4]["output"]; !hasOutput || out != true {
			t.Errorf("item 4: expected output true, got %v", results[4])
		}
	})
}

// TestJQUpdate tests PATCH with the jq= parameter (atomic read-modify-write)
func TestJQUpdate(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		{
			Method: "DELETE",
			Query:  "/databases/dbtest/tables/jqtab",
		},
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables",
			Body: `{
				"name": "jqtab",
				"columns": [
					{"name": "id", "type": "integer"},
					{"name": "counter", "type": "integer"},
					{"name": "name", "type": "text"},
					{"name": "tags", "type": "jsonb"}
				]}`,
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
			Query:       "/jqtab",
			Body: `[
				{"id": 1, "counter": 10, "name": "a", "tags": ["x"]},
				{"id": 2, "counter": 20, "name": "b"},
				{"id": 3, "counter": 30, "name": "c"}]`,
			Status: 201,
		},
		{
			Description: "single row read-modify-write",
			Method:      "PATCH",
			Query:       jqQuery("/jqtab", "id=eq.1", `{"counter": (.counter + 1)}`, ""),
			Status:      204,
		},
		{
			Description: "verify increment",
			Method:      "GET",
			Query:       "/jqtab?id=eq.1&select=counter",
			Expected:    `[{"counter": 11}]`,
			Status:      200,
		},
		{
			Description: "bulk update with representation",
			Method:      "PATCH",
			Query:       jqQuery("/jqtab", "id=gte.2", `{"counter": (.counter * 2)}`, ""),
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected: `[
				{"id": 2, "counter": 40, "name": "b", "tags": null},
				{"id": 3, "counter": 60, "name": "c", "tags": null}]`,
			Status: 200,
		},
		{
			Description: "update with jq_args",
			Method:      "PATCH",
			Query:       jqQuery("/jqtab", "id=eq.1", `{"counter": $n}`, `{"n": 100}`),
			Status:      204,
		},
		{
			Description: "verify jq_args update",
			Method:      "GET",
			Query:       "/jqtab?id=eq.1&select=counter",
			Expected:    `[{"counter": 100}]`,
			Status:      200,
		},
		{
			Description: "jsonb column update",
			Method:      "PATCH",
			Query:       jqQuery("/jqtab", "id=eq.1", `{"tags": (.tags + ["y"])}`, ""),
			Status:      204,
		},
		{
			Description: "verify jsonb update",
			Method:      "GET",
			Query:       "/jqtab?id=eq.1&select=tags",
			Expected:    `[{"tags": ["x", "y"]}]`,
			Status:      200,
		},
		{
			Description: "empty object output is a no-op",
			Method:      "PATCH",
			Query:       jqQuery("/jqtab", "id=eq.1", `{}`, ""),
			Status:      204,
		},
		{
			Description: "body and jq are mutually exclusive",
			Method:      "PATCH",
			Query:       jqQuery("/jqtab", "id=eq.1", `{"counter": 1}`, ""),
			Body:        `{"counter": 1}`,
			Status:      400,
		},
		{
			// raw program in the body: no URL encoding, newlines and comments allowed
			Description: "program in the body with Content-Type application/vnd.smoothdb.jq",
			Method:      "PATCH",
			Query:       "/jqtab?id=eq.1",
			Headers:     test.Headers{"Content-Type": {"application/vnd.smoothdb.jq"}},
			Body: `# double the counter
{
  "counter": (.counter * 2)
}`,
			Status: 204,
		},
		{
			Description: "verify body-program update",
			Method:      "GET",
			Query:       "/jqtab?id=eq.1&select=counter",
			Expected:    `[{"counter": 200}]`,
			Status:      200,
		},
		{
			Description: "body program with jq_args",
			Method:      "PATCH",
			Query:       "/jqtab?id=eq.1&jq_args=" + url.QueryEscape(`{"n": 100}`),
			Headers:     test.Headers{"Content-Type": {"application/vnd.smoothdb.jq"}},
			Body:        `{"counter": $n}`,
			Status:      204,
		},
		{
			Description: "body program and jq= parameter cannot be used together",
			Method:      "PATCH",
			Query:       jqQuery("/jqtab", "id=eq.1", `{"counter": 1}`, ""),
			Headers:     test.Headers{"Content-Type": {"application/vnd.smoothdb.jq"}},
			Body:        `{"counter": 2}`,
			Status:      400,
		},
		{
			Description: "empty body program",
			Method:      "PATCH",
			Query:       "/jqtab?id=eq.1",
			Headers:     test.Headers{"Content-Type": {"application/vnd.smoothdb.jq"}},
			Body:        " ",
			Status:      400,
		},
		{
			Description: "non-object output",
			Method:      "PATCH",
			Query:       jqQuery("/jqtab", "id=eq.1", `.counter`, ""),
			Status:      400,
		},
		{
			Description: "unknown column",
			Method:      "PATCH",
			Query:       jqQuery("/jqtab", "id=eq.1", `{"nope": 1}`, ""),
			Status:      400,
		},
		{
			Description: "jq parse error",
			Method:      "PATCH",
			Query:       jqQuery("/jqtab", "id=eq.1", `.counter |`, ""),
			Status:      400,
		},
		{
			Description: "matching no rows is fine",
			Method:      "PATCH",
			Query:       jqQuery("/jqtab", "id=eq.999", `{"counter": 1}`, ""),
			Status:      204,
		},
		{
			Description: "state unchanged after errors",
			Method:      "GET",
			Query:       "/jqtab?id=eq.1&select=counter",
			Expected:    `[{"counter": 100}]`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)

	t.Run("RowCap", func(t *testing.T) {
		jqeval.Configure(&jqeval.Config{Enabled: true, MaxUpdateRows: 2})
		defer jqeval.Configure(&jqeval.Config{Enabled: true})

		test.Execute(t, testConfig, []test.Test{
			{
				Description: "more rows than the cap",
				Method:      "PATCH",
				Query:       jqQuery("/jqtab", "", `{"counter": 0}`, ""),
				Status:      400,
			},
			{
				Description: "within the cap",
				Method:      "PATCH",
				Query:       jqQuery("/jqtab", "id=lte.2", `{"counter": 0}`, ""),
				Status:      204,
			},
		})
	})
}

// TestJQUpdateAtomicity: an error on any row must roll back the whole update
func TestJQUpdateAtomicity(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		{
			Method: "DELETE",
			Query:  "/databases/dbtest/tables/jqatomic",
		},
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables",
			Body: `{
				"name": "jqatomic",
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

	tests := []test.Test{
		{
			Description: "insert records",
			Method:      "POST",
			Query:       "/jqatomic",
			Body:        `[{"id": 1, "counter": 1}, {"id": 2, "counter": 2}, {"id": 3, "counter": 3}]`,
			Status:      201,
		},
		{
			// rows 1 and 2 are updated before the program fails on row 3:
			// the transaction must roll back all of them
			Description: "eval error on the last row",
			Method:      "PATCH",
			Query:       jqQuery("/jqatomic", "", `if .id == 3 then error("boom") else {"counter": -1} end`, ""),
			Status:      400,
		},
		{
			Description: "no row was changed",
			Method:      "GET",
			Query:       "/jqatomic?select=counter&order=id",
			Expected:    `[{"counter": 1}, {"counter": 2}, {"counter": 3}]`,
			Status:      200,
		},
		{
			// same, with a database error (unknown column) instead of an eval error
			Description: "database error on the last row",
			Method:      "PATCH",
			Query:       jqQuery("/jqatomic", "", `if .id == 3 then {"nope": 1} else {"counter": -1} end`, ""),
			Status:      400,
		},
		{
			Description: "no row was changed",
			Method:      "GET",
			Query:       "/jqatomic?select=counter&order=id",
			Expected:    `[{"counter": 1}, {"counter": 2}, {"counter": 3}]`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}

// TestJQUpdateRLS: jq-update goes through row level security like a normal update
func TestJQUpdateRLS(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		{
			Method: "DELETE",
			Query:  "/databases/dbtest/tables/jq_rls",
		},
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables",
			Body: `{
				"name": "jq_rls",
				"columns": [
					{"name": "name", "type": "text"},
					{"name": "n", "type": "integer"},
					{"name": "creator", "type": "text", "default": "current_user"}
				]}`,
		},
		{
			Method: "PATCH",
			Query:  "/databases/dbtest/tables/jq_rls",
			Body:   `{"rowsecurity": true}`,
		},
		{
			Method: "POST",
			Query:  "/grants/dbtest/table/jq_rls",
			Body:   `{"types": ["all"], "grantee": "public"}`,
		},
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables/jq_rls/policies",
			Body:   `{"name": "perm_read", "command": "select", "roles": ["public"], "using": "creator = current_user"}`,
		},
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables/jq_rls/policies",
			Body:   `{"name": "perm_write", "command": "insert", "roles": ["public"], "check": "creator = current_user"}`,
		},
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables/jq_rls/policies",
			Body:   `{"name": "perm_update", "command": "update", "roles": ["public"], "using": "creator = current_user"}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: nil,
	}

	tests := []test.Test{
		{
			Description: "user1 creates a record",
			Method:      "POST",
			Query:       "/jq_rls",
			Body:        `{"name": "r1", "n": 1}`,
			Headers:     test.Headers{"Authorization": {user1Token}},
			Status:      201,
		},
		{
			Description: "user2 creates a record",
			Method:      "POST",
			Query:       "/jq_rls",
			Body:        `{"name": "r2", "n": 1}`,
			Headers:     test.Headers{"Authorization": {user2Token}},
			Status:      201,
		},
		{
			// no filters: the update sees only the rows visible to user1
			Description: "user1 jq-updates every visible row",
			Method:      "PATCH",
			Query:       jqQuery("/jq_rls", "", `{"n": (.n + 10)}`, ""),
			Headers:     test.Headers{"Authorization": {user1Token}},
			Status:      204,
		},
		{
			Description: "user1 row is updated",
			Method:      "GET",
			Query:       "/jq_rls?select=name,n",
			Expected:    `[{"name": "r1", "n": 11}]`,
			Headers:     test.Headers{"Authorization": {user1Token}},
			Status:      200,
		},
		{
			Description: "user2 row is untouched",
			Method:      "GET",
			Query:       "/jq_rls?select=name,n",
			Expected:    `[{"name": "r2", "n": 1}]`,
			Headers:     test.Headers{"Authorization": {user2Token}},
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}

// TestJQTransform tests the jq= response transform on reads
func TestJQTransform(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		{
			Method: "DELETE",
			Query:  "/databases/dbtest/tables/jqtr",
		},
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables",
			Body: `{
				"name": "jqtr",
				"columns": [
					{"name": "id", "type": "integer"},
					{"name": "n", "type": "integer"}
				]}`,
		},
		{
			Method: "DELETE",
			Query:  "/databases/dbtest/functions/jq_fn",
		},
		{
			Method: "POST",
			Query:  "/databases/dbtest/functions",
			Body: `{
				"name": "jq_fn",
				"arguments": [
					{"name": "a", "type": "integer"},
					{"name": "b", "type": "text"}
				],
				"returns": "table(a int, b text)",
				"definition": "select $1, $2"
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
			Query:       "/jqtr",
			Body:        `[{"id": 1, "n": 10}, {"id": 2, "n": 20}, {"id": 3, "n": 30}]`,
			Status:      201,
		},
		{
			Description: "transform to a projection",
			Method:      "GET",
			Query:       jqQuery("/jqtr", "order=id", `map(.n)`, ""),
			Expected:    `[10, 20, 30]`,
			Status:      200,
		},
		{
			Description: "transform to an aggregate object with args",
			Method:      "GET",
			Query:       jqQuery("/jqtr", "order=id", `{total: (map(.n) | add), label: $label}`, `{"label": "sum"}`),
			Expected:    `{"total": 60, "label": "sum"}`,
			Status:      200,
		},
		{
			Description: "transform composes with filters",
			Method:      "GET",
			Query:       jqQuery("/jqtr", "n=gte.20&order=id", `map(.id)`, ""),
			Expected:    `[2, 3]`,
			Status:      200,
		},
		{
			// Content-Range reflects the pre-transform result set
			Description:     "content-range is pre-transform",
			Method:          "GET",
			Query:           jqQuery("/jqtr", "order=id", `length`, ""),
			Expected:        `3`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-2/*"},
			Status:          200,
		},
		{
			Description: "csv content type is rejected",
			Method:      "GET",
			Query:       jqQuery("/jqtr", "", `map(.n)`, ""),
			Headers:     test.Headers{"Accept": {"text/csv"}},
			Status:      400,
		},
		{
			Description: "multiple outputs are rejected",
			Method:      "GET",
			Query:       jqQuery("/jqtr", "", `.[]`, ""),
			Status:      400,
		},
		{
			Description: "eval error",
			Method:      "GET",
			Query:       jqQuery("/jqtr", "", `.foo.bar`, ""),
			Status:      400,
		},
		{
			Description: "invalid jq_args",
			Method:      "GET",
			Query:       jqQuery("/jqtr", "", `.`, `[1,2]`),
			Status:      400,
		},
		{
			Description: "transform on RPC GET",
			Method:      "GET",
			Query:       jqQuery("/rpc/jq_fn", "a=6&b=six", `.[0].a * 7`, ""),
			Expected:    `42`,
			Status:      200,
		},
		{
			Description: "transform on RPC POST",
			Method:      "POST",
			Query:       jqQuery("/rpc/jq_fn", "", `{echo: .[0].b}`, ""),
			Body:        `{"a": 1, "b": "hello"}`,
			Expected:    `{"echo": "hello"}`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}

// TestJQDisabledParams: with jq disabled, the jq= parameter is rejected.
// (The /jq route itself is not registered when disabled: that is decided
// at server startup, so it is not observable by flipping the flag here.)
func TestJQDisabledParams(t *testing.T) {

	jqeval.Configure(&jqeval.Config{Enabled: false})
	defer jqeval.Configure(&jqeval.Config{Enabled: true})

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	tests := []test.Test{
		{
			Description: "jq transform rejected when disabled",
			Method:      "GET",
			Query:       jqQuery("/jqtr", "", `map(.n)`, ""),
			Status:      400,
		},
		{
			Description: "jq update rejected when disabled",
			Method:      "PATCH",
			Query:       jqQuery("/jqtr", "id=eq.1", `{"n": 0}`, ""),
			Status:      400,
		},
		{
			Description: "normal reads unaffected",
			Method:      "GET",
			Query:       "/jqtr?id=eq.1&select=n",
			Expected:    `[{"n": 10}]`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}
