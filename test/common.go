package test

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
)

type Test struct {
	Description string
	Method      string
	Query       string
	Body        string
	Expected    string
	Headers     map[string]string
	Status      int
}

type TestConfig struct {
	BaseUrl       string
	CommonHeaders map[string]string
	NoCookies     bool
}

func Execute(t *testing.T, config TestConfig, tests []Test) {
	var client *http.Client
	if !config.NoCookies {
		jar, _ := cookiejar.New(nil)
		client = &http.Client{Jar: jar}
	} else {
		client = &http.Client{}
	}

	for i, test := range tests {
		query, err := url.Parse(config.BaseUrl + test.Query)
		if err != nil {
			log.Fatal(err)
		}
		query.RawQuery = query.Query().Encode()
		method := test.Method
		if method == "" {
			method = "GET"
		}
		var bodyReader io.Reader
		if test.Body != "" {
			bodyReader = strings.NewReader(test.Body)
		}
		req, err := http.NewRequest(method, query.String(), bodyReader)
		if err != nil {
			log.Fatal(err)
		}
		for k, v := range config.CommonHeaders {
			req.Header.Add(k, v)
		}
		for k, v := range test.Headers {
			req.Header.Add(k, v)
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		if test.Expected != "" {
			var v1, v2 any
			var j1, j2 []byte
			_ = json.Unmarshal([]byte(test.Expected), &v1)
			j1, _ = json.Marshal(v1)
			_ = json.Unmarshal(body, &v2)
			j2, _ = json.Marshal(v2)
			json1, json2 := string(j1), string(j2)
			if json1 != json2 {
				t.Errorf("\n\n%d. %v\nExpected \n\t\"%v\", \ngot \n\t\"%v\" \n\n(query string -> \"%v\")", i,
					test.Description, json1, json2, test.Query)
			}
		} else if test.Status != 0 && test.Status != resp.StatusCode {
			t.Errorf("\n%d. %v\nExpected status \n\t\"%v\", \ngot \n\t\"%v\" \n\n(query string -> \"%v\")", i,
				test.Description, test.Status, resp.StatusCode, query.String())
		}
		resp.Body.Close()
	}
}
