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
		// a filter without a valid operator is a parse failure (PGRST100).
		// Gap: PostgREST quotes the message, adds the parsec position
		// ("(line 1, column 1)") and lists the unexpected and expected tokens
		// in the details; smoothdb's parser does not track them.
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
		// limit and offset beyond 64 bits reach PostgreSQL, which refuses them
		{
			Description: "refuses an offset beyond 64 bits with 22003",
			Query:       "/items?select=id&order=id&offset=9223372036854775808&limit=1",
			Expected:    `{"code":"22003","details":null,"hint":null,"message":"value \"9223372036854775808\" is out of range for type bigint"}`,
			Status:      400,
		},
		{
			Description: "refuses a limit beyond 64 bits with 22003",
			Query:       "/items?select=id&order=id&limit=9223372036854775808",
			Expected:    `{"code":"22003","details":null,"hint":null,"message":"value \"9223372036854775808\" is out of range for type bigint"}`,
			Status:      400,
		},
		// limit and offset make one range, intersected with the rows from 0
		// (QueryParams.hs, ApiRequest.hs getRanges)
		{
			Description: "refuses a range that ends before the first row",
			Query:       "/items?select=id&order=id&limit=3&offset=-4",
			Expected:    `{"code":"PGRST103","details":"Limit should be greater than or equal to zero.","hint":null,"message":"Requested range not satisfiable"}`,
			Status:      416,
		},
		{
			Description:     "cuts a range that starts before the first row",
			Query:           "/items?select=id&order=id&limit=3&offset=-1",
			Expected:        `[{"id":1},{"id":2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
		},
		{
			Description:     "answers no row for a range that ends just before the first row",
			Query:           "/items?select=id&order=id&limit=3&offset=-3",
			Expected:        `[]`,
			ExpectedHeaders: map[string]string{"Content-Range": "*/*"},
			Status:          200,
		},
		{
			Description: "refuses limit 0 with an offset",
			Query:       "/items?select=id&order=id&limit=0&offset=5",
			Expected:    `{"code":"PGRST103","details":"Limit should be greater than or equal to zero.","hint":null,"message":"Requested range not satisfiable"}`,
			Status:      416,
		},
		{
			Description: "refuses a negative offset with a limit that is not an integer",
			Query:       "/items?select=id&order=id&offset=-4&limit=abc",
			Expected:    `{"code":"PGRST103","details":"Limit should be greater than or equal to zero.","hint":null,"message":"Requested range not satisfiable"}`,
			Status:      416,
		},
		// with ?columns the body goes to PostgreSQL as it is
		{
			Description: "sends a scalar body with columns to PostgreSQL",
			Method:      "POST",
			Query:       "/articles?columns=id",
			Body:        `true`,
			Expected:    `{"code":"22023","details":null,"hint":null,"message":"cannot call json_to_recordset on a scalar"}`,
			Status:      400,
		},
		// the errors name the request's media types in full
		{
			Description: "lists every Accept alternative in PGRST107",
			Query:       "/images?select=img,name&name=eq.A.png",
			Headers:     test.Headers{"Accept": {"application/octet-stream, image/png"}},
			Expected:    `{"code":"PGRST107","details":null,"hint":null,"message":"None of these media types are available: application/octet-stream, image/png"}`,
			Status:      406,
		},
		{
			Description:     "keeps the parameters of an unsupported Content-Type",
			Method:          "POST",
			Query:           "/articles",
			Body:            `<a/>`,
			Headers:         test.Headers{"Content-Type": {"application/xml; profile=foo"}},
			Expected:        `{"code":"PGRST102","details":null,"hint":null,"message":"Content-Type not acceptable: application/xml; profile=foo"}`,
			ExpectedHeaders: map[string]string{"Content-Length": "116"},
			Status:          400,
		},
		// an empty value is a value that is not an integer, and the last
		// occurrence of a parameter counts (QueryParams.hs, nonemptyParams)
		{
			Description: "refuses an empty limit with an offset",
			Query:       "/items?select=id&order=id&limit=&offset=5",
			Expected:    `{"code":"PGRST103","details":"Limit should be greater than or equal to zero.","hint":null,"message":"Requested range not satisfiable"}`,
			Status:      416,
		},
		{
			Description:     "takes the last limit, even when it is empty",
			Query:           "/items?select=id&order=id&id=lt.4&limit=2&limit=",
			Expected:        `[{"id":1},{"id":2},{"id":3}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-2/*"},
			Status:          200,
		},
		// readMaybe reads an integer between spaces and with leading zeros
		{
			Description:     "reads a limit between spaces",
			Query:           "/items?select=id&order=id&limit=%202%20",
			Expected:        `[{"id":1},{"id":2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
		},
		{
			Description:     "reads a limit with leading zeros",
			Query:           "/items?select=id&order=id&limit=00002",
			Expected:        `[{"id":1},{"id":2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
		},
		// the Range header narrows a limit beyond 64 bits before PostgreSQL
		{
			Description:     "narrows a limit beyond 64 bits with the Range header",
			Query:           "/items?select=id&order=id&limit=9223372036854775808",
			Headers:         test.Headers{"Range": {"0-1"}},
			Expected:        `[{"id":1},{"id":2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/*"},
			Status:          200,
		},
		// with ?columns a PATCH sends the body as it is, as a POST does
		{
			Description: "sends a scalar PATCH body with columns to PostgreSQL",
			Method:      "PATCH",
			Query:       "/articles?id=eq.1&columns=body",
			Body:        `true`,
			Expected:    `{"code":"22023","details":null,"hint":null,"message":"cannot call json_to_recordset on a scalar"}`,
			Status:      400,
		},
	}

	test.Execute(t, testConfig, tests)
}
