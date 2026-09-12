package test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type Headers http.Header

type Config struct {
	BaseUrl       string
	CommonHeaders Headers
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
	// Expected is the response body to compare against, JSON-normalized.
	// An empty Expected means "no expectation" and the body is NOT checked —
	// so it cannot express PostgREST's `shouldRespondWith ""`. Use
	// ExpectedEmpty for that.
	Expected string
	// ExpectedEmpty asserts the response body is empty. Needed because many
	// ported PostgREST tests expect exactly that (writes default to
	// `Prefer: return=minimal`), and encoding it as `Expected: ""` silently
	// degraded those tests to status-only checks.
	ExpectedEmpty bool
	// ExpectedBase64 says that Expected is the base64 encoding of a binary
	// body (an application/octet-stream download): the body is compared to
	// its decoding, byte by byte.
	ExpectedBase64  bool
	ExpectedHeaders map[string]string
	Status          int
}

func InitClient() *http.Client {
	return &http.Client{}
}

func PrepareRequest(config Config, cmd *Command) (*http.Request, error) {
	rawURL := cmd.Query
	if !strings.HasPrefix(rawURL, "http") {
		rawURL = config.BaseUrl + rawURL
	}
	query, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
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
		return nil, err
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
	return req, nil
}

func ReadResponse(resp *http.Response) ([]byte, *http.Header, int, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, 0, err
	}
	resp.Body.Close()
	return body, &resp.Header, resp.StatusCode, nil
}

func Exec(client *http.Client, config Config, cmd *Command) ([]byte, *http.Header, int, error) {
	req, err := PrepareRequest(config, cmd)
	if err != nil {
		return nil, nil, 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, 0, err
	}
	return ReadResponse(resp)
}

// WaitForServer polls baseURL/live until smoothdb answers. Any listener would
// not do: when the port is already taken by an unrelated process, Start()
// fails with a bind error while the squatter happily answers the probe, and
// the suite then runs against the wrong server with baffling failures. The
// /live body is the fingerprint — the Server header would not do, /live is
// registered outside the middleware that sets it.
func WaitForServer(baseURL string) error {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	var lastErr error
	for i := 0; i < 40; i++ {
		resp, err := client.Get(baseURL + "/live")
		if err == nil {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
			resp.Body.Close()
			var live struct {
				Status string `json:"status"`
			}
			if resp.StatusCode == http.StatusOK && json.Unmarshal(body, &live) == nil && live.Status == "ok" {
				return nil
			}
			return fmt.Errorf("%s/live is answered by something other than smoothdb (HTTP %d, Server %q, body %.80q): is the port taken by another process?",
				baseURL, resp.StatusCode, resp.Header.Get("Server"), body)
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("server at %s did not become ready: %v", baseURL, lastErr)
}

func Prepare(config Config, commands []Command) {
	client := InitClient()
	for _, cmd := range commands {
		_, _, _, err := Exec(client, config, &cmd)
		if err != nil {
			fmt.Printf("Error: %v", err)
		}
	}
}

func Execute(t *testing.T, config Config, tests []Test) {
	var s1, s2 string
	client := InitClient()
	for i, test := range tests {
		command := &Command{test.Description, test.Method, test.Query, test.Body, test.Headers}
		body, headers, status, err := Exec(client, config, command)
		if err != nil {
			t.Errorf("Error: %v", err)
		} else if test.ExpectedEmpty {
			if len(bytes.TrimSpace(body)) != 0 {
				t.Errorf("\n\n%d. %v\nExpected an empty body, \ngot \n\t\"%v\" \n\n(query string -> \"%v\")", i,
					test.Description, string(body), test.Query)
				break
			}
		} else if test.Expected != "" {
			accept := ""
			if v, ok := test.Headers["Accept"]; ok {
				accept = v[0]
			}
			if test.ExpectedBase64 {
				want, err := base64.StdEncoding.DecodeString(test.Expected)
				if err != nil {
					t.Fatalf("%d. %v: ExpectedBase64 is not base64: %v", i, test.Description, err)
				}
				s1 = string(want)
				s2 = string(body)
			} else if strings.Contains(accept, "text/csv") || strings.Contains(accept, "application/octet-stream") {
				// not JSON: compared as they are (decoding both as JSON made
				// every non-JSON pair equal, null against null)
				s1 = test.Expected
				s2 = string(body)
			} else {
				var v1, v2 any
				var j1, j2 []byte
				d1 := json.NewDecoder(bytes.NewBuffer([]byte(test.Expected)))
				d1.UseNumber()
				d1.Decode(&v1)
				j1, _ = json.Marshal(v1)
				d2 := json.NewDecoder(bytes.NewBuffer(body))
				d2.UseNumber()
				d2.Decode(&v2)
				j2, _ = json.Marshal(v2)
				s1, s2 = string(j1), string(j2)
			}
			if s1 != s2 {
				t.Errorf("\n\n%d. %v\nExpected \n\t\"%v\", \ngot \n\t\"%v\" \n\n(query string -> \"%v\")", i,
					test.Description, s1, s2, test.Query)
				break
			}
		}
		if test.ExpectedHeaders != nil {
			for k, v := range test.ExpectedHeaders {
				header := (*headers)[k]
				var first string
				if header != nil {
					first = header[0]
				}
				if v != first {
					t.Errorf("\n%d. %v\nExpected header %v\n\t\"%v\", \ngot \n\t\"%v\"",
						i, test.Description, k, v, first)
				}
			}

		}
		if test.Status != 0 && test.Status != status {
			t.Errorf("\n%d. %v\nExpected status \n\t\"%v\", \ngot \n\t\"%v\"", i,
				test.Description, test.Status, status)
		}
	}
}
