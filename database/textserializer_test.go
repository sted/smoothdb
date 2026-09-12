package database

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func BenchmarkSerializer(b *testing.B) {

	dbe_ctx, dbe_conn, _ := ContextWithDb(context.Background(), nil, "admin")
	defer ReleaseConn(dbe_ctx, dbe_conn)

	dbe.DeleteDatabase(dbe_ctx, "bench")
	db, err := dbe.GetOrCreateActiveDatabase(dbe_ctx, "bench")
	if err != nil {
		b.Fatal(err)
	}

	ctx, conn, _ := ContextWithDb(dbe_ctx, db, "admin")
	defer ReleaseConn(ctx, conn)

	CreateTable(ctx, &Table{Name: "b1", Columns: []Column{
		{Name: "name", Type: "text"},
		{Name: "number", Type: "integer"},
		{Name: "date", Type: "timestamp"},
	}})

	for i := 0; i < 10000; i++ {
		_, _, err := CreateRecords(ctx, "b1",
			[]Record{
				{"name": "Morpheus😆", "number": 42, "date": "2022-10-11T19:00"},
				{"name": "Sted", "number": 55, "date": nil},
			},
			nil)
		if err != nil {
			b.Fatal(err)
		}
	}
	gi := GetSmoothContext(ctx)
	info := gi.Db.info.Load()
	rows, err := gi.Conn.Query(ctx, "select * from b1")
	if err != nil {
		b.Fatal(err)
	}
	copiedRows, err := CopyRows(rows)
	if err != nil {
		b.Fatal(err)
	}
	defer rows.Close()

	b.Run("Serialize", func(b *testing.B) {
		serializer := gi.QueryBuilder.preferredSerializer()
		for i := 0; i < b.N; i++ {
			copiedRows.CurrentRow = -1
			_, _, err := serializer.Serialize(copiedRows, false, false, info)
			if err != nil {
				log.Print(err)
				return
			}
		}
	})

	b.Run("DynStructs", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			copiedRows.CurrentRow = -1
			_, err := rowsToDynStructs(copiedRows)
			if err != nil {
				log.Print(err)
				return
			}
			//fmt.Printf("%v", s)
		}
	})

	b.Run("DynStructPointers", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			copiedRows.CurrentRow = -1
			_, err := rowsToDynStructsWithPointers(copiedRows)
			if err != nil {
				log.Print(err)
				return
			}
		}
	})

	b.Run("Maps", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			copiedRows.CurrentRow = -1
			_, err := rowsToMaps(copiedRows)
			if err != nil {
				log.Print(err)
				return
			}
		}
	})
}

// The JSON serializer decodes raw pgx wire buffers with fixed offsets. A short
// or truncated buffer (a format/type mismatch, a driver change, an unexpected
// value) must be turned into a SerializeError, never a panic that unwinds the
// request goroutine. Each case below would index past the end of the buffer.
func TestSerializerMalformedBuffer(t *testing.T) {
	cases := []struct {
		name string
		oid  uint32
		buf  []byte
	}{
		{"int2 short", pgtype.Int2OID, []byte{0x00}},
		{"int8 short", pgtype.Int8OID, []byte{0x00}},
		{"float8 short", pgtype.Float8OID, []byte{0x00, 0x01}},
		{"timestamp short", pgtype.TimestampOID, []byte{0x00}},
		{"uuid short", pgtype.UUIDOID, []byte{0x01, 0x02}},
		{"interval short", pgtype.IntervalOID, []byte{0x00, 0x00, 0x00}},
	}
	for _, c := range cases {
		cr := &CustomRows{
			FieldDescriptions_: []pgconn.FieldDescription{{Name: "x", DataTypeOID: c.oid, Format: pgtype.BinaryFormatCode}},
			RawValues_:         [][][]byte{{c.buf}},
			CurrentRow:         -1,
		}
		s := &JSONSerializer{}
		if _, _, err := s.Serialize(cr, false, false, nil); err == nil {
			t.Errorf("%s: expected a SerializeError for a truncated buffer, got nil", c.name)
		}
	}
}

// A multi-dimensional array has a header/element layout the flat decoder does
// not implement. It must return an error rather than silently misparse.
func TestArrayMultiDimRejected(t *testing.T) {
	buf := make([]byte, 20)
	binary.BigEndian.PutUint32(buf[0:], 2) // 2 dimensions
	var j JSONSerializer
	if err := j.appendArray(buf, pgtype.Int4OID, nil, &j); err == nil {
		t.Error("expected error for a multi-dimensional array, got nil")
	}
}

// When the wire row carries a different field count than the cached composite
// definition (a stale cache), the decoder must error rather than read fields at
// the wrong offsets.
func TestCompositeFieldCountMismatch(t *testing.T) {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf[0:], 3) // wire says 3 fields
	typ := &Type{SubTypeIds: []uint32{pgtype.Int4OID}, SubTypeNames: []string{"a"}} // cache says 1
	var j JSONSerializer
	if err := j.appendComposite(buf, typ, nil, &j); err == nil {
		t.Error("expected error for a composite field-count mismatch, got nil")
	}
}

// A decoder error must propagate all the way out of Serialize, not just out of
// the decoder — the Serialize loop must surface appendType's return value.
func TestSerializeSurfacesDecoderError(t *testing.T) {
	const arrOID = 999999
	info := &SchemaInfo{cachedTypes: map[uint32]Type{
		arrOID: {Id: arrOID, IsArray: true, ArraySubType: pgtype.Int4OID},
	}}
	buf := make([]byte, 20)
	binary.BigEndian.PutUint32(buf[0:], 2) // 2 dimensions -> unsupported
	cr := &CustomRows{
		FieldDescriptions_: []pgconn.FieldDescription{{Name: "x", DataTypeOID: arrOID, Format: pgtype.BinaryFormatCode}},
		RawValues_:         [][][]byte{{buf}},
		CurrentRow:         -1,
	}
	if _, _, err := (&JSONSerializer{}).Serialize(cr, false, false, info); err == nil {
		t.Error("expected Serialize to surface the decoder error, got nil")
	}
}

// A range on the wire is a flag byte followed only by the bounds that exist:
// 'empty' and '(,)' carry no bound at all, a half-unbounded range carries just
// the one. The serializer must read the bounds the flags announce and nothing
// more, and print what PostgreSQL's to_json prints for the same value (the
// server canonicalizes int4range/int8range to the [) form). The bounded rows
// are the positive controls: they already serialize, so a failure there is a
// broken probe, not the defect. Integer and numeric subtypes are used because
// their bounds are printed bare, so the text is valid JSON independently of
// how the bounds are quoted.
func TestSerializeRangeBounds(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_ranges")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_ranges")
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

	_, err = CreateTable(ctx, &Table{
		Name: "ranges",
		Columns: []Column{
			{Name: "id", Type: "integer"},
			{Name: "r4", Type: "int4range"},
			{Name: "r8", Type: "int8range"},
			{Name: "rn", Type: "numrange"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Expected strings are what `select to_json(col)` returns for the value.
	cases := []struct {
		id   int
		in   string // the literal inserted in the three columns
		json string // JSON serializer output for select r4, r8, rn
		csv  string // CSV serializer output for the same select
	}{
		{1, "[1,10)", `[{"r4":"[1,10)","r8":"[1,10)","rn":"[1,10)"}]`, "r4,r8,rn\n\"[1,10)\",\"[1,10)\",\"[1,10)\""},
		{2, "[1,10]", `[{"r4":"[1,11)","r8":"[1,11)","rn":"[1,10]"}]`, "r4,r8,rn\n\"[1,11)\",\"[1,11)\",\"[1,10]\""},
		{3, "(1,5]", `[{"r4":"[2,6)","r8":"[2,6)","rn":"(1,5]"}]`, "r4,r8,rn\n\"[2,6)\",\"[2,6)\",\"(1,5]\""},
		{4, "[10,)", `[{"r4":"[10,)","r8":"[10,)","rn":"[10,)"}]`, "r4,r8,rn\n\"[10,)\",\"[10,)\",\"[10,)\""},
		{5, "(,10)", `[{"r4":"(,10)","r8":"(,10)","rn":"(,10)"}]`, "r4,r8,rn\n\"(,10)\",\"(,10)\",\"(,10)\""},
		{6, "(,)", `[{"r4":"(,)","r8":"(,)","rn":"(,)"}]`, "r4,r8,rn\n\"(,)\",\"(,)\",\"(,)\""},
		// PostgreSQL's record text (what PostgREST returns as CSV) leaves
		// 'empty' unquoted, having no separator in it; the CSV value is
		// compared without its optional quotes so that this test pins the
		// bound reading, not the quoting of the range.
		{7, "empty", `[{"r4":"empty","r8":"empty","rn":"empty"}]`, "r4,r8,rn\nempty,empty,empty"},
	}
	for _, c := range cases {
		_, err = gi.Conn.Exec(ctx, fmt.Sprintf("insert into ranges values (%d, '%s', '%s', '%s')", c.id, c.in, c.in, c.in))
		if err != nil {
			t.Fatal(err)
		}
	}
	info := gi.Db.info.Load()
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			query := fmt.Sprintf("select r4, r8, rn from ranges where id = %d", c.id)

			rows, err := gi.Conn.Query(ctx, query)
			if err != nil {
				t.Fatal(err)
			}
			out, _, err := (&JSONSerializer{}).Serialize(rows, false, false, info)
			rows.Close()
			if err != nil {
				t.Errorf("JSON: unexpected error: %v", err)
			} else if string(out) != c.json {
				t.Errorf("JSON: expected %s, got %s", c.json, out)
			}

			rows, err = gi.Conn.Query(ctx, query)
			if err != nil {
				t.Fatal(err)
			}
			out, _, err = (&CSVSerializer{}).Serialize(rows, false, false, info)
			rows.Close()
			got := string(out)
			if c.in == "empty" {
				got = strings.ReplaceAll(got, `"`, "")
			}
			if err != nil {
				t.Errorf("CSV: unexpected error: %v", err)
			} else if got != c.csv {
				t.Errorf("CSV: expected %q, got %q", c.csv, got)
			}
		})
	}
}

// pgx asks the server for each result column in the format its type map
// prefers: binary for the types it has a binary codec for (bytea, inet, cidr,
// macaddr, time, point, bit, the geometric types, tsvector, arrays of them),
// text for the rest (enums, custom types, extension types). The serializer
// must dispatch on FieldDescription.Format: it used to assume text for every
// OID outside its binary switch and copy the raw binary bytes into the JSON
// string. The oracle is PostgreSQL's own row_to_json, which is byte for byte
// what PostgREST returns (its body is json_agg of the row). The id, n, txt,
// f8_arr and txt_arr columns are positive controls: they serialize today, so a
// mismatch there is a broken probe, not the defect.
//
// The table is read twice: on the connection that created the types, where
// pgx knows none of them (every custom type arrives in text), and on a fresh
// connection after a schema cache reload, where AfterConnect has registered
// the composites (pair arrives in binary; blob stays text because it has
// text-only fields).
func TestSerializeWireFormats(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_formats")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_formats")
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
		`create type mood as enum ('sad', 'ok', 'happy')`,
		// labels that need quoting and escaping in an array literal, and one
		// that spells NULL
		`create type weird as enum ('a b', 'c"d', 'e\f', 'NULL', '{x}')`,
		`create type pair as (n int, s text)`,
		`create type blob as (id int, data bytea, at time, inner_ pair, tags text[], j jsonb, ok bool, f float8, dec numeric, ips inet[])`,
		`create table formats (
			id int primary key,
			n int, txt text, f8_arr float8[], txt_arr text[], jb jsonb, jb_arr jsonb[],
			by bytea, ip inet, net cidr, mac macaddr, mac8 macaddr8, tm time, tmz timetz,
			pt point, ln line, ls lseg, bx box, pa path, pg polygon, ci circle,
			bt bit(4), vb varbit, ch "char", ti tid, xd xid, cd cid, x8 xid8,
			by_arr bytea[], by_2d bytea[][], ip_arr inet[], tm_arr time[], bx_arr box[],
			md mood, md_arr mood[], wd_arr weird[], tsv tsvector, tsv_arr tsvector[],
			pr pair, bl blob, pr_arr pair[], bl_arr blob[]
		)`,
		`insert into formats values (
			1,
			42, 'plain', '{1.5,2.5}', '{a,b}', '{"k": "v"}', '{"{\"a\": 1}","[true]"}',
			'\x0102', '192.168.1.1', '192.168.1.0/24', '08:00:2b:01:02:03', '08:00:2b:01:02:03:04:05', '12:34:56.5', '12:34:56+02',
			'(1,2)', '{1,2,3}', '[(1,2),(3,4)]', '((1,2),(3,4))', '[(1,2),(3,4)]', '((1,2),(3,4),(5,6))', '<(1,2),3>',
			B'1010', B'10', 'a', '(1,2)', '123', '456', '789',
			'{"\\x01","\\x02"}', '{{"\\x01"},{"\\x02"}}', '{10.0.0.1,"::1"}', '{01:02:03,04:05:06}', '{"(1,2),(0,0)";"(3,4),(2,2)"}',
			'happy', '{sad,happy}', '{"a b","c\"d","e\\f","NULL",NULL,"{x}"}', 'a b', '{"a b","c"}',
			'(1,"x")', '(1,"\\x01",12:00:00,"(2,""y"")","{a,""b c""}","{""k"": 1}",t,1.5,1.50,"{10.0.0.1}")',
			'{"(1,x)","(2,\"a \"\"q\"\" (b), \\\\ c\")"}', '{"(1,\"\\\\x01\",,\"(2,y)\",{a},[1],f,-0.5,,{})"}'
		)`,
		`insert into formats (id) values (2)`,
		// empty arrays, NULL elements and empty values
		`insert into formats (id, txt_arr, by, by_arr, ip_arr, md_arr, bl_arr) values (
			3, '{"a,b","c\"d","e\\f",NULL,""}', '\x', '{"\\x",NULL}', '{}', '{NULL}', '{NULL}'
		)`,
	}
	for _, q := range ddl {
		if _, err := gi.Conn.Exec(ctx, q); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	if err := db.ReloadSchemaCache(ctx); err != nil {
		t.Fatal(err)
	}

	// One projection per family, so that a failure names the family and the
	// binary-only refusals (a multi-dimensional bytea[][] before the fix) do
	// not hide the mojibake of the scalar columns. Expected bodies come from
	// PostgreSQL itself.
	projections := []struct{ name, columns string }{
		{"controls", "id, n, txt, f8_arr, txt_arr, jb"},
		{"card types", "by, ip, net, mac, mac8, tm, tmz, pt, bt, vb, ch"},
		{"geometric and system types", "ln, ls, bx, pa, pg, ci, ti, xd, cd, x8"},
		{"arrays", "by_arr, by_2d, ip_arr, tm_arr, bx_arr, md_arr, wd_arr, tsv_arr, jb_arr"},
		{"enums and composites", "md, tsv, pr, bl, pr_arr, bl_arr"},
		{"whole row", "*"},
	}
	expected := map[string]string{}
	for _, pr := range projections {
		for _, id := range []int{1, 2, 3} {
			var row string
			q := fmt.Sprintf("select row_to_json(t)::text from (select %s from formats where id = %d) t", pr.columns, id)
			if err := gi.Conn.QueryRow(ctx, q).Scan(&row); err != nil {
				t.Fatal(err)
			}
			expected[fmt.Sprintf("%s/row%d", pr.name, id)] = "[" + row + "]"
		}
	}

	check := func(t *testing.T, ctx context.Context) {
		gi := GetSmoothContext(ctx)
		info := gi.Db.info.Load()
		for _, pr := range projections {
			for _, id := range []int{1, 2, 3} {
				name := fmt.Sprintf("%s/row%d", pr.name, id)
				t.Run(name, func(t *testing.T) {
					rows, err := gi.Conn.Query(ctx, fmt.Sprintf("select %s from formats where id = %d", pr.columns, id))
					if err != nil {
						t.Fatal(err)
					}
					out, _, err := (&JSONSerializer{}).Serialize(rows, false, false, info)
					rows.Close()
					if err != nil {
						t.Fatalf("unexpected error: %v", err)
					}
					if got := unescapeHTML(string(out)); got != expected[name] {
						t.Errorf("expected %s\n     got %s", expected[name], got)
					}
				})
			}
		}
		// The CSV serializer shares the dispatch; a text-format value is
		// PostgreSQL's text for it, which for an array is the literal that
		// PostgREST's CSV carries too (quoted here for its commas and quotes).
		t.Run("csv", func(t *testing.T) {
			rows, err := gi.Conn.Query(ctx, "select by, ip, net, mac, tm, pt, bt, md, by_arr from formats where id = 1")
			if err != nil {
				t.Fatal(err)
			}
			out, _, err := (&CSVSerializer{}).Serialize(rows, false, false, info)
			rows.Close()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := "by,ip,net,mac,tm,pt,bt,md,by_arr\n" +
				`\x0102,192.168.1.1,192.168.1.0/24,08:00:2b:01:02:03,12:34:56.5,"(1,2)",1010,happy,"{""\\x01"",""\\x02""}"`
			if string(out) != want {
				t.Errorf("expected %q\n     got %q", want, out)
			}
		})
	}

	t.Run("types unknown to the connection", func(t *testing.T) { check(t, ctx) })
	ReleaseConn(ctx, conn)

	// A fresh connection runs AfterConnect with the reloaded schema cache.
	db.pool.Reset()
	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)
	t.Run("composites registered", func(t *testing.T) { check(t, ctx) })
}

// smoothdb escapes the HTML-significant characters in JSON strings, as Go's
// encoding/json does; row_to_json does not. Same JSON, so the comparison
// undoes it.
func unescapeHTML(s string) string {
	return strings.NewReplacer(`\u003c`, "<", `\u003e`, ">", `\u0026`, "&").Replace(s)
}

// The text-format converters, DB-free: PostgreSQL's array and record literal
// grammars (quotes, backslash and doubled-quote escapes, NULL, empty and
// nested arrays, the dimension prefix, the box delimiter) and the scalar rule
// (json verbatim, t/f as booleans, numbers bare unless NaN or infinite, all
// else a JSON string), plus the loud failure for a binary value the
// serializer cannot decode.
func TestSerializeTextFormat(t *testing.T) {
	const (
		enumOID    = 900001
		enumArrOID = 900002
		pairOID    = 900003
		pairArrOID = 900004
		int4ArrOID = pgtype.Int4ArrayOID
		boxArrOID  = pgtype.BoxArrayOID
		unknownOID = 900005
	)
	info := &SchemaInfo{cachedTypes: map[uint32]Type{
		enumOID:    {Id: enumOID, Name: "mood", IsEnum: true},
		enumArrOID: {Id: enumArrOID, Name: "_mood", IsArray: true, ArraySubType: enumOID},
		pairOID:    {Id: pairOID, Name: "pair", IsComposite: true, SubTypeIds: []uint32{pgtype.Int4OID, pgtype.TextOID, enumArrOID}, SubTypeNames: []string{"n", "s", "moods"}},
		pairArrOID: {Id: pairArrOID, Name: "_pair", IsArray: true, ArraySubType: pairOID},
		int4ArrOID: {Id: int4ArrOID, Name: "_int4", IsArray: true, ArraySubType: pgtype.Int4OID},
		boxArrOID:  {Id: boxArrOID, Name: "_box", IsArray: true, ArraySubType: pgtype.BoxOID},
		unknownOID: {Id: unknownOID, Name: "mystery"},
	}}
	cases := []struct {
		name string
		oid  uint32
		in   string
		want string // "" means a SerializeError is expected
	}{
		{"unknown scalar", unknownOID, `a "q" \ b`, `"a \"q\" \\ b"`},
		{"json verbatim", pgtype.JSONBOID, `{"k": [1, null]}`, `{"k": [1, null]}`},
		{"bool t", pgtype.BoolOID, `t`, `true`},
		{"bool f", pgtype.BoolOID, `f`, `false`},
		{"int", pgtype.Int8OID, `-12`, `-12`},
		{"float", pgtype.Float8OID, `1.5e-07`, `1.5e-07`},
		{"numeric", pgtype.NumericOID, `1.50`, `1.50`},
		{"NaN", pgtype.Float8OID, `NaN`, `"NaN"`},
		{"-Infinity", pgtype.NumericOID, `-Infinity`, `"-Infinity"`},
		{"enum", enumOID, `a b`, `"a b"`},
		{"empty array", enumArrOID, `{}`, `[]`},
		{"array", enumArrOID, `{sad,"a b","c\"d","e\\f","NULL",NULL,"{x}"}`, `["sad","a b","c\"d","e\\f","NULL",null,"{x}"]`},
		{"int array with dimension prefix", int4ArrOID, `[0:1]={1,2}`, `[1,2]`},
		{"nested array", int4ArrOID, `{{1,2},{3,4}}`, `[[1,2],[3,4]]`},
		{"box array uses ; as delimiter", boxArrOID, `{(1,2),(0,0);(3,4),(2,2)}`, `["(1,2),(0,0)","(3,4),(2,2)"]`},
		{"composite", pairOID, `(1,"a ""q"" \\ (b)","{sad,""a b""}")`, `{"n":1,"s":"a \"q\" \\ (b)","moods":["sad","a b"]}`},
		{"composite with nulls", pairOID, `(,,)`, `{"n":null,"s":null,"moods":null}`},
		{"composite with empty string", pairOID, `(1,"",{})`, `{"n":1,"s":"","moods":[]}`},
		{"array of composites", pairArrOID, `{"(1,x,)","(2,\"a \\\\ \\\"q\\\"\",\"{sad}\")"}`, `[{"n":1,"s":"x","moods":null},{"n":2,"s":"a \\ \"q\"","moods":["sad"]}]`},
		{"malformed array", enumArrOID, `{a`, ""},
		{"malformed array trailer", enumArrOID, `{a}b`, ""},
		{"malformed composite", pairOID, `(1,x`, ""},
		{"composite with too many fields", pairOID, `(1,x,{},4)`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cr := &CustomRows{
				FieldDescriptions_: []pgconn.FieldDescription{{Name: "x", DataTypeOID: c.oid, Format: pgtype.TextFormatCode}},
				RawValues_:         [][][]byte{{[]byte(c.in)}},
				CurrentRow:         -1,
			}
			out, _, err := (&JSONSerializer{}).Serialize(cr, false, false, info)
			if c.want == "" {
				if err == nil {
					t.Errorf("expected a SerializeError, got %s", out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if want := `[{"x":` + c.want + `}]`; string(out) != want {
				t.Errorf("expected %s, got %s", want, out)
			}
		})
	}

	// A binary value of a type the serializer has no decoder for must be a
	// SerializeError naming the type, never a copy-through as if it were text.
	// The text-format case above is the positive control for the same OID.
	t.Run("unknown binary type fails loudly", func(t *testing.T) {
		for _, oid := range []uint32{unknownOID, pgtype.RecordOID} {
			cr := &CustomRows{
				FieldDescriptions_: []pgconn.FieldDescription{{Name: "x", DataTypeOID: oid, Format: pgtype.BinaryFormatCode}},
				RawValues_:         [][][]byte{{{0x00, 0x00, 0x00, 0x01}}},
				CurrentRow:         -1,
			}
			out, _, err := (&JSONSerializer{}).Serialize(cr, false, false, info)
			if err == nil {
				t.Errorf("oid %d: expected a SerializeError, got %s", oid, out)
			} else if !strings.Contains(err.Error(), fmt.Sprint(oid)) {
				t.Errorf("oid %d: the error should name the OID, got %q", oid, err)
			}
			cr.Close() // rewinds the rows
			out, _, err = (&CSVSerializer{}).Serialize(cr, false, false, info)
			if err == nil {
				t.Errorf("oid %d: CSV: expected a SerializeError, got %s", oid, out)
			}
		}
	})
}
