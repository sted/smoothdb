package database

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
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

// TestFindFunction resolves calls on the overloaded functions of PostgREST's
// fixtures (test/postgrest/fixtures/schema.sql) as Plan.hs findProc does:
// by the argument names, an argument with a default being optional.
func TestFindFunction(t *testing.T) {
	arg := func(name, typ string, typeId uint32) Argument {
		return Argument{Name: name, Type: typ, Mode: 'i', TypeId: typeId}
	}
	fn := func(name string, defaults int, args ...Argument) Function {
		if len(args) == 0 {
			// what the query gives for a function with no argument
			args = []Argument{{}}
		}
		return Function{Name: name, Schema: "test", Arguments: args, ArgDefaults: defaults}
	}
	integer, text := uint32(23), uint32(25)
	// listed out of order: the cache sorts each name's overloads
	info := &SchemaInfo{cachedFunctions: cacheFunctions([]Function{
		fn("overloaded", 0, arg("a", "text", text), arg("b", "text", text), arg("c", "text", text)),
		fn("overloaded", 0),
		fn("overloaded", 0, arg("a", "integer", integer), arg("b", "integer", integer)),
		{Name: "overloaded", Schema: "test", HasUnnamed: true, Arguments: []Argument{{Type: "json", TypeId: 114}}},
		fn("overloaded_default", 1, arg("a", "integer", integer), arg("opt_param", "text", text)),
		fn("overloaded_default", 1, arg("opt_param", "text", text)),
		fn("overloaded_default", 0, arg("a", "integer", integer), arg("must_param", "integer", integer)),
		fn("overloaded_default", 0, arg("must_param", "integer", integer)),
		fn("overloaded_same_args", 1, arg("arg", "text", text), arg("num", "integer", integer)),
		fn("overloaded_same_args", 0, arg("arg", "xml", 142)),
		fn("overloaded_same_args", 0, arg("arg", "integer", integer)),
		fn("three_defaults", 3, arg("a", "integer", integer), arg("b", "integer", integer), arg("c", "integer", integer)),
		{Name: "ret_table", Schema: "test", HasOut: true, Arguments: []Argument{
			arg("p", "jsonb", 3802),
			{Name: "inserted", Type: "integer", Mode: 't', TypeId: integer},
			{Name: "deleted", Type: "integer", Mode: 't', TypeId: integer},
		}},
	})}

	signature := func(f *Function) string {
		var names []string
		for _, a := range f.inputArguments() {
			names = append(names, a.Name+" "+a.Type)
		}
		return f.Name + "(" + strings.Join(names, ", ") + ")"
	}
	tests := []struct {
		name string
		keys []string
		want string // the signature found, "" for none
	}{
		{"overloaded", nil, "overloaded()"},
		{"overloaded", []string{"a", "b"}, "overloaded(a integer, b integer)"},
		{"overloaded", []string{"a", "b", "c"}, "overloaded(a text, b text, c text)"},
		{"overloaded", []string{"a"}, ""},
		{"overloaded", []string{"wrong_arg"}, ""},
		{"overloaded", []string{"a", "b", "wrong_arg"}, ""},
		{"overloaded_default", nil, "overloaded_default(opt_param text)"},
		{"overloaded_default", []string{"opt_param"}, "overloaded_default(opt_param text)"},
		{"overloaded_default", []string{"must_param"}, "overloaded_default(must_param integer)"},
		{"overloaded_default", []string{"a"}, "overloaded_default(a integer, opt_param text)"},
		{"overloaded_default", []string{"a", "opt_param"}, "overloaded_default(a integer, opt_param text)"},
		{"overloaded_default", []string{"a", "must_param"}, "overloaded_default(a integer, must_param integer)"},
		{"three_defaults", nil, "three_defaults(a integer, b integer, c integer)"},
		{"three_defaults", []string{"b"}, "three_defaults(a integer, b integer, c integer)"},
		{"three_defaults", []string{"b", "d"}, ""},
		{"ret_table", []string{"p"}, "ret_table(p jsonb)"},
		{"ret_table", []string{"p", "inserted"}, ""},
		{"no_such_function", nil, ""},
	}
	for _, tt := range tests {
		f, err := info.FindFunction(_s(tt.name, "test"), tt.keys)
		if err != nil {
			t.Errorf("%s%v: %v", tt.name, tt.keys, err)
			continue
		}
		got := ""
		if f != nil {
			got = signature(f)
		}
		if got != tt.want {
			t.Errorf("%s%v: want %q, got %q", tt.name, tt.keys, tt.want, got)
		}
	}

	// three overloads accept arg alone; the message lists them as PostgREST
	// does, fewest arguments first (RpcSpec.hs, PGRST203)
	_, err := info.FindFunction("test.overloaded_same_args", []string{"arg"})
	var ambiguous *AmbiguousFunctionError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("overloaded_same_args(arg): want an AmbiguousFunctionError, got %v", err)
	}
	want := "Could not choose the best candidate function between: test.overloaded_same_args(arg => integer), test.overloaded_same_args(arg => xml), test.overloaded_same_args(arg => text, num => integer)"
	if ambiguous.Error() != want {
		t.Errorf("overloaded_same_args(arg):\n want %s\n got  %s", want, ambiguous.Error())
	}
	if ambiguous.Hint != "Try renaming the parameters or the function itself in the database so function overloading can be resolved" {
		t.Errorf("overloaded_same_args(arg): unexpected hint %q", ambiguous.Hint)
	}
	// with num the text overload is the only one left
	f, err := info.FindFunction("test.overloaded_same_args", []string{"arg", "num"})
	if err != nil || f == nil || signature(f) != "overloaded_same_args(arg text, num integer)" {
		t.Errorf("overloaded_same_args(arg, num): got %v, %v", f, err)
	}

	// a nil SchemaInfo finds nothing
	if f, err := (*SchemaInfo)(nil).FindFunction("test.overloaded", nil); f != nil || err != nil {
		t.Errorf("nil SchemaInfo: got %v, %v", f, err)
	}
}
