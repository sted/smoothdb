package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_ErrorBodies holds @@ added cases, not ported from the spec:
// requests whose answer was read live on PostgREST 16.4 (2026-11-14).
func TestPostgREST_ErrorBodies(t *testing.T) {

	tests := []test.Test{
		// a JSON body with text after the first value is invalid as a whole
		{
			Description: "rejects a JSON body with trailing text",
			Method:      "PATCH",
			Query:       "/articles?id=eq.-1",
			Body:        `{"body":"changed"} garbage`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `{"code":"PGRST102","details":null,"hint":null,"message":"Empty or invalid json"}`,
			Status:      400,
		},
		// a scalar JSON body is an empty array (Payload.hs, payloadAttributes)
		// (PostgREST answers 201 to `42` without return=representation;
		// smoothdb answers 204 to any empty insert; these cases leave the write status out)
		{
			Description: "inserts nothing for a scalar boolean body, with representation",
			Method:      "POST",
			Query:       "/articles",
			Body:        `true`,
			Headers:     test.Headers{"Prefer": {"return=representation"}},
			Expected:    `[]`,
			Status:      201,
		},
		{
			Description:   "updates nothing for a scalar string body",
			Method:        "PATCH",
			Query:         "/articles?id=eq.1",
			Body:          `"text"`,
			ExpectedEmpty: true,
			Status:        204,
		},
		// an input media type without a reader is an invalid body
		{
			Description:     "rejects an unsupported Content-Type",
			Method:          "POST",
			Query:           "/articles",
			Body:            `<a/>`,
			Headers:         test.Headers{"Content-Type": {"application/xml"}},
			Expected:        `{"code":"PGRST102","details":null,"hint":null,"message":"Content-Type not acceptable: application/xml"}`,
			ExpectedHeaders: map[string]string{"Content-Length": "103"},
			Status:          400,
		},
		// limit and offset that are not integers are ignored
		{
			Description:     "ignores an offset that is not an integer",
			Query:           "/items?offset=abc&select=id&order=id&limit=2",
			Expected:        `[{"id":1},{"id":2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
		},
		{
			Description:     "ignores a limit that is not an integer",
			Query:           "/items?limit=abc&select=id&order=id&id=lt.3",
			Expected:        `[{"id":1},{"id":2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
		},
		{
			Description: "refuses a negative limit beyond 64 bits",
			Query:       "/items?limit=-99999999999999999999",
			Expected:    `{"code":"PGRST103","details":"Limit should be greater than or equal to zero.","hint":null,"message":"Requested range not satisfiable"}`,
			Status:      416,
		},
		// a filter without a valid operator is a parse failure (PGRST100);
		// PostgREST also quotes the message and adds the parsec position
		{
			Description: "refuses an unknown operator with PGRST100",
			Query:       "/items?id=noop.1",
			Expected:    `{"code":"PGRST100","details":"valid sql operator expected","hint":null,"message":"failed to parse filter (noop.1)"}`,
			Status:      400,
		},
		{
			Description: "refuses an operator without a value with PGRST100",
			Query:       "/items?id=eq",
			Expected:    `{"code":"PGRST100","details":"'.' expected","hint":null,"message":"failed to parse filter (eq)"}`,
			Status:      400,
		},
	}

	test.Execute(t, testConfig, tests)
}
