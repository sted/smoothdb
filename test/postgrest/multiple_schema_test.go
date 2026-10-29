package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// Ported from test/spec/Feature/Query/MultipleSchemaSpec.hs, which runs with
// db-schemas = v1, v2, SPECIAL "@/\#~_- (v1 the default). The schemas and
// their tables are in the fixtures, but TestMain exposes only test and تست,
// so every case here needs a configuration the suite lacks. The two cases on
// an unknown schema, and the OpenAPI one, do not depend on which schemas are
// exposed: they are ported in schema_profile_test.go against the suite's own.
func TestPostgREST_MultipleSchema(t *testing.T) {

	const (
		msConfig  = "needs configuration Database.ExposedSchemas v1, v2, SPECIAL \"@/\\#~_- (TestMain exposes test, تست)"
		msProfile = msConfig + "; and PostgREST 16.4: the response names its schema in a Content-Profile header; no card"
		msMedia   = "PostgREST 16.4: a media type handler (a domain named after the type) answers text/plain and custom types; card 32503 (media types); and " + msConfig
		msPut     = "PostgREST 16.4: PUT upserts the row its primary key filter addresses; card 32478 (put); and " + msConfig
		msOpenAPI = "PostgREST 16.4: the root endpoint serves the OpenAPI description of the selected schema; card 32493 (openapi); and " + msConfig
	)

	tests := []test.Test{
		// spec withConfig = withConfig (baseCfg{configDbSchemas = fromList ["v1", "v2", "SPECIAL \"@/\\#~_-"]}) $
		//   describe "PostGIS features" $
		//     describe "multiple schemas in single instance" $ do
		//       context "Reading tables on different schemas" $ do
		//         it "succeeds in reading table from default schema v1 if no schema is selected via header" $
		//           request methodGet "/parents" [] ""
		//             `shouldRespondWith` [json|[
		//             {"id":1,"name":"parent v1-1"},
		//             {"id":2,"name":"parent v1-2"}
		//           ]|]
		//               { matchStatus = 200
		//               , matchHeaders = [matchContentTypeJson, "Content-Profile" <:> "v1"]
		//               }
		{
			Description:     "succeeds in reading table from default schema v1 if no schema is selected via header",
			Query:           "/parents",
			Expected:        `[{"id":1,"name":"parent v1-1"},{"id":2,"name":"parent v1-2"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Profile": "v1"},
			Status:          200,
			Skip:            msProfile,
		},
		//         it "succeeds in reading table from default schema v1 after explicitly passing it in the header" $
		//           request methodGet "/parents" [("Accept-Profile", "v1")] ""
		//             `shouldRespondWith` [json|[
		//             {"id":1,"name":"parent v1-1"},
		//             {"id":2,"name":"parent v1-2"}
		//           ]|]
		//               { matchStatus = 200
		//               , matchHeaders = [matchContentTypeJson, "Content-Profile" <:> "v1"]
		//               }
		{
			Description:     "succeeds in reading table from default schema v1 after explicitly passing it in the header",
			Query:           "/parents",
			Headers:         test.Headers{"Accept-Profile": {"v1"}},
			Expected:        `[{"id":1,"name":"parent v1-1"},{"id":2,"name":"parent v1-2"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Profile": "v1"},
			Status:          200,
			Skip:            msProfile,
		},
		//         it "succeeds in reading table from schema v2" $
		//           request methodGet "/parents" [("Accept-Profile", "v2")] ""
		//             `shouldRespondWith` [json|[
		//             {"id":3,"name":"parent v2-3"},
		//             {"id":4,"name":"parent v2-4"}
		//           ]|]
		//               { matchStatus = 200
		//               , matchHeaders = [matchContentTypeJson, "Content-Profile" <:> "v2"]
		//               }
		{
			Description:     "succeeds in reading table from schema v2",
			Query:           "/parents",
			Headers:         test.Headers{"Accept-Profile": {"v2"}},
			Expected:        `[{"id":3,"name":"parent v2-3"},{"id":4,"name":"parent v2-4"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Profile": "v2"},
			Status:          200,
			Skip:            msProfile,
		},
		//         it "succeeds in reading another_table from schema v2" $
		//           request methodGet "/another_table" [("Accept-Profile", "v2")] ""
		//             `shouldRespondWith` [json|[
		//             {"id":5,"another_value":"value 5"},
		//             {"id":6,"another_value":"value 6"}
		//           ]|]
		//               { matchStatus = 200
		//               , matchHeaders = [matchContentTypeJson, "Content-Profile" <:> "v2"]
		//               }
		{
			Description:     "succeeds in reading another_table from schema v2",
			Query:           "/another_table",
			Headers:         test.Headers{"Accept-Profile": {"v2"}},
			Expected:        `[{"id":5,"another_value":"value 5"},{"id":6,"another_value":"value 6"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Profile": "v2"},
			Status:          200,
			Skip:            msProfile,
		},
		//         it "doesn't find another_table in schema v1" $
		//           request
		//             methodGet
		//             "/another_table"
		//             [("Accept-Profile", "v1")]
		//             ""
		//             `shouldRespondWith` [json| {"code":"PGRST205","details":null,"hint":null,"message":"Could not find the table 'v1.another_table' in the schema cache"} |]
		//               { matchStatus = 404
		//               , matchHeaders = []
		//               }
		// the PGRST205 body is PostgREST's error shape (card 32494): status only
		{
			Description: "doesn't find another_table in schema v1",
			Query:       "/another_table",
			Headers:     test.Headers{"Accept-Profile": {"v1"}},
			Status:      404,
			Skip:        msConfig,
		},
		//         it "succeeds in reading a table from a schema with uppercase and special characters in its name" $
		//           request methodGet "/names?select=id,name" [("Accept-Profile", "SPECIAL \"@/\\#~_-")] ""
		//             `shouldRespondWith` [json|[
		//             {"id": 1, "name":"John"},
		//             {"id": 2, "name":"Mary"},
		//             {"id": 3, "name":"José"}
		//           ]|]
		//               { matchStatus = 200
		//               , matchHeaders = [matchContentTypeJson, "Content-Profile" <:> "SPECIAL \"@/\\#~_-"]
		//               }
		{
			Description:     "succeeds in reading a table from a schema with uppercase and special characters in its name",
			Query:           "/names?select=id,name",
			Headers:         test.Headers{"Accept-Profile": {"SPECIAL \"@/\\#~_-"}},
			Expected:        `[{"id": 1, "name":"John"},{"id": 2, "name":"Mary"},{"id": 3, "name":"José"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Profile": "SPECIAL \"@/\\#~_-"},
			Status:          200,
			Skip:            msProfile,
		},
		//         it "succeeds in embedding with FK when the schema name has special characters" $
		//           request methodGet "/names?select=name,languages(name)&id=in.(1,3)" [("Accept-Profile", "SPECIAL \"@/\\#~_-")] ""
		//             `shouldRespondWith` [json|[
		//             {"name": "John", languages: {"name": "English"}},
		//             {"name": "José", languages: {"name": "Spanish"}}
		//           ]|]
		//               { matchStatus = 200
		//               , matchHeaders = [matchContentTypeJson, "Content-Profile" <:> "SPECIAL \"@/\\#~_-"]
		//               }
		{
			Description:     "succeeds in embedding with FK when the schema name has special characters",
			Query:           "/names?select=name,languages(name)&id=in.(1,3)",
			Headers:         test.Headers{"Accept-Profile": {"SPECIAL \"@/\\#~_-"}},
			Expected:        `[{"name": "John", "languages": {"name": "English"}},{"name": "José", "languages": {"name": "Spanish"}}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Profile": "SPECIAL \"@/\\#~_-"},
			Status:          200,
			Skip:            msProfile,
		},
		//         it "succeeds in embedding with computed relationships when the schema name has special characters" $
		//           request methodGet "/names?select=name,computed_languages(name)&id=in.(1,3)" [("Accept-Profile", "SPECIAL \"@/\\#~_-")] ""
		//             `shouldRespondWith` [json|[
		//             {"name": "John", computed_languages: {"name": "English"}},
		//             {"name": "José", computed_languages: {"name": "Spanish"}}
		//           ]|]
		//               { matchStatus = 200
		//               , matchHeaders = [matchContentTypeJson, "Content-Profile" <:> "SPECIAL \"@/\\#~_-"]
		//               }
		{
			Description:     "succeeds in embedding with computed relationships when the schema name has special characters",
			Query:           "/names?select=name,computed_languages(name)&id=in.(1,3)",
			Headers:         test.Headers{"Accept-Profile": {"SPECIAL \"@/\\#~_-"}},
			Expected:        `[{"name": "John", "computed_languages": {"name": "English"}},{"name": "José", "computed_languages": {"name": "Spanish"}}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Profile": "SPECIAL \"@/\\#~_-"},
			Status:          200,
			Skip:            msProfile,
		},
		//       context "Inserting tables on different schemas" $ do
		//         it "succeeds inserting on default schema and returning it" $
		//           request
		//             methodPost
		//             "/children"
		//             [("Prefer", "return=representation")]
		//             [json|{"id": 0, "name": "child v1-1", "parent_id": 1}|]
		//             `shouldRespondWith` [json|[{"id": 0, "name": "child v1-1", "parent_id": 1}]|]
		//               { matchStatus = 201
		//               , matchHeaders = ["Content-Profile" <:> "v1"]
		//               }
		{
			Description:     "succeeds inserting on default schema and returning it",
			Method:          "POST",
			Query:           "/children",
			Headers:         test.Headers{"Prefer": {"return=representation"}},
			Body:            `{"id": 0, "name": "child v1-1", "parent_id": 1}`,
			Expected:        `[{"id": 0, "name": "child v1-1", "parent_id": 1}]`,
			ExpectedHeaders: map[string]string{"Content-Profile": "v1"},
			Status:          201,
			Skip:            msProfile,
		},
		//         it "succeeds inserting on the v1 schema and returning its parent" $
		//           request
		//             methodPost
		//             "/children?select=id,parent(*)"
		//             [("Prefer", "return=representation"), ("Content-Profile", "v1")]
		//             [json|{"id": 0, "name": "child v1-2", "parent_id": 2}|]
		//             `shouldRespondWith` [json|[{"id": 0, "parent": {"id": 2, "name": "parent v1-2"}}]|]
		//               { matchStatus = 201
		//               , matchHeaders = ["Content-Profile" <:> "v1"]
		//               }
		{
			Description:     "succeeds inserting on the v1 schema and returning its parent",
			Method:          "POST",
			Query:           "/children?select=id,parent(*)",
			Headers:         test.Headers{"Prefer": {"return=representation"}, "Content-Profile": {"v1"}},
			Body:            `{"id": 0, "name": "child v1-2", "parent_id": 2}`,
			Expected:        `[{"id": 0, "parent": {"id": 2, "name": "parent v1-2"}}]`,
			ExpectedHeaders: map[string]string{"Content-Profile": "v1"},
			Status:          201,
			Skip:            msProfile,
		},
		//         it "succeeds inserting on the v2 schema and returning its parent" $
		//           request
		//             methodPost
		//             "/children?select=id,parent(*)"
		//             [("Prefer", "return=representation"), ("Content-Profile", "v2")]
		//             [json|{"id": 0, "name": "child v2-3", "parent_id": 3}|]
		//             `shouldRespondWith` [json|[{"id": 0, "parent": {"id": 3, "name": "parent v2-3"}}]|]
		//               { matchStatus = 201
		//               , matchHeaders = ["Content-Profile" <:> "v2"]
		//               }
		{
			Description:     "succeeds inserting on the v2 schema and returning its parent",
			Method:          "POST",
			Query:           "/children?select=id,parent(*)",
			Headers:         test.Headers{"Prefer": {"return=representation"}, "Content-Profile": {"v2"}},
			Body:            `{"id": 0, "name": "child v2-3", "parent_id": 3}`,
			Expected:        `[{"id": 0, "parent": {"id": 3, "name": "parent v2-3"}}]`,
			ExpectedHeaders: map[string]string{"Content-Profile": "v2"},
			Status:          201,
			Skip:            msProfile,
		},
		//         it "succeeds in calling handler with a domain on another schema" $
		//           request methodGet "/another_table" [("Accept-Profile", "v2"), (hAccept, "text/plain")] ""
		//             `shouldRespondWith` "plain"
		//               { matchStatus = 200
		//               , matchHeaders = ["Content-Type" <:> "text/plain; charset=utf-8", "Content-Profile" <:> "v2"]
		//               }
		{
			Description:     "succeeds in calling handler with a domain on another schema",
			Query:           "/another_table",
			Headers:         test.Headers{"Accept-Profile": {"v2"}, "Accept": {"text/plain"}},
			Expected:        `plain`,
			ExpectedHeaders: map[string]string{"Content-Type": "text/plain; charset=utf-8", "Content-Profile": "v2"},
			Status:          200,
			Skip:            msMedia,
		},
		//         it "succeeds in calling handler with a domain on an exposed schema" $
		//           request methodGet "/another_table" [("Accept-Profile", "v2"), (hAccept, "text/special")] ""
		//             `shouldRespondWith` "special"
		//               { matchStatus = 200
		//               , matchHeaders = ["Content-Type" <:> "text/special", "Content-Profile" <:> "v2"]
		//               }
		{
			Description:     "succeeds in calling handler with a domain on an exposed schema",
			Query:           "/another_table",
			Headers:         test.Headers{"Accept-Profile": {"v2"}, "Accept": {"text/special"}},
			Expected:        `special`,
			ExpectedHeaders: map[string]string{"Content-Type": "text/special", "Content-Profile": "v2"},
			Status:          200,
			Skip:            msMedia,
		},
		//       context "calling procs on different schemas" $ do
		//         it "succeeds in calling the default schema proc" $
		//           request methodGet "/rpc/get_parents_below?id=6" [] ""
		//             `shouldRespondWith` [json|[{"id":1,"name":"parent v1-1"}, {"id":2,"name":"parent v1-2"}]|]
		//               { matchStatus = 200
		//               , matchHeaders = [matchContentTypeJson, "Content-Profile" <:> "v1"]
		//               }
		{
			Description:     "succeeds in calling the default schema proc",
			Query:           "/rpc/get_parents_below?id=6",
			Expected:        `[{"id":1,"name":"parent v1-1"}, {"id":2,"name":"parent v1-2"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Profile": "v1"},
			Status:          200,
			Skip:            msProfile,
		},
		//         it "succeeds in calling the v1 schema proc and embedding" $
		//           request methodGet "/rpc/get_parents_below?id=6&select=id,name,children(id,name)" [("Accept-Profile", "v1")] ""
		//             `shouldRespondWith` [json| [
		//             {"id":1,"name":"parent v1-1","children":[{"id":1,"name":"child v1-1"}]},
		//             {"id":2,"name":"parent v1-2","children":[{"id":2,"name":"child v1-2"}]}] |]
		//               { matchStatus = 200
		//               , matchHeaders = [matchContentTypeJson, "Content-Profile" <:> "v1"]
		//               }
		{
			Description:     "succeeds in calling the v1 schema proc and embedding",
			Query:           "/rpc/get_parents_below?id=6&select=id,name,children(id,name)",
			Headers:         test.Headers{"Accept-Profile": {"v1"}},
			Expected:        `[{"id":1,"name":"parent v1-1","children":[{"id":1,"name":"child v1-1"}]},{"id":2,"name":"parent v1-2","children":[{"id":2,"name":"child v1-2"}]}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Profile": "v1"},
			Status:          200,
			Skip:            msProfile,
		},
		//         it "succeeds in calling the v2 schema proc and embedding" $
		//           request methodGet "/rpc/get_parents_below?id=6&select=id,name,children(id,name)" [("Accept-Profile", "v2")] ""
		//             `shouldRespondWith` [json| [
		//             {"id":3,"name":"parent v2-3","children":[{"id":1,"name":"child v2-3"}]},
		//             {"id":4,"name":"parent v2-4","children":[]}] |]
		//               { matchStatus = 200
		//               , matchHeaders = [matchContentTypeJson, "Content-Profile" <:> "v2"]
		//               }
		{
			Description:     "succeeds in calling the v2 schema proc and embedding",
			Query:           "/rpc/get_parents_below?id=6&select=id,name,children(id,name)",
			Headers:         test.Headers{"Accept-Profile": {"v2"}},
			Expected:        `[{"id":3,"name":"parent v2-3","children":[{"id":1,"name":"child v2-3"}]},{"id":4,"name":"parent v2-4","children":[]}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Profile": "v2"},
			Status:          200,
			Skip:            msProfile,
		},
		//         it "succeeds in calling the v2 schema proc with POST by using Content-Profile" $
		//           request
		//             methodPost
		//             "/rpc/get_parents_below?select=id,name"
		//             [("Content-Profile", "v2")]
		//             [json|{"id": "6"}|]
		//             `shouldRespondWith` [json| [
		//             {"id":3,"name":"parent v2-3"},
		//             {"id":4,"name":"parent v2-4"}]|]
		//               { matchStatus = 200
		//               , matchHeaders = [matchContentTypeJson, "Content-Profile" <:> "v2"]
		//               }
		{
			Description:     "succeeds in calling the v2 schema proc with POST by using Content-Profile",
			Method:          "POST",
			Query:           "/rpc/get_parents_below?select=id,name",
			Headers:         test.Headers{"Content-Profile": {"v2"}},
			Body:            `{"id": "6"}`,
			Expected:        `[{"id":3,"name":"parent v2-3"},{"id":4,"name":"parent v2-4"}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Profile": "v2"},
			Status:          200,
			Skip:            msProfile,
		},
		//         it "succeeds in calling handler with a domain on another schema" $
		//           request methodGet "/rpc/get_plain_text" [("Accept-Profile", "v2"), (hAccept, "text/plain")] ""
		//             `shouldRespondWith` "plain"
		//               { matchStatus = 200
		//               , matchHeaders = ["Content-Type" <:> "text/plain; charset=utf-8", "Content-Profile" <:> "v2"]
		//               }
		{
			Description:     "succeeds in calling handler with a domain on another schema",
			Query:           "/rpc/get_plain_text",
			Headers:         test.Headers{"Accept-Profile": {"v2"}, "Accept": {"text/plain"}},
			Expected:        `plain`,
			ExpectedHeaders: map[string]string{"Content-Type": "text/plain; charset=utf-8", "Content-Profile": "v2"},
			Status:          200,
			Skip:            msMedia,
		},
		//         it "succeeds in calling handler with a domain on an exposed schema" $
		//           request methodGet "/rpc/get_special_text" [("Accept-Profile", "v2"), (hAccept, "text/special")] ""
		//             `shouldRespondWith` "special"
		//               { matchStatus = 200
		//               , matchHeaders = ["Content-Type" <:> "text/special", "Content-Profile" <:> "v2"]
		//               }
		{
			Description:     "succeeds in calling handler with a domain on an exposed schema",
			Query:           "/rpc/get_special_text",
			Headers:         test.Headers{"Accept-Profile": {"v2"}, "Accept": {"text/special"}},
			Expected:        `special`,
			ExpectedHeaders: map[string]string{"Content-Type": "text/special", "Content-Profile": "v2"},
			Status:          200,
			Skip:            msMedia,
		},
		//       context "Modifying tables on different schemas" $ do
		//         it "succeeds in patching on the v1 schema and returning its parent" $
		//           request
		//             methodPatch
		//             "/children?select=name,parent(name)&id=eq.1"
		//             [("Content-Profile", "v1"), ("Prefer", "return=representation")]
		//             [json|{"name": "child v1-1 updated"}|]
		//             `shouldRespondWith` [json|[{"name":"child v1-1 updated", "parent": {"name": "parent v1-1"}}]|]
		//               { matchStatus = 200
		//               , matchHeaders = [matchContentTypeJson, "Content-Profile" <:> "v1"]
		//               }
		{
			Description:     "succeeds in patching on the v1 schema and returning its parent",
			Method:          "PATCH",
			Query:           "/children?select=name,parent(name)&id=eq.1",
			Headers:         test.Headers{"Content-Profile": {"v1"}, "Prefer": {"return=representation"}},
			Body:            `{"name": "child v1-1 updated"}`,
			Expected:        `[{"name":"child v1-1 updated", "parent": {"name": "parent v1-1"}}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Profile": "v1"},
			Status:          200,
			Skip:            msProfile,
		},
		//         it "succeeds in patching on the v2 schema and returning its parent" $
		//           request
		//             methodPatch
		//             "/children?select=name,parent(name)&id=eq.1"
		//             [("Content-Profile", "v2"), ("Prefer", "return=representation")]
		//             [json|{"name": "child v2-1 updated"}|]
		//             `shouldRespondWith` [json|[{"name":"child v2-1 updated", "parent": {"name": "parent v2-3"}}]|]
		//               { matchStatus = 200
		//               , matchHeaders = [matchContentTypeJson, "Content-Profile" <:> "v2"]
		//               }
		{
			Description:     "succeeds in patching on the v2 schema and returning its parent",
			Method:          "PATCH",
			Query:           "/children?select=name,parent(name)&id=eq.1",
			Headers:         test.Headers{"Content-Profile": {"v2"}, "Prefer": {"return=representation"}},
			Body:            `{"name": "child v2-1 updated"}`,
			Expected:        `[{"name":"child v2-1 updated", "parent": {"name": "parent v2-3"}}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Profile": "v2"},
			Status:          200,
			Skip:            msProfile,
		},
		//         it "succeeds on deleting on the v2 schema" $ do
		//           request
		//             methodDelete
		//             "/children?id=eq.1"
		//             [("Content-Profile", "v2"), ("Prefer", "return=representation")]
		//             ""
		//             `shouldRespondWith` [json|[{"id": 1, "name": "child v2-3", "parent_id": 3}]|]
		//               { matchHeaders = ["Content-Profile" <:> "v2"]
		//               }
		{
			Description:     "succeeds on deleting on the v2 schema",
			Method:          "DELETE",
			Query:           "/children?id=eq.1",
			Headers:         test.Headers{"Content-Profile": {"v2"}, "Prefer": {"return=representation"}},
			Expected:        `[{"id": 1, "name": "child v2-3", "parent_id": 3}]`,
			ExpectedHeaders: map[string]string{"Content-Profile": "v2"},
			Status:          200,
			Skip:            msProfile,
		},
		//         it "succeeds on PUT on the v2 schema" $
		//           request
		//             methodPut
		//             "/children?id=eq.111"
		//             [("Content-Profile", "v2"), ("Prefer", "return=representation")]
		//             [json|[{"id": 111, "name": "child v2-111", "parent_id": null}]|]
		//             `shouldRespondWith` [json|[{"id": 111, "name": "child v2-111", "parent_id": null}]|]
		//               { matchStatus = 201
		//               , matchHeaders = [matchContentTypeJson, "Content-Profile" <:> "v2"]
		//               }
		{
			Description:     "succeeds on PUT on the v2 schema",
			Method:          "PUT",
			Query:           "/children?id=eq.111",
			Headers:         test.Headers{"Content-Profile": {"v2"}, "Prefer": {"return=representation"}},
			Body:            `[{"id": 111, "name": "child v2-111", "parent_id": null}]`,
			Expected:        `[{"id": 111, "name": "child v2-111", "parent_id": null}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Profile": "v2"},
			Status:          201,
			Skip:            msPut,
		},
		//       context "OpenAPI output" $ do
		//         it "succeeds in reading table definition from default schema v1 if no schema is selected via header" $ do
		//           req <- request methodGet "/" [] ""
		//
		//           liftIO $ do
		//             simpleHeaders req `shouldSatisfy` matchHeader "Content-Profile" "v1"
		//
		//             let def = simpleBody req ^? key "definitions" . key "parents"
		//
		//             def
		//               `shouldBe` Just
		//                 [aesonQQ|
		//                   {
		//                     "type" : "object",
		//                     "properties" : {
		//                       "id" : {
		//                         "description" : "Note:\nThis is a Primary Key.<pk/>",
		//                         "format" : "int32",
		//                         "type" : "integer"
		//                       },
		//                       "name" : {
		//                         "format" : "text",
		//                         "type" : "string"
		//                       }
		//                     },
		//                     "required" : [
		//                       "id"
		//                     ]
		//                   }
		//                 |]
		// the definition of parents is checked inside the body: not expressible with Expected
		{
			Description:     "succeeds in reading table definition from default schema v1 if no schema is selected via header",
			Query:           "/",
			ExpectedHeaders: map[string]string{"Content-Profile": "v1"},
			Status:          200,
			Skip:            msOpenAPI,
		},
		//         it "succeeds in reading table definition from default schema v1 after explicitly passing it in the header" $ do
		//           r <- request methodGet "/" [("Accept-Profile", "v1")] ""
		//
		//           liftIO $ do
		//             simpleHeaders r `shouldSatisfy` matchHeader "Content-Profile" "v1"
		//
		//             let def = simpleBody r ^? key "definitions" . key "parents"
		//
		//             def
		//               `shouldBe` Just
		//                 [aesonQQ|
		//                   {
		//                     "type" : "object",
		//                     "properties" : {
		//                       "id" : {
		//                         "description" : "Note:\nThis is a Primary Key.<pk/>",
		//                         "format" : "int32",
		//                         "type" : "integer"
		//                       },
		//                       "name" : {
		//                         "format" : "text",
		//                         "type" : "string"
		//                       }
		//                     },
		//                     "required" : [
		//                       "id"
		//                     ]
		//                   }
		//                 |]
		// the definition of parents is checked inside the body: not expressible with Expected
		{
			Description:     "succeeds in reading table definition from default schema v1 after explicitly passing it in the header",
			Query:           "/",
			Headers:         test.Headers{"Accept-Profile": {"v1"}},
			ExpectedHeaders: map[string]string{"Content-Profile": "v1"},
			Status:          200,
			Skip:            msOpenAPI,
		},
		//         it "succeeds in reading table definition from schema v2" $ do
		//           r <- request methodGet "/" [("Accept-Profile", "v2")] ""
		//
		//           liftIO $ do
		//             simpleHeaders r `shouldSatisfy` matchHeader "Content-Profile" "v2"
		//
		//             let def = simpleBody r ^? key "definitions" . key "parents"
		//
		//             def
		//               `shouldBe` Just
		//                 [aesonQQ|
		//                   {
		//                     "type" : "object",
		//                     "properties" : {
		//                       "id" : {
		//                         "description" : "Note:\nThis is a Primary Key.<pk/>",
		//                         "format" : "int32",
		//                         "type" : "integer"
		//                       },
		//                       "name" : {
		//                         "format" : "text",
		//                         "type" : "string"
		//                       }
		//                     },
		//                     "required" : [
		//                       "id"
		//                     ]
		//                   }
		//                 |]
		// the definition of parents is checked inside the body: not expressible with Expected
		{
			Description:     "succeeds in reading table definition from schema v2",
			Query:           "/",
			Headers:         test.Headers{"Accept-Profile": {"v2"}},
			ExpectedHeaders: map[string]string{"Content-Profile": "v2"},
			Status:          200,
			Skip:            msOpenAPI,
		},
		//         it "succeeds in reading another_table definition from schema v2" $ do
		//           r <- request methodGet "/" [("Accept-Profile", "v2")] ""
		//
		//           liftIO $ do
		//             simpleHeaders r `shouldSatisfy` matchHeader "Content-Profile" "v2"
		//
		//             let def = simpleBody r ^? key "definitions" . key "another_table"
		//
		//             def
		//               `shouldBe` Just
		//                 [aesonQQ|
		//                   {
		//                     "type" : "object",
		//                     "properties" : {
		//                       "id" : {
		//                         "description" : "Note:\nThis is a Primary Key.<pk/>",
		//                         "format" : "int32",
		//                         "type" : "integer"
		//                       },
		//                       "another_value" : {
		//                         "format" : "text",
		//                         "type" : "string"
		//                       }
		//                     },
		//                     "required" : [
		//                       "id"
		//                     ]
		//                   }
		//                 |]
		// the definition of another_table is checked inside the body: not expressible with Expected
		{
			Description:     "succeeds in reading another_table definition from schema v2",
			Query:           "/",
			Headers:         test.Headers{"Accept-Profile": {"v2"}},
			ExpectedHeaders: map[string]string{"Content-Profile": "v2"},
			Status:          200,
			Skip:            msOpenAPI,
		},
		//         it "doesn't find another_table definition in schema v1" $ do
		//           r <- request methodGet "/" [("Accept-Profile", "v1")] ""
		//
		//           liftIO $ do
		//             let def = simpleBody r ^? key "definitions" . key "another_table"
		//             def `shouldBe` Nothing
		// the absence of another_table among the definitions is checked inside the body: not expressible with Expected
		{
			Description: "doesn't find another_table definition in schema v1",
			Query:       "/",
			Headers:     test.Headers{"Accept-Profile": {"v1"}},
			Status:      200,
			Skip:        msOpenAPI,
		},
	}

	test.Execute(t, testConfig, tests)
}
