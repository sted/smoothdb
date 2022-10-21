package main

import (
	"testing"
)

func TestPostgREST_Unicode(t *testing.T) {

	tests := []Test{
		// 	describe "Reading and writing to unicode schema and table names" $
		// it "Can read and write values" $ do
		//   get "/%D9%85%D9%88%D8%A7%D8%B1%D8%AF"
		//     `shouldRespondWith` "[]"
		{
			"Can read and write values",
			"/%D9%85%D9%88%D8%A7%D8%B1%D8%AF",
			`[]`,
			nil,
			200,
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

	executeTests(t, tests)
}
