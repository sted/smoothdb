package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func TestPostgREST_Singular(t *testing.T) {

	tests := []test.Test{
		// 	describe "Requesting singular json object" $ do
		// let singular = ("Accept", "application/vnd.pgrst.object+json")

		// context "with GET request" $ do
		//   it "fails for zero rows" $
		//     request methodGet  "/items?id=gt.0&id=lt.0" [singular] ""
		//       `shouldRespondWith` 406
		{
			Description: "fails for zero rows",
			Method:      "GET",
			Query:       "/items?id=gt.0&id=lt.0",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
			Status:      406,
		},
		//   it "will select an existing object" $ do
		//     request methodGet "/items?id=eq.5" [singular] ""
		//       `shouldRespondWith`
		//         [json|{"id":5}|]
		//         { matchHeaders = [matchContentTypeSingular] }
		{
			Description: "will select an existing object",
			Method:      "GET",
			Query:       "/items?id=eq.5",
			Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
			Expected:    `{"id":5}`,
			Status:      200,
		},
		//     -- also test without the +json suffix
		//     request methodGet "/items?id=eq.5"
		//         [("Accept", "application/vnd.pgrst.object")] ""
		//       `shouldRespondWith`
		//         [json|{"id":5}|]
		//         { matchHeaders = [matchContentTypeSingular] }

		//   it "can combine multiple prefer values" $
		//     request methodGet "/items?id=eq.5" [singular, ("Prefer","count=none")] ""
		//       `shouldRespondWith`
		//         [json|{"id":5}|]
		//         { matchHeaders = [matchContentTypeSingular] }

		//   it "can shape plurality singular object routes" $
		//     request methodGet "/projects_view?id=eq.1&select=id,name,clients(*),tasks(id,name)" [singular] ""
		//       `shouldRespondWith`
		//         [json|{"id":1,"name":"Windows 7","clients":{"id":1,"name":"Microsoft"},"tasks":[{"id":1,"name":"Design w7"},{"id":2,"name":"Code w7"}]}|]
		//         { matchHeaders = [matchContentTypeSingular] }
		// {
		// 	Description: "can shape plurality singular object routes",
		// 	Method:      "GET",
		// 	Query:       "/projects_view?id=eq.1&select=id,name,clients(*),tasks(id,name)",
		// 	Headers:     test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
		// 	Expected:    `{"id":1,"name":"Windows 7","clients":{"id":1,"name":"Microsoft"},"tasks":[{"id":1,"name":"Design w7"},{"id":2,"name":"Code w7"}]}`,
		// 	Status:      200,
		// },
		// context "when updating rows" $ do
		//   it "works for one row with return=rep" $ do
		//     request methodPatch "/addresses?id=eq.1"
		//         [("Prefer", "return=representation"), singular]
		//         [json| { address: "B Street" } |]
		//       `shouldRespondWith`
		//         [json|{"id":1,"address":"B Street"}|]
		//         { matchHeaders = [matchContentTypeSingular] }
		{
			Description: "works for one row with return=rep",
			Method:      "PATCH",
			Query:       "/addresses?id=eq.1",
			Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
				"Prefer": {"return=representation"}},
			Body:     `{ "address": "B Street" }`,
			Expected: `{"id":1,"address":"B Street"}`,
			Status:   200,
		},
		//   it "works for one row with return=minimal" $
		//     request methodPatch "/addresses?id=eq.1"
		//         [("Prefer", "return=minimal"), singular]
		//         [json| { address: "C Street" } |]
		//       `shouldRespondWith`
		//         ""
		//         { matchStatus  = 204
		//         , matchHeaders = [matchHeaderAbsent hContentType]
		//         }
		{
			Description: "works for one row with return=minimal",
			Method:      "PATCH",
			Query:       "/addresses?id=eq.1",
			Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
				"Prefer": {"return=minimal"}},
			Body:     `{ "address": "C Street" }`,
			Expected: ``,
			Status:   204,
		},
		//   it "raises an error for multiple rows" $ do
		//     request methodPatch "/addresses"
		//         [("Prefer", "tx=commit"), singular]
		//         [json| { address: "zzz" } |]
		//       `shouldRespondWith`
		//         [json|{"details":"Results contain 4 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//         { matchStatus  = 406
		//         , matchHeaders = [ matchContentTypeSingular
		//                          , "Preference-Applied" <:> "tx=commit" ]
		//         }
		// {
		// 	Description: "raises an error for multiple rows",
		// 	Method:      "PATCH",
		// 	Query:       "/addresses",
		// 	Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
		// 		"Prefer": {"tx=commit"}},
		// 	Body:   `{ "address": "zzz" }`,
		// 	Status: 406,
		// },
		//     -- the rows should not be updated, either
		//     get "/addresses?id=eq.1"
		//       `shouldRespondWith`
		//         [json|[{"id":1,"address":"address 1"}]|]
		// {
		// 	Description: "the rows should not be updated, either",
		// 	Query:       "/addresses?id=eq.1",
		// 	Expected:    `[{"id":1,"address":"address 1"}]`,
		// },
		//   it "raises an error for multiple rows with return=rep" $ do
		//     request methodPatch "/addresses"
		//         [("Prefer", "tx=commit"), ("Prefer", "return=representation"), singular]
		//         [json| { address: "zzz" } |]
		//       `shouldRespondWith`
		//         [json|{"details":"Results contain 4 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//         { matchStatus  = 406
		//         , matchHeaders = [ matchContentTypeSingular
		//                          , "Preference-Applied" <:> "tx=commit" ]
		//         }
		{
			Description: "raises an error for multiple rows with return=rep",
			Method:      "PATCH",
			Query:       "/addresses",
			Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
				"Prefer": {"return=representation", "tx=commit"}},
			Body:   `{ "address": "zzz" }`,
			Status: 406,
		},
		//     -- the rows should not be updated, either
		//     get "/addresses?id=eq.1"
		//       `shouldRespondWith`
		//         [json|[{"id":1,"address":"address 1"}]|]
		{
			Description: "the rows should not be updated, either",
			Query:       "/addresses?id=eq.1",
			Expected:    `[{"id":1,"address":"address 1"}]`,
		},
		//   it "raises an error for zero rows" $
		//     request methodPatch "/items?id=gt.0&id=lt.0"
		//             [singular] [json|{"id":1}|]
		//       `shouldRespondWith`
		//               [json|{"details":"Results contain 0 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//               { matchStatus  = 406
		//               , matchHeaders = [matchContentTypeSingular]
		//               }

		//   it "raises an error for zero rows with return=rep" $
		//     request methodPatch "/items?id=gt.0&id=lt.0"
		//             [("Prefer", "return=representation"), singular] [json|{"id":1}|]
		//       `shouldRespondWith`
		//               [json|{"details":"Results contain 0 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//               { matchStatus  = 406
		//               , matchHeaders = [matchContentTypeSingular]
		//               }
		{
			Description: "raises an error for zero rows with return=rep",
			Method:      "PATCH",
			Query:       "/items?id=gt.0&id=lt.0",
			Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
				"Prefer": {"return=representation"}},
			Body:   `{"id":1}`,
			Status: 406,
		},
		// context "when creating rows" $ do
		//   it "works for one row with return=rep" $ do
		//     request methodPost "/addresses"
		//         [("Prefer", "return=representation"), singular]
		//         [json| [ { id: 102, address: "xxx" } ] |]
		//       `shouldRespondWith`
		//         [json|{"id":102,"address":"xxx"}|]
		//         { matchStatus  = 201
		//         , matchHeaders = [matchContentTypeSingular]
		//         }
		{
			Description: "works for one row with return=rep",
			Method:      "POST",
			Query:       "/addresses",
			Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
				"Prefer": {"return=representation"}},
			Body:     `[ { "id": 102, "address": "xxx" } ] `,
			Expected: `{"id":102,"address":"xxx"}`,
			Status:   201,
		},
		//   it "works for one row with return=minimal" $ do
		//     request methodPost "/addresses"
		//         [("Prefer", "return=minimal"), singular]
		//         [json| [ { id: 103, address: "xxx" } ] |]
		//       `shouldRespondWith`
		//         ""
		//         { matchStatus  = 201
		//         , matchHeaders = [ matchHeaderAbsent hContentType
		//                          , "Content-Range" <:> "*/*" ]
		//         }
		{
			Description: "works for one row with return=minimal",
			Method:      "POST",
			Query:       "/addresses?id=eq.1",
			Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
				"Prefer": {"return=minimal"}},
			Body:     `[ { "id": 103, "address": "xxx" } ]`,
			Expected: ``,
			Status:   201,
		},
		//   it "raises an error when attempting to create multiple entities" $ do
		//     request methodPost "/addresses"
		//         [("Prefer", "tx=commit"), singular]
		//         [json| [ { id: 200, address: "xxx" }, { id: 201, address: "yyy" } ] |]
		//       `shouldRespondWith`
		//         [json|{"details":"Results contain 2 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//         { matchStatus  = 406
		//         , matchHeaders = [ matchContentTypeSingular
		//                          , "Preference-Applied" <:> "tx=commit" ]
		//         }
		//@@ is this right? even with no return=rep ?
		// {
		// 	Description: "raises an error when attempting to create multiple entities",
		// 	Method:      "POST",
		// 	Query:       "/addresses",
		// 	Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
		// 		"Prefer": {"tx=commit"}},
		// 	Body:   `[ { "id": 200, "address": "xxx" }, { "id": 201, "address": "yyy" } ] `,
		// 	Status: 406,
		// },
		//     -- the rows should not exist, either
		//     get "/addresses?id=eq.200"
		//       `shouldRespondWith`
		//         "[]"

		//   it "raises an error when attempting to create multiple entities with return=rep" $ do
		//     request methodPost "/addresses"
		//         [("Prefer", "tx=commit"), ("Prefer", "return=representation"), singular]
		//         [json| [ { id: 202, address: "xxx" }, { id: 203, address: "yyy" } ] |]
		//       `shouldRespondWith`
		//         [json|{"details":"Results contain 2 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//         { matchStatus  = 406
		//         , matchHeaders = [ matchContentTypeSingular
		//                          , "Preference-Applied" <:> "tx=commit" ]
		//         }
		{
			Description: "raises an error for multiple rows with return=rep",
			Method:      "POST",
			Query:       "/addresses",
			Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
				"Prefer": {"return=representation", "tx=commit"}},
			Body:   `[ { "id": 202, "address": "xxx" }, { "id": 203, "address": "yyy" } ] `,
			Status: 406,
		},
		//     -- the rows should not exist, either
		//     get "/addresses?id=eq.202"
		//       `shouldRespondWith`
		//         "[]"
		{
			Description: "the rows should not be updated, either",
			Query:       "/addresses?id=eq.202",
			Expected:    `[]`,
		},
		//   it "raises an error regardless of return=minimal" $ do
		//     request methodPost "/addresses"
		//         [("Prefer", "tx=commit"), ("Prefer", "return=minimal"), singular]
		//         [json| [ { id: 204, address: "xxx" }, { id: 205, address: "yyy" } ] |]
		//       `shouldRespondWith`
		//         [json|{"details":"Results contain 2 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//         { matchStatus  = 406
		//         , matchHeaders = [ matchContentTypeSingular
		//                          , "Preference-Applied" <:> "tx=commit" ]
		//         }

		//     -- the rows should not exist, either
		//     get "/addresses?id=eq.204"
		//       `shouldRespondWith`
		//         "[]"

		//   it "raises an error when creating zero entities" $
		//     request methodPost "/addresses"
		//             [singular]
		//             [json| [ ] |]
		//       `shouldRespondWith`
		//               [json|{"details":"Results contain 0 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//               { matchStatus  = 406
		//               , matchHeaders = [matchContentTypeSingular]
		//               }

		//   it "raises an error when creating zero entities with return=rep" $
		//     request methodPost "/addresses"
		//             [("Prefer", "return=representation"), singular]
		//             [json| [ ] |]
		//       `shouldRespondWith`
		//               [json|{"details":"Results contain 0 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//               { matchStatus  = 406
		//               , matchHeaders = [matchContentTypeSingular]
		//               }
		// {
		// 	Description: "raises an error when creating zero entities with return=rep",
		// 	Method:      "POST",
		// 	Query:       "/addresses",
		// 	Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
		// 		"Prefer": {"return=representation"}},
		// 	Body:   `[ ] `,
		// 	Status: 406,
		// },
		// context "when deleting rows" $ do
		//   it "works for one row with return=rep" $ do
		//     p <- request methodDelete
		//       "/items?id=eq.11"
		//       [("Prefer", "return=representation"), singular] ""
		//     liftIO $ simpleBody p `shouldBe` [json|{"id":11}|]
		{
			Description: "works for one row with return=rep",
			Method:      "DELETE",
			Query:       "/items?id=eq.11",
			Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
				"Prefer": {"return=representation"}},
			Expected: `{"id":11}`,
			Status:   200,
		},
		//   it "works for one row with return=minimal" $ do
		//     p <- request methodDelete
		//       "/items?id=eq.12"
		//       [("Prefer", "return=minimal"), singular] ""
		//     liftIO $ simpleBody p `shouldBe` ""
		{
			Description: "works for one row with return=minimal",
			Method:      "DELETE",
			Query:       "/items?id=eq.12",
			Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
				"Prefer": {"return=minimal"}},
			Body:     ``,
			Expected: ``,
			Status:   204,
		},
		//   it "raises an error when attempting to delete multiple entities" $ do
		//     request methodDelete "/items?id=gt.0&id=lt.6"
		//         [("Prefer", "tx=commit"), singular]
		//         ""
		//       `shouldRespondWith`
		//         [json|{"details":"Results contain 5 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//         { matchStatus  = 406
		//         , matchHeaders = [ matchContentTypeSingular
		//                          , "Preference-Applied" <:> "tx=commit" ]
		//         }
		{
			Description: "works for one row with return=rep",
			Method:      "DELETE",
			Query:       "/items?id=gt.0&id=lt.6",
			Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
				"Prefer": {"return=representation", "tx=commit"}},
			Status: 406,
		},
		//     -- the rows should still exist
		//     get "/items?id=gt.0&id=lt.6&order=id"
		//       `shouldRespondWith`
		//         [json| [{"id":1},{"id":2},{"id":3},{"id":4},{"id":5}] |]
		//         { matchStatus  = 200
		//         , matchHeaders = ["Content-Range" <:> "0-4/*"]
		//         }
		{
			Description: "the rows should still exist",
			Query:       "/items?id=gt.0&id=lt.6&order=id",
			Expected:    `[{"id":1},{"id":2},{"id":3},{"id":4},{"id":5}]`,
		},
		//   it "raises an error when attempting to delete multiple entities with return=rep" $ do
		//     request methodDelete "/items?id=gt.5&id=lt.11"
		//         [("Prefer", "tx=commit"), ("Prefer", "return=representation"), singular] ""
		//       `shouldRespondWith`
		//         [json|{"details":"Results contain 5 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//         { matchStatus  = 406
		//         , matchHeaders = [ matchContentTypeSingular
		//                          , "Preference-Applied" <:> "tx=commit" ]
		//         }
		{
			Description: "raises an error when attempting to delete multiple entities with return=rep",
			Method:      "DELETE",
			Query:       "/items?id=gt.5&id=lt.11",
			Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
				"Prefer": {"return=representation", "tx=commit"}},
			Status: 406,
		},
		//     -- the rows should still exist
		//     get "/items?id=gt.5&id=lt.11"
		//       `shouldRespondWith` [json| [{"id":6},{"id":7},{"id":8},{"id":9},{"id":10}] |]
		//         { matchStatus  = 200
		//         , matchHeaders = ["Content-Range" <:> "0-4/*"]
		//         }
		{
			Description: "the rows should still exist",
			Query:       "/items?id=gt.5&id=lt.11",
			Expected:    `[{"id":6},{"id":7},{"id":8},{"id":9},{"id":10}]`,
		},
		//   it "raises an error when deleting zero entities" $
		//     request methodDelete "/items?id=lt.0"
		//             [singular] ""
		//       `shouldRespondWith`
		//             [json|{"details":"Results contain 0 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//             { matchStatus  = 406
		//             , matchHeaders = [matchContentTypeSingular]
		//             }

		//   it "raises an error when deleting zero entities with return=rep" $
		//     request methodDelete "/items?id=lt.0"
		//             [("Prefer", "return=representation"), singular] ""
		//       `shouldRespondWith`
		//             [json|{"details":"Results contain 0 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//             { matchStatus  = 406
		//             , matchHeaders = [matchContentTypeSingular]
		//             }
		{
			Description: "raises an error when deleting zero entities with return=rep",
			Method:      "DELETE",
			Query:       "/items?id=lt.0",
			Headers: test.Headers{"Accept": {"application/vnd.pgrst.object+json"},
				"Prefer": {"return=representation"}},
			Status: 406,
		},
		// context "when calling a stored proc" $ do
		//   it "fails for zero rows" $
		//     request methodPost "/rpc/getproject"
		//             [singular] [json|{ "id": 9999999}|]
		//       `shouldRespondWith`
		//             [json|{"details":"Results contain 0 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//             { matchStatus  = 406
		//             , matchHeaders = [matchContentTypeSingular]
		//             }

		//   -- this one may be controversial, should vnd.pgrst.object include
		//   -- the likes of 2 and "hello?"
		//   it "succeeds for scalar result" $
		//     request methodPost "/rpc/sayhello"
		//       [singular] [json|{ "name": "world"}|]
		//       `shouldRespondWith` 200

		//   it "returns a single object for json proc" $
		//     request methodPost "/rpc/getproject"
		//         [singular] [json|{ "id": 1}|]
		//       `shouldRespondWith`
		//         [json|{"id":1,"name":"Windows 7","client_id":1}|]
		//         { matchHeaders = [matchContentTypeSingular] }

		//   it "fails for multiple rows" $
		//     request methodPost "/rpc/getallprojects"
		//             [singular] "{}"
		//       `shouldRespondWith`
		//             [json|{"details":"Results contain 5 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//             { matchStatus  = 406
		//             , matchHeaders = [matchContentTypeSingular]
		//             }

		//   it "fails for multiple rows with rolled back changes" $ do
		//     post "/rpc/getproject?select=id,name"
		//         [json| {"id": 1} |]
		//       `shouldRespondWith`
		//         [json|[{"id":1,"name":"Windows 7"}]|]

		//     request methodPost "/rpc/setprojects"
		//         [("Prefer", "tx=commit"), singular]
		//         [json| {"id_l": 1, "id_h": 2, "name": "changed"} |]
		//       `shouldRespondWith`
		//         [json|{"details":"Results contain 2 rows, application/vnd.pgrst.object+json requires 1 row","message":"JSON object requested, multiple (or no) rows returned","code":"PGRST116","hint":null}|]
		//         { matchStatus  = 406
		//         , matchHeaders = [ matchContentTypeSingular
		//                          , "Preference-Applied" <:> "tx=commit" ]
		//         }

		//     -- should rollback function
		//     post "/rpc/getproject?select=id,name"
		//         [json| {"id": 1} |]
		//       `shouldRespondWith`
		//         [json|[{"id":1,"name":"Windows 7"}]|]
	}

	test.Execute(t, testConfig, tests)
}
