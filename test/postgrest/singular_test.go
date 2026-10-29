package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func TestPostgREST_Singular(t *testing.T) {

	tests := []test.Test{
		// describe "Requesting singular json object" $ do
		//   let singular = ("Accept", "application/vnd.pgrst.object+json")
		//
		//   context "with GET request" $ do
		//     it "fails for zero rows" $
		//       request methodGet "/items?id=gt.0&id=lt.0" [singular] ""
		//         `shouldRespondWith` 406
		{
			Description: "with GET request fails for zero rows",
			Method:      "GET",
			Query:       "/items?id=gt.0&id=lt.0",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
			Status:      406,
		},
		//     it "will select an existing object" $ do
		//       request methodGet "/items?id=eq.5" [singular] ""
		//         `shouldRespondWith` [json|{"id":5}|]
		//           { matchHeaders = [matchContentTypeSingular]
		//           }
		// @@ not asserted, PostgREST 16.4: Content-Type "application/vnd.pgrst.object+json; charset=utf-8" (smoothdb sends no charset); card 32503 (media types)
		{
			Description: "with GET request will select an existing object",
			Method:      "GET",
			Query:       "/items?id=eq.5",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
			Expected:    `{"id":5}`,
			Status:      200,
		},
		//       -- also test without the +json suffix
		//       request
		//         methodGet
		//         "/items?id=eq.5"
		//         [("Accept", "application/vnd.pgrst.object")]
		//         ""
		//         `shouldRespondWith` [json|{"id":5}|]
		//           { matchHeaders = [matchContentTypeSingular]
		//           }
		{
			Description: "with GET request will select an existing object, Accept without the +json suffix",
			Method:      "GET",
			Query:       "/items?id=eq.5",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object"}},
			Expected:    `{"id":5}`,
			Status:      200,
			Skip:        "PostgREST 16.4: Accept application/vnd.pgrst.object (without +json) selects the singular object, smoothdb answers 406; card 32503 (media types)",
		},
		//     it "can combine multiple prefer values" $
		//       request methodGet "/items?id=eq.5" [singular, ("Prefer", "count=none")] ""
		//         `shouldRespondWith` [json|{"id":5}|]
		//           { matchHeaders = [matchContentTypeSingular]
		//           }
		// @@ not asserted, PostgREST 16.4: Content-Type "application/vnd.pgrst.object+json; charset=utf-8" (smoothdb sends no charset); card 32503 (media types)
		{
			Description: "with GET request can combine multiple prefer values",
			Method:      "GET",
			Query:       "/items?id=eq.5",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"count=none"}},
			Expected:    `{"id":5}`,
			Status:      200,
		},
		//     it "can shape plurality singular object routes" $
		//       request methodGet "/projects_view?id=eq.1&select=id,name,clients(*),tasks(id,name)" [singular] ""
		//         `shouldRespondWith` [json|{"id":1,"name":"Windows 7","clients":{"id":1,"name":"Microsoft"},"tasks":[{"id":1,"name":"Design w7"},{"id":2,"name":"Code w7"}]}|]
		//           { matchHeaders = [matchContentTypeSingular]
		//           }
		// @@ not asserted, PostgREST 16.4: Content-Type "application/vnd.pgrst.object+json; charset=utf-8" (smoothdb sends no charset); card 32503 (media types)
		{
			Description: "with GET request can shape plurality singular object routes",
			Method:      "GET",
			Query:       "/projects_view?id=eq.1&select=id,name,clients(*),tasks(id,name)",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
			Expected:    `{"id":1,"name":"Windows 7","clients":{"id":1,"name":"Microsoft"},"tasks":[{"id":1,"name":"Design w7"},{"id":2,"name":"Code w7"}]}`,
			Status:      200,
		},
		//   context "when updating rows" $ do
		//     it "works for one row with return=rep" $ do
		//       request
		//         methodPatch
		//         "/addresses?id=eq.1"
		//         [("Prefer", "return=representation"), singular]
		//         [json| { address: "B Street" } |]
		//         `shouldRespondWith` [json|{"id":1,"address":"B Street"}|]
		//           { matchHeaders = [matchContentTypeSingular]
		//           }
		// @@ not asserted, PostgREST 16.4: Content-Type "application/vnd.pgrst.object+json; charset=utf-8" (smoothdb sends no charset); card 32503 (media types)
		{
			Description: "when updating rows works for one row with return=rep",
			Method:      "PATCH",
			Query:       "/addresses?id=eq.1",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"return=representation"}},
			Body:        `{ "address": "B Street" }`,
			Expected:    `{"id":1,"address":"B Street"}`,
			Status:      200,
		},
		//     it "works for one row with return=minimal" $
		//       request
		//         methodPatch
		//         "/addresses?id=eq.1"
		//         [("Prefer", "return=minimal"), singular]
		//         [json| { address: "C Street" } |]
		//         `shouldRespondWith` ""
		//           { matchStatus = 204
		//           , matchHeaders =
		//               [ matchHeaderAbsent hContentType
		//               , "Preference-Applied" <:> "return=minimal"
		//               ]
		//           }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "when updating rows works for one row with return=minimal",
			Method:          "PATCH",
			Query:           "/addresses?id=eq.1",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"return=minimal"}},
			Body:            `{ "address": "C Street" }`,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": ""},
			Status:          204,
		},
		//     it "raises an error for multiple rows" $ do
		//       request
		//         methodPatch
		//         "/addresses"
		//         [("Prefer", "tx=commit"), singular]
		//         [json| { address: "zzz" } |]
		//         `shouldRespondWith` [json|{"details":"The result contains 4 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description: "when updating rows raises an error for multiple rows",
			Method:      "PATCH",
			Query:       "/addresses",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"tx=commit"}},
			Body:        `{ "address": "zzz" }`,
			Status:      406,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on a write without return=representation still refuses more than one row (406, PGRST116) and rolls back; no card",
		},
		//       -- the rows should not be updated, either
		//       get "/addresses?id=eq.1"
		//         `shouldRespondWith` [json|[{"id":1,"address":"address 1"}]|]
		{
			Description: "when updating rows raises an error for multiple rows: the rows should not be updated, either",
			Method:      "GET",
			Query:       "/addresses?id=eq.1",
			Expected:    `[{"id":1,"address":"address 1"}]`,
			Status:      200,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on a write without return=representation still refuses more than one row (406, PGRST116) and rolls back; no card",
		},
		//     it "raises an error for multiple rows with return=rep" $ do
		//       request
		//         methodPatch
		//         "/addresses"
		//         [("Prefer", "tx=commit"), ("Prefer", "return=representation"), singular]
		//         [json| { address: "zzz" } |]
		//         `shouldRespondWith` [json|{"details":"The result contains 4 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		// @@ not asserted, PostgREST 16.4: the PGRST116 body and its application/json Content-Type (smoothdb answers 406 with an empty body); card 32494 (error body)
		{
			Description: "when updating rows raises an error for multiple rows with return=rep",
			Method:      "PATCH",
			Query:       "/addresses",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"tx=commit", "return=representation"}},
			Body:        `{ "address": "zzz" }`,
			Status:      406,
		},
		//       -- the rows should not be updated, either
		//       get "/addresses?id=eq.1"
		//         `shouldRespondWith` [json|[{"id":1,"address":"address 1"}]|]
		{
			Description: "when updating rows raises an error for multiple rows with return=rep: the rows should not be updated, either",
			Method:      "GET",
			Query:       "/addresses?id=eq.1",
			Expected:    `[{"id":1,"address":"address 1"}]`,
			Status:      200,
		},
		//     it "raises an error for zero rows" $
		//       request
		//         methodPatch
		//         "/items?id=gt.0&id=lt.0"
		//         [singular]
		//         [json|{"id":1}|]
		//         `shouldRespondWith` [json|{"details":"The result contains 0 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description: "when updating rows raises an error for zero rows",
			Method:      "PATCH",
			Query:       "/items?id=gt.0&id=lt.0",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
			Body:        `{"id":1}`,
			Status:      406,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on a write without return=representation still refuses zero rows (406, PGRST116); no card",
		},
		//     it "raises an error for zero rows with return=rep" $
		//       request
		//         methodPatch
		//         "/items?id=gt.0&id=lt.0"
		//         [("Prefer", "return=representation"), singular]
		//         [json|{"id":1}|]
		//         `shouldRespondWith` [json|{"details":"The result contains 0 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		// @@ not asserted, PostgREST 16.4: the PGRST116 body and its application/json Content-Type (smoothdb answers 406 with an empty body); card 32494 (error body)
		{
			Description: "when updating rows raises an error for zero rows with return=rep",
			Method:      "PATCH",
			Query:       "/items?id=gt.0&id=lt.0",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"return=representation"}},
			Body:        `{"id":1}`,
			Status:      406,
		},
		//
		//   context "when creating rows" $ do
		//     it "works for one row with return=rep" $ do
		//       request
		//         methodPost
		//         "/addresses"
		//         [("Prefer", "return=representation"), singular]
		//         [json| [ { id: 102, address: "xxx" } ] |]
		//         `shouldRespondWith` [json|{"id":102,"address":"xxx"}|]
		//           { matchStatus = 201
		//           , matchHeaders = [matchContentTypeSingular]
		//           }
		// @@ not asserted, PostgREST 16.4: Content-Type "application/vnd.pgrst.object+json; charset=utf-8" (smoothdb sends no charset); card 32503 (media types)
		{
			Description: "when creating rows works for one row with return=rep",
			Method:      "POST",
			Query:       "/addresses",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"return=representation"}},
			Body:        `[ { "id": 102, "address": "xxx" } ]`,
			Expected:    `{"id":102,"address":"xxx"}`,
			Status:      201,
		},
		//     it "works for one row with return=minimal" $ do
		//       request
		//         methodPost
		//         "/addresses"
		//         [("Prefer", "return=minimal"), singular]
		//         [json| [ { id: 103, address: "xxx" } ] |]
		//         `shouldRespondWith` ""
		//           { matchStatus = 201
		//           , matchHeaders =
		//               [ matchHeaderAbsent hContentType
		//               , "Content-Range" <:> "*/*"
		//               , "Preference-Applied" <:> "return=minimal"
		//               ]
		//           }
		// @@ not asserted, PostgREST 16.4: Preference-Applied; card 32479 (write headers)
		{
			Description:     "when creating rows works for one row with return=minimal",
			Method:          "POST",
			Query:           "/addresses",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"return=minimal"}},
			Body:            `[ { "id": 103, "address": "xxx" } ]`,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Content-Range": "*/*"},
			Status:          201,
		},
		//     it "raises an error when attempting to create multiple entities" $ do
		//       request
		//         methodPost
		//         "/addresses"
		//         [("Prefer", "tx=commit"), singular]
		//         [json| [ { id: 200, address: "xxx" }, { id: 201, address: "yyy" } ] |]
		//         `shouldRespondWith` [json|{"details":"The result contains 2 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description: "when creating rows raises an error when attempting to create multiple entities",
			Method:      "POST",
			Query:       "/addresses",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"tx=commit"}},
			Body:        `[ { "id": 200, "address": "xxx" }, { "id": 201, "address": "yyy" } ]`,
			Status:      406,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on a write without return=representation still refuses more than one row (406, PGRST116) and rolls back; no card",
		},
		//       -- the rows should not exist, either
		//       get "/addresses?id=eq.200"
		//         `shouldRespondWith` "[]"
		{
			Description: "when creating rows raises an error when attempting to create multiple entities: the rows should not exist, either",
			Method:      "GET",
			Query:       "/addresses?id=eq.200",
			Expected:    `[]`,
			Status:      200,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on a write without return=representation still refuses more than one row (406, PGRST116) and rolls back; no card",
		},
		//     it "raises an error when attempting to create multiple entities with return=rep" $ do
		//       request
		//         methodPost
		//         "/addresses"
		//         [("Prefer", "tx=commit"), ("Prefer", "return=representation"), singular]
		//         [json| [ { id: 202, address: "xxx" }, { id: 203, address: "yyy" } ] |]
		//         `shouldRespondWith` [json|{"details":"The result contains 2 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		// @@ not asserted, PostgREST 16.4: the PGRST116 body and its application/json Content-Type (smoothdb answers 406 with an empty body); card 32494 (error body)
		{
			Description: "when creating rows raises an error when attempting to create multiple entities with return=rep",
			Method:      "POST",
			Query:       "/addresses",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"tx=commit", "return=representation"}},
			Body:        `[ { "id": 202, "address": "xxx" }, { "id": 203, "address": "yyy" } ]`,
			Status:      406,
		},
		//       -- the rows should not exist, either
		//       get "/addresses?id=eq.202"
		//         `shouldRespondWith` "[]"
		{
			Description: "when creating rows raises an error when attempting to create multiple entities with return=rep: the rows should not exist, either",
			Method:      "GET",
			Query:       "/addresses?id=eq.202",
			Expected:    `[]`,
			Status:      200,
		},
		//     it "raises an error regardless of return=minimal" $ do
		//       request
		//         methodPost
		//         "/addresses"
		//         [("Prefer", "tx=commit"), ("Prefer", "return=minimal"), singular]
		//         [json| [ { id: 204, address: "xxx" }, { id: 205, address: "yyy" } ] |]
		//         `shouldRespondWith` [json|{"details":"The result contains 2 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description: "when creating rows raises an error regardless of return=minimal",
			Method:      "POST",
			Query:       "/addresses",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"tx=commit", "return=minimal"}},
			Body:        `[ { "id": 204, "address": "xxx" }, { "id": 205, "address": "yyy" } ]`,
			Status:      406,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on a write without return=representation still refuses more than one row (406, PGRST116) and rolls back; no card",
		},
		//       -- the rows should not exist, either
		//       get "/addresses?id=eq.204"
		//         `shouldRespondWith` "[]"
		{
			Description: "when creating rows raises an error regardless of return=minimal: the rows should not exist, either",
			Method:      "GET",
			Query:       "/addresses?id=eq.204",
			Expected:    `[]`,
			Status:      200,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on a write without return=representation still refuses more than one row (406, PGRST116) and rolls back; no card",
		},
		//     it "raises an error when creating zero entities" $
		//       request
		//         methodPost
		//         "/addresses"
		//         [singular]
		//         [json| [ ] |]
		//         `shouldRespondWith` [json|{"details":"The result contains 0 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description: "when creating rows raises an error when creating zero entities",
			Method:      "POST",
			Query:       "/addresses",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
			Body:        `[ ]`,
			Status:      406,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on a write without return=representation still refuses zero rows (406, PGRST116); no card",
		},
		//     it "raises an error when creating zero entities with return=rep" $
		//       request
		//         methodPost
		//         "/addresses"
		//         [("Prefer", "return=representation"), singular]
		//         [json| [ ] |]
		//         `shouldRespondWith` [json|{"details":"The result contains 0 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description: "when creating rows raises an error when creating zero entities with return=rep",
			Method:      "POST",
			Query:       "/addresses",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"return=representation"}},
			Body:        `[ ]`,
			Status:      406,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on a POST of zero rows refuses it (406, PGRST116), smoothdb answers 201 with an empty body; no card",
		},
		//
		//   context "when deleting rows" $ do
		//     it "works for one row with return=rep" $ do
		//       p <-
		//         request
		//           methodDelete
		//           "/items?id=eq.11"
		//           [("Prefer", "return=representation"), singular]
		//           ""
		//       liftIO $ simpleBody p `shouldBe` [json|{"id":11}|]
		{
			Description: "when deleting rows works for one row with return=rep",
			Method:      "DELETE",
			Query:       "/items?id=eq.11",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"return=representation"}},
			Expected:    `{"id":11}`,
			Status:      200,
		},
		//     it "works for one row with return=minimal" $ do
		//       p <-
		//         request
		//           methodDelete
		//           "/items?id=eq.12"
		//           [("Prefer", "return=minimal"), singular]
		//           ""
		//       liftIO $ simpleBody p `shouldBe` ""
		{
			Description:   "when deleting rows works for one row with return=minimal",
			Method:        "DELETE",
			Query:         "/items?id=eq.12",
			Headers:       test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"return=minimal"}},
			ExpectedEmpty: true,
			Status:        204,
		},
		//     it "raises an error when attempting to delete multiple entities" $ do
		//       request
		//         methodDelete
		//         "/items?id=gt.0&id=lt.6"
		//         [("Prefer", "tx=commit"), singular]
		//         ""
		//         `shouldRespondWith` [json|{"details":"The result contains 5 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description: "when deleting rows raises an error when attempting to delete multiple entities",
			Method:      "DELETE",
			Query:       "/items?id=gt.0&id=lt.6",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"tx=commit"}},
			Status:      406,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on a write without return=representation still refuses more than one row (406, PGRST116) and rolls back; no card",
		},
		//       -- the rows should still exist
		//       get "/items?id=gt.0&id=lt.6&order=id"
		//         `shouldRespondWith` [json| [{"id":1},{"id":2},{"id":3},{"id":4},{"id":5}] |]
		//           { matchStatus = 200
		//           , matchHeaders = ["Content-Range" <:> "0-4/*"]
		//           }
		{
			Description:     "when deleting rows raises an error when attempting to delete multiple entities: the rows should still exist",
			Method:          "GET",
			Query:           "/items?id=gt.0&id=lt.6&order=id",
			Expected:        `[{"id":1},{"id":2},{"id":3},{"id":4},{"id":5}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-4/*"},
			Status:          200,
			Skip:            "PostgREST 16.4: vnd.pgrst.object on a write without return=representation still refuses more than one row (406, PGRST116) and rolls back; no card",
		},
		//     it "raises an error when attempting to delete multiple entities with return=rep" $ do
		//       request
		//         methodDelete
		//         "/items?id=gt.5&id=lt.11"
		//         [("Prefer", "tx=commit"), ("Prefer", "return=representation"), singular]
		//         ""
		//         `shouldRespondWith` [json|{"details":"The result contains 5 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		// @@ not asserted, PostgREST 16.4: the PGRST116 body and its application/json Content-Type (smoothdb answers 406 with an empty body); card 32494 (error body)
		{
			Description: "when deleting rows raises an error when attempting to delete multiple entities with return=rep",
			Method:      "DELETE",
			Query:       "/items?id=gt.5&id=lt.11",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"tx=commit", "return=representation"}},
			Status:      406,
		},
		//       -- the rows should still exist
		//       get "/items?id=gt.5&id=lt.11"
		//         `shouldRespondWith` [json| [{"id":6},{"id":7},{"id":8},{"id":9},{"id":10}] |]
		//           { matchStatus = 200
		//           , matchHeaders = ["Content-Range" <:> "0-4/*"]
		//           }
		{
			Description:     "when deleting rows raises an error when attempting to delete multiple entities with return=rep: the rows should still exist",
			Method:          "GET",
			Query:           "/items?id=gt.5&id=lt.11",
			Expected:        `[{"id":6},{"id":7},{"id":8},{"id":9},{"id":10}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-4/*"},
			Status:          200,
		},
		//     it "raises an error when deleting zero entities" $
		//       request
		//         methodDelete
		//         "/items?id=lt.0"
		//         [singular]
		//         ""
		//         `shouldRespondWith` [json|{"details":"The result contains 0 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description: "when deleting rows raises an error when deleting zero entities",
			Method:      "DELETE",
			Query:       "/items?id=lt.0",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
			Status:      406,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on a write without return=representation still refuses zero rows (406, PGRST116); no card",
		},
		//     it "raises an error when deleting zero entities with return=rep" $
		//       request
		//         methodDelete
		//         "/items?id=lt.0"
		//         [("Prefer", "return=representation"), singular]
		//         ""
		//         `shouldRespondWith` [json|{"details":"The result contains 0 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		// @@ not asserted, PostgREST 16.4: the PGRST116 body and its application/json Content-Type (smoothdb answers 406 with an empty body); card 32494 (error body)
		{
			Description: "when deleting rows raises an error when deleting zero entities with return=rep",
			Method:      "DELETE",
			Query:       "/items?id=lt.0",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"return=representation"}},
			Status:      406,
		},
		//
		//   context "when calling a stored proc" $ do
		//     it "fails for zero rows" $
		//       request
		//         methodPost
		//         "/rpc/getproject"
		//         [singular]
		//         [json|{ "id": 9999999}|]
		//         `shouldRespondWith` [json|{"details":"The result contains 0 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description: "when calling a stored proc fails for zero rows",
			Method:      "POST",
			Query:       "/rpc/getproject",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
			Body:        `{ "id": 9999999}`,
			Status:      406,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on an RPC returns one object, and refuses zero or many rows (406, PGRST116); card 32490 (rpc details)",
		},
		//     -- this one may be controversial, should vnd.pgrst.object include
		//     -- the likes of 2 and "hello?"
		//     it "succeeds for scalar result" $
		//       request
		//         methodPost
		//         "/rpc/sayhello"
		//         [singular]
		//         [json|{ "name": "world"}|]
		//         `shouldRespondWith` 200
		{
			Description: "when calling a stored proc succeeds for scalar result",
			Method:      "POST",
			Query:       "/rpc/sayhello",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
			Body:        `{ "name": "world"}`,
			Status:      200,
		},
		//     it "returns a single object for json proc" $
		//       request
		//         methodPost
		//         "/rpc/getproject"
		//         [singular]
		//         [json|{ "id": 1}|]
		//         `shouldRespondWith` [json|{"id":1,"name":"Windows 7","client_id":1}|]
		//           { matchHeaders = [matchContentTypeSingular]
		//           }
		{
			Description: "when calling a stored proc returns a single object for json proc",
			Method:      "POST",
			Query:       "/rpc/getproject",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
			Body:        `{ "id": 1}`,
			Expected:    `{"id":1,"name":"Windows 7","client_id":1}`,
			Status:      200,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on an RPC returns one object, and refuses zero or many rows (406, PGRST116); card 32490 (rpc details)",
		},
		//     it "fails for multiple rows" $
		//       request
		//         methodPost
		//         "/rpc/getallprojects"
		//         [singular]
		//         "{}"
		//         `shouldRespondWith` [json|{"details":"The result contains 5 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description: "when calling a stored proc fails for multiple rows",
			Method:      "POST",
			Query:       "/rpc/getallprojects",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
			Body:        `{}`,
			Status:      406,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on an RPC returns one object, and refuses zero or many rows (406, PGRST116); card 32490 (rpc details)",
		},
		//     it "fails for multiple rows with rolled back changes" $ do
		//       post
		//         "/rpc/getproject?select=id,name"
		//         [json| {"id": 1} |]
		//         `shouldRespondWith` [json|[{"id":1,"name":"Windows 7"}]|]
		{
			Description: "when calling a stored proc fails for multiple rows with rolled back changes",
			Method:      "POST",
			Query:       "/rpc/getproject?select=id,name",
			Body:        `{"id": 1}`,
			Expected:    `[{"id":1,"name":"Windows 7"}]`,
			Status:      200,
		},
		//       request
		//         methodPost
		//         "/rpc/setprojects"
		//         [("Prefer", "tx=commit"), singular]
		//         [json| {"id_l": 1, "id_h": 2, "name": "changed"} |]
		//         `shouldRespondWith` [json|{"details":"The result contains 2 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description: "when calling a stored proc fails for multiple rows with rolled back changes",
			Method:      "POST",
			Query:       "/rpc/setprojects",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}, "Prefer": {"tx=commit"}},
			Body:        `{"id_l": 1, "id_h": 2, "name": "changed"}`,
			Status:      406,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on an RPC returns one object, and refuses zero or many rows (406, PGRST116); card 32490 (rpc details)",
		},
		//       -- should rollback function
		//       post
		//         "/rpc/getproject?select=id,name"
		//         [json| {"id": 1} |]
		//         `shouldRespondWith` [json|[{"id":1,"name":"Windows 7"}]|]
		{
			Description: "when calling a stored proc fails for multiple rows with rolled back changes: the function is rolled back",
			Method:      "POST",
			Query:       "/rpc/getproject?select=id,name",
			Body:        `{"id": 1}`,
			Expected:    `[{"id":1,"name":"Windows 7"}]`,
			Status:      200,
			Skip:        "PostgREST 16.4: vnd.pgrst.object on an RPC returns one object, and refuses zero or many rows (406, PGRST116); card 32490 (rpc details)",
		},
	}

	test.Execute(t, testConfig, tests)
}
