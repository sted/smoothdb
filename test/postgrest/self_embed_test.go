package postgrest

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestPostgREST_SelfEmbed covers the filters on an embed that follows a
// foreign key back to the same table, on a GET. Upstream covers them only
// after a PATCH (UpdateSpec "tables with self reference foreign keys", ported
// in update_test.go); every case here is added and follows PostgREST's read
// plan, where a filter belongs to the embed its path names whatever the
// embedded table (Plan.hs addFilters/updateNode).
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
	}

	test.Execute(t, testConfig, tests)
}
