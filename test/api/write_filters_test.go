package test_api

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

// TestWriteFiltersOnEmbeds: a write without a representation still takes a
// filter on an embed its select names (the filter shapes the embed, not the
// write), and refuses one on an embed it does not name (PGRST108 in
// PostgREST), on PATCH, DELETE and an upsert. A nested and/or inside an embed
// filter belongs to that embed.
func TestWriteFiltersOnEmbeds(t *testing.T) {

	execSQLAndReload(t, `
		drop table if exists wfchild, wfparent;
		create table wfparent (id int4 primary key, n int4);
		create table wfchild (id int4 primary key, parent_id int4 references wfparent (id));`)

	testConfig := test.Config{
		BaseUrl:       "http://localhost:8082/api/dbtest",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
	minimal := test.Headers{"Prefer": {"return=minimal"}}
	test.Execute(t, testConfig, []test.Test{
		{Description: "parents", Method: "POST", Query: "/wfparent", Body: `[{"id": 1, "n": 1}, {"id": 2, "n": 2}]`, Status: 201},
		{Description: "children", Method: "POST", Query: "/wfchild", Body: `[{"id": 1, "parent_id": 1}, {"id": 2, "parent_id": 1}, {"id": 3, "parent_id": 2}]`, Status: 201},
		{
			Description: "PATCH, minimal, filter on a selected embed: the write runs",
			Method:      "PATCH",
			Query:       "/wfparent?id=eq.1&select=*,wfchild(*)&wfchild.id=eq.1",
			Headers:     minimal,
			Body:        `{"n": 10}`,
			Status:      204,
		},
		{
			Description: "the PATCH changed only its row",
			Method:      "GET",
			Query:       "/wfparent?select=id,n&order=id",
			Expected:    `[{"id": 1, "n": 10}, {"id": 2, "n": 2}]`,
			Status:      200,
		},
		{
			Description: "PATCH, minimal, filter on an embed not selected: refused",
			Method:      "PATCH",
			Query:       "/wfparent?id=eq.1&wfchild.id=eq.1",
			Headers:     minimal,
			Body:        `{"n": 0}`,
			Status:      400,
		},
		{
			Description: "upsert, minimal, filter on an embed not selected: refused",
			Method:      "POST",
			Query:       "/wfparent?on_conflict=id&wfchild.id=eq.99",
			Headers:     test.Headers{"Prefer": {"return=minimal,resolution=merge-duplicates"}},
			Body:        `{"id": 2, "n": 0}`,
			Status:      400,
		},
		{
			Description: "the refused writes changed nothing",
			Method:      "GET",
			Query:       "/wfparent?select=id,n&order=id",
			Expected:    `[{"id": 1, "n": 10}, {"id": 2, "n": 2}]`,
			Status:      200,
		},
		{
			Description: "nested and inside an embed filter applies to the embed",
			Method:      "GET",
			Query:       "/wfparent?select=id,wfchild(id)&wfchild.and=(and(id.eq.1))&order=id",
			Expected:    `[{"id": 1, "wfchild": [{"id": 1}]}, {"id": 2, "wfchild": []}]`,
			Status:      200,
		},
		{
			Description: "DELETE, minimal, filter on a selected embed: the write runs",
			Method:      "DELETE",
			Query:       "/wfchild?id=eq.3&select=*,wfparent(*)&wfparent.id=eq.2",
			Headers:     minimal,
			Status:      204,
		},
		{
			Description: "the DELETE removed only its row",
			Method:      "GET",
			Query:       "/wfchild?select=id&order=id",
			Expected:    `[{"id": 1}, {"id": 2}]`,
			Status:      200,
		},
	})
}
