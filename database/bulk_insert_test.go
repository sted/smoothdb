package database

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"math"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

// CreateRecords is the Go entry point of an insert, with no request body: the
// records themselves are encoded as the one json parameter. Over 65 535
// values (33 000 rows × 2 columns here) one parameter per value made pgx
// refuse the statement with "extended protocol limited to 65535 parameters".
// A []byte is bytea's hex form in that encoding, not json's base64.
func TestCreateRecordsOverParameterLimit(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_bulk")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_bulk")
	if err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)

	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)

	_, err = CreateTable(ctx, &Table{
		Name: "bulk",
		Columns: []Column{
			{Name: "a", Type: "int4"},
			{Name: "b", Type: "int4"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = CreateTable(ctx, &Table{
		Name:    "blobs",
		Columns: []Column{{Name: "id", Type: "int4"}, {Name: "data", Type: "bytea"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ReloadSchemaCache(ctx); err != nil {
		t.Fatal(err)
	}

	const rows = 33000
	records := make([]Record, rows)
	for i := range records {
		records[i] = Record{"a": i, "b": i * 2}
	}
	_, count, err := CreateRecords(ctx, "bulk", records, nil)
	if err != nil {
		t.Fatalf("inserting %d values: %v", rows*2, err)
	}
	if count != rows {
		t.Errorf("rows affected: want %d, got %d", rows, count)
	}
	var n, sumB int64
	err = conn.QueryRow(ctx, `SELECT count(*), sum(b) FROM bulk`).Scan(&n, &sumB)
	if err != nil {
		t.Fatal(err)
	}
	if n != rows || sumB != int64(rows*(rows-1)) {
		t.Errorf("table content: want %d rows summing b to %d, got %d rows summing to %d", rows, rows*(rows-1), n, sumB)
	}

	_, _, err = CreateRecords(ctx, "blobs", []Record{{"id": 1, "data": []byte{0x01, 0xff}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var hex string
	if err := conn.QueryRow(ctx, `SELECT encode(data, 'hex') FROM blobs WHERE id = 1`).Scan(&hex); err != nil {
		t.Fatal(err)
	}
	if hex != "01ff" {
		t.Errorf("bytea from []byte: want 01ff, got %s", hex)
	}
}

// GetColumnTypes empties the search path for its query only: called inside a
// transaction of the caller's (a TransactionMode other than none begins one on
// every request connection) it must neither commit that transaction nor leave
// the search path changed in it.
func TestGetColumnTypesInsideATransaction(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_bulk_tx")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_bulk_tx")
	if err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)

	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)

	if _, err := conn.Exec(ctx, `CREATE TABLE tx_probe (id int)`); err != nil {
		t.Fatal(err)
	}
	var before, after string
	if _, err := conn.Exec(ctx, `BEGIN; INSERT INTO tx_probe VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(ctx, `SHOW search_path`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	types, err := GetColumnTypes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(ctx, `SHOW search_path`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx, `ROLLBACK`); err != nil {
		t.Fatal(err)
	}
	if len(types) == 0 {
		t.Error("no column types read")
	}
	if after != before {
		t.Errorf("search path in the transaction: %q before, %q after", before, after)
	}
	var n int
	if err := conn.QueryRow(ctx, `SELECT count(*) FROM tx_probe`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("the caller's transaction was committed: %d rows after its rollback", n)
	}
}

// The column definition list names each type as the schema cache holds it,
// so the cache must hold it schema-qualified, as PostgREST loads it with an
// empty search path (SchemaCache.hs): a type cached by its bare name does not
// resolve on a connection whose search path does not show its schema, which
// SchemaSearchPath, and the first load (a connection without it), make
// possible. And the Go values a caller gives keep the meaning pgx gave them.
func TestCreateRecordsTypeNamesAndGoValues(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_bulk_types")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_bulk_types")
	if err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)

	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)

	_, err = conn.Exec(ctx, `
		CREATE TYPE bulk_mood AS ENUM ('sad', 'ok');
		CREATE TABLE bulk_typed (id int, mood bulk_mood, data bytea, iv interval, f float8)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ReloadSchemaCache(ctx); err != nil {
		t.Fatal(err)
	}
	if ct := db.info.Load().GetColumnType("public.bulk_typed", "mood"); ct == nil || ct.DataType != "public.bulk_mood" {
		t.Fatalf("cached type of mood: want public.bulk_mood, got %+v", ct)
	}

	// the request's connection does not see public
	if _, err := conn.Exec(ctx, `SET search_path TO pg_catalog`); err != nil {
		t.Fatal(err)
	}
	_, _, err = CreateRecords(ctx, "bulk_typed", []Record{
		{"id": 1, "mood": "ok", "data": []byte(nil), "iv": pgtype.Interval{Days: 1, Valid: true}, "f": math.NaN()},
		{"id": 2, "mood": nil, "data": []byte{}, "iv": pgtype.Interval{}, "f": math.Inf(1)},
	}, nil)
	if _, rerr := conn.Exec(ctx, `RESET search_path`); rerr != nil {
		t.Fatal(rerr)
	}
	if err != nil {
		t.Fatalf("insert with an enum outside the search path: %v", err)
	}
	var got string
	err = conn.QueryRow(ctx, `SELECT string_agg(format('%s|%s|%s|%s|%s', id, mood, coalesce(encode(data, 'hex'), 'NULL'),
		coalesce(iv::text, 'NULL'), f), ';' ORDER BY id) FROM bulk_typed`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if want := "1|ok|NULL|1 day|NaN;2|||NULL|Infinity"; got != want {
		t.Errorf("rows: want %s, got %s", want, got)
	}
}

// CreateRecords takes Go values with the meaning pgx gives them: for a json
// or jsonb column a string, a []byte or a json.RawMessage is the JSON itself,
// as pgx binds it, and must be valid JSON; other Go values are encoded as
// JSON (a slice an array, a map an object). The JSON null, from "null" too,
// is SQL NULL, as json_to_recordset reads it and as in PostgREST, where pgx
// stored a jsonb null. A request body keeps PostgREST's meaning instead, where
// a JSON string is a string (see test/api).
func TestCreateRecordsJSONColumns(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_bulk_json")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_bulk_json")
	if err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)

	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)

	if _, err := conn.Exec(ctx, `CREATE TABLE bulk_json (id int, j json, jb jsonb)`); err != nil {
		t.Fatal(err)
	}
	if err := db.ReloadSchemaCache(ctx); err != nil {
		t.Fatal(err)
	}
	_, _, err = CreateRecords(ctx, "bulk_json", []Record{
		{"id": 1, "j": "[123]", "jb": []byte(`{"a":1}`)},
		{"id": 2, "j": json.RawMessage(`{"z": 1, "a": 2}`), "jb": "null"},
		{"id": 3, "j": []any{1, "x"}, "jb": map[string]any{"k": 2}},
		{"id": 4, "j": nil, "jb": []byte(nil)},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got string
	err = conn.QueryRow(ctx, `SELECT string_agg(format('%s|%s|%s', id, coalesce(j::text, 'NULL'), coalesce(jb::text, 'NULL')), ';' ORDER BY id) FROM bulk_json`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if want := `1|[123]|{"a": 1};2|{"z": 1, "a": 2}|NULL;3|[1,"x"]|{"k": 2};4|NULL|NULL`; got != want {
		t.Errorf("rows\n  want: %s\n  got:  %s", want, got)
	}

	// a string that is not JSON is refused, as pgx's binding was by PostgreSQL
	_, _, err = CreateRecords(ctx, "bulk_json", []Record{{"id": 5, "j": "not json"}}, nil)
	if err == nil {
		t.Error("a string that is not JSON was accepted for a json column")
	}
}

type namedJSON string

// ptrValuer has a pointer-receiver Value, which pgx (as database/sql) calls
// on a nil pointer too.
type ptrValuer struct{}

func (p *ptrValuer) Value() (driver.Value, error) {
	if p == nil {
		return "[1]", nil
	}
	return "[2]", nil
}

// marshaledInt marshals itself through a pointer receiver.
type marshaledInt struct{}

func (*marshaledInt) MarshalJSON() ([]byte, error) { return []byte("42"), nil }

// bytesValuer gives a []byte, which a text column takes as its text.
type bytesValuer struct{}

func (bytesValuer) Value() (driver.Value, error) { return []byte("def"), nil }

// The rest of pgx's JSON codec (pgtype/json.go): a driver.Valuer gives its
// value before a json.Marshaler is asked, a pointer is followed, a named
// string type is a string; a json[] element is JSON like a json column; a
// []byte for a text column is its text. Only the built-in json types are JSON:
// an enum named json is not. The table is created after the schema cache was
// loaded, under a mixed-case name, so its types are read for the insert.
func TestCreateRecordsGoValuesAsPgx(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_bulk_pgx")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_bulk_pgx")
	if err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)

	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)

	_, err = conn.Exec(ctx, `
		CREATE TYPE public."json" AS ENUM ('abc');
		CREATE TABLE "MixedPgx" (id int, j json, ja json[], tx text, e public."json")`)
	if err != nil {
		t.Fatal(err)
	}
	s := `{"p": 1}`
	b := []byte("abc")
	rawText := json.RawMessage(`"q"`)
	bv := &bytesValuer{}
	bvp := &bv
	var anyBytes any = []byte("abc")
	_, _, err = CreateRecords(ctx, "MixedPgx", []Record{
		{"id": 1, "j": pgtype.Text{String: "[7]", Valid: true}, "ja": []string{"[123]", `{"a":1}`}, "tx": []byte("abc"), "e": "abc"},
		{"id": 2, "j": &s, "ja": [][]byte{[]byte("1"), nil}, "tx": nil, "e": nil},
		{"id": 3, "j": namedJSON("[8]"), "ja": nil, "tx": "x", "e": nil},
		// an []any element holding a slice is a JSON value, not a dimension;
		// a pointer to a []byte is text, as is a []byte from a Valuer
		{"id": 4, "j": (*ptrValuer)(nil), "ja": []any{[]int{1, 2}, []int{3}}, "tx": &b, "e": nil},
		// an invalid pgtype element is SQL NULL, not the JSON null
		{"id": 5, "j": nil, "ja": []pgtype.Text{{}, {String: "[5]", Valid: true}}, "tx": bytesValuer{}, "e": nil},
		// json[] is one-dimensional, as pgx makes it: a typed inner slice is a
		// JSON element too, ragged or not; a json.RawMessage for a text column
		// is its bytes, quotes included
		{"id": 6, "j": nil, "ja": [][]int{{1, 2}, {3, 4}}, "tx": json.RawMessage(`"abc"`), "e": nil},
		{"id": 7, "j": nil, "ja": [][]int{{1}, {2, 3}}, "tx": &rawText, "e": nil},
		// a pointer whose MarshalJSON has a pointer receiver keeps it
		{"id": &marshaledInt{}, "j": nil, "ja": nil, "tx": nil, "e": nil},
		// pointers are followed through Valuers and interfaces, as pgx
		// dereferences and plans again at each level
		{"id": 9, "j": nil, "ja": nil, "tx": &bvp, "e": nil},
		{"id": 10, "j": nil, "ja": nil, "tx": &anyBytes, "e": nil},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got string
	err = conn.QueryRow(ctx, `SELECT string_agg(format('%s|%s|%s|%s|%s', id, coalesce(j::text, 'NULL'), coalesce(ja::text, 'NULL'),
		coalesce(tx, 'NULL'), coalesce(e::text, 'NULL')), ';' ORDER BY id) FROM "MixedPgx"`).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	// array_out quotes an element only where it must: [123] is the json array
	if want := `1|[7]|{[123],"{\"a\":1}"}|abc|abc;2|{"p": 1}|{1,NULL}|NULL|NULL;3|[8]|NULL|x|NULL;4|[1]|{"[1,2]",[3]}|abc|NULL;5|NULL|{NULL,[5]}|def|NULL;6|NULL|{"[1,2]","[3,4]"}|"abc"|NULL;7|NULL|{[1],"[2,3]"}|"q"|NULL;9|NULL|NULL|def|NULL;10|NULL|NULL|abc|NULL;42|NULL|NULL|NULL|NULL`; got != want {
		t.Errorf("rows\n  want: %s\n  got:  %s", want, got)
	}
}

// selfValuer answers itself: a malformed Valuer must end in an error, not in
// a stack overflow.
type selfValuer struct{}

func (s selfValuer) Value() (driver.Value, error) { return s, nil }

func TestInsertRowsSelfValuer(t *testing.T) {
	_, err := insertRows([]Record{{"t": selfValuer{}}}, []string{"t"}, []goKind{goOther}, nil)
	if err == nil {
		t.Fatal("a Valuer that answers itself was encoded")
	}
}
