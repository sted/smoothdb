package database

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// The fixture of PostgREST's self embed on a view: api.job is a view over
// private.internal_job, which has a foreign key to itself (parent_fk). Two
// other relations give the positive controls: api.client <- api.project is
// a foreign key between two exposed tables, and api.tag <-> api.project is
// a many-to-many through a junction in the private schema
// (private.project_tag) and one through a junction in api (api.project_note).
var relationshipOrderDDL = []string{
	`create schema private`,
	`create schema api`,
	`create table private.internal_job (
		id int primary key,
		parent_id int,
		constraint parent_fk foreign key (parent_id) references private.internal_job (id)
	)`,
	`create view api.job as select j.id, j.parent_id from private.internal_job j`,
	`create table api.client (id int primary key, name text)`,
	`create table api.project (id int primary key, client_id int references api.client (id))`,
	`create table api.tag (id int primary key, name text)`,
	`create table private.project_tag (
		project_id int references api.project (id),
		tag_id int references api.tag (id),
		primary key (project_id, tag_id)
	)`,
	`create table api.note (id int primary key)`,
	`create table api.project_note (
		project_id int references api.project (id),
		note_id int references api.note (id),
		primary key (project_id, note_id)
	)`,
}

func relationshipOrderDb(t *testing.T) (context.Context, *Database, func()) {
	t.Helper()
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_rel_order")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_rel_order")
	if err != nil {
		t.Fatal(err)
	}
	ReleaseConn(ctx, conn)
	ctx, conn, err = ContextWithDb(context.Background(), db, "test")
	if err != nil {
		t.Fatal(err)
	}
	gi := GetSmoothContext(ctx)
	for _, q := range relationshipOrderDDL {
		if _, err := gi.Conn.Exec(ctx, q); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}
	return ctx, db, func() { ReleaseConn(ctx, conn) }
}

func describeRelationships(rels []Relationship) string {
	var b strings.Builder
	for _, r := range rels {
		fmt.Fprintf(&b, "%d %s%v -> %s%v via %q fk %s\n", r.Type, r.Table, r.Columns, r.RelatedTable, r.RelatedColumns, r.JunctionTable, r.ForeignKey)
	}
	return b.String()
}

// The relationships of a view over a self-referencing table came out in map
// order: api.job got a many-to-one to api.job and one to private.internal_job,
// both on parent_id, in a random order. An embed by column, parent_id(*),
// takes the first one, so on some schema loads it read the private table
// (42501 for a role without USAGE on it). Every reload must give the same
// relationships in the same order, the view's own first. ExposedSchemas is
// empty here, so nothing is removed: the order alone decides.
func TestRelationshipOrderIsDeterministic(t *testing.T) {
	ctx, db, release := relationshipOrderDb(t)
	defer release()

	var first string
	for i := 0; i < 30; i++ {
		if err := db.ReloadSchemaCache(ctx); err != nil {
			t.Fatal(err)
		}
		info := db.info.Load()
		rel := info.FindRelationshipByCol("api.job", "parent_id")
		if rel == nil || rel.Type != M2O || rel.RelatedTable != "api.job" {
			t.Fatalf("reload %d: parent_id must resolve to the many-to-one api.job -> api.job, got\n%s\nof\n%s",
				i, describeRelationships([]Relationship{derefRel(rel)}), describeRelationships(info.GetRelationships("api.job")))
		}
		all := describeRelationships(info.GetRelationships("api.job")) +
			describeRelationships(info.GetRelationships("private.internal_job")) +
			describeRelationships(info.GetRelationships("api.project"))
		if i == 0 {
			first = all
		} else if all != first {
			t.Fatalf("reload %d: the relationships changed\nfirst:\n%s\nnow:\n%s", i, first, all)
		}
	}
}

func derefRel(r *Relationship) Relationship {
	if r == nil {
		return Relationship{}
	}
	return *r
}

// PostgREST's removeInternal (SchemaCache.hs): with exposed schemas
// configured, a relationship to a relation outside them, or through a
// junction outside them, is not in the schema cache. The controls: the
// same relationships are there when every schema is exposed, and the
// relationships between exposed relations stay.
func TestRelationshipsToInternalSchemasAreRemoved(t *testing.T) {
	ctx, db, release := relationshipOrderDb(t)
	defer release()

	saved := dbe.config.ExposedSchemas
	defer func() { dbe.config.ExposedSchemas = saved }()

	internal := func(info *SchemaInfo, table string) []Relationship {
		var found []Relationship
		for _, r := range info.GetRelationships(table) {
			if strings.HasPrefix(r.RelatedTable, "private.") || strings.HasPrefix(r.JunctionTable, "private.") {
				found = append(found, r)
			}
		}
		return found
	}
	has := func(info *SchemaInfo, table, related, junction string, typ RelType) bool {
		for _, r := range info.GetRelationships(table) {
			if r.RelatedTable == related && r.JunctionTable == junction && r.Type == typ {
				return true
			}
		}
		return false
	}

	// control: every schema exposed, the internal relationships are there
	dbe.config.ExposedSchemas = nil
	if err := db.ReloadSchemaCache(ctx); err != nil {
		t.Fatal(err)
	}
	info := db.info.Load()
	if n := len(internal(info, "api.job")); n == 0 {
		t.Fatalf("control: expected relationships from api.job to private, got none of\n%s", describeRelationships(info.GetRelationships("api.job")))
	}
	if !has(info, "api.project", "api.tag", "private.project_tag", M2M) {
		t.Fatalf("control: expected the M2M api.project -> api.tag via private.project_tag in\n%s", describeRelationships(info.GetRelationships("api.project")))
	}

	dbe.config.ExposedSchemas = []string{"api"}
	if err := db.ReloadSchemaCache(ctx); err != nil {
		t.Fatal(err)
	}
	info = db.info.Load()
	for _, table := range []string{"api.job", "api.project", "api.tag", "api.client", "api.note"} {
		if found := internal(info, table); len(found) > 0 {
			t.Errorf("%s: expected no relationship to or through private, got\n%s", table, describeRelationships(found))
		}
	}
	// the exposed relationships stay
	for _, c := range []struct {
		table, related, junction string
		typ                      RelType
	}{
		{"api.job", "api.job", "", M2O},
		{"api.job", "api.job", "", O2M},
		{"api.project", "api.client", "", M2O},
		{"api.client", "api.project", "", O2M},
		{"api.project", "api.note", "api.project_note", M2M},
		{"api.note", "api.project", "api.project_note", M2M},
	} {
		if !has(info, c.table, c.related, c.junction, c.typ) {
			t.Errorf("%s: expected the relationship %d to %s via %q, got\n%s", c.table, c.typ, c.related, c.junction,
				describeRelationships(info.GetRelationships(c.table)))
		}
	}
}

// A many-to-many between two views goes through the junction an exposed
// schema holds, even when a view in a hidden schema exposes the same
// junction. The surfaces of the junction were sorted views first: the hidden
// view won the deduplication of the endpoints, removeInternal then dropped
// it, and the two views lost the relationship.
func TestHiddenJunctionViewDoesNotHideExposedJunction(t *testing.T) {
	ddl := []string{
		`create schema private`,
		`create schema api`,
		`create table private.p (id int primary key)`,
		`create table private.n (id int primary key)`,
		`create table api.pn (
			p_id int references private.p (id),
			n_id int references private.n (id),
			primary key (p_id, n_id)
		)`,
		`create view api.pv as select id from private.p`,
		`create view api.nv as select id from private.n`,
		`create view private.pn_v as select p_id, n_id from api.pn`,
	}
	ctx, conn, err := ContextWithDb(context.Background(), nil, "test")
	if err != nil {
		t.Fatal(err)
	}
	dbe.DeleteDatabase(ctx, "test_rel_junction")
	db, err := dbe.GetOrCreateActiveDatabase(ctx, "test_rel_junction")
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
	for _, q := range ddl {
		if _, err := gi.Conn.Exec(ctx, q); err != nil {
			t.Fatalf("%v\n%s", err, q)
		}
	}

	saved := dbe.config.ExposedSchemas
	defer func() { dbe.config.ExposedSchemas = saved }()
	dbe.config.ExposedSchemas = []string{"api"}
	if err := db.ReloadSchemaCache(ctx); err != nil {
		t.Fatal(err)
	}
	info := db.info.Load()
	for _, c := range []struct{ table, related string }{{"api.pv", "api.nv"}, {"api.nv", "api.pv"}} {
		found := false
		for _, r := range info.GetRelationships(c.table) {
			if r.Type == M2M && r.RelatedTable == c.related && r.JunctionTable == "api.pn" {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: expected the M2M to %s via api.pn, got\n%s", c.table, c.related,
				describeRelationships(info.GetRelationships(c.table)))
		}
	}
}
