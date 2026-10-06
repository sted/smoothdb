package database

import (
	"context"
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
