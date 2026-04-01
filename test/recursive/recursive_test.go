package test_recursive

import (
	"testing"

	"github.com/sted/smoothdb/test"
)

func testConfig() test.Config {
	return test.Config{
		BaseUrl:       "http://localhost:8083/api/recursive_test",
		CommonHeaders: test.Headers{"Authorization": {adminToken}},
	}
}

func TestBasicRecursion(t *testing.T) {
	tests := []test.Test{
		{
			Description: "recurse from CEO depth 1 — root + direct reports",
			Query:       "/tree_node?id=start.1&parent_id=recurse.1&select=id,name&order=id",
			Expected:    `[{"id":1,"name":"CEO"},{"id":2,"name":"VP Eng"},{"id":3,"name":"VP Sales"}]`,
			Status:      200,
		},
		{
			Description: "recurse from CEO depth 2 — root + two levels",
			Query:       "/tree_node?id=start.1&parent_id=recurse.2&select=id,name&order=id",
			Expected:    `[{"id":1,"name":"CEO"},{"id":2,"name":"VP Eng"},{"id":3,"name":"VP Sales"},{"id":4,"name":"Senior Dev"},{"id":6,"name":"Inactive Dev"},{"id":7,"name":"Sales Rep"}]`,
			Status:      200,
		},
		{
			Description: "recurse from VP Eng depth 1",
			Query:       "/tree_node?id=start.2&parent_id=recurse.1&select=id,name&order=id",
			Expected:    `[{"id":2,"name":"VP Eng"},{"id":4,"name":"Senior Dev"},{"id":6,"name":"Inactive Dev"}]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig(), tests)
}

func TestFullRecursion(t *testing.T) {
	tests := []test.Test{
		{
			Description: "recurse.all from CEO — root + entire subtree",
			Query:       "/tree_node?id=start.1&parent_id=recurse.all&select=id,name&order=id",
			Expected:    `[{"id":1,"name":"CEO"},{"id":2,"name":"VP Eng"},{"id":3,"name":"VP Sales"},{"id":4,"name":"Senior Dev"},{"id":5,"name":"Junior Dev"},{"id":6,"name":"Inactive Dev"},{"id":7,"name":"Sales Rep"}]`,
			Status:      200,
		},
		{
			Description: "recurse.all from leaf node — just the leaf itself",
			Query:       "/tree_node?id=start.5&parent_id=recurse.all&select=id,name",
			Expected:    `[{"id":5,"name":"Junior Dev"}]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig(), tests)
}

func TestRecursionWithFilters(t *testing.T) {
	tests := []test.Test{
		{
			Description: "recurse with is_active filter — prunes inactive branches",
			Query:       "/tree_node?id=start.1&parent_id=recurse.all&is_active=is.true&select=id,name&order=id",
			Expected:    `[{"id":1,"name":"CEO"},{"id":2,"name":"VP Eng"},{"id":3,"name":"VP Sales"},{"id":4,"name":"Senior Dev"},{"id":5,"name":"Junior Dev"},{"id":7,"name":"Sales Rep"}]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig(), tests)
}

func TestRecursionWithSelectAndOrder(t *testing.T) {
	tests := []test.Test{
		{
			Description: "select specific columns with order desc",
			Query:       "/tree_node?id=start.1&parent_id=recurse.all&select=name&order=name.desc",
			Expected:    `[{"name":"VP Sales"},{"name":"VP Eng"},{"name":"Senior Dev"},{"name":"Sales Rep"},{"name":"Junior Dev"},{"name":"Inactive Dev"},{"name":"CEO"}]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig(), tests)
}

func TestRecursionWithLimitOffset(t *testing.T) {
	tests := []test.Test{
		{
			Description: "recurse with limit",
			Query:       "/tree_node?id=start.1&parent_id=recurse.all&select=id,name&order=id&limit=3",
			Expected:    `[{"id":1,"name":"CEO"},{"id":2,"name":"VP Eng"},{"id":3,"name":"VP Sales"}]`,
			Status:      200,
		},
		{
			Description: "recurse with limit and offset",
			Query:       "/tree_node?id=start.1&parent_id=recurse.all&select=id,name&order=id&limit=2&offset=2",
			Expected:    `[{"id":3,"name":"VP Sales"},{"id":4,"name":"Senior Dev"}]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig(), tests)
}

func TestRecursionNonExistentStart(t *testing.T) {
	tests := []test.Test{
		{
			Description: "start from non-existent id — empty result",
			Query:       "/tree_node?id=start.999&parent_id=recurse.all&select=id,name",
			Expected:    `[]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig(), tests)
}

func TestRecursionErrors(t *testing.T) {
	tests := []test.Test{
		{
			Description: "start without recurse — error",
			Query:       "/tree_node?id=start.1",
			Status:      400,
		},
		{
			Description: "recurse without start — error",
			Query:       "/tree_node?parent_id=recurse.3",
			Status:      400,
		},
		{
			Description: "invalid recurse depth — error",
			Query:       "/tree_node?id=start.1&parent_id=recurse.abc",
			Status:      400,
		},
		{
			Description: "recurse depth zero — error",
			Query:       "/tree_node?id=start.1&parent_id=recurse.0",
			Status:      400,
		},
	}
	test.Execute(t, testConfig(), tests)
}

func TestAfterOperator(t *testing.T) {
	tests := []test.Test{
		{
			Description: "after excludes the seed row",
			Query:       "/tree_node?id=after.1&parent_id=recurse.all&select=id,name&order=id",
			Expected:    `[{"id":2,"name":"VP Eng"},{"id":3,"name":"VP Sales"},{"id":4,"name":"Senior Dev"},{"id":5,"name":"Junior Dev"},{"id":6,"name":"Inactive Dev"},{"id":7,"name":"Sales Rep"}]`,
			Status:      200,
		},
		{
			Description: "after depth 1 — only direct reports, no root",
			Query:       "/tree_node?id=after.1&parent_id=recurse.1&select=id,name&order=id",
			Expected:    `[{"id":2,"name":"VP Eng"},{"id":3,"name":"VP Sales"}]`,
			Status:      200,
		},
		{
			Description: "after from leaf — empty result",
			Query:       "/tree_node?id=after.5&parent_id=recurse.all&select=id,name",
			Expected:    `[]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig(), tests)
}

// TestVia tests multi-table recursive queries using the via() operator.
// Data: Composite(1) --contains--> Block Hero(2), Block CTA(3)
//       Composite(1) --references--> Hero Title(4)
//       Block Hero(2) --contains--> Hero Title(4), Hero Text(5)
//       Block CTA(3) --contains--> CTA Text(6)
func TestVia(t *testing.T) {
	tests := []test.Test{
		{
			Description: "via with after — all descendants, excluding root (DISTINCT deduplicates multi-path nodes)",
			Query:       "/doc?id=after.1&id=recurse.all&doc_rel=via(src_id,dst_id)&select=id,name&order=id",
			Expected:    `[{"id":2,"name":"Block Hero"},{"id":3,"name":"Block CTA"},{"id":4,"name":"Hero Title"},{"id":5,"name":"Hero Text"},{"id":6,"name":"CTA Text"}]`,
			Status:   200,
		},
		{
			Description: "via with start — includes root node",
			Query:       "/doc?id=start.1&id=recurse.all&doc_rel=via(src_id,dst_id)&select=id,name&order=id",
			Expected:    `[{"id":1,"name":"Composite"},{"id":2,"name":"Block Hero"},{"id":3,"name":"Block CTA"},{"id":4,"name":"Hero Title"},{"id":5,"name":"Hero Text"},{"id":6,"name":"CTA Text"}]`,
			Status:      200,
		},
		{
			Description: "via — depth 1 only direct relationships",
			Query:       "/doc?id=after.1&id=recurse.1&doc_rel=via(src_id,dst_id)&select=id,name&order=id",
			// Direct edges from doc 1: 2 (contains), 3 (contains), 4 (references)
			Expected: `[{"id":2,"name":"Block Hero"},{"id":3,"name":"Block CTA"},{"id":4,"name":"Hero Title"}]`,
			Status:   200,
		},
		{
			Description: "via — with edge filter, only contains relationships",
			Query:       "/doc?id=after.1&id=recurse.all&doc_rel=via(src_id,dst_id)&doc_rel.rel_type=eq.contains&select=id,name&order=id",
			// Only containment edges: 1→2, 1→3, 2→4, 2→5, 3→6
			Expected: `[{"id":2,"name":"Block Hero"},{"id":3,"name":"Block CTA"},{"id":4,"name":"Hero Title"},{"id":5,"name":"Hero Text"},{"id":6,"name":"CTA Text"}]`,
			Status:   200,
		},
		{
			Description: "via — with edge filter, only references",
			Query:       "/doc?id=after.1&id=recurse.all&doc_rel=via(src_id,dst_id)&doc_rel.rel_type=eq.references&select=id,name&order=id",
			// Only references edge: 1→4, then no further references edges from 4
			Expected: `[{"id":4,"name":"Hero Title"}]`,
			Status:   200,
		},
		{
			Description: "via — main table filter prunes branches",
			Query:       "/doc?id=after.1&id=recurse.all&doc_rel=via(src_id,dst_id)&type_name=eq.Block&select=id,name&order=id",
			// Only Block-typed docs; Fragments pruned so their children unreachable
			Expected: `[{"id":2,"name":"Block Hero"},{"id":3,"name":"Block CTA"}]`,
			Status:   200,
		},
		{
			Description: "via — start from Block Hero includes root",
			Query:       "/doc?id=start.2&id=recurse.all&doc_rel=via(src_id,dst_id)&doc_rel.rel_type=eq.contains&select=id,name&order=id",
			Expected:    `[{"id":2,"name":"Block Hero"},{"id":4,"name":"Hero Title"},{"id":5,"name":"Hero Text"}]`,
			Status:      200,
		},
		{
			Description: "via — from leaf, no outgoing edges",
			Query:       "/doc?id=after.6&id=recurse.all&doc_rel=via(src_id,dst_id)&select=id,name",
			Expected:    `[]`,
			Status:      200,
		},
		{
			Description: "via — edge filter with or (multiple rel types)",
			Query:       "/doc?id=after.1&id=recurse.1&doc_rel=via(src_id,dst_id)&doc_rel.or=(rel_type.eq.contains,rel_type.eq.references)&select=id,name&order=id",
			// Both contains and references edges from doc 1: 2, 3 (contains) + 4 (references)
			Expected: `[{"id":2,"name":"Block Hero"},{"id":3,"name":"Block CTA"},{"id":4,"name":"Hero Title"}]`,
			Status:   200,
		},
		{
			Description: "via — edge filter with in operator",
			Query:       "/doc?id=after.1&id=recurse.1&doc_rel=via(src_id,dst_id)&doc_rel.rel_type=in.(contains,references)&select=id,name&order=id",
			// Same result as or — both rel types
			Expected: `[{"id":2,"name":"Block Hero"},{"id":3,"name":"Block CTA"},{"id":4,"name":"Hero Title"}]`,
			Status:   200,
		},
	}
	test.Execute(t, testConfig(), tests)
}

func TestViaErrors(t *testing.T) {
	tests := []test.Test{
		{
			Description: "via without start/recurse — error",
			Query:       "/doc?doc_rel=via(src_id,dst_id)",
			Status:      400,
		},
		{
			Description: "via with mismatched start/recurse fields — error",
			Query:       "/doc?id=start.1&name=recurse.all&doc_rel=via(src_id,dst_id)",
			Status:      400,
		},
	}
	test.Execute(t, testConfig(), tests)
}
