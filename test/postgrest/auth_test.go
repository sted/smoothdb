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
			Expected:        `{"subsystem":"database","message":"permission denied for table authors_only","code":"42501","hint":"","details":"","position":0}`,
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
			Expected:        `{"subsystem":"database","message":"permission denied for table private_table","code":"42501","hint":"","details":"","position":0}`,
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
