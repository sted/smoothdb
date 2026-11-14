package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_RpcPreRequestGucs holds the cases of RpcPreRequestGucsSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_RpcPreRequestGucs(t *testing.T) {
	tests := []test.Test{
		// describe "GUC headers on all methods via pre-request" $ do
		//   it "succeeds setting the headers on POST" $
		//     post
		//       "/items"
		//       [json|[{"id": 11111}]|]
		//       `shouldRespondWith` ""
		//         { matchStatus = 201
		//         , matchHeaders =
		//             [ matchHeaderAbsent hContentType
		//             , "X-Custom-Header" <:> "mykey=myval"
		//             ]
		//         }
		{
			Description:     "succeeds setting the headers on POST",
			Method:          "POST",
			Query:           "/items",
			Body:            `[{"id": 11111}]`,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "X-Custom-Header": "mykey=myval"},
			Status:          201,
			Skip:            "PostgREST 16.4: the db-pre-request function sets response headers; card 31811 (pre-request) and card 32488 (response shaping)",
		},
		//   it "succeeds setting the headers on GET and HEAD" $ do
		//     request
		//       methodGet
		//       "/items?id=eq.1"
		//       [("User-Agent", "MSIE 6.0")]
		//       ""
		//       `shouldRespondWith` [json|[{"id": 1}]|]
		//         { matchHeaders = ["Cache-Control" <:> "no-cache, no-store, must-revalidate"]
		//         }
		{
			Description:     "succeeds setting the headers on GET and HEAD",
			Method:          "GET",
			Query:           "/items?id=eq.1",
			Headers:         test.Headers{"User-Agent": {"MSIE 6.0"}},
			Expected:        `[{"id": 1}]`,
			ExpectedHeaders: map[string]string{"Cache-Control": "no-cache, no-store, must-revalidate"},
			Status:          200,
			Skip:            "PostgREST 16.4: the db-pre-request function sets response headers; card 31811 (pre-request) and card 32488 (response shaping)",
		},
		//
		//     request
		//       methodHead
		//       "/items?id=eq.1"
		//       [("User-Agent", "MSIE 7.0")]
		//       ""
		//       `shouldRespondWith` ""
		//         { matchHeaders =
		//             [ matchContentTypeJson
		//             , "Cache-Control" <:> "no-cache, no-store, must-revalidate"
		//             ]
		//         }
		{
			Description:     "succeeds setting the headers on GET and HEAD",
			Method:          "HEAD",
			Query:           "/items?id=eq.1",
			Headers:         test.Headers{"User-Agent": {"MSIE 7.0"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8", "Cache-Control": "no-cache, no-store, must-revalidate"},
			Status:          200,
			Skip:            "PostgREST 16.4: the db-pre-request function sets response headers; card 31811 (pre-request) and card 32488 (response shaping)",
		},
		//
		//     request
		//       methodHead
		//       "/projects"
		//       [("Accept", "text/csv")]
		//       ""
		//       `shouldRespondWith` ""
		//         { matchHeaders =
		//             [ "Content-Type" <:> "text/csv; charset=utf-8"
		//             , "Content-Disposition" <:> "attachment; filename=projects.csv"
		//             ]
		//         }
		{
			Description:     "succeeds setting the headers on GET and HEAD",
			Method:          "HEAD",
			Query:           "/projects",
			Headers:         test.Headers{"Accept": {"text/csv"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "text/csv; charset=utf-8", "Content-Disposition": "attachment; filename=projects.csv"},
			Status:          200,
			Skip:            "PostgREST 16.4: the db-pre-request function sets response headers; card 31811 (pre-request) and card 32488 (response shaping)",
		},
		//   it "succeeds setting the headers on PATCH" $
		//     patch
		//       "/items?id=eq.1"
		//       [json|[{"id": 11111}]|]
		//       `shouldRespondWith` ""
		//         { matchStatus = 204
		//         , matchHeaders =
		//             [ matchHeaderAbsent hContentType
		//             , matchHeaderAbsent hContentLength
		//             , "X-Custom-Header" <:> "mykey=myval"
		//             ]
		//         }
		{
			Description:     "succeeds setting the headers on PATCH",
			Method:          "PATCH",
			Query:           "/items?id=eq.1",
			Body:            `[{"id": 11111}]`,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Content-Length": "", "X-Custom-Header": "mykey=myval"},
			Status:          204,
			Skip:            "PostgREST 16.4: the db-pre-request function sets response headers; card 31811 (pre-request) and card 32488 (response shaping)",
		},
		//   it "succeeds setting the headers on PUT" $
		//     put
		//       "/items?id=eq.1"
		//       [json|[{"id": 1}]|]
		//       `shouldRespondWith` ""
		//         { matchStatus = 204
		//         , matchHeaders =
		//             [ matchHeaderAbsent hContentType
		//             , matchHeaderAbsent hContentLength
		//             , "X-Custom-Header" <:> "mykey=myval"
		//             ]
		//         }
		{
			Description:     "succeeds setting the headers on PUT",
			Method:          "PUT",
			Query:           "/items?id=eq.1",
			Body:            `[{"id": 1}]`,
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Content-Length": "", "X-Custom-Header": "mykey=myval"},
			Status:          204,
			Skip:            "PostgREST 16.4: the db-pre-request function sets response headers; card 31811 (pre-request) and card 32488 (response shaping)",
		},
		//   it "succeeds setting the headers on DELETE" $
		//     delete "/items?id=eq.1"
		//       `shouldRespondWith` ""
		//         { matchStatus = 204
		//         , matchHeaders =
		//             [ matchHeaderAbsent hContentType
		//             , matchHeaderAbsent hContentLength
		//             , "X-Custom-Header" <:> "mykey=myval"
		//             ]
		//         }
		{
			Description:     "succeeds setting the headers on DELETE",
			Method:          "DELETE",
			Query:           "/items?id=eq.1",
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "", "Content-Length": "", "X-Custom-Header": "mykey=myval"},
			Status:          204,
			Skip:            "PostgREST 16.4: the db-pre-request function sets response headers; card 31811 (pre-request) and card 32488 (response shaping)",
		},
		//   it "can override the Content-Type header" $ do
		//     request
		//       methodHead
		//       "/clients?id=eq.1"
		//       []
		//       ""
		//       `shouldRespondWith` ""
		//         { matchStatus = 200
		//         , matchHeaders = ["Content-Type" <:> "application/custom+json"]
		//         }
		{
			Description:     "can override the Content-Type header",
			Method:          "HEAD",
			Query:           "/clients?id=eq.1",
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "application/custom+json"},
			Status:          200,
			Skip:            "PostgREST 16.4: the db-pre-request function sets response headers; card 31811 (pre-request) and card 32488 (response shaping)",
		},
		//     request
		//       methodHead
		//       "/rpc/getallprojects"
		//       []
		//       ""
		//       `shouldRespondWith` ""
		//         { matchStatus = 200
		//         , matchHeaders = ["Content-Type" <:> "application/custom+json"]
		//         }
		{
			Description:     "can override the Content-Type header",
			Method:          "HEAD",
			Query:           "/rpc/getallprojects",
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Type": "application/custom+json"},
			Status:          200,
			Skip:            "PostgREST 16.4: the db-pre-request function sets response headers; card 31811 (pre-request) and card 32488 (response shaping)",
		},
	}
	test.Execute(t, testConfig, tests)
}

// TestPostgREST_HttpHeader holds the cases of HttpHeaderSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_HttpHeader(t *testing.T) {
	tests := []test.Test{
		// describe "Test HTTP Custom Header" $
		//   it "Test default Vary header value is overridden in pre-request database function" $
		//     request
		//       methodGet
		//       "/projects"
		//       []
		//       ""
		//       `shouldRespondWith` ResponseMatcher
		//         { matchStatus = 200
		//         , matchBody = MatchBody (\_ _ -> Nothing) -- match any body
		//         , matchHeaders = ["Vary" <:> "X-Test-Accept"]
		//         }
		// TODO(hspec2go): expectation not converted: ResponseMatcher
		{
			Description: "Test default Vary header value is overridden in pre-request database function",
			Method:      "GET",
			Query:       "/projects",
			Skip:        "PostgREST 16.4: the db-pre-request function sets response headers; card 31811 (pre-request) and card 32488 (response shaping); the expectations are not converted",
		},
		// describe "Test HTTP Default Header" $
		//   it "Test default Vary header value matches default one" $
		//     request
		//       methodGet
		//       "/projects"
		//       []
		//       ""
		//       `shouldRespondWith` ResponseMatcher
		//         { matchStatus = 200
		//         , matchBody = MatchBody (\_ _ -> Nothing) -- match any body
		//         , matchHeaders = ["Vary" <:> "Accept, Prefer, Range"]
		//         }
		// TODO(hspec2go): expectation not converted: ResponseMatcher
		{
			Description: "Test default Vary header value matches default one",
			Method:      "GET",
			Query:       "/projects",
			Skip:        "PostgREST 16.4: the db-pre-request function sets response headers; card 31811 (pre-request) and card 32488 (response shaping); the expectations are not converted",
		},
	}
	test.Execute(t, testConfig, tests)
}
