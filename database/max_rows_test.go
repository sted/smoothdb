package database

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"testing"
)

// TestMaxRows checks Database.MaxRows, PostgREST's db-max-rows (16.4 run with
// PGRST_DB_MAX_ROWS=2 on its fixtures): every node of a read takes at most
// that many rows from its offset (Plan.hs treeRestrictRange), the root and
// each to-many embed, whatever the limit asked; the total of a count is not
// capped; a function call is capped too; the representation of a write is
// not (treeRestrictRange skips ActRelationMut).
func TestMaxRows(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)
	dbe.DeleteDatabase(ctx, "test_max_rows")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_max_rows")
	if err != nil {
		t.Fatal(err)
	}
	dctx, dconn, err := ContextWithDb(ctx, db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(dctx, dconn)
	gi := GetSmoothContext(dctx)
	if _, err := gi.Conn.Exec(dctx, `
		CREATE TABLE clients (id int PRIMARY KEY);
		CREATE TABLE projects (id int PRIMARY KEY, client_id int REFERENCES clients, name text);
		INSERT INTO clients VALUES (1), (2);
		INSERT INTO projects VALUES (1, 1), (2, 1), (3, 1), (4, 2), (5, 2);
		CREATE FUNCTION all_projects() RETURNS SETOF projects STABLE LANGUAGE sql AS
			$$ SELECT * FROM projects $$;`); err != nil {
		t.Fatal(err)
	}
	if err := db.ReloadSchemaCache(dctx); err != nil {
		t.Fatal(err)
	}

	saved := dbe.config.MaxRows
	dbe.config.MaxRows = 2
	defer func() { dbe.config.MaxRows = saved }()

	base := gi.QueryOptions
	t.Run("the representation of a write", func(t *testing.T) {
		gi.QueryOptions = base
		gi.QueryOptions.ReturnRepresentation = true
		q, _ := url.ParseQuery("select=id&order=id")
		out, _, err := UpdateRecords(dctx, "projects", Record{"name": "x"}, q)
		if err != nil {
			t.Fatal(err)
		}
		if want := `[{"id":1},{"id":2},{"id":3},{"id":4},{"id":5}]`; string(out) != want {
			t.Errorf("rows\n\twant %s\n\tgot  %s", want, out)
		}
	})

	tests := []struct {
		name  string
		run   func(ctx context.Context, q url.Values) ([]byte, int64, error)
		query string
		count string
		want  string
		total int64
	}{
		{"no limit", read("projects"), "select=id&order=id", "", `[{"id":1},{"id":2}]`, 0},
		{"a larger limit", read("projects"), "select=id&order=id&limit=10", "", `[{"id":1},{"id":2}]`, 0},
		{"a smaller limit", read("projects"), "select=id&order=id&limit=1", "", `[{"id":1}]`, 0},
		{"limit=0", read("projects"), "select=id&order=id&limit=0", "", `[]`, 0},
		{"an offset", read("projects"), "select=id&order=id&offset=1", "", `[{"id":2},{"id":3}]`, 0},
		{"an offset near the end", read("projects"), "select=id&order=id&offset=4", "", `[{"id":5}]`, 0},
		{"a count is not capped", read("projects"), "select=id&order=id", "exact", `[{"id":1},{"id":2}]`, 5},
		{"a count with an offset", read("projects"), "select=id&order=id&offset=1", "exact", `[{"id":2},{"id":3}]`, 5},
		{"a to-many embed", read("clients"), "select=id,projects(id)&order=id&projects.order=id",
			"", `[{"id":1,"projects":[{"id":1},{"id":2}]},{"id":2,"projects":[{"id":4},{"id":5}]}]`, 0},
		{"a function call", call("all_projects"), "select=id&order=id", "", `[{"id":1},{"id":2}]`, 0},
		{"a function call with a count", call("all_projects"), "select=id&order=id", "exact", `[{"id":1},{"id":2}]`, 5},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			q, err := url.ParseQuery(test.query)
			if err != nil {
				t.Fatal(err)
			}
			gi.QueryOptions = base
			gi.QueryOptions.Count = test.count
			out, _, err := test.run(dctx, q)
			if err != nil {
				t.Fatal(err)
			}
			if !sameJSON(t, out, test.want) {
				t.Errorf("rows\n\twant %s\n\tgot  %s", test.want, out)
			}
			if test.count != "" && gi.QueryOptions.Total != test.total {
				t.Errorf("total: want %d, got %d", test.total, gi.QueryOptions.Total)
			}
		})
	}

}

func read(table string) func(ctx context.Context, q url.Values) ([]byte, int64, error) {
	return func(ctx context.Context, q url.Values) ([]byte, int64, error) {
		return GetRecords(ctx, table, q)
	}
}

func call(fn string) func(ctx context.Context, q url.Values) ([]byte, int64, error) {
	return func(ctx context.Context, q url.Values) ([]byte, int64, error) {
		return ExecFunction(ctx, fn, Record{}, q, false)
	}
}

// sameJSON compares a JSON document with the one expected, whatever their
// spacing (an embed is json_agg's text).
func sameJSON(t *testing.T, got []byte, want string) bool {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("not JSON: %s", got)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("not JSON: %s", want)
	}
	gj, _ := json.Marshal(g)
	wj, _ := json.Marshal(w)
	return bytes.Equal(gj, wj)
}
