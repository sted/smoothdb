package test_api

import (
	"encoding/json"
	"testing"

	"github.com/sted/smoothdb/test"
)

// With no Database.ExposedSchemas every user schema is reachable, but a
// Profile header that names a system schema answers 406 before any SQL.
// Without this check, pg_catalog.query_to_xml ran any SQL, and that SQL could
// switch to any role granted to the authenticator.
func TestSystemSchemasAreNotExposed(t *testing.T) {
	client := test.InitClient()
	api := test.Config{
		BaseUrl:       "http://localhost:8082/api/postgres",
		CommonHeaders: test.Headers{"Authorization": {user1Token}},
	}
	cases := []struct {
		description string
		cmd         test.Command
		schema      string
	}{
		{"query_to_xml through GET",
			test.Command{Query: "/rpc/query_to_xml?query=select%20current_user&nulls=true&tableforest=false&targetns=",
				Headers: test.Headers{"Accept-Profile": {"pg_catalog"}}},
			"pg_catalog"},
		{"set_config through POST",
			test.Command{Method: "POST", Query: "/rpc/set_config",
				Body:    `{"setting_name": "search_path", "new_value": "pg_catalog", "is_local": false}`,
				Headers: test.Headers{"Content-Profile": {"pg_catalog"}}},
			"pg_catalog"},
		{"a catalog table",
			test.Command{Query: "/pg_class", Headers: test.Headers{"Accept-Profile": {"pg_catalog"}}},
			"pg_catalog"},
		{"information_schema",
			test.Command{Query: "/tables", Headers: test.Headers{"Accept-Profile": {"information_schema"}}},
			"information_schema"},
		{"pg_toast",
			test.Command{Query: "/x", Headers: test.Headers{"Accept-Profile": {"pg_toast"}}},
			"pg_toast"},
		{"the temporary schema",
			test.Command{Method: "POST", Query: "/x", Body: `{}`,
				Headers: test.Headers{"Content-Profile": {"pg_temp"}}},
			"pg_temp"},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			body, _, status, err := test.Exec(client, api, &c.cmd)
			if err != nil {
				t.Fatal(err)
			}
			if status != 406 {
				t.Fatalf("status %d, want 406: %s", status, body)
			}
			var e struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(body, &e); err != nil {
				t.Fatalf("%v in %s", err, body)
			}
			if e.Code != "PGRST106" || e.Message != "Invalid schema: "+c.schema {
				t.Errorf("got %s", body)
			}
		})
	}

	// positive control: a user schema is still reachable by header
	body, _, status, err := test.Exec(client, api, &test.Command{Query: "/",
		Headers: test.Headers{"Accept-Profile": {"public"}}})
	if err != nil {
		t.Fatal(err)
	}
	if status != 200 {
		t.Errorf("Accept-Profile public: status %d, want 200: %s", status, body)
	}
}
