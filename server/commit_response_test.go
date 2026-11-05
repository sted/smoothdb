package server

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/sted/heligo"
	"github.com/sted/smoothdb/database"
)

// commitSetup creates a table with a DEFERRABLE INITIALLY DEFERRED foreign
// key, which PostgreSQL checks at COMMIT, and a function that writes a row
// that violates it and returns five rows. It answers a superuser connection
// to look at the rows; the objects are dropped when the test ends.
func commitSetup(t *testing.T) *pgx.Conn {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, superuserURL)
	if err != nil {
		t.Fatal(err)
	}
	drop := []string{
		"DROP FUNCTION IF EXISTS txr_write_bad()",
		"DROP TABLE IF EXISTS txr_child",
		"DROP TABLE IF EXISTS txr_parent",
	}
	stmts := append(drop,
		"CREATE TABLE txr_parent (id int PRIMARY KEY)",
		`CREATE TABLE txr_child (id int PRIMARY KEY,
			parent int REFERENCES txr_parent DEFERRABLE INITIALLY DEFERRED)`,
		`CREATE FUNCTION txr_write_bad() RETURNS SETOF int LANGUAGE plpgsql VOLATILE AS $$
			BEGIN
				INSERT INTO txr_child VALUES (10, 99);
				RETURN QUERY SELECT generate_series(1, 5);
			END
			$$`,
	)
	for _, q := range stmts {
		if _, err := conn.Exec(ctx, q); err != nil {
			t.Fatalf("%q: %v", q, err)
		}
	}
	t.Cleanup(func() {
		for _, q := range drop {
			if _, err := conn.Exec(ctx, q); err != nil {
				t.Errorf("%q: %v", q, err)
			}
		}
		conn.Close(ctx)
	})
	return conn
}

func childExists(t *testing.T, conn *pgx.Conn, id int) bool {
	t.Helper()
	var n int
	if err := conn.QueryRow(context.Background(), "SELECT count(*) FROM txr_child WHERE id = $1", id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n == 1
}

type commitAnswer struct {
	status int
	body   string
	header http.Header
}

func commitPost(t *testing.T, url, body string, headers map[string]string) commitAnswer {
	t.Helper()
	req, _ := http.NewRequest("POST", url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := slowClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return commitAnswer{resp.StatusCode, string(b), resp.Header}
}

func startCommitServer(t *testing.T, address, mode string) *Server {
	t.Helper()
	s := newTestServer(t, map[string]any{
		"Address":                  address,
		"AllowAnon":                true,
		"Database.AnonRole":        "postgres",
		"Database.TransactionMode": mode,
		"EnableAdminRoute":         false,
	})
	return s
}

func runServer(t *testing.T, s *Server) {
	t.Helper()
	done := startTestServer(t, s)
	t.Cleanup(func() {
		s.Shutdown(context.Background())
		<-done
	})
}

// PostgREST ends the transaction before it builds the response (App.hs:
// usePool runs the whole DbTx, its failure becomes a PgErr, and only then
// actionResponse builds the status). A COMMIT that fails on a deferred
// constraint therefore answers the error of its SQLSTATE (23503 -> 409), not
// the 201 of the write, and the response carries none of the write's headers.
func TestCommitErrorAnswersTheError(t *testing.T) {
	conn := commitSetup(t)
	s := startCommitServer(t, "localhost:8131", "commit")
	runServer(t, s)
	base := "http://localhost:8131/api/postgres"

	a := commitPost(t, base+"/txr_child", `{"id": 1, "parent": 99}`, nil)
	if a.status != http.StatusConflict || !strings.Contains(a.body, `"23503"`) {
		t.Errorf("deferred FK violated at COMMIT: got %d %s, want 409 with code 23503", a.status, a.body)
	}
	for _, h := range []string{"Content-Range", "Content-Location", "Location"} {
		if v := a.header.Get(h); v != "" {
			t.Errorf("the error of a failed COMMIT carries the write's %s: %q", h, v)
		}
	}
	if childExists(t, conn, 1) {
		t.Error("row 1 is present after a failed COMMIT")
	}

	a = commitPost(t, base+"/txr_child", `{"id": 2, "parent": 99}`,
		map[string]string{"Prefer": "return=representation"})
	if a.status != http.StatusConflict || !strings.Contains(a.body, `"23503"`) {
		t.Errorf("return=representation, deferred FK violated at COMMIT: got %d %s, want 409 with code 23503", a.status, a.body)
	}

	// positive control: a write that commits answers its 201 and persists
	a = commitPost(t, base+"/txr_child", `{"id": 3}`, map[string]string{"Prefer": "return=representation"})
	if a.status != http.StatusCreated || !strings.Contains(a.body, `"id":3`) && !strings.Contains(a.body, `"id": 3`) {
		t.Errorf("a valid write: got %d %s, want 201 with the row", a.status, a.body)
	}
	if !childExists(t, conn, 3) {
		t.Error("row 3 is absent after its 201")
	}

	// The 416 of an offset past the total commits like a 200 (see
	// TestRangeNotSatisfiableCommitsTheCall); so its COMMIT error is the
	// answer too, as in PostgREST, where the 416 comes after the transaction.
	a = commitPost(t, base+"/rpc/txr_write_bad?offset=10", `{}`, map[string]string{"Prefer": "count=exact"})
	if a.status != http.StatusConflict || !strings.Contains(a.body, `"23503"`) {
		t.Errorf("416 with a failed COMMIT: got %d %s, want 409 with code 23503", a.status, a.body)
	}
	if childExists(t, conn, 10) {
		t.Error("row 10 is present after a failed COMMIT")
	}
}

// With TransactionMode "rollback" a write answers its status and is rolled
// back, also when a deferred constraint is violated: ROLLBACK checks nothing.
func TestRollbackModeStillRollsBack(t *testing.T) {
	conn := commitSetup(t)
	s := startCommitServer(t, "localhost:8132", "rollback")
	runServer(t, s)
	base := "http://localhost:8132/api/postgres"

	a := commitPost(t, base+"/txr_child", `{"id": 1}`, map[string]string{"Prefer": "return=representation"})
	if a.status != http.StatusCreated {
		t.Errorf("rollback mode: got %d %s, want 201", a.status, a.body)
	}
	a = commitPost(t, base+"/txr_child", `{"id": 2, "parent": 99}`, nil)
	if a.status != http.StatusCreated {
		t.Errorf("rollback mode, deferred FK violated: got %d %s, want 201", a.status, a.body)
	}
	if childExists(t, conn, 1) || childExists(t, conn, 2) {
		t.Error("a write of rollback mode persisted")
	}
}

// With TransactionMode "none" the write is its own transaction, committed
// with the statement: a deferred violation is the statement's error, and a
// valid write persists. Nothing changes there.
func TestNoneModeWritesUnchanged(t *testing.T) {
	conn := commitSetup(t)
	s := startCommitServer(t, "localhost:8133", "none")
	runServer(t, s)
	base := "http://localhost:8133/api/postgres"

	a := commitPost(t, base+"/txr_child", `{"id": 1, "parent": 99}`, nil)
	if a.status != http.StatusConflict || !strings.Contains(a.body, `"23503"`) {
		t.Errorf("none mode, deferred FK violated: got %d %s, want 409 with code 23503", a.status, a.body)
	}
	a = commitPost(t, base+"/txr_child", `{"id": 2}`, nil)
	if a.status != http.StatusCreated {
		t.Errorf("none mode: got %d %s, want 201", a.status, a.body)
	}
	if childExists(t, conn, 1) || !childExists(t, conn, 2) {
		t.Error("none mode: want row 2 only")
	}

	// A write with a representation runs in a transaction of the request
	// (card 32942): the deferred check moves to its COMMIT, before the
	// response, and still answers 409.
	rep := map[string]string{"Prefer": "return=representation"}
	a = commitPost(t, base+"/txr_child", `{"id": 3, "parent": 99}`, rep)
	if a.status != http.StatusConflict || !strings.Contains(a.body, `"23503"`) {
		t.Errorf("none mode, representation, deferred FK violated: got %d %s, want 409 with code 23503", a.status, a.body)
	}
	a = commitPost(t, base+"/txr_child", `{"id": 4}`, rep)
	if a.status != http.StatusCreated || !strings.Contains(a.body, `"id":4`) {
		t.Errorf("none mode, representation: got %d %s, want 201 with the row", a.status, a.body)
	}
	if childExists(t, conn, 3) || !childExists(t, conn, 4) {
		t.Error("none mode, representation: want row 4 only")
	}
}

type cancelKey struct{}

// cancelRoute adds to s a route under the database middleware whose handler
// writes a row, then cancels the request context before it returns: what a
// client that hangs up right after the response does to the end of the
// request.
func cancelRoute(s *Server, id string) {
	cancelling := func(next heligo.Handler) heligo.Handler {
		return func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
			cctx, cancel := context.WithCancel(c)
			defer cancel()
			return next(context.WithValue(cctx, cancelKey{}, cancel), w, r)
		}
	}
	g := s.GetRouter().Group("/txcancel", cancelling, s.MiddlewareWithDbName("postgres"))
	g.Handle("POST", "", func(c context.Context, w http.ResponseWriter, r heligo.Request) (int, error) {
		conn := database.GetSmoothContext(c).Conn
		if _, err := conn.Exec(c, "INSERT INTO txr_child VALUES ("+id+", NULL)"); err != nil {
			return heligo.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		c.Value(cancelKey{}).(context.CancelFunc)()
		return heligo.WriteHeader(w, http.StatusCreated)
	})
}

// A request context canceled once the handler has answered (the client hung
// up) must not stop the end of the transaction: the COMMIT runs on a context
// without the cancellation, bounded by a timeout.
func TestCanceledContextStillCommits(t *testing.T) {
	conn := commitSetup(t)
	s := startCommitServer(t, "localhost:8134", "commit")
	cancelRoute(s, "20")
	runServer(t, s)

	a := commitPost(t, "http://localhost:8134/txcancel", `{}`, nil)
	if a.status != http.StatusCreated {
		t.Fatalf("got %d %s, want 201", a.status, a.body)
	}
	if !childExists(t, conn, 20) {
		t.Error("commit mode: the row of a request whose context was canceled after the handler is absent")
	}
}

// The same with TransactionMode "none": the row is committed with its
// statement, and the request ends normally.
func TestCanceledContextNoneMode(t *testing.T) {
	conn := commitSetup(t)
	s := startCommitServer(t, "localhost:8135", "none")
	cancelRoute(s, "21")
	runServer(t, s)

	a := commitPost(t, "http://localhost:8135/txcancel", `{}`, nil)
	if a.status != http.StatusCreated {
		t.Fatalf("got %d %s, want 201", a.status, a.body)
	}
	if !childExists(t, conn, 21) {
		t.Error("none mode: the row is absent")
	}
}
