package database

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/samber/lo"
)

func TestMain(m *testing.M) {
	var err error
	config := DefaultConfig()
	config.URL = "postgresql://auth:@0.0.0.0:5432/postgres"
	dbe, err = InitDbEngine(config, nil)
	if err != nil {
		os.Exit(1)
	}
	ctx, conn, err := ContextWithDb(context.Background(), nil, "admin")
	if err != nil {
		os.Exit(1)
	}
	defer ReleaseConn(ctx, conn)

	_, err = CreateUser(ctx, &User{Name: "test", CanCreateDatabases: true})
	if err != nil && IsExist(err) {
		os.Exit(1)
	}

	code := m.Run()

	os.Exit(code)
}

func TestBase(t *testing.T) {

	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}

	dbe.DeleteDatabase(ctx, "test_base")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_base")
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
		Name: "b1",
		Columns: []Column{
			{Name: "name", Type: "text"},
			{Name: "number", Type: "integer"},
			{Name: "date", Type: "timestamp"},
			{Name: "bool", Type: "bool"},
			{Name: "float4", Type: "float4"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		CreateRecords(ctx, "b1", []Record{
			{"name": "Morpheus😆", "number": 42, "date": "2022-10-11T19:00", "bool": true, "float4": 3.1},
			{"name": "Sted😆", "number": 43, "date": "2022-10-11T06:00", "bool": false}}, nil)
	}

	t.Run("Select1", func(t *testing.T) {
		_, err := GetRecords(ctx, "b1", nil)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Select2", func(t *testing.T) {
		SetQueryBuilder(ctx, QueryWithJSON{})
		_, err := GetRecords(ctx, "b1", nil)
		if err != nil {
			t.Error(err)
		}
	})

	// t.Run("Select3", func(t *testing.T) {
	// 	j, err := GetRecords3(ctx, "b1")
	// 	if err != nil {
	// 		log.Print(err)
	// 	}
	// 	fmt.Println("3", string(j))
	// })
}

func TestDDL(t *testing.T) {

	dbe_ctx, dbe_conn, _ := ContextWithDb(context.Background(), nil, "test")
	defer ReleaseConn(dbe_ctx, dbe_conn)

	dbe.DeleteDatabase(dbe_ctx, "test_ddl")
	db, err := dbe.GetOrCreateActiveDatabase(dbe_ctx, "test_ddl")
	if err != nil {
		t.Fatal(err)
	}

	ctx, conn, _ := ContextWithDb(dbe_ctx, db, "test")
	defer ReleaseConn(ctx, conn)

	table := Table{
		Name: "b2",
		Columns: []Column{
			{Name: "id", Type: "serial", Constraints: []string{"PRIMARY KEY"}},
			{Name: "name", Type: "text", Default: lo.ToPtr("'pippo'")},
			{Name: "number", Type: "integer", Constraints: []string{"UNIQUE"}},
			{Name: "date", Type: "timestamp", Constraints: []string{"CHECK (date > now())"}},
			{Name: "bool", Type: "boolean", NotNull: true},
		},
		Constraints: []string{"CHECK (number < 100000 AND bool)"},
	}

	t.Run("Create and drop table", func(t *testing.T) {
		_, err := CreateTable(ctx, &table)
		if err != nil {
			t.Fatal(err)
		}

		table_, err := GetTable(ctx, "b2")
		if err != nil {
			t.Fatal(err)
		}
		if table_.Name != "public.b2" ||
			len(table_.Constraints) != 2 ||
			table_.Constraints[0] != "CHECK (number < 100000 AND bool)" ||
			table_.Constraints[1] != "PRIMARY KEY (id)" {
			t.Fatal("the returned table is not correct")
		}
		columns, err := GetColumns(ctx, "b2")
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
			columns[2].Constraints[0] != "UNIQUE (number)" {
			t.Fatal(err)
		}
		if columns[3].Name != "date" ||
			columns[3].Type != "timestamp without time zone" ||
			columns[3].Constraints[0] != "CHECK (date > now())" {
			t.Fatal(err)
		}
		if columns[4].Name != "bool" ||
			columns[4].Type != "boolean" ||
			!columns[4].NotNull {
			t.Fatal(err)
		}
		err = DeleteTable(ctx, "b2", false)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Add and drop columns", func(t *testing.T) {
		_, err := CreateTable(ctx, &Table{Name: "b3"})
		if err != nil {
			t.Fatal(err)
		}

		_, err = CreateColumn(ctx, &Column{Name: "c1", Type: "numeric", Table: "b3"})
		if err != nil {
			t.Fatal(err)
		}

		_, err = CreateColumn(ctx, &Column{Name: "c2", Type: "text", Table: "b3"})
		if err != nil {
			t.Fatal(err)
		}

		columns, err := GetColumns(ctx, "b3")
		if err != nil {
			t.Fatal(err)
		}
		if len(columns) != 2 || columns[0].Name != "c1" || columns[1].Name != "c2" {
			t.Fatal("columns are not correct")
		}

		err = DeleteColumn(ctx, "b3", "c2", false)
		if err != nil {
			t.Fatal(err)
		}

		columns, err = GetColumns(ctx, "b3")
		if err != nil {
			t.Fatal(err)
		}
		if len(columns) != 1 || columns[0].Name != "c1" {
			t.Fatal("columns are not correct")
		}

		err = DeleteTable(ctx, "b3", false)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("Update columns", func(t *testing.T) {
		_, err := CreateTable(ctx, &Table{Name: "b4"})
		if err != nil {
			t.Fatal(err)
		}

		_, err = CreateColumn(ctx, &Column{Name: "c1", Type: "numeric", Table: "b4"})
		if err != nil {
			t.Fatal(err)
		}

		_, err = UpdateColumn(ctx, &ColumnUpdate{
			Name:    "c1",
			NewName: lo.ToPtr("ccc"),
			Type:    lo.ToPtr("text"),
			NotNull: lo.ToPtr(true),
			Default: lo.ToPtr("'pippo'"),
			//Constraints: []string{"CHECK c1 <> 'pluto'", "UNIQUE"},
			Table: "b4"})
		if err != nil {
			t.Fatal(err)
		}

		column, err := GetColumn(ctx, "b4", "ccc")
		if err != nil {
			t.Fatal(err)
		}
		if column.Name != "ccc" || column.Type != "text" || column.NotNull != true ||
			*column.Default != "'pippo'::text" {
			//|| column.Check != "CHECK (ccc <> 'pluto'::text)"
			t.Fatal("column is not correct after the update")
		}
		err = DeleteTable(ctx, "b4", false)
		if err != nil {
			t.Fatal(err)
		}
	})
}

func BenchmarkBase(b *testing.B) {

	dbe_ctx, dbe_conn, _ := ContextWithDb(context.Background(), nil, "test")
	defer ReleaseConn(dbe_ctx, dbe_conn)

	dbe.DeleteDatabase(dbe_ctx, "bench")
	db, err := dbe.GetOrCreateActiveDatabase(dbe_ctx, "bench")
	if err != nil {
		b.Fatal(err)
	}

	ctx, conn, _ := ContextWithDb(dbe_ctx, db, "test")
	defer ReleaseConn(ctx, conn)

	CreateTable(ctx, &Table{Name: "b1", Columns: []Column{
		{Name: "name", Type: "text"},
		{Name: "number", Type: "integer"},
		{Name: "date", Type: "timestamp"}}})

	for i := 0; i < 10000; i++ {
		_, _, err := CreateRecords(ctx, "b1", []Record{
			{"name": "Morpheus😆", "number": 42, "date": "2022-10-11T19:00"},
			{"name": "Sted", "number": 55, "date": "1940-10-22T17:00"}}, nil)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.Run("Select1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := GetRecords(ctx, "b1", nil)
			if err != nil {
				log.Print(err)
				return
			}
		}
	})

	b.Run("Select2", func(b *testing.B) {
		SetQueryBuilder(ctx, QueryWithJSON{})
		for i := 0; i < b.N; i++ {
			_, err := GetRecords(ctx, "b1", nil)
			if err != nil {
				log.Print(err)
				return
			}
		}
	})

	// b.Run("Select3", func(b *testing.B) {
	// 	for i := 0; i < b.N; i++ {
	// 		_, err := GetRecords3(ctx, "b1")
	// 		if err != nil {
	// 			log.Print(err)
	// 		}
	// 	}
	// })
}
