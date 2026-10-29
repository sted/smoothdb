package server

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// PostgREST builds the 416 of an offset past the total in the response
// (Response.hs, actionResponse: rangeStatusHeader, then InvalidRange), after
// the transaction has run, so the transaction ends as it would on a 200: a
// function that writes commits with TransactionMode "commit". The 416 was
// taken for an error status and rolled the write back.
func TestRangeNotSatisfiableCommitsTheCall(t *testing.T) {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, superuserURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	drop := []string{
		"DROP FUNCTION IF EXISTS range_commit_write()",
		"DROP TABLE IF EXISTS range_commit_log",
	}
	stmts := append(drop,
		"CREATE TABLE range_commit_log (at timestamptz DEFAULT now())",
		`CREATE FUNCTION range_commit_write() RETURNS SETOF int LANGUAGE plpgsql VOLATILE AS $$
			BEGIN
				INSERT INTO range_commit_log DEFAULT VALUES;
				RETURN QUERY SELECT generate_series(1, 5);
			END
			$$`,
	)
	for _, q := range stmts {
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("%q: %v", q, err)
		}
	}
	defer func() {
		for _, q := range drop {
			if _, err := conn.Exec(ctx, q); err != nil {
				t.Errorf("%q: %v", q, err)
			}
		}
	}()

	s := newTestServer(t, map[string]any{
		"Address":                  "localhost:8120",
		"AllowAnon":                true,
		"Database.AnonRole":        "postgres",
		"Database.TransactionMode": "commit",
		"EnableAdminRoute":         false,
	})
	done := startTestServer(t, s)
	defer func() {
		s.Shutdown(context.Background())
		<-done
	}()

	req, _ := http.NewRequest("POST", "http://localhost:8120/api/postgres/rpc/range_commit_write?offset=10", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "count=exact")
	resp, err := slowClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusRequestedRangeNotSatisfiable || resp.Header.Get("Content-Range") != "*/5" {
		t.Fatalf("offset past the total: got %d %q, want 416 */5", resp.StatusCode, resp.Header.Get("Content-Range"))
	}
	var n int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM range_commit_log").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("the call's write after a 416: %d rows, want 1 (committed)", n)
	}
}
