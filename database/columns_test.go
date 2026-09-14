package database

import (
	"context"
	"testing"
)

// The schema cache is loaded by the connecting role, which in a PostgREST-style
// deployment is an authenticator with no privilege on the tables it serves
// (ragtool connects as authenticator and SET ROLEs per request).
// information_schema.columns hides from a role the columns of every table it
// has no privilege on, so a cache read through it is empty for those tables:
// ?columns= is then refused on all of them (PGRST204) and their generated
// columns go undetected. The columns must come from pg_catalog, which hides
// nothing.
func TestColumnsWithoutPrivileges(t *testing.T) {
	ctx, conn, err := ContextWithDb(context.Background(), nil, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_privileges")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_privileges")
	if err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)

	ctx, conn, err = ContextWithDb(context.Background(), db, "postgres")
	if err != nil {
		t.Fatal(err)
	}
	defer ReleaseConn(ctx, conn)

	c := GetConn(ctx)
	for _, sql := range []string{
		`CREATE DOMAIN shade AS text`,
		`CREATE TABLE unseen (id int PRIMARY KEY, color shade NOT NULL DEFAULT 'red',
			twice int GENERATED ALWAYS AS (id * 2) STORED, tags text[])`,
		`COMMENT ON COLUMN unseen.color IS 'the shade'`,
		`DO $$ BEGIN CREATE ROLE unseen_reader NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
		`SET ROLE unseen_reader`,
	} {
		if _, err := c.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	defer c.Exec(ctx, "RESET ROLE")

	columns, err := GetColumns(ctx, "unseen")
	if err != nil {
		t.Fatal(err)
	}
	if len(columns) != 4 {
		t.Fatalf("expected the 4 columns of unseen as a role with no privilege on it, got %d", len(columns))
	}
	if columns[0].Name != "id" || columns[0].Type != "int4" || !columns[0].NotNull || columns[0].Default != nil {
		t.Errorf("id: %+v", columns[0])
	}
	// A domain reports its base type, as information_schema did.
	if columns[1].Name != "color" || columns[1].Type != "text" || !columns[1].NotNull ||
		columns[1].Default == nil || *columns[1].Default != "'red'::text" ||
		columns[1].Comment == nil || *columns[1].Comment != "the shade" {
		t.Errorf("color: %+v", columns[1])
	}
	// A generated column has no default: the expression is not one.
	if columns[2].Name != "twice" || columns[2].Generated != "stored" || !columns[2].ReadOnly || columns[2].Default != nil {
		t.Errorf("twice: %+v", columns[2])
	}
	if columns[3].Name != "tags" || columns[3].Type != "_text" || columns[3].NotNull {
		t.Errorf("tags: %+v", columns[3])
	}

	types, err := GetColumnTypes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]ColumnType{}
	for _, ct := range types {
		if ct.Table == "unseen" && ct.Schema == "public" {
			seen[ct.Name] = ct
		}
	}
	if len(seen) != 4 {
		t.Fatalf("expected the 4 column types of unseen as a role with no privilege on it, got %d: %v", len(seen), seen)
	}
	if seen["color"].Type != "text" || seen["tags"].Type != "_text" || !seen["tags"].IsArray || seen["id"].IsArray {
		t.Errorf("column types: %+v", seen)
	}
}
