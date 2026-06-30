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
			// Plain filters are RESULT filters: the whole subtree is walked, then the
			// filter drops non-matching rows. Here the only inactive node (6) is a leaf,
			// so the result set matches what pruning would give.
			Description: "recurse with is_active result filter — drops inactive rows",
			Query:       "/tree_node?id=start.1&parent_id=recurse.all&is_active=is.true&select=id,name&order=id",
			Expected:    `[{"id":1,"name":"CEO"},{"id":2,"name":"VP Eng"},{"id":3,"name":"VP Sales"},{"id":4,"name":"Senior Dev"},{"id":5,"name":"Junior Dev"},{"id":7,"name":"Sales Rep"}]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig(), tests)
}

// TestResultVsWalkFilter pins down the difference between a result filter (plain
// key) and a walk-prune filter (walk. prefix) using the isolated subtree where an
// active node (102) hangs beneath an inactive one (101).
func TestResultVsWalkFilter(t *testing.T) {
	tests := []test.Test{
		{
			// Result filter: 100/101/102 are all walked; the filter keeps the active
			// nodes 100 and 102 even though 102's parent (101) is inactive.
			Description: "result filter keeps active node under inactive parent",
			Query:       "/tree_node?id=start.100&parent_id=recurse.all&is_active=is.true&select=id,name&order=id",
			Expected:    `[{"id":100,"name":"Region"},{"id":102,"name":"Remote Worker"}]`,
			Status:      200,
		},
		{
			// Walk-prune filter: traversal stops at the inactive node 101, so 102 is
			// never reached. Only 100 survives.
			Description: "walk-prune filter stops the walk at the inactive node",
			Query:       "/tree_node?id=start.100&parent_id=recurse.all&walk.is_active=is.true&select=id,name&order=id",
			Expected:    `[{"id":100,"name":"Region"}]`,
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

// TestDepthColumn checks that __depth is a selectable pseudo-column reporting the
// traversal depth of each node (seed = 0).
func TestDepthColumn(t *testing.T) {
	tests := []test.Test{
		{
			Description: "select __depth on a single-table walk",
			Query:       "/tree_node?id=start.1&parent_id=recurse.all&select=id,__depth&order=id",
			Expected:    `[{"id":1,"__depth":0},{"id":2,"__depth":1},{"id":3,"__depth":1},{"id":4,"__depth":2},{"id":5,"__depth":3},{"id":6,"__depth":2},{"id":7,"__depth":2}]`,
			Status:      200,
		},
		{
			Description: "__path is internal and cannot be selected",
			Query:       "/tree_node?id=start.1&parent_id=recurse.all&select=id,__path",
			Status:      400,
		},
	}
	test.Execute(t, testConfig(), tests)
}

// TestViaMinDepth checks that in via mode __depth reports the SHALLOWEST depth at
// which a node is reachable. Doc 4 is reachable directly (1->4, depth 1) and via a
// block (1->2->4, depth 2); the min-depth dedup must report 1.
func TestViaMinDepth(t *testing.T) {
	tests := []test.Test{
		{
			Description: "via __depth reports min depth for multi-path nodes",
			Query:       "/doc?id=start.1&id=recurse.all&doc_rel=via(src_id,dst_id)&select=id,__depth&order=id",
			Expected:    `[{"id":1,"__depth":0},{"id":2,"__depth":1},{"id":3,"__depth":1},{"id":4,"__depth":1},{"id":5,"__depth":2},{"id":6,"__depth":2}]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig(), tests)
}

// TestViaBidirectional checks via!both traversal. Starting from the leaf CTA Text(6)
// and following edges in EITHER direction reaches its ancestors and their subtree,
// without looping (the __path cycle guard blocks back-edges).
func TestViaBidirectional(t *testing.T) {
	tests := []test.Test{
		{
			Description: "via!both from a leaf reaches ancestors and the rest of the graph",
			Query:       "/doc?id=after.6&id=recurse.all&doc_rel=via!both(src_id,dst_id)&select=id,name&order=id",
			Expected:    `[{"id":1,"name":"Composite"},{"id":2,"name":"Block Hero"},{"id":3,"name":"Block CTA"},{"id":4,"name":"Hero Title"},{"id":5,"name":"Hero Text"}]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig(), tests)
}

// TestRecursionEmbed checks that embedding a related resource composes with
// single-table recursion: each walked node carries its embedded node_tag rows.
func TestRecursionEmbed(t *testing.T) {
	tests := []test.Test{
		{
			Description: "embed node_tag while recursing the tree",
			Query:       "/tree_node?id=start.2&parent_id=recurse.1&select=id,name,node_tag(tag)&order=id",
			Expected:    `[{"id":2,"name":"VP Eng","node_tag":[{"tag":"eng"}]},{"id":4,"name":"Senior Dev","node_tag":[]},{"id":6,"name":"Inactive Dev","node_tag":[]}]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig(), tests)
}

// TestRecursionComputedEmbed checks that a COMPUTED-relationship embed (a function
// taking the parent row, like CollHub's labels_objects) composes with single-table
// recursion. Regression guard for the `__row` rewrite: without it the function's
// row argument pointed at the base-table alias, which doesn't exist in the recursive
// outer query, failing with `column "tree_node" does not exist`.
func TestRecursionComputedEmbed(t *testing.T) {
	tests := []test.Test{
		{
			Description: "embed computed relationship node_tags while recursing the tree",
			Query:       "/tree_node?id=start.2&parent_id=recurse.1&select=id,name,node_tags(tag)&order=id",
			Expected:    `[{"id":2,"name":"VP Eng","node_tags":[{"tag":"eng"}]},{"id":4,"name":"Senior Dev","node_tags":[]},{"id":6,"name":"Inactive Dev","node_tags":[]}]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig(), tests)
}

// TestViaEmbedRejected checks that embedding combined with via() recursion is
// rejected (it would fight the min-depth dedup), rather than emitting wrong SQL.
func TestViaEmbedRejected(t *testing.T) {
	tests := []test.Test{
		{
			// Assert the message so this stays a guard test: the embed must resolve
			// (relationship found) and then be rejected by the guard, not 400 for an
			// unknown relationship.
			Description: "via() recursion with an embed — rejected",
			Query:       "/doc?id=after.1&id=recurse.all&doc_rel=via(src_id,dst_id)&select=id,doc_rel!src_id(rel_type)",
			Expected:    `{"code":"","details":null,"hint":"","message":"embedding is not supported with via() recursion","position":0,"subsystem":"network"}`,
			Status:      400,
		},
	}
	test.Execute(t, testConfig(), tests)
}

// TestRecurseUp checks recurse!up — the single-table walk reversed to follow the FK
// toward ANCESTORS. Junior Dev(5) -> Senior Dev(4) -> VP Eng(2) -> CEO(1).
func TestRecurseUp(t *testing.T) {
	tests := []test.Test{
		{
			Description: "recurse!up.all from a leaf — the full ancestor chain to the root",
			Query:       "/tree_node?id=start.5&parent_id=recurse!up.all&select=id,name&order=__depth",
			Expected:    `[{"id":5,"name":"Junior Dev"},{"id":4,"name":"Senior Dev"},{"id":2,"name":"VP Eng"},{"id":1,"name":"CEO"}]`,
			Status:      200,
		},
		{
			Description: "recurse!up depth cap — only two levels up",
			Query:       "/tree_node?id=start.5&parent_id=recurse!up.2&select=id,name&order=__depth",
			Expected:    `[{"id":5,"name":"Junior Dev"},{"id":4,"name":"Senior Dev"},{"id":2,"name":"VP Eng"}]`,
			Status:      200,
		},
		{
			Description: "recurse!up from the root — just the root itself",
			Query:       "/tree_node?id=start.1&parent_id=recurse!up.all&select=id,name",
			Expected:    `[{"id":1,"name":"CEO"}]`,
			Status:      200,
		},
		{
			Description: "recurse!up with after — excludes the seed, returns its ancestors",
			Query:       "/tree_node?id=after.5&parent_id=recurse!up.all&select=id,name&order=__depth",
			Expected:    `[{"id":4,"name":"Senior Dev"},{"id":2,"name":"VP Eng"},{"id":1,"name":"CEO"}]`,
			Status:      200,
		},
		{
			// Result filter keeps only active ancestors; the walk still passes THROUGH the
			// inactive 101 to reach 100. Mirrors the down-walk result-filter test.
			Description: "recurse!up with is_active result filter keeps active ancestors past an inactive one",
			Query:       "/tree_node?id=start.102&parent_id=recurse!up.all&is_active=is.true&select=id,name&order=__depth",
			Expected:    `[{"id":102,"name":"Remote Worker"},{"id":100,"name":"Region"}]`,
			Status:      200,
		},
		{
			Description: "recurse!up __depth reports distance to each ancestor",
			Query:       "/tree_node?id=start.5&parent_id=recurse!up.all&select=id,__depth&order=__depth",
			Expected:    `[{"id":5,"__depth":0},{"id":4,"__depth":1},{"id":2,"__depth":2},{"id":1,"__depth":3}]`,
			Status:      200,
		},
		{
			Description: "'!up' on start is rejected (only valid on recurse)",
			Query:       "/tree_node?id=start!up.5&parent_id=recurse.all&select=id",
			Expected:    `{"code":"","details":null,"hint":"","message":"'!up' is only valid on 'recurse'","position":0,"subsystem":"network"}`,
			Status:      400,
		},
		{
			Description: "recurse!up is rejected with via (reverse a via walk by swapping its columns)",
			Query:       "/doc?id=start.1&id=recurse!up.all&doc_rel=via(src_id,dst_id)&select=id",
			Expected:    `{"code":"","details":null,"hint":"","message":"'recurse!up' is single-table only; reverse a 'via' walk by swapping its columns","position":0,"subsystem":"network"}`,
			Status:      400,
		},
	}
	test.Execute(t, testConfig(), tests)
}

// TestViaOrderByDepthUnselected checks that ordering a via() walk by __depth WITHOUT
// selecting it works (surfacing __depth in the projection) instead of 500ing with
// "ORDER BY expressions must appear in select list" from the min-depth SELECT DISTINCT.
func TestViaOrderByDepthUnselected(t *testing.T) {
	tests := []test.Test{
		{
			Description: "via order=__depth without selecting __depth — surfaces __depth, no 500",
			Query:       "/doc?id=start.1&id=recurse.all&doc_rel=via(src_id,dst_id)&select=id&order=__depth,id",
			Expected:    `[{"id":1,"__depth":0},{"id":2,"__depth":1},{"id":3,"__depth":1},{"id":4,"__depth":1},{"id":5,"__depth":2},{"id":6,"__depth":2}]`,
			Status:      200,
		},
	}
	test.Execute(t, testConfig(), tests)
}
