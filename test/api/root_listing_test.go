package test_api

import (
	"encoding/json"
	"testing"

	"github.com/sted/smoothdb/test"
)

// With no Database.ExposedSchemas the root lists the tables of the default
// schema (the first of SchemaSearchPath, else public), as the other routes
// read it. Accept-Profile selects another schema; every schema is reachable
// when none is exposed. The root listed the tables of every schema.
func TestRootListsTheDefaultSchema(t *testing.T) {
	admin := test.Config{
		BaseUrl:       "http://localhost:8082/admin/databases",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	test.Prepare(admin, []test.Command{
		{Method: "DELETE", Query: "/rootlist_db"},
		{Method: "POST", Query: "", Body: `{"name": "rootlist_db"}`},
		{Method: "POST", Query: "/rootlist_db/schemas", Body: `{"name": "rl_other"}`},
		{Method: "POST", Query: "/rootlist_db/tables",
			Body: `{"name": "rl_pub", "columns": [{"name": "id", "type": "int4"}]}`},
		{Method: "POST", Query: "/rootlist_db/tables",
			Body: `{"name": "rl_hidden", "schema": "rl_other", "columns": [{"name": "id", "type": "int4"}]}`},
	})
	defer test.Prepare(admin, []test.Command{{Method: "DELETE", Query: "/rootlist_db"}})

	client := test.InitClient()
	api := test.Config{
		BaseUrl:       "http://localhost:8082/api/rootlist_db",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	list := func(t *testing.T, headers test.Headers) []string {
		t.Helper()
		body, _, status, err := test.Exec(client, api, &test.Command{Query: "/", Headers: headers})
		if err != nil {
			t.Fatal(err)
		}
		if status != 200 {
			t.Fatalf("GET / %v: %d %s", headers, status, body)
		}
		var tables []struct {
			Name   string `json:"name"`
			Schema string `json:"schema"`
		}
		if err := json.Unmarshal(body, &tables); err != nil {
			t.Fatalf("%v in %s", err, body)
		}
		names := []string{}
		for _, table := range tables {
			names = append(names, table.Schema+"."+table.Name)
		}
		return names
	}
	if got := list(t, nil); len(got) != 1 || got[0] != "public.rl_pub" {
		t.Errorf("root with no Accept-Profile: %v, want [public.rl_pub]", got)
	}
	if got := list(t, test.Headers{"Accept-Profile": {"rl_other"}}); len(got) != 1 || got[0] != "rl_other.rl_hidden" {
		t.Errorf("root with Accept-Profile rl_other: %v, want [rl_other.rl_hidden]", got)
	}
}
