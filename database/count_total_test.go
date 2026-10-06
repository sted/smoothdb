package database

import (
	"net/url"
	"strings"
	"testing"
)

// TestRangedCountTotal checks the count of a ranged count=exact request, the
// Total CTE: PostgREST's readPlanToCountQuery, the root rows with their
// filters and one EXISTS per !inner embed of the root, no GROUP BY.
func TestRangedCountTotal(t *testing.T) {
	info := &SchemaInfo{
		cachedRelationships: map[string][]Relationship{
			"projects": {{Type: M2O, Table: "projects", Columns: []string{"client_id"},
				RelatedTable: "clients", RelatedColumns: []string{"id"}}},
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
			total: `SELECT COUNT(*) AS __count FROM "projects" WHERE EXISTS ( SELECT "clients_1"."id" FROM "clients" AS "clients_1" WHERE "clients_1"."id" = "projects"."client_id" AND "clients_1"."id" = '1')`,
		},
		{
			name:  "the root filters and the !inner embed are both applied",
			table: "projects",
			query: "?select=id,clients!inner(id)&or=(id.eq.2,id.eq.5)&offset=1",
			total: `SELECT COUNT(*) AS __count FROM "projects" WHERE ("projects"."id" = $1 OR "projects"."id" = $2) AND EXISTS ( SELECT "clients_1"."id" FROM "clients" AS "clients_1" WHERE "clients_1"."id" = "projects"."client_id")`,
		},
		{
			name:  "a to-many !inner embed is an EXISTS on its rows",
			table: "clients",
			query: "?select=id,projects!inner(id)&projects.id=gt.3&limit=1",
			total: `SELECT COUNT(*) AS __count FROM "clients" WHERE EXISTS ( SELECT "projects_1"."id" FROM "projects" AS "projects_1" WHERE "projects_1"."client_id" = "clients"."id" AND "projects_1"."id" > '3')`,
		},
		{
			name:  "a nested !inner embed restricts through its parent",
			table: "tasks",
			query: "?select=id,projects!inner(id,clients!inner(id))&projects.clients.id=eq.1&limit=1",
			total: `SELECT COUNT(*) AS __count FROM "tasks" WHERE EXISTS ( SELECT "projects_1"."id",  row_to_json("projects_clients_2".*) AS "clients" FROM "projects" AS "projects_1"  INNER JOIN LATERAL ( SELECT "clients_2"."id" FROM "clients" AS "clients_2" WHERE "clients_2"."id" = "projects_1"."client_id" AND "clients_2"."id" = '1') AS "projects_clients_2" ON TRUE WHERE "projects_1"."id" = "tasks"."project_id")`,
		},
		{
			name:  "a left embed does not restrict the total",
			table: "projects",
			query: "?select=id,clients(id)&clients.id=eq.1&limit=1",
			total: `SELECT COUNT(*) AS __count FROM "projects"`,
		},
		{
			name:  "an !inner embed under a left one does not restrict the total",
			table: "tasks",
			query: "?select=id,projects(id,clients!inner(id))&projects.clients.id=eq.1&limit=1",
			total: `SELECT COUNT(*) AS __count FROM "tasks"`,
		},
		{
			name:  "a grouped aggregate totals its rows, without GROUP BY",
			table: "tasks",
			query: "?select=project_id,count()&project_id=gt.2&limit=1",
			total: `SELECT COUNT(*) AS __count FROM "tasks" WHERE "tasks"."project_id" > $1`,
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
			query, _, err := DirectQueryBuilder{}.BuildSelect(test.table, parts, &QueryOptions{Count: "exact"}, info)
			if err != nil {
				t.Fatalf("build: %v", err)
			}
			start := strings.Index(query, "WITH Total AS (")
			end := strings.Index(query, "), Data AS (")
			if start == -1 || end == -1 {
				t.Fatalf("no Total CTE in\n\t%s", query)
			}
			if got := query[start+len("WITH Total AS (") : end]; got != test.total {
				t.Errorf("total\n\twant %s\n\tgot  %s", test.total, got)
			}
		})
	}
}
