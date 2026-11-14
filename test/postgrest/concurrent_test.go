package postgrest

import (
	"sync"
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_Concurrent holds the cases of ConcurrentSpec.hs (PostgREST 16.4), converted by hspec2go.
// The raceTest helper of the spec is written in Go below.
func TestPostgREST_Concurrent(t *testing.T) {
	tests := []test.Test{
		// describe "Querying in parallel" $
		//   it "should not raise 'transaction in progress' error" $
		//     raceTest 10 $
		//       get "/fakefake"
		//         `shouldRespondWith` [json| {"code":"PGRST205","details":null,"hint":null,"message":"Could not find the table 'test.fakefake' in the schema cache"} |]
		//           { matchStatus = 404
		//           , matchHeaders = []
		//           }
		{
			Description: "should not raise 'transaction in progress' error",
			Method:      "GET",
			Query:       "/fakefake",
			Expected:    `{"code":"PGRST205","details":null,"hint":null,"message":"Could not find the table 'test.fakefake' in the schema cache"}`,
			Status:      404,
		},
	}
	// raceTest 10: the same request ten times in parallel. Execute then runs
	// it once more with the full check, so that it counts as one case.
	var wg sync.WaitGroup
	client := test.InitClient()
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := tests[0]
			_, _, status, err := test.Exec(client, testConfig, &test.Command{Description: c.Description, Method: c.Method, Query: c.Query})
			if err != nil || status != c.Status {
				t.Errorf("parallel %s: status %d, error %v", c.Query, status, err)
			}
		}()
	}
	wg.Wait()
	test.Execute(t, testConfig, tests)
}
