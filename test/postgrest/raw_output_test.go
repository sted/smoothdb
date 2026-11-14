package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_RawOutputTypes holds the cases of RawOutputTypesSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_RawOutputTypes(t *testing.T) {
	tests := []test.Test{
		// it "responds json to a GET request with Firefox Accept headers" $
		//   request methodGet "/items?id=eq.1" firefoxAcceptHdrs ""
		//     `shouldRespondWith` [json| [{"id":1}] |]
		//       { matchHeaders = ["Content-Type" <:> "application/json; charset=utf-8"]
		//       }
		{
			Description:     "responds json to a GET request with Firefox Accept headers",
			Method:          "GET",
			Query:           "/items?id=eq.1",
			Headers:         test.Headers{"Accept": {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"}},
			Expected:        `[{"id":1}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// it "responds json to a GET request with Chrome Accept headers" $
		//   request methodGet "/items?id=eq.1" chromeAcceptHdrs ""
		//     `shouldRespondWith` [json| [{"id":1}] |]
		//       { matchHeaders = ["Content-Type" <:> "application/json; charset=utf-8"]
		//       }
		{
			Description:     "responds json to a GET request with Chrome Accept headers",
			Method:          "GET",
			Query:           "/items?id=eq.1",
			Headers:         test.Headers{"Accept": {"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3"}},
			Expected:        `[{"id":1}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// it "responds json to a GET request to RPC with Firefox Accept headers" $
		//   request methodGet "/rpc/get_projects_below?id=3" firefoxAcceptHdrs ""
		//     `shouldRespondWith` [json|[{"id":1,"name":"Windows 7","client_id":1}, {"id":2,"name":"Windows 10","client_id":1}]|]
		//       { matchHeaders = ["Content-Type" <:> "application/json; charset=utf-8"]
		//       }
		{
			Description:     "responds json to a GET request to RPC with Firefox Accept headers",
			Method:          "GET",
			Query:           "/rpc/get_projects_below?id=3",
			Headers:         test.Headers{"Accept": {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"}},
			Expected:        `[{"id":1,"name":"Windows 7","client_id":1}, {"id":2,"name":"Windows 10","client_id":1}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
		// it "responds json to a GET request to RPC with Chrome Accept headers" $
		//   request methodGet "/rpc/get_projects_below?id=3" chromeAcceptHdrs ""
		//     `shouldRespondWith` [json|[{"id":1,"name":"Windows 7","client_id":1}, {"id":2,"name":"Windows 10","client_id":1}]|]
		//       { matchHeaders = ["Content-Type" <:> "application/json; charset=utf-8"]
		//       }
		{
			Description:     "responds json to a GET request to RPC with Chrome Accept headers",
			Method:          "GET",
			Query:           "/rpc/get_projects_below?id=3",
			Headers:         test.Headers{"Accept": {"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3"}},
			Expected:        `[{"id":1,"name":"Windows 7","client_id":1}, {"id":2,"name":"Windows 10","client_id":1}]`,
			ExpectedHeaders: map[string]string{"Content-Type": "application/json; charset=utf-8"},
			Status:          200,
		},
	}
	test.Execute(t, testConfig, tests)
}
