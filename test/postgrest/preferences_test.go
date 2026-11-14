package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_PreferHandling holds the cases of HandlingSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_PreferHandling(t *testing.T) {
	tests := []test.Test{
		// describe "test Prefer: handling" $ do
		//   context "check behaviour of Prefer: handling=strict" $ do
		//     it "throws error when handling=strict and invalid prefs are given" $
		//       request methodGet "/items" [("Prefer", "handling=strict, anything")] ""
		//         `shouldRespondWith` [json|{"details":"Invalid preferences: anything","message":"Invalid preferences given with handling=strict","code":"PGRST122","hint":null}|]
		//           { matchStatus = 400
		//           }
		{
			Description: "throws error when handling=strict and invalid prefs are given",
			Method:      "GET",
			Query:       "/items",
			Headers:     test.Headers{"Prefer": {"handling=strict, anything"}},
			Expected:    `{"details":"Invalid preferences: anything","message":"Invalid preferences given with handling=strict","code":"PGRST122","hint":null}`,
			Status:      400,
			Skip:        "PostgREST 16.4: Prefer handling=strict rejects unknown preferences with PGRST122; card 32482 (preferences)",
		},
		//     it "throw error when handling=strict and invalid prefs are given with multiples in separate prefers" $
		//       request methodGet "/items" [("Prefer", "handling=strict"), ("Prefer", "something, else")] ""
		//         `shouldRespondWith` [json|{"details":"Invalid preferences: something, else","message":"Invalid preferences given with handling=strict","code":"PGRST122","hint":null}|]
		//           { matchStatus = 400
		//           }
		{
			Description: "throw error when handling=strict and invalid prefs are given with multiples in separate prefers",
			Method:      "GET",
			Query:       "/items",
			Headers:     test.Headers{"Prefer": {"handling=strict", "something, else"}},
			Expected:    `{"details":"Invalid preferences: something, else","message":"Invalid preferences given with handling=strict","code":"PGRST122","hint":null}`,
			Status:      400,
			Skip:        "PostgREST 16.4: Prefer handling=strict rejects unknown preferences with PGRST122; card 32482 (preferences)",
		},
		//     it "throws error with post request" $
		//       request
		//         methodPost
		//         "/organizations?select=*"
		//         [("Prefer", "return=representation, handling=strict, anything")]
		//         [json|{"id":7,"name":"John","referee":null,"auditor":null,"manager_id":6}|]
		//         `shouldRespondWith` [json|{"details":"Invalid preferences: anything","message":"Invalid preferences given with handling=strict","code":"PGRST122","hint":null}|]
		//           { matchStatus = 400
		//           }
		{
			Description: "throws error with post request",
			Method:      "POST",
			Query:       "/organizations?select=*",
			Headers:     test.Headers{"Prefer": {"return=representation, handling=strict, anything"}},
			Body:        `{"id":7,"name":"John","referee":null,"auditor":null,"manager_id":6}`,
			Expected:    `{"details":"Invalid preferences: anything","message":"Invalid preferences given with handling=strict","code":"PGRST122","hint":null}`,
			Status:      400,
			Skip:        "PostgREST 16.4: Prefer handling=strict rejects unknown preferences with PGRST122; card 32482 (preferences)",
		},
		//     it "throws error with rpc" $
		//       request
		//         methodPost
		//         "/rpc/overloaded_unnamed_param"
		//         [("Content-Type", "application/json"), ("Prefer", "handling=strict, anything")]
		//         [json|{}|]
		//         `shouldRespondWith` [json|{"details":"Invalid preferences: anything","message":"Invalid preferences given with handling=strict","code":"PGRST122","hint":null}|]
		//           { matchStatus = 400
		//           }
		{
			Description: "throws error with rpc",
			Method:      "POST",
			Query:       "/rpc/overloaded_unnamed_param",
			Headers:     test.Headers{"Content-Type": {"application/json"}, "Prefer": {"handling=strict, anything"}},
			Body:        `{}`,
			Expected:    `{"details":"Invalid preferences: anything","message":"Invalid preferences given with handling=strict","code":"PGRST122","hint":null}`,
			Status:      400,
			Skip:        "PostgREST 16.4: Prefer handling=strict rejects unknown preferences with PGRST122; card 32482 (preferences)",
		},
		//   context "check behaviour of Prefer: handling=lenient" $ do
		//     it "does not throw error when handling=lenient and invalid prefs" $
		//       request methodGet "/items" [("Prefer", "handling=lenient, anything")] ""
		//         `shouldRespondWith` 200
		{
			Description: "does not throw error when handling=lenient and invalid prefs",
			Method:      "GET",
			Query:       "/items",
			Headers:     test.Headers{"Prefer": {"handling=lenient, anything"}},
			Status:      200,
		},
		//     it "does not throw error when handling=lenient and invalid prefs in multiples prefers" $
		//       request methodGet "/items" [("Prefer", "handling=lenient"), ("Prefer", "anything")] ""
		//         `shouldRespondWith` 200
		{
			Description: "does not throw error when handling=lenient and invalid prefs in multiples prefers",
			Method:      "GET",
			Query:       "/items",
			Headers:     test.Headers{"Prefer": {"handling=lenient", "anything"}},
			Status:      200,
		},
		//     it "does not throw error with post request" $
		//       request
		//         methodPost
		//         "/organizations?select=*"
		//         [("Prefer", "return=representation, handling=lenient, anything")]
		//         [json|{"id":7,"name":"John","referee":null,"auditor":null,"manager_id":6}|]
		//         `shouldRespondWith` [json|[{"id":7,"name":"John","referee":null,"auditor":null,"manager_id":6}]|]
		//           { matchStatus = 201
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description:     "does not throw error with post request",
			Method:          "POST",
			Query:           "/organizations?select=*",
			Headers:         test.Headers{"Prefer": {"return=representation, handling=lenient, anything"}},
			Body:            `{"id":7,"name":"John","referee":null,"auditor":null,"manager_id":6}`,
			Expected:        `[{"id":7,"name":"John","referee":null,"auditor":null,"manager_id":6}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          201,
		},
		//     it "does not throw error with rpc" $
		//       request
		//         methodPost
		//         "/rpc/overloaded_unnamed_param"
		//         [("Content-Type", "application/json"), ("Prefer", "handling=lenient, anything")]
		//         [json|{}|]
		//         `shouldRespondWith` [json| 1 |]
		//           { matchStatus = 200
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description:     "does not throw error with rpc",
			Method:          "POST",
			Query:           "/rpc/overloaded_unnamed_param",
			Headers:         test.Headers{"Content-Type": {"application/json"}, "Prefer": {"handling=lenient, anything"}},
			Body:            `{}`,
			Expected:        `1`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
	}
	test.Execute(t, testConfig, tests)
}

// TestPostgREST_MaxAffected holds the cases of MaxAffectedSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_MaxAffected(t *testing.T) {
	tests := []test.Test{
		// context "test Prefer: max-affected with handling=strict" $ do
		//   it "should fail if items deleted more than 10" $
		//     request
		//       methodDelete
		//       "/items?id=lt.15"
		//       [("Prefer", "handling=strict, max-affected=10")]
		//       ""
		//       `shouldRespondWith` [json|{"code":"PGRST124","details":"The query affects 14 rows","hint":null,"message":"Query result exceeds max-affected preference constraint"}|]
		//         { matchStatus = 400
		//         }
		{
			Description: "should fail if items deleted more than 10",
			Method:      "DELETE",
			Query:       "/items?id=lt.15",
			Headers:     test.Headers{"Prefer": {"handling=strict, max-affected=10"}},
			Expected:    `{"code":"PGRST124","details":"The query affects 14 rows","hint":null,"message":"Query result exceeds max-affected preference constraint"}`,
			Status:      400,
			Skip:        "PostgREST 16.4: Prefer max-affected with handling=strict fails with PGRST124; card 32482 (preferences)",
		},
		//   it "should succeed if items deleted less than 10" $
		//     request
		//       methodDelete
		//       "/items?id=lt.10"
		//       [("Prefer", "handling=strict, max-affected=10")]
		//       ""
		//       `shouldRespondWith` ""
		//         { matchStatus = 204
		//         , matchHeaders = ["Preference-Applied" <:> "handling=strict, max-affected=10"]
		//         }
		{
			Description:     "should succeed if items deleted less than 10",
			Method:          "DELETE",
			Query:           "/items?id=lt.10",
			Headers:         test.Headers{"Prefer": {"handling=strict, max-affected=10"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Preference-Applied": "handling=strict, max-affected=10"},
			Status:          204,
			Skip:            "PostgREST 16.4: Preference-Applied names handling and max-affected; card 32482 (preferences)",
		},
		//   it "should fail if items updated more than 0" $
		//     request
		//       methodPatch
		//       "/tiobe_pls?name=eq.Java"
		//       [("Prefer", "handling=strict, max-affected=0")]
		//       [json| [{"name":"Java", "rank":19}] |]
		//       `shouldRespondWith` [json|{"code":"PGRST124","details":"The query affects 1 rows","hint":null,"message":"Query result exceeds max-affected preference constraint"}|]
		//         { matchStatus = 400
		//         }
		{
			Description: "should fail if items updated more than 0",
			Method:      "PATCH",
			Query:       "/tiobe_pls?name=eq.Java",
			Headers:     test.Headers{"Prefer": {"handling=strict, max-affected=0"}},
			Body:        `[{"name":"Java", "rank":19}]`,
			Expected:    `{"code":"PGRST124","details":"The query affects 1 rows","hint":null,"message":"Query result exceeds max-affected preference constraint"}`,
			Status:      400,
			Skip:        "PostgREST 16.4: Prefer max-affected with handling=strict fails with PGRST124; card 32482 (preferences)",
		},
		//   it "should succeed if items updated equal 1" $
		//     request
		//       methodDelete
		//       "/tiobe_pls?name=eq.Java"
		//       [("Prefer", "handling=strict, max-affected=1")]
		//       [json| [{"name":"Java", "rank":19}] |]
		//       `shouldRespondWith` ""
		//         { matchStatus = 204
		//         , matchHeaders = ["Preference-Applied" <:> "handling=strict, max-affected=1"]
		//         }
		{
			Description:     "should succeed if items updated equal 1",
			Method:          "DELETE",
			Query:           "/tiobe_pls?name=eq.Java",
			Headers:         test.Headers{"Prefer": {"handling=strict, max-affected=1"}},
			Body:            `[{"name":"Java", "rank":19}]`,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Preference-Applied": "handling=strict, max-affected=1"},
			Status:          204,
			Skip:            "PostgREST 16.4: Preference-Applied names handling and max-affected; card 32482 (preferences)",
		},
		// context "test Prefer: max-affected with handling=lenient" $ do
		//   it "should not fail" $
		//     request
		//       methodDelete
		//       "/items?id=lt.15"
		//       [("Prefer", "handling=lenient, max-affected=10")]
		//       ""
		//       `shouldRespondWith` ""
		//         { matchStatus = 204
		//         , matchHeaders = ["Preference-Applied" <:> "handling=lenient"]
		//         }
		{
			Description:     "should not fail",
			Method:          "DELETE",
			Query:           "/items?id=lt.15",
			Headers:         test.Headers{"Prefer": {"handling=lenient, max-affected=10"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Preference-Applied": "handling=lenient"},
			Status:          204,
			Skip:            "PostgREST 16.4: Preference-Applied names handling and max-affected; card 32482 (preferences)",
		},
		//   it "should succeed if items deleted less than 10" $
		//     request
		//       methodDelete
		//       "/items?id=lt.10"
		//       [("Prefer", "handling=lenient, max-affected=10")]
		//       ""
		//       `shouldRespondWith` ""
		//         { matchStatus = 204
		//         , matchHeaders = ["Preference-Applied" <:> "handling=lenient"]
		//         }
		{
			Description:     "should succeed if items deleted less than 10",
			Method:          "DELETE",
			Query:           "/items?id=lt.10",
			Headers:         test.Headers{"Prefer": {"handling=lenient, max-affected=10"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Preference-Applied": "handling=lenient"},
			Status:          204,
			Skip:            "PostgREST 16.4: Preference-Applied names handling and max-affected; card 32482 (preferences)",
		},
		//   it "should not fail" $
		//     request
		//       methodPatch
		//       "/tiobe_pls?name=eq.Java"
		//       [("Prefer", "handling=lenient, max-affected=0")]
		//       [json| [{"name":"Java", "rank":19}] |]
		//       `shouldRespondWith` ""
		//         { matchStatus = 204
		//         , matchHeaders = ["Preference-Applied" <:> "handling=lenient"]
		//         }
		{
			Description:     "should not fail",
			Method:          "PATCH",
			Query:           "/tiobe_pls?name=eq.Java",
			Headers:         test.Headers{"Prefer": {"handling=lenient, max-affected=0"}},
			Body:            `[{"name":"Java", "rank":19}]`,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Preference-Applied": "handling=lenient"},
			Status:          204,
			Skip:            "PostgREST 16.4: Preference-Applied names handling and max-affected; card 32482 (preferences)",
		},
		//   it "should succeed if items updated equal 1" $
		//     request
		//       methodDelete
		//       "/tiobe_pls?name=eq.Java"
		//       [("Prefer", "handling=lenient, max-affected=1")]
		//       [json| [{"name":"Java", "rank":19}] |]
		//       `shouldRespondWith` ""
		//         { matchStatus = 204
		//         , matchHeaders = ["Preference-Applied" <:> "handling=lenient"]
		//         }
		{
			Description:     "should succeed if items updated equal 1",
			Method:          "DELETE",
			Query:           "/tiobe_pls?name=eq.Java",
			Headers:         test.Headers{"Prefer": {"handling=lenient, max-affected=1"}},
			Body:            `[{"name":"Java", "rank":19}]`,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Preference-Applied": "handling=lenient"},
			Status:          204,
			Skip:            "PostgREST 16.4: Preference-Applied names handling and max-affected; card 32482 (preferences)",
		},
		// context "test Prefer: max-affected with rpc" $ do
		//   it "should fail with rpc when deleting rows more than prefered with returns setof" $
		//     request
		//       methodPost
		//       "/rpc/delete_items_returns_setof"
		//       [("Prefer", "handling=strict, max-affected=10")]
		//       ""
		//       `shouldRespondWith` [json| {"code":"PGRST124","details":"The query affects 15 rows","hint":null,"message":"Query result exceeds max-affected preference constraint"} |]
		//         { matchStatus = 400
		//         }
		{
			Description: "should fail with rpc when deleting rows more than prefered with returns setof",
			Method:      "POST",
			Query:       "/rpc/delete_items_returns_setof",
			Headers:     test.Headers{"Prefer": {"handling=strict, max-affected=10"}},
			Expected:    `{"code":"PGRST124","details":"The query affects 15 rows","hint":null,"message":"Query result exceeds max-affected preference constraint"}`,
			Status:      400,
			Skip:        "PostgREST 16.4: Prefer max-affected with handling=strict fails with PGRST124; card 32482 (preferences)",
		},
		//   it "should fail with rpc when deleting rows more than prefered with returns table" $
		//     request
		//       methodPost
		//       "/rpc/delete_items_returns_table"
		//       [("Prefer", "handling=strict, max-affected=10")]
		//       ""
		//       `shouldRespondWith` [json| {"code":"PGRST124","details":"The query affects 15 rows","hint":null,"message":"Query result exceeds max-affected preference constraint"} |]
		//         { matchStatus = 400
		//         }
		{
			Description: "should fail with rpc when deleting rows more than prefered with returns table",
			Method:      "POST",
			Query:       "/rpc/delete_items_returns_table",
			Headers:     test.Headers{"Prefer": {"handling=strict, max-affected=10"}},
			Expected:    `{"code":"PGRST124","details":"The query affects 15 rows","hint":null,"message":"Query result exceeds max-affected preference constraint"}`,
			Status:      400,
			Skip:        "PostgREST 16.4: Prefer max-affected with handling=strict fails with PGRST124; card 32482 (preferences)",
		},
		//   it "should succeed with rpc deleting rows less than prefered with returns setof" $
		//     request
		//       methodPost
		//       "/rpc/delete_items_returns_setof"
		//       [("Prefer", "handling=strict, max-affected=20")]
		//       ""
		//       `shouldRespondWith` [json|[{"id":1},{"id":2},{"id":3},{"id":4},{"id":5},{"id":6},{"id":7},
		//                {"id":8},{"id":9},{"id":10},{"id":11},{"id":12},{"id":13},
		//                {"id":14},{"id":15}]|]
		//         { matchStatus = 200
		//         }
		{
			Description: "should succeed with rpc deleting rows less than prefered with returns setof",
			Method:      "POST",
			Query:       "/rpc/delete_items_returns_setof",
			Headers:     test.Headers{"Prefer": {"handling=strict, max-affected=20"}},
			Expected: `[{"id":1},{"id":2},{"id":3},{"id":4},{"id":5},{"id":6},{"id":7},
                 {"id":8},{"id":9},{"id":10},{"id":11},{"id":12},{"id":13},
                 {"id":14},{"id":15}]`,
			Status: 200,
		},
		//   it "should succeed with rpc deleting rows less than prefered with returns table" $
		//     request
		//       methodPost
		//       "/rpc/delete_items_returns_table"
		//       [("Prefer", "handling=strict, max-affected=20")]
		//       ""
		//       `shouldRespondWith` [json|[{"id":1},{"id":2},{"id":3},{"id":4},{"id":5},{"id":6},{"id":7},
		//                {"id":8},{"id":9},{"id":10},{"id":11},{"id":12},{"id":13},
		//                {"id":14},{"id":15}]|]
		//         { matchStatus = 200
		//         }
		{
			Description: "should succeed with rpc deleting rows less than prefered with returns table",
			Method:      "POST",
			Query:       "/rpc/delete_items_returns_table",
			Headers:     test.Headers{"Prefer": {"handling=strict, max-affected=20"}},
			Expected: `[{"id":1},{"id":2},{"id":3},{"id":4},{"id":5},{"id":6},{"id":7},
                 {"id":8},{"id":9},{"id":10},{"id":11},{"id":12},{"id":13},
                 {"id":14},{"id":15}]`,
			Status: 200,
		},
		//   it "should fail with rpc when returns void with handling=strict" $
		//     request
		//       methodPost
		//       "/rpc/delete_items_returns_void"
		//       [("Prefer", "handling=strict, max-affected=20")]
		//       ""
		//       `shouldRespondWith` [json| {"code":"PGRST128","details":null,"hint":null,"message":"Function must return SETOF or TABLE when max-affected preference is used with handling=strict"} |]
		//         { matchStatus = 400
		//         }
		{
			Description: "should fail with rpc when returns void with handling=strict",
			Method:      "POST",
			Query:       "/rpc/delete_items_returns_void",
			Headers:     test.Headers{"Prefer": {"handling=strict, max-affected=20"}},
			Expected:    `{"code":"PGRST128","details":null,"hint":null,"message":"Function must return SETOF or TABLE when max-affected preference is used with handling=strict"}`,
			Status:      400,
			Skip:        "PostgREST 16.4: PGRST128 for max-affected on a function that does not return SETOF or TABLE; card 32482 (preferences)",
		},
	}
	test.Execute(t, testConfig, tests)
}
