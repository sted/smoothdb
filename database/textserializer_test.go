package database

import (
	"context"
	"encoding/binary"
	"log"
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
			FieldDescriptions_: []pgconn.FieldDescription{{Name: "x", DataTypeOID: c.oid}},
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
		FieldDescriptions_: []pgconn.FieldDescription{{Name: "x", DataTypeOID: arrOID}},
		RawValues_:         [][][]byte{{buf}},
		CurrentRow:         -1,
	}
	if _, _, err := (&JSONSerializer{}).Serialize(cr, false, false, info); err == nil {
		t.Error("expected Serialize to surface the decoder error, got nil")
	}
}
