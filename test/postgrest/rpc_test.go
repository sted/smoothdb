package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func TestPostgREST_RPC(t *testing.T) {

	tests := []test.Test{
		// context "a proc that returns a set" $ do
		//   it "returns paginated results" $ do
		//     request methodPost "/rpc/getitemrange"
		//             (rangeHdrs (ByteRangeFromTo 0 0))  [json| { "min": 2, "max": 4 } |]
		//        `shouldRespondWith` [json| [{"id":3}] |]
		//         { matchStatus = 200
		//         , matchHeaders = ["Content-Range" <:> "0-0/*"]
		//         }
		{
			Description:     "a proc that returns a set returns paginated results",
			Method:          "POST",
			Query:           "/rpc/getitemrange",
			Body:            `{ "min": 2, "max": 4 }`,
			Headers:         test.Headers{"Range": []string{"0-0"}},
			Expected:        `[ {"id": 3} ]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/*"},
			Status:          200,
		},
		//     request methodGet "/rpc/getitemrange?min=2&max=4"
		//             (rangeHdrs (ByteRangeFromTo 0 0)) ""
		//        `shouldRespondWith` [json| [{"id":3}] |]
		//         { matchStatus = 200
		//         , matchHeaders = ["Content-Range" <:> "0-0/*"]
		//         }
		{
			Description:     "a proc that returns a set returns paginated results",
			Method:          "GET",
			Query:           "/rpc/getitemrange?min=2&max=4",
			Body:            ``,
			Headers:         test.Headers{"Range": []string{"0-0"}},
			Expected:        `[ {"id": 3} ]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/*"},
			Status:          200,
		},
		//     request methodHead "/rpc/getitemrange?min=2&max=4"
		//         (rangeHdrs (ByteRangeFromTo 0 0)) ""
		//       `shouldRespondWith`
		//         ""
		//         { matchStatus = 200
		//         , matchHeaders = [ matchContentTypeJson
		//                          , "Content-Range" <:> "0-0/*" ]
		//         }
		{
			Description:     "a proc that returns a set returns paginated results",
			Method:          "HEAD",
			Query:           "/rpc/getitemrange?min=2&max=4",
			Body:            ``,
			Headers:         test.Headers{"Range": []string{"0-0"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/*"},
			Status:          200,
		},

		// it "using limit and offset" $ do
		// post "/rpc/getitemrange?limit=1&offset=1" [json| { "min": 2, "max": 4 } |]
		//    `shouldRespondWith` [json| [{"id":4}] |]
		// 	{ matchStatus = 200
		// 	, matchHeaders = ["Content-Range" <:> "1-1/*"]
		// 	}
		{
			Description:     "using limit and offset",
			Method:          "POST",
			Query:           "/rpc/getitemrange?limit=1&offset=1",
			Body:            `{ "min": 2, "max": 4 } `,
			Headers:         nil,
			Expected:        `[{"id":4}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "1-1/*"},
			Status:          200,
		},
		// get "/rpc/getitemrange?min=2&max=4&limit=1&offset=1"
		//    `shouldRespondWith` [json| [{"id":4}] |]
		// 	{ matchStatus = 200
		// 	, matchHeaders = ["Content-Range" <:> "1-1/*"]
		// 	}
		{
			Description:     "using limit and offset",
			Method:          "GET",
			Query:           "/rpc/getitemrange?min=2&max=4&limit=1&offset=1",
			Body:            ``,
			Headers:         nil,
			Expected:        `[{"id":4}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "1-1/*"},
			Status:          200,
		},
		// request methodHead "/rpc/getitemrange?min=2&max=4&limit=1&offset=1" mempty mempty
		//   `shouldRespondWith`
		// 	""
		// 	{ matchStatus = 200
		// 	, matchHeaders = [ matchContentTypeJson
		// 					 , "Content-Range" <:> "1-1/*" ]
		// 	}
		{
			Description:     "using limit and offset",
			Method:          "HEAD",
			Query:           "/rpc/getitemrange?min=2&max=4&limit=1&offset=1",
			Body:            ``,
			Headers:         nil,
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "1-1/*"},
			Status:          200,
		},
		//   it "includes total count if requested" $ do
		//     request methodPost "/rpc/getitemrange"
		//             (rangeHdrsWithCount (ByteRangeFromTo 0 0))
		//             [json| { "min": 2, "max": 4 } |]
		//        `shouldRespondWith` [json| [{"id":3}] |]
		//         { matchStatus = 206 -- it now knows the response is partial
		//         , matchHeaders = ["Content-Range" <:> "0-0/2"]
		//         }
		// @@ we will not support this (count with POST)
		// {
		// 	Description:     "includes total count if requested",
		// 	Method:          "POST",
		// 	Query:           "/rpc/getitemrange",
		// 	Body:            `{ "min": 2, "max": 4 }`,
		// 	Headers:         test.Headers{"Range": []string{"0-0"}, "Prefer": {"count=exact"}},
		// 	Expected:        ``,
		// 	ExpectedHeaders: map[string]string{"Content-Range": "0-0/2"},
		// 	Status:          206,
		// },
		//     request methodGet "/rpc/getitemrange?min=2&max=4"
		//             (rangeHdrsWithCount (ByteRangeFromTo 0 0)) ""
		//        `shouldRespondWith` [json| [{"id":3}] |]
		//         { matchStatus = 206
		//         , matchHeaders = ["Content-Range" <:> "0-0/2"]
		//         }
		{
			Description:     "includes total count if requested",
			Method:          "GET",
			Query:           "/rpc/getitemrange?min=2&max=4",
			Body:            ``,
			Headers:         test.Headers{"Range": {"0-0"}, "Prefer": {"count=exact"}},
			Expected:        `[{"id":3}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/2"},
			Status:          206,
		},
		//     request methodHead "/rpc/getitemrange?min=2&max=4"
		//         (rangeHdrsWithCount (ByteRangeFromTo 0 0)) ""
		//       `shouldRespondWith`
		//         ""
		//         { matchStatus = 206
		//         , matchHeaders = [ matchContentTypeJson
		//                          , "Content-Range" <:> "0-0/2" ]
		//         }
		{
			Description:     "includes total count if requested",
			Method:          "HEAD",
			Query:           "/rpc/getitemrange?min=2&max=4",
			Body:            ``,
			Headers:         test.Headers{"Range": {"0-0"}, "Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/2"},
			Status:          206,
		},
		//   it "includes exact count if requested" $ do
		//     request methodHead "/rpc/getallprojects"
		//             [("Prefer", "count=exact")] ""
		//        `shouldRespondWith` ""
		//         { matchStatus = 200
		//         , matchHeaders = ["Content-Range" <:> "0-4/5"]
		//         }
		{
			Description:     "includes exact count if requested",
			Method:          "HEAD",
			Query:           "/rpc/getallprojects",
			Body:            ``,
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-4/5"},
			Status:          200,
		},
		//     request methodHead "/rpc/getallprojects?select=*,clients!inner(*)&clients.id=eq.1"
		//             [("Prefer", "count=exact")] ""
		//        `shouldRespondWith` ""
		//         { matchStatus = 200
		//         , matchHeaders = ["Content-Range" <:> "0-1/2"]
		//         }
		{
			Description:     "includes exact count if requested",
			Method:          "HEAD",
			Query:           "/rpc/getallprojects?select=*,clients!inner(*)&clients.id=eq.1",
			Body:            ``,
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/2"},
			Status:          200,
		},
		// it "includes exact count of 1 for functions that return a single scalar, domain or composite" $ do
		// request methodGet "/rpc/add_them?a=3&b=4"
		//         [("Prefer", "count=exact")] ""
		//    `shouldRespondWith` "7"
		//     { matchStatus = 200
		//     , matchHeaders = ["Content-Range" <:> "0-0/1"]
		//     }
		{
			Description:     "includes exact count of 1 for functions that return a single scalar, domain or composite",
			Method:          "GET",
			Query:           "/rpc/add_them?a=3&b=4",
			Body:            ``,
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `7`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
			Status:          200,
		},
		// request methodGet "/rpc/ret_domain?val=8"
		//         [("Prefer", "count=exact")] ""
		//    `shouldRespondWith` "8"
		//     { matchStatus = 200
		//     , matchHeaders = ["Content-Range" <:> "0-0/1"]
		//     }
		{
			Description:     "includes exact count of 1 for functions that return a single scalar, domain or composite",
			Method:          "GET",
			Query:           "/rpc/ret_domain?val=8",
			Body:            ``,
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `8`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
			Status:          200,
		},
		// request methodGet "/rpc/ret_point_2d"
		//         [("Prefer", "count=exact")] ""
		//    `shouldRespondWith`
		//     [json|{"x": 10, "y": 5}|]
		//     { matchStatus = 200
		//     , matchHeaders = ["Content-Range" <:> "0-0/1"]
		//     }
		{
			Description:     "includes exact count of 1 for functions that return a single scalar, domain or composite",
			Method:          "GET",
			Query:           "/rpc/ret_point_2d",
			Body:            ``,
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `{"x": 10, "y": 5}`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
			Status:          200,
		},
		//   it "returns proper json" $ do
		//     post "/rpc/getitemrange" [json| { "min": 2, "max": 4 } |] `shouldRespondWith`
		//       [json| [ {"id": 3}, {"id":4} ] |]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns proper json",
			Method:      "POST",
			Query:       "/rpc/getitemrange",
			Body:        `{ "min": 2, "max": 4 }`,
			Headers:     nil,
			Expected:    `[ {"id": 3}, {"id":4} ]`,
			Status:      200,
		},
		//     get "/rpc/getitemrange?min=2&max=4" `shouldRespondWith`
		//       [json| [ {"id": 3}, {"id":4} ] |]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns proper json",
			Method:      "GET",
			Query:       "/rpc/getitemrange?min=2&max=4",
			Body:        ``,
			Headers:     nil,
			Expected:    `[ {"id": 3}, {"id":4} ]`,
			Status:      200,
		},
		//   it "returns CSV" $ do
		//     request methodPost "/rpc/getitemrange"
		//             (acceptHdrs "text/csv")
		//             [json| { "min": 2, "max": 4 } |]
		//        `shouldRespondWith` "id\n3\n4"
		//         { matchStatus = 200
		//         , matchHeaders = ["Content-Type" <:> "text/csv; charset=utf-8"]
		//         }
		{
			Description:     "returns CSV",
			Method:          "POST",
			Query:           "/rpc/getitemrange",
			Body:            `{ "min": 2, "max": 4 }`,
			Headers:         test.Headers{"Accept": []string{"text/csv"}},
			Expected:        "id\n3\n4",
			ExpectedHeaders: map[string]string{"Content-Type": "text/csv; charset=utf-8"},
			Status:          200,
		},
		//     request methodGet "/rpc/getitemrange?min=2&max=4"
		//             (acceptHdrs "text/csv") ""
		//        `shouldRespondWith` "id\n3\n4"
		//         { matchStatus = 200
		//         , matchHeaders = ["Content-Type" <:> "text/csv; charset=utf-8"]
		//         }
		{
			Description:     "returns CSV",
			Method:          "GET",
			Query:           "/rpc/getitemrange?min=2&max=4",
			Body:            ``,
			Headers:         test.Headers{"Accept": []string{"text/csv"}},
			Expected:        "id\n3\n4",
			ExpectedHeaders: map[string]string{"Content-Type": "text/csv; charset=utf-8"},
			Status:          200,
		},
		//     request methodHead "/rpc/getitemrange?min=2&max=4"
		//             (acceptHdrs "text/csv") ""
		//       `shouldRespondWith`
		//         ""
		//         { matchStatus = 200
		//         , matchHeaders = ["Content-Type" <:> "text/csv; charset=utf-8"]
		//         }

		// context "unknown function" $ do
		//   it "returns 404" $
		//     post "/rpc/fakefunc" [json| {} |] `shouldRespondWith` 404
		{
			Description: "unknown function returns 404",
			Method:      "POST",
			Query:       "/rpc/fakefunc",
			Body:        `{}`,
			Headers:     nil,
			Expected:    ``,
			Status:      404,
		},
		//   it "should fail with 404 on unknown proc name" $
		//     get "/rpc/fake" `shouldRespondWith` 404
		{
			Description: "should fail with 404 on unknown proc name",
			Method:      "GET",
			Query:       "/rpc/fake",
			Status:      404,
		},
		//   it "should fail with 404 and hint the closest proc on unknown proc name" $
		//     get "/rpc/sayhell" `shouldRespondWith`
		//       [json| {
		//         "hint":"Perhaps you meant to call the function test.sayhello",
		//         "message":"Could not find the function test.sayhell without parameters in the schema cache",
		//         "code":"PGRST202",
		//         "details":"Searched for the function test.sayhell without parameters, but no matches were found in the schema cache."} |]
		//       { matchStatus  = 404
		//       , matchHeaders = [matchContentTypeJson]
		//       }

		//   it "should fail with 404 on unknown proc args" $ do
		//     get "/rpc/sayhello" `shouldRespondWith` 404
		{
			Description: "should fail with 404 on unknown proc args",
			Method:      "GET",
			Query:       "/rpc/sayhello",
			Status:      404,
		},
		//     get "/rpc/sayhello?any_arg=value" `shouldRespondWith` 404
		{
			Description: "should fail with 404 on unknown proc args",
			Method:      "GET",
			Query:       "/rpc/sayhello?any_arg=value",
			Status:      404,
		},
		//   it "should fail with 404 and hint the closest args on unknown proc args" $
		//     get "/rpc/sayhello?nam=Peter" `shouldRespondWith`
		//       [json| {
		//         "hint":"Perhaps you meant to call the function test.sayhello(name)",
		//         "message":"Could not find the function test.sayhello(nam) in the schema cache",
		//         "code":"PGRST202",
		//         "details":"Searched for the function test.sayhello with parameter nam, but no matches were found in the schema cache."} |]
		//       { matchStatus  = 404
		//       , matchHeaders = [matchContentTypeJson]
		//       }
		{
			Description: "should fail with 404 and hint the closest args on unknown proc args",
			Method:      "GET",
			Query:       "/rpc/sayhello?any_arg=value",
			Status:      404,
		},
		//   it "should not ignore unknown args and fail with 404" $
		//     get "/rpc/add_them?a=1&b=2&smthelse=blabla" `shouldRespondWith`
		//     [json| {
		//       "hint":"Perhaps you meant to call the function test.add_them(a, b)",
		//       "message":"Could not find the function test.add_them(a, b, smthelse) in the schema cache",
		//       "code":"PGRST202",
		//       "details":"Searched for the function test.add_them with parameters a, b, smthelse, but no matches were found in the schema cache."} |]
		//     { matchStatus  = 404
		//     , matchHeaders = [matchContentTypeJson]
		//     }
		{
			Description: "should fail with 404 and hint the closest args on unknown proc args",
			Method:      "GET",
			Query:       "/rpc/add_them?a=1&b=2&smthelse=blabla",
			Status:      404,
		},
		//   it "should fail with 404 when no json arg is found with prefer single object" $
		//     request methodPost "/rpc/sayhello"
		//       [("Prefer","params=single-object")]
		//       [json|{}|]
		//     `shouldRespondWith`
		//       [json| {
		//         "hint":null,
		//         "message":"Could not find the function test.sayhello in the schema cache",
		//         "code":"PGRST202",
		//         "details":"Searched for the function test.sayhello with a single json/jsonb parameter, but no matches were found in the schema cache."} |]
		//     { matchStatus  = 404
		//     , matchHeaders = [matchContentTypeJson]
		//     }
		{
			Description: "should fail with 404 when no json arg is found with prefer single object",
			Method:      "GET",
			Query:       "/rpc/add_them?a=1&b=2&smthelse=blabla",
			Status:      404,
		},
		//   it "should fail with 404 for overloaded functions with unknown args" $ do
		//     get "/rpc/overloaded?wrong_arg=value" `shouldRespondWith`
		//       [json| {
		//         "hint":null,
		//         "message":"Could not find the function test.overloaded(wrong_arg) in the schema cache",
		//         "code":"PGRST202",
		//         "details":"Searched for the function test.overloaded with parameter wrong_arg, but no matches were found in the schema cache."} |]
		//       { matchStatus  = 404
		//       , matchHeaders = [matchContentTypeJson]
		//       }
		//     get "/rpc/overloaded?a=1&b=2&wrong_arg=value" `shouldRespondWith`
		//       [json| {
		//         "hint":"Perhaps you meant to call the function test.overloaded(a, b, c)",
		//         "message":"Could not find the function test.overloaded(a, b, wrong_arg) in the schema cache",
		//         "code":"PGRST202",
		//         "details":"Searched for the function test.overloaded with parameters a, b, wrong_arg, but no matches were found in the schema cache."} |]
		//       { matchStatus  = 404
		//       , matchHeaders = [matchContentTypeJson]
		//       }

		// context "ambiguous overloaded functions with same parameters' names but different types" $ do
		//   it "should fail with 300 Multiple Choices without explicit type casts" $
		//     get "/rpc/overloaded_same_args?arg=value" `shouldRespondWith`
		//       [json| {
		//         "hint":"Try renaming the parameters or the function itself in the database so function overloading can be resolved",
		//         "message":"Could not choose the best candidate function between: test.overloaded_same_args(arg => integer), test.overloaded_same_args(arg => xml), test.overloaded_same_args(arg => text, num => integer)",
		//         "code":"PGRST203",
		//         "details":null} |]
		//       { matchStatus  = 300
		//       , matchHeaders = [matchContentTypeJson]
		//       }

		// it "works when having uppercase identifiers" $ do
		//   get "/rpc/quotedFunction?user=mscott&fullName=Michael Scott&SSN=401-32-XXXX" `shouldRespondWith`
		//     [json|{"user": "mscott", "fullName": "Michael Scott", "SSN": "401-32-XXXX"}|]
		//     { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works when having uppercase identifiers",
			Method:      "GET",
			Query:       "/rpc/quotedFunction?user=mscott&fullName=Michael Scott&SSN=401-32-XXXX",
			Body:        ``,
			Headers:     nil,
			Expected:    `{"user": "mscott", "fullName": "Michael Scott", "SSN": "401-32-XXXX"}`,
			Status:      200,
		},
		//   post "/rpc/quotedFunction"
		//     [json|{"user": "dschrute", "fullName": "Dwight Schrute", "SSN": "030-18-XXXX"}|]
		//     `shouldRespondWith`
		//     [json|{"user": "dschrute", "fullName": "Dwight Schrute", "SSN": "030-18-XXXX"}|]
		//     { matchHeaders = [matchContentTypeJson] }
		{
			Description: "works when having uppercase identifiers",
			Method:      "POST",
			Query:       "/rpc/quotedFunction",
			Body:        `{"user": "dschrute", "fullName": "Dwight Schrute", "SSN": "030-18-XXXX"}`,
			Headers:     nil,
			Expected:    `{"user": "dschrute", "fullName": "Dwight Schrute", "SSN": "030-18-XXXX"}`,
			Status:      200,
		},
		// context "shaping the response returned by a proc" $ do
		//   it "returns a project" $ do
		//     post "/rpc/getproject" [json| { "id": 1} |] `shouldRespondWith`
		//       [json|[{"id":1,"name":"Windows 7","client_id":1}]|]
		{
			Description: "returns a project",
			Method:      "POST",
			Query:       "/rpc/getproject",
			Body:        `{"id": 1}`,
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Windows 7","client_id":1}]`,
			Status:      200,
		},
		//     get "/rpc/getproject?id=1" `shouldRespondWith`
		//       [json|[{"id":1,"name":"Windows 7","client_id":1}]|]
		{
			Description: "returns a project",
			Method:      "GET",
			Query:       "/rpc/getproject?id=1",
			Body:        ``,
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Windows 7","client_id":1}]`,
			Status:      200,
		},
		//   it "can filter proc results" $ do
		//     post "/rpc/getallprojects?id=gt.1&id=lt.5&select=id" [json| {} |] `shouldRespondWith`
		//       [json|[{"id":2},{"id":3},{"id":4}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter proc results",
			Method:      "POST",
			Query:       "/rpc/getallprojects?id=gt.1&id=lt.5&select=id",
			Body:        `{}`,
			Headers:     nil,
			Expected:    `[{"id":2},{"id":3},{"id":4}]`,
			Status:      200,
		},
		//     get "/rpc/getallprojects?id=gt.1&id=lt.5&select=id" `shouldRespondWith`
		//       [json|[{"id":2},{"id":3},{"id":4}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can filter proc results",
			Method:      "GET",
			Query:       "/rpc/getallprojects?id=gt.1&id=lt.5&select=id",
			Body:        ``,
			Headers:     nil,
			Expected:    `[{"id":2},{"id":3},{"id":4}]`,
			Status:      200,
		},
		//   it "can limit proc results" $ do
		//     post "/rpc/getallprojects?id=gt.1&id=lt.5&select=id&limit=2&offset=1" [json| {} |]
		//       `shouldRespondWith` [json|[{"id":3},{"id":4}]|]
		//          { matchStatus = 200
		//          , matchHeaders = ["Content-Range" <:> "1-2/*"] }
		{
			Description:     "can limit proc results",
			Method:          "POST",
			Query:           "/rpc/getallprojects?id=gt.1&id=lt.5&select=id&limit=2&offset=1",
			Body:            `{}`,
			Headers:         nil,
			Expected:        `[{"id":3},{"id":4}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "1-2/*"},
			Status:          200,
		},
		//     get "/rpc/getallprojects?id=gt.1&id=lt.5&select=id&limit=2&offset=1"
		//       `shouldRespondWith` [json|[{"id":3},{"id":4}]|]
		//          { matchStatus = 200
		//          , matchHeaders = ["Content-Range" <:> "1-2/*"] }
		{
			Description:     "can limit proc results",
			Method:          "GET",
			Query:           "/rpc/getallprojects?id=gt.1&id=lt.5&select=id&limit=2&offset=1",
			Body:            `{}`,
			Headers:         nil,
			Expected:        `[{"id":3},{"id":4}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "1-2/*"},
			Status:          200,
		},
		//   it "select works on the first level" $ do
		//     post "/rpc/getproject?select=id,name" [json| { "id": 1} |] `shouldRespondWith`
		//       [json|[{"id":1,"name":"Windows 7"}]|]
		{
			Description: "select works on the first level",
			Method:      "POST",
			Query:       "/rpc/getproject?select=id,name",
			Body:        `{"id": 1}`,
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Windows 7"}]`,
			Status:      200,
		},
		//     get "/rpc/getproject?id=1&select=id,name" `shouldRespondWith`
		//       [json|[{"id":1,"name":"Windows 7"}]|]
		{
			Description: "select works on the first level",
			Method:      "GET",
			Query:       "/rpc/getproject?id=1&select=id,name",
			Body:        ``,
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Windows 7"}]`,
			Status:      200,
		},
		// context "foreign entities embedding" $ do
		//   it "can embed if related tables are in the exposed schema" $ do
		//     post "/rpc/getproject?select=id,name,client:clients(id),tasks(id)" [json| { "id": 1} |] `shouldRespondWith`
		//       [json|[{"id":1,"name":"Windows 7","client":{"id":1},"tasks":[{"id":1},{"id":2}]}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can embed if related tables are in the exposed schema",
			Method:      "POST",
			Query:       "/rpc/getproject?select=id,name,client:clients(id),tasks(id)",
			Body:        `{"id": 1}`,
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Windows 7","client":{"id":1},"tasks":[{"id":1},{"id":2}]}]`,
			Status:      200,
		},
		//     get "/rpc/getproject?id=1&select=id,name,client:clients(id),tasks(id)" `shouldRespondWith`
		//       [json|[{"id":1,"name":"Windows 7","client":{"id":1},"tasks":[{"id":1},{"id":2}]}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can embed if related tables are in the exposed schema",
			Method:      "GET",
			Query:       "/rpc/getproject?id=1&select=id,name,client:clients(id),tasks(id)",
			Body:        `{"id": 1}`,
			Headers:     nil,
			Expected:    `[{"id":1,"name":"Windows 7","client":{"id":1},"tasks":[{"id":1},{"id":2}]}]`,
			Status:      200,
		},
		//   it "cannot embed if the related table is not in the exposed schema" $ do
		//     post "/rpc/single_article?select=*,article_stars(*)" [json|{ "id": 1}|]
		//       `shouldRespondWith` 400
		{
			Description: "cannot embed if the related table is not in the exposed schema",
			Method:      "POST",
			Query:       "/rpc/single_article?select=*,article_stars(*)",
			Body:        `{"id": 1}`,
			Headers:     nil,
			Expected:    ``,
			Status:      400,
		},
		//     get "/rpc/single_article?id=1&select=*,article_stars(*)"
		//       `shouldRespondWith` 400
		{
			Description: "cannot embed if the related table is not in the exposed schema",
			Method:      "GET",
			Query:       "/rpc/single_article?id=1&select=*,article_stars(*)",
			Body:        `{"id": 1}`,
			Headers:     nil,
			Expected:    ``,
			Status:      400,
		},
		//   it "can embed if the related tables are in a hidden schema but exposed as views" $ do
		//     post "/rpc/single_article?select=id,articleStars(userId)"
		//         [json|{ "id": 2}|]
		//       `shouldRespondWith`
		//         [json|{"id": 2, "articleStars": [{"userId": 3}]}|]
		//     get "/rpc/single_article?id=2&select=id,articleStars(userId)"
		//       `shouldRespondWith`
		//         [json|{"id": 2, "articleStars": [{"userId": 3}]}|]

		//   it "can embed an M2M relationship table" $ do
		//     get "/rpc/getallusers?select=name,tasks(name)&id=gt.1"
		//       `shouldRespondWith` [json|[
		//         {"name":"Michael Scott", "tasks":[{"name":"Design IOS"}, {"name":"Code IOS"}, {"name":"Design OSX"}]},
		//         {"name":"Dwight Schrute","tasks":[{"name":"Design w7"}, {"name":"Design IOS"}]}
		//       ]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can embed an M2M relationship table",
			Method:      "GET",
			Query:       "/rpc/getallusers?select=name,tasks(name)&id=gt.1",
			Expected: `[
				{"name":"Michael Scott", "tasks":[{"name":"Design IOS"}, {"name":"Code IOS"}, {"name":"Design OSX"}]},
				{"name":"Dwight Schrute","tasks":[{"name":"Design w7"}, {"name":"Design IOS"}]}]`,
			Status: 200,
		},
		//     -- https://github.com/PostgREST/postgrest/issues/2565
		//     get "/rpc/get_yards?select=groups(*)"
		//       `shouldRespondWith` [json|[]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can embed an M2M relationship table",
			Method:      "GET",
			Query:       "/rpc/get_yards?select=groups(*)",
			Expected:    `[]`,
			Status:      200,
		},
		//   it "can embed an M2M relationship table that has a parent relationship table" $
		//     get "/rpc/getallusers?select=name,tasks(name,project:projects(name))&id=gt.1"
		//       `shouldRespondWith` [json|[
		//         {"name":"Michael Scott","tasks":[
		//           {"name":"Design IOS","project":{"name":"IOS"}},
		//           {"name":"Code IOS","project":{"name":"IOS"}},
		//           {"name":"Design OSX","project":{"name":"OSX"}}
		//         ]},
		//         {"name":"Dwight Schrute","tasks":[
		//           {"name":"Design w7","project":{"name":"Windows 7"}},
		//           {"name":"Design IOS","project":{"name":"IOS"}}
		//         ]}
		//       ]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can embed an M2M relationship table that has a parent relationship table",
			Method:      "GET",
			Query:       "/rpc/getallusers?select=name,tasks(name,project:projects(name))&id=gt.1",
			Body:        ``,
			Headers:     nil,
			Expected: `[
				        {"name":"Michael Scott","tasks":[
				          {"name":"Design IOS","project":{"name":"IOS"}},
				          {"name":"Code IOS","project":{"name":"IOS"}},
				          {"name":"Design OSX","project":{"name":"OSX"}}
				        ]},
				        {"name":"Dwight Schrute","tasks":[
				          {"name":"Design w7","project":{"name":"Windows 7"}},
				          {"name":"Design IOS","project":{"name":"IOS"}}
				        ]}
				      ]`,
			Status: 200,
		},
		//   it "can embed an O2O relationship" $ do
		//     get "/rpc/allcapitals?select=name,country(name)"
		//       `shouldRespondWith` [json|[
		//         {"name":"Kabul","country":{"name":"Afghanistan"}},
		//         {"name":"Algiers","country":{"name":"Algeria"}}]
		//       |]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can embed an O2O relationship",
			Method:      "GET",
			Query:       "/rpc/allcapitals?select=name,country(name)",
			Expected: `[
				        {"name":"Kabul","country":{"name":"Afghanistan"}},
				        {"name":"Algiers","country":{"name":"Algeria"}}]`,
			Status: 200,
		},
		//     get "/rpc/allcountries?select=name,capital(name)"
		//       `shouldRespondWith` [json|[
		//         {"name":"Afghanistan","capital":{"name":"Kabul"}},
		//         {"name":"Algeria","capital":{"name":"Algiers"}}
		//       ]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can embed an O2O relationship",
			Method:      "GET",
			Query:       "/rpc/allcountries?select=name,capital(name)",
			Expected: `[
				        {"name":"Afghanistan","capital":{"name":"Kabul"}},
				        {"name":"Algeria","capital":{"name":"Algiers"}}]`,
			Status: 200,
		},
		//   -- https://github.com/PostgREST/postgrest/pull/2677#issuecomment-1444976849
		//   when (actualPgVersion >= pgVersion130) $
		//     it "can embed if rpc returns domain of table type" $ do
		//       post "/rpc/getproject_domain?select=id,name,client:clients(id),tasks(id)"
		//           [json| { "id": 1} |]
		//         `shouldRespondWith`
		//           [json|[{"id":1,"name":"Windows 7","client":{"id":1},"tasks":[{"id":1},{"id":2}]}]|]
		{
			Description: "can embed if rpc returns domain of table type",
			Method:      "POST",
			Query:       "/rpc/getproject_domain?select=id,name,client:clients(id),tasks(id)",
			Body:        `{ "id": 1}`,
			Expected:    `[{"id":1,"name":"Windows 7","client":{"id":1},"tasks":[{"id":1},{"id":2}]}]`,
			Status:      200,
		},
		//       get "/rpc/getproject_domain?id=1&select=id,name,client:clients(id),tasks(id)"
		//         `shouldRespondWith`
		//           [json|[{"id":1,"name":"Windows 7","client":{"id":1},"tasks":[{"id":1},{"id":2}]}]|]
		{
			Description: "can embed if rpc returns domain of table type",
			Method:      "GET",
			Query:       "/rpc/getproject_domain?id=1&select=id,name,client:clients(id),tasks(id)",
			Body:        ``,
			Expected:    `[{"id":1,"name":"Windows 7","client":{"id":1},"tasks":[{"id":1},{"id":2}]}]`,
			Status:      200,
		},
		// context "a proc that returns an empty rowset" $
		//   it "returns empty json array" $ do
		//     post "/rpc/test_empty_rowset" [json| {} |] `shouldRespondWith`
		//       [json| [] |]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns empty json array",
			Method:      "POST",
			Query:       "/rpc/test_empty_rowset",
			Body:        `{}`,
			Expected:    `[]`,
			Status:      200,
		},
		//     get "/rpc/test_empty_rowset" `shouldRespondWith`
		//       [json| [] |]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns empty json array",
			Method:      "GET",
			Query:       "/rpc/test_empty_rowset",
			Expected:    `[]`,
			Status:      200,
		},
		// context "proc return types" $ do
		//   context "returns text" $ do
		//     it "returns proper json" $
		//       post "/rpc/sayhello" [json| { "name": "world" } |] `shouldRespondWith`
		//         [json|"Hello, world"|]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns proper json",
			Method:      "POST",
			Query:       "/rpc/sayhello",
			Body:        `{ "name": "world" }`,
			Headers:     nil,
			Expected:    `"Hello, world"`,
			Status:      200,
		},
		//     it "can handle unicode" $
		//       post "/rpc/sayhello" [json| { "name": "￥" } |] `shouldRespondWith`
		//         [json|"Hello, ￥"|]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "can handle unicode",
			Method:      "POST",
			Query:       "/rpc/sayhello",
			Body:        `{ "name": "￥" }`,
			Headers:     nil,
			Expected:    `"Hello, ￥"`,
			Status:      200,
		},
		//   it "returns array" $
		//     post "/rpc/ret_array" [json|{}|] `shouldRespondWith`
		//       [json|[1, 2, 3]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns array",
			Method:      "POST",
			Query:       "/rpc/ret_array",
			Body:        `{}`,
			Headers:     nil,
			Expected:    `[1, 2, 3]`,
			Status:      200,
		},
		//   it "returns setof integers" $
		//     post "/rpc/ret_setof_integers"
		//         [json|{}|]
		//       `shouldRespondWith`
		//         [json|[1,2,3]|]
		{
			Description: "returns setof integers",
			Method:      "POST",
			Query:       "/rpc/ret_setof_integers",
			Body:        `{}`,
			Headers:     nil,
			Expected:    `[1, 2, 3]`,
			Status:      200,
		},
		//   it "returns enum value" $
		//     post "/rpc/ret_enum" [json|{ "val": "foo" }|] `shouldRespondWith`
		//       [json|"foo"|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns enum value",
			Method:      "POST",
			Query:       "/rpc/ret_enum",
			Body:        `{ "val": "foo" }`,
			Headers:     nil,
			Expected:    `"foo"`,
			Status:      200,
		},
		//   it "returns domain value" $
		//     post "/rpc/ret_domain" [json|{ "val": "8" }|] `shouldRespondWith`
		//       [json|8|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns domain value",
			Method:      "POST",
			Query:       "/rpc/ret_domain",
			Body:        `{ "val": "8" }`,
			Headers:     nil,
			Expected:    `8`,
			Status:      200,
		},
		//   it "returns range" $
		//     post "/rpc/ret_range" [json|{ "low": 10, "up": 20 }|] `shouldRespondWith`
		//       [json|"[10,20)"|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns range",
			Method:      "POST",
			Query:       "/rpc/ret_range",
			Body:        `{ "low": 10, "up": 20 }`,
			Headers:     nil,
			Expected:    `"[10,20)"`,
			Status:      200,
		},
		//   it "returns row of scalars" $
		//     post "/rpc/ret_scalars" [json|{}|] `shouldRespondWith`
		//       [json|[{"a":"scalars", "b":"foo", "c":1, "d":"[10,20)"}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns row of scalars",
			Method:      "POST",
			Query:       "/rpc/ret_scalars",
			Body:        `{}`,
			Headers:     nil,
			Expected:    `[{"a":"scalars", "b":"foo", "c":1, "d":"[10,20)"}]`,
			Status:      200,
		},
		//   it "returns composite type in exposed schema" $
		//     post "/rpc/ret_point_2d"
		//         [json|{}|]
		//       `shouldRespondWith`
		//         [json|{"x": 10, "y": 5}|]
		{
			Description: "returns composite type in exposed schema",
			Method:      "POST",
			Query:       "/rpc/ret_point_2d",
			Body:        `{}`,
			Headers:     nil,
			Expected:    `{"x": 10, "y": 5}`,
			Status:      200,
		},
		//   it "cannot return composite type in hidden schema" $
		//     post "/rpc/ret_point_3d" [json|{}|] `shouldRespondWith` 401
		{
			Description: "cannot return composite type in hidden schema",
			Method:      "POST",
			Query:       "/rpc/ret_point_3d",
			Body:        `{}`,
			Headers:     nil,
			Expected:    ``,
			Status:      401,
		},
		//   when (actualPgVersion >= pgVersion110) $
		//     it "returns domain of composite type" $
		//       post "/rpc/ret_composite_domain"
		//           [json|{}|]
		//         `shouldRespondWith`
		//           [json|{"x": 10, "y": 5}|]
		{
			Description: "returns domain of composite type",
			Method:      "POST",
			Query:       "/rpc/ret_composite_domain",
			Body:        `{}`,
			Headers:     nil,
			Expected:    `{"x": 10, "y": 5}`,
			Status:      200,
		},
		//   it "returns single row from table" $
		//     post "/rpc/single_article?select=id"
		//         [json|{"id": 2}|]
		//       `shouldRespondWith`
		//         [json|{"id": 2}|]
		{
			Description: "returns single row from table",
			Method:      "POST",
			Query:       "/rpc/single_article?select=id",
			Body:        `{"id": 2}`,
			Headers:     nil,
			Expected:    `{"id": 2}`,
			Status:      200,
		},
		//   it "returns 204, no Content-Type header and no content for void" $
		//     post "/rpc/ret_void"
		//         [json|{}|]
		//       `shouldRespondWith`
		//         ""
		//         { matchStatus = 204
		//         , matchHeaders = [matchHeaderAbsent hContentType]
		//         }

		//   it "returns null for an integer with null value" $
		//     post "/rpc/ret_null"
		//         [json|{}|]
		//       `shouldRespondWith`
		//         [json|null|]
		{
			Description: "returns null for an integer with null value",
			Method:      "POST",
			Query:       "/rpc/ret_null",
			Body:        `{}`,
			Headers:     nil,
			Expected:    `null`,
			Status:      200,
		},
		//   context "different types when overloaded" $ do
		//     it "returns composite type" $
		//       post "/rpc/ret_point_overloaded"
		//           [json|{"x": 1, "y": 2}|]
		//         `shouldRespondWith`
		//           [json|{"x": 1, "y": 2}|]
		// {
		// 	Description: "different types when overloaded",
		// 	Method:      "POST",
		// 	Query:       "/rpc/ret_point_overloaded",
		// 	Body:        `{"x": 1, "y": 2}`,
		// 	Headers:     nil,
		// 	Expected:    `{"x": 1, "y": 2}`,
		// 	Status:      200,
		// },
		//     it "returns json scalar with prefer single object" $
		//       request methodPost "/rpc/ret_point_overloaded" [("Prefer","params=single-object")]
		//         [json|{"x": 1, "y": 2}|]
		//         `shouldRespondWith`
		//         [json|{"x": 1, "y": 2}|]
		//         { matchHeaders = [matchContentTypeJson] }

		//   when (actualPgVersion >= pgVersion100) $
		//     it "accepts a variety of arguments (Postgres >= 10)" $
		//       post "/rpc/varied_arguments"
		//           [json| {
		//             "double": 3.1,
		//             "varchar": "hello",
		//             "boolean": true,
		//             "date": "20190101",
		//             "money": 0,
		//             "enum": "foo",
		//             "arr": ["a", "b", "c"],
		//             "integer": 43,
		//             "json": {"some key": "some value"},
		//             "jsonb": {"another key": [1, 2, "3"]}
		//           } |]
		//         `shouldRespondWith`
		//           [json| {
		//             "double": 3.1,
		//             "varchar": "hello",
		//             "boolean": true,
		//             "date": "2019-01-01",
		//             "money": "$0.00",
		//             "enum": "foo",
		//             "arr": ["a", "b", "c"],
		//             "integer": 43,
		//             "json": {"some key": "some value"},
		//             "jsonb": {"another key": [1, 2, "3"]}
		//           } |]
		//           { matchHeaders = [matchContentTypeJson] }
		{
			Description: "accepts a variety of arguments",
			Method:      "POST",
			Query:       "/rpc/varied_arguments",
			Body: `{
						"double": 3.1,
						"varchar": "hello",
						"boolean": true,
						"date": "20190101",
						"money": 0,
						"enum": "foo",
						"arr": ["a", "b", "c"],
						"integer": 43,
						"json": {"some key": "some value"},
						"jsonb": {"another key": [1, 2, "3"]}
						}`,
			Headers: nil,
			Expected: `{
				"double": 3.1,
				"varchar": "hello",
				"boolean": true,
				"date": "2019-01-01",
				"money": "$0.00",
				"enum": "foo",
				"arr": ["a", "b", "c"],
				"integer": 43,
				"json": {"some key": "some value"},
				"jsonb": {"another key": [1, 2, "3"]}
			  }`,
			Status: 200,
		},
		//   it "accepts a variety of arguments with GET" $
		//     -- without JSON / JSONB here, because passing those via query string is useless - they just become a "json string" all the time
		//     get "/rpc/varied_arguments?double=3.1&varchar=hello&boolean=true&date=20190101&money=0&enum=foo&arr=%7Ba,b,c%7D&integer=43"
		//       `shouldRespondWith`
		//           [json| {
		//             "double": 3.1,
		//             "varchar": "hello",
		//             "boolean": true,
		//             "date": "2019-01-01",
		//             "money": "$0.00",
		//             "enum": "foo",
		//             "arr": ["a", "b", "c"],
		//             "integer": 43,
		//             "json": {},
		//             "jsonb": {}
		//           } |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "accepts a variety of arguments with GET",
			Method:      "GET",
			Query:       "/rpc/varied_arguments?double=3.1&varchar=hello&boolean=true&date=20190101&money=0&enum=foo&arr=%7Ba,b,c%7D&integer=43",
			Body:        ``,
			Headers:     nil,
			Expected: `{
				"double": 3.1,
				"varchar": "hello",
				"boolean": true,
				"date": "2019-01-01",
				"money": "$0.00",
				"enum": "foo",
				"arr": ["a", "b", "c"],
				"integer": 43,
				"json": {},
				"jsonb": {}
			  }`,
			Status: 200,
		},
		//   it "accepts a variety of arguments from an html form" $
		//     request methodPost "/rpc/varied_arguments"
		//         [("Content-Type", "application/x-www-form-urlencoded")]
		//         "double=3.1&varchar=hello&boolean=true&date=20190101&money=0&enum=foo&arr=%7Ba,b,c%7D&integer=43"
		//       `shouldRespondWith`
		//           [json| {
		//             "double": 3.1,
		//             "varchar": "hello",
		//             "boolean": true,
		//             "date": "2019-01-01",
		//             "money": "$0.00",
		//             "enum": "foo",
		//             "arr": ["a", "b", "c"],
		//             "integer": 43,
		//             "json": {},
		//             "jsonb": {}
		//           } |]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "accepts a variety of arguments from an html form",
			Method:      "POST",
			Query:       "/rpc/varied_arguments",
			Body:        `double=3.1&varchar=hello&boolean=true&date=20190101&money=0&enum=foo&arr=%7Ba,b,c%7D&integer=43`,
			Headers:     test.Headers{"Content-Type": []string{"application/x-www-form-urlencoded"}},
			Expected: `{
				"double": 3.1,
				"varchar": "hello",
				"boolean": true,
				"date": "2019-01-01",
				"money": "$0.00",
				"enum": "foo",
				"arr": ["a", "b", "c"],
				"integer": 43,
				"json": {},
				"jsonb": {}
			  }`,
			Status: 200,
		},
		//   it "parses embedded JSON arguments as JSON" $
		//     post "/rpc/json_argument"
		//         [json| { "arg": { "key": 3 } } |]
		//       `shouldRespondWith`
		//         [json|"object"|]
		//         { matchHeaders = [matchContentTypeJson] }
		{
			Description: "parses embedded JSON arguments as JSON",
			Method:      "POST",
			Query:       "/rpc/json_argument",
			Body:        `{ "arg": { "key": 3 } }`,
			Headers:     nil,
			Expected:    `"object"`,
			Status:      200,
		},
		//   when (actualPgVersion < pgVersion100) $
		//     it "parses quoted JSON arguments as JSON (Postgres < 10)" $
		//       post "/rpc/json_argument"
		//           [json| { "arg": "{ \"key\": 3 }" } |]
		//         `shouldRespondWith`
		//           [json|"object"|]
		//           { matchHeaders = [matchContentTypeJson] }

		//   when ((actualPgVersion >= pgVersion109 && actualPgVersion < pgVersion110)
		//         || actualPgVersion >= pgVersion114) $
		//     it "parses quoted JSON arguments as JSON string (from Postgres 10.9, 11.4)" $
		//       post "/rpc/json_argument"
		//           [json| { "arg": "{ \"key\": 3 }" } |]
		//         `shouldRespondWith`
		//           [json|"string"|]
		//           { matchHeaders = [matchContentTypeJson] }
		//@@ returns "object"
		// {
		// 	Description: "parses quoted JSON arguments as JSON string (from Postgres 10.9, 11.4)",
		// 	Method:      "POST",
		// 	Query:       "/rpc/json_argument",
		// 	Body:        `{ "arg": "{ \"key\": 3 }" }`,
		// 	Headers:     nil,
		// 	Expected:    `"string"`,
		// 	Status:      200,
		// },
		// context "improper input" $ do
		//   it "rejects unknown content type even if payload is good" $ do
		//     request methodPost "/rpc/sayhello"
		//       (acceptHdrs "audio/mpeg3") [json| { "name": "world" } |]
		//         `shouldRespondWith` 4o6
		{
			Description: "rejects unknown content type even if payload is good",
			Method:      "POST",
			Query:       "/rpc/sayhello",
			Body:        `{ "name": "world" }`,
			Headers:     test.Headers{"Accept": []string{"audio/mpeg3"}},
			Expected:    ``,
			Status:      406,
		},
		//     request methodGet "/rpc/sayhello?name=world"
		//       (acceptHdrs "audio/mpeg3") ""
		//         `shouldRespondWith` 406
		{
			Description: "rejects unknown content type even if payload is good",
			Method:      "GET",
			Query:       "/rpc/sayhello?name=world",
			Body:        ``,
			Headers:     test.Headers{"Accept": []string{"audio/mpeg3"}},
			Expected:    ``,
			Status:      406,
		},
		//   it "rejects malformed json payload" $ do
		//     p <- request methodPost "/rpc/sayhello"
		//       (acceptHdrs "application/json") "sdfsdf"
		//     liftIO $ do
		//       simpleStatus p `shouldBe` badRequest400
		//       isErrorFormat (simpleBody p) `shouldBe` True
		{
			Description: "rejects malformed json payload",
			Method:      "POST",
			Query:       "/rpc/sayhello",
			Body:        `sdfsdf`,
			Headers:     test.Headers{"Accept": []string{"application/json"}},
			Expected:    ``,
			Status:      400,
		},
		//   it "treats simple plpgsql raise as invalid input" $ do
		//     p <- post "/rpc/problem" "{}"
		//     liftIO $ do
		//       simpleStatus p `shouldBe` badRequest400
		//       isErrorFormat (simpleBody p) `shouldBe` True
		//   it "treats plpgsql assert as internal server error" $ do
		//     p <- post "/rpc/assert" "{}"
		//     liftIO $ do
		//       simpleStatus p `shouldBe` internalServerError500
		//       isErrorFormat (simpleBody p) `shouldBe` True

		// context "unsupported method" $ do
		//   it "DELETE fails" $
		//     request methodDelete "/rpc/sayhello" [] ""
		//       `shouldRespondWith`
		//       [json|{"message":"Cannot use the DELETE method on RPC","code":"PGRST101","details":null,"hint":null}|]
		//       { matchStatus  = 405
		//       , matchHeaders = [matchContentTypeJson]
		//       }
		//   it "PATCH fails" $
		//     request methodPatch "/rpc/sayhello" [] ""
		//       `shouldRespondWith` 405

		// it "executes the proc exactly once per request" $ do
		//   -- callcounter is persistent even with rollback, because it uses a sequence
		//   -- reset counter first to make test repeatable
		//   request methodPost "/rpc/reset_sequence"
		//       [("Prefer", "tx=commit")]
		//       [json|{"name": "callcounter_count", "value": 1}|]
		//     `shouldRespondWith`
		//       ""
		//       { matchStatus = 204
		//       , matchHeaders = [ matchHeaderAbsent hContentType ]
		//       }

		//   -- now the test
		//   post "/rpc/callcounter"
		//       [json|{}|]
		//     `shouldRespondWith`
		//       [json|1|]

		//   post "/rpc/callcounter"
		//       [json|{}|]
		//     `shouldRespondWith`
		//       [json|2|]

		// context "a proc that receives no parameters" $ do
		//   it "interprets empty string as empty json object on a post request" $
		//     post "/rpc/noparamsproc" BL.empty `shouldRespondWith`
		//       [json| "Return value of no parameters procedure." |]
		//       { matchHeaders = [matchContentTypeJson] }
		//   it "interprets empty string as a function with no args on a get request" $
		//     get "/rpc/noparamsproc" `shouldRespondWith`
		//       [json| "Return value of no parameters procedure." |]
		//       { matchHeaders = [matchContentTypeJson] }

		// it "returns proper output when having the same return col name as the proc name" $ do
		//   post "/rpc/test" [json|{}|] `shouldRespondWith`
		//     [json|[{"test":"hello","value":1}]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns proper output when having the same return col name as the proc name",
			Method:      "POST",
			Query:       "/rpc/test",
			Body:        `{}`,
			Expected:    `[{"test":"hello","value":1}]`,
			Status:      200,
		},
		//   get "/rpc/test" `shouldRespondWith`
		//     [json|[{"test":"hello","value":1}]|] { matchHeaders = [matchContentTypeJson] }
		{
			Description: "returns proper output when having the same return col name as the proc name",
			Method:      "GET",
			Query:       "/rpc/test",
			Expected:    `[{"test":"hello","value":1}]`,
			Status:      200,
		},
		// context "procs with OUT/INOUT params" $ do
		//   it "returns an object result when there is a single OUT param" $ do
		//     get "/rpc/single_out_param?num=5"
		//       `shouldRespondWith`
		//         [json|{"num_plus_one":6}|]
		{
			Description: "returns an object result when there  is a single OUT params",
			Method:      "GET",
			Query:       "/rpc/single_out_param?num=5",
			Expected:    `{"num_plus_one":6}`,
			Status:      200,
		},
		//     get "/rpc/single_json_out_param?a=1&b=two"
		//       `shouldRespondWith`
		//         [json|{"my_json": {"a": 1, "b": "two"}}|]
		{
			Description: "returns an object result when there  is a single OUT params",
			Method:      "GET",
			Query:       "/rpc/single_json_out_param?a=1&b=two",
			Expected:    `{"my_json": {"a": 1, "b": "two"}}`,
			Status:      200,
		},
		//   it "returns an object result when there is a single INOUT param" $
		//     get "/rpc/single_inout_param?num=2"
		//       `shouldRespondWith`
		//         [json|{"num":3}|]
		{
			Description: "returns an object result when there is a single INOUT params",
			Method:      "GET",
			Query:       "/rpc/single_inout_param?num=2",
			Expected:    `{"num":3}`,
			Status:      200,
		},
		//   it "returns an object result when there are many OUT params" $
		//     get "/rpc/many_out_params"
		//       `shouldRespondWith`
		//         [json|{"my_json":{"a": 1, "b": "two"},"num":3,"str":"four"}|]
		{
			Description: "returns an object result when there are many OUT params",
			Method:      "GET",
			Query:       "/rpc/many_out_params",
			Expected:    `{"my_json":{"a": 1, "b": "two"},"num":3,"str":"four"}`,
			Status:      200,
		},
		//   it "returns an object result when there are many INOUT params" $
		//     get "/rpc/many_inout_params?num=1&str=two&b=false"
		//       `shouldRespondWith`
		//         [json|{"num":1,"str":"two","b":false}|]
		{
			Description: "returns an object result when there are many INOUT params",
			Method:      "GET",
			Query:       "/rpc/many_inout_params?num=1&str=two&b=false",
			Expected:    `{"num":1,"str":"two","b":false}`,
			Status:      200,
		},
		// context "procs with TABLE return" $ do
		//   it "returns an object result when there is a single-column TABLE return type" $
		//     get "/rpc/single_column_table_return"
		//       `shouldRespondWith`
		//         [json|[{"a": "A"}]|]
		{
			Description: "returns an object result when there is a single-column TABLE return type",
			Method:      "GET",
			Query:       "/rpc/single_column_table_return",
			Expected:    `[{"a": "A"}]`,
			Status:      200,
		},
		//   it "returns an object result when there is a multi-column TABLE return type" $
		//     get "/rpc/multi_column_table_return"
		//       `shouldRespondWith`
		//         [json|[{"a": "A", "b": "B"}]|]
		{
			Description: "returns an object result when there is a multi-column TABLE return type",
			Method:      "GET",
			Query:       "/rpc/multi_column_table_return",
			Expected:    `[{"a": "A", "b": "B"}]`,
			Status:      200,
		},
		// context "procs with VARIADIC params" $ do
		//   when (actualPgVersion < pgVersion100) $
		//     it "works with POST (Postgres < 10)" $
		//       post "/rpc/variadic_param"
		//           [json| { "v": "{hi,hello,there}" } |]
		//         `shouldRespondWith`
		//           [json|["hi", "hello", "there"]|]

		//   when (actualPgVersion >= pgVersion100) $ do
		//     it "works with POST (Postgres >= 10)" $
		//       post "/rpc/variadic_param"
		//           [json| { "v": ["hi", "hello", "there"] } |]
		//         `shouldRespondWith`
		//           [json|["hi", "hello", "there"]|]

		//     context "works with GET and repeated params" $ do
		//       it "n=0 (through DEFAULT)" $
		//         get "/rpc/variadic_param"
		//           `shouldRespondWith`
		//             [json|[]|]
		{
			Description: "works with GET and repeated params",
			Method:      "GET",
			Query:       "/rpc/variadic_param",
			Expected:    `[]`,
			Status:      200,
		},
		//       it "n=1" $
		//         get "/rpc/variadic_param?v=hi"
		//           `shouldRespondWith`
		//             [json|["hi"]|]
		// @@ support for variadic is minimal (should not use the sintax "v := ...")
		// {
		// 	Description: "works with GET and repeated params",
		// 	Method:      "GET",
		// 	Query:       "/rpc/variadic_param?v=hi",
		// 	Expected:    `["hi"]`,
		// 	Status:      200,
		// },
		//       it "n>1" $
		//         get "/rpc/variadic_param?v=hi&v=there"
		//           `shouldRespondWith`
		//             [json|["hi", "there"]|]

		//     context "works with POST and repeated params from html form" $ do
		//       it "n=0 (through DEFAULT)" $
		//         request methodPost "/rpc/variadic_param"
		//             [("Content-Type", "application/x-www-form-urlencoded")]
		//             ""
		//           `shouldRespondWith`
		//             [json|[]|]

		//       it "n=1" $
		//         request methodPost "/rpc/variadic_param"
		//             [("Content-Type", "application/x-www-form-urlencoded")]
		//             "v=hi"
		//           `shouldRespondWith`
		//             [json|["hi"]|]

		//       it "n>1" $
		//         request methodPost "/rpc/variadic_param"
		//             [("Content-Type", "application/x-www-form-urlencoded")]
		//             "v=hi&v=there"
		//           `shouldRespondWith`
		//             [json|["hi", "there"]|]

		// it "returns last value for repeated params without VARIADIC" $
		//   get "/rpc/sayhello?name=ignored&name=world"
		//     `shouldRespondWith`
		//       [json|"Hello, world"|]
		{
			Description: "returns last value for repeated params without VARIADIC",
			Method:      "GET",
			Query:       "/rpc/sayhello?name=ignored&name=world",
			Expected:    `"Hello, world"`,
			Status:      200,
		},
		// when (actualPgVersion >= pgVersion100) $
		//   it "returns last value for repeated non-variadic params in function with other VARIADIC arguments" $
		//     get "/rpc/sayhello_variadic?name=ignored&name=world&v=unused"
		//       `shouldRespondWith`
		//         [json|"Hello, world"|]

		// it "can handle procs with args that have a DEFAULT value" $ do
		//   get "/rpc/many_inout_params?num=1&str=two"
		//     `shouldRespondWith`
		//       [json| {"num":1,"str":"two","b":true}|]
		{
			Description: "can handle procs with args that have a DEFAULT value",
			Method:      "GET",
			Query:       "/rpc/many_inout_params?num=1&str=two",
			Expected:    `{"num":1,"str":"two","b":true}`,
			Status:      200,
		},
		//   get "/rpc/three_defaults?b=4"
		//     `shouldRespondWith`
		//       [json|8|]
		{
			Description: "can handle procs with args that have a DEFAULT value",
			Method:      "GET",
			Query:       "/rpc/three_defaults?b=4",
			Expected:    `8`,
			Status:      200,
		},
		// it "can map a RAISE error code and message to a http status" $
		//   get "/rpc/raise_pt402"
		//     `shouldRespondWith` [json|{ "hint": "Upgrade your plan", "details": "Quota exceeded", "code": "PT402", "message": "Payment Required" }|]
		//     { matchStatus  = 402
		//     , matchHeaders = [matchContentTypeJson]
		//     }

		// it "defaults to status 500 if RAISE code is PT not followed by a number" $
		//   get "/rpc/raise_bad_pt"
		//     `shouldRespondWith`
		//     [json|{"hint": null, "details": null, "code": "PT40A", "message": "Wrong"}|]
		//     { matchStatus  = 500
		//     , matchHeaders = [ matchContentTypeJson ]
		//     }

		// context "expects a single json object" $ do
		//   it "does not expand posted json into parameters" $
		//     request methodPost "/rpc/singlejsonparam"
		//       [("prefer","params=single-object")] [json| { "p1": 1, "p2": "text", "p3" : {"obj":"text"} } |] `shouldRespondWith`
		//       [json| { "p1": 1, "p2": "text", "p3" : {"obj":"text"} } |]
		//       { matchHeaders = [matchContentTypeJson] }

		//   it "accepts parameters from an html form" $
		//     request methodPost "/rpc/singlejsonparam"
		//       [("Prefer","params=single-object"),("Content-Type", "application/x-www-form-urlencoded")]
		//       ("integer=7&double=2.71828&varchar=forms+are+fun&" <>
		//        "boolean=false&date=1900-01-01&money=$3.99&enum=foo") `shouldRespondWith`
		//       [json| { "integer": "7", "double": "2.71828", "varchar" : "forms are fun"
		//              , "boolean":"false", "date":"1900-01-01", "money":"$3.99", "enum":"foo" } |]
		//              { matchHeaders = [matchContentTypeJson] }

		//   it "works with GET" $
		//     request methodGet "/rpc/singlejsonparam?p1=1&p2=text" [("Prefer","params=single-object")] ""
		//       `shouldRespondWith` [json|{ "p1": "1", "p2": "text"}|]
		//       { matchHeaders = [matchContentTypeJson] }

		// context "should work with an overloaded function" $ do
		//   it "overloaded()" $
		//     get "/rpc/overloaded"
		//       `shouldRespondWith`
		//         [json|[1,2,3]|]

		//   it "overloaded(json) single-object" $
		//     request methodPost "/rpc/overloaded"
		//         [("Prefer","params=single-object")]
		//         [json|[{"x": 1, "y": "first"}, {"x": 2, "y": "second"}]|]
		//       `shouldRespondWith`
		//         [json|[{"x": 1, "y": "first"}, {"x": 2, "y": "second"}]|]

		//   it "overloaded(int, int)" $
		//     get "/rpc/overloaded?a=1&b=2" `shouldRespondWith` [str|3|]

		//   it "overloaded(text, text, text)" $
		//     get "/rpc/overloaded?a=1&b=2&c=3" `shouldRespondWith` [json|"123"|]

		//   it "overloaded_html_form()" $
		//     request methodPost "/rpc/overloaded_html_form"
		//         [("Content-Type", "application/x-www-form-urlencoded")]
		//         ""
		//       `shouldRespondWith`
		//         [json|[1,2,3]|]

		//   it "overloaded_html_form(json) single-object" $
		//     request methodPost "/rpc/overloaded_html_form"
		//         [("Content-Type", "application/x-www-form-urlencoded"), ("Prefer","params=single-object")]
		//         "a=1&b=2&c=3"
		//       `shouldRespondWith`
		//         [json|{"a": "1", "b": "2", "c": "3"}|]

		//   it "overloaded_html_form(int, int)" $
		//     request methodPost "/rpc/overloaded_html_form"
		//         [("Content-Type", "application/x-www-form-urlencoded")]
		//         "a=1&b=2"
		//       `shouldRespondWith`
		//         [str|3|]

		//   it "overloaded_html_form(text, text, text)" $
		//     request methodPost "/rpc/overloaded_html_form"
		//         [("Content-Type", "application/x-www-form-urlencoded")]
		//         "a=1&b=2&c=3"
		//       `shouldRespondWith`
		//         [json|"123"|]

		//   -- https://github.com/PostgREST/postgrest/issues/1672
		//   context "embedding overloaded functions with the same signature except for the last param with a default value" $ do
		//     it "overloaded_default(text default)" $ do
		//       request methodPost "/rpc/overloaded_default?select=id,name,users(name)"
		//           [("Content-Type", "application/json")]
		//           [json|{}|]
		//         `shouldRespondWith`
		//           [json|[{"id": 2, "name": "Code w7", "users": [{"name": "Angela Martin"}]}] |]

		//     it "overloaded_default(int)" $
		//       request methodPost "/rpc/overloaded_default"
		//           [("Content-Type", "application/json")]
		//           [json|{"must_param":1}|]
		//         `shouldRespondWith`
		//           [json|{"val":1}|]

		//     it "overloaded_default(int, text default)" $ do
		//       request methodPost "/rpc/overloaded_default?select=id,name,users(name)"
		//           [("Content-Type", "application/json")]
		//           [json|{"a":4}|]
		//         `shouldRespondWith`
		//           [json|[{"id": 5, "name": "Design IOS", "users": [{"name": "Michael Scott"}, {"name": "Dwight Schrute"}]}] |]

		//     it "overloaded_default(int, int)" $
		//       request methodPost "/rpc/overloaded_default"
		//           [("Content-Type", "application/json")]
		//           [json|{"a":2,"must_param":4}|]
		//         `shouldRespondWith`
		//           [json|{"a":2,"val":4}|]

		// context "only for POST rpc" $ do
		//   it "gives a parse filter error if GET style proc args are specified" $
		//     post "/rpc/sayhello?name=John" [json|{name: "John"}|] `shouldRespondWith` 400

		//   it "ignores json keys not included in ?columns" $
		//     post "/rpc/sayhello?columns=name"
		//       [json|{"name": "John", "smth": "here", "other": "stuff", "fake_id": 13}|]
		//       `shouldRespondWith`
		//       [json|"Hello, John"|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "ignores json keys not included in ?columns",
			Method:      "POST",
			Query:       "/rpc/sayhello?columns=name",
			Body:        `{"name": "John", "smth": "here", "other": "stuff", "fake_id": 13}`,
			Expected:    `"Hello, John"`,
			Status:      200,
		},
		//   it "only takes the first object in case of array of objects payload" $
		//     post "/rpc/add_them"
		//       [json|[
		//         {"a": 1, "b": 2},
		//         {"a": 4, "b": 6},
		//         {"a": 100, "b": 200} ]|]
		//       `shouldRespondWith` "3"
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "only takes the first object in case of array of objects payload",
			Method:      "POST",
			Query:       "/rpc/add_them",
			Body: `[
				        {"a": 1, "b": 2},
				        {"a": 4, "b": 6},
				        {"a": 100, "b": 200} ]`,
			Expected: `3`,
			Status:   200,
		},
		// context "bulk RPC with params=multiple-objects" $ do
		//   it "works with a scalar function an returns a json array" $
		//     request methodPost "/rpc/add_them" [("Prefer", "params=multiple-objects")]
		//       [json|[
		//         {"a": 1, "b": 2},
		//         {"a": 4, "b": 6},
		//         {"a": 100, "b": 200} ]|]
		//       `shouldRespondWith`
		//       [json|
		//         [3, 10, 300]
		//       |] { matchHeaders = [matchContentTypeJson] }

		//   it "works with a scalar function an returns a json array when posting CSV" $
		//     request methodPost "/rpc/add_them" [("Content-Type", "text/csv"), ("Prefer", "params=multiple-objects")]
		//       "a,b\n1,2\n4,6\n100,200"
		//       `shouldRespondWith`
		//       [json|
		//         [3, 10, 300]
		//       |]
		//       { matchStatus  = 200
		//       , matchHeaders = [matchContentTypeJson]
		//       }

		//   it "works with a non-scalar result" $
		//     request methodPost "/rpc/get_projects_below?select=id,name" [("Prefer", "params=multiple-objects")]
		//       [json|[
		//         {"id": 1},
		//         {"id": 5} ]|]
		//       `shouldRespondWith`
		//       [json|
		//         [{"id":1,"name":"Windows 7"},
		//          {"id":2,"name":"Windows 10"},
		//          {"id":3,"name":"IOS"},
		//          {"id":4,"name":"OSX"}]
		//       |] { matchHeaders = [matchContentTypeJson] }

		// context "HTTP request env vars" $ do
		//   it "custom header is set" $
		//     request methodPost "/rpc/get_guc_value"
		//               [("Custom-Header", "test")]
		//         (
		//         if actualPgVersion >= pgVersion140 then
		//           [json| { "prefix": "request.headers", "name": "custom-header" } |]
		//         else
		//           [json| { "name": "request.header.custom-header" } |]
		//         )
		//         `shouldRespondWith`
		//         [json|"test"|]
		//         { matchStatus  = 200
		//         , matchHeaders = [ matchContentTypeJson ]
		//         }
		//   it "standard header is set" $
		//     request methodPost "/rpc/get_guc_value"
		//               [("Origin", "http://example.com")]
		//         (
		//         if actualPgVersion >= pgVersion140 then
		//           [json| { "prefix": "request.headers", "name": "origin" } |]
		//         else
		//           [json| { "name": "request.header.origin" } |]
		//         )
		//         `shouldRespondWith`
		//         [json|"http://example.com"|]
		//         { matchStatus  = 200
		//         , matchHeaders = [ matchContentTypeJson ]
		//         }
		//   it "current role is available as GUC claim" $
		//     request methodPost "/rpc/get_guc_value" []
		//         (
		//         if actualPgVersion >= pgVersion140 then
		//           [json| { "prefix": "request.jwt.claims", "name": "role" } |]
		//         else
		//           [json| { "name": "request.jwt.claim.role" } |]
		//         )
		//         `shouldRespondWith`
		//         [json|"postgrest_test_anonymous"|]
		//         { matchStatus  = 200
		//         , matchHeaders = [ matchContentTypeJson ]
		//         }
		{
			Description: "current role is available as GUC claim",
			Method:      "POST",
			Query:       "/rpc/get_guc_value",
			Body: `{ "prefix": "request.jwt.claims", "name": "role" }
			`,
			Expected: `"postgrest_test_anonymous"`,
			Status:   200,
		},
		//   it "single cookie ends up as claims" $
		//     request methodPost "/rpc/get_guc_value" [("Cookie","acookie=cookievalue")]
		//       (
		//       if actualPgVersion >= pgVersion140 then
		//         [json| {"prefix": "request.cookies", "name":"acookie"} |]
		//       else
		//         [json| {"name":"request.cookie.acookie"} |]
		//       )
		//         `shouldRespondWith`
		//         [json|"cookievalue"|]
		//         { matchStatus = 200
		//         , matchHeaders = []
		//         }
		//   it "multiple cookies ends up as claims" $
		//     request methodPost "/rpc/get_guc_value" [("Cookie","acookie=cookievalue;secondcookie=anothervalue")]
		//       (
		//       if actualPgVersion >= pgVersion140 then
		//         [json| {"prefix": "request.cookies", "name":"secondcookie"} |]
		//       else
		//         [json| {"name":"request.cookie.secondcookie"} |]
		//       )
		//         `shouldRespondWith`
		//         [json|"anothervalue"|]
		//         { matchStatus = 200
		//         , matchHeaders = []
		//         }
		//   it "app settings available" $
		//     request methodPost "/rpc/get_guc_value" []
		//       [json| { "name": "app.settings.app_host" } |]
		//         `shouldRespondWith`
		//         [json|"localhost"|]
		//         { matchStatus  = 200
		//         , matchHeaders = [ matchContentTypeJson ]
		//         }
		//   it "gets the Authorization value" $
		//     request methodPost "/rpc/get_guc_value" [authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIn0.Xod-F15qsGL0WhdOCr2j3DdKuTw9QJERVgoFD3vGaWA"]
		//       (
		//       if actualPgVersion >= pgVersion140 then
		//         [json| {"prefix": "request.headers", "name":"authorization"} |]
		//       else
		//         [json| {"name":"request.header.authorization"} |]
		//       )
		//         `shouldRespondWith`
		//         [json|"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIn0.Xod-F15qsGL0WhdOCr2j3DdKuTw9QJERVgoFD3vGaWA"|]
		//         { matchStatus = 200
		//         , matchHeaders = []
		//         }
		//   it "gets the http method" $
		//     request methodPost "/rpc/get_guc_value" []
		//       [json| {"name":"request.method"} |]
		//         `shouldRespondWith`
		//         [json|"POST"|]
		//         { matchStatus = 200
		//         , matchHeaders = []
		//         }
		//   it "gets the http path" $
		//     request methodPost "/rpc/get_guc_value" []
		//       [json| {"name":"request.path"} |]
		//         `shouldRespondWith`
		//         [json|"/rpc/get_guc_value"|]
		//         { matchStatus = 200
		//         , matchHeaders = []
		//         }

		// context "binary output" $ do
		//   context "Proc that returns scalar" $ do
		//     it "can query without selecting column" $
		//       request methodPost "/rpc/ret_base64_bin" (acceptHdrs "application/octet-stream") ""
		//         `shouldRespondWith` "iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeAQMAAAAB/jzhAAAABlBMVEUAAAD/AAAb/40iAAAAP0lEQVQI12NgwAbYG2AE/wEYwQMiZB4ACQkQYZEAIgqAhAGIKLCAEQ8kgMT/P1CCEUwc4IMSzA3sUIIdCHECAGSQEkeOTUyCAAAAAElFTkSuQmCC"
		//         { matchStatus = 200
		//         , matchHeaders = ["Content-Type" <:> "application/octet-stream"]
		//         }
		{
			Description:     "binary output: Proc that returns scalar can query without selecting column",
			Method:          "GET", // @@ POST -> GET
			Query:           "/rpc/ret_image",
			Headers:         test.Headers{"Accept": []string{"application/octet-stream"}},
			Expected:        `iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeAQMAAAAB/jzhAAAABlBMVEUAAAD/AAAb/40iAAAAP0lEQVQI12NgwAbYG2AE/wEYwQMiZB4ACQkQYZEAIgqAhAGIKLCAEQ8kgMT/P1CCEUwc4IMSzA3sUIIdCHECAGSQEkeOTUyCAAAAAElFTkSuQmCC`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/octet-stream"},
			Status:          200,
		},
		//     it "can get raw output with Accept: text/plain" $
		//       request methodGet "/rpc/welcome" (acceptHdrs "text/plain") ""
		//         `shouldRespondWith` "Welcome to PostgREST"
		//         { matchStatus = 200
		//         , matchHeaders = ["Content-Type" <:> "text/plain; charset=utf-8"]
		//         }

		//     it "can get raw xml output with Accept: text/xml" $
		//       request methodGet "/rpc/return_scalar_xml" (acceptHdrs "text/xml") ""
		//         `shouldRespondWith`
		//         "<my-xml-tag/>"
		//         { matchStatus = 200
		//         , matchHeaders = ["Content-Type" <:> "text/xml; charset=utf-8"]
		//         }

		//     it "can get raw xml output with Accept: text/xml" $
		//       request methodGet "/rpc/welcome.xml" (acceptHdrs "text/xml") ""
		//         `shouldRespondWith`
		//         "<html>\n  <head>\n    <title>PostgREST</title>\n  </head>\n  <body>\n    <h1>Welcome to PostgREST</h1>\n  </body>\n</html>"
		//         { matchStatus = 200
		//         , matchHeaders = ["Content-Type" <:> "text/xml; charset=utf-8"]
		//         }

		//     it "should fail with function returning text and Accept: text/xml" $
		//       request methodGet "/rpc/welcome" (acceptHdrs "text/xml") ""
		//         `shouldRespondWith`
		//         [json|
		//           {
		//             "hint":"No function matches the given name and argument types. You might need to add explicit type casts.",
		//             "details":null,
		//             "code":"42883",
		//             "message":"function xmlagg(text) does not exist"
		//           }
		//         |]
		//         { matchStatus = 406
		//         , matchHeaders = ["Content-Type" <:> "application/json; charset=utf-8"]
		//         }

		//   context "Proc that returns set of scalars" $
		//     it "can query without selecting column" $
		//       request methodGet "/rpc/welcome_twice"
		//           (acceptHdrs "text/plain")
		//           ""
		//         `shouldRespondWith`
		//           "Welcome to PostgRESTWelcome to PostgREST"
		//           { matchStatus = 200
		//           , matchHeaders = ["Content-Type" <:> "text/plain; charset=utf-8"]
		//           }

		//   context "Proc that returns rows" $ do
		//     it "can query if a single column is selected" $
		//       request methodPost "/rpc/ret_rows_with_base64_bin?select=img" (acceptHdrs "application/octet-stream") ""
		//         `shouldRespondWith` "iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeAQMAAAAB/jzhAAAABlBMVEUAAAD/AAAb/40iAAAAP0lEQVQI12NgwAbYG2AE/wEYwQMiZB4ACQkQYZEAIgqAhAGIKLCAEQ8kgMT/P1CCEUwc4IMSzA3sUIIdCHECAGSQEkeOTUyCAAAAAElFTkSuQmCCiVBORw0KGgoAAAANSUhEUgAAAB4AAAAeAQMAAAAB/jzhAAAABlBMVEX///8AAP94wDzzAAAAL0lEQVQIW2NgwAb+HwARH0DEDyDxwAZEyGAhLODqHmBRzAcn5GAS///A1IF14AAA5/Adbiiz/0gAAAAASUVORK5CYII="
		//         { matchStatus = 200
		//         , matchHeaders = ["Content-Type" <:> "application/octet-stream"]
		//         }
		{
			Description:     "binary output: Proc that returns scalar can query if a single column is selected",
			Method:          "GET", // @@ POST -> GET
			Query:           "/rpc/ret_rows_with_base64_bin?select=img",
			Headers:         test.Headers{"Accept": []string{"application/octet-stream"}},
			Expected:        `iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeAQMAAAAB/jzhAAAABlBMVEUAAAD/AAAb/40iAAAAP0lEQVQI12NgwAbYG2AE/wEYwQMiZB4ACQkQYZEAIgqAhAGIKLCAEQ8kgMT/P1CCEUwc4IMSzA3sUIIdCHECAGSQEkeOTUyCAAAAAElFTkSuQmCCiVBORw0KGgoAAAANSUhEUgAAAB4AAAAeAQMAAAAB/jzhAAAABlBMVEX///8AAP94wDzzAAAAL0lEQVQIW2NgwAb+HwARH0DEDyDxwAZEyGAhLODqHmBRzAcn5GAS///A1IF14AAA5/Adbiiz/0gAAAAASUVORK5CYII=`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/octet-stream"},
			Status:          200,
		},
		//     it "fails if a single column is not selected" $
		//       request methodPost "/rpc/ret_rows_with_base64_bin"
		//           (acceptHdrs "application/octet-stream") ""
		//         `shouldRespondWith`
		//           [json| {"message":"application/octet-stream requested but more than one column was selected","code":"PGRST113","details":null,"hint":null} |]
		//           { matchStatus = 406 }
		{
			Description:     "binary output: Proc that returns scalar fails if a single column is not selected",
			Method:          "GET", // @@ POST -> GET
			Query:           "/rpc/ret_rows_with_base64_bin",
			Headers:         test.Headers{"Accept": []string{"application/octet-stream"}},
			Expected:        ``,
			ExpectedHeaders: nil,
			Status:          406,
		},
		// context "only for GET rpc" $ do
		//   it "should fail on mutating procs" $ do
		//     get "/rpc/callcounter" `shouldRespondWith` 405
		//     get "/rpc/setprojects?id_l=1&id_h=5&name=FreeBSD" `shouldRespondWith` 405

		//   it "should filter a proc that has arg name = filter name" $
		//     get "/rpc/get_projects_below?id=5&id=gt.2&select=id" `shouldRespondWith`
		//       [json|[{ "id": 3 }, { "id": 4 }]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "should filter a proc that has arg name = filter name",
			Method:      "GET",
			Query:       "/rpc/get_projects_below?id=5&id=gt.2&select=id",
			Expected:    `[{ "id": 3 }, { "id": 4 }]`,
			Status:      200,
		},
		//   it "should work with filters that have the not operator" $ do
		//     get "/rpc/get_projects_below?id=5&id=not.gt.2&select=id" `shouldRespondWith`
		//       [json|[{ "id": 1 }, { "id": 2 }]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "should work with filters that have the not operator",
			Method:      "GET",
			Query:       "/rpc/get_projects_below?id=5&id=not.gt.2&select=id",
			Expected:    `[{ "id": 1 }, { "id": 2 }]`,
			Status:      200,
		},
		//     get "/rpc/get_projects_below?id=5&id=not.in.(1,3)&select=id" `shouldRespondWith`
		//       [json|[{ "id": 2 }, { "id": 4 }]|]
		//       { matchHeaders = [matchContentTypeJson] }
		{
			Description: "should work with filters that have the not operator",
			Method:      "GET",
			Query:       "/rpc/get_projects_below?id=5&id=not.in.(1,3)&select=id",
			Expected:    `[{ "id": 2 }, { "id": 4 }]`,
			Status:      200,
		},
		//   it "should work with filters that use the plain with language fts operator" $ do
		//     get "/rpc/get_tsearch?text_search_vector=fts(english).impossible" `shouldRespondWith`
		//       [json|[{"text_search_vector":"'fun':5 'imposs':9 'kind':3"}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		// @@
		// No function matches the given name and argument types. You might need to add explicit type casts.",
		// "Message":"function test.get_tsearch(text_search_vector =\u003e unknown) does not exist
		// {
		// 	Description: "should work with filters that use the plain with language fts operator",
		// 	Method:      "GET",
		// 	Query:       "/rpc/get_tsearch?text_search_vector=fts(english).impossible",
		// 	Expected:    `[{"text_search_vector":"'fun':5 'imposs':9 'kind':3"}]`,
		// 	Status:      200,
		// },
		//     get "/rpc/get_tsearch?text_search_vector=plfts.impossible" `shouldRespondWith`
		//       [json|[{"text_search_vector":"'fun':5 'imposs':9 'kind':3"}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		//     get "/rpc/get_tsearch?text_search_vector=not.fts(english).fun%7Crat" `shouldRespondWith`
		//       [json|[{"text_search_vector":"'amus':5 'fair':7 'impossibl':9 'peu':4"},{"text_search_vector":"'art':4 'spass':5 'unmog':7"}]|]
		//       { matchHeaders = [matchContentTypeJson] }
		//     when (actualPgVersion >= pgVersion112) $
		//         get "/rpc/get_tsearch?text_search_vector=wfts.impossible" `shouldRespondWith`
		//             [json|[{"text_search_vector":"'fun':5 'imposs':9 'kind':3"}]|]
		//             { matchHeaders = [matchContentTypeJson] }

		//   it "should work with the phraseto_tsquery function" $
		//     get "/rpc/get_tsearch?text_search_vector=phfts(english).impossible" `shouldRespondWith`
		//       [json|[{"text_search_vector":"'fun':5 'imposs':9 'kind':3"}]|]
		//       { matchHeaders = [matchContentTypeJson] }

		// it "should work with an argument of custom type in public schema" $
		//     get "/rpc/test_arg?my_arg=something" `shouldRespondWith`
		//       [json|"foobar"|]
		//       { matchHeaders = [matchContentTypeJson] }

		// context "GUC headers on function calls" $ do
		//   it "succeeds setting the headers" $ do
		//     get "/rpc/get_projects_and_guc_headers?id=eq.2&select=id"
		//       `shouldRespondWith` [json|[{"id": 2}]|]
		//       {matchHeaders = [
		//           matchContentTypeJson,
		//           "X-Test"   <:> "key1=val1; someValue; key2=val2",
		//           "X-Test-2" <:> "key1=val1"]}
		//     get "/rpc/get_int_and_guc_headers?num=1"
		//       `shouldRespondWith` [json|1|]
		//       {matchHeaders = [
		//           matchContentTypeJson,
		//           "X-Test"   <:> "key1=val1; someValue; key2=val2",
		//           "X-Test-2" <:> "key1=val1"]}
		//     post "/rpc/get_int_and_guc_headers" [json|{"num": 1}|]
		//       `shouldRespondWith` [json|1|]
		//       {matchHeaders = [
		//           matchContentTypeJson,
		//           "X-Test"   <:> "key1=val1; someValue; key2=val2",
		//           "X-Test-2" <:> "key1=val1"]}

		//   it "fails when setting headers with wrong json structure" $ do
		//     get "/rpc/bad_guc_headers_1"
		//       `shouldRespondWith`
		//       [json|{"message":"response.headers guc must be a JSON array composed of objects with a single key and a string value","code":"PGRST111","details":null,"hint":null}|]
		//       { matchStatus  = 500
		//       , matchHeaders = [ matchContentTypeJson ]
		//       }
		//     get "/rpc/bad_guc_headers_2"
		//       `shouldRespondWith`
		//       [json|{"message":"response.headers guc must be a JSON array composed of objects with a single key and a string value","code":"PGRST111","details":null,"hint":null}|]
		//       { matchStatus  = 500
		//       , matchHeaders = [ matchContentTypeJson ]
		//       }
		//     get "/rpc/bad_guc_headers_3"
		//       `shouldRespondWith`
		//       [json|{"message":"response.headers guc must be a JSON array composed of objects with a single key and a string value","code":"PGRST111","details":null,"hint":null}|]
		//       { matchStatus  = 500
		//       , matchHeaders = [ matchContentTypeJson ]
		//       }
		//     post "/rpc/bad_guc_headers_1" [json|{}|]
		//       `shouldRespondWith`
		//       [json|{"message":"response.headers guc must be a JSON array composed of objects with a single key and a string value","code":"PGRST111","details":null,"hint":null}|]
		//       { matchStatus  = 500
		//       , matchHeaders = [ matchContentTypeJson ]
		//       }

		//   it "can set the same http header twice" $
		//     get "/rpc/set_cookie_twice"
		//       `shouldRespondWith`
		//         ""
		//         { matchStatus = 204
		//         , matchHeaders = [ matchHeaderAbsent hContentType
		//                          , "Set-Cookie" <:> "sessionid=38afes7a8; HttpOnly; Path=/"
		//                          , "Set-Cookie" <:> "id=a3fWa; Expires=Wed, 21 Oct 2015 07:28:00 GMT; Secure; HttpOnly" ]
		//         }

		//   it "can override the Location header on a trigger" $
		//     post "/stuff"
		//         [json|[{"id": 2, "name": "stuff 2"}]|]
		//       `shouldRespondWith`
		//         ""
		//         { matchStatus = 201
		//         , matchHeaders = [ matchHeaderAbsent hContentType
		//                          , "Location" <:> "/stuff?id=eq.2&overriden=true" ]
		//         }

		//   -- On https://github.com/PostgREST/postgrest/issues/1427#issuecomment-595907535
		//   -- it was reported that blank headers ` : ` where added and that cause proxies to fail the requests.
		//   -- These tests are to ensure no blank headers are added.
		//   context "Blank headers bug" $ do
		//     it "shouldn't add blank headers on POST" $ do
		//       r <- request methodPost "/loc_test" [] [json|{"id": "1", "c": "c1"}|]
		//       liftIO $ do
		//         let respHeaders = simpleHeaders r
		//         respHeaders `shouldSatisfy` noBlankHeader

		//     it "shouldn't add blank headers on PATCH" $ do
		//       r <- request methodPatch "/loc_test?id=eq.1" [] [json|{"c": "c2"}|]
		//       liftIO $ do
		//         let respHeaders = simpleHeaders r
		//         respHeaders `shouldSatisfy` noBlankHeader

		//     it "shouldn't add blank headers on GET" $ do
		//       r <- request methodGet "/loc_test" [] ""
		//       liftIO $ do
		//         let respHeaders = simpleHeaders r
		//         respHeaders `shouldSatisfy` noBlankHeader

		//     it "shouldn't add blank headers on DELETE" $ do
		//       r <- request methodDelete "/loc_test?id=eq.1" [] ""
		//       liftIO $ do
		//         let respHeaders = simpleHeaders r
		//         respHeaders `shouldSatisfy` noBlankHeader

		//   context "GUC status override" $ do
		//     it "can override the status on RPC" $
		//       get "/rpc/send_body_status_403"
		//         `shouldRespondWith`
		//         [json|{"message" : "invalid user or password"}|]
		//         { matchStatus  = 403
		//         , matchHeaders = [ matchContentTypeJson ]
		//         }

		//     it "can override the status through trigger" $
		//       patch "/stuff?id=eq.1"
		//           [json|[{"name": "updated stuff 1"}]|]
		//         `shouldRespondWith`
		//           205

		//     it "fails when setting invalid status guc" $
		//       get "/rpc/send_bad_status"
		//         `shouldRespondWith`
		//         [json|{"message":"response.status guc must be a valid status code","code":"PGRST112","details":null,"hint":null}|]
		//         { matchStatus  = 500
		//         , matchHeaders = [ matchContentTypeJson ]
		//         }

		//   context "single unnamed param" $ do
		//     it "can insert json directly" $
		//       post "/rpc/unnamed_json_param"
		//           [json|{"A": 1, "B": 2, "C": 3}|]
		//         `shouldRespondWith`
		//           [json|{"A": 1, "B": 2, "C": 3}|]

		//     it "can insert text directly" $
		//       request methodPost "/rpc/unnamed_text_param"
		//         [("Content-Type", "text/plain"), ("Accept", "text/plain")]
		//         [str|unnamed text arg|]
		//         `shouldRespondWith`
		//         [str|unnamed text arg|]

		//     it "can insert xml directly" $
		//       request methodPost "/rpc/unnamed_xml_param"
		//         [("Content-Type", "text/xml"), ("Accept", "text/xml")]
		//         [str|<note><from>John</from><to>Jane</to><message>Remember me</message></note>|]
		//         `shouldRespondWith`
		//         [str|<note><from>John</from><to>Jane</to><message>Remember me</message></note>|]

		//     it "can insert bytea directly" $ do
		//       let file = unsafePerformIO $ BL.readFile "test/spec/fixtures/image.png"
		//       r <- request methodPost "/rpc/unnamed_bytea_param"
		//         [("Content-Type", "application/octet-stream"), ("Accept", "application/octet-stream")]
		//         file
		//       liftIO $ do
		//         let respBody = simpleBody r
		//         respBody `shouldBe` file
		{
			Description:     "can insert bytea directly",
			Method:          "POST",
			Query:           "/rpc/unnamed_bytea_param",
			Body:            `iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeAQMAAAAB/jzhAAAABlBMVEUAAAD/AAAb/40iAAAAP0lEQVQI12NgwAbYG2AE/wEYwQMiZB4ACQkQYZEAIgqAhAGIKLCAEQ8kgMT/P1CCEUwc4IMSzA3sUIIdCHECAGSQEkeOTUyCAAAAAElFTkSuQmCC`,
			Headers:         test.Headers{"Accept": []string{"application/octet-stream"}, "Content-Type": []string{"application/octet-stream"}},
			Expected:        `iVBORw0KGgoAAAANSUhEUgAAAB4AAAAeAQMAAAAB/jzhAAAABlBMVEUAAAD/AAAb/40iAAAAP0lEQVQI12NgwAbYG2AE/wEYwQMiZB4ACQkQYZEAIgqAhAGIKLCAEQ8kgMT/P1CCEUwc4IMSzA3sUIIdCHECAGSQEkeOTUyCAAAAAElFTkSuQmCC`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/octet-stream"},
			Status:          200,
		},
		//     it "will err when no function with single unnamed json parameter exists and application/json is specified" $
		//       request methodPost "/rpc/unnamed_int_param" [("Content-Type", "application/json")]
		//           [json|{"x": 1, "y": 2}|]
		//         `shouldRespondWith`
		//           [json|{
		//             "hint": "Perhaps you meant to call the function test.unnamed_text_param",
		//             "message": "Could not find the function test.unnamed_int_param(x, y) in the schema cache",
		//             "code":"PGRST202",
		//             "details":"Searched for the function test.unnamed_int_param with parameters x, y or with a single unnamed json/jsonb parameter, but no matches were found in the schema cache."
		//           }|]
		//           { matchStatus  = 404
		//           , matchHeaders = [ matchContentTypeJson ]
		//           }

		//     it "will err when no function with single unnamed text parameter exists and text/plain is specified" $
		//       request methodPost "/rpc/unnamed_int_param"
		//           [("Content-Type", "text/plain")]
		//           [str|a simple text|]
		//         `shouldRespondWith`
		//           [json|{
		//             "hint": null,
		//             "message": "Could not find the function test.unnamed_int_param in the schema cache",
		//             "code":"PGRST202",
		//             "details":"Searched for the function test.unnamed_int_param with a single unnamed text parameter, but no matches were found in the schema cache."
		//           }|]
		//           { matchStatus  = 404
		//           , matchHeaders = [ matchContentTypeJson ]
		//           }

		//     it "will err when no function with single unnamed text parameter exists and text/xml is specified" $
		//       request methodPost "/rpc/unnamed_int_param"
		//           [("Content-Type", "text/xml")]
		//           [str|a simple text|]
		//         `shouldRespondWith`
		//           [json|{
		//             "hint": null,
		//             "message": "Could not find the function test.unnamed_int_param in the schema cache",
		//             "code":"PGRST202",
		//             "details":"Searched for the function test.unnamed_int_param with a single unnamed xml parameter, but no matches were found in the schema cache."
		//           }|]
		//           { matchStatus  = 404
		//           , matchHeaders = [ matchContentTypeJson ]
		//           }

		//     it "will err when no function with single unnamed bytea parameter exists and application/octet-stream is specified" $
		//       let file = unsafePerformIO $ BL.readFile "test/spec/fixtures/image.png" in
		//       request methodPost "/rpc/unnamed_int_param"
		//           [("Content-Type", "application/octet-stream")]
		//           file
		//       `shouldRespondWith`
		//         [json|{
		//           "hint": null,
		//           "message": "Could not find the function test.unnamed_int_param in the schema cache",
		//           "code":"PGRST202",
		//           "details":"Searched for the function test.unnamed_int_param with a single unnamed bytea parameter, but no matches were found in the schema cache."
		//         }|]
		//         { matchStatus  = 404
		//         , matchHeaders = [ matchContentTypeJson ]
		//         }

		//     it "should be able to resolve when a single unnamed json parameter exists and other overloaded functions are found" $ do
		//       request methodPost "/rpc/overloaded_unnamed_param" [("Content-Type", "application/json")]
		//           [json|{}|]
		//         `shouldRespondWith`
		//           [json| 1 |]
		//           { matchStatus  = 200
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		//       request methodPost "/rpc/overloaded_unnamed_param" [("Content-Type", "application/json")]
		//           [json|{"x": 1, "y": 2}|]
		//         `shouldRespondWith`
		//           [json| 3 |]
		//           { matchStatus  = 200
		//           , matchHeaders = [matchContentTypeJson]
		//           }

		//     it "should be able to fallback to the single unnamed parameter function when other overloaded functions are not found" $ do
		//       request methodPost "/rpc/overloaded_unnamed_param"
		//           [("Content-Type", "application/json")]
		//           [json|{"A": 1, "B": 2, "C": 3}|]
		//         `shouldRespondWith`
		//           [json|{"A": 1, "B": 2, "C": 3}|]
		//       request methodPost "/rpc/overloaded_unnamed_param"
		//           [("Content-Type", "text/plain"), ("Accept", "text/plain")]
		//           [str|unnamed text arg|]
		//         `shouldRespondWith`
		//           [str|unnamed text arg|]
		//       let file = unsafePerformIO $ BL.readFile "test/spec/fixtures/image.png"
		//       r <- request methodPost "/rpc/overloaded_unnamed_param"
		//         [("Content-Type", "application/octet-stream"), ("Accept", "application/octet-stream")]
		//         file
		//       liftIO $ do
		//         let respBody = simpleBody r
		//         respBody `shouldBe` file

		//     it "should call the function with no parameters and not fallback to the single unnamed parameter function when using GET with Content-Type headers" $ do
		//       request methodGet "/rpc/overloaded_unnamed_param" [("Content-Type", "text/plain")] ""
		//         `shouldRespondWith`
		//           [json| 1|]
		//           { matchStatus  = 200 }
		//       request methodGet "/rpc/overloaded_unnamed_param" [("Content-Type", "application/octet-stream")] ""
		//         `shouldRespondWith`
		//           [json| 1|]
		//           { matchStatus  = 200 }

		//     it "should fail to fallback to any single unnamed parameter function when using an unsupported Content-Type header" $ do
		//       request methodPost "/rpc/overloaded_unnamed_param"
		//           [("Content-Type", "text/csv")]
		//           "a,b\n1,2\n4,6\n100,200"
		//         `shouldRespondWith`
		//           [json| {
		//             "hint":"Perhaps you meant to call the function test.overloaded_unnamed_param(x, y)",
		//             "message":"Could not find the function test.overloaded_unnamed_param(a, b) in the schema cache",
		//             "code":"PGRST202",
		//             "details":"Searched for the function test.overloaded_unnamed_param with parameters a, b, but no matches were found in the schema cache."
		//           }|]
		//           { matchStatus  = 404
		//           , matchHeaders = [matchContentTypeJson]
		//           }

		//     it "should fail with multiple choices when two fallback functions with single unnamed json and jsonb parameters exist" $ do
		//       request methodPost "/rpc/overloaded_unnamed_json_jsonb_param" [("Content-Type", "application/json")]
		//           [json|{"A": 1, "B": 2, "C": 3}|]
		//         `shouldRespondWith`
		//           [json| {
		//             "hint":"Try renaming the parameters or the function itself in the database so function overloading can be resolved",
		//             "message":"Could not choose the best candidate function between: test.overloaded_unnamed_json_jsonb_param( => json), test.overloaded_unnamed_json_jsonb_param( => jsonb)",
		//             "code":"PGRST203",
		//             "details":null
		//           }|]
		//           { matchStatus  = 300
		//           , matchHeaders = [matchContentTypeJson]
		//           }

		//     it "should fail on /rpc/unnamed_xml_param when posting invalid xml" $ do
		//       request methodPost "/rpc/unnamed_xml_param"
		//         [("Content-Type", "text/xml"), ("Accept", "text/xml")]
		//         [str|<|]
		//         `shouldRespondWith`
		//           [json| {
		//             "hint":null,
		//             "message":"invalid XML content",
		//             "code":"2200N",
		//             "details":"line 1: StartTag: invalid element name\n<\n ^"
		//           }|]
		//           { matchStatus  = 400
		//           , matchHeaders = [matchContentTypeJson]
		//           }
	}

	test.Execute(t, testConfig, tests)
}
