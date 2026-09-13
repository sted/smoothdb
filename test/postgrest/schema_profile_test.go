package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// Ported from Feature.Query.MultipleSchemaSpec: a Profile header naming a
// schema outside Database.ExposedSchemas (PostgREST db-schemas) answers 406,
// before any SQL. The suite exposes "test" and "تست" (see TestMain), with
// only "test" on the search path.
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
		// it "fails trying to insert into a table from unknown schema" $
		//   request methodPost "/children" [("Content-Profile", "unknown")]
		//     [json|{"name": "child 4", "parent_id": 4}|]
		//     `shouldRespondWith`
		//     [json|{"message":"Invalid schema: unknown","code":"PGRST106","details":null,"hint":"Only the following schemas are exposed: v1, v2, SPECIAL \"@/\\#~_-"}|]
		//     { matchStatus = 406 }
		{
			Description: "fails trying to insert into a table from unknown schema",
			Method:      "POST",
			Query:       "/projects",
			Body:        `{"id": 999, "name": "x"}`,
			Headers:     test.Headers{"Content-Profile": {"unknown"}},
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
