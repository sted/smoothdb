package database

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/samber/lo"
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

	ctx := WithDb(context.Background(), db)
	defer ReleaseContext(ctx)

	db.CreateTable(ctx, &Table{Name: "b1", Columns: []Column{
		{Name: "name", Type: "text"},
		{Name: "number", Type: "integer"},
		{Name: "date", Type: "timestamp"}}})

	//b.Run("Insert", func(b *testing.B) {
	for i := 0; i < 10000; i++ {
		_, _, err := db.CreateRecords(ctx, "b1", []Record{
			{"name": "Morpheus😆", "number": 42, "date": "2022-10-11T19:00"},
			{"name": "Sted", "number": 55, "date": "1940-10-22T17:00"}})
		if err != nil {
			b.Fatal(err)
		}
	}

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

	db, _ := dbe.CreateDatabase(f_ctx, "test_base")
	defer dbe.DeleteDatabase(f_ctx, "test_base")

	ctx := WithDb(context.Background(), db)
	defer ReleaseContext(ctx)

	_, err := db.CreateTable(ctx, &Table{Name: "b1", Columns: []Column{
		{Name: "name", Type: "text"},
		{Name: "number", Type: "integer"},
		{Name: "date", Type: "timestamp"},
		{Name: "bool", Type: "boolean"}}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		db.CreateRecords(ctx, "b1", []Record{
			{"name": "Morpheus😆", "number": 42, "date": "2022-10-11T19:00", "bool": true},
			{"name": "Sted😆", "number": 43, "date": "2022-10-11T06:00", "bool": false}})
	}

	t.Run("Select1", func(t *testing.T) {
		_, err := db.GetRecords(ctx, "b1", nil)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Select2", func(t *testing.T) {
		SetQueryBuilder(ctx, QueryWithJSON{})
		_, err := db.GetRecords(ctx, "b1", nil)
		if err != nil {
			t.Error(err)
		}
	})

	// t.Run("Select3", func(t *testing.T) {
	// 	j, err := db.GetRecords3(ctx, "b1")
	// 	if err != nil {
	// 		log.Print(err)
	// 	}
	// 	fmt.Println("3", string(j))
	// })
}

func TestDDL(t *testing.T) {

	f_ctx := context.Background()

	db, _ := dbe.CreateDatabase(f_ctx, "test_ddl")
	defer dbe.DeleteDatabase(f_ctx, "test_ddl")

	ctx := WithDb(context.Background(), db)
	defer ReleaseContext(ctx)

	table := Table{Name: "b2", Columns: []Column{
		{Name: "id", Type: "serial", Primary: true},
		{Name: "name", Type: "text", Default: lo.ToPtr("pippo")},
		{Name: "number", Type: "integer", Unique: true},
		{Name: "date", Type: "timestamp", Check: "date > now()"},
		{Name: "bool", Type: "boolean", NotNull: true}},
		Check: []string{"number < 100000 and bool"},
	}

	t.Run("Create and drop table", func(t *testing.T) {
		_, err := db.CreateTable(ctx, &table)
		if err != nil {
			t.Fatal(err)
		}

		table_, err := db.GetTable(ctx, "b2")
		if err != nil {
			t.Fatal(err)
		}
		if table_.Name != "public.b2" ||
			table_.Primary != "PRIMARY KEY (id)" ||
			table_.Check[0] != "CHECK (number < 100000 AND bool)" ||
			len(table_.Unique) != 0 ||
			len(table_.Foreign) != 0 {
			t.Fatal("the returned table is not correct")
		}
		columns, err := db.GetColumns(ctx, "b2")
		if err != nil {
			t.Fatal(err)
		}
		if columns[0].Name != "id" ||
			columns[0].Type != "integer" {
			t.Fatal(err)
		}
		if columns[1].Name != "name" ||
			columns[1].Type != "text" ||
			*columns[1].Default != "'pippo'::text" {
			t.Fatal(err)
		}
		if columns[2].Name != "number" ||
			columns[2].Type != "integer" ||
			!columns[2].Unique {
			t.Fatal(err)
		}
		if columns[3].Name != "date" ||
			columns[3].Type != "timestamp without time zone" ||
			columns[3].Check != "CHECK (date > now())" {
			t.Fatal(err)
		}
		if columns[4].Name != "bool" ||
			columns[4].Type != "boolean" ||
			!columns[4].NotNull {
			t.Fatal(err)
		}
		err = db.DeleteTable(ctx, "b2")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Add and drop columns", func(t *testing.T) {
		_, err := db.CreateTable(ctx, &Table{Name: "b3", Temporary: false})
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.CreateColumn(ctx, &Column{Name: "c1", Type: "numeric", Table: "b3"})
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.CreateColumn(ctx, &Column{Name: "c2", Type: "text", Table: "b3"})
		if err != nil {
			t.Fatal(err)
		}

		columns, err := db.GetColumns(ctx, "b3")
		if err != nil {
			t.Fatal(err)
		}
		if len(columns) != 2 || columns[0].Name != "c1" || columns[1].Name != "c2" {
			t.Fatal("columns are not correct")
		}

		err = db.DeleteColumn(ctx, "b3", "c2", false)
		if err != nil {
			t.Fatal(err)
		}

		columns, err = db.GetColumns(ctx, "b3")
		if err != nil {
			t.Fatal(err)
		}
		if len(columns) != 1 || columns[0].Name != "c1" {
			t.Fatal("columns are not correct")
		}

		err = db.DeleteTable(ctx, "b3")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Update columns", func(t *testing.T) {
		_, err := db.CreateTable(ctx, &Table{Name: "b4"})
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.CreateColumn(ctx, &Column{Name: "c1", Type: "numeric", Table: "b4"})
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.UpdateColumn(ctx, &ColumnUpdate{
			Name:    "c1",
			NewName: lo.ToPtr("ccc"),
			Type:    lo.ToPtr("text"),
			NotNull: lo.ToPtr(true),
			Default: lo.ToPtr("'pippo'"),
			Check:   lo.ToPtr("c1 <> 'pluto'"),
			Unique:  lo.ToPtr(true),
			Table:   "b4"})
		if err != nil {
			t.Fatal(err)
		}

		column, err := db.GetColumn(ctx, "b4", "ccc")
		if err != nil {
			t.Fatal(err)
		}
		if column.Name != "ccc" || column.Type != "text" || column.NotNull != true ||
			*column.Default != "'pippo'::text" || column.Check != "CHECK (ccc <> 'pluto'::text)" {
			t.Fatal("column is not correct after the update")
		}
		err = db.DeleteTable(ctx, "b4")
		if err != nil {
			t.Fatal(err)
		}
	})
}
