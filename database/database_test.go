package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"encoding/json"
	"strings"

	"github.com/samber/lo"
)

func TestMain(m *testing.M) {
	var err error
	config := DefaultConfig()
	config.URL = "postgresql://postgres:postgres@0.0.0.0:5432/postgres"
	dbe, err = InitDbEngine(config, nil)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	ctx, conn, err := ContextWithDb(context.Background(), nil, "postgres")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(2)
	}
	defer ReleaseConn(ctx, conn)

	_, err = CreateUser(ctx, &User{Name: "test", CanCreateDatabases: true})
	if err != nil && !IsExist(err) {
		fmt.Println(err.Error())
		os.Exit(3)
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
			{Name: "name", Type: "text", Default: lo.ToPtr("'sted'")},
			{Name: "number", Type: "integer", Default: lo.ToPtr("1")},
			{Name: "date", Type: "timestamp", Default: lo.ToPtr("now()")},
			{Name: "bool", Type: "bool", Default: lo.ToPtr("true")},
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
		_, _, err := GetRecords(ctx, "b1", nil)
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Select2", func(t *testing.T) {
		SetQueryBuilder(ctx, QueryWithJSON{})
		_, _, err := GetRecords(ctx, "b1", nil)
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
		if table_.Name != "b2" ||
			table_.Schema != "public" ||
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
			columns[0].Type != "int4" {
			t.Fatal(err)
		}
		if columns[1].Name != "name" ||
			columns[1].Type != "text" ||
			*columns[1].Default != "'pippo'::text" {
			t.Fatal(err)
		}
		if columns[2].Name != "number" ||
			columns[2].Type != "int4" ||
			columns[2].Constraints[0] != "UNIQUE (number)" {
			t.Fatal(err)
		}
		if columns[3].Name != "date" ||
			columns[3].Type != "timestamp" ||
			columns[3].Constraints[0] != "CHECK (date > now())" {
			t.Fatal(err)
		}
		if columns[4].Name != "bool" ||
			columns[4].Type != "bool" ||
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

		err = UpdateColumn(ctx, "b4", "c1", &ColumnUpdate{
			Name:    lo.ToPtr("ccc"),
			Type:    lo.ToPtr("text"),
			NotNull: lo.ToPtr(true),
			Default: lo.ToPtr("'pippo'"),
		})
		//Constraints: []string{"CHECK c1 <> 'pluto'", "UNIQUE"},
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

func TestUnknownTypeSerialize(t *testing.T) {
	// Test that unrecognized PostgreSQL types (e.g. ltree) are serialized
	// as text strings instead of producing broken JSON.

	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}

	dbe.DeleteDatabase(ctx, "test_unk_type")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_unk_type")
	if err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)

	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)

	gi := GetSmoothContext(ctx)
	_, err = gi.Conn.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS ltree")
	if err != nil {
		t.Skip("ltree extension not available:", err)
	}

	_, err = CreateTable(ctx, &Table{
		Name: "label",
		Columns: []Column{
			{Name: "id", Type: "serial"},
			{Name: "path", Type: "ltree"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = gi.Conn.Exec(ctx, "INSERT INTO label (path) VALUES ('Top.Science.Astronomy')")
	if err != nil {
		t.Fatal(err)
	}

	// Refresh schema info to pick up the ltree type
	err = db.ReloadSchemaCache(ctx)

	result, _, err := GetRecords(ctx, "label", nil)
	if err != nil {
		t.Fatal(err)
	}

	raw := string(result)

	// The JSON must be valid
	if !json.Valid(result) {
		t.Fatalf("invalid JSON output: %s", raw)
	}

	// The ltree value must appear as a quoted string
	if !strings.Contains(raw, `"Top.Science.Astronomy"`) {
		t.Fatalf("expected ltree value as quoted string, got: %s", raw)
	}
}

func TestCompositeTypeInFunctionOutput(t *testing.T) {
	// Test that composite types (table row types) in function outputs
	// are serialized as nested JSON objects.

	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}

	dbe.DeleteDatabase(ctx, "test_comp_func")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_comp_func")
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
		Name: "documents",
		Columns: []Column{
			{Name: "id", Type: "serial"},
			{Name: "name", Type: "text"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = CreateRecords(ctx, "documents", []Record{
		{"name": "Doc A"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	gi := GetSmoothContext(ctx)
	_, err = gi.Conn.Exec(ctx, `
		CREATE FUNCTION search_documents()
		RETURNS TABLE(document documents, score float8)
		LANGUAGE sql AS $$
			SELECT d::documents, 0.5::float8 FROM documents d;
		$$;
	`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.ReloadSchemaCache(ctx)
	if err != nil {
		t.Fatal(err)
	}

	result, _, err := ExecFunction(ctx, "search_documents", nil, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	raw := string(result)

	if !json.Valid(result) {
		t.Fatalf("invalid JSON output: %s", raw)
	}

	// The composite value should be a nested object with the document fields
	if !strings.Contains(raw, `"Doc A"`) {
		t.Fatalf("expected document name in output, got: %s", raw)
	}
}

func TestPartitionedTableFunction(t *testing.T) {
	// Test that functions returning rows from partitioned tables
	// include field names in JSON output.

	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}

	dbe.DeleteDatabase(ctx, "test_partition")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_partition")
	if err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)

	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)

	gi := GetSmoothContext(ctx)
	_, err = gi.Conn.Exec(ctx, `
		CREATE TABLE events (
			id serial,
			pid int NOT NULL,
			ts timestamptz NOT NULL DEFAULT now(),
			PRIMARY KEY (pid, id)
		) PARTITION BY LIST (pid);
		CREATE TABLE events_1 PARTITION OF events FOR VALUES IN (1);
		INSERT INTO events (pid) VALUES (1);

		CREATE FUNCTION get_events()
		RETURNS SETOF events
		LANGUAGE sql AS $$
			SELECT * FROM events;
		$$;
	`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.ReloadSchemaCache(ctx)
	if err != nil {
		t.Fatal(err)
	}

	result, _, err := ExecFunction(ctx, "get_events", nil, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	raw := string(result)

	if !json.Valid(result) {
		t.Fatalf("invalid JSON output: %s", raw)
	}

	// Must have field names, not just values
	if !strings.Contains(raw, `"id"`) || !strings.Contains(raw, `"pid"`) || !strings.Contains(raw, `"ts"`) {
		t.Fatalf("expected field names in output, got: %s", raw)
	}
}

func TestComputedRelationship(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}

	dbe.DeleteDatabase(ctx, "test_comp_rel")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_comp_rel")
	if err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)

	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)

	gi := GetSmoothContext(ctx)
	_, err = gi.Conn.Exec(ctx, `
		CREATE TABLE principals (
			id serial PRIMARY KEY,
			type text NOT NULL,
			name text NOT NULL
		);
		CREATE TABLE documents (
			id serial PRIMARY KEY,
			name text NOT NULL,
			read int[] NOT NULL DEFAULT '{}'
		);
		INSERT INTO principals (type, name) VALUES ('user', 'Alice'), ('user', 'Bob'), ('team', 'Admins');
		INSERT INTO documents (name, read) VALUES
			('Document A', '{1,2}'),
			('Document B', '{1,3}'),
			('Document C', '{}');

		CREATE FUNCTION read_principals(documents)
		RETURNS SETOF principals
		LANGUAGE sql STABLE
		AS $$
			SELECT * FROM principals WHERE id = ANY($1.read)
		$$;
	`)
	if err != nil {
		t.Fatal(err)
	}

	err = db.ReloadSchemaCache(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("computed relationship is discovered", func(t *testing.T) {
		rels := db.info.GetRelationships(_s("documents", "public"))
		found := false
		for _, rel := range rels {
			if rel.Type == Computed && rel.FunctionName == "read_principals" {
				found = true
				if !rel.ReturnIsSet {
					t.Error("expected ReturnIsSet to be true")
				}
				if rel.Table != "public.documents" {
					t.Errorf("expected Table=public.documents, got %s", rel.Table)
				}
				if rel.RelatedTable != "public.principals" {
					t.Errorf("expected RelatedTable=public.principals, got %s", rel.RelatedTable)
				}
				break
			}
		}
		if !found {
			t.Fatal("computed relationship 'read_principals' not found in relationships for documents")
		}
	})

	t.Run("select with computed relationship", func(t *testing.T) {
		filters := Filters{
			"select": {"id,name,read_principals(type,name)"},
			"order":  {"id"},
		}
		result, _, err := GetRecords(ctx, "documents", filters)
		if err != nil {
			t.Fatal(err)
		}

		raw := string(result)
		if !json.Valid(result) {
			t.Fatalf("invalid JSON output: %s", raw)
		}

		var rows []map[string]any
		if err := json.Unmarshal(result, &rows); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v (raw: %s)", err, raw)
		}
		if len(rows) != 3 {
			t.Fatalf("expected 3 rows, got %d (raw: %s)", len(rows), raw)
		}

		// Document A has read={1,2} -> Alice, Bob
		principals, ok := rows[0]["read_principals"].([]any)
		if !ok {
			t.Fatalf("expected read_principals to be array, got %T (raw: %s)", rows[0]["read_principals"], raw)
		}
		if len(principals) != 2 {
			t.Fatalf("expected 2 principals for Document A, got %d (raw: %s)", len(principals), raw)
		}

		// Document C has read={} -> empty array
		principals, ok = rows[2]["read_principals"].([]any)
		if !ok {
			t.Fatalf("expected read_principals to be array for Document C, got %T (raw: %s)", rows[2]["read_principals"], raw)
		}
		if len(principals) != 0 {
			t.Fatalf("expected 0 principals for Document C, got %d (raw: %s)", len(principals), raw)
		}
	})

	t.Run("select with computed relationship and filter", func(t *testing.T) {
		filters := Filters{
			"select":                {"id,name,read_principals(type,name)"},
			"read_principals.type":  {"eq.user"},
			"order":                 {"id"},
		}
		result, _, err := GetRecords(ctx, "documents", filters)
		if err != nil {
			t.Fatal(err)
		}

		raw := string(result)
		if !json.Valid(result) {
			t.Fatalf("invalid JSON output: %s", raw)
		}

		var rows []map[string]any
		if err := json.Unmarshal(result, &rows); err != nil {
			t.Fatalf("failed to unmarshal JSON: %v (raw: %s)", err, raw)
		}
		if len(rows) != 3 {
			t.Fatalf("expected 3 rows, got %d (raw: %s)", len(rows), raw)
		}

		// Document B has read={1,3} but filtered to type=user -> only Alice (id=1)
		principals := rows[1]["read_principals"].([]any)
		if len(principals) != 1 {
			t.Fatalf("expected 1 user principal for Document B (filter type=user), got %d (raw: %s)", len(principals), raw)
		}
		p := principals[0].(map[string]any)
		if p["name"] != "Alice" {
			t.Fatalf("expected Alice, got %v (raw: %s)", p["name"], raw)
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
			_, _, err := GetRecords(ctx, "b1", nil)
			if err != nil {
				log.Print(err)
				return
			}
		}
	})

	b.Run("Select2", func(b *testing.B) {
		SetQueryBuilder(ctx, QueryWithJSON{})
		for i := 0; i < b.N; i++ {
			_, _, err := GetRecords(ctx, "b1", nil)
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
