package database

import (
	"net/url"
	"strings"
	"testing"
)

// TestRangedCountTotal checks the count of a ranged count=exact request, the
// statement sent after the page: PostgREST's readPlanToCountQuery, the root
// rows with their filters and one EXISTS per !inner embed of the root, no
// GROUP BY.
func TestRangedCountTotal(t *testing.T) {
	info := &SchemaInfo{
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
	tests := []struct {
		name  string
		table string
		query string
		total string
	}{
		{
			name:  "a filtered !inner embed restricts the total",
			table: "projects",
			query: "?select=id,clients!inner(id)&clients.id=eq.1&limit=1",
			total: `SELECT COUNT(*) FROM "projects" WHERE EXISTS ( SELECT 1 FROM "clients" AS "clients_1" WHERE "clients_1"."id" = "projects"."client_id" AND "clients_1"."id" = '1')`,
		},
		{
			name:  "the root filters and the !inner embed are both applied",
			table: "projects",
			query: "?select=id,clients!inner(id)&or=(id.eq.2,id.eq.5)&offset=1",
			total: `SELECT COUNT(*) FROM "projects" WHERE ("projects"."id" = $1 OR "projects"."id" = $2) AND EXISTS ( SELECT 1 FROM "clients" AS "clients_1" WHERE "clients_1"."id" = "projects"."client_id")`,
		},
		{
			name:  "a to-many !inner embed is an EXISTS on its rows",
			table: "clients",
			query: "?select=id,projects!inner(id)&projects.id=gt.3&limit=1",
			total: `SELECT COUNT(*) FROM "clients" WHERE EXISTS ( SELECT 1 FROM "projects" AS "projects_1" WHERE "projects_1"."client_id" = "clients"."id" AND "projects_1"."id" > '3')`,
		},
		{
			name:  "a nested !inner embed restricts through its parent",
			table: "tasks",
			query: "?select=id,projects!inner(id,clients!inner(id))&projects.clients.id=eq.1&limit=1",
			total: `SELECT COUNT(*) FROM "tasks" WHERE EXISTS ( SELECT 1 FROM "projects" AS "projects_1" WHERE "projects_1"."id" = "tasks"."project_id" AND EXISTS ( SELECT 1 FROM "clients" AS "clients_2" WHERE "clients_2"."id" = "projects_1"."client_id" AND "clients_2"."id" = '1'))`,
		},
		{
			name:  "an aggregate in the !inner embed's select is dropped from the EXISTS",
			table: "clients",
			query: "?select=id,projects!inner(count())&limit=1",
			total: `SELECT COUNT(*) FROM "clients" WHERE EXISTS ( SELECT 1 FROM "projects" AS "projects_1" WHERE "projects_1"."client_id" = "clients"."id")`,
		},
		{
			name:  "a nested aggregate is an EXISTS at its own level",
			table: "clients",
			query: "?select=id,projects!inner(id,tasks!inner(count()))&projects.tasks.id=eq.999&limit=1",
			total: `SELECT COUNT(*) FROM "clients" WHERE EXISTS ( SELECT 1 FROM "projects" AS "projects_1" WHERE "projects_1"."client_id" = "clients"."id" AND EXISTS ( SELECT 1 FROM "tasks" AS "tasks_2" WHERE "tasks_2"."project_id" = "projects_1"."id" AND "tasks_2"."id" = '999'))`,
		},
		{
			name:  "a left embed does not restrict the total",
			table: "projects",
			query: "?select=id,clients(id)&clients.id=eq.1&limit=1",
			total: `SELECT COUNT(*) FROM "projects"`,
		},
		{
			name:  "an !inner embed under a left one does not restrict the total",
			table: "tasks",
			query: "?select=id,projects(id,clients!inner(id))&projects.clients.id=eq.1&limit=1",
			total: `SELECT COUNT(*) FROM "tasks"`,
		},
		{
			name:  "a grouped aggregate totals its rows, without GROUP BY",
			table: "tasks",
			query: "?select=project_id,count()&project_id=gt.2&limit=1",
			total: `SELECT COUNT(*) FROM "tasks" WHERE "tasks"."project_id" > $1`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			u, err := url.Parse(test.query)
			if err != nil {
				t.Fatal(err)
			}
			parts, err := PostgRestParser{}.parse(test.table, u.Query())
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			options := &QueryOptions{Count: "exact"}
			if _, _, err := (DirectQueryBuilder{}).BuildSelect(test.table, parts, options, info); err != nil {
				t.Fatalf("build: %v", err)
			}
			if options.totalQuery != test.total {
				t.Errorf("total\n\twant %s\n\tgot  %s", test.total, options.totalQuery)
			}
		})
	}
}

// TestRangedCountOrderIsOutermost checks that the rows of a ranged count
// come out of their ORDER BY: SQL keeps the order of a subquery or a CTE
// through no join, so the statement that returns the page must end with its
// ORDER BY, LIMIT and OFFSET (card 32702: the page used to be a CTE read back
// through LEFT JOIN Data d ON true).
func TestRangedCountOrderIsOutermost(t *testing.T) {
	info := &SchemaInfo{}
	tests := []struct {
		name   string
		rpc    bool
		query  string
		suffix string
	}{
		{"table", false, "?select=id&order=id.desc&limit=2&offset=1", ` ORDER BY "projects"."id" DESC LIMIT $1 OFFSET $2`},
		{"table, offset only", false, "?select=id&order=id.desc&offset=1", ` ORDER BY "projects"."id" DESC OFFSET $1`},
		{"rpc", true, "?select=id&order=id.desc&limit=2&offset=1", ` ORDER BY "t"."id" DESC LIMIT $1 OFFSET $2`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			u, err := url.Parse(test.query)
			if err != nil {
				t.Fatal(err)
			}
			parts, err := PostgRestParser{}.parse("projects", u.Query())
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			options := &QueryOptions{Count: "exact"}
			var query string
			if test.rpc {
				query, _, err = DirectQueryBuilder{}.BuildExecute("getallprojects", nil, Record{}, parts, options, info)
			} else {
				query, _, err = DirectQueryBuilder{}.BuildSelect("projects", parts, options, info)
			}
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			if !strings.HasSuffix(strings.TrimSpace(query), test.suffix) {
				t.Errorf("the page does not end with its order\n\twant suffix %s\n\tgot %s", test.suffix, query)
			}
		})
	}
}

// TestRangedCountFunctionVolatility checks the two shapes of a ranged count
// on a function. A VOLATILE function must run once, so the page reads it
// through a materialized CTE and stores the total (and a function missing
// from the cache is taken for VOLATILE). A STABLE or IMMUTABLE one is
// counted like a table, the call in both statements: the CTE would keep
// PostgreSQL from inlining a SQL function into the page.
func TestRangedCountFunctionVolatility(t *testing.T) {
	info := &SchemaInfo{cachedTypes: map[uint32]Type{
		1: {Id: 1, Name: "projects", Schema: "test", IsTable: true},
	}}
	fn := func(volatility string) *Function {
		return &Function{Name: "getallprojects", Schema: "test", ReturnTypeId: 1, ReturnIsSet: true, Volatility: volatility}
	}
	tests := []struct {
		name string
		f    *Function
		once bool
	}{
		{"volatile", fn("v"), true},
		{"not in the cache", nil, true},
		{"stable", fn("s"), false},
		{"immutable", fn("i"), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			u, _ := url.Parse("?select=id&id=gt.1&limit=2")
			parts, err := PostgRestParser{}.parse("getallprojects", u.Query())
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			options := &QueryOptions{Count: "exact", Schema: "test"}
			query, values, err := DirectQueryBuilder{}.BuildExecute("getallprojects", test.f, Record{}, parts, options, info)
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			if got := strings.Contains(query, "__source"); got != test.once {
				t.Errorf("materialized source: got %v, want %v\n\t%s", got, test.once, query)
			}
			if test.once {
				if !strings.Contains(options.totalQuery, "current_setting(") {
					t.Errorf("total: want the stored total, got %s", options.totalQuery)
				}
				return
			}
			want := `SELECT COUNT(*) FROM "test"."getallprojects"() t WHERE "t"."id" > $1`
			if strings.TrimSpace(options.totalQuery) != want {
				t.Errorf("total\n\twant %s\n\tgot  %s", want, options.totalQuery)
			}
			if len(options.totalValues) != 1 || options.totalValues[0] != "1" || len(values) != 2 {
				t.Errorf("values: total %v, page %v", options.totalValues, values)
			}
		})
	}
}
