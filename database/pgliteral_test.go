package database

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"testing"
)

// Card 38672 csv-arrays. The CSV output wrote a binary array with the JSON
// array writer ([1,2,3]) and a composite as a JSON object, both unquoted, so
// their commas split the field. PostgREST's CSV carries PostgreSQL's text
// for the value (the record text of the row): {1,2,3}, quoted as one CSV
// field. The oracle is PostgreSQL itself: each column must equal its ::text,
// CSV-encoded. It reads on a fresh connection after a schema cache reload,
// where the composite pair arrives in binary.
func TestCSVArraysAndCompositesAsPostgreSQLText(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_csvlit")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_csvlit")
	if err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)
	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	gi := GetSmoothContext(ctx)
	columns := []struct{ name, def, value string }{
		{"i4", "int[]", `'{1,NULL,3}'`},
		{"e", "int[]", `'{}'`},
		{"i2d", "int[]", `'{{1,2},{3,4}}'`},
		{"i3d", "int[]", `'{{{1,2}},{{3,NULL}}}'`},
		{"lb", "int[]", `'[0:1][2:3]={{1,2},{3,4}}'`},
		{"neg", "int[]", `'[-2:-1]={5,6}'`},
		{"i8", "int8[]", `'{-9223372036854775808,9223372036854775807}'`},
		{"i2", "int2[]", `'{-1,2}'`},
		{"txt", "text[]", `'{a,"b c","",NULL,"NULL","null","x,y","q\"uote","back\\slash","{br}","(p)","tab\there"}'`},
		{"vc", "varchar[]", `'{x,"y z"}'`},
		{"bp", "char(3)[]", `'{ab,c}'`},
		{"b", "bool[]", `'{t,f,NULL}'`},
		{"f8", "float8[]", `'{1.5,NaN,Infinity,-Infinity,1e+100,1234567,123456789012345,1e15,0.1,-0,1e-05}'`},
		{"f4", "float4[]", `'{1.5,0.1,1234567,123456}'`},
		{"num", "numeric[]", `'{1.50,NaN,-3,0.001}'`},
		{"u", "uuid[]", `'{a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11,NULL}'`},
		{"d", "date[]", `'{2015-12-08,infinity,NULL}'`},
		{"ts", "timestamp[]", `'{"2015-12-08 04:22:57.472738","2015-12-08 04:22:57",infinity}'`},
		{"tz", "timestamptz[]", `'{"2015-12-08 04:22:57.472738+00","2015-12-08 04:22:57+00"}'`},
		{"jb", "jsonb[]", `'{"{\"k\": 1}","[1, 2]",null}'`},
		{"js", "json[]", `'{"{\"k\":1}"}'`},
		{"pr", "pair", `'(1,"x y")'`},
		{"prn", "pair", `'(2,)'`},
		{"prq", "pair", `'(3,"a \"q\" (b), \\ c")'`},
		{"pre", "pair", `'(4,"")'`},
		{"pra", "pair[]", `'{"(1,x)","(2,\"a,b\")",NULL}'`},
		{"h", "holder", `'(1,"{a,\"b c\"}","(2,z)")'`},
		{"vt", "pair", `row(5, 'a' || chr(11) || 'b')::pair`},
		{"vta", "text[]", `array['a' || chr(11) || 'b']`},
		{"dbc", "date[]", `'{"0001-01-01 BC","0044-03-15 BC",2015-01-01}'`},
		{"tbc", "timestamp[]", `'{"0044-03-15 12:00:00.5 BC"}'`},
		{"zbc", "timestamptz[]", `'{"0044-03-15 12:00:00+00 BC"}'`},
	}
	ddl := []string{
		`create type pair as (n int, s text)`,
		`create type holder as (id int, tags text[], p pair)`,
	}
	var defs, names, values []string
	for _, c := range columns {
		defs = append(defs, c.name+" "+c.def)
		names = append(names, c.name)
		values = append(values, c.value)
	}
	ddl = append(ddl,
		"create table lit ("+strings.Join(defs, ", ")+")",
		"insert into lit values ("+strings.Join(values, ", ")+")")
	for _, q := range ddl {
		if _, err := gi.Conn.Exec(ctx, q); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	if err := db.ReloadSchemaCache(ctx); err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)
	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)
	gi = GetSmoothContext(ctx)
	info := gi.Db.info.Load()
	// timestamptz is written in UTC (card 38673 timestamptz-utc): compare
	// under a UTC session
	if _, err := gi.Conn.Exec(ctx, "set timezone = 'UTC'"); err != nil {
		t.Fatal(err)
	}
	defer gi.Conn.Exec(context.Background(), "reset timezone")

	for _, c := range columns {
		t.Run(c.name, func(t *testing.T) {
			var text *string
			if err := gi.Conn.QueryRow(ctx, fmt.Sprintf("select %s::text from lit", c.name)).Scan(&text); err != nil {
				t.Fatal(err)
			}
			var want bytes.Buffer
			w := csv.NewWriter(&want)
			field := ""
			if text != nil {
				field = *text
			}
			w.Write([]string{c.name})
			w.Write([]string{field})
			w.Flush()
			rows, err := gi.Conn.Query(ctx, fmt.Sprintf("select %s from lit", c.name))
			if err != nil {
				t.Fatal(err)
			}
			out, _, err := (&CSVSerializer{}).Serialize(rows, false, false, info)
			rows.Close()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := string(out); got != strings.TrimSuffix(want.String(), "\n") {
				t.Errorf("expected %q\n     got %q", strings.TrimSuffix(want.String(), "\n"), got)
			}
		})
	}
	// the whole row: one CSV field per column, as many as the header
	t.Run("whole row", func(t *testing.T) {
		rows, err := gi.Conn.Query(ctx, "select * from lit")
		if err != nil {
			t.Fatal(err)
		}
		out, _, err := (&CSVSerializer{}).Serialize(rows, false, false, info)
		rows.Close()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		records, err := csv.NewReader(bytes.NewReader(out)).ReadAll()
		if err != nil {
			t.Fatalf("the CSV does not parse: %v\n%s", err, out)
		}
		if len(records) != 2 || len(records[1]) != len(names) {
			t.Errorf("want a header and one row of %d fields, got %d records: %q", len(names), len(records), records)
		}
	})
}

// float8out writes a float8 in plain notation up to an exponent of 14
// (DBL_DIG); Go's shortest form switches at 6, so 1234567 came out as
// 1.234567e+06 in JSON, where to_json prints 1234567. float4 switches at 6
// in both.
func TestSerializeFloat8LikeToJSON(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)
	toJSONControl(t, ctx, `select 1234567::float8 a, 123456789012345::float8 b, 1e15::float8 c,
		1e-05::float8 d, 0.0001::float8 e, -7654321.5::float8 f, 1.5::float8 g,
		1234567::float4 h, 123456::float4 i, '{1234567,1e20}'::float8[] j`)
	// a BC date as to_json writes it
	toJSONControl(t, ctx, `select '0001-01-01 BC'::date a, '{"0044-03-15 BC",2015-01-01}'::date[] b`)
}
