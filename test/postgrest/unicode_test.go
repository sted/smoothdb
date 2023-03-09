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
			Headers:     test.Headers{"Accept-Profile": {"تست"}},
			Expected:    `[]`,
			Status:      200,
		},
		//   request methodPost "/%D9%85%D9%88%D8%A7%D8%B1%D8%AF"
		//       [("Prefer", "tx=commit"), ("Prefer", "return=representation")]
		//       [json| { "هویت": 1 } |]
		//     `shouldRespondWith`
		//       [json| [{ "هویت": 1 }] |]
		//       { matchStatus = 201 }
		{
			Description: "Can read and write values",
			Method:      "POST",
			Query:       "/%D9%85%D9%88%D8%A7%D8%B1%D8%AF",
			Headers: test.Headers{"Content-Profile": {"تست"},
				"Prefer": {"tx=commit", "return=representation"}},
			Body:     `{ "هویت": 1 }`,
			Expected: `[{ "هویت": 1 }]`,
			Status:   201,
		},
		//   get "/%D9%85%D9%88%D8%A7%D8%B1%D8%AF"
		//     `shouldRespondWith`
		//       [json| [{ "هویت": 1 }] |]
		{
			Description: "Can read and write values",
			Method:      "GET",
			Query:       "/%D9%85%D9%88%D8%A7%D8%B1%D8%AF",
			Body:        "",
			Headers:     test.Headers{"Accept-Profile": {"تست"}},
			Expected:    `[{ "هویت": 1 }]`,
			Status:      200,
		},
		//   request methodDelete "/%D9%85%D9%88%D8%A7%D8%B1%D8%AF"
		//       [("Prefer", "tx=commit")]
		//       ""
		//     `shouldRespondWith`
		//       ""
		//       { matchStatus = 204
		//       , matchHeaders = [matchHeaderAbsent hContentType]
		//       }
		{
			Description: "Can read and write values",
			Method:      "DELETE",
			Query:       "/%D9%85%D9%88%D8%A7%D8%B1%D8%AF",
			Body:        "",
			Headers:     test.Headers{"Content-Profile": {"تست"}, "Prefer": {"tx=commit"}},
			Expected:    ``,
			Status:      204,
		},
	}

	test.Execute(t, testConfig, tests)
}
