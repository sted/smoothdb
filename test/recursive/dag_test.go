package test_recursive

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/sted/smoothdb/test"
)

// A layered DAG: dagLayers layers of dagPerLayer nodes, every node linked to dagOutDeg
// nodes of the next layer. 13 x 50 = 650 nodes, 12 x 50 x 3 = 1800 edges, depth 12 —
// the graph of card 32699 via-cte. From the root, (3^13-1)/2 = 797,161 simple paths
// lead to just 490 reachable nodes: a via() CTE that enumerates paths materialises
// every one of them, a CTE that enumerates nodes stops at 490 rows.
const (
	dagLayers   = 13
	dagPerLayer = 50
	dagOutDeg   = 3
)

func dagID(layer, i int) int { return layer*dagPerLayer + i + 1 }

// dagEdges lists the edges deterministically: node i of a layer links to nodes
// (3i+j) mod 50, j = 0..2, of the next layer — three distinct targets, and 3 is
// invertible mod 50, so every node of the next layer has in-degree 3 too.
func dagEdges() [][2]int {
	edges := make([][2]int, 0, (dagLayers-1)*dagPerLayer*dagOutDeg)
	for l := 0; l < dagLayers-1; l++ {
		for i := 0; i < dagPerLayer; i++ {
			for j := 0; j < dagOutDeg; j++ {
				edges = append(edges, [2]int{dagID(l, i), dagID(l+1, (i*dagOutDeg+j)%dagPerLayer)})
			}
		}
	}
	return edges
}

// dagBFS computes in Go the depth of every node reachable from root — the reference
// the via() walks are compared against.
func dagBFS(root int) map[int]int {
	next := map[int][]int{}
	for _, e := range dagEdges() {
		next[e[0]] = append(next[e[0]], e[1])
	}
	depth := map[int]int{root: 0}
	for frontier := []int{root}; len(frontier) > 0; {
		var nextFrontier []int
		for _, n := range frontier {
			for _, m := range next[n] {
				if _, seen := depth[m]; !seen {
					depth[m] = depth[n] + 1
					nextFrontier = append(nextFrontier, m)
				}
			}
		}
		frontier = nextFrontier
	}
	return depth
}

// dagDataCommands inserts the nodes and the edges (one bulk POST each).
func dagDataCommands() []test.Command {
	var nodes []string
	for l := 0; l < dagLayers; l++ {
		for i := 0; i < dagPerLayer; i++ {
			id := dagID(l, i)
			nodes = append(nodes, fmt.Sprintf(`{"id": %d, "name": "n%d"}`, id, id))
		}
	}
	var edges []string
	for _, e := range dagEdges() {
		edges = append(edges, fmt.Sprintf(`{"src_id": %d, "dst_id": %d}`, e[0], e[1]))
	}
	return []test.Command{
		{Method: "POST", Query: "/dag_node", Body: "[" + strings.Join(nodes, ", ") + "]"},
		{Method: "POST", Query: "/dag_link", Body: "[" + strings.Join(edges, ", ") + "]"},
	}
}

type dagRow struct {
	ID    int   `json:"id"`
	Depth int   `json:"__depth"`
	Path  []int `json:"__path"`
}

func dagWalk(t *testing.T, query string) []dagRow {
	t.Helper()
	body, _, status, err := test.Exec(test.InitClient(), testConfig(), &test.Command{Query: query})
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	if status != 200 {
		t.Fatalf("%s: status %d, body %s", query, status, body)
	}
	var rows []dagRow
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("%s: %v (body %.200s)", query, err, body)
	}
	return rows
}

// checkDagDepths compares a walk with the reference: the same node set, each node
// at its BFS depth, no node twice.
func checkDagDepths(t *testing.T, query string, rows []dagRow, want map[int]int) {
	t.Helper()
	seen := map[int]bool{}
	for _, r := range rows {
		if seen[r.ID] {
			t.Errorf("%s: node %d returned twice", query, r.ID)
		}
		seen[r.ID] = true
		d, ok := want[r.ID]
		if !ok {
			t.Errorf("%s: node %d is not reachable", query, r.ID)
		} else if d != r.Depth {
			t.Errorf("%s: node %d at depth %d, want %d", query, r.ID, r.Depth, d)
		}
	}
	if len(rows) != len(want) {
		t.Errorf("%s: %d nodes, want %d", query, len(rows), len(want))
	}
}

// TestViaLayeredDag is the same-result-set proof for the via() CTE: on the layered
// DAG the walk must return exactly the nodes a BFS reaches, each at its BFS depth,
// with `after`, with a depth cap, and when the path array is requested (the shape
// that enumerates every simple path — 797,161 of them here — must agree with the
// shape that enumerates nodes).
func TestViaLayeredDag(t *testing.T) {
	want := dagBFS(1)
	if len(want) != 490 {
		t.Fatalf("reference BFS reaches %d nodes, want 490", len(want))
	}
	edge := map[[2]int]bool{}
	for _, e := range dagEdges() {
		edge[e] = true
	}

	q := "/dag_node?id=start.1&id=recurse.all&dag_link=via(src_id,dst_id)&select=id,__depth&order=id"
	checkDagDepths(t, q, dagWalk(t, q), want)

	q = "/dag_node?id=after.1&id=recurse.all&dag_link=via(src_id,dst_id)&select=id,__depth&order=id"
	wantAfter := map[int]int{}
	for id, d := range want {
		if d > 0 {
			wantAfter[id] = d
		}
	}
	checkDagDepths(t, q, dagWalk(t, q), wantAfter)

	q = "/dag_node?id=start.1&id=recurse.4&dag_link=via(src_id,dst_id)&select=id,__depth&order=id"
	wantCapped := map[int]int{}
	for id, d := range want {
		if d <= 4 {
			wantCapped[id] = d
		}
	}
	checkDagDepths(t, q, dagWalk(t, q), wantCapped)

	// The path-carrying shape: same nodes and depths, and every path is a walk from
	// the root to the node along edges, as long as the depth says.
	q = "/dag_node?id=start.1&id=recurse.all&dag_link=via(src_id,dst_id)&select=id,__depth,__path&order=id"
	rows := dagWalk(t, q)
	checkDagDepths(t, q, rows, want)
	for _, r := range rows {
		if len(r.Path) != r.Depth+1 || r.Path[0] != 1 || r.Path[len(r.Path)-1] != r.ID {
			t.Errorf("%s: node %d at depth %d has path %v", q, r.ID, r.Depth, r.Path)
			continue
		}
		for i := 1; i < len(r.Path); i++ {
			if !edge[[2]int{r.Path[i-1], r.Path[i]}] {
				t.Errorf("%s: node %d: %d -> %d in path %v is not an edge", q, r.ID, r.Path[i-1], r.Path[i], r.Path)
			}
		}
	}
}

// BenchmarkViaLayeredDag walks the layered DAG from the root: "nodes" is the
// default via() shape, "paths" the path-carrying one (select=__path), which is the
// shape via() always used before card 32699 via-cte.
func BenchmarkViaLayeredDag(b *testing.B) {
	client := test.InitClient()
	config := testConfig()
	for _, bc := range []struct{ name, query string }{
		{"nodes", "/dag_node?id=start.1&id=recurse.all&dag_link=via(src_id,dst_id)&select=id,__depth"},
		{"paths", "/dag_node?id=start.1&id=recurse.all&dag_link=via(src_id,dst_id)&select=id,__depth,__path"},
	} {
		b.Run(bc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				body, _, status, err := test.Exec(client, config, &test.Command{Query: bc.query})
				if err != nil {
					b.Fatal(err)
				}
				if status != 200 {
					b.Fatalf("status %d, body %.200s", status, body)
				}
			}
		})
	}
}
