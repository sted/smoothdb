package database

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

// TestFunctionOverloads calls each overload of a name through the Go API. The
// schema cache kept one overload per name and its return type decided the
// shape of every call: the table overload of ov answered "2,0", not JSON,
// and its jsonb overload [{"t": …}]. Each call now answers the shape of the
// overload its argument names invoke, and two overloads that accept the same
// names are refused as PostgREST refuses them (PGRST203).
func TestFunctionOverloads(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}

	dbe.DeleteDatabase(ctx, "test_overload")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_overload")
	if err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)

	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)

	gi := GetSmoothContext(ctx)
	_, err = gi.Conn.Exec(ctx, `
		CREATE FUNCTION ov(p jsonb) RETURNS TABLE (inserted int, deleted int)
			LANGUAGE sql AS $$ SELECT jsonb_array_length(p), 0 $$;
		CREATE FUNCTION ov(p jsonb, mode text) RETURNS jsonb
			LANGUAGE sql AS $$
			SELECT jsonb_build_object('inserted', jsonb_array_length(p), 'deleted', 0, 'mode', mode)
		$$;
		CREATE FUNCTION sc(x int) RETURNS int
			LANGUAGE sql AS $$ SELECT x $$;
		CREATE FUNCTION sc(x int, y int) RETURNS SETOF int
			LANGUAGE sql AS $$ VALUES (x), (y) $$;
		CREATE FUNCTION f_over(a int) RETURNS int
			LANGUAGE sql AS $$ SELECT a $$;
		CREATE FUNCTION f_over(a text) RETURNS text
			LANGUAGE sql AS $$ SELECT a $$;
	`)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.ReloadSchemaCache(ctx); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		record Record
		want   string
	}{
		{"ov", Record{"p": []any{1, 2}}, `[{"inserted": 2, "deleted": 0}]`},
		{"ov", Record{"p": []any{1, 2}, "mode": "x"}, `{"inserted": 2, "deleted": 0, "mode": "x"}`},
		{"sc", Record{"x": 3}, `3`},
		{"sc", Record{"x": 1, "y": 2}, `[1, 2]`},
	}
	for _, tt := range tests {
		got, _, err := ExecFunction(ctx, tt.name, tt.record, nil, false)
		if err != nil {
			t.Errorf("%s(%v): %v", tt.name, tt.record, err)
			continue
		}
		var g, w any
		if err := json.Unmarshal(got, &g); err != nil {
			t.Errorf("%s(%v): not JSON: %s", tt.name, tt.record, got)
			continue
		}
		json.Unmarshal([]byte(tt.want), &w)
		if !reflect.DeepEqual(g, w) {
			t.Errorf("%s(%v): want %s, got %s", tt.name, tt.record, tt.want, got)
		}
	}

	_, _, err = ExecFunction(ctx, "f_over", Record{"a": 5}, nil, false)
	want := "Could not choose the best candidate function between: public.f_over(a => integer), public.f_over(a => text)"
	if err == nil || err.Error() != want {
		t.Errorf("f_over(a): want the error %q, got %v", want, err)
	}
}
