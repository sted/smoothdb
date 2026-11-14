package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_ServerTiming holds the cases of ServerTimingSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_ServerTiming(t *testing.T) {
	tests := []test.Test{
		// describe "Show Duration on Server-Timing header" $ do
		//   context "responds with Server-Timing header" $ do
		//     it "works with get request" $ do
		//       request
		//         methodGet
		//         "/organizations?id=eq.6"
		//         []
		//         ""
		//         `shouldRespondWith` [json|[{"id":6,"name":"Oscorp","referee":3,"auditor":4,"manager_id":6}]|]
		//           { matchStatus = 200
		//           , matchHeaders = matchContentTypeJson : map matchServerTimingHasTiming ["jwt", "parse", "plan", "transaction", "response"]
		//           }
		// TODO(hspec2go): not a list: matchContentTypeJson : map matchServerTimingHasTiming [ "jwt" , "parse" , "plan" , "transaction" , "response" ]
		{
			Description: "works with get request",
			Method:      "GET",
			Query:       "/organizations?id=eq.6",
			Skip:        "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability); the expectations are not converted",
		},
		//     it "works with post request" $
		//       request
		//         methodPost
		//         "/organizations?select=*"
		//         [("Prefer", "return=representation")]
		//         [json|{"id":7,"name":"John","referee":null,"auditor":null,"manager_id":6}|]
		//         `shouldRespondWith` [json|[{"id":7,"name":"John","referee":null,"auditor":null,"manager_id":6}]|]
		//           { matchStatus = 201
		//           , matchHeaders = matchContentTypeJson : map matchServerTimingHasTiming ["jwt", "parse", "plan", "transaction", "response"]
		//           }
		// TODO(hspec2go): not a list: matchContentTypeJson : map matchServerTimingHasTiming [ "jwt" , "parse" , "plan" , "transaction" , "response" ]
		{
			Description: "works with post request",
			Method:      "POST",
			Query:       "/organizations?select=*",
			Skip:        "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability); the expectations are not converted",
		},
		//     it "works with patch request" $
		//       request
		//         methodPatch
		//         "/no_pk?b=eq.0"
		//         mempty
		//         [json| { b: "1" } |]
		//         `shouldRespondWith` ""
		//           { matchStatus = 204
		//           , matchHeaders = matchHeaderAbsent hContentType : map matchServerTimingHasTiming ["jwt", "parse", "plan", "transaction", "response"]
		//           }
		// TODO(hspec2go): not a list: matchHeaderAbsent hContentType : map matchServerTimingHasTiming [ "jwt" , "parse" , "plan" , "transaction" , "response" ]
		{
			Description: "works with patch request",
			Method:      "PATCH",
			Query:       "/no_pk?b=eq.0",
			Skip:        "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability); the expectations are not converted",
		},
		//     it "works with put request" $
		//       request
		//         methodPut
		//         "/tiobe_pls?name=eq.Python"
		//         [("Prefer", "return=representation")]
		//         [json| [ { "name": "Python", "rank": 19 } ]|]
		//         `shouldRespondWith` [json| [ { "name": "Python", "rank": 19 } ]|]
		//           { matchStatus = 200
		//           , matchHeaders = map matchServerTimingHasTiming ["jwt", "parse", "plan", "transaction", "response"]
		//           }
		// TODO(hspec2go): not a list: map matchServerTimingHasTiming [ "jwt" , "parse" , "plan" , "transaction" , "response" ]
		{
			Description: "works with put request",
			Method:      "PUT",
			Query:       "/tiobe_pls?name=eq.Python",
			Skip:        "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability); the expectations are not converted",
		},
		//     it "works with delete request" $
		//       request
		//         methodDelete
		//         "/items?id=eq.1"
		//         []
		//         ""
		//         `shouldRespondWith` ""
		//           { matchStatus = 204
		//           , matchHeaders = matchHeaderAbsent hContentType : map matchServerTimingHasTiming ["jwt", "parse", "plan", "transaction", "response"]
		//           }
		// TODO(hspec2go): not a list: matchHeaderAbsent hContentType : map matchServerTimingHasTiming [ "jwt" , "parse" , "plan" , "transaction" , "response" ]
		{
			Description: "works with delete request",
			Method:      "DELETE",
			Query:       "/items?id=eq.1",
			Skip:        "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability); the expectations are not converted",
		},
		//     it "works with rpc call" $
		//       request
		//         methodPost
		//         "/rpc/ret_point_overloaded"
		//         []
		//         [json|{"x": 1, "y": 2}|]
		//         `shouldRespondWith` [json|{"x": 1, "y": 2}|]
		//           { matchStatus = 200
		//           , matchHeaders = map matchServerTimingHasTiming ["jwt", "parse", "plan", "transaction", "response"]
		//           }
		// TODO(hspec2go): not a list: map matchServerTimingHasTiming [ "jwt" , "parse" , "plan" , "transaction" , "response" ]
		{
			Description: "works with rpc call",
			Method:      "POST",
			Query:       "/rpc/ret_point_overloaded",
			Skip:        "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability); the expectations are not converted",
		},
		//     it "works with root spec" $
		//       request
		//         methodHead
		//         "/"
		//         []
		//         ""
		//         `shouldRespondWith` ""
		//           { matchStatus = 200
		//           , matchHeaders = map matchServerTimingHasTiming ["jwt", "parse", "plan", "transaction", "response"]
		//           }
		// TODO(hspec2go): not a list: map matchServerTimingHasTiming [ "jwt" , "parse" , "plan" , "transaction" , "response" ]
		{
			Description: "works with root spec",
			Method:      "HEAD",
			Query:       "/",
			Skip:        "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability); the expectations are not converted",
		},
		//     it "works with OPTIONS method" $ do
		//       request
		//         methodOptions
		//         "/organizations"
		//         []
		//         ""
		//         `shouldRespondWith` ""
		//           { matchStatus = 200
		//           , matchHeaders = map matchServerTimingHasTiming ["jwt", "parse", "response"]
		//           }
		// TODO(hspec2go): not a list: map matchServerTimingHasTiming [ "jwt" , "parse" , "response" ]
		{
			Description: "works with OPTIONS method",
			Method:      "OPTIONS",
			Query:       "/organizations",
			Skip:        "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability); the expectations are not converted",
		},
		//       request
		//         methodOptions
		//         "/rpc/getallprojects"
		//         []
		//         ""
		//         `shouldRespondWith` ""
		//           { matchStatus = 200
		//           , matchHeaders = map matchServerTimingHasTiming ["jwt", "parse", "response"]
		//           }
		// TODO(hspec2go): not a list: map matchServerTimingHasTiming [ "jwt" , "parse" , "response" ]
		{
			Description: "works with OPTIONS method",
			Method:      "OPTIONS",
			Query:       "/rpc/getallprojects",
			Skip:        "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability); the expectations are not converted",
		},
		//       request
		//         methodOptions
		//         "/"
		//         []
		//         ""
		//         `shouldRespondWith` ""
		//           { matchStatus = 200
		//           , matchHeaders = map matchServerTimingHasTiming ["jwt", "parse", "response"]
		//           }
		// TODO(hspec2go): not a list: map matchServerTimingHasTiming [ "jwt" , "parse" , "response" ]
		{
			Description: "works with OPTIONS method",
			Method:      "OPTIONS",
			Query:       "/",
			Skip:        "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability); the expectations are not converted",
		},
	}
	test.Execute(t, testConfig, tests)
}

// TestPostgREST_Observability holds the cases of ObservabilitySpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_Observability(t *testing.T) {
	tests := []test.Test{
		// describe "Observability" $ do
		//   it "includes the server trace header on the response" $ do
		//     request
		//       methodHead
		//       "/"
		//       [("X-Request-Id", "1")]
		//       ""
		//       `shouldRespondWith` ""
		//         { matchHeaders = ["X-Request-Id" <:> "1"]
		//         }
		{
			Description:     "includes the server trace header on the response",
			Method:          "HEAD",
			Query:           "/",
			Headers:         test.Headers{"X-Request-Id": {"1"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"X-Request-Id": "1"},
			Status:          200,
			Skip:            "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability)",
		},
		//
		//     request
		//       methodHead
		//       "/projects"
		//       [("X-Request-Id", "2")]
		//       ""
		//       `shouldRespondWith` ""
		//         { matchHeaders = ["X-Request-Id" <:> "2"]
		//         }
		{
			Description:     "includes the server trace header on the response",
			Method:          "HEAD",
			Query:           "/projects",
			Headers:         test.Headers{"X-Request-Id": {"2"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"X-Request-Id": "2"},
			Status:          200,
			Skip:            "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability)",
		},
		//
		//     request
		//       methodHead
		//       "/rpc/add_them?a=2&b=4"
		//       [("X-Request-Id", "3")]
		//       ""
		//       `shouldRespondWith` ""
		//         { matchHeaders = ["X-Request-Id" <:> "3"]
		//         }
		{
			Description:     "includes the server trace header on the response",
			Method:          "HEAD",
			Query:           "/rpc/add_them?a=2&b=4",
			Headers:         test.Headers{"X-Request-Id": {"3"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"X-Request-Id": "3"},
			Status:          200,
			Skip:            "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability)",
		},
	}
	test.Execute(t, testConfig, tests)
}

// TestPostgREST_JwtCache holds the cases of JwtCacheSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_JwtCache(t *testing.T) {
	tests := []test.Test{
		// it "server-timing duration is exposed for JWT with expiry" $ do
		//   !currentTime <- liftIO $ relativeSeconds 1800 -- 30 minutes, evaluate strictly
		//   let
		//     jwtPayload = [json|{ "role": "postgrest_test_author", "exp": #{currentTime} }|]
		//     auth = authHeaderJWT $ generateJWT jwtPayload
		//
		//   res1 <- request methodGet "/authors_only" [auth] ""
		//   let jwtDur1 = M.lookup "jwt" $ parseServerTimingHeader $ simpleHeaders res1
		//   res2 <- request methodGet "/authors_only" [auth] ""
		//   let jwtDur2 = M.lookup "jwt" $ parseServerTimingHeader $ simpleHeaders res2
		//
		//   -- With jwt caching the parse time of second request with the same token
		//   -- should be at least as fast as the first one
		//   let dur2IsLessThanEq = fromMaybe False $ liftA2 (<=) jwtDur2 jwtDur1
		//   liftIO $ dur2IsLessThanEq `shouldBe` True
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "server-timing duration is exposed for JWT with expiry",
			Method:      "GET",
			Query:       "/authors_only",
			Skip:        "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability); the expectations are not converted",
		},
		// it "server-timing duration is exposed for JWT without expiry" $ do
		//   let
		//     jwtPayload = [json|{ "role": "postgrest_test_author" }|]
		//     auth = authHeaderJWT $ generateJWT jwtPayload
		//
		//   res1 <- request methodGet "/authors_only" [auth] ""
		//   let jwtDur1 = M.lookup "jwt" $ parseServerTimingHeader $ simpleHeaders res1
		//   res2 <- request methodGet "/authors_only" [auth] ""
		//   let jwtDur2 = M.lookup "jwt" $ parseServerTimingHeader $ simpleHeaders res2
		//
		//   liftIO $ do
		//     simpleStatus res1 `shouldBe` status200
		//     simpleStatus res2 `shouldBe` status200
		//
		//   let dur1Positive = maybe False (>= 0) jwtDur1
		//   let dur2Positive = maybe False (>= 0) jwtDur2
		//   liftIO $ do
		//     dur1Positive `shouldBe` True
		//     dur2Positive `shouldBe` True
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		{
			Description: "server-timing duration is exposed for JWT without expiry",
			Method:      "GET",
			Query:       "/authors_only",
			Skip:        "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability); the expectations are not converted",
		},
		// it "JWT cache does not break requests with server-timing disabled" $ do
		//   let
		// TODO(hspec2go): not a request with shouldRespondWith (assertions in Haskell code)
		//     jwtPayload = [json|{ "role": "postgrest_test_author" }|]
		//     auth = authHeaderJWT $ generateJWT jwtPayload
		//
		//   request methodGet "/authors_only" [auth] "" `shouldRespondWith` 200
		// TODO(hspec2go): header not converted: authHeaderJWT $ generateJWT jwtPayload
		{
			Description: "JWT cache does not break requests with server-timing disabled",
			Method:      "GET",
			Query:       "/authors_only",
			Skip:        "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability); the expectations are not converted",
		},
		//   request methodGet "/authors_only" [auth] "" `shouldRespondWith` 200
		// TODO(hspec2go): header not converted: authHeaderJWT $ generateJWT jwtPayload
		{
			Description: "JWT cache does not break requests with server-timing disabled",
			Method:      "GET",
			Query:       "/authors_only",
			Skip:        "PostgREST 16.4: the Server-Timing and trace headers; card 32501 (observability); the expectations are not converted",
		},
	}
	test.Execute(t, testConfig, tests)
}
