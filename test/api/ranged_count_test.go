package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// createRangedCountFixtures builds the ranged_count schema: a VOLATILE
// function that logs every call in a table and returns the integers from 1
// to n, so that a test can count how many times a request ran it.
func createRangedCountFixtures(t *testing.T) {
	t.Helper()
	execSQLAndReload(t, `
		DROP SCHEMA IF EXISTS ranged_count CASCADE;
		CREATE SCHEMA ranged_count;
		CREATE TABLE ranged_count.calls (n int);
		CREATE FUNCTION ranged_count.probe_log(n int) RETURNS SETOF int
			LANGUAGE plpgsql VOLATILE AS $$
			BEGIN
				INSERT INTO ranged_count.calls VALUES (n);
				RETURN QUERY SELECT generate_series(1, n);
			END
		$$;
	`)
}

// A function runs once whatever the count and the range: PostgREST reads
// the page and the total from one pgrst_source CTE (Query/Statements.hs
// mainCall), so a VOLATILE function writes once: PostgREST 16.4 writes one
// row for each call below.
func TestRangedCountRunsFunctionOnce(t *testing.T) {
	createRangedCountFixtures(t)

	testConfig := test.Config{
		BaseUrl: "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{
			"Authorization":   {adminToken},
			"Accept-Profile":  {"ranged_count"},
			"Content-Profile": {"ranged_count"},
		},
	}

	tests := []test.Test{
		{
			Description: "a ranged count of a function, limit",
			Method:      "POST",
			Query:       "/rpc/probe_log?limit=1",
			Body:        `{"n": 7}`,
			Headers:     test.Headers{"Prefer": {"count=exact"}},
			Expected:    `[1]`,
		},
		{
			Description: "ran the function once",
			Method:      "GET",
			Query:       "/calls?select=n",
			Expected:    `[{"n":7}]`,
			Status:      200,
		},
		{
			Description: "a ranged count of a function, limit and offset",
			Method:      "POST",
			Query:       "/rpc/probe_log?limit=2&offset=5",
			Body:        `{"n": 8}`,
			Headers:     test.Headers{"Prefer": {"count=exact"}},
			Expected:    `[6,7]`,
		},
		{
			Description: "ran the function once more",
			Method:      "GET",
			Query:       "/calls?select=n",
			Expected:    `[{"n":7},{"n":8}]`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}
