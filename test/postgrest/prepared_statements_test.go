package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_PreparedStatements holds the cases of PreparedStatementsSpec.hs (PostgREST 16.4), converted by hspec2go.
func TestPostgREST_PreparedStatements(t *testing.T) {
	tests := []test.Test{
		// it "should use prepared statements when the setting is enabled" $
		//   request
		//     methodPost
		//     "/rpc/uses_prepared_statements"
		//     []
		//     ""
		//     `shouldRespondWith` [str|true|]
		//       { matchStatus = 200
		//       }
		{
			Description: "should use prepared statements when the setting is enabled",
			Method:      "POST",
			Query:       "/rpc/uses_prepared_statements",
			Expected:    `true`,
			Status:      200,
			Skip:        "PostgREST 16.4: db-prepared-statements can be disabled; card 32500 (deployment knobs)",
		},
		// it "should not use any prepared statements when the setting is disabled" $ do
		//   request
		//     methodGet
		//     "/never_prepared"
		//     []
		//     ""
		//     `shouldRespondWith` 200
		{
			Description: "should not use any prepared statements when the setting is disabled",
			Method:      "GET",
			Query:       "/never_prepared",
			Status:      200,
			Skip:        "PostgREST 16.4: db-prepared-statements can be disabled; card 32500 (deployment knobs)",
		},
		//
		//   request
		//     methodPost
		//     "/rpc/never_uses_prepared_statements"
		//     []
		//     ""
		//     `shouldRespondWith` [str|true|]
		//       { matchStatus = 200
		//       }
		{
			Description: "should not use any prepared statements when the setting is disabled",
			Method:      "POST",
			Query:       "/rpc/never_uses_prepared_statements",
			Expected:    `true`,
			Status:      200,
			Skip:        "PostgREST 16.4: db-prepared-statements can be disabled; card 32500 (deployment knobs)",
		},
	}
	test.Execute(t, testConfig, tests)
}
