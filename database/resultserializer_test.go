package database

import (
	"context"
	"log"
	"testing"
)

func BenchmarkSerializer(b *testing.B) {

	dbe_ctx, dbe_conn, _ := ContextWithDb(context.Background(), nil, "admin")
	defer ReleaseConn(dbe_ctx, dbe_conn)

	dbe.DeleteDatabase(dbe_ctx, "bench")
	db, err := dbe.CreateActiveDatabase(dbe_ctx, "bench", true)
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
			_, err := serializer.Serialize(copiedRows, false, info)
			if err != nil {
				log.Print(err)
				return
			}
		}
	})

	b.Run("Serialize_2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			copiedRows.CurrentRow = -1
			_, err := serializer.Serialize(copiedRows, false, info)
			if err != nil {
				log.Print(err)
				return
			}
		}
	})

	b.Run("Structs", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			copiedRows.CurrentRow = -1
			_, err := rowsToStructs(copiedRows)
			if err != nil {
				log.Print(err)
				return
			}
			//fmt.Printf("%v", s)
		}
	})

	b.Run("Structs_2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			copiedRows.CurrentRow = -1
			_, err := rowsToStructsWithPointers(copiedRows)
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
			_, err := rowsToMaps(copiedRows)
			if err != nil {
				log.Print(err)
				return
			}
			//fmt.Printf("%v", m)
		}
	})
}
