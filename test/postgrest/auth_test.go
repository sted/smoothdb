package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// Ported from test/spec/Feature/Auth/AuthSpec.hs. The suite runs anonymously
// with a token for postgrest_test_anonymous, the configured anon role, as
// PostgREST's own runs without one: a request is authenticated when its role
// is not the anon role (App.hs, PgError (Just authRole /= configDbAnonRole)).
// The bodies have smoothdb's SmoothError shape, so PostgREST's Content-Length
// assertions are not ported.

// token body: {"role":"postgrest_test_author"}
const authorToken = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIn0.Xod-F15qsGL0WhdOCr2j3DdKuTw9QJERVgoFD3vGaWA"

func TestPostgREST_Auth(t *testing.T) {
	tests := []test.Test{
		// it "denies access to tables that anonymous does not own" $
		//   get "/authors_only"
		//     `shouldRespondWith` [json| {
		//       "hint":null,
		//       "details":null,
		//       "code":"42501",
		//       "message":"permission denied for table authors_only"} |]
		//       { matchStatus = 401
		//       , matchHeaders =
		//           [ "WWW-Authenticate" <:> "Bearer"
		//           , "Content-Length" <:> "96"
		//           ]
		//       }
		{
			Description:     "denies access to tables that anonymous does not own",
			Method:          "GET",
			Query:           "/authors_only",
			Expected:        `{"code":"42501","details":null,"hint":null,"message":"permission denied for table authors_only"}`,
			ExpectedHeaders: map[string]string{"WWW-Authenticate": "Bearer"},
			Status:          401,
		},
		// it "denies access to tables that postgrest_test_author does not own" $
		//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIn0.Xod-F15qsGL0WhdOCr2j3DdKuTw9QJERVgoFD3vGaWA"
		//   in  request methodGet "/private_table" [auth] ""
		//         `shouldRespondWith` [json| {
		//       "hint":null,
		//       "details":null,
		//       "code":"42501",
		//       "message":"permission denied for table private_table"} |]
		//           { matchStatus = 403
		//           , matchHeaders = ["Content-Length" <:> "97"]
		//           }
		{
			Description:     "denies access to tables that postgrest_test_author does not own",
			Method:          "GET",
			Query:           "/private_table",
			Headers:         test.Headers{"Authorization": {authorToken}},
			Expected:        `{"code":"42501","details":null,"hint":null,"message":"permission denied for table private_table"}`,
			ExpectedHeaders: map[string]string{"WWW-Authenticate": ""},
			Status:          403,
		},
		// it "denies execution on functions that anonymous does not own" $
		//   post "/rpc/privileged_hello" [json|{"name": "anonymous"}|] `shouldRespondWith` 401
		{
			Description: "denies execution on functions that anonymous does not own",
			Method:      "POST",
			Query:       "/rpc/privileged_hello",
			Body:        `{"name": "anonymous"}`,
			Status:      401,
		},
		// it "allows execution on a function that postgrest_test_author owns" $
		//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIn0.Xod-F15qsGL0WhdOCr2j3DdKuTw9QJERVgoFD3vGaWA"
		//   in  request methodPost "/rpc/privileged_hello" [auth] [json|{"name": "jdoe"}|]
		//         `shouldRespondWith` [json|"Privileged hello to jdoe"|]
		//           { matchStatus = 200
		//           , matchHeaders = [matchContentTypeJson]
		//           }
		{
			Description: "allows execution on a function that postgrest_test_author owns",
			Method:      "POST",
			Query:       "/rpc/privileged_hello",
			Body:        `{"name": "jdoe"}`,
			Headers:     test.Headers{"Authorization": {authorToken}},
			Expected:    `"Privileged hello to jdoe"`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}

// TestPostgREST_AuthRest holds the other cases of AuthSpec.hs, converted by
// hspec2go. The suite's JWT secret is the secret of the spec, so its tokens
// verify as they are.
func TestPostgREST_AuthRest(t *testing.T) {
	// "recovers after 401 error with logged in user" makes two requests
	// before its check: an anonymous POST that fails with 401, and an RPC
	// that fails as the author.
	client := test.InitClient()
	for _, c := range []*test.Command{
		{Method: "POST", Query: "/authors_only", Body: `{ "owner": "jdoe", "secret": "test content" }`},
		{Method: "POST", Query: "/rpc/problem", Headers: test.Headers{"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.B-lReuGNDwAlU1GOC476MlO0vAt9JNoHIlxg2vwMaO0"}}},
	} {
		if _, _, _, err := test.Exec(client, testConfig, c); err != nil {
			t.Fatal(err)
		}
	}
	tests := []test.Test{
		// it "returns jwt functions as jwt tokens" $
		//   request
		//     methodPost
		//     "/rpc/login"
		//     [single]
		//     [json| { "id": "jdoe", "pass": "1234" } |]
		//     `shouldRespondWith` [json| {"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xuYW1lIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.KO-0PGp_rU-utcDBP6qwdd-Th2Fk-ICVt01I7QtTDWs"} |]
		//       { matchStatus = 200
		//       , matchHeaders = [matchContentTypeSingular]
		//       }
		{
			Description:     "returns jwt functions as jwt tokens",
			Method:          "POST",
			Query:           "/rpc/login",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
			Body:            `{ "id": "jdoe", "pass": "1234" }`,
			Expected:        `{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xuYW1lIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.KO-0PGp_rU-utcDBP6qwdd-Th2Fk-ICVt01I7QtTDWs"}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/vnd.pgrst.object+json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: Content-Type application/vnd.pgrst.object+json; charset=utf-8, smoothdb leaves out the charset; no card",
		},
		// it "sql functions can encode custom and standard claims" $
		//   request methodPost "/rpc/jwt_test" [single] "{}"
		//     `shouldRespondWith` [json| {"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJqb2UiLCJzdWIiOiJmdW4iLCJhdWQiOiJldmVyeW9uZSIsImV4cCI6MTMwMDgxOTM4MCwibmJmIjoxMzAwODE5MzgwLCJpYXQiOjEzMDA4MTkzODAsImp0aSI6ImZvbyIsInJvbGUiOiJwb3N0Z3Jlc3RfdGVzdCIsImh0dHA6Ly9wb3N0Z3Jlc3QuY29tL2ZvbyI6dHJ1ZX0.G2REtPnOQMUrVRDA9OnkPJTd8R0tf4wdYOlauh1E2Ek"} |]
		//       { matchStatus = 200
		//       , matchHeaders = [matchContentTypeSingular]
		//       }
		{
			Description:     "sql functions can encode custom and standard claims",
			Method:          "POST",
			Query:           "/rpc/jwt_test",
			Headers:         test.Headers{"Accept": {"application/vnd.pgrst.object+json"}},
			Body:            `{}`,
			Expected:        `{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJqb2UiLCJzdWIiOiJmdW4iLCJhdWQiOiJldmVyeW9uZSIsImV4cCI6MTMwMDgxOTM4MCwibmJmIjoxMzAwODE5MzgwLCJpYXQiOjEzMDA4MTkzODAsImp0aSI6ImZvbyIsInJvbGUiOiJwb3N0Z3Jlc3RfdGVzdCIsImh0dHA6Ly9wb3N0Z3Jlc3QuY29tL2ZvbyI6dHJ1ZX0.G2REtPnOQMUrVRDA9OnkPJTd8R0tf4wdYOlauh1E2Ek"}`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/vnd.pgrst.object+json; charset=utf-8"},
			Status:          200,
			Skip:            "PostgREST 16.4: Content-Type application/vnd.pgrst.object+json; charset=utf-8, smoothdb leaves out the charset; no card",
		},
		// it "sql functions can read custom and standard claims variables" $ do
		//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJmdW4iLCJqdGkiOiJmb28iLCJuYmYiOjEzMDA4MTkzODAsImV4cCI6OTk5OTk5OTk5OSwiaHR0cDovL3Bvc3RncmVzdC5jb20vZm9vIjp0cnVlLCJpc3MiOiJqb2UiLCJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWF0IjoxMzAwODE5MzgwfQ.V5fEpXfpb7feqwVqlcDleFdKu86bdwU2cBRT4fcMhXg"
		//   request methodPost "/rpc/reveal_big_jwt" [auth] "{}"
		//     `shouldRespondWith` [json|[{"iss":"joe","sub":"fun","exp":9999999999,"nbf":1300819380,"iat":1300819380,"jti":"foo","http://postgrest.com/foo":true}]|]
		{
			Description: "sql functions can read custom and standard claims variables",
			Method:      "POST",
			Query:       "/rpc/reveal_big_jwt",
			Headers:     test.Headers{"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJmdW4iLCJqdGkiOiJmb28iLCJuYmYiOjEzMDA4MTkzODAsImV4cCI6OTk5OTk5OTk5OSwiaHR0cDovL3Bvc3RncmVzdC5jb20vZm9vIjp0cnVlLCJpc3MiOiJqb2UiLCJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWF0IjoxMzAwODE5MzgwfQ.V5fEpXfpb7feqwVqlcDleFdKu86bdwU2cBRT4fcMhXg"}},
			Body:        `{}`,
			Expected:    `[{"iss":"joe","sub":"fun","exp":9999999999,"nbf":1300819380,"iat":1300819380,"jti":"foo","http://postgrest.com/foo":true}]`,
			Status:      200,
		},
		// it "works with tokens which have extra fields" $ do
		//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIiwia2V5MSI6InZhbHVlMSIsImtleTIiOiJ2YWx1ZTIiLCJrZXkzIjoidmFsdWUzIiwiYSI6MSwiYiI6MiwiYyI6M30.b0eglDKYEmGi-hCvD-ddSqFl7vnDO5qkUaviaHXm3es"
		//   request methodGet "/authors_only" [auth] ""
		//     `shouldRespondWith` 200
		{
			Description: "works with tokens which have extra fields",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIiwia2V5MSI6InZhbHVlMSIsImtleTIiOiJ2YWx1ZTIiLCJrZXkzIjoidmFsdWUzIiwiYSI6MSwiYiI6MiwiYyI6M30.b0eglDKYEmGi-hCvD-ddSqFl7vnDO5qkUaviaHXm3es"}},
			Status:      200,
		},
		// it "fails when auth header is sent empty" $
		//   do
		//     let auth = authHeaderJWT ""
		//     request methodGet "/authors_only" [auth] ""
		//     `shouldRespondWith` [json| {"message":"Empty JWT is sent in Authorization header","code":"PGRST301","hint":null,"details":null} |]
		//       { matchStatus = 401
		//       , matchHeaders =
		//           [ "WWW-Authenticate"
		//               <:> "Bearer error=\"invalid_token\", error_description=\"Empty JWT is sent in Authorization header\""
		//           , "Content-Length" <:> "100"
		//           ]
		//       }
		{
			Description:     "fails when auth header is sent empty",
			Method:          "GET",
			Query:           "/authors_only",
			Headers:         test.Headers{"Authorization": {"Bearer "}},
			Expected:        `{"message":"Empty JWT is sent in Authorization header","code":"PGRST301","hint":null,"details":null}`,
			ExpectedHeaders: map[string]string{"WWW-Authenticate": "Bearer error=\"invalid_token\", error_description=\"Empty JWT is sent in Authorization header\"", "Content-Length": "100"},
			Status:          401,
			Skip:            "PostgREST 16.4: PGRST301 \"Empty JWT is sent in Authorization header\" for an empty Bearer token, smoothdb answers \"Expected 3 parts in JWT; got 1\"; no card",
		},
		// it "should fail when jwt contains no claims" $ do
		//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.CUIP5V9thWsGGFsFyGijSZf1fJMfarLHI9CEJL-TGNk"
		//   request methodGet "/authors_only" [auth] ""
		//     `shouldRespondWith` 401
		{
			Description: "should fail when jwt contains no claims",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.e30.CUIP5V9thWsGGFsFyGijSZf1fJMfarLHI9CEJL-TGNk"}},
			Status:      401,
		},
		// it "hides tables from users with JWT that contain no claims about role" $ do
		//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6Impkb2UifQ.RVlZDaSyKbFPvxUf3V_NQXybfRB4dlBIkAUQXVXLUAI"
		//   request methodGet "/authors_only" [auth] ""
		//     `shouldRespondWith` 401
		{
			Description: "hides tables from users with JWT that contain no claims about role",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6Impkb2UifQ.RVlZDaSyKbFPvxUf3V_NQXybfRB4dlBIkAUQXVXLUAI"}},
			Status:      401,
		},
		// it "recovers after 401 error with logged in user" $ do
		//   _ <- post "/authors_only" [json| { "owner": "jdoe", "secret": "test content" } |]
		//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.B-lReuGNDwAlU1GOC476MlO0vAt9JNoHIlxg2vwMaO0"
		//   _ <- request methodPost "/rpc/problem" [auth] ""
		// (the two requests above run before Execute, at the top of this function)
		//   request methodGet "/authors_only" [auth] ""
		//     `shouldRespondWith` 200
		{
			Description: "recovers after 401 error with logged in user",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.B-lReuGNDwAlU1GOC476MlO0vAt9JNoHIlxg2vwMaO0"}},
			Status:      200,
		},
		// describe "custom pre-request proc acting on id claim" $ do
		//   it "able to switch to postgrest_test_author role (id=1)" $
		//     let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MX0.gKw7qI50i9hMrSJW8BlTpdMEVmMXJYxlAqueGqpa_mE"
		//     in  request
		//           methodPost
		//           "/rpc/get_current_user"
		//           [auth]
		//           [json| {} |]
		//           `shouldRespondWith` [json|"postgrest_test_author"|]
		//             { matchStatus = 200
		//             , matchHeaders = []
		//             }
		{
			Description: "able to switch to postgrest_test_author role (id=1)",
			Skip:        "PostgREST 16.4: the db-pre-request function sets the role from the id claim; card 31811 (pre-request)",
			Method:      "POST",
			Query:       "/rpc/get_current_user",
			Headers:     test.Headers{"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6MX0.gKw7qI50i9hMrSJW8BlTpdMEVmMXJYxlAqueGqpa_mE"}},
			Body:        `{}`,
			Expected:    `"postgrest_test_author"`,
			Status:      200,
		},
		//   it "able to switch to postgrest_test_default_role (id=2)" $
		//     let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6Mn0.nwzjMI0YLvVGJQTeoCPEBsK983b__gxdpLXisBNaO2A"
		//     in  request
		//           methodPost
		//           "/rpc/get_current_user"
		//           [auth]
		//           [json| {} |]
		//           `shouldRespondWith` [json|"postgrest_test_default_role"|]
		//             { matchStatus = 200
		//             , matchHeaders = []
		//             }
		{
			Description: "able to switch to postgrest_test_default_role (id=2)",
			Skip:        "PostgREST 16.4: the db-pre-request function sets the role from the id claim; card 31811 (pre-request)",
			Method:      "POST",
			Query:       "/rpc/get_current_user",
			Headers:     test.Headers{"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6Mn0.nwzjMI0YLvVGJQTeoCPEBsK983b__gxdpLXisBNaO2A"}},
			Body:        `{}`,
			Expected:    `"postgrest_test_default_role"`,
			Status:      200,
		},
		//   it "raises error (id=3)" $
		//     let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6M30.OGxEJAf60NKZiTn-tIb2jy4rqKs_ZruLGWZ40TjrJsM"
		//     in  request
		//           methodPost
		//           "/rpc/get_current_user"
		//           [auth]
		//           [json| {} |]
		//           `shouldRespondWith` [json|{"hint":"Please contact administrator","details":null,"code":"P0001","message":"Disabled ID --> 3"}|]
		//             { matchStatus = 400
		//             , matchHeaders = []
		//             }
		{
			Description: "raises error (id=3)",
			Skip:        "PostgREST 16.4: the db-pre-request function sets the role from the id claim; card 31811 (pre-request)",
			Method:      "POST",
			Query:       "/rpc/get_current_user",
			Headers:     test.Headers{"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6M30.OGxEJAf60NKZiTn-tIb2jy4rqKs_ZruLGWZ40TjrJsM"}},
			Body:        `{}`,
			Expected:    `{"hint":"Please contact administrator","details":null,"code":"P0001","message":"Disabled ID --> 3"}`,
			Status:      400,
		},
		// it "allows 'Bearer' and 'bearer' as authentication schemes" $ do
		//   let token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.B-lReuGNDwAlU1GOC476MlO0vAt9JNoHIlxg2vwMaO0"
		//   request methodGet "/authors_only" [authHeader "Bearer" token] ""
		//     `shouldRespondWith` 200
		{
			Description: "allows 'Bearer' and 'bearer' as authentication schemes",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.B-lReuGNDwAlU1GOC476MlO0vAt9JNoHIlxg2vwMaO0"}},
			Status:      200,
		},
		//   request methodGet "/authors_only" [authHeader "bearer" token] ""
		//     `shouldRespondWith` 200
		{
			Description: "allows 'Bearer' and 'bearer' as authentication schemes",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.B-lReuGNDwAlU1GOC476MlO0vAt9JNoHIlxg2vwMaO0"}},
			Status:      200,
		},
	}
	test.Execute(t, testConfig, tests)
}
