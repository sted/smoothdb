package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_SelfEmbed covers the filters and the orders on an embed that
// follows a foreign key back to the same table, on a GET. Upstream covers them
// only after a PATCH (UpdateSpec "tables with self reference foreign keys" and
// "with ordering", ported in update_test.go); every case here is added and
// follows PostgREST's read plan, where a filter or an order belongs to the
// embed its path names whatever the embedded table (Plan.hs addFilters,
// addOrders, updateNode).
func TestPostgREST_SelfEmbed(t *testing.T) {

	tests := []test.Test{
		// @@ added
		{
			Description: "a filter on a self-referencing embed filters the children",
			Query:       "/web_content?id=eq.0&select=id,name,web_content(name)&web_content.name=eq.bar",
			Expected:    `[{"id":0,"name":"tardis","web_content":[{"name":"bar"}]}]`,
			Status:      200,
		},
		// @@ added
		{
			Description: "a filter on a self-referencing embed filters the parent",
			Query:       "/family_tree?id=in.(4,5)&select=id,parent(id,name)&parent.name=eq.Kid%20One&order=id",
			Expected:    `[{"id":"4","parent":{"id":"2","name":"Kid One"}},{"id":"5","parent":null}]`,
			Status:      200,
		},
		// @@ added
		{
			Description: "a filter on an inner self-referencing embed constrains the parent row",
			Query:       "/family_tree?select=id,parent!inner(id,name)&parent.name=eq.Kid%20One",
			Expected:    `[{"id":"4","parent":{"id":"2","name":"Kid One"}}]`,
			Status:      200,
		},
		// @@ added
		{
			Description: "a filter on inner self-referencing children constrains the parent row",
			Query:       "/web_content?select=id,web_content!inner(name)&web_content.name=eq.wut",
			Expected:    `[{"id":1,"web_content":[{"name":"wut"}]}]`,
			Status:      200,
		},
		// @@ added
		{
			Description: "a filter on an aliased self-referencing embed",
			Query:       "/web_content?id=eq.0&select=id,children:web_content(name)&children.name=eq.foo",
			Expected:    `[{"id":0,"children":[{"name":"foo"}]}]`,
			Status:      200,
		},
		// @@ added
		{
			Description: "an order on a self-referencing embed orders the children",
			Query:       "/web_content?id=eq.0&select=id,name,web_content(name)&web_content.order=name.asc",
			Expected:    `[{"id":0,"name":"tardis","web_content":[{"name":"bar"},{"name":"fezz"},{"name":"foo"}]}]`,
			Status:      200,
		},
		// @@ added: each order stays on its level
		{
			Description: "the top-level order and the self-referencing embed's order apart",
			Query:       "/web_content?id=in.(0,1)&select=id,web_content(name)&order=id.desc&web_content.order=name.desc",
			Expected:    `[{"id":1,"web_content":[{"name":"wut"}]},{"id":0,"web_content":[{"name":"foo"},{"name":"fezz"},{"name":"bar"}]}]`,
			Status:      200,
		},
		// @@ added
		{
			Description: "an order on an aliased self-referencing embed",
			Query:       "/web_content?id=eq.0&select=id,children:web_content(name)&children.order=name.desc",
			Expected:    `[{"id":0,"children":[{"name":"foo"},{"name":"fezz"},{"name":"bar"}]}]`,
			Status:      200,
		},
		// @@ added
		{
			Description: "an order on the second level of a self-referencing embed",
			Query:       "/web_content?id=eq.5&select=id,web_content(name,web_content(name))&web_content.web_content.order=name.desc",
			Expected:    `[{"id":5,"web_content":[{"name":"tardis","web_content":[{"name":"foo"},{"name":"fezz"},{"name":"bar"}]}]}]`,
			Status:      200,
		},
	}

	test.Execute(t, testConfig, tests)
}
