package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
)

var dbe *DBEngine

func TestMain(m *testing.M) {

	dbe, _ = InitDBEngine("postgres://localhost:5432")

	code := m.Run()

	os.Exit(code)
}

func BenchmarkBase(b *testing.B) {

	f_ctx := context.Background()

	db, _ := dbe.CreateDatabase(f_ctx, "bench")
	defer dbe.DeleteDatabase(f_ctx, "bench")

	ctx := NewContextForDb(db)
	defer ReleaseContext(ctx)

	db.CreateTable(ctx, &Table{Name: "b1", Columns: []Column{
		{Name: "name", Type: "text"},
		{Name: "number", Type: "integer"},
		{Name: "date", Type: "timestamp"},
	}})

	b.Run("Select1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := db.GetRecords(ctx, "b1", nil)
			if err != nil {
				log.Print(err)
				return
			}
		}
	})

	b.Run("Select2", func(b *testing.B) {
		SetQueryBuilder(ctx, QueryWithJSON{})
		for i := 0; i < b.N; i++ {
			_, err := db.GetRecords(ctx, "b1", nil)
			if err != nil {
				log.Print(err)
				return
			}
		}
	})

	// b.Run("Select3", func(b *testing.B) {
	// 	for i := 0; i < b.N; i++ {
	// 		_, err := db.GetRecords3(ctx, "b1")
	// 		if err != nil {
	// 			log.Print(err)
	// 		}
	// 	}
	// })
}

func TestBase(t *testing.T) {

	f_ctx := context.Background()

	db, _ := dbe.CreateDatabase(f_ctx, "bench")
	defer dbe.DeleteDatabase(f_ctx, "bench")

	ctx := NewContextForDb(db)
	defer ReleaseContext(ctx)

	db.CreateTable(ctx, &Table{Name: "b1"})
	db.CreateColumn(ctx, &Column{Name: "name", Type: "text", Table: "b1"})
	db.CreateColumn(ctx, &Column{Name: "number", Type: "integer", Table: "b1"})
	db.CreateColumn(ctx, &Column{Name: "date", Type: "timestamp", Table: "b1"})
	db.CreateColumn(ctx, &Column{Name: "bool", Type: "boolean", Table: "b1"})

	for i := 0; i < 5; i++ {
		db.CreateRecords(ctx, "b1", []Record{
			{"name": "Morpheus😆", "number": 42, "date": "2022-10-11T19:00", "bool": true},
			{"name": "Sted😆", "number": 43, "date": "2022-10-11T06:00", "bool": false}})
	}

	t.Run("Select1", func(t *testing.T) {
		j, err := db.GetRecords(ctx, "b1", nil)
		if err != nil {
			log.Print(err)
			return
		}
		fmt.Println("1", string(j))
	})

	t.Run("Select2", func(t *testing.T) {
		SetQueryBuilder(ctx, QueryWithJSON{})
		j, err := db.GetRecords(ctx, "b1", nil)
		if err != nil {
			log.Print(err)
			return
		}
		fmt.Println("2", string(j))
	})

	// t.Run("Select3", func(t *testing.T) {
	// 	j, err := db.GetRecords3(ctx, "b1")
	// 	if err != nil {
	// 		log.Print(err)
	// 	}
	// 	fmt.Println("3", string(j))
	// })
}
