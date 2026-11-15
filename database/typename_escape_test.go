package database

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

// An enum label and a composite attribute name are names that a role with DDL
// rights chooses: they can hold a quote, a backslash, a comma or a line break.
// The binary decoders wrote them into JSON and CSV without escapes, so such a
// name broke the body. The oracle for JSON is PostgreSQL's row_to_json; the
// CSV quotes a field as for any other value. The table is read on a fresh
// connection after a schema cache reload, where the composite arrives in
// binary (its fields all have binary codecs). An enum arrives in text, which
// is escaped already: the enum column is a control, and the binary enum
// decoder is called directly. The plain names are positive controls.
func TestSerializeEscapesTypeNames(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_typenames")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_typenames")
	if err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)

	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	gi := GetSmoothContext(ctx)
	ddl := []string{
		`create type odd as enum ('plain', 'a"b', 'c\d', 'e,f', E'g\nh')`,
		`create type named as ("plain" int, "q""x" int, "b\s" text, "n,m" text)`,
		`create table names (id int primary key, o odd, n named)`,
		`insert into names values
			(1, 'plain', '(1,2,x,y)'),
			(2, 'a"b', '(1,2,x,y)'),
			(3, 'c\d', '(1,2,x,y)'),
			(4, 'e,f', '(1,2,x,y)'),
			(5, E'g\nh', '(1,2,x,y)')`,
	}
	for _, q := range ddl {
		if _, err := gi.Conn.Exec(ctx, q); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	if err := db.ReloadSchemaCache(ctx); err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)

	// a fresh connection registers the composite: it arrives in binary
	db.pool.Reset()
	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)
	gi = GetSmoothContext(ctx)
	info := gi.Db.info.Load()

	for id := 1; id <= 5; id++ {
		var want string
		if err := gi.Conn.QueryRow(ctx, "select '[' || row_to_json(t)::text || ']' from (select o, n from names where id = $1) t", id).Scan(&want); err != nil {
			t.Fatal(err)
		}
		rows, err := gi.Conn.Query(ctx, "select o, n from names where id = $1", id)
		if err != nil {
			t.Fatal(err)
		}
		// the probe: the composite must arrive in binary
		if f := rows.FieldDescriptions()[1].Format; f != pgx.BinaryFormatCode {
			t.Fatalf("the composite arrived in format %d, not in binary", f)
		}
		out, _, err := (&JSONSerializer{}).Serialize(rows, false, false, info)
		rows.Close()
		if err != nil {
			t.Errorf("row %d: JSON: %v", id, err)
			continue
		}
		if got := unescapeHTML(string(out)); got != want {
			t.Errorf("row %d: JSON\nexpected %s\n     got %s", id, want, got)
		}
	}

	rows, err := gi.Conn.Query(ctx, "select o from names order by id")
	if err != nil {
		t.Fatal(err)
	}
	out, _, err := (&CSVSerializer{}).Serialize(rows, false, false, info)
	rows.Close()
	if err != nil {
		t.Fatalf("CSV: %v", err)
	}
	wantCSV := "o\nplain\n\"a\"\"b\"\nc\\d\n\"e,f\"\n\"g\nh\""
	if string(out) != wantCSV {
		t.Errorf("CSV\nexpected %q\n     got %q", wantCSV, out)
	}

	// the binary enum decoder, with a label in the bytes of its wire value
	var enumOID uint32
	if err := gi.Conn.QueryRow(ctx, "select 'odd'::regtype::oid").Scan(&enumOID); err != nil {
		t.Fatal(err)
	}
	labels := []struct{ label, json, csv string }{
		{"plain", `"plain"`, "plain"},
		{`a"b`, `"a\"b"`, `"a""b"`},
		{`c\d`, `"c\\d"`, `c\d`},
		{"e,f", `"e,f"`, `"e,f"`},
		{"g\nh", `"g\nh"`, "\"g\nh\""},
	}
	for _, l := range labels {
		j := &JSONSerializer{}
		if err := j.appendType([]byte(l.label), enumOID, info); err != nil {
			t.Fatal(err)
		}
		if got := j.String(); got != l.json {
			t.Errorf("binary enum %q: JSON %s, expected %s", l.label, got, l.json)
		}
		c := &CSVSerializer{}
		if err := c.appendType([]byte(l.label), enumOID, info); err != nil {
			t.Fatal(err)
		}
		if got := c.String(); got != l.csv {
			t.Errorf("binary enum %q: CSV %q, expected %q", l.label, got, l.csv)
		}
	}
}
