package database

import (
	"net/url"
	"testing"
)

// selfEmbedInfo is the schema cache of a table with a foreign key to itself,
// web_content.p_web_id -> web_content.id: the M2O relationship to the parent
// and the O2M one to the children, in the order the cache builds them.
func selfEmbedInfo() *SchemaInfo {
	return &SchemaInfo{
		cachedRelationships: map[string][]Relationship{
			"web_content": {
				{Type: M2O, Table: "web_content", Columns: []string{"p_web_id"},
					RelatedTable: "web_content", RelatedColumns: []string{"id"}, ForeignKey: "web_content_p_web_id_fkey"},
				{Type: O2M, Table: "web_content", Columns: []string{"id"},
					RelatedTable: "web_content", RelatedColumns: []string{"p_web_id"}, ForeignKey: "web_content_p_web_id_fkey"},
			},
		},
	}
}

type selfEmbedTest struct {
	query       string
	expectedSQL string
	values      []any
}

func runSelfEmbedTests(t *testing.T, tests []selfEmbedTest) {
	t.Helper()
	for i, test := range tests {
		u, err := url.Parse(test.query)
		if err != nil {
			t.Fatal(err)
		}
		parts, err := PostgRestParser{}.parse("web_content", u.Query())
		if err != nil {
			t.Fatalf("%d. unexpected parse error for %q: %v", i, test.query, err)
		}
		query, values, err := DirectQueryBuilder{}.BuildSelect("web_content", parts, &QueryOptions{}, selfEmbedInfo())
		if err != nil {
			t.Errorf("%d. unexpected build error for %q: %v", i, test.query, err)
			continue
		}
		if query != test.expectedSQL {
			t.Errorf("\n%d. Expected \n\t\"%v\", \ngot \n\t\"%v\" \n(query string -> \"%v\")", i, test.expectedSQL, query, test.query)
			continue
		}
		if !compareValues(values, test.values) {
			t.Errorf("\n%d. Expected values\n\t\"%v\", \ngot \n\t\"%v\" \n(query string -> \"%v\")", i, test.values, values, test.query)
		}
	}
}

// TestSelfEmbedFilter: a filter on an embed that follows a foreign key back to
// the same table goes into that embed's lateral subquery, on the alias of the
// related side (web_content_1, web_content_2 at the next level), as on any
// other embed. The top-level filters stay on the top level.
func TestSelfEmbedFilter(t *testing.T) {
	runSelfEmbedTests(t, []selfEmbedTest{
		{
			// the children (O2M)
			"?select=id,web_content(name)&web_content.name=eq.fezz",
			`SELECT "web_content"."id",  COALESCE("web_content_web_content_1"."_web_content_web_content_1", '[]') AS "web_content" FROM "web_content"  LEFT JOIN LATERAL ( SELECT json_agg("_web_content_web_content_1") AS "_web_content_web_content_1" FROM ( SELECT "web_content_1"."name" FROM "web_content" AS "web_content_1" WHERE "web_content_1"."p_web_id" = "web_content"."id" AND "web_content_1"."name" = 'fezz' ) AS "_web_content_web_content_1") AS "web_content_web_content_1" ON TRUE`,
			nil,
		},
		{
			// the parent (M2O, by the fk column), as an inner join: the filter
			// constrains the parent row and so the top-level rows
			"?select=id,p_web_id!inner(name)&p_web_id.name=eq.wat",
			`SELECT "web_content"."id",  row_to_json("web_content_p_web_id_1".*) AS "p_web_id" FROM "web_content"  INNER JOIN LATERAL ( SELECT "web_content_1"."name" FROM "web_content" AS "web_content_1" WHERE "web_content_1"."id" = "web_content"."p_web_id" AND "web_content_1"."name" = 'wat') AS "web_content_p_web_id_1" ON TRUE`,
			nil,
		},
		{
			// by the embed's alias
			"?select=id,children:web_content(name)&children.name=eq.bar",
			`SELECT "web_content"."id",  COALESCE("web_content_children_1"."_web_content_children_1", '[]') AS "children" FROM "web_content"  LEFT JOIN LATERAL ( SELECT json_agg("_web_content_children_1") AS "_web_content_children_1" FROM ( SELECT "web_content_1"."name" FROM "web_content" AS "web_content_1" WHERE "web_content_1"."p_web_id" = "web_content"."id" AND "web_content_1"."name" = 'bar' ) AS "_web_content_children_1") AS "web_content_children_1" ON TRUE`,
			nil,
		},
		{
			// the parent and the children side by side, both embedding
			// web_content: each filter goes to one embed, once (UpdateSpec
			// "embeds parent, children and grandchildren after update")
			"?select=id,web_content(name),parent_content:p_web_id(name)&parent_content.name=neq.wat&web_content.name=eq.fezz",
			`SELECT "web_content"."id",  COALESCE("web_content_web_content_1"."_web_content_web_content_1", '[]') AS "web_content",  row_to_json("web_content_parent_content_1".*) AS "parent_content" FROM "web_content"  LEFT JOIN LATERAL ( SELECT json_agg("_web_content_web_content_1") AS "_web_content_web_content_1" FROM ( SELECT "web_content_1"."name" FROM "web_content" AS "web_content_1" WHERE "web_content_1"."p_web_id" = "web_content"."id" AND "web_content_1"."name" = 'fezz' ) AS "_web_content_web_content_1") AS "web_content_web_content_1" ON TRUE LEFT JOIN LATERAL ( SELECT "web_content_1"."name" FROM "web_content" AS "web_content_1" WHERE "web_content_1"."id" = "web_content"."p_web_id" AND "web_content_1"."name" <> 'wat') AS "web_content_parent_content_1" ON TRUE`,
			nil,
		},
		{
			// the same with a logic tree: it goes to the first embed its path
			// names, and only there
			"?select=id,web_content(name),parent_content:p_web_id(name)&web_content.or=(name.eq.fezz,name.eq.foo)",
			`SELECT "web_content"."id",  COALESCE("web_content_web_content_1"."_web_content_web_content_1", '[]') AS "web_content",  row_to_json("web_content_parent_content_1".*) AS "parent_content" FROM "web_content"  LEFT JOIN LATERAL ( SELECT json_agg("_web_content_web_content_1") AS "_web_content_web_content_1" FROM ( SELECT "web_content_1"."name" FROM "web_content" AS "web_content_1" WHERE "web_content_1"."p_web_id" = "web_content"."id" AND ("web_content_1"."name" = 'fezz' OR "web_content_1"."name" = 'foo') ) AS "_web_content_web_content_1") AS "web_content_web_content_1" ON TRUE LEFT JOIN LATERAL ( SELECT "web_content_1"."name" FROM "web_content" AS "web_content_1" WHERE "web_content_1"."id" = "web_content"."p_web_id") AS "web_content_parent_content_1" ON TRUE`,
			nil,
		},
		{
			// two levels: each filter on its own level, the top-level one on top
			"?id=eq.0&select=id,web_content(name,web_content(name))&web_content.name=eq.fezz&web_content.web_content.id=eq.4",
			`SELECT "web_content"."id",  COALESCE("web_content_web_content_1"."_web_content_web_content_1", '[]') AS "web_content" FROM "web_content"  LEFT JOIN LATERAL ( SELECT json_agg("_web_content_web_content_1") AS "_web_content_web_content_1" FROM ( SELECT "web_content_1"."name",  COALESCE("web_content_web_content_2"."_web_content_web_content_2", '[]') AS "web_content" FROM "web_content" AS "web_content_1"  LEFT JOIN LATERAL ( SELECT json_agg("_web_content_web_content_2") AS "_web_content_web_content_2" FROM ( SELECT "web_content_2"."name" FROM "web_content" AS "web_content_2" WHERE "web_content_2"."p_web_id" = "web_content_1"."id" AND "web_content_2"."id" = '4' ) AS "_web_content_web_content_2") AS "web_content_web_content_2" ON TRUE WHERE "web_content_1"."p_web_id" = "web_content"."id" AND "web_content_1"."name" = 'fezz' ) AS "_web_content_web_content_1") AS "web_content_web_content_1" ON TRUE WHERE "web_content"."id" = $1`,
			[]any{"0"},
		},
	})
}
