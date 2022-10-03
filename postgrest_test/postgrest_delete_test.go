package main

import (
	"io"
	"log"
	"net/http"
	"testing"
)

func TestPostgREST_Delete(t *testing.T) {

	tests := []struct {
		description     string
		query           string
		expectedResults string
		headers         []string
		status          int
	}{}

	for i, test := range tests {
		query := "http://localhost:8081/databases/pgrest" + test.query
		req, err := http.NewRequest("GET", query, nil)
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
