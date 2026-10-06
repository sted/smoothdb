package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_RangedCount covers the total of Prefer: count=exact when the
// request takes a range (limit, offset or a Range header). Without one the
// total is the number of rows read, and the ported specs (EmbedInnerJoinSpec,
// ComputedRelsSpec, RpcSpec) only ask for counts that way; none asks for a
// ranged count with an !inner embed or an aggregate, so these cases are ours.
//
// The expected values come from PostgREST 14.15 run against these fixtures.
// Main 2697ea0 builds the same total: readPlanToCountQuery (QueryBuilder.hs)
// counts the base rows with their filters and one EXISTS per !inner embed,
// nested ones included, ignores left embeds, and has no GROUP BY.
//
// PostgREST answers 206 to a limit or an offset that leaves rows out of the
// total; smoothdb answers 206 only to a Range header, so those cases leave
// the status unchecked.
func TestPostgREST_RangedCount(t *testing.T) {

	tests := []test.Test{
		// many-to-one !inner, filtered on the embed
		{
			Description:     "ranged count of a filtered many-to-one !inner embed, limit",
			Query:           "/projects?select=id,clients!inner(id)&clients.id=eq.1&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":1,"clients":{"id":1}}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/2"},
		},
		{
			Description:     "ranged count of a filtered many-to-one !inner embed, offset",
			Query:           "/projects?select=id,clients!inner(id)&clients.id=eq.1&order=id&offset=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":2,"clients":{"id":1}}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "1-1/2"},
		},
		{
			Description:     "ranged count of a filtered many-to-one !inner embed, Range header",
			Query:           "/projects?select=id,clients!inner(id)&clients.id=eq.1&order=id",
			Headers:         test.Headers{"Range": {"0-0"}, "Prefer": {"count=exact"}},
			Expected:        `[{"id":1,"clients":{"id":1}}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/2"},
			Status:          206,
		},
		{
			Description:     "ranged count of a filtered many-to-one !inner embed, HEAD",
			Method:          "HEAD",
			Query:           "/projects?select=id,clients!inner(id)&clients.id=eq.1&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/2"},
		},
		// an unfiltered !inner embed drops the rows without a client
		{
			Description:     "ranged count of an unfiltered many-to-one !inner embed",
			Query:           "/projects?select=id,clients!inner(id)&order=id&limit=2",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":1,"clients":{"id":1}},{"id":2,"clients":{"id":1}}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-1/4"},
		},
		// the root filters and the embed both restrict the total
		{
			Description:     "ranged count of an !inner embed with an or= root filter",
			Query:           "/projects?select=id,clients!inner(id)&clients.id=eq.1&or=(id.eq.2,id.eq.5)&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":2,"clients":{"id":1}}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
			Status:          200,
		},
		// nested !inner embeds
		{
			Description:     "ranged count of nested !inner embeds, limit",
			Query:           "/tasks?select=id,projects!inner(id,clients!inner(id))&projects.clients.id=eq.1&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":1,"projects":{"id":1,"clients":{"id":1}}}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/4"},
		},
		{
			Description:     "ranged count of nested !inner embeds, offset",
			Query:           "/tasks?select=id,projects!inner(id,clients!inner(id))&projects.clients.id=eq.1&order=id&offset=3",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":4,"projects":{"id":2,"clients":{"id":1}}}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "3-3/4"},
		},
		{
			Description:     "ranged count of nested !inner embeds, HEAD",
			Method:          "HEAD",
			Query:           "/tasks?select=id,projects!inner(id,clients!inner(id))&projects.clients.id=eq.1&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/4"},
		},
		// an !inner embed under a left one does not restrict the root
		{
			Description:     "ranged count ignores an !inner embed nested in a left one",
			Query:           "/tasks?select=id,projects(id,clients!inner(id))&projects.clients.id=eq.2&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":1,"projects":null}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/8"},
		},
		// one-to-many and many-to-many !inner
		{
			Description:     "ranged count of a filtered one-to-many !inner embed",
			Query:           "/client?select=id,contact!inner(name)&contact.name=eq.Wally%20Walton&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":1,"contact":[{"name":"Wally Walton"}]}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
			Status:          200,
		},
		{
			Description:     "ranged count of a filtered many-to-many !inner embed",
			Query:           "/products?select=id,suppliers!inner(id)&suppliers.id=eq.2&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":1,"suppliers":[{"id":2}]}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
			Status:          200,
		},
		// computed relationships, the ranged form of ComputedRelsSpec's
		// "works with !inner and count=exact"
		{
			Description:     "ranged count of a filtered computed to-many !inner embed",
			Query:           "/designers?select=name,videogames:computed_videogames!inner(name)&videogames.name=eq.Civilization%20I&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"name":"Sid Meier","videogames":[{"name":"Civilization I"}]}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
			Status:          200,
		},
		{
			Description:     "ranged count of a filtered computed to-one !inner embed, limit",
			Query:           "/videogames?select=name,designer:computed_designers!inner(name)&designer.name=like.*Hironobu*&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"name":"Final Fantasy I","designer":{"name":"Hironobu Sakaguchi"}}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/2"},
		},
		{
			Description:     "ranged count of a filtered computed to-one !inner embed, offset",
			Query:           "/videogames?select=name,designer:computed_designers!inner(name)&designer.name=like.*Hironobu*&order=id&offset=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"name":"Final Fantasy II","designer":{"name":"Hironobu Sakaguchi"}}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "1-1/2"},
		},
		{
			Description:     "ranged count of a filtered computed to-one !inner embed, HEAD",
			Method:          "HEAD",
			Query:           "/videogames?select=name,designer:computed_designers!inner(name)&designer.name=like.*Hironobu*&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/2"},
		},
		// the ranged form of EmbedInnerJoinSpec's "works alongside another
		// embedding" (#2342): book 10 has no publisher
		{
			Description:     "ranged count of an !inner embed after a left one, limit",
			Query:           "/books?select=id,authors(name),publishers!inner(name)&id=gte.7&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":7,"authors":{"name":"Harper Lee"},"publishers":{"name":"J. B. Lippincott & Co."}}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/3"},
		},
		{
			Description:     "ranged count of an !inner embed before a left one, offset",
			Query:           "/books?select=id,publishers!inner(name),authors(name)&id=gte.7&order=id&offset=2",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":9,"publishers":{"name":"Viking Press & Signet Books"},"authors":{"name":"Ken Kesey"}}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "2-2/3"},
		},
		{
			Description:     "ranged count of an !inner embed after a left one, HEAD",
			Method:          "HEAD",
			Query:           "/books?select=id,authors(name),publishers!inner(name)&id=gte.7&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/3"},
		},
		// rpc, the ranged form of RpcSpec's "includes exact count if requested"
		{
			Description:     "ranged count of an rpc with a filtered !inner embed, limit",
			Query:           "/rpc/getallprojects?select=id,clients!inner(id)&clients.id=eq.1&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":1,"clients":{"id":1}}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/2"},
		},
		{
			Description:     "ranged count of an rpc with a filtered !inner embed, offset",
			Query:           "/rpc/getallprojects?select=id,clients!inner(id)&clients.id=eq.1&order=id&offset=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":2,"clients":{"id":1}}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "1-1/2"},
		},
		{
			Description:     "ranged count of an rpc with a filtered !inner embed, HEAD",
			Method:          "HEAD",
			Query:           "/rpc/getallprojects?select=id,clients!inner(id)&clients.id=eq.1&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        ``,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/2"},
		},
		// An !inner embed whose select is an aggregate: the total counts the
		// root rows with a row in the embed (readPlanToCountQuery's SELECT 1),
		// not an EXISTS over the aggregate, which always has its one row.
		// Values from PostgREST 14.15 on the fixtures; like smoothdb's, its
		// rows keep project 5 with a count of 0.
		{
			Description:     "ranged count of an !inner embed selecting an aggregate",
			Query:           "/projects?select=id,tasks!inner(count())&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":1,"tasks":[{"count":2}]}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/4"},
		},
		{
			Description:     "ranged count of a filtered !inner embed selecting an aggregate",
			Query:           "/clients?select=id,projects!inner(count())&projects.name=eq.Windows%207&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":1,"projects":[{"count":1}]}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
		},
		// A nested !inner embed restricts through an EXISTS at its own level,
		// whatever its select: readPlanToCountQuery is recursive, SELECT 1 and
		// one EXISTS per !inner child (PostgREST 14.15 on the fixtures)
		{
			Description:     "ranged count of a nested !inner aggregate that matches nothing",
			Query:           "/clients?select=id,projects!inner(id,tasks!inner(count()))&projects.tasks.id=eq.999&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			ExpectedHeaders: map[string]string{"Content-Range": "*/0"},
		},
		{
			Description:     "ranged count of a nested !inner aggregate",
			Query:           "/clients?select=id,projects!inner(id,tasks!inner(count()))&projects.tasks.name=eq.Design%20w7&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
		},
		{
			Description:     "ranged count of a nested filtered !inner embed",
			Query:           "/clients?select=id,projects!inner(id,tasks!inner(id))&projects.tasks.name=eq.Design%20w7&order=id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"id":1,"projects":[{"id":1,"tasks":[{"id":1}]}]}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/1"},
		},
		// A grouped aggregate counts the rows it groups, not the groups: the
		// total of a ranged request has no GROUP BY in PostgREST (4 groups
		// of 8 tasks here).
		{
			Description:     "ranged count of a grouped aggregate counts the rows, limit",
			Query:           "/tasks?select=project_id,count()&order=project_id&limit=1",
			Headers:         test.Headers{"Prefer": {"count=exact"}},
			Expected:        `[{"project_id":1,"count":2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/8"},
		},
		{
			Description:     "ranged count of a grouped aggregate counts the rows, Range header",
			Query:           "/tasks?select=project_id,count()&order=project_id",
			Headers:         test.Headers{"Range": {"0-0"}, "Prefer": {"count=exact"}},
			Expected:        `[{"project_id":1,"count":2}]`,
			ExpectedHeaders: map[string]string{"Content-Range": "0-0/8"},
			Status:          206,
		},
	}

	test.Execute(t, testConfig, tests)
}
