package database

import (
	"net/url"
	"strings"
	"testing"
)

// columnHintInfo is the schema cache of a table, document_full, whose
// target_id column reaches two relations: the view document_core and the
// view zz_tgt_base, both carrying the same foreign key (card 34668
// column-hint-embed). The table t has two foreign keys on the same column c
// to the same table r, so a hint by c cannot choose between them.
func columnHintInfo() *SchemaInfo {
	return &SchemaInfo{
		cachedRelationships: map[string][]Relationship{
			"document_full": {
				{Type: M2O, Table: "document_full", Columns: []string{"target_id"},
					RelatedTable: "document_core", RelatedColumns: []string{"id"}, ForeignKey: "document_target_id_fkey"},
				{Type: M2O, Table: "document_full", Columns: []string{"target_id"},
					RelatedTable: "zz_tgt_base", RelatedColumns: []string{"id"}, ForeignKey: "document_target_id_fkey"},
			},
			"t": {
				{Type: M2O, Table: "t", Columns: []string{"c"},
					RelatedTable: "r", RelatedColumns: []string{"id"}, ForeignKey: "t_c_fkey1"},
				{Type: M2O, Table: "t", Columns: []string{"c"},
					RelatedTable: "r", RelatedColumns: []string{"code"}, ForeignKey: "t_c_fkey2"},
			},
		},
	}
}

func buildColumnHint(t *testing.T, table, query string) (string, error) {
	t.Helper()
	u, err := url.Parse(query)
	if err != nil {
		t.Fatal(err)
	}
	parts, err := PostgRestParser{}.parse(table, u.Query())
	if err != nil {
		t.Fatalf("unexpected parse error for %q: %v", query, err)
	}
	sql, _, err := DirectQueryBuilder{}.BuildSelect(table, parts, &QueryOptions{}, columnHintInfo())
	return sql, err
}

// TestColumnHintEmbed: an embed hinted by a column, view!fk_column(...),
// reads the relation it names, as PostgREST does (Plan.hs findRel matches the
// target first, then the hint against the constraint or its single column).
// It used to read the first relation that carried the column, whatever its
// name, and so bypassed the filter of the view named.
func TestColumnHintEmbed(t *testing.T) {
	for _, tc := range []struct {
		query, want, notWant string
	}{
		{"?select=id,target:zz_tgt_base!target_id(id,name)", `FROM "zz_tgt_base"`, `"document_core"`},
		{"?select=id,target:document_core!target_id(id,name)", `FROM "document_core"`, `"zz_tgt_base"`},
		// the constraint name hint, which already worked
		{"?select=id,target:zz_tgt_base!document_target_id_fkey(id,name)", `FROM "zz_tgt_base"`, `"document_core"`},
		// the referenced column as the hint
		{"?select=id,target:zz_tgt_base!id(id,name)", `FROM "zz_tgt_base"`, `"document_core"`},
	} {
		sql, err := buildColumnHint(t, "document_full", tc.query)
		if err != nil {
			t.Errorf("%q: unexpected build error: %v", tc.query, err)
			continue
		}
		if !strings.Contains(sql, tc.want) || strings.Contains(sql, tc.notWant) {
			t.Errorf("%q: want %s and no %s, got\n\t%s", tc.query, tc.want, tc.notWant, sql)
		}
	}

	// a column hint naming no relationship of the relation named is not found,
	// even if another relation carries the column
	if _, err := buildColumnHint(t, "document_full", "?select=id,nosuch!target_id(id)"); err == nil ||
		!strings.Contains(err.Error(), "cannot find relationship") {
		t.Errorf("nosuch!target_id: want a not-found error, got %v", err)
	}

	// two relationships to the relation named carry the hinted column: the
	// embed is ambiguous, as without a hint
	_, err := buildColumnHint(t, "t", "?select=c,r!c(id)")
	if err == nil || !strings.Contains(err.Error(), "more than one possible relationship") {
		t.Errorf("r!c: want an ambiguity error, got %v", err)
	}
	// the constraint name settles it
	if sql, err := buildColumnHint(t, "t", "?select=c,r!t_c_fkey2(id)"); err != nil || !strings.Contains(sql, `"code"`) {
		t.Errorf("r!t_c_fkey2: want the join on code, got %v\n\t%s", err, sql)
	}
}

// TestColumnHintSelfEmbed: in a self-relationship a column hint selects the
// one-to-many side by the referencing column, the children, as PostgREST
// does (/organizations?select=auditees:organizations!auditor(*)); it used to
// select the parent.
func TestColumnHintSelfEmbed(t *testing.T) {
	runSelfEmbedTests(t, []selfEmbedTest{
		{
			"?select=id,children:web_content!p_web_id(name)",
			`SELECT "web_content"."id",  COALESCE("web_content_children_1"."_web_content_children_1", '[]') AS "children" FROM "web_content"  LEFT JOIN LATERAL ( SELECT json_agg("_web_content_children_1") AS "_web_content_children_1" FROM ( SELECT "web_content_1"."name" FROM "web_content" AS "web_content_1" WHERE "web_content_1"."p_web_id" = "web_content"."id" ) AS "_web_content_children_1") AS "web_content_children_1" ON TRUE`,
			nil,
		},
	})
}
