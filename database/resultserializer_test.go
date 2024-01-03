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
				{"name": "Sted", "number": 55, "date": nil},
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

	b.Run("Structs", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			copiedRows.CurrentRow = -1
			_, err := db.rowsToStructs(copiedRows)
			if err != nil {
				log.Print(err)
				return
			}
			//fmt.Printf("%v", s)
		}
	})

	b.Run("Maps", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			copiedRows.CurrentRow = -1
			_, err := db.rowsToMaps(copiedRows)
			if err != nil {
				log.Print(err)
				return
			}
			//fmt.Printf("%v", m)
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

	var rawValues [][][]byte

	// Iterate over the rows and read the values
	for rows.Next() {
		rowValues := make([][]byte, len(columns))

		// Copying the values from the current row
		for i, val := range rows.RawValues() {
			rowValues[i] = make([]byte, len(val))
			copy(rowValues[i], val)
		}

		rawValues = append(rawValues, rowValues)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Create a new CustomRows instance with the extracted values and column descriptions
	customRows := &CustomRows{
		FieldDescriptions_: columns,
		RawValues_:         rawValues,
		CurrentRow:         -1,
	}

	return customRows, nil
}
