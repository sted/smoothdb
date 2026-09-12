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

// 'empty', '[10,)', '(,10)' and '(,)' used to crash the binary range decoder,
// which read both bounds unconditionally; a range now arrives in text
// (textFormatOnlyTypes) and every shape must print what PostgreSQL's to_json
// prints for the same value (the server canonicalizes int4range/int8range to
// the [) form). The bounded rows are the positive controls: they already
// serialized, so a failure there is a broken probe, not the defect. Integer
// and numeric subtypes are used because their bounds are printed bare, so the
// text is valid JSON independently of how the bounds are quoted.
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
		// 'empty' unquoted, having no separator in it.
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
			if err != nil {
				t.Errorf("CSV: unexpected error: %v", err)
			} else if string(out) != c.csv {
				t.Errorf("CSV: expected %q, got %q", c.csv, out)
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
		rangeOID   = pgtype.DaterangeOID
		multiOID   = pgtype.Int4multirangeOID
	)
	dateOID := uint32(pgtype.DateOID)
	info := &SchemaInfo{cachedTypes: map[uint32]Type{
		enumOID:    {Id: enumOID, Name: "mood", IsEnum: true},
		enumArrOID: {Id: enumArrOID, Name: "_mood", IsArray: true, ArraySubType: enumOID},
		pairOID:    {Id: pairOID, Name: "pair", IsComposite: true, SubTypeIds: []uint32{pgtype.Int4OID, pgtype.TextOID, enumArrOID}, SubTypeNames: []string{"n", "s", "moods"}},
		pairArrOID: {Id: pairArrOID, Name: "_pair", IsArray: true, ArraySubType: pairOID},
		int4ArrOID: {Id: int4ArrOID, Name: "_int4", IsArray: true, ArraySubType: pgtype.Int4OID},
		boxArrOID:  {Id: boxArrOID, Name: "_box", IsArray: true, ArraySubType: pgtype.BoxOID},
		unknownOID: {Id: unknownOID, Name: "mystery"},
		rangeOID:   {Id: rangeOID, Name: "daterange", IsRange: true, RangeSubType: &dateOID},
		multiOID:   {Id: multiOID, Name: "int4multirange", IsMultirange: true},
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
		// a range is one JSON string, range_out's quotes escaped
		{"range", pgtype.DaterangeOID, `[2024-01-01,2024-06-01)`, `"[2024-01-01,2024-06-01)"`},
		{"range with quoted bounds", pgtype.TsrangeOID, `["2024-01-01 10:00:00","2024-06-01 12:00:00")`, `"[\"2024-01-01 10:00:00\",\"2024-06-01 12:00:00\")"`},
		// so is a multirange, multirange_out's braces around the ranges
		{"multirange", multiOID, `{[1,3),[5,7)}`, `"{[1,3),[5,7)}"`},
		{"empty multirange", multiOID, `{}`, `"{}"`},
		{"multirange with quoted bounds", pgtype.TsmultirangeOID, `{["2024-01-01 10:00:00","2024-06-01 12:00:00")}`, `"{[\"2024-01-01 10:00:00\",\"2024-06-01 12:00:00\")}"`},
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
	// The text-format case above is the positive control for the same OID. A
	// range is one of them: it has no binary decoder, since only PostgreSQL
	// prints its bounds as range_out does, and arrives in text. So is a
	// multirange.
	t.Run("unknown binary type fails loudly", func(t *testing.T) {
		for _, oid := range []uint32{unknownOID, pgtype.RecordOID, rangeOID, multiOID} {
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

// A range prints as PostgreSQL's range_out text, which to_json (and so
// PostgREST) returns as one JSON string: a bound is quoted only when its text
// contains a bracket, a parenthesis, a comma, a quote, a backslash or
// whitespace, or is empty, and a quote inside it is doubled. int4range,
// int8range and numrange print their bounds bare, so they were valid JSON by
// luck and are the positive controls here; a daterange is unquoted too but
// was emitted with quoted bounds, and tsrange, tstzrange and a range over
// text carry quotes that must be escaped in the JSON string, at the top level
// and inside arrays and composites alike. The oracle is row_to_json, as in
// TestSerializeWireFormats, on the creating connection (pgx knows no custom
// type) and on a fresh one after a schema cache reload, where the composites
// are registered: one of them pairs the custom range, which pgx does not
// know, with an int, and must still come out as PostgreSQL prints it.
func TestSerializeRangeQuoting(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_range_quoting")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_range_quoting")
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
		`create type textrange as range (subtype = text)`,
		`create type period as (n int, days daterange, span tsrange)`,
		`create type labelled as (label textrange, nested period)`,
		// a custom range, unknown to pgx, next to a field it decodes in binary
		`create type tagged as (n int, label textrange)`,
		`create table ranges (
			id int primary key,
			n int, r4 int4range, r8 int8range, rn numrange,
			rd daterange, rts tsrange, rtz tstzrange, rt textrange,
			rd_arr daterange[], rts_arr tsrange[], rt_arr textrange[],
			pd period, pd_arr period[], lb labelled, tg tagged
		)`,
		`insert into ranges values (
			1,
			42, '[1,10)', '[1,10]', '[1.5,2.5]',
			'[2024-01-01,2024-06-01)', '[2024-01-01 10:00:00,2024-06-01 12:00:00)', '[2024-01-01 10:00:00+00,)', '["a,b","c\"d"]',
			'{"[2024-01-01,2024-06-01)","[2024-02-01,)"}', '{"[\"2024-01-01 10:00:00\",\"2024-06-01 12:00:00\")",NULL}', '{"[\"a,b\",\"c\\\"d\"]"}',
			'(1,"[2024-01-01,2024-06-01)","[""2024-01-01 10:00:00"",""2024-06-01 12:00:00"")")',
			'{"(2,\"[2024-02-01,)\",empty)"}',
			'("[""a,b"",""c""""d""]","(3,\"[2024-03-01,2024-04-01)\",\"[\"\"2024-03-01 00:00:00\"\",)\")")',
			'(7,"[""a,b"",""c""""d""]")'
		)`,
		`insert into ranges (id) values (2)`,
		// the shapes with no bound to quote, an empty-string bound, and a
		// bound with a fractional second
		`insert into ranges (id, rd, rts, rtz, rt, rd_arr, pd, tg) values (
			3, 'empty', '(,"2024-06-01 12:00:00.5")', '(,)', '["",z]', '{}', '(,,)', '(,)'
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

	projections := []struct{ name, columns string }{
		{"controls", "id, n, r4, r8, rn"},
		{"quoted subtypes", "rd, rts, rtz, rt"},
		{"arrays", "rd_arr, rts_arr, rt_arr"},
		{"composites", "pd, pd_arr, lb, tg"},
		{"whole row", "*"},
	}
	expected := map[string]string{}
	for _, pr := range projections {
		for _, id := range []int{1, 2, 3} {
			var row string
			q := fmt.Sprintf("select row_to_json(t)::text from (select %s from ranges where id = %d) t", pr.columns, id)
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
					rows, err := gi.Conn.Query(ctx, fmt.Sprintf("select %s from ranges where id = %d", pr.columns, id))
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
		// The CSV value is PostgreSQL's text for the range, quoted for its
		// commas and quotes as PostgREST's CSV (the record text) quotes it.
		t.Run("csv", func(t *testing.T) {
			rows, err := gi.Conn.Query(ctx, "select rd, rts, rt, rn from ranges where id = 1")
			if err != nil {
				t.Fatal(err)
			}
			out, _, err := (&CSVSerializer{}).Serialize(rows, false, false, info)
			rows.Close()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := "rd,rts,rt,rn\n" +
				`"[2024-01-01,2024-06-01)","[""2024-01-01 10:00:00"",""2024-06-01 12:00:00"")","[""a,b"",""c""""d""]","[1.5,2.5]"`
			if string(out) != want {
				t.Errorf("expected %q\n     got %q", want, out)
			}
		})
	}

	t.Run("types unknown to the connection", func(t *testing.T) { check(t, ctx) })
	ReleaseConn(ctx, conn)

	db.pool.Reset()
	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)
	t.Run("composites registered", func(t *testing.T) { check(t, ctx) })
}

// A multirange prints as PostgreSQL's multirange_out text, which to_json (and
// so PostgREST) returns as one JSON string: the ranges between braces, each
// with range_out's quoting of its bounds, '{}' for the empty one. pgx requests
// the six builtin multiranges in binary (its multirange codec follows the
// range codec it captured when its default map was built, before the ranges
// were re-registered as text-only) and the serializers have no binary decoder
// for them, so every such column fails; the introspection query also takes a
// multirange for a range with no subtype (typcategory 'R', but its pg_range
// row is the one of its range, under rngmultitypid). The oracle is
// row_to_json, as in TestSerializeRangeQuoting, on the creating connection
// (pgx knows no custom type) and on a fresh one after a schema cache reload,
// where the composites are registered: one has a builtin multirange field and
// must be requested in text like the ranges, the other pairs textmultirange
// (created with textrange, unknown to pgx and text already) with an int. The
// int4range column and the ints are the positive controls.
func TestSerializeMultirange(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_multirange")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_multirange")
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
		`create type textrange as range (subtype = text)`, // creates textmultirange with it
		`create type span as (n int, days datemultirange)`,
		`create type tagged_mr as (n int, label textmultirange)`,
		`create table multiranges (
			id int primary key,
			n int, r4 int4range,
			m4 int4multirange, m8 int8multirange, mn nummultirange,
			md datemultirange, mts tsmultirange, mtz tstzmultirange, mt textmultirange,
			m4_arr int4multirange[], mts_arr tsmultirange[],
			sp span, sp_arr span[], tg tagged_mr
		)`,
		`insert into multiranges values (
			1, 42, '[1,10)',
			'{[1,3),[5,7)}', '{[1,10]}', '{[1.5,2.5],[3,)}',
			'{[2024-01-01,2024-02-01),[2024-03-01,)}', '{[2024-01-01 10:00:00,2024-06-01 12:00:00)}', '{[2024-01-01 10:00:00+00,)}', '{["a,b","c\"d"],[x,y)}',
			'{"{[1,3)}","{}",NULL}', '{"{[\"2024-01-01 10:00:00\",\"2024-06-01 12:00:00\")}"}',
			'(1,"{[2024-01-01,2024-02-01)}")', '{"(2,\"{[2024-02-01,)}\")","(3,{})"}', '(7,"{[""a,b"",""c""""d""],[x,y)}")'
		)`,
		`insert into multiranges (id) values (2)`,
		// the empty multirange everywhere it can appear
		`insert into multiranges (id, m4, md, mt, m4_arr, sp, tg) values (3, '{}', '{}', '{}', '{}', '(,)', '(,{})')`,
	}
	for _, q := range ddl {
		if _, err := gi.Conn.Exec(ctx, q); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	if err := db.ReloadSchemaCache(ctx); err != nil {
		t.Fatal(err)
	}

	// The schema cache must tell a multirange from a range: a range has a
	// subtype, a multirange is not a range at all, and nothing in the cache
	// may be a range without a subtype, which is what the serializers used
	// to dereference.
	t.Run("classification", func(t *testing.T) {
		info := db.info.Load()
		byName := map[string]Type{}
		for _, ct := range info.cachedTypes {
			byName[ct.Name] = ct
		}
		for _, name := range []string{"int4range", "textrange"} {
			ct, ok := byName[name]
			if !ok || !ct.IsRange || ct.RangeSubType == nil || ct.IsMultirange {
				t.Errorf("%s: expected a range with a subtype, got %+v", name, ct)
			}
		}
		for _, name := range []string{"int4multirange", "textmultirange"} {
			ct, ok := byName[name]
			if !ok {
				t.Errorf("%s: not in the schema cache", name)
			} else if ct.IsRange {
				t.Errorf("%s: classified as a range, subtype %v", name, ct.RangeSubType)
			} else if !ct.IsMultirange {
				t.Errorf("%s: not classified as a multirange", name)
			}
		}
		for _, ct := range info.cachedTypes {
			if ct.IsRange && ct.RangeSubType == nil {
				t.Errorf("%s.%s: a range with no subtype", ct.Schema, ct.Name)
			}
		}
	})

	projections := []struct{ name, columns string }{
		{"controls", "id, n, r4"},
		{"bare bounds", "m4, m8, mn, md"},
		{"quoted bounds", "mts, mtz, mt"},
		{"arrays", "m4_arr, mts_arr"},
		{"composites", "sp, sp_arr, tg"},
		{"whole row", "*"},
	}
	expected := map[string]string{}
	for _, pr := range projections {
		for _, id := range []int{1, 2, 3} {
			var row string
			q := fmt.Sprintf("select row_to_json(t)::text from (select %s from multiranges where id = %d) t", pr.columns, id)
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
					rows, err := gi.Conn.Query(ctx, fmt.Sprintf("select %s from multiranges where id = %d", pr.columns, id))
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
		// The CSV value is PostgreSQL's text for the multirange (and for the
		// composite holding one), quoted for its commas and quotes as
		// PostgREST's CSV (the record text) quotes it.
		t.Run("csv", func(t *testing.T) {
			rows, err := gi.Conn.Query(ctx, "select m4, mts, mt, sp from multiranges where id = 1")
			if err != nil {
				t.Fatal(err)
			}
			out, _, err := (&CSVSerializer{}).Serialize(rows, false, false, info)
			rows.Close()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := "m4,mts,mt,sp\n" +
				`"{[1,3),[5,7)}","{[""2024-01-01 10:00:00"",""2024-06-01 12:00:00"")}","{[""a,b"",""c""""d""],[x,y)}","(1,""{[2024-01-01,2024-02-01)}"")"`
			if string(out) != want {
				t.Errorf("expected %q\n     got %q", want, out)
			}
		})
	}

	t.Run("types unknown to the connection", func(t *testing.T) { check(t, ctx) })
	ReleaseConn(ctx, conn)

	db.pool.Reset()
	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)
	t.Run("composites registered", func(t *testing.T) { check(t, ctx) })
}

// The text path converts by the shape of the type, and three shapes reached
// it only through the review: a domain (its base type: 42, not "42"), a
// timestamp inside a composite that goes text because of another field (T
// between date and time, +hh:00 offsets, as to_json prints), and xml, which
// pgx prefers in text alone but in binary inside an array or a composite. A
// bytea download decodes the hex text it now arrives in.
func TestSerializeTextShapes(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_shapes")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_shapes")
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
		`create domain posint as int check (value > 0)`,
		`create domain yesno as bool`,
		`create type withdom as (a posint, b text, c yesno)`,
		`create type withts as (at timestamp, tz timestamptz, span int4range)`,
		`create type withxml as (x xml, n int)`,
		`create table shapes (id int, pi posint, by bytea, xa xml[], wd withdom, wt withts, wx withxml, x xml)`,
		`insert into shapes values (1, 7, '\x4142', '{<a/>,<b/>}', '(42,hi,t)',
			'("2024-01-01 10:00:00.25","2024-01-01 10:00:00+00","[1,3)")', '(<c/>,7)', '<d/>')`,
		`insert into shapes (id, wd, wt) values (2, '(,,)', '(infinity,,empty)')`,
		// a BC timestamp, a half-hour zone, a five-digit year
		`insert into shapes (id, wt) values (3, '("0044-03-15 12:00:00 BC","2024-01-01 15:30:00+05:30","[1,2)")')`,
		`insert into shapes (id, wt) values (4, '("10000-01-01 10:00:00","2024-01-01 10:00:00-03","[1,2)")')`,
		`insert into shapes (id, by) values (5, NULL)`,
	}
	for _, q := range ddl {
		if _, err := gi.Conn.Exec(ctx, q); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	if err := db.ReloadSchemaCache(ctx); err != nil {
		t.Fatal(err)
	}

	columns := []string{"pi", "xa", "x", "wd", "wt", "wx"}
	expected := map[string]string{}
	for _, c := range columns {
		for _, id := range []int{1, 2, 3, 4} {
			var row string
			q := fmt.Sprintf("select row_to_json(t)::text from (select %s from shapes where id = %d) t", c, id)
			if err := gi.Conn.QueryRow(ctx, q).Scan(&row); err != nil {
				t.Fatal(err)
			}
			expected[fmt.Sprintf("%s/row%d", c, id)] = "[" + row + "]"
		}
	}

	check := func(t *testing.T, ctx context.Context) {
		gi := GetSmoothContext(ctx)
		info := gi.Db.info.Load()
		for _, c := range columns {
			for _, id := range []int{1, 2, 3, 4} {
				name := fmt.Sprintf("%s/row%d", c, id)
				t.Run(name, func(t *testing.T) {
					rows, err := gi.Conn.Query(ctx, fmt.Sprintf("select %s from shapes where id = %d", c, id))
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
		t.Run("bytea download", func(t *testing.T) {
			rows, err := gi.Conn.Query(ctx, "select by from shapes where id = 1")
			if err != nil {
				t.Fatal(err)
			}
			out, _, err := (&BinarySerializer{}).Serialize(rows, true, true, info)
			rows.Close()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(out) != "AB" {
				t.Errorf("expected the bytes AB, got %q", out)
			}
			// a NULL adds nothing, the escape output format decodes too
			rows, err = gi.Conn.Query(ctx, "select by from shapes where id in (1, 5) order by id")
			if err != nil {
				t.Fatal(err)
			}
			out, _, err = (&BinarySerializer{}).Serialize(rows, true, false, info)
			rows.Close()
			if err != nil || string(out) != "AB" {
				t.Errorf("with a NULL row: expected the bytes AB, got %q (%v)", out, err)
			}
			if _, err := gi.Conn.Exec(ctx, "set bytea_output = escape"); err != nil {
				t.Fatal(err)
			}
			rows, err = gi.Conn.Query(ctx, `select '\x41005c42'::bytea`)
			if err != nil {
				t.Fatal(err)
			}
			out, _, err = (&BinarySerializer{}).Serialize(rows, true, true, info)
			rows.Close()
			gi.Conn.Exec(ctx, "reset bytea_output")
			if err != nil || string(out) != "A\x00\\B" {
				t.Errorf("escape format: expected A, NUL, backslash, B, got %q (%v)", out, err)
			}
		})
	}

	t.Run("types unknown to the connection", func(t *testing.T) { check(t, ctx) })
	// the conversion reads ISO text: the pool pins the output style at
	// connection startup whatever the server's default is
	t.Run("datestyle is ISO on the connection", func(t *testing.T) {
		var style string
		if err := gi.Conn.QueryRow(ctx, "show DateStyle").Scan(&style); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(style, "ISO") {
			t.Errorf("expected an ISO DateStyle, got %q", style)
		}
	})
	ReleaseConn(ctx, conn)

	db.pool.Reset()
	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)
	t.Run("composites registered", func(t *testing.T) { check(t, ctx) })
}

func TestIsoDateStyle(t *testing.T) {
	tests := []struct {
		params map[string]string
		want   string
	}{
		{map[string]string{}, "ISO"},
		{map[string]string{"DateStyle": "ISO"}, "ISO"},
		{map[string]string{"datestyle": "SQL, DMY"}, "ISO, DMY"},
		{map[string]string{"DateStyle": "German"}, "ISO, DMY"},
		{map[string]string{"DateStyle": "Postgres, Euro"}, "ISO, DMY"},
		{map[string]string{"DateStyle": "SQL, US"}, "ISO, MDY"},
		{map[string]string{"DATESTYLE": "ymd,postgres"}, "ISO, YMD"},
	}
	for _, test := range tests {
		got := isoDateStyle(test.params)
		if got != test.want {
			t.Errorf("%v: expected %q, got %q", test.params, test.want, got)
		}
		for key := range test.params {
			if strings.EqualFold(key, "datestyle") {
				t.Errorf("%v: the old key %q is still there", test.params, key)
			}
		}
	}
}
