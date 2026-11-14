package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_NullsStrip holds the cases of NullsStripSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_NullsStrip(t *testing.T) {
	tests := []test.Test{
		// describe "Stripping null values from JSON response" $ do
		//   context "strip nulls from response" $ do
		//     it "strips nulls when Accept: application/vnd.pgrst.array+json;nulls=stripped" $ do
		//       request
		//         methodGet
		//         "/organizations?select=*"
		//         [arrayStrip]
		//         ""
		//         `shouldRespondWith` [json|[{"id":1,"name":"Referee Org","manager_id":1},{"id":2,"name":"Auditor Org","manager_id":2},{"id":3,"name":"Acme","referee":1,"auditor":2,"manager_id":3},{"id":4,"name":"Umbrella","referee":1,"auditor":2,"manager_id":4},{"id":5,"name":"Cyberdyne","referee":3,"auditor":4,"manager_id":5},{"id":6,"name":"Oscorp","referee":3,"auditor":4,"manager_id":6}]|]
		//           { matchStatus = 200
		//           , matchHeaders = [matchCTArrayStrip]
		//           }
		{
			Description:     "strips nulls when Accept: application/vnd.pgrst.array+json;nulls=stripped",
			Method:          "GET",
			Query:           "/organizations?select=*",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.array+json;nulls=stripped"}},
			Expected:        `[{"id":1,"name":"Referee Org","manager_id":1},{"id":2,"name":"Auditor Org","manager_id":2},{"id":3,"name":"Acme","referee":1,"auditor":2,"manager_id":3},{"id":4,"name":"Umbrella","referee":1,"auditor":2,"manager_id":4},{"id":5,"name":"Cyberdyne","referee":3,"auditor":4,"manager_id":5},{"id":6,"name":"Oscorp","referee":3,"auditor":4,"manager_id":6}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/vnd.pgrst.array+json;nulls=stripped; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: application/vnd.pgrst.array+json and nulls=stripped; card 38805 (nulls stripped)",
		},
		//
		//       request
		//         methodPost
		//         "/organizations?select=*"
		//         [arrayStrip, ("Prefer", "return=representation")]
		//         [json|{"id":7,"name":"John","referee":null,"auditor":null,"manager_id":6}|]
		//         `shouldRespondWith` [json|[{"id":7,"name":"John","manager_id":6}]|]
		//           { matchStatus = 201
		//           , matchHeaders = [matchCTArrayStrip]
		//           }
		{
			Description:     "strips nulls when Accept: application/vnd.pgrst.array+json;nulls=stripped",
			Method:          "POST",
			Query:           "/organizations?select=*",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.array+json;nulls=stripped"}, "Prefer": {"return=representation"}},
			Body:            `{"id":7,"name":"John","referee":null,"auditor":null,"manager_id":6}`,
			Expected:        `[{"id":7,"name":"John","manager_id":6}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/vnd.pgrst.array+json;nulls=stripped; charset=utf-8"},
			Status:          201,
			Skip:            "PostgREST 16.4: application/vnd.pgrst.array+json and nulls=stripped; card 38805 (nulls stripped)",
		},
		//
		//       request
		//         methodPatch
		//         "/organizations?id=eq.3&select=*"
		//         [arrayStrip, ("Prefer", "return=representation")]
		//         [json|{"name":"John","referee":null}|]
		//         `shouldRespondWith` [json|[{"id":3,"name":"John","auditor":2,"manager_id":3}]|]
		//           { matchStatus = 200
		//           , matchHeaders = [matchCTArrayStrip]
		//           }
		{
			Description:     "strips nulls when Accept: application/vnd.pgrst.array+json;nulls=stripped",
			Method:          "PATCH",
			Query:           "/organizations?id=eq.3&select=*",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.array+json;nulls=stripped"}, "Prefer": {"return=representation"}},
			Body:            `{"name":"John","referee":null}`,
			Expected:        `[{"id":3,"name":"John","auditor":2,"manager_id":3}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/vnd.pgrst.array+json;nulls=stripped; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: application/vnd.pgrst.array+json and nulls=stripped; card 38805 (nulls stripped)",
		},
		//     it "strips nulls when Accept: application/vnd.pgrst.array;nulls=stripped" $
		//       request
		//         methodGet
		//         "/organizations?select=*"
		//         [("Accept", "application/vnd.pgrst.array;nulls=stripped")]
		//         ""
		//         `shouldRespondWith` [json|[{"id":1,"name":"Referee Org","manager_id":1},{"id":2,"name":"Auditor Org","manager_id":2},{"id":3,"name":"Acme","referee":1,"auditor":2,"manager_id":3},{"id":4,"name":"Umbrella","referee":1,"auditor":2,"manager_id":4},{"id":5,"name":"Cyberdyne","referee":3,"auditor":4,"manager_id":5},{"id":6,"name":"Oscorp","referee":3,"auditor":4,"manager_id":6}]|]
		//           { matchStatus = 200
		//           , matchHeaders = [matchCTArrayStrip]
		//           }
		{
			Description:     "strips nulls when Accept: application/vnd.pgrst.array;nulls=stripped",
			Method:          "GET",
			Query:           "/organizations?select=*",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.array;nulls=stripped"}},
			Expected:        `[{"id":1,"name":"Referee Org","manager_id":1},{"id":2,"name":"Auditor Org","manager_id":2},{"id":3,"name":"Acme","referee":1,"auditor":2,"manager_id":3},{"id":4,"name":"Umbrella","referee":1,"auditor":2,"manager_id":4},{"id":5,"name":"Cyberdyne","referee":3,"auditor":4,"manager_id":5},{"id":6,"name":"Oscorp","referee":3,"auditor":4,"manager_id":6}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/vnd.pgrst.array+json;nulls=stripped; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: application/vnd.pgrst.array+json and nulls=stripped; card 38805 (nulls stripped)",
		},
		//     it "strips nulls when Accept: application/vnd.pgrst.object+json;nulls=stripped" $
		//       request
		//         methodGet
		//         "/organizations?limit=1"
		//         [singularStrip]
		//         ""
		//         `shouldRespondWith` [json|{"id":1,"name":"Referee Org","manager_id":1}|]
		//           { matchStatus = 200
		//           , matchHeaders = [matchCTSingularStrip]
		//           }
		{
			Description:     "strips nulls when Accept: application/vnd.pgrst.object+json;nulls=stripped",
			Method:          "GET",
			Query:           "/organizations?limit=1",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.object+json;nulls=stripped"}},
			Expected:        `{"id":1,"name":"Referee Org","manager_id":1}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/vnd.pgrst.object+json;nulls=stripped; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: application/vnd.pgrst.array+json and nulls=stripped; card 38805 (nulls stripped)",
		},
		//     it "throws error when Accept: application/vnd.pgrst.object+json;nulls=stripped and result not singular" $
		//       request
		//         methodGet
		//         "/organizations?select=*"
		//         [singularStrip]
		//         ""
		//         `shouldRespondWith` [json|{"details":"The result contains 6 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}|]
		//           { matchStatus = 406
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description:     "throws error when Accept: application/vnd.pgrst.object+json;nulls=stripped and result not singular",
			Method:          "GET",
			Query:           "/organizations?select=*",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.object+json;nulls=stripped"}},
			Expected:        `{"details":"The result contains 6 rows","message":"Cannot coerce the result to a single JSON object","code":"PGRST116","hint":null}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          406,
		},
		//   context "strip nulls from response even if explicitly selected" $ do
		//     it "strips nulls when Accept: application/vnd.pgrst.array+json;nulls=stripped" $ do
		//       request
		//         methodGet
		//         "/organizations?select=id,referee,auditor"
		//         [arrayStrip]
		//         ""
		//         `shouldRespondWith` [json|[{"id":1},{"id":2},{"id":3,"referee":1,"auditor":2},{"id":4,"referee":1,"auditor":2},{"id":5,"referee":3,"auditor":4},{"id":6,"referee":3,"auditor":4}]|]
		//           { matchStatus = 200
		//           , matchHeaders = [matchCTArrayStrip]
		//           }
		{
			Description:     "strips nulls when Accept: application/vnd.pgrst.array+json;nulls=stripped",
			Method:          "GET",
			Query:           "/organizations?select=id,referee,auditor",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.array+json;nulls=stripped"}},
			Expected:        `[{"id":1},{"id":2},{"id":3,"referee":1,"auditor":2},{"id":4,"referee":1,"auditor":2},{"id":5,"referee":3,"auditor":4},{"id":6,"referee":3,"auditor":4}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/vnd.pgrst.array+json;nulls=stripped; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: application/vnd.pgrst.array+json and nulls=stripped; card 38805 (nulls stripped)",
		},
		//
		//       request
		//         methodPost
		//         "/organizations?select=id,referee,auditor"
		//         [arrayStrip, ("Prefer", "return=representation")]
		//         [json|{"id":7,"name":"John","referee":null,"auditor":null,"manager_id":6}|]
		//         `shouldRespondWith` [json|[{"id":7}]|]
		//           { matchStatus = 201
		//           , matchHeaders = [matchCTArrayStrip]
		//           }
		{
			Description:     "strips nulls when Accept: application/vnd.pgrst.array+json;nulls=stripped",
			Method:          "POST",
			Query:           "/organizations?select=id,referee,auditor",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.array+json;nulls=stripped"}, "Prefer": {"return=representation"}},
			Body:            `{"id":7,"name":"John","referee":null,"auditor":null,"manager_id":6}`,
			Expected:        `[{"id":7}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/vnd.pgrst.array+json;nulls=stripped; charset=utf-8"},
			Status:          201,
			Skip:            "PostgREST 16.4: application/vnd.pgrst.array+json and nulls=stripped; card 38805 (nulls stripped)",
		},
		//
		//       request
		//         methodPatch
		//         "/organizations?id=eq.3&select=id,name,referee,auditor"
		//         [arrayStrip, ("Prefer", "return=representation")]
		//         [json|{"name":"John","referee":null}|]
		//         `shouldRespondWith` [json|[{"id":3,"name":"John","auditor":2}]|]
		//           { matchStatus = 200
		//           , matchHeaders = [matchCTArrayStrip]
		//           }
		{
			Description:     "strips nulls when Accept: application/vnd.pgrst.array+json;nulls=stripped",
			Method:          "PATCH",
			Query:           "/organizations?id=eq.3&select=id,name,referee,auditor",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.array+json;nulls=stripped"}, "Prefer": {"return=representation"}},
			Body:            `{"name":"John","referee":null}`,
			Expected:        `[{"id":3,"name":"John","auditor":2}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/vnd.pgrst.array+json;nulls=stripped; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: application/vnd.pgrst.array+json and nulls=stripped; card 38805 (nulls stripped)",
		},
		//     it "strips nulls when Accept: application/vnd.pgrst.object+json;nulls=stripped" $
		//       request
		//         methodGet
		//         "/organizations?select=id,referee,auditor&limit=1"
		//         [singularStrip]
		//         ""
		//         `shouldRespondWith` [json|{"id":1}|]
		//           { matchStatus = 200
		//           , matchHeaders = [matchCTSingularStrip]
		//           }
		{
			Description:     "strips nulls when Accept: application/vnd.pgrst.object+json;nulls=stripped",
			Method:          "GET",
			Query:           "/organizations?select=id,referee,auditor&limit=1",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.object+json;nulls=stripped"}},
			Expected:        `{"id":1}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/vnd.pgrst.object+json;nulls=stripped; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: application/vnd.pgrst.array+json and nulls=stripped; card 38805 (nulls stripped)",
		},
		//   context "doesn't strip nulls" $ do
		//     it "doesn't strips nulls when Accept: application/vnd.pgrst.array+json" $
		//       request
		//         methodGet
		//         "/organizations?select=id,referee,auditor"
		//         [("Accept", "application/vnd.pgrst.array+json")]
		//         ""
		//         `shouldRespondWith` [json|[{"id":1,"referee":null,"auditor":null},{"id":2,"referee":null,"auditor":null},{"id":3,"referee":1,"auditor":2},{"id":4,"referee":1,"auditor":2},{"id":5,"referee":3,"auditor":4},{"id":6,"referee":3,"auditor":4}]|]
		//           { matchStatus = 200
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description:     "doesn't strips nulls when Accept: application/vnd.pgrst.array+json",
			Method:          "GET",
			Query:           "/organizations?select=id,referee,auditor",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.array+json"}},
			Expected:        `[{"id":1,"referee":null,"auditor":null},{"id":2,"referee":null,"auditor":null},{"id":3,"referee":1,"auditor":2},{"id":4,"referee":1,"auditor":2},{"id":5,"referee":3,"auditor":4},{"id":6,"referee":3,"auditor":4}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: application/vnd.pgrst.array+json and nulls=stripped; card 38805 (nulls stripped)",
		},
	}
	test.Execute(t, testConfig, tests)
}
