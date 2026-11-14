package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_NoAnon holds the cases of NoAnonSpec.hs (PostgREST 16.4), converted by hspec2go.
// The spec runs without db-anon-role: the cases run on a variant server
// without Database.AnonRole and with AllowAnon false.
func TestPostgREST_NoAnon(t *testing.T) {
	tests := []test.Test{
		// it "behaves normally on attempted auth" $ do
		//   -- token body: { "role": "postgrest_test_author" }
		//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIn0.Xod-F15qsGL0WhdOCr2j3DdKuTw9QJERVgoFD3vGaWA"
		//   request
		//     methodGet
		//     "/authors_only"
		//     [auth]
		//     ""
		//     `shouldRespondWith` 200
		{
			Description: "behaves normally on attempted auth",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIn0.Xod-F15qsGL0WhdOCr2j3DdKuTw9QJERVgoFD3vGaWA"}},
			Status:      200,
		},
		// it "responds with error when user does not attempt auth" $
		//   get "/items"
		//     `shouldRespondWith` [json|
		//         {"hint": null,
		//          "details": null,
		//          "code": "PGRST302",
		//          "message":"Anonymous access is disabled"}|]
		//       { matchStatus = 401
		//       , matchHeaders = ["WWW-Authenticate" <:> "Bearer"]
		//       }
		{
			Description: "responds with error when user does not attempt auth",
			Method:      "GET",
			Query:       "/items",
			Expected: `{"hint": null,
           "details": null,
           "code": "PGRST302",
           "message":"Anonymous access is disabled"}`,
			ExpectedHeaders: map[string]string{"WWW-Authenticate": "Bearer"},
			Status:          401,
			Skip:            "PostgREST 16.4: PGRST302 has details null, smoothdb writes \"unauthorized access\"; no card",
		},
	}
	withVariant(t, map[string]any{"Database.AnonRole": "", "AllowAnon": false}, false, func(cfg test.Config) {
		test.Execute(t, withoutToken(cfg), tests)
	})
}

// TestPostgREST_NoJwtSecret holds the cases of NoJwtSecretSpec.hs (PostgREST 16.4), converted by hspec2go.
// The spec runs without jwt-secret: the cases run on a variant server
// without JWTSecret. smoothdb does not start without JWTSecret when
// authentication is on (server/config.go), so the cases are skipped.
func TestPostgREST_NoJwtSecret(t *testing.T) {
	tests := []test.Test{
		// it "responds with error on attempted auth" $ do
		//   -- token body: { "role": "postgrest_test_author" }
		//   let auth = authHeaderJWT "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIn0.Xod-F15qsGL0WhdOCr2j3DdKuTw9QJERVgoFD3vGaWA"
		//   request
		//     methodGet
		//     "/authors_only"
		//     [auth]
		//     ""
		//     `shouldRespondWith` [json|
		//       {"hint": null,
		//        "details": null,
		//        "code": "PGRST300",
		//        "message": "Server lacks JWT secret"}|]
		//       { matchStatus = 500
		//       }
		{
			Description: "responds with error on attempted auth",
			Method:      "GET",
			Query:       "/authors_only",
			Headers:     test.Headers{"Authorization": {"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJyb2xlIjoicG9zdGdyZXN0X3Rlc3RfYXV0aG9yIn0.Xod-F15qsGL0WhdOCr2j3DdKuTw9QJERVgoFD3vGaWA"}},
			Expected: `{"hint": null,
           "details": null,
           "code": "PGRST300",
           "message": "Server lacks JWT secret"}`,
			Status: 500,
			Skip:   "PostgREST 16.4: PGRST300 Server lacks JWT secret; smoothdb does not start without JWTSecret when authentication is on (a security check of the server configuration); no card",
		},
		// it "behaves normally when user does not attempt auth" $
		//   get "/items" `shouldRespondWith` 200
		{
			Description: "behaves normally when user does not attempt auth",
			Method:      "GET",
			Query:       "/items",
			Status:      200,
			Skip:        "PostgREST 16.4: a server without jwt-secret serves anonymous requests; smoothdb does not start without JWTSecret when authentication is on (a security check of the server configuration); no card",
		},
	}
	// both cases are skipped: smoothdb does not start without JWTSecret
	test.Execute(t, withoutToken(testConfig), tests)
}

// withoutToken returns cfg without the anonymous token that the suite sends
// on every request: the spec's requests without auth send no Authorization.
func withoutToken(cfg test.Config) test.Config {
	h := test.Headers{}
	for k, v := range cfg.CommonHeaders {
		if k != "Authorization" {
			h[k] = v
		}
	}
	cfg.CommonHeaders = h
	return cfg
}
