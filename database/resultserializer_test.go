package database

import (
	"context"
	"log"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func BenchmarkSerializer(b *testing.B) {

	dbe_ctx, dbe_conn, _ := ContextWithDb(context.Background(), nil, "admin")
	defer ReleaseConn(dbe_ctx, dbe_conn)

	dbe.DeleteDatabase(dbe_ctx, "bench")
	db, err := dbe.CreateActiveDatabase(dbe_ctx, "bench")
	if err != nil {
		b.Fatal(err)
	}

	ctx, conn, _ := ContextWithDb(dbe_ctx, db, "admin")
	defer ReleaseConn(ctx, conn)

	db.CreateTable(ctx, &Table{Name: "b1", Columns: []Column{
		{Name: "name", Type: "text"},
		{Name: "number", Type: "integer"},
		{Name: "date", Type: "timestamp"},
	}})

	for i := 0; i < 10000; i++ {
		_, _, err := db.CreateRecords(ctx, "b1",
			[]Record{
				{"name": "Morpheus😆", "number": 42, "date": "2022-10-11T19:00"},
				{"name": "Sted", "number": 55, "date": "1940-10-22T17:00"},
			},
			nil)
		if err != nil {
			b.Fatal(err)
		}
	}
	gi := GetSmoothContext(ctx)
	info := gi.Db.info
	rows, err := gi.Conn.Query(ctx, "select * from b1")
	if err != nil {
		b.Fatal(err)
	}
	copiedRows, err := CopyRows(rows)
	if err != nil {
		b.Fatal(err)
	}
	defer rows.Close()
	serializer := gi.QueryBuilder.preferredSerializer()

	b.Run("Serialize", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			copiedRows.CurrentRow = -1
			_, err := serializer.Serialize(ctx, copiedRows, false, info)
			if err != nil {
				log.Print(err)
				return
			}
		}
	})

	b.Run("Serialize_2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			copiedRows.CurrentRow = -1
			_, err := serializer.Serialize(ctx, copiedRows, false, info)
			if err != nil {
				log.Print(err)
				return
			}
		}
	})
}

// CustomRows is used do be able to do more iterations on a single query result.
// This is not permitted by pgx.Rows.

type CustomRows struct {
	FieldDescriptions_ []pgconn.FieldDescription
	RawValues_         [][][]byte
	CurrentRow         int
}

func (cr *CustomRows) Next() bool {
	cr.CurrentRow++
	return cr.CurrentRow < len(cr.RawValues_)
}

func (cr *CustomRows) Err() error {
	return nil
}

func (cr *CustomRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}

func (cr *CustomRows) Conn() *pgx.Conn {
	return nil
}

func (cr *CustomRows) Scan(dest ...any) error {
	return nil
}

func (cr *CustomRows) FieldDescriptions() []pgconn.FieldDescription {
	return cr.FieldDescriptions_
}

func (cr *CustomRows) Values() ([]any, error) {
	return nil, nil
}

func (cr *CustomRows) RawValues() [][]byte {
	return cr.RawValues_[cr.CurrentRow]
}

func (cr *CustomRows) Close() {
	cr.CurrentRow = -1
}

func CopyRows(rows pgx.Rows) (*CustomRows, error) {
	// Get the column descriptions
	columns := rows.FieldDescriptions()

	// Create a new CustomRows instance with the extracted values and column descriptions
	customRows := &CustomRows{
		FieldDescriptions_: columns,
		RawValues_:         make([][][]byte, 20000),
		CurrentRow:         -1,
	}

	// Iterate over the rows and read the values
	i := 0
	for rows.Next() {
		rowValues := make([][]byte, len(columns))
		copy(rowValues, rows.RawValues())
		customRows.RawValues_[i] = rowValues
		i += 1
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return customRows, nil
}
