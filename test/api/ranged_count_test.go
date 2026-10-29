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

// The ranged count of a function stores its total from a one-time filter of
// the page (see buildAfterSelect); a filter that PostgreSQL folds to a
// constant false (id=eq.null is t.id = null) drops that filter with the rest
// of the plan, and the total was never stored: reading it back failed. The
// count over a false filter is 0, which is what an unset total reads now.
func TestRangedCountFunctionConstantFalseFilter(t *testing.T) {
	createRangedCountFixtures(t)
	execSQLAndReload(t, `
		CREATE FUNCTION ranged_count.numbers(n int) RETURNS TABLE (id int, label text)
			LANGUAGE sql VOLATILE AS $$
				SELECT i, 'n' || i FROM generate_series(1, n) i
			$$;
	`)

	testConfig := test.Config{
		BaseUrl: "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{
			"Authorization":   {adminToken},
			"Accept-Profile":  {"ranged_count"},
			"Content-Profile": {"ranged_count"},
			"Prefer":          {"count=exact"},
		},
	}

	tests := []test.Test{
		// control: a filter that is not constant
		{
			Description:     "a ranged count of a function, a filter",
			Method:          "POST",
			Query:           "/rpc/numbers?id=eq.2&limit=1",
			Body:            `{"n": 5}`,
			Expected:        `[{"id":2,"label":"n2"}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
			Status:          200,
		},
		{
			Description:     "a ranged count of a function, a constant false filter",
			Method:          "POST",
			Query:           "/rpc/numbers?id=eq.null&limit=1",
			Body:            `{"n": 5}`,
			Expected:        `[]`,
			ExpectedHeaders: map[string]string{"Content-Range": "*/0"},
			Status:          200,
		},
	}

	test.Execute(t, testConfig, tests)
}

// A STABLE function is counted like a table: the page and a COUNT(*) over
// the same call, which PostgreSQL may inline; its total and range status are
// those of the VOLATILE path.
func TestRangedCountStableFunction(t *testing.T) {
	createRangedCountFixtures(t)
	execSQLAndReload(t, `
		CREATE FUNCTION ranged_count.stable_numbers(n int) RETURNS TABLE (id int, label text)
			LANGUAGE sql STABLE AS $$
				SELECT i, 'n' || i FROM generate_series(1, n) i
			$$;
	`)

	testConfig := test.Config{
		BaseUrl: "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{
			"Authorization":   {adminToken},
			"Accept-Profile":  {"ranged_count"},
			"Content-Profile": {"ranged_count"},
			"Prefer":          {"count=exact"},
		},
	}

	tests := []test.Test{
		{
			Description:     "a ranged count of a STABLE function",
			Method:          "POST",
			Query:           "/rpc/stable_numbers?id=gt.2&order=id.desc&limit=2&offset=1",
			Body:            `{"n": 9}`,
			Expected:        `[{"id":8,"label":"n8"},{"id":7,"label":"n7"}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "1-2/7"},
			Status:          206,
		},
		{
			Description:     "a ranged count of a STABLE function, GET",
			Method:          "GET",
			Query:           "/rpc/stable_numbers?n=4&limit=1",
			Expected:        `[{"id":1,"label":"n1"}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/4"},
			Status:          206,
		},
		{
			Description:     "a ranged count of a STABLE function, offset past the total",
			Method:          "POST",
			Query:           "/rpc/stable_numbers?offset=20",
			Body:            `{"n": 3}`,
			ExpectedHeaders: map[string]string{"Content-Range": "*/3"},
			Status:          416,
		},
	}

	test.Execute(t, testConfig, tests)
}
