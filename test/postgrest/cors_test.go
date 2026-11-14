package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// Ported from test/spec/Feature/CorsSpec.hs. smoothdb answers CORS through
// rs/cors (server/cors.go), registered only when CORSAllowedOrigins is not
// empty. TestMain sets no CORS configuration, so each spec block runs on a
// variant server (see withVariant) with the CORS configuration of the block.
func TestPostgREST_Cors(t *testing.T) {

	const (
		// rs/cors answers a preflight 204, and aborts it, with no CORS
		// header at all, when a requested header is not in its fixed list
		// or is not written in lowercase
		corsPreflight = "PostgREST 16.4: a preflight answers 200, allowing every method and whatever headers it asks for; card 32497 (options cors)"
		corsNoOrigin  = "PostgREST 16.4: Access-Control-Allow-Origin: * on every response, a request without Origin too; card 32497 (options cors)"
	)

	tests := []test.Test{
		// withConfig baseCfg $ describe "CORS" $ do
		//   it "replies naively and permissively to preflight request" $
		//     request
		//       methodOptions
		//       "/"
		//       [ ("Accept", "*/*")
		//       , ("Origin", "http://example.com")
		//       , ("Access-Control-Request-Method", "POST")
		//       , ("Access-Control-Request-Headers", "Foo,Bar")
		//       ]
		//       ""
		//       `shouldRespondWith` ""
		//         { matchHeaders =
		//             [ "Access-Control-Allow-Origin" <:> "*"
		//             , "Access-Control-Allow-Methods" <:> "GET, POST, PATCH, PUT, DELETE, OPTIONS, HEAD"
		//             , "Access-Control-Allow-Headers" <:> "Authorization, Foo, Bar, Accept, Accept-Language, Content-Language"
		//             , "Access-Control-Max-Age" <:> "86400"
		//             ]
		//         }
		{
			Description: "replies naively and permissively to preflight request",
			Method:      "OPTIONS",
			Query:       "/",
			Headers: test.Headers{
				"Accept":                         {"*/*"},
				"Origin":                         {"http://example.com"},
				"Access-Control-Request-Method":  {"POST"},
				"Access-Control-Request-Headers": {"Foo,Bar"},
			},
			ExpectedEmpty: true,
			ExpectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST, PATCH, PUT, DELETE, OPTIONS, HEAD",
				"Access-Control-Allow-Headers": "Authorization, Foo, Bar, Accept, Accept-Language, Content-Language",
				"Access-Control-Max-Age":       "86400",
			},
			Status: 200,
			Skip:   corsPreflight,
		},
		//   it "exposes necesssary response headers to regular request" $
		//     request
		//       methodGet
		//       "/items"
		//       [("Origin", "http://example.com")]
		//       ""
		//       `shouldRespondWith` ResponseMatcher
		//         { matchStatus = 200
		//         , matchBody = MatchBody (\_ _ -> Nothing) -- match any body
		//         , matchHeaders =
		//             [ "Access-Control-Expose-Headers"
		//                 <:> "Content-Encoding, Content-Location, Content-Range, Content-Type, \
		//                     \Date, Location, Server, Transfer-Encoding, Range-Unit"
		//             ]
		//         }
		{
			Description: "exposes necesssary response headers to regular request",
			Query:       "/items",
			Headers:     test.Headers{"Origin": {"http://example.com"}},
			ExpectedHeaders: map[string]string{
				"Access-Control-Expose-Headers": "Content-Encoding, Content-Location, Content-Range, Content-Type, Date, Location, Server, Transfer-Encoding, Range-Unit",
			},
			Status: 200,
			Skip:   "PostgREST 16.4: Access-Control-Expose-Headers lists Content-Range, Location and the other response headers a client reads; card 32497 (options cors)",
		},
		//   it "allows INFO body through even with CORS request headers present to postflight request" $ do
		//     request
		//       methodOptions
		//       "/items"
		//       [ ("Host", "localhost:3000")
		//       , ("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.9; rv:32.0) Gecko/20100101 Firefox/32.0")
		//       , ("Origin", "http://localhost:8000")
		//       , ("Accept", "text/csv, */*; q=0.01")
		//       , ("Accept-Language", "en-US,en;q=0.5")
		//       , ("Accept-Encoding", "gzip, deflate")
		//       , ("Referer", "http://localhost:8000/")
		//       , ("Connection", "keep-alive")
		//       ]
		//       ""
		//       `shouldRespondWith` ""
		//         { matchHeaders = ["Access-Control-Allow-Origin" <:> "*"]
		//         }
		{
			Description: "allows INFO body through even with CORS request headers present to postflight request",
			Method:      "OPTIONS",
			Query:       "/items",
			Headers: test.Headers{
				"User-Agent":      {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.9; rv:32.0) Gecko/20100101 Firefox/32.0"},
				"Origin":          {"http://localhost:8000"},
				"Accept":          {"text/csv, */*; q=0.01"},
				"Accept-Language": {"en-US,en;q=0.5"},
				"Referer":         {"http://localhost:8000/"},
			},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Access-Control-Allow-Origin": "*"},
			Status:          200,
		},
		//     request
		//       methodOptions
		//       "/items"
		//       [("Accept", "application/json")]
		//       ""
		//       `shouldRespondWith` ""
		//         { matchHeaders = ["Access-Control-Allow-Origin" <:> "*"]
		//         }
		{
			Description:     "allows INFO body through even with CORS request headers present to postflight request",
			Method:          "OPTIONS",
			Query:           "/items",
			Headers:         test.Headers{"Accept": {"application/json"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Access-Control-Allow-Origin": "*"},
			Skip:            corsNoOrigin,
			Status:          200,
		},
		//     request
		//       methodOptions
		//       "/shops"
		//       [("Accept", "application/geo+json")]
		//       ""
		//       `shouldRespondWith` ""
		//         { matchHeaders = ["Access-Control-Allow-Origin" <:> "*"]
		//         }
		{
			Description:     "allows INFO body through even with CORS request headers present to postflight request",
			Method:          "OPTIONS",
			Query:           "/shops",
			Headers:         test.Headers{"Accept": {"application/geo+json"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Access-Control-Allow-Origin": "*"},
			Skip:            corsNoOrigin,
			Status:          200,
		},
	}
	withVariant(t, map[string]any{"CORSAllowedOrigins": []string{"*"}}, false, func(cfg test.Config) {
		test.Execute(t, cfg, tests)
	})

	tests = []test.Test{
		// withConfig baseCfg{configServerCorsAllowedOrigins = ["http://example.com", "http://example2.com"]} $
		//   describe "test preflight/non-preflight request and cors server allowed config" $ do
		//     it "OPTIONS preflight request should return Access-Control-Allow-Origin equal to origin" $
		//       request
		//         methodOptions
		//         "/items"
		//         [ ("Accept", "*/*")
		//         , ("Origin", "http://example.com")
		//         , ("Access-Control-Request-Method", "POST")
		//         , ("Access-Control-Request-Headers", "Content-Type")
		//         ]
		//         ""
		//         `shouldRespondWith` ResponseMatcher
		//           { matchStatus = 200
		//           , matchBody = MatchBody (\_ _ -> Nothing) -- match any body
		//           , matchHeaders =
		//               [ "Access-Control-Allow-Origin" <:> "http://example.com"
		//               , "Access-Control-Allow-Credentials" <:> "true"
		//               ]
		//           }
		{
			Description: "OPTIONS preflight request should return Access-Control-Allow-Origin equal to origin",
			Method:      "OPTIONS",
			Query:       "/items",
			Headers: test.Headers{
				"Accept":                         {"*/*"},
				"Origin":                         {"http://example.com"},
				"Access-Control-Request-Method":  {"POST"},
				"Access-Control-Request-Headers": {"Content-Type"},
			},
			ExpectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":      "http://example.com",
				"Access-Control-Allow-Credentials": "true",
			},
			Status: 200,
			Skip:   corsPreflight,
		},
		//     it "GET no preflight request should return Access-Control-Allow-Origin equal to origin" $
		//       request
		//         methodGet
		//         "/items"
		//         [ ("Accept", "*/*")
		//         , ("Origin", "http://example.com")
		//         ]
		//         ""
		//         `shouldRespondWith` ResponseMatcher
		//           { matchStatus = 200
		//           , matchBody = MatchBody (\_ _ -> Nothing) -- match any body
		//           , matchHeaders = ["Access-Control-Allow-Origin" <:> "http://example.com"]
		//           }
		{
			Description:     "GET no preflight request should return Access-Control-Allow-Origin equal to origin",
			Query:           "/items",
			Headers:         test.Headers{"Accept": {"*/*"}, "Origin": {"http://example.com"}},
			ExpectedHeaders: map[string]string{"Access-Control-Allow-Origin": "http://example.com"},
			Status:          200,
		},
		//     it "GET no preflight request should not return Access-Control-Allow-Origin" $
		//       request
		//         methodGet
		//         "/items"
		//         [ ("Accept", "*/*")
		//         , ("Origin", "http://invalid.com")
		//         ]
		//         ""
		//         `shouldRespondWith` ResponseMatcher
		//           { matchStatus = 200
		//           , matchBody = MatchBody (\_ _ -> Nothing) -- match any body
		//           , matchHeaders = [matchHeaderAbsent "Access-Control-Allow-Origin"]
		//           }
		{
			Description:     "GET no preflight request should not return Access-Control-Allow-Origin",
			Query:           "/items",
			Headers:         test.Headers{"Accept": {"*/*"}, "Origin": {"http://invalid.com"}},
			ExpectedHeaders: map[string]string{"Access-Control-Allow-Origin": ""},
			Status:          200,
		},
	}
	withVariant(t, map[string]any{
		"CORSAllowedOrigins":   []string{"http://example.com", "http://example2.com"},
		"CORSAllowCredentials": true,
	}, false, func(cfg test.Config) {
		test.Execute(t, cfg, tests)
	})

	tests = []test.Test{
		// withConfig baseCfg{configServerCorsAllowedOrigins = []} $
		//   describe "test preflight request with empty cors allowed origin config" $
		//     it "OPTIONS preflight request should allow all origins when config is not set or empty" $
		//       request
		//         methodOptions
		//         "/items"
		//         [ ("Accept", "*/*")
		//         , ("Origin", "http://anyorigin.com")
		//         , ("Access-Control-Request-Method", "POST")
		//         , ("Access-Control-Request-Headers", "Content-Type")
		//         ]
		//         ""
		//         `shouldRespondWith` ResponseMatcher
		//           { matchStatus = 200
		//           , matchBody = MatchBody (\_ _ -> Nothing) -- match any body
		//           , matchHeaders =
		//               [ "Access-Control-Allow-Origin" <:> "*"
		//               , matchHeaderValuePresent "Access-Control-Allow-Methods" "POST"
		//               ]
		//           }
		{
			Description: "OPTIONS preflight request should allow all origins when config is not set or empty",
			Method:      "OPTIONS",
			Query:       "/items",
			Headers: test.Headers{
				"Accept":                         {"*/*"},
				"Origin":                         {"http://anyorigin.com"},
				"Access-Control-Request-Method":  {"POST"},
				"Access-Control-Request-Headers": {"Content-Type"},
			},
			ExpectedHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "POST",
			},
			Status: 200,
			Skip:   "PostgREST 16.4: an empty CORSAllowedOrigins allows every origin, where smoothdb turns CORS off; card 32497 (options cors); and " + corsPreflight[len("PostgREST 16.4: "):],
		},
	}
	withVariant(t, map[string]any{"CORSAllowedOrigins": []string{}}, false, func(cfg test.Config) {
		test.Execute(t, cfg, tests)
	})
}
