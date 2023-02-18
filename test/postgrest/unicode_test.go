package postgrest

import (
	"testing"

	"github.com/smoothdb/smoothdb/test"
)

func TestPostgREST_Unicode(t *testing.T) {

	tests := []test.Test{
		// 	describe "Reading and writing to unicode schema and table names" $
		// it "Can read and write values" $ do
		//   get "/%D9%85%D9%88%D8%A7%D8%B1%D8%AF"
		//     `shouldRespondWith` "[]"
		{
			Description: "Can read and write values",
			Query:       "/%D9%85%D9%88%D8%A7%D8%B1%D8%AF",
			Expected:    `[]`,
			Headers:     map[string]string{"Accept-Profile": "تست"},
			Status:      200,
		},
		//   request methodPost "/%D9%85%D9%88%D8%A7%D8%B1%D8%AF"
		//       [("Prefer", "tx=commit"), ("Prefer", "return=representation")]
		//       [json| { "هویت": 1 } |]
		//     `shouldRespondWith`
		//       [json| [{ "هویت": 1 }] |]
		//       { matchStatus = 201 }

		//   get "/%D9%85%D9%88%D8%A7%D8%B1%D8%AF"
		//     `shouldRespondWith`
		//       [json| [{ "هویت": 1 }] |]

		//   request methodDelete "/%D9%85%D9%88%D8%A7%D8%B1%D8%AF"
		//       [("Prefer", "tx=commit")]
		//       ""
		//     `shouldRespondWith`
		//       ""
		//       { matchStatus = 204
		//       , matchHeaders = [matchHeaderAbsent hContentType]
		//       }
	}

	test.Execute(t, testConfig, tests)
}
