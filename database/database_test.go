package database

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"
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

	ctx := NewContextFromDb(db)
	defer GetConn(ctx).Release()

	db.CreateSource(ctx, "b1")
	db.CreateField(ctx, &Field{Name: "name", Type: "text", Source: "b1"})
	db.CreateField(ctx, &Field{Name: "number", Type: "integer", Source: "b1"})
	db.CreateField(ctx, &Field{Name: "date", Type: "timestamp", Source: "b1"})

	b.Run("Insert", func(b *testing.B) {
		for i := 0; i < 10000; i++ {
			db.CreateRecord(ctx, "b1", &Record{"name": "Morpheus", "number": 42, "date": "20221011T19:00"})
			db.CreateRecord(ctx, "b1", &Record{"name": "Sted", "number": 55, "date": "194010122T17:00"})
		}
	})

	b.Run("Select1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := db.GetRecords(ctx, "b1")
			if err != nil {
				log.Fatal(err)
			}
		}
	})

	b.Run("Select2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := db.GetRecords2(ctx, "b1")
			if err != nil {
				log.Fatal(err)
			}
		}
	})

	b.Run("Select3", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := db.GetRecords3(ctx, "b1")
			if err != nil {
				log.Fatal(err)
			}
		}
	})

	b.Run("Select4", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := db.GetRecords4(ctx, "b1")
			if err != nil {
				log.Fatal(err)
			}
		}
	})

	b.Run("Select5", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := db.GetRecords5(ctx, "b1")
			if err != nil {
				log.Fatal(err)
			}
		}
	})

	b.Run("Select6", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := GetRecords6(ctx, "b1")
			if err != nil {
				log.Fatal(err)
			}
		}
	})
}

func GetRecords6(ctx context.Context, source string) ([]byte, error) {

	type Record struct {
		id     int32
		name   string
		number int32
		date   *time.Time
	}

	conn := GetConn(ctx)
	records := []Record{}
	rows, err := conn.Query(ctx, "SELECT * FROM "+source)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	record := Record{}

	for rows.Next() {
		err := rows.Scan(&record.id, &record.name, &record.number, &record.date)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	if rows.Err() != nil {
		return nil, err
	}

	var w bytes.Buffer
	encoder := json.NewEncoder(&w)
	encoder.Encode(records)
	return w.Bytes(), nil
}

func TestBase(t *testing.T) {

	f_ctx := context.Background()

	db, _ := dbe.CreateDatabase(f_ctx, "bench")
	defer dbe.DeleteDatabase(f_ctx, "bench")

	ctx := NewContextFromDb(db)

	db.CreateSource(ctx, "b1")
	db.CreateField(ctx, &Field{Name: "name", Type: "text", Source: "b1"})
	db.CreateField(ctx, &Field{Name: "number", Type: "integer", Source: "b1"})
	db.CreateField(ctx, &Field{Name: "date", Type: "timestamp", Source: "b1"})

	for i := 0; i < 5; i++ {
		db.CreateRecord(ctx, "b1", &Record{"name": "Morpheus", "number": 42, "date": "20221011T19:00"})
	}

	t.Run("Select1", func(t *testing.T) {
		j, err := db.GetRecords(ctx, "b1")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(j))
	})

	t.Run("Select2", func(t *testing.T) {
		j, err := db.GetRecords2(ctx, "b1")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(j))
	})

	t.Run("Select3", func(t *testing.T) {
		j, err := db.GetRecords3(ctx, "b1")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(j))
	})

	t.Run("Select4", func(t *testing.T) {
		j, err := db.GetRecords4(ctx, "b1")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(j))
	})

	t.Run("Select5", func(t *testing.T) {
		j, err := db.GetRecords5(ctx, "b1")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(j))
	})
}
