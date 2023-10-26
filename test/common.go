package test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
)

type Headers http.Header

type Config struct {
	BaseUrl       string
	CommonHeaders Headers
	NoCookies     bool
}

type Command struct {
	Description string
	Method      string
	Query       string
	Body        string
	Headers     Headers
}

// We don't embed Command to simplify tests writing
type Test struct {
	Description string
	Method      string
	Query       string
	Body        string
	Headers     Headers
	Expected    string
	Status      int
}

func InitClient(cookies bool) *http.Client {
	var client *http.Client
	if cookies {
		jar, _ := cookiejar.New(nil)
		client = &http.Client{Jar: jar}
	} else {
		client = &http.Client{}
	}
	return client
}

func Exec(client *http.Client, config Config, cmd *Command) ([]byte, int, error) {
	rawURL := cmd.Query
	if !strings.HasPrefix(rawURL, "http") {
		rawURL = config.BaseUrl + rawURL
	}
	query, err := url.Parse(rawURL)
	if err != nil {
		return nil, 0, err
	}
	query.RawQuery = query.Query().Encode()
	method := cmd.Method
	if method == "" {
		method = "GET"
	}
	var bodyReader io.Reader
	if cmd.Body != "" {
		bodyReader = strings.NewReader(cmd.Body)
	}
	req, err := http.NewRequest(method, query.String(), bodyReader)
	if err != nil {
		return nil, 0, err
	}
	for k, values := range config.CommonHeaders {
		for _, v := range values {
			req.Header.Add(k, v)
		}
	}
	for k, values := range cmd.Headers {
		for i, v := range values {
			if i == 0 && req.Header.Get(k) != "" {
				req.Header.Set(k, v)
			} else {
				req.Header.Add(k, v)
			}
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	resp.Body.Close()
	return body, resp.StatusCode, nil
}

func Prepare(config Config, commands []Command) {
	client := InitClient(!config.NoCookies)
	for _, cmd := range commands {
		_, _, err := Exec(client, config, &cmd)
		if err != nil {
			fmt.Printf("Error: %v", err)
		}
	}
}

func Execute(t *testing.T, config Config, tests []Test) {
	client := InitClient(!config.NoCookies)
	for i, test := range tests {
		command := &Command{test.Description, test.Method, test.Query, test.Body, test.Headers}
		body, status, err := Exec(client, config, command)
		if err != nil {
			t.Errorf("Error: %v", err)
		} else if test.Expected != "" {
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
		}
		if test.Status != 0 && test.Status != status {
			t.Errorf("\n%d. %v\nExpected status \n\t\"%v\", \ngot \n\t\"%v\"", i,
				test.Description, test.Status, status)
		}
	}
}
