package database

import (
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Card 35451 builder-correctness.

var markerRe = regexp.MustCompile(`\$([0-9]+)`)

// checkMarkers fails unless the markers of sql are exactly $1..$len(values):
// every value is bound and every marker has its value.
func checkMarkers(t *testing.T, what, sql string, values []any) {
	t.Helper()
	seen := map[int]bool{}
	for _, m := range markerRe.FindAllStringSubmatch(sql, -1) {
		n, _ := strconv.Atoi(m[1])
		seen[n] = true
	}
	for n := 1; n <= len(values); n++ {
		if !seen[n] {
			t.Errorf("%s: $%d is not in the SQL (values %v): %s", what, n, values, sql)
		}
	}
	for n := range seen {
		if n < 1 || n > len(values) {
			t.Errorf("%s: $%d has no value (values %v): %s", what, n, values, sql)
		}
	}
}

func parseFor(t *testing.T, table, query string) *QueryParts {
	t.Helper()
	u, err := url.Parse(query)
	if err != nil {
		t.Fatal(err)
	}
	parts, err := PostgRestParser{}.parse(table, u.Query())
	if err != nil {
		t.Fatalf("%s: parse error: %v", query, err)
	}
	return parts
}

// countInfo is the schema cache of TestRangedCountTotal.
func countInfo() *SchemaInfo {
	return &SchemaInfo{
		cachedRelationships: map[string][]Relationship{
			"projects": {{Type: M2O, Table: "projects", Columns: []string{"client_id"},
				RelatedTable: "clients", RelatedColumns: []string{"id"}},
				{Type: O2M, Table: "projects", Columns: []string{"id"},
					RelatedTable: "tasks", RelatedColumns: []string{"project_id"}}},
			"tasks": {{Type: M2O, Table: "tasks", Columns: []string{"project_id"},
				RelatedTable: "projects", RelatedColumns: []string{"id"}}},
			"clients": {{Type: O2M, Table: "clients", Columns: []string{"id"},
				RelatedTable: "projects", RelatedColumns: []string{"client_id"}}},
		},
	}
}

// 1. An embed filter value was inlined with quoteLit, never bound: each value
// was its own statement text, and its safety rested on quoteLit alone.
func TestEmbedFilterValuesAreBound(t *testing.T) {
	tests := []struct {
		query  string
		sql    string
		values []any
	}{
		{
			"?select=id,web_content(id)&web_content.name=eq.x",
			`SELECT "web_content"."id",  COALESCE("web_content_web_content_1"."_web_content_web_content_1", '[]') AS "web_content" FROM "web_content"  LEFT JOIN LATERAL ( SELECT json_agg("_web_content_web_content_1") AS "_web_content_web_content_1" FROM ( SELECT "web_content_1"."id" FROM "web_content" AS "web_content_1" WHERE "web_content_1"."p_web_id" = "web_content"."id" AND "web_content_1"."name" = $1 ) AS "_web_content_web_content_1") AS "web_content_web_content_1" ON TRUE`,
			[]any{"x"},
		},
		{
			// the root filters come first, then the embeds'
			"?select=id,web_content(id)&web_content.name=in.(x,y)&id=eq.5",
			`SELECT "web_content"."id",  COALESCE("web_content_web_content_1"."_web_content_web_content_1", '[]') AS "web_content" FROM "web_content"  LEFT JOIN LATERAL ( SELECT json_agg("_web_content_web_content_1") AS "_web_content_web_content_1" FROM ( SELECT "web_content_1"."id" FROM "web_content" AS "web_content_1" WHERE "web_content_1"."p_web_id" = "web_content"."id" AND "web_content_1"."name" IN ($2, $3) ) AS "_web_content_web_content_1") AS "web_content_web_content_1" ON TRUE WHERE "web_content"."id" = $1`,
			[]any{"5", "x", "y"},
		},
		{
			// a value with a quote is a value, not SQL
			"?select=id,web_content(id)&web_content.name=eq.O'Brien&web_content.or=(id.eq.1,id.eq.2)&limit=3",
			`SELECT "web_content"."id",  COALESCE("web_content_web_content_1"."_web_content_web_content_1", '[]') AS "web_content" FROM "web_content"  LEFT JOIN LATERAL ( SELECT json_agg("_web_content_web_content_1") AS "_web_content_web_content_1" FROM ( SELECT "web_content_1"."id" FROM "web_content" AS "web_content_1" WHERE "web_content_1"."p_web_id" = "web_content"."id" AND "web_content_1"."name" = $1 AND ("web_content_1"."id" = $2 OR "web_content_1"."id" = $3) ) AS "_web_content_web_content_1") AS "web_content_web_content_1" ON TRUE LIMIT $4`,
			nil,
		},
	}
	for _, test := range tests {
		parts := parseFor(t, "web_content", test.query)
		sql, values, err := DirectQueryBuilder{}.BuildSelect("web_content", parts, &QueryOptions{}, selfEmbedInfo())
		if err != nil {
			t.Fatalf("%s: build error: %v", test.query, err)
		}
		if sql != test.sql {
			t.Errorf("%s\n want: %s\n  got: %s", test.query, test.sql, sql)
		}
		if test.values != nil && !reflect.DeepEqual(values, test.values) {
			t.Errorf("%s: values %#v, want %#v", test.query, values, test.values)
		}
		checkMarkers(t, test.query, sql, values)
	}
}

// The total of a ranged count reads the root filters and the !inner embeds
// only: its statement must bind exactly the values it references, whatever
// the left embeds bind in the page.
func TestRangedCountBindsEmbedValues(t *testing.T) {
	tests := []struct {
		table, query string
		total        string
		totalValues  []any
	}{
		{
			"projects", "?select=id,clients!inner(id)&clients.id=eq.1&limit=1",
			`SELECT COUNT(*) FROM "projects" WHERE EXISTS ( SELECT 1 FROM "clients" AS "clients_1" WHERE "clients_1"."id" = "projects"."client_id" AND "clients_1"."id" = $1)`,
			[]any{"1"},
		},
		{
			// a left embed's filter is in the page, not in the total
			"projects", "?select=id,tasks(id),clients!inner(id)&tasks.id=eq.7&clients.id=eq.1&id=gt.0&limit=1",
			`SELECT COUNT(*) FROM "projects" WHERE "projects"."id" > $1 AND EXISTS ( SELECT 1 FROM "clients" AS "clients_1" WHERE "clients_1"."id" = "projects"."client_id" AND "clients_1"."id" = $2)`,
			[]any{"0", "1"},
		},
		{
			"tasks", "?select=id,projects!inner(id,clients!inner(id))&projects.clients.id=eq.1&projects.id=lt.9&limit=1",
			`SELECT COUNT(*) FROM "tasks" WHERE EXISTS ( SELECT 1 FROM "projects" AS "projects_1" WHERE "projects_1"."id" = "tasks"."project_id" AND "projects_1"."id" < $2 AND EXISTS ( SELECT 1 FROM "clients" AS "clients_2" WHERE "clients_2"."id" = "projects_1"."client_id" AND "clients_2"."id" = $1))`,
			[]any{"1", "9"},
		},
	}
	for _, test := range tests {
		parts := parseFor(t, test.table, test.query)
		options := &QueryOptions{Count: "exact"}
		sql, values, err := DirectQueryBuilder{}.BuildSelect(test.table, parts, options, countInfo())
		if err != nil {
			t.Fatalf("%s: build error: %v", test.query, err)
		}
		checkMarkers(t, test.query+" (page)", sql, values)
		checkMarkers(t, test.query+" (total)", options.totalQuery, options.totalValues)
		if options.totalQuery != test.total {
			t.Errorf("%s: total\n want: %s\n  got: %s", test.query, test.total, options.totalQuery)
		}
		if !reflect.DeepEqual(options.totalValues, test.totalValues) {
			t.Errorf("%s: total values %#v, want %#v", test.query, options.totalValues, test.totalValues)
		}
	}
}

// 2. A dotted column name ("a.b", a quoted name in the URL) was split at the
// root, "t"."a"."b", and kept whole on an embed. PostgREST 16.4 takes it as
// one column at every level (live: ?"a.b"=eq.1 answers its rows).
func TestDottedColumnNameIsOneColumn(t *testing.T) {
	tests := []struct {
		query string
		want  []string // fragments of the SQL
	}{
		{`?"a.b"=eq.1`, []string{`WHERE "web_content"."a.b" = $1`}},
		{`?order="a.b".desc`, []string{`ORDER BY "web_content"."a.b" DESC`}},
		{`?select="a.b"`, []string{`SELECT "web_content"."a.b" FROM`}},
		{`?select=id,web_content(id)&web_content."a.b"=eq.1`, []string{`"web_content_1"."a.b" = $1`}},
		{`?select=id,web_content(id)&web_content.order="a.b"`, []string{`ORDER BY "web_content_1"."a.b"`}},
		{`?or=("a.b".eq.1,id.eq.2)`, []string{`("web_content"."a.b" = $1 OR "web_content"."id" = $2)`}},
	}
	for _, test := range tests {
		parts := parseFor(t, "web_content", test.query)
		sql, _, err := DirectQueryBuilder{}.BuildSelect("web_content", parts, &QueryOptions{}, selfEmbedInfo())
		if err != nil {
			t.Fatalf("%s: build error: %v", test.query, err)
		}
		for _, w := range test.want {
			if !strings.Contains(sql, w) {
				t.Errorf("%s: want %s in\n %s", test.query, w, sql)
			}
		}
		if strings.Contains(sql, `"a"."b"`) {
			t.Errorf("%s: the name is split: %s", test.query, sql)
		}
	}
}

// 3. Building marked the filters of the parsed request as taken (and patched
// its orders), so a second build of one QueryParts dropped the WHERE.
func TestBuildingTwiceGivesTheSameSQL(t *testing.T) {
	selects := []string{
		"?name=eq.x",
		"?or=(id.eq.1,name.eq.y)&id=gt.0",
		"?select=id,web_content(id)&web_content.name=eq.x&id=eq.5&order=id.desc",
		"?select=id,p_web_id!inner(name)&p_web_id.name=eq.wat&web_content.or=(id.eq.1)",
	}
	for _, q := range selects {
		parts := parseFor(t, "web_content", q)
		for _, count := range []string{"", "exact"} {
			sql1, v1, err1 := DirectQueryBuilder{}.BuildSelect("web_content", parts, &QueryOptions{Count: count}, selfEmbedInfo())
			sql2, v2, err2 := DirectQueryBuilder{}.BuildSelect("web_content", parts, &QueryOptions{Count: count}, selfEmbedInfo())
			if err1 != nil || err2 != nil {
				t.Errorf("%s: errors %v, %v", q, err1, err2)
				continue
			}
			if sql1 != sql2 || !reflect.DeepEqual(v1, v2) {
				t.Errorf("%s (count %q): second build differs\n first: %s %v\nsecond: %s %v", q, count, sql1, v1, sql2, v2)
			}
		}
	}
	// the writes
	parts := parseFor(t, "web_content", "?id=eq.1&name=neq.x")
	u1, uv1, _ := CommonBuilder{}.BuildUpdate("web_content", Record{"name": "z"}, parts, &QueryOptions{}, nil)
	u2, uv2, _ := CommonBuilder{}.BuildUpdate("web_content", Record{"name": "z"}, parts, &QueryOptions{}, nil)
	if u1 != u2 || !reflect.DeepEqual(uv1, uv2) || !strings.Contains(u2, "WHERE") {
		t.Errorf("update: second build differs\n first: %s %v\nsecond: %s %v", u1, uv1, u2, uv2)
	}
	d1, dv1, _ := CommonBuilder{}.BuildDelete("web_content", parts, &QueryOptions{}, nil)
	d2, dv2, _ := CommonBuilder{}.BuildDelete("web_content", parts, &QueryOptions{}, nil)
	if d1 != d2 || !reflect.DeepEqual(dv1, dv2) || !strings.Contains(d2, "WHERE") {
		t.Errorf("delete: second build differs\n first: %s %v\nsecond: %s %v", d1, dv1, d2, dv2)
	}
	// a function call, its filters and its orders
	parts = parseFor(t, "fn", "?id=eq.1&order=id.desc")
	orderBefore := parts.orderFields[0].field.tablename
	e1, ev1, _ := CommonBuilder{}.BuildExecute("fn", nil, Record{}, parts, &QueryOptions{}, nil)
	e2, ev2, _ := CommonBuilder{}.BuildExecute("fn", nil, Record{}, parts, &QueryOptions{}, nil)
	if e1 != e2 || !reflect.DeepEqual(ev1, ev2) || !strings.Contains(e2, "WHERE") {
		t.Errorf("execute: second build differs\n first: %s %v\nsecond: %s %v", e1, ev1, e2, ev2)
	}
	if parts.orderFields[0].field.tablename != orderBefore {
		t.Errorf("execute: the order of the parsed request was patched: %q, was %q",
			parts.orderFields[0].field.tablename, orderBefore)
	}
}

// 4. The recursive select rewrote the finished SQL with strings.ReplaceAll,
// "table". to "__recursive"., values included.
func TestRecursiveSelectKeepsValues(t *testing.T) {
	value := `a"web_content".b`
	parts := parseFor(t, "web_content",
		`?id=start.1&p_web_id=recurse.all&select=id,web_content(id)&web_content.name=eq.`+url.QueryEscape(value)+`&name=neq.`+url.QueryEscape(value))
	sql, values, err := DirectQueryBuilder{}.BuildSelect("web_content", parts, &QueryOptions{}, selfEmbedInfo())
	if err != nil {
		t.Fatalf("build error: %v", err)
	}
	n := 0
	for _, v := range values {
		if v == value {
			n++
		}
	}
	if n != 2 {
		t.Errorf("want the value %q twice in %#v", value, values)
	}
	if strings.Contains(sql, `__recursive".b`) || strings.Contains(sql, "'a") {
		t.Errorf("a value was rewritten into the SQL: %s", sql)
	}
	checkMarkers(t, "recursive", sql, values)
	for _, w := range []string{
		`FROM "__recursive"  LEFT JOIN LATERAL`,
		`"web_content_1"."p_web_id" = "__recursive"."id"`,
		`WHERE "__recursive"."name" <> $`,
		`SELECT "__recursive"."id",`,
	} {
		if !strings.Contains(sql, w) {
			t.Errorf("want %s in\n %s", w, sql)
		}
	}
}

// 5. Small ones.
func TestBuilderSmallDefects(t *testing.T) {
	// an empty embed first in the select left a stray comma (42601);
	// PostgREST 16.4 answers the other columns (live: [{"id":1}])
	for _, q := range []string{"?select=web_content(),id", "?select=web_content(),id,name", "?select=id,web_content()"} {
		parts := parseFor(t, "web_content", q)
		sql, _, err := DirectQueryBuilder{}.BuildSelect("web_content", parts, &QueryOptions{}, selfEmbedInfo())
		if err != nil {
			t.Errorf("%s: build error: %v", q, err)
			continue
		}
		if strings.Contains(sql, "SELECT ,") || strings.Contains(sql, ", ,") || strings.Contains(sql, ",  FROM") {
			t.Errorf("%s: stray comma: %s", q, sql)
		}
		if !strings.HasPrefix(sql, `SELECT "web_content"."id"`) {
			t.Errorf("%s: %s", q, sql)
		}
	}

	// the spread error named a table not yet assigned
	parts := parseFor(t, "web_content", "?select=id,...web_content(id)")
	_, _, err := DirectQueryBuilder{}.BuildSelect("web_content", parts, &QueryOptions{}, selfEmbedInfo())
	if err == nil || !strings.Contains(err.Error(), "spread operation on web_content ") {
		t.Errorf("spread on a to-many embed: want an error naming web_content, got %v", err)
	}

	// a return type missing from the cache panicked
	info := &SchemaInfo{cachedTypes: map[uint32]Type{}}
	f := &Function{Name: "fn", ReturnTypeId: 424242, Volatility: "s"}
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("BuildExecute panicked: %v", r)
			}
		}()
		_, _, err := CommonBuilder{}.BuildExecute("fn", f, Record{}, &QueryParts{}, &QueryOptions{}, info)
		if err == nil {
			t.Errorf("BuildExecute: want an error for an unknown return type")
		}
	}()
}

// The total of a ranged count on a function that must run once is counted
// inside the page, with the page's markers; a stable function's total is a
// statement of its own, with its own.
func TestRangedCountOnFunctionBindsEmbedValues(t *testing.T) {
	info := countInfo()
	info.cachedTypes = map[uint32]Type{100: {Id: 100, Name: "projects", IsTable: true}}
	query := "?select=id,tasks(id),clients!inner(id)&tasks.id=eq.7&clients.id=eq.1&id=gt.0&limit=1"
	for _, volatility := range []string{"v", "s"} {
		f := &Function{Name: "fn", ReturnTypeId: 100, ReturnIsSet: true, Volatility: volatility}
		parts := parseFor(t, "fn", query)
		options := &QueryOptions{Count: "exact"}
		sql, values, err := CommonBuilder{}.BuildExecute("fn", f, Record{"a": 1}, parts, options, info)
		if err != nil {
			t.Fatalf("%s: build error: %v", volatility, err)
		}
		checkMarkers(t, volatility+" page", sql, values)
		if volatility == "v" {
			if !strings.Contains(sql, `set_config('smoothdb.total'`) || !strings.Contains(sql, `"clients_1"."id" = $4`) {
				t.Errorf("volatile: the total is not counted in the page: %s", sql)
			}
			continue
		}
		checkMarkers(t, volatility+" total", options.totalQuery, options.totalValues)
		want := `SELECT COUNT(*) FROM "fn"("a" := $1) t WHERE "t"."id" > $2 AND EXISTS ( SELECT 1 FROM "clients" AS "clients_1" WHERE "clients_1"."id" = t."client_id" AND "clients_1"."id" = $3)`
		if options.totalQuery != want {
			t.Errorf("stable: total\n want: %s\n  got: %s", want, options.totalQuery)
		}
		if !reflect.DeepEqual(options.totalValues, []any{1, "0", "1"}) {
			t.Errorf("stable: total values %#v", options.totalValues)
		}
	}
}
