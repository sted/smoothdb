package postgrest

import (
	"testing"

	"github.com/smoothdb/smoothdb/test"
)

func TestPostgREST_Update(t *testing.T) {

	tests := []test.Test{
		// 		describe "Patching record" $ do
		//     context "to unknown uri" $
		//       it "indicates no table found by returning 404" $
		//         request methodPatch "/fake" []
		//           [json| { "real": false } |]
		//             `shouldRespondWith` 404
		{
			Description: "indicates no table found by returning 404",
			Method:      "PATCH",
			Query:       "/fake",
			Body:        `{ "real": false }`,
			Headers:     nil,
			Status:      404,
		},
		//     context "on an empty table" $
		//       it "succeeds with status code 204" $
		//         request methodPatch "/empty_table" []
		//             [json| { "extra":20 } |]
		//           `shouldRespondWith`
		//             ""
		//             { matchStatus  = 204,
		//               matchHeaders = [matchHeaderAbsent hContentType]
		//             }
		{
			Description: "on an empty table succeeds with status code 204",
			Method:      "PATCH",
			Query:       "/empty_table",
			Body:        `{ "extra":"20" }`, //@@ changed 20 -> "20"
			Headers:     nil,
			Expected:    ``,
			Status:      204,
		},
		//     context "with invalid json payload" $
		//       it "fails with 400 and error" $
		//         request methodPatch "/simple_pk" [] "}{ x = 2"
		//           `shouldRespondWith`
		//           [json|{"message":"Error in $: Failed reading: not a valid json value at '}{x=2'","code":"PGRST102","details":null,"hint":null}|]
		//           { matchStatus  = 400,
		//             matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description: "with invalid json payload fails with 400 and error",
			Method:      "PATCH",
			Query:       "/simple_pk",
			Body:        `}{ x = 2`,
			Headers:     nil,
			//Expected:    ``,
			Status: 400,
		},
		//     context "with no payload" $
		//       it "fails with 400 and error" $
		//         request methodPatch "/items" [] ""
		//           `shouldRespondWith`
		//           [json|{"message":"Error in $: not enough input","code":"PGRST102","details":null,"hint":null}|]
		//           { matchStatus  = 400,
		//             matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description: "with no payload fails with 400 and error",
			Method:      "PATCH",
			Query:       "/items",
			Body:        ``,
			Headers:     nil,
			//Expected:    ``,
			Status: 400,
		},
		//     context "in a nonempty table" $ do
		//       it "can update a single item" $ do
		//         patch "/items?id=eq.2"
		//             [json| { "id":42 } |]
		//           `shouldRespondWith`
		//             ""
		//             { matchStatus  = 204
		//             , matchHeaders = [ matchHeaderAbsent hContentType
		//                              , "Content-Range" <:> "0-0/*" ]
		//             }
		{
			Description: "in a nonempty table can update a single item",
			Method:      "PATCH",
			Query:       "/items?id=eq.2",
			Body:        `{ "id":42 }`,
			Headers:     nil,
			Expected:    ``,
			Status:      204,
		},
		//       it "returns empty array when no rows updated and return=rep" $
		//         request methodPatch "/items?id=eq.999999"
		//           [("Prefer", "return=representation")] [json| { "id":999999 } |]
		//           `shouldRespondWith` "[]"
		//           {
		//             matchStatus  = 200,
		//             matchHeaders = []
		//           }
		{
			Description: "returns empty array when no rows updated and return=rep",
			Method:      "PATCH",
			Query:       "/items?id=eq.999999",
			Body:        `{ "id":999999 }`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[]`,
			Status:      200,
		},
		//       it "returns status code 200 when no rows updated" $
		//         request methodPatch "/items?id=eq.99999999" []
		//           [json| { "id": 42 } |]
		//             `shouldRespondWith` 204
		{
			Description: "returns status code 204 when no rows updated",
			Method:      "PATCH",
			Query:       "/items?id=eq.999999",
			Body:        `{ "id":42 }`,
			Headers:     nil,
			Expected:    ``,
			Status:      204,
		},
		//       it "returns updated object as array when return=rep" $
		//         request methodPatch "/items?id=eq.2"
		//           [("Prefer", "return=representation")] [json| { "id":2 } |]
		//           `shouldRespondWith` [json|[{"id":2}]|]
		//           { matchStatus  = 200,
		//             matchHeaders = ["Content-Range" <:> "0-0/*"]
		//           }
		{
			Description: "returns updated object as array when return=rep",
			Method:      "PATCH",
			Query:       "/items?id=eq.2",
			Body:        `{ "id":2 }`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":2}]`,
			Status:      200,
		},
		//       it "can update multiple items" $ do
		//         get "/no_pk?select=a&b=eq.1"
		//           `shouldRespondWith`
		//             [json|[]|]
		{
			Description: "can update multiple items: check initial status",
			Method:      "GET",
			Query:       "/no_pk?select=a&b=eq.1",
			Expected:    `[]`,
			Status:      200,
		},
		//         request methodPatch "/no_pk?b=eq.0"
		//             [("Prefer", "tx=commit")]
		//             [json| { b: "1" } |]
		//           `shouldRespondWith`
		//             ""
		//             { matchStatus  = 204
		//             , matchHeaders = [ matchHeaderAbsent hContentType
		//                              , "Content-Range" <:> "0-1/*"
		//                              , "Preference-Applied" <:> "tx=commit" ]
		//             }
		{
			Description: "can update multiple items",
			Method:      "PATCH",
			Query:       "/no_pk?b=eq.0",
			Body:        `{ "b": "1" }`, // b -> "b"
			Headers:     test.Headers{"Prefer": {"tx=commit"}},
			Expected:    ``,
			Status:      204,
		},
		//         -- check it really got updated
		//         get "/no_pk?select=a&b=eq.1"
		//           `shouldRespondWith`
		//             [json|[ { a: "1" }, { a: "2" } ]|]
		{
			Description: "check it really got updated",
			Method:      "GET",
			Query:       "/no_pk?select=a&b=eq.1&order=a", // adder oreder=a
			Expected:    `[ { "a": "1" }, { "a": "2" } ]`, // quoted
			Status:      200,
		},
		//         -- put value back for other tests
		//         request methodPatch "/no_pk?b=eq.1"
		//             [("Prefer", "tx=commit")]
		//             [json| { b: "0" } |]
		//           `shouldRespondWith`
		//             ""
		//             { matchStatus = 204
		//             , matchHeaders = [matchHeaderAbsent hContentType]
		//             }
		{
			Description: "put value back for other tests",
			Method:      "PATCH",
			Query:       "/no_pk?b=eq.1",
			Body:        `{ "b": "0" }`, // quoted
			Headers:     test.Headers{"Prefer": {"tx=commit"}},
			Expected:    ``,
			Status:      204,
		},
		//       it "can set a column to NULL" $ do
		//         request methodPatch "/no_pk?a=eq.1"
		//             [("Prefer", "return=representation")]
		//             [json| { b: null } |]
		//           `shouldRespondWith`
		//             [json| [{ a: "1", b: null }] |]
		{
			Description: "can set a column to NULL",
			Method:      "PATCH",
			Query:       "/no_pk?a=eq.1",
			Body:        `{ "b": null }`, // quoted
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "a": "1", "b": null }]`,
			Status:      200,
		},
		//       context "filtering by a computed column" $ do
		//         it "is successful" $
		//           request methodPatch
		//             "/items?is_first=eq.true"
		//             [("Prefer", "return=representation")]
		//             [json| { id: 100 } |]
		//             `shouldRespondWith` [json| [{ id: 100 }] |]
		//             { matchStatus  = 200,
		//               matchHeaders = [matchContentTypeJson, "Content-Range" <:> "0-0/*"]
		//             }
		{
			Description: "filtering by a computed column is successful",
			Method:      "PATCH",
			Query:       "/items?is_first=eq.true",
			Body:        `{ "id": 100 }`, // quoted
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "id": 100 }]`,
			Status:      200,
		},
		//         it "returns empty array when no rows updated and return=rep" $
		//           request methodPatch
		//             "/items?always_true=eq.false"
		//             [("Prefer", "return=representation")]
		//             [json| { id: 100 } |]
		//             `shouldRespondWith` "[]"
		//             { matchStatus  = 200,
		//               matchHeaders = []
		//             }
		{
			Description: "returns empty array when no rows updated and return=rep",
			Method:      "PATCH",
			Query:       "/items?always_true=eq.false",
			Body:        `{ "id": 100 }`, // quoted
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[]`,
			Status:      200,
		},
		//       context "with representation requested" $ do
		//         it "can provide a representation" $ do
		//           _ <- post "/items"
		//             [json| { id: 1 } |]
		//           request methodPatch
		//             "/items?id=eq.1"
		//             [("Prefer", "return=representation")]
		//             [json| { id: 99 } |]
		//             `shouldRespondWith` [json| [{id:99}] |]
		//             { matchHeaders = [matchContentTypeJson] }
		{
			Description: "with representation requested can provide a representation",
			Method:      "PATCH",
			Query:       "/items?id=eq.1",
			Body:        `{ "id": 99 }`, // quoted
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id":99}]`,
			Status:      200,
		},
		//           -- put value back for other tests
		//           void $ request methodPatch "/items?id=eq.99" [] [json| { "id":1 } |]

		//         it "can return computed columns" $
		//           request methodPatch
		//             "/items?id=eq.1&select=id,always_true"
		//             [("Prefer", "return=representation")]
		//             [json| { id: 1 } |]
		//             `shouldRespondWith` [json| [{ id: 1, always_true: true }] |]
		//             { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can return computed columns",
			Method:      "PATCH",
			Query:       "/items?id=eq.1&select=id,always_true",
			Body:        `{ "id": 1 }`, // quoted
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "id": 1, "always_true": true }]`,
			Status:      200,
		},
		//         it "can select overloaded computed columns" $ do
		//           request methodPatch
		//             "/items?id=eq.1&select=id,computed_overload"
		//             [("Prefer", "return=representation")]
		//             [json| { id: 1 } |]
		//             `shouldRespondWith` [json| [{ id: 1, computed_overload: true }] |]
		//             { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can select overloaded computed columns",
			Method:      "PATCH",
			Query:       "/items?id=eq.1&select=id,computed_overload",
			Body:        `{ "id": 1 }`, // quoted
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "id": 1, "computed_overload": true }]`,
			Status:      200,
		},
		//           request methodPatch
		//             "/items2?id=eq.1&select=id,computed_overload"
		//             [("Prefer", "return=representation")]
		//             [json| { id: 1 } |]
		//             `shouldRespondWith` [json| [{ id: 1, computed_overload: true }] |]
		//             { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can select overloaded computed columns",
			Method:      "PATCH",
			Query:       "/items2?id=eq.1&select=id,computed_overload",
			Body:        `{ "id": 1 }`, // quoted
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "id": 1, "computed_overload": true }]`,
			Status:      200,
		},
		//       it "ignores ?select= when return not set or return=minimal" $ do
		//         request methodPatch "/items?id=eq.1&select=id"
		//             [] [json| { id:1 } |]
		//           `shouldRespondWith`
		//             ""
		//             { matchStatus  = 204
		//             , matchHeaders = [ matchHeaderAbsent hContentType
		//                              , "Content-Range" <:> "0-0/*" ]
		//             }
		{
			Description: "ignores ?select= when return not set or return=minimal",
			Method:      "PATCH",
			Query:       "/items?id=eq.1&select=id",
			Body:        `{ "id": 1 }`, // quoted
			Headers:     nil,
			Expected:    ``,
			Status:      204,
		},
		//         request methodPatch "/items?id=eq.1&select=id"
		//             [("Prefer", "return=minimal")]
		//             [json| { id:1 } |]
		//           `shouldRespondWith`
		//             ""
		//             { matchStatus  = 204
		//             , matchHeaders = [ matchHeaderAbsent hContentType
		//                              , "Content-Range" <:> "0-0/*" ]
		//             }
		{
			Description: "ignores ?select= when return not set or return=minimal",
			Method:      "PATCH",
			Query:       "/items?id=eq.1&select=id",
			Body:        `{ "id": 1 }`, // quoted
			Headers:     test.Headers{"Prefer": {"return=minimal"}},
			Expected:    ``,
			Status:      204,
		},
		//       context "when patching with an empty body" $ do
		//         it "makes no updates and returns 204 without return= and without ?select=" $ do
		//           request methodPatch "/items"
		//               []
		//               [json| {} |]
		//             `shouldRespondWith`
		//               ""
		//               { matchStatus  = 204
		//               , matchHeaders = [ matchHeaderAbsent hContentType
		//                                , "Content-Range" <:> "*/*" ]
		//               }
		{
			Description: "when patching with an empty body makes no updates and returns 204 without return= and without ?select=",
			Method:      "PATCH",
			Query:       "/items",
			Body:        `{}`,
			Headers:     nil,
			Expected:    ``,
			Status:      204,
		},
		//           request methodPatch "/items"
		//               []
		//               [json| [] |]
		//             `shouldRespondWith`
		//               ""
		//               { matchStatus  = 204
		//               , matchHeaders = [ matchHeaderAbsent hContentType
		//                                , "Content-Range" <:> "*/*" ]
		//               }
		{
			Description: "when patching with an empty body makes no updates and returns 204 without return= and without ?select=",
			Method:      "PATCH",
			Query:       "/items",
			Body:        `[]`,
			Headers:     nil,
			Expected:    ``,
			Status:      204,
		},
		//           request methodPatch "/items"
		//               []
		//               [json| [{}] |]
		//             `shouldRespondWith`
		//               ""
		//               { matchStatus  = 204
		//               , matchHeaders = [ matchHeaderAbsent hContentType
		//                                , "Content-Range" <:> "*/*" ]
		//               }
		{
			Description: "when patching with an empty body makes no updates and returns 204 without return= and without ?select=",
			Method:      "PATCH",
			Query:       "/items",
			Body:        `[{}]`,
			Headers:     nil,
			Expected:    ``,
			Status:      204,
		},
		//         it "makes no updates and returns 204 without return= and with ?select=" $ do
		//           request methodPatch "/items?select=id"
		//               []
		//               [json| {} |]
		//             `shouldRespondWith`
		//               ""
		//               { matchStatus  = 204
		//               , matchHeaders = [ matchHeaderAbsent hContentType
		//                                , "Content-Range" <:> "*/*" ]
		//               }
		{
			Description: "when patching with an empty body makes no updates and returns 204 without return= and without ?select=",
			Method:      "PATCH",
			Query:       "/items?select=id",
			Body:        `{}`,
			Headers:     nil,
			Expected:    ``,
			Status:      204,
		},
		//           request methodPatch "/items?select=id"
		//               []
		//               [json| [] |]
		//             `shouldRespondWith`
		//               ""
		//               { matchStatus  = 204
		//               , matchHeaders = [ matchHeaderAbsent hContentType
		//                                , "Content-Range" <:> "*/*" ]
		//               }
		{
			Description: "when patching with an empty body makes no updates and returns 204 without return= and without ?select=",
			Method:      "PATCH",
			Query:       "/items?select=id",
			Body:        `[]`,
			Headers:     nil,
			Expected:    ``,
			Status:      204,
		},
		//           request methodPatch "/items?select=id"
		//               []
		//               [json| [{}] |]
		//             `shouldRespondWith`
		//               ""
		//               { matchStatus  = 204
		//               , matchHeaders = [ matchHeaderAbsent hContentType
		//                                , "Content-Range" <:> "*/*" ]
		//               }
		{
			Description: "when patching with an empty body makes no updates and returns 204 without return= and without ?select=",
			Method:      "PATCH",
			Query:       "/items?select=id",
			Body:        `[{}]`,
			Headers:     nil,
			Expected:    ``,
			Status:      204,
		},
		//         it "makes no updates and returns 200 with return=rep and without ?select=" $
		//           request methodPatch "/items" [("Prefer", "return=representation")] [json| {} |]
		//             `shouldRespondWith` "[]"
		//             {
		//               matchStatus  = 200,
		//               matchHeaders = ["Content-Range" <:> "*/*"]
		//             }
		{
			Description: "when patching with an empty body makes no updates and returns 200 with return=rep and without ?select=",
			Method:      "PATCH",
			Query:       "/items",
			Body:        `{}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[]`,
			Status:      200,
		},

		//         it "makes no updates and returns 200 with return=rep and with ?select=" $
		//           request methodPatch "/items?select=id" [("Prefer", "return=representation")] [json| {} |]
		//             `shouldRespondWith` "[]"
		//             {
		//               matchStatus  = 200,
		//               matchHeaders = ["Content-Range" <:> "*/*"]
		//             }
		{
			Description: "when patching with an empty body makes no updates and returns 200 with return=rep and without ?select=",
			Method:      "PATCH",
			Query:       "/items?select=id",
			Body:        `{}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[]`,
			Status:      200,
		},

		//         it "makes no updates and returns 200 with return=rep and with ?select= for overloaded computed columns" $
		//           request methodPatch "/items?select=id,computed_overload" [("Prefer", "return=representation")] [json| {} |]
		//             `shouldRespondWith` "[]"
		//             {
		//               matchStatus  = 200,
		//               matchHeaders = ["Content-Range" <:> "*/*"]
		//             }
		{
			Description: "when patching with an empty body makes no updates and returns 200 with return=rep and without ?select=",
			Method:      "PATCH",
			Query:       "/items?select=id,computed_overload",
			Body:        `{}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[]`,
			Status:      200,
		},

		//     context "with unicode values" $
		//       it "succeeds and returns values intact" $ do
		//         request methodPatch "/no_pk?a=eq.1"
		//             [("Prefer", "return=representation")]
		//             [json| { "a":"圍棋", "b":"￥" } |]
		//           `shouldRespondWith`
		//             [json|[ { "a":"圍棋", "b":"￥" } ]|]
		{
			Description: "with unicode values succeeds and returns values intact",
			Method:      "PATCH",
			Query:       "/no_pk?a=eq.1",
			Body:        `{ "a":"圍棋", "b":"￥" }`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "a":"圍棋", "b":"￥" }]`,
			Status:      200,
		},
		//     context "PATCH with ?columns parameter" $ do
		//       it "ignores json keys not included in ?columns" $ do
		//         request methodPatch "/articles?id=eq.1&columns=body"
		//             [("Prefer", "return=representation")]
		//             [json| {"body": "Some real content", "smth": "here", "other": "stuff", "fake_id": 13} |]
		//           `shouldRespondWith`
		//             [json|[{"id": 1, "body": "Some real content", "owner": "postgrest_test_anonymous"}]|]
		{
			Description: "PATCH with ?columns parameter ignores json keys not included in ?columns",
			Method:      "PATCH",
			Query:       "/articles?id=eq.1&columns=body",
			Body:        `{"body": "Some real content", "smth": "here", "other": "stuff", "fake_id": 13}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{"id": 1, "body": "Some real content", "owner": "postgrest_test_anonymous"}]`,
			Status:      200,
		},
		//       it "ignores json keys and gives 200 if no record updated" $
		//         request methodPatch "/articles?id=eq.2001&columns=body" [("Prefer", "return=representation")]
		//           [json| {"body": "Some real content", "smth": "here", "other": "stuff", "fake_id": 13} |] `shouldRespondWith` 200
		{
			Description: "PATCH with ?columns parameter ignores json keys and gives 200 if no record updated",
			Method:      "PATCH",
			Query:       "/articles?id=eq.2001&columns=body",
			Body:        `{"body": "Some real content", "smth": "here", "other": "stuff", "fake_id": 13}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Status:      200,
		},
		//       it "disallows ?columns which don't exist" $ do
		//         request methodPatch "/articles?id=eq.1&columns=helicopter"
		//           [("Prefer", "return=representation")]
		//           [json|{"body": "yyy"}|]
		//           `shouldRespondWith`
		//           [json|{"code":"PGRST118","details":null,"hint":null,"message":"Column 'helicopter' of relation 'articles' does not exist"} |]
		//           { matchStatus  = 400
		//           , matchHeaders = []
		//           }

		//       it "returns missing table error even if also has invalid ?columns" $ do
		//         request methodPatch "/garlic?columns=helicopter"
		//           [("Prefer", "return=representation")]
		//           [json|[
		//             {"id": 204, "body": "yyy"},
		//             {"id": 205, "body": "zzz"}]|]
		//           `shouldRespondWith`
		//           [json|{} |]
		//           { matchStatus  = 404
		//           , matchHeaders = []
		//           }
		// {
		// 	Description: "PATCH with ?columns parameter returns missing table error even if also has invalid ?columns",
		// 	Method:      "PATCH",
		// 	Query:       "/garlic?columns=helicopter",
		// 	//Body:        `[{"id": 204, "body": "yyy"}, {"id": 205, "body": "zzz"}]`, // @@ accepts multiple records?
		// 	Body:     `[{"id": 204, "body": "yyy"}]`,
		// 	Headers:  test.Headers{"Prefer": {"return=representation"}},
		// 	Expected: ``,
		// 	Status:   404, //@@ returns 500
		// },
		//   context "tables with self reference foreign keys" $ do
		//     it "embeds children after update" $
		//       request methodPatch "/web_content?id=eq.0&select=id,name,web_content(name)"
		//               [("Prefer", "return=representation")]
		//         [json|{"name": "tardis-patched"}|]
		//         `shouldRespondWith`
		//         [json|
		//           [ { "id": 0, "name": "tardis-patched", "web_content": [ { "name": "fezz" }, { "name": "foo" }, { "name": "bar" } ]} ]
		//         |]
		//         { matchStatus  = 200,
		//           matchHeaders = [matchContentTypeJson]
		//         }
		{
			Description: "tables with self reference foreign keys embeds children after update",
			Method:      "PATCH",
			Query:       "/web_content?id=eq.0&select=id,name,web_content(name)",
			Body:        `{"name": "tardis-patched"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[ { "id": 0, "name": "tardis-patched", "web_content": [ { "name": "fezz" }, { "name": "foo" }, { "name": "bar" } ]} ]`,
			Status:      200,
		},
		//     it "embeds parent, children and grandchildren after update" $
		//       request methodPatch "/web_content?id=eq.0&select=id,name,web_content(name,web_content(name)),parent_content:p_web_id(name)"
		//               [("Prefer", "return=representation")]
		//         [json|{"name": "tardis-patched-2"}|]
		//         `shouldRespondWith`
		//         [json| [
		//           {
		//             "id": 0,
		//             "name": "tardis-patched-2",
		//             "parent_content": { "name": "wat" },
		//             "web_content": [
		//                 { "name": "fezz", "web_content": [ { "name": "wut" } ] },
		//                 { "name": "foo",  "web_content": [] },
		//                 { "name": "bar",  "web_content": [] }
		//             ]
		//           }
		//         ] |]
		//         { matchStatus  = 200,
		//           matchHeaders = [matchContentTypeJson]
		//         }
		{
			Description: "embeds parent, children and grandchildren after update",
			Method:      "PATCH",
			Query:       "/web_content?id=eq.0&select=id,name,web_content(name,web_content(name)),parent_content:p_web_id(name)",
			Body:        `{"name": "tardis-patched-2"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected: `[
				          {
				            "id": 0,
				            "name": "tardis-patched-2",
				            "parent_content": { "name": "wat" },
				            "web_content": [
				                { "name": "fezz", "web_content": [ { "name": "wut" } ] },
				                { "name": "foo",  "web_content": [] },
				                { "name": "bar",  "web_content": [] }
				            ]
				          }
				        ] `,
			Status: 200,
		},
		//     it "embeds children after update without explicitly including the id in the ?select" $
		//       request methodPatch "/web_content?id=eq.0&select=name,web_content(name)"
		//               [("Prefer", "return=representation")]
		//         [json|{"name": "tardis-patched"}|]
		//         `shouldRespondWith`
		//         [json|
		//           [ { "name": "tardis-patched", "web_content": [ { "name": "fezz" }, { "name": "foo" }, { "name": "bar" } ]} ]
		//         |]
		//         { matchStatus  = 200,
		//           matchHeaders = [matchContentTypeJson]
		//         }
		{
			Description: "embeds children after update without explicitly including the id in the ?select",
			Method:      "PATCH",
			Query:       "/web_content?id=eq.0&select=name,web_content(name)",
			Body:        `{"name": "tardis-patched"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[ { "name": "tardis-patched", "web_content": [ { "name": "fezz" }, { "name": "foo" }, { "name": "bar" } ]} ]`,
			Status:      200,
		},
		//     it "embeds an M2M relationship plus parent after update" $
		//       request methodPatch "/users?id=eq.1&select=name,tasks(name,project:projects(name))"
		//               [("Prefer", "return=representation")]
		//         [json|{"name": "Kevin Malone"}|]
		//         `shouldRespondWith`
		//         [json|[
		//           {
		//             "name": "Kevin Malone",
		//             "tasks": [
		//                 { "name": "Design w7", "project": { "name": "Windows 7" } },
		//                 { "name": "Code w7", "project": { "name": "Windows 7" } },
		//                 { "name": "Design w10", "project": { "name": "Windows 10" } },
		//                 { "name": "Code w10", "project": { "name": "Windows 10" } }
		//             ]
		//           }
		//         ]|]
		//         { matchStatus  = 200,
		//           matchHeaders = [matchContentTypeJson]
		//         }
		{
			Description: "embeds an M2M relationship plus parent after update",
			Method:      "PATCH",
			Query:       "/users?id=eq.1&select=name,tasks(name,project:projects(name))",
			Body:        `{"name": "Kevin Malone"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected: `[
				          {
				            "name": "Kevin Malone",
				            "tasks": [
				                { "name": "Design w7", "project": { "name": "Windows 7" } },
				                { "name": "Code w7", "project": { "name": "Windows 7" } },
				                { "name": "Design w10", "project": { "name": "Windows 10" } },
				                { "name": "Code w10", "project": { "name": "Windows 10" } }
				            ]
				          }
				        ]`,
			Status: 200,
		},
		//     it "embeds an O2O relationship after update" $ do
		//       request methodPatch "/students?id=eq.1&select=name,students_info(address)"
		//               [("Prefer", "return=representation")]
		//         [json|{"name": "Johnny Doe"}|]
		//         `shouldRespondWith`
		//         [json|[
		//           {
		//             "name": "Johnny Doe",
		//             "students_info":{"address":"Street 1"}
		//           }
		//         ]|]
		//         { matchStatus  = 200,
		//           matchHeaders = [matchContentTypeJson]
		//         }
		{
			Description: "embeds an O2O relationship after update",
			Method:      "PATCH",
			Query:       "/students?id=eq.1&select=name,students_info(address)",
			Body:        `{"name": "Johnny Doe"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected: `[
				           {
				             "name": "Johnny Doe",
				             "students_info":{"address":"Street 1"}
				           }
				         ]`,
			Status: 200,
		},
		//       request methodPatch "/students_info?id=eq.1&select=address,students(name)"
		//               [("Prefer", "return=representation")]
		//         [json|{"address": "New Street 1"}|]
		//         `shouldRespondWith`
		//         [json|[
		//           {
		//             "address": "New Street 1",
		//             "students":{"name": "John Doe"}
		//           }
		//         ]|]
		//         { matchStatus  = 200,
		//           matchHeaders = [matchContentTypeJson]
		//         }
		{
			Description: "embeds an O2O relationship after update",
			Method:      "PATCH",
			Query:       "/students_info?id=eq.1&select=address,students(name)",
			Body:        `{"address": "New Street 1"}`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected: `[
				           {
				             "address": "New Street 1",
				             "students":{"name": "John Doe"}
				           }
				         ]`,
			Status: 200,
		},
		//   context "table with limited privileges" $ do
		//     it "succeeds updating row and gives a 204 when using return=minimal" $
		//       request methodPatch "/app_users?id=eq.1"
		//           [("Prefer", "return=minimal")]
		//           [json| { "password": "passxyz" } |]
		//         `shouldRespondWith`
		//           ""
		//           { matchStatus = 204
		//           , matchHeaders = [matchHeaderAbsent hContentType]
		//           }
		{
			Description: "table with limited privileges succeeds updating row and gives a 204 when using return=minimal",
			Method:      "PATCH",
			Query:       "/app_users?id=eq.1",
			Body:        `{ "password": "passxyz" }`,
			Headers:     test.Headers{"Prefer": {"return=minimal"}},
			Status:      204,
		},
		//     it "can update without return=minimal and no explicit select" $
		//       request methodPatch "/app_users?id=eq.1"
		//           []
		//           [json| { "password": "passabc" } |]
		//         `shouldRespondWith`
		//           ""
		//           { matchStatus = 204
		//           , matchHeaders = [matchHeaderAbsent hContentType]
		//           }
		{
			Description: "table with limited privileges can update without return=minimal and no explicit select",
			Method:      "PATCH",
			Query:       "/app_users?id=eq.1",
			Body:        `{ "password": "passabc" }`,
			Headers:     nil,
			Status:      204,
		},
		//   context "limited update" $ do
		//     it "works with the limit query param" $
		//       baseTable "limited_update_items" "id" tblDataBefore
		//       `mutatesWith`
		//       requestMutation methodPatch "/limited_update_items?order=id&limit=2"
		//         [json| {"name": "updated-item"} |]
		//       `shouldMutateInto`
		//       [json|[
		//         { "id": 1, "name": "updated-item" }
		//       , { "id": 2, "name": "updated-item" }
		//       , { "id": 3, "name": "item-3" }
		//       ]|]

		//     it "works with the limit query param plus a filter" $
		//       baseTable "limited_update_items" "id" tblDataBefore
		//       `mutatesWith`
		//       requestMutation methodPatch "/limited_update_items?order=id&limit=1&id=gt.2"
		//         [json| {"name": "updated-item"} |]
		//       `shouldMutateInto`
		//       [json|[
		//         { "id": 1, "name": "item-1" }
		//       , { "id": 2, "name": "item-2" }
		//       , { "id": 3, "name": "updated-item" }
		//       ]|]

		//     it "works with the limit and offset query params" $
		//       baseTable "limited_update_items" "id" tblDataBefore
		//       `mutatesWith`
		//       requestMutation methodPatch "/limited_update_items?order=id&limit=1&offset=1"
		//         [json| {"name": "updated-item"} |]
		//       `shouldMutateInto`
		//       [json|[
		//         { "id": 1, "name": "item-1" }
		//       , { "id": 2, "name": "updated-item" }
		//       , { "id": 3, "name": "item-3" }
		//       ]|]

		//     it "fails without an explicit order by" $
		//       request methodPatch "/limited_update_items?limit=1&offset=1"
		//           [("Prefer", "tx=commit")]
		//           [json| {"name": "updated-item"} |]
		//         `shouldRespondWith`
		//           [json| {
		//             "code":"PGRST109",
		//             "hint": "Apply an 'order' using unique column(s)",
		//             "details": null,
		//             "message": "A 'limit' was applied without an explicit 'order'"
		//             }|]
		//           { matchStatus  = 400 }

		//     it "fails when not ordering by a unique column" $
		//       request methodPatch "/limited_update_items_wnonuniq_view?order=static&limit=1"
		//           [("Prefer", "tx=commit")]
		//           [json| {"name": "updated-item"} |]
		//         `shouldRespondWith`
		//           [json| {
		//             "code":"PGRST110",
		//             "hint": null,
		//             "details":"Results contain 3 rows changed but the maximum number allowed is 1",
		//             "message":"The maximum number of rows allowed to change was surpassed"
		//             }|]
		//           { matchStatus  = 400 }

		//     it "works with views with an explicit order by unique col" $
		//       baseTable "limited_update_items_view" "id" tblDataBefore
		//       `mutatesWith`
		//       requestMutation methodPatch "/limited_update_items_view?order=id&limit=1&offset=1"
		//         [json| {"name": "updated-item"} |]
		//       `shouldMutateInto`
		//       [json|[
		//         { "id": 1, "name": "item-1" }
		//       , { "id": 2, "name": "updated-item" }
		//       , { "id": 3, "name": "item-3" }
		//       ]|]

		//     it "works with views with an explicit order by composite pk" $
		//       baseTable "limited_update_items_cpk_view" "id" tblDataBefore
		//       `mutatesWith`
		//       requestMutation methodPatch "/limited_update_items_cpk_view?order=id,name&limit=1&offset=1"
		//         [json| {"name": "updated-item"} |]
		//       `shouldMutateInto`
		//       [json|[
		//         { "id": 1, "name": "item-1" }
		//       , { "id": 2, "name": "updated-item" }
		//       , { "id": 3, "name": "item-3" }
		//       ]|]

		//     it "works on a table without a pk by ordering by 'ctid'" $
		//       baseTable "limited_update_items_no_pk" "id" tblDataBefore
		//       `mutatesWith`
		//       requestMutation methodPatch "/limited_update_items_no_pk?order=ctid&limit=1"
		//         [json| {"name": "updated-item"} |]
		//       `shouldMutateInto`
		//       [json|[
		//         { "id": 1, "name": "updated-item" }
		//       , { "id": 2, "name": "item-2" }
		//       , { "id": 3, "name": "item-3" }
		//       ]|]
		// @@ added
		{
			Description: "with camel case columns works",
			Method:      "PATCH",
			Query:       "/UnitTest?idUnitTest=eq.1",
			Body:        `{ "nameUnitTest": "name of unittest 2" }`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[{ "idUnitTest": 1, "nameUnitTest": "name of unittest 2" }]`,
			Status:      201,
		},
	}

	test.Execute(t, testConfig, tests)
}
