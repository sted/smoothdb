package postgrest

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_Errors holds the cases of ErrorSpec.hs (PostgREST 16.4), converted by hspec2go.
// The JWT cases build their tokens in Haskell: they are written by hand.
func TestPostgREST_Errors(t *testing.T) {
	tests := []test.Test{
		// it "should return 500 for cardinality_violation" $
		//   get "/bad_subquery" `shouldRespondWith` 500
		{
			Description: "should return 500 for cardinality_violation",
			Method:      "GET",
			Query:       "/bad_subquery",
			Status:      500,
		},
		// it "should return 500 for statement too complex" $
		//   request
		//     methodPost
		//     "/infinite_inserts"
		//     []
		//     [json|{"id": 3, "name": "qwer"}|]
		//     `shouldRespondWith` [json|
		//       {"code": "54001",
		//        "details": null,
		//        "hint": "Increase the configuration parameter \"max_stack_depth\" (currently 2048kB), after ensuring the platform's stack depth limit is adequate.",
		//         "message": "stack depth limit exceeded"}|]
		//       { matchStatus = 500
		//       , matchHeaders = ["Content-Length" <:> "217"]
		//       }
		{
			Description: "should return 500 for statement too complex",
			Method:      "POST",
			Query:       "/infinite_inserts",
			Body:        `{"id": 3, "name": "qwer"}`,
			Expected: `{"code": "54001",
           "details": null,
           "hint": "Increase the configuration parameter \"max_stack_depth\" (currently 2048kB), after ensuring the platform's stack depth limit is adequate.",
            "message": "stack depth limit exceeded"}`,
			ExpectedHeaders: map[string]string{"Content-Length": "217"},
			Status:          500,
		},
		// context "includes the proxy-status header on the response" $ do
		//   it "works with ApiRequest error" $
		//     get "/invalid/nested/paths"
		//       `shouldRespondWith` [json| {"code":"PGRST125","details":null,"hint":null,"message":"Invalid path specified in request URL"} |]
		//         { matchStatus = 404
		//         , matchHeaders =
		//             [ "Proxy-Status" <:> "PostgREST; error=PGRST125"
		//             , "Content-Length" <:> "96"
		//             ]
		//         }
		{
			Description:     "works with ApiRequest error",
			Method:          "GET",
			Query:           "/invalid/nested/paths",
			Expected:        `{"code":"PGRST125","details":null,"hint":null,"message":"Invalid path specified in request URL"}`,
			ExpectedHeaders: map[string]string{"Proxy-Status": "PostgREST; error=PGRST125", "Content-Length": "96"},
			Status:          404,
			Skip:            "PostgREST 16.4: PGRST125 for a path with more than one segment, smoothdb answers 404 with a plain body; card 32494 (error taxonomy)",
		},
		//   it "works with SchemaCache error" $
		//     get "/non_existent_table"
		//       `shouldRespondWith` [json| {"code":"PGRST205","details":null,"hint":null,"message":"Could not find the table 'test.non_existent_table' in the schema cache"} |]
		//         { matchStatus = 404
		//         , matchHeaders =
		//             [ "Proxy-Status" <:> "PostgREST; error=PGRST205"
		//             , "Content-Length" <:> "129"
		//             ]
		//         }
		{
			Description:     "works with SchemaCache error",
			Method:          "GET",
			Query:           "/non_existent_table",
			Expected:        `{"code":"PGRST205","details":null,"hint":null,"message":"Could not find the table 'test.non_existent_table' in the schema cache"}`,
			ExpectedHeaders: map[string]string{"Proxy-Status": "PostgREST; error=PGRST205", "Content-Length": "129"},
			Status:          404,
		},
		//   it "works with Jwt error" $ do
		//     let auth = authHeaderJWT "ey9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.y4vZuu1dDdwAl0-S00MCRWRYMlJ5YAMSir6Es6WtWx0"
		//     request methodGet "/authors_only" [auth] ""
		//       `shouldRespondWith` [json| {"message":"Expected 3 parts in JWT; got 2","code":"PGRST301","hint":null,"details":null} |]
		//         { matchStatus = 401
		//         , matchHeaders =
		//             [ "Proxy-Status" <:> "PostgREST; error=PGRST301"
		//             , "Content-Length" <:> "89"
		//             ]
		//         }
		{
			Description:     "works with Jwt error",
			Method:          "GET",
			Query:           "/authors_only",
			Headers:         test.Headers{"Authorization": {"Bearer ey9zdGdyZXN0X3Rlc3RfYXV0aG9yIiwiaWQiOiJqZG9lIn0.y4vZuu1dDdwAl0-S00MCRWRYMlJ5YAMSir6Es6WtWx0"}},
			Expected:        `{"message":"Expected 3 parts in JWT; got 2","code":"PGRST301","hint":null,"details":null}`,
			ExpectedHeaders: map[string]string{"Proxy-Status": "PostgREST; error=PGRST301", "Content-Length": "89"},
			Status:          401,
		},
		//   it "works with raise sqlstate custom error" $
		//     get "/rpc/raise_pt402"
		//       `shouldRespondWith` [json| {"code":"PT402","details":"Quota exceeded","hint":"Upgrade your plan","message":"Payment Required"} |]
		//         { matchStatus = 402
		//         , matchHeaders =
		//             [ "Proxy-Status" <:> "PostgREST; error=PT402"
		//             , "Content-Length" <:> "99"
		//             ]
		//         }
		{
			Description:     "works with raise sqlstate custom error",
			Method:          "GET",
			Query:           "/rpc/raise_pt402",
			Expected:        `{"code":"PT402","details":"Quota exceeded","hint":"Upgrade your plan","message":"Payment Required"}`,
			ExpectedHeaders: map[string]string{"Proxy-Status": "PostgREST; error=PT402", "Content-Length": "99"},
			Status:          402,
		},
		//   it "works with sqlstate PGRST custom error" $
		//     get "/rpc/raise_sqlstate_test1"
		//       `shouldRespondWith` [json| {"code":"123","details":"DEF","hint":"XYZ","message":"ABC"} |]
		//         { matchStatus = 332
		//         , matchHeaders =
		//             [ "Proxy-Status" <:> "PostgREST; error=123"
		//             , "Content-Length" <:> "59"
		//             ]
		//         }
		{
			Description:     "works with sqlstate PGRST custom error",
			Method:          "GET",
			Query:           "/rpc/raise_sqlstate_test1",
			Expected:        `{"code":"123","details":"DEF","hint":"XYZ","message":"ABC"}`,
			ExpectedHeaders: map[string]string{"Proxy-Status": "PostgREST; error=123", "Content-Length": "59"},
			Status:          332,
			Skip:            "PostgREST 16.4: RAISE SQLSTATE 'PGRST' sets the status, the headers and the error body; card 32488 (response shaping)",
		},
		// context "show hint on PGRST205 table not found error" $ do
		//   it "show hint when similarity score is at least 75%" $ do
		//     get "/projectx" -- at least 75% similar to "projects"
		//       `shouldRespondWith` [json| {"code":"PGRST205","details":null,"hint":"Perhaps you meant the table 'test.projects'","message":"Could not find the table 'test.projectx' in the schema cache"} |]
		//         { matchStatus = 404
		//         , matchHeaders =
		//             [ "Proxy-Status" <:> "PostgREST; error=PGRST205"
		//             , "Content-Length" <:> "160"
		//             ]
		//         }
		{
			Description:     "show hint when similarity score is at least 75%",
			Method:          "GET",
			Query:           "/projectx",
			Expected:        `{"code":"PGRST205","details":null,"hint":"Perhaps you meant the table 'test.projects'","message":"Could not find the table 'test.projectx' in the schema cache"}`,
			ExpectedHeaders: map[string]string{"Proxy-Status": "PostgREST; error=PGRST205", "Content-Length": "160"},
			Status:          404,
		},
		//
		//     get "/projecxx" -- at least 75% similar to "projects"
		//       `shouldRespondWith` [json| {"code":"PGRST205","details":null,"hint":"Perhaps you meant the table 'test.projects'","message":"Could not find the table 'test.projecxx' in the schema cache"} |]
		//         { matchStatus = 404
		//         , matchHeaders =
		//             [ "Proxy-Status" <:> "PostgREST; error=PGRST205"
		//             , "Content-Length" <:> "160"
		//             ]
		//         }
		{
			Description:     "show hint when similarity score is at least 75%",
			Method:          "GET",
			Query:           "/projecxx",
			Expected:        `{"code":"PGRST205","details":null,"hint":"Perhaps you meant the table 'test.projects'","message":"Could not find the table 'test.projecxx' in the schema cache"}`,
			ExpectedHeaders: map[string]string{"Proxy-Status": "PostgREST; error=PGRST205", "Content-Length": "160"},
			Status:          404,
		},
		//   it "don't show hint when similarity score is less than 75%" $
		//     get "/projxxxx" -- less than 75% similar to "projects"
		//       `shouldRespondWith` [json| {"code":"PGRST205","details":null,"hint":null,"message":"Could not find the table 'test.projxxxx' in the schema cache"} |]
		//         { matchStatus = 404
		//         , matchHeaders =
		//             [ "Proxy-Status" <:> "PostgREST; error=PGRST205"
		//             , "Content-Length" <:> "119"
		//             ]
		//         }
		{
			Description:     "don't show hint when similarity score is less than 75%",
			Method:          "GET",
			Query:           "/projxxxx",
			Expected:        `{"code":"PGRST205","details":null,"hint":null,"message":"Could not find the table 'test.projxxxx' in the schema cache"}`,
			ExpectedHeaders: map[string]string{"Proxy-Status": "PostgREST; error=PGRST205", "Content-Length": "119"},
			Status:          404,
		},
		// context "JWT Errors" $ do
		//   it "error on jwt encoded with wrong secret" $ do
		//     let
		//       jwtPayload = [json|{}|]
		//       auth = authHeaderJWT $ generateJWTWithSecret jwtPayload "wrong secret"
		//     request methodGet "/authors_only" [auth] ""
		//       `shouldRespondWith` [json|{
		//         "code":"PGRST301",
		//         "details":"None of the keys was able to decode the JWT",
		//         "hint":null,
		//         "message":"No suitable key or wrong key type"
		//       }|]
		//         { matchStatus = 401
		//         }
		// (the token is signed in Go: see signJWT)
		{
			Description: "error on jwt encoded with wrong secret",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer " + signJWT(t, `{}`, "wrong secret")}},
			Expected:    `{"code":"PGRST301","details":"None of the keys was able to decode the JWT","hint":null,"message":"No suitable key or wrong key type"}`,
			Status:      401,
		},
		//   it "when role does not exist" $ do
		//     let
		//       jwtPayload = [json|{ "role": "not existing" }|]
		//       auth = authHeaderJWT $ generateJWT jwtPayload
		//     request methodGet "/authors_only" [auth] ""
		//       `shouldRespondWith` [json|{
		//         "code":"22023",
		//         "details":null,
		//         "hint":null,
		//         "message":"role \"not existing\" does not exist"
		//       }|]
		//         { matchStatus = 401
		//         }
		// (the token is signed in Go: see signJWT)
		{
			Description: "when role does not exist",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer " + signJWT(t, `{"role":"not existing"}`, suiteSecret)}},
			Expected:    `{"code":"22023","details":null,"hint":null,"message":"role \"not existing\" does not exist"}`,
			Status:      401,
		},
		//   context "we allow 30 seconds clock skew" $ do
		//     it "it should return error if expired" $ do
		//       currentTime <- liftIO $ relativeSeconds (-35)
		//       let
		//         jwtPayload = [json|{ "exp": #{currentTime} }|]
		//         auth = authHeaderJWT $ generateJWT jwtPayload
		//       request methodGet "/authors_only" [auth] ""
		//         `shouldRespondWith` [json|{
		//           "code":"PGRST303",
		//           "details":null,
		//           "hint":null,
		//           "message":"JWT expired"
		//         }|]
		//           { matchStatus = 401
		//           }
		// (the token is signed in Go: see signJWT)
		{
			Description: "it should return error if expired",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer " + signJWT(t, fmt.Sprintf(`{"exp":%d}`, time.Now().Unix()-35), suiteSecret)}},
			Expected:    `{"code":"PGRST303","details":null,"hint":null,"message":"JWT expired"}`,
			Status:      401,
		},
		//     it "it should return error if used before it is valid" $ do
		//       currentTime <- liftIO $ relativeSeconds 35
		//       let
		//         jwtPayload = [json|{ "nbf": #{currentTime} }|]
		//         auth = authHeaderJWT $ generateJWT jwtPayload
		//       request methodGet "/authors_only" [auth] ""
		//         `shouldRespondWith` [json|{
		//           "code":"PGRST303",
		//           "details":null,
		//           "hint":null,
		//           "message":"JWT not yet valid"
		//         }|]
		//           { matchStatus = 401
		//           }
		// (the token is signed in Go: see signJWT)
		{
			Description: "it should return error if used before it is valid",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer " + signJWT(t, fmt.Sprintf(`{"nbf":%d}`, time.Now().Unix()+35), suiteSecret)}},
			Expected:    `{"code":"PGRST303","details":null,"hint":null,"message":"JWT not yet valid"}`,
			Status:      401,
		},
		//     it "it should return error if issued at future" $ do
		//       currentTime <- liftIO $ relativeSeconds 35
		//       let
		//         jwtPayload = [json|{ "iat": #{currentTime} }|]
		//         auth = authHeaderJWT $ generateJWT jwtPayload
		//       request methodGet "/authors_only" [auth] ""
		//         `shouldRespondWith` [json|{
		//           "code":"PGRST303",
		//           "details":null,
		//           "hint":null,
		//           "message":"JWT issued at future"
		//         }|]
		//           { matchStatus = 401
		//           }
		// (the token is signed in Go: see signJWT)
		{
			Description: "it should return error if issued at future",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer " + signJWT(t, fmt.Sprintf(`{"iat":%d}`, time.Now().Unix()+35), suiteSecret)}},
			Expected:    `{"code":"PGRST303","details":null,"hint":null,"message":"JWT issued at future"}`,
			Status:      401,
		},
		// it "when partial token is provided" $ do
		//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.bm90IGFuIG9iamVjdA"
		//   request methodGet "/authors_only" [auth] ""
		//     `shouldRespondWith` [json|{
		//       "code":"PGRST301",
		//       "details":null,
		//       "hint":null,
		//       "message":"Expected 3 parts in JWT; got 2"
		//     }|]
		//       { matchStatus = 401
		//       }
		{
			Description: "when partial token is provided",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.bm90IGFuIG9iamVjdA"}},
			Expected: `{
          "code":"PGRST301",
          "details":null,
          "hint":null,
          "message":"Expected 3 parts in JWT; got 2"
        }`,
			Status: 401,
		},
		// it "when token is complete but random characters" $ do
		//   let auth = authHeaderJWT "quifquirndsjagnrgniur.fonvoienqhhdj.iuqvnvhojah"
		//   request methodGet "/authors_only" [auth] ""
		//     `shouldRespondWith` [json|{
		//       "code":"PGRST301",
		//       "details":null,
		//       "hint":null,
		//       "message":"JWT cryptographic operation failed"
		//     }|]
		//       { matchStatus = 401
		//       }
		{
			Description: "when token is complete but random characters",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer quifquirndsjagnrgniur.fonvoienqhhdj.iuqvnvhojah"}},
			Expected: `{
          "code":"PGRST301",
          "details":null,
          "hint":null,
          "message":"JWT cryptographic operation failed"
        }`,
			Status: 401,
		},
		// it "when token with algorithm 'none' is used" $ do
		//   let auth = authHeaderJWT "eyJ0eXAiOiJKV1QiLCJhbGciOiJub25lIn0.e30.yOBhlOIqn56T-4NvyEXCjfi3UmyQZ-BzXtePMO2NgRI"
		//   request methodGet "/authors_only" [auth] ""
		//     `shouldRespondWith` [json|{
		//       "code":"PGRST301",
		//       "details":"JWT is unsecured but expected 'alg' was not 'none'",
		//       "hint":null,
		//       "message":"Wrong or unsupported encoding algorithm"
		//     }|]
		//       { matchStatus = 401
		//       }
		{
			Description: "when token with algorithm 'none' is used",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJub25lIn0.e30.yOBhlOIqn56T-4NvyEXCjfi3UmyQZ-BzXtePMO2NgRI"}},
			Expected: `{
          "code":"PGRST301",
          "details":"JWT is unsecured but expected 'alg' was not 'none'",
          "hint":null,
          "message":"Wrong or unsupported encoding algorithm"
        }`,
			Status: 401,
		},
	}
	test.Execute(t, testConfig, tests)
}

// TestPostgREST_ErrorsAudience holds the case of ErrorSpec.hs that runs with
// jwt-aud = "spec tests": it runs on a variant server with JWTAud.
func TestPostgREST_ErrorsAudience(t *testing.T) {
	tests := []test.Test{
		// withConfig baseCfg{configJwtAudience = Just "spec tests"} $ describe "Test JWT Audience error" $ do
		// it "it should return error if JWT not in audience" $ do
		//   let
		//     jwtPayload = [json|{ "aud": "not set" }|]
		//     auth = authHeaderJWT $ generateJWT jwtPayload
		//   request methodGet "/authors_only" [auth] ""
		//     `shouldRespondWith` [json|{
		//       "code":"PGRST303",
		//       "details":null,
		//       "hint":null,
		//       "message":"JWT not in audience"
		//     }|]
		//       { matchStatus = 401
		//       }
		// (the token is signed in Go: see signJWT)
		{
			Description: "it should return error if JWT not in audience",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer " + signJWT(t, `{"aud":"not set"}`, suiteSecret)}},
			Expected:    `{"code":"PGRST303","details":null,"hint":null,"message":"JWT not in audience"}`,
			Status:      401,
		},
	}
	withVariant(t, map[string]any{"JWTAud": "spec tests"}, false, func(cfg test.Config) {
		test.Execute(t, cfg, tests)
	})
}

// TestPostgREST_ErrorsMinimal holds the case of ErrorSpec.hs that runs with
// client-error-verbosity = minimal: it runs on a variant server with
// ErrorVerbosity minimal. VerboseErrors false: a config file can set it,
// and true contradicts minimal.
func TestPostgREST_ErrorsMinimal(t *testing.T) {
	tests := []test.Test{
		// withConfig baseCfg{configClientErrorVerbosity = Minimal} $
		// describe "Test client-error-verbosity config" $
		//   it "hides details and hint when set to 'minimal'" $
		//     request methodGet "/itemsx" [] ""
		//       `shouldRespondWith` [json|{
		//       "code":"PGRST205",
		//       "message":"Could not find the table 'test.itemsx' in the schema cache"
		//     }|]
		//         { matchStatus = 404
		//         }
		{
			Description: "hides details and hint when set to 'minimal'",
			Method:      "GET",
			Query:       "/itemsx",
			Expected: `{
          "code":"PGRST205",
          "message":"Could not find the table 'test.itemsx' in the schema cache"
        }`,
			Status: 404,
		},
	}
	withVariant(t, map[string]any{"ErrorVerbosity": "minimal", "VerboseErrors": false}, false, func(cfg test.Config) {
		test.Execute(t, cfg, tests)
	})
}

// suiteSecret is the JWT secret of the suite, the secret of the spec
// (generateSecret in SpecHelper.hs).
const suiteSecret = "reallyreallyreallyreallyverysafe"

// signJWT signs payload with HS256 and secret, as generateJWT and
// generateJWTWithSecret of SpecHelper.hs.
func signJWT(t *testing.T, payload, secret string) string {
	t.Helper()
	enc := base64.RawURLEncoding
	unsigned := enc.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." + enc.EncodeToString([]byte(payload))
	sig, err := jwt.SigningMethodHS256.Sign(unsigned, []byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + enc.EncodeToString(sig)
}
