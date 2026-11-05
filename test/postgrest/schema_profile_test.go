package postgrest

import (
	"encoding/json"
	"testing"

	"github.com/sted/smoothdb/authn"
	"github.com/sted/smoothdb/test"
)

// Ported from Feature.Query.MultipleSchemaSpec: a Profile header naming a
// schema outside Database.ExposedSchemas (PostgREST db-schemas) answers 406,
// before any SQL. The suite exposes "test" and "تست" (see TestMain), with
// only "test" on the search path. These are the spec's cases that do not
// depend on which schemas are exposed; the others, which need v1 and v2, are
// in multiple_schema_test.go.
func TestPostgREST_SchemaProfile(t *testing.T) {

	tests := []test.Test{
		// it "fails trying to read table from unknown schema" $
		//   request methodGet "/parents" [("Accept-Profile", "unknown")] "" `shouldRespondWith`
		//     [json|{"message":"Invalid schema: unknown","code":"PGRST106","details":null,"hint":"Only the following schemas are exposed: v1, v2, SPECIAL \"@/\\#~_-"}|]
		//     { matchStatus = 406 }
		{
			Description: "fails trying to read table from unknown schema",
			Query:       "/projects",
			Headers:     test.Headers{"Accept-Profile": {"unknown"}},
			Expected:    `{"subsystem":"network","message":"Invalid schema: unknown","code":"","hint":"Only the following schemas are exposed: test, تست","details":null,"position":0}`,
			Status:      406,
		},
		// it "fails when inserting on an unknown schema" $
		//   request
		//     methodPost
		//     "/children"
		//     [("Content-Profile", "unknown")]
		//     [json|{"name": "child 4", "parent_id": 4}|]
		//     `shouldRespondWith` [json|{"message":"Invalid schema: unknown","code":"PGRST106","details":null,"hint":"Only the following schemas are exposed: v1, v2, SPECIAL \"@/\\#~_-"}|]
		//       { matchStatus = 406
		//       }
		{
			Description: "fails when inserting on an unknown schema",
			Method:      "POST",
			Query:       "/projects",
			Body:        `{"id": 999, "name": "x"}`,
			Headers:     test.Headers{"Content-Profile": {"unknown"}},
			Expected:    `{"subsystem":"network","message":"Invalid schema: unknown","code":"","hint":"Only the following schemas are exposed: test, تست","details":null,"position":0}`,
			Status:      406,
		},
		// it "fails trying to read definitions from unknown schema" $
		//   request methodGet "/" [("Accept-Profile", "unknown")] ""
		//     `shouldRespondWith` [json|{"message":"Invalid schema: unknown","code":"PGRST106","details":null,"hint":"Only the following schemas are exposed: v1, v2, SPECIAL \"@/\\#~_-"}|]
		//       { matchStatus = 406
		//       }
		{
			Description: "fails trying to read definitions from unknown schema",
			Query:       "/",
			Headers:     test.Headers{"Accept-Profile": {"unknown"}},
			Expected:    `{"subsystem":"network","message":"Invalid schema: unknown","code":"","hint":"Only the following schemas are exposed: test, تست","details":null,"position":0}`,
			Status:      406,
		},
		// @@ added: an RPC call checks the schema too
		{
			Description: "fails trying to call a function from unknown schema",
			Query:       "/rpc/getallprojects",
			Headers:     test.Headers{"Accept-Profile": {"unknown"}},
			Expected:    `{"subsystem":"network","message":"Invalid schema: unknown","code":"","hint":"Only the following schemas are exposed: test, تست","details":null,"position":0}`,
			Status:      406,
		},
		// @@ added: an exposed schema other than the first is accepted even
		// when it is not on the search path (the unicode suite reads تست
		// through the same header)
		{
			Description: "reads from another exposed schema",
			Query:       "/%D9%85%D9%88%D8%A7%D8%B1%D8%AF",
			Headers:     test.Headers{"Accept-Profile": {"تست"}},
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}

// @@ added: the root lists the tables of one schema, as PostgREST's root
// describes one schema (ApiRequest.hs, getSchema and ActSchemaRead): the
// schema of the Accept-Profile header, or the first exposed schema without
// it. A schema that is not exposed answers 406 like the other routes. The
// body stays smoothdb's table list (PostgREST serves OpenAPI there; card
// 32493 openapi). The /admin listing still shows every schema.
func TestPostgREST_RootListsOneSchema(t *testing.T) {
	client := test.InitClient()
	schemasOf := func(t *testing.T, config test.Config, query string, headers test.Headers) map[string]int {
		t.Helper()
		body, _, status, err := test.Exec(client, config, &test.Command{Query: query, Headers: headers})
		if err != nil {
			t.Fatal(err)
		}
		if status != 200 {
			t.Fatalf("GET %s %v: %d %s", query, headers, status, body)
		}
		var tables []struct {
			Name   string `json:"name"`
			Schema string `json:"schema"`
		}
		if err := json.Unmarshal(body, &tables); err != nil {
			t.Fatalf("GET %s: %v in %s", query, err, body)
		}
		schemas := map[string]int{}
		for _, table := range tables {
			schemas[table.Schema]++
		}
		return schemas
	}
	only := func(t *testing.T, schemas map[string]int, want string) {
		t.Helper()
		if schemas[want] == 0 {
			t.Errorf("no table of schema %q in the root listing: %v", want, schemas)
		}
		for s, n := range schemas {
			if s != want {
				t.Errorf("the root listing for %q shows %d tables of schema %q", want, n, s)
			}
		}
	}

	t.Run("Accept-Profile on an exposed schema", func(t *testing.T) {
		only(t, schemasOf(t, testConfig, "/", test.Headers{"Accept-Profile": {"test"}}), "test")
	})
	t.Run("Accept-Profile on the second exposed schema", func(t *testing.T) {
		only(t, schemasOf(t, testConfig, "/", test.Headers{"Accept-Profile": {"تست"}}), "تست")
	})
	t.Run("no Accept-Profile selects the first exposed schema", func(t *testing.T) {
		only(t, schemasOf(t, testConfig, "/", test.Headers{"Accept-Profile": {""}}), "test")
	})
	test.Execute(t, testConfig, []test.Test{
		{
			Description: "the root refuses a schema that exists but is not exposed",
			Query:       "/",
			Headers:     test.Headers{"Accept-Profile": {"private"}},
			Expected:    `{"subsystem":"network","message":"Invalid schema: private","code":"","hint":"Only the following schemas are exposed: test, تست","details":null,"position":0}`,
			Status:      406,
		},
	})
	t.Run("the admin listing still shows every schema", func(t *testing.T) {
		token, err := authn.GenerateToken("postgres", "reallyreallyreallyreallyverysafe")
		if err != nil {
			t.Fatal(err)
		}
		admin := test.Config{
			BaseUrl:       "http://localhost:8084/admin",
			CommonHeaders: test.Headers{"Authorization": {token}},
		}
		schemas := schemasOf(t, admin, "/databases/pgrest/tables", nil)
		for _, s := range []string{"test", "private", "v1", "تست"} {
			if schemas[s] == 0 {
				t.Errorf("no table of schema %q in the admin listing: %v", s, schemas)
			}
		}
	})
}

// Card 38674 info-leak. GET /$info/:table described a table of any schema
// the Accept-Profile header named: columns, types, constraints and owner of a
// table the API does not expose. It refuses a schema that is not exposed
// with the 406 of the other routes. PostgREST has no such route; its schema
// description, the root OpenAPI, covers the exposed schemas only. The /admin
// description of a table still reaches every schema.
func TestPostgREST_InfoDescribesOnlyExposedTables(t *testing.T) {
	test.Execute(t, testConfig, []test.Test{
		{
			Description: "$info refuses a table of a schema that is not exposed",
			Query:       "/$info/internal_job",
			Headers:     test.Headers{"Accept-Profile": {"private"}},
			Expected:    `{"subsystem":"network","message":"Invalid schema: private","code":"","hint":"Only the following schemas are exposed: test, تست","details":null,"position":0}`,
			Status:      406,
		},
	})
	client := test.InitClient()
	describe := func(t *testing.T, config test.Config, query string, headers test.Headers) (name, schema string) {
		t.Helper()
		body, _, status, err := test.Exec(client, config, &test.Command{Query: query, Headers: headers})
		if err != nil {
			t.Fatal(err)
		}
		if status != 200 {
			t.Fatalf("GET %s %v: %d %s", query, headers, status, body)
		}
		var table struct {
			Name   string `json:"name"`
			Schema string `json:"schema"`
		}
		if err := json.Unmarshal(body, &table); err != nil {
			t.Fatalf("GET %s: %v in %s", query, err, body)
		}
		return table.Name, table.Schema
	}
	t.Run("$info on an exposed table", func(t *testing.T) {
		if name, schema := describe(t, testConfig, "/$info/clients", nil); name != "clients" || schema != "test" {
			t.Errorf("got %s.%s, want test.clients", schema, name)
		}
	})
	t.Run("the admin description still reaches a hidden schema", func(t *testing.T) {
		token, err := authn.GenerateToken("postgres", "reallyreallyreallyreallyverysafe")
		if err != nil {
			t.Fatal(err)
		}
		admin := test.Config{
			BaseUrl:       "http://localhost:8084/admin",
			CommonHeaders: test.Headers{"Authorization": {token}},
		}
		if name, schema := describe(t, admin, "/databases/pgrest/tables/internal_job", test.Headers{"Accept-Profile": {"private"}}); name != "internal_job" || schema != "private" {
			t.Errorf("got %s.%s, want private.internal_job", schema, name)
		}
	})
}
