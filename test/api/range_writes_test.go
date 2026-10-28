package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestRangeOnTableWrites: a Range header has no effect on a write to a
// table, neither on its representation nor on Content-Range, as in PostgREST
// (the header is ignored for table mutations); a GET still honours it. RPC
// keeps it too, see the getitemrange spec in test/postgrest.
func TestRangeOnTableWrites(t *testing.T) {

	cmdConfig := test.Config{
		BaseUrl:       "http://localhost:8082/admin",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	commands := []test.Command{
		{
			Method: "DELETE",
			Query:  "/databases/dbtest/tables/rangewrites",
		},
		{
			Method: "POST",
			Query:  "/databases/dbtest/tables",
			Body: `{
				"name": "rangewrites",
				"columns": [
					{"name": "id", "type": "integer"},
					{"name": "v", "type": "integer"}
				]}`,
		},
	}
	test.Prepare(cmdConfig, commands)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}

	repr := test.Headers{"Prefer": {"return=representation"}}
	reprRange := func(r string) test.Headers {
		return test.Headers{"Prefer": {"return=representation"}, "Range": {r}}
	}
	tests := []test.Test{
		{
			Description:     "POST without Range",
			Method:          "POST",
			Query:           "/rangewrites",
			Body:            `[{"id": 1, "v": 1}, {"id": 2, "v": 2}]`,
			Headers:         repr,
			Expected:        `[{"id": 1, "v": 1}, {"id": 2, "v": 2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "*/*"},
			Status:          201,
		},
		{
			Description:     "POST with Range: ignored",
			Method:          "POST",
			Query:           "/rangewrites",
			Body:            `[{"id": 3, "v": 3}, {"id": 4, "v": 4}]`,
			Headers:         reprRange("0-0"),
			Expected:        `[{"id": 3, "v": 3}, {"id": 4, "v": 4}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "*/*"},
			Status:          201,
		},
		{
			Description:     "PATCH without Range",
			Method:          "PATCH",
			Query:           "/rangewrites?order=id",
			Body:            `{"v": 0}`,
			Headers:         repr,
			Expected:        `[{"id": 1, "v": 0}, {"id": 2, "v": 0}, {"id": 3, "v": 0}, {"id": 4, "v": 0}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "*/*"},
			Status:          200,
		},
		{
			Description:     "PATCH with Range: ignored",
			Method:          "PATCH",
			Query:           "/rangewrites?order=id",
			Body:            `{"v": 5}`,
			Headers:         reprRange("0-0"),
			Expected:        `[{"id": 1, "v": 5}, {"id": 2, "v": 5}, {"id": 3, "v": 5}, {"id": 4, "v": 5}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "*/*"},
			Status:          200,
		},
		{
			Description:     "PATCH with an inverted Range: ignored, not 416",
			Method:          "PATCH",
			Query:           "/rangewrites?order=id",
			Body:            `{"v": 6}`,
			Headers:         reprRange("1-0"),
			Expected:        `[{"id": 1, "v": 6}, {"id": 2, "v": 6}, {"id": 3, "v": 6}, {"id": 4, "v": 6}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "*/*"},
			Status:          200,
		},
		{
			Description:     "PATCH return=minimal with Range: ignored",
			Method:          "PATCH",
			Query:           "/rangewrites",
			Body:            `{"v": 7}`,
			Headers:         test.Headers{"Range": {"0-0"}},
			ExpectedEmpty:   true,
			ExpectedHeaders: map[string]string{"Content-Range": "*/*"},
			Status:          204,
		},
		{
			Description:     "jq PATCH with Range: ignored",
			Method:          "PATCH",
			Query:           jqQuery("/rangewrites", "order=id", `{"v": .id}`, ""),
			Headers:         reprRange("0-0"),
			Expected:        `[{"id": 1, "v": 1}, {"id": 2, "v": 2}, {"id": 3, "v": 3}, {"id": 4, "v": 4}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "*/*"},
			Status:          200,
		},
		{
			Description:     "GET with Range: honoured",
			Method:          "GET",
			Query:           "/rangewrites?order=id",
			Headers:         test.Headers{"Range": {"0-0"}},
			Expected:        `[{"id": 1, "v": 1}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/*"},
			Status:          200,
		},
		{
			Description:     "DELETE with Range: ignored",
			Method:          "DELETE",
			Query:           "/rangewrites?id=gte.3&order=id",
			Headers:         reprRange("0-0"),
			Expected:        `[{"id": 3, "v": 3}, {"id": 4, "v": 4}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "*/*"},
			Status:          200,
		},
		{
			Description:     "DELETE without Range",
			Method:          "DELETE",
			Query:           "/rangewrites?order=id",
			Headers:         repr,
			Expected:        `[{"id": 1, "v": 1}, {"id": 2, "v": 2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "*/*"},
			Status:          200,
		},
	}

	test.Execute(t, testConfig, tests)
}
