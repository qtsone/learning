package graphs

import (
	"slices"
	"testing"
)

func build(edges [][2]int) *Graph {
	g := New()
	for _, e := range edges {
		g.AddEdge(e[0], e[1])
	}
	return g
}

func TestAddEdgeAndNeighbors(t *testing.T) {
	g := build([][2]int{{1, 2}, {1, 3}, {2, 3}})
	if got, want := g.Neighbors(1), []int{2, 3}; !slices.Equal(got, want) {
		t.Errorf("Neighbors(1) = %v, want %v (targets in the order the edges were added)", got, want)
	}
	if got, want := g.Neighbors(2), []int{3}; !slices.Equal(got, want) {
		t.Errorf("Neighbors(2) = %v, want %v (edges are directed: 1→2 gives 2 no neighbor)", got, want)
	}
	if got := g.Neighbors(3); len(got) != 0 {
		t.Errorf("Neighbors(3) = %v, want none (3 has only incoming edges)", got)
	}
	if got := g.Neighbors(99); len(got) != 0 {
		t.Errorf("Neighbors(99) = %v, want none (99 was never added)", got)
	}
}

// Diamond: 4 is reachable via 2 AND via 3. Without visited tracking it
// is discovered twice and appears twice.
func TestBFSVisitsEachNodeOnce(t *testing.T) {
	g := build([][2]int{{1, 2}, {1, 3}, {2, 4}, {3, 4}})
	got := g.BFS(1)
	want := []int{1, 2, 3, 4}
	if !slices.Equal(got, want) {
		t.Errorf("BFS(1) = %v, want %v (4 has two routes in but must appear once — track visited nodes)", got, want)
	}
}

func TestBFSVisitsLevelByLevel(t *testing.T) {
	g := build([][2]int{{1, 2}, {1, 3}, {2, 4}, {3, 5}, {4, 6}})
	got := g.BFS(1)
	want := []int{1, 2, 3, 4, 5, 6}
	if !slices.Equal(got, want) {
		t.Errorf("BFS(1) = %v, want %v (everything at distance 1 before anything at distance 2)", got, want)
	}
}

// WARNING: if this test hangs, your traversal never terminates on a
// cycle — pass TestBFSVisitsEachNodeOnce (visited tracking) first.
func TestBFSHandlesCycle(t *testing.T) {
	g := build([][2]int{{1, 2}, {2, 3}, {3, 1}})
	got := g.BFS(1)
	want := []int{1, 2, 3}
	if !slices.Equal(got, want) {
		t.Errorf("BFS(1) = %v, want %v (the cycle back to 1 must not revisit it)", got, want)
	}
}

func TestBFSStartWithNoEdges(t *testing.T) {
	g := build([][2]int{{1, 2}})
	got := g.BFS(7)
	want := []int{7}
	if !slices.Equal(got, want) {
		t.Errorf("BFS(7) = %v, want %v (a start with no outgoing edges is just itself)", got, want)
	}
}

func TestBFSStaysInItsComponent(t *testing.T) {
	g := build([][2]int{{1, 2}, {3, 4}})
	got := g.BFS(1)
	want := []int{1, 2}
	if !slices.Equal(got, want) {
		t.Errorf("BFS(1) = %v, want %v (3 and 4 are a separate island — unreachable from 1)", got, want)
	}
}

func TestDFSGoesDeepFirst(t *testing.T) {
	g := build([][2]int{{1, 2}, {1, 3}, {2, 4}, {3, 5}, {4, 6}})
	got := g.DFS(1)
	want := []int{1, 2, 4, 6, 3, 5}
	if !slices.Equal(got, want) {
		t.Errorf("DFS(1) = %v, want %v (follow the first neighbor all the way down, then backtrack)", got, want)
	}
}

func TestDFSHandlesCycle(t *testing.T) {
	g := build([][2]int{{1, 2}, {2, 3}, {3, 1}})
	got := g.DFS(1)
	want := []int{1, 2, 3}
	if !slices.Equal(got, want) {
		t.Errorf("DFS(1) = %v, want %v (the cycle back to 1 must not recurse forever)", got, want)
	}
}

func TestShortestPathLen(t *testing.T) {
	// Two routes from 1 to 5: the four-edge 1→2→3→4→5 added first, the
	// two-edge 1→6→5 added second. DFS following first-added edges would
	// report 4; BFS's level order must find 2.
	g := build([][2]int{{1, 2}, {2, 3}, {3, 4}, {4, 5}, {1, 6}, {6, 5}})
	cases := []struct {
		name       string
		start, end int
		want       int
	}{
		{"same node", 1, 1, 0},
		{"direct edge", 1, 2, 1},
		{"short route beats first-added long route", 1, 5, 2},
		{"against edge direction", 2, 1, -1},
		{"unknown target", 1, 42, -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := g.ShortestPathLen(c.start, c.end); got != c.want {
				t.Errorf("ShortestPathLen(%d, %d) = %d, want %d", c.start, c.end, got, c.want)
			}
		})
	}
}

// A 1000-node chain 0→1→…→999 plus a shortcut 0→999 added last: the
// first time BFS reaches a node is via the fewest edges, no matter how
// tempting the long way is.
func TestShortestPathLenPrefersShortcut(t *testing.T) {
	g := New()
	for i := 0; i < 999; i++ {
		g.AddEdge(i, i+1)
	}
	g.AddEdge(0, 999)
	if got := g.ShortestPathLen(0, 999); got != 1 {
		t.Errorf("ShortestPathLen(0, 999) = %d, want 1 (the direct edge, not the 999-edge chain)", got)
	}
	if got := g.ShortestPathLen(0, 500); got != 500 {
		t.Errorf("ShortestPathLen(0, 500) = %d, want 500 (only route: 500 hops along the chain)", got)
	}
}

func TestReachable(t *testing.T) {
	g := build([][2]int{{1, 2}, {2, 3}, {4, 5}})
	cases := []struct {
		name     string
		from, to int
		want     bool
	}{
		{"direct edge", 1, 2, true},
		{"transitive", 1, 3, true},
		{"self", 1, 1, true},
		{"self, node without edges", 9, 9, true},
		{"against edge direction", 3, 1, false},
		{"other component", 1, 5, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := g.Reachable(c.from, c.to); got != c.want {
				t.Errorf("Reachable(%d, %d) = %v, want %v", c.from, c.to, got, c.want)
			}
		})
	}
}

// An undirected graph 1—2—3 is both directions of each edge. The 1⇄2
// pair is itself a two-node cycle: traversal must not bounce forever.
func TestUndirectedIsBothDirections(t *testing.T) {
	g := build([][2]int{{1, 2}, {2, 1}, {2, 3}, {3, 2}})
	if got, want := g.BFS(1), []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("BFS(1) = %v, want %v", got, want)
	}
	if !g.Reachable(3, 1) {
		t.Error("Reachable(3, 1) = false, want true (undirected edges go both ways)")
	}
	if got := g.ShortestPathLen(3, 1); got != 2 {
		t.Errorf("ShortestPathLen(3, 1) = %d, want 2 (3—2—1)", got)
	}
}
