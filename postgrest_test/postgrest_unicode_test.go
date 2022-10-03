package main

import (
	"io"
	"log"
	"net/http"
	"testing"
)

func TestPostgREST_Unicode(t *testing.T) {

	tests := []struct {
		description     string
		query           string
		expectedResults string
		headers         []string
		status          int
	}{
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

	for i, test := range tests {
		query := "http://localhost:8081/databases/pgrest" + test.query
		req, err := http.NewRequest("GET", query, nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Add("Accept-Profile", "تست")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		json := string(body)
		if test.expectedResults != "" {
			if test.expectedResults != json {
				t.Errorf("\n\n%d. %v\nExpected \n\t\"%v\", \ngot \n\t\"%v\" \n(query string -> \"%v\")", i, test.description, test.expectedResults, json, test.query)
			}
		} else if test.status != resp.StatusCode {
			t.Errorf("\n%d. %v\nExpected status \n\t\"%v\", \ngot \n\t\"%v\" \n(query string -> \"%v\")", i, test.description, test.status, resp.StatusCode, test.query)

		}
		resp.Body.Close()
	}
}
