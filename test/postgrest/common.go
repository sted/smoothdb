package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"testing"
)

type Test struct {
	description     string
	query           string
	expectedResults string
	headers         []string
	status          int
}

func executeTests(t *testing.T, tests []Test) {
	for i, test := range tests {
		query, err := url.Parse("http://localhost:8081/databases/pgrest" + test.query)
		if err != nil {
			log.Fatal(err)
		}
		query.RawQuery = query.Query().Encode()
		req, err := http.NewRequest("GET", query.String(), nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Add("Accept-Profile", "test")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		if test.expectedResults != "" {
			var v any
			var j1, j2 []byte
			_ = json.Unmarshal([]byte(test.expectedResults), &v)
			j1, _ = json.Marshal(v)
			_ = json.Unmarshal(body, &v)
			j2, _ = json.Marshal(v)
			json1, json2 := string(j1), string(j2)
			if json1 != json2 {
				t.Errorf("\n\n%d. %v\nExpected \n\t\"%v\", \ngot \n\t\"%v\" \n\n(query string -> \"%v\")", i, test.description, json1, json2, test.query)
			}
		} else if test.status != resp.StatusCode {
			t.Errorf("\n%d. %v\nExpected status \n\t\"%v\", \ngot \n\t\"%v\" \n\n(query string -> \"%v\")", i, test.description, test.status, resp.StatusCode, query.String())
		}
		resp.Body.Close()
	}
}
