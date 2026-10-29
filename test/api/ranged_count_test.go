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
			Description:     "a ranged count of a function, limit",
			Method:          "POST",
			Query:           "/rpc/probe_log?limit=1",
			Body:            `{"n": 7}`,
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[1]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/7"},
			Status:          206,
		},
		{
			Description: "ran the function once",
			Method:      "GET",
			Query:       "/calls?select=n",
			Expected:    `[{"n":7}]`,
			Status:      200,
		},
		{
			Description:     "a ranged count of a function, limit and offset",
			Method:          "POST",
			Query:           "/rpc/probe_log?limit=2&offset=5",
			Body:            `{"n": 8}`,
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[6,7]`,
			ExpectedHeaders: map[string]string{"Content-Range": "5-6/8"},
			Status:          206,
		},
		{
			Description: "ran the function once more",
			Method:      "GET",
			Query:       "/calls?select=n",
			Expected:    `[{"n":7},{"n":8}]`,
			Status:      200,
		},
		// an empty page runs the function once too (PostgREST 16.4 writes one
		// row for each: limit=0 answers 206 */10, an offset past the end 416)
		{
			Description:     "a ranged count of a function, limit=0",
			Method:          "POST",
			Query:           "/rpc/probe_log?limit=0",
			Body:            `{"n": 10}`,
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[]`,
			ExpectedHeaders: map[string]string{"Content-Range": "*/10"},
			Status:          206,
		},
		{
			Description:     "a ranged count of a function, offset past the end",
			Method:          "POST",
			Query:           "/rpc/probe_log?offset=20",
			Body:            `{"n": 11}`,
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			ExpectedHeaders: map[string]string{"Content-Range": "*/11"},
			Status:          416,
		},
		{
			Description: "ran the function once for each",
			Method:      "GET",
			Query:       "/calls?select=n",
			Expected:    `[{"n":7},{"n":8},{"n":10},{"n":11}]`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}

// With autocommit (TransactionMode "none") the page and the total of a ranged
// count are one batch, one implicit transaction that PostgreSQL commits when
// the batch ends: a DEFERRABLE INITIALLY DEFERRED constraint is checked then,
// and its violation rolls the write back. PostgREST checks it at the COMMIT
// of its transaction and answers the error (23505 is 409); the commit error
// of the batch was dropped, and the request answered the page of a write that
// did not happen.
func TestRangedCountReportsTheCommitError(t *testing.T) {
	createRangedCountFixtures(t)
	execSQLAndReload(t, `
		CREATE TABLE ranged_count.deferred (v int UNIQUE DEFERRABLE INITIALLY DEFERRED);
		CREATE FUNCTION ranged_count.insert_twice(v int) RETURNS SETOF ranged_count.deferred
			LANGUAGE sql VOLATILE AS $$
				INSERT INTO ranged_count.deferred VALUES (v), (v) RETURNING *
			$$;
	`)

	testConfig := test.Config{
		BaseUrl: "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{
			"Authorization":   {adminToken},
			"Accept-Profile":  {"ranged_count"},
			"Content-Profile": {"ranged_count"},
		},
	}

	tests := []test.Test{
		// control: without a count the call is one statement, whose commit
		// error the rows report
		{
			Description: "a deferred violation without a count",
			Method:      "POST",
			Query:       "/rpc/insert_twice",
			Body:        `{"v": 1}`,
			Status:      409,
		},
		{
			Description: "a deferred violation with a ranged count",
			Method:      "POST",
			Query:       "/rpc/insert_twice?limit=1",
			Body:        `{"v": 2}`,
			Headers:     test.Headers{"Prefer": {"count=exact"}},
			Status:      409,
		},
		{
			Description: "both writes were rolled back",
			Method:      "GET",
			Query:       "/deferred",
			Expected:    `[]`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}
