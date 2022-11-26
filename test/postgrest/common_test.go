package main

import (
	"encoding/json"
	"green/green-ds/server"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	s, err := server.NewServer()
	if err != nil {
		log.Fatal(err)
	}
	go s.Start()
	code := m.Run()
	os.Exit(code)
}

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
			var v1, v2 any
			var j1, j2 []byte
			_ = json.Unmarshal([]byte(test.expectedResults), &v1)
			j1, _ = json.Marshal(v1)
			_ = json.Unmarshal(body, &v2)
			j2, _ = json.Marshal(v2)
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
