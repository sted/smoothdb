package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_QueryLimited holds the cases of QueryLimitedSpec.hs (PostgREST 16.4), converted by hspec2go.
// The spec runs with db-max-rows = 2: the cases run on a variant server
// with Database.MaxRows 2.
func TestPostgREST_QueryLimited(t *testing.T) {
	tests := []test.Test{
		// describe "Requesting many items with server limits(max-rows) enabled" $ do
		//   it "restricts results" $
		//     get "/items?order=id"
		//       `shouldRespondWith` [json| [{"id":1},{"id":2}] |]
		//         { matchHeaders = ["Content-Range" <:> "0-1/*"]
		//         }
		{
			Description:     "restricts results",
			Method:          "GET",
			Query:           "/items?order=id",
			Expected:        `[{"id":1},{"id":2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
		},
		//   it "respects additional client limiting" $ do
		//     request
		//       methodGet
		//       "/items"
		//       (rangeHdrs $ ByteRangeFromTo 0 0)
		//       ""
		//       `shouldRespondWith` [json| [{"id":1}] |]
		//         { matchHeaders = ["Content-Range" <:> "0-0/*"]
		//         }
		{
			Description:     "respects additional client limiting",
			Method:          "GET",
			Query:           "/items",
			Headers:         test.Headers{"Range": {"0-0"}},
			Expected:        `[{"id":1}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/*"},
			Status:          200,
		},
		//   it "works on all levels" $
		//     get "/users?select=id,tasks(id)&order=id.asc&tasks.order=id.asc"
		//       `shouldRespondWith` [json|[{"id":1,"tasks":[{"id":1},{"id":2}]},{"id":2,"tasks":[{"id":5},{"id":6}]}]|]
		//         { matchHeaders = ["Content-Range" <:> "0-1/*"]
		//         }
		{
			Description:     "works on all levels",
			Method:          "GET",
			Query:           "/users?select=id,tasks(id)&order=id.asc&tasks.order=id.asc",
			Expected:        `[{"id":1,"tasks":[{"id":1},{"id":2}]},{"id":2,"tasks":[{"id":5},{"id":6}]}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
		},
		//   it "succeeds in getting parent embeds despite the limit, see #647" $
		//     get "/tasks?select=id,project:projects(id)&id=gt.5"
		//       `shouldRespondWith` [json|[{"id":6,"project":{"id":3}},{"id":7,"project":{"id":4}}]|]
		//         { matchHeaders = ["Content-Range" <:> "0-1/*"]
		//         }
		{
			Description:     "succeeds in getting parent embeds despite the limit, see #647",
			Method:          "GET",
			Query:           "/tasks?select=id,project:projects(id)&id=gt.5",
			Expected:        `[{"id":6,"project":{"id":3}},{"id":7,"project":{"id":4}}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
		},
		//   it "can offset the parent embed, being consistent with the other embed types" $
		//     get "/tasks?select=id,project:projects(id)&id=gt.5&project.offset=1"
		//       `shouldRespondWith` [json|[{"id":6,"project":null}, {"id":7,"project":null}]|]
		//         { matchHeaders = ["Content-Range" <:> "0-1/*"]
		//         }
		{
			Description:     "can offset the parent embed, being consistent with the other embed types",
			Method:          "GET",
			Query:           "/tasks?select=id,project:projects(id)&id=gt.5&project.offset=1",
			Expected:        `[{"id":6,"project":null}, {"id":7,"project":null}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
			Skip:            "PostgREST 16.4: offset on a to-one embed (project.offset=1); card 32476 (related limit and offset)",
		},
		//   context "count=estimated" $ do
		//     it "uses the query planner guess when query rows > maxRows" $
		//       request
		//         methodHead
		//         "/getallprojects_view"
		//         [("Prefer", "count=estimated")]
		//         ""
		//         `shouldRespondWith` ""
		//           { matchStatus = 206
		//           , matchHeaders =
		//               [ matchContentTypeJson
		//               , "Content-Range" <:> "0-1/2019"
		//               ]
		//           }
		{
			Description:     "uses the query planner guess when query rows > maxRows",
			Method:          "HEAD",
			Query:           "/getallprojects_view",
			Headers:         test.Headers{"Prefer": {"count=estimated"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Range": "0-1/2019"},
			Status:          206,
			Skip:            "PostgREST 16.4: count=estimated with db-max-rows; card 32483 (planned count)",
		},
		//     it "gives exact count when query rows <= maxRows" $
		//       request
		//         methodHead
		//         "/getallprojects_view?id=lt.3"
		//         [("Prefer", "count=estimated")]
		//         ""
		//         `shouldRespondWith` ""
		//           { matchHeaders =
		//               [ matchContentTypeJson
		//               , "Content-Range" <:> "0-1/2"
		//               ]
		//           }
		{
			Description:     "gives exact count when query rows <= maxRows",
			Method:          "HEAD",
			Query:           "/getallprojects_view?id=lt.3",
			Headers:         test.Headers{"Prefer": {"count=estimated"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Range": "0-1/2"},
			Status:          200,
			Skip:            "PostgREST 16.4: count=estimated with db-max-rows; card 32483 (planned count)",
		},
		//     it "only uses the query planner guess if it's indeed greater than the exact count" $
		//       request
		//         methodHead
		//         "/get_projects_above_view"
		//         [("Prefer", "count=estimated")]
		//         ""
		//         `shouldRespondWith` ""
		//           { matchStatus = 206
		//           , matchHeaders =
		//               [ matchContentTypeJson
		//               , "Content-Range" <:> "0-1/3"
		//               ]
		//           }
		{
			Description:     "only uses the query planner guess if it's indeed greater than the exact count",
			Method:          "HEAD",
			Query:           "/get_projects_above_view",
			Headers:         test.Headers{"Prefer": {"count=estimated"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Content-Range": "0-1/3"},
			Status:          206,
			Skip:            "PostgREST 16.4: count=estimated with db-max-rows; card 32483 (planned count)",
		},
		//   context "max-rows=2 on mutations" $ do
		//     it "doesn't affect insertions" $
		//       request
		//         methodPost
		//         "/projects?select=id,name"
		//         [("Prefer", "return=representation")]
		//         [json| [
		//             { "id": 6, "name": "BeOS" },
		//             { "id": 7, "name": "PopOS" },
		//             { "id": 8, "name": "HaikuOS" } ]|]
		//         `shouldRespondWith` [json| [
		//             { "id": 6, "name": "BeOS" },
		//             { "id": 7, "name": "PopOS" },
		//             { "id": 8, "name": "HaikuOS" } ]|]
		//           { matchStatus = 201
		//           }
		{
			Description: "doesn't affect insertions",
			Method:      "POST",
			Query:       "/projects?select=id,name",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Body: `[
              { "id": 6, "name": "BeOS" },
              { "id": 7, "name": "PopOS" },
              { "id": 8, "name": "HaikuOS" } ]`,
			Expected: `[
              { "id": 6, "name": "BeOS" },
              { "id": 7, "name": "PopOS" },
              { "id": 8, "name": "HaikuOS" } ]`,
			Status: 201,
		},
		//     it "doesn't affect updates(2 rows would be modified if it did)" $
		//       request
		//         methodPatch
		//         "/employees?select=first_name,last_name,occupation&order=last_name"
		//         [("Prefer", "return=representation")]
		//         [json| [{"occupation": "Barista"}] |]
		//         `shouldRespondWith` [json|[
		//               { "first_name": "Daniel B.", "last_name": "Lyon", "occupation": "Barista" },
		//               { "first_name": "Frances M.", "last_name": "Roe", "occupation": "Barista" },
		//               { "first_name": "Edwin S.", "last_name": "Smith", "occupation": "Barista" } ]|]
		//           { matchStatus = 200
		//           }
		{
			Description: "doesn't affect updates(2 rows would be modified if it did)",
			Method:      "PATCH",
			Query:       "/employees?select=first_name,last_name,occupation&order=last_name",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Body:        `[{"occupation": "Barista"}]`,
			Expected: `[
                { "first_name": "Daniel B.", "last_name": "Lyon", "occupation": "Barista" },
                { "first_name": "Frances M.", "last_name": "Roe", "occupation": "Barista" },
                { "first_name": "Edwin S.", "last_name": "Smith", "occupation": "Barista" } ]`,
			Status: 200,
		},
		//     it "doesn't affect deletions" $
		//       request
		//         methodDelete
		//         "/employees?select=first_name,last_name&order=last_name"
		//         [("Prefer", "return=representation")]
		//         mempty
		//         `shouldRespondWith` [json| [
		//             { "first_name": "Daniel B.", "last_name": "Lyon" },
		//             { "first_name": "Frances M.", "last_name": "Roe" },
		//             { "first_name": "Edwin S.", "last_name": "Smith" } ]|]
		//           { matchStatus = 200
		//           }
		{
			Description: "doesn't affect deletions",
			Method:      "DELETE",
			Query:       "/employees?select=first_name,last_name&order=last_name",
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected: `[
              { "first_name": "Daniel B.", "last_name": "Lyon" },
              { "first_name": "Frances M.", "last_name": "Roe" },
              { "first_name": "Edwin S.", "last_name": "Smith" } ]`,
			Status: 200,
		},
		//   context "max-rows is set and limits are requested" $ do
		//     it "should work with limit 0" $
		//       get "/items?limit=0"
		//         `shouldRespondWith` [json| [] |]
		//           { matchHeaders = ["Content-Range" <:> "*/*"]
		//           }
		{
			Description:     "should work with limit 0",
			Method:          "GET",
			Query:           "/items?limit=0",
			Expected:        `[]`,
			ExpectedHeaders: map[string]string{"Content-Range": "*/*"},
			Status:          200,
		},
	}
	withVariant(t, map[string]any{"Database.MaxRows": int64(2)}, false, func(cfg test.Config) {
		test.Execute(t, cfg, tests)
	})
}
