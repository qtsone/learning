// Package graphs implements a directed graph as an adjacency list,
// with breadth-first and depth-first traversal.
package graphs

// Graph is a directed graph over int node ids. An undirected edge is
// modeled as two directed edges, one in each direction.
type Graph struct {
	adj map[int][]int
}

// New returns an empty graph.
func New() *Graph {
	return &Graph{adj: make(map[int][]int)}
}

// AddEdge adds the directed edge from → to.
func (g *Graph) AddEdge(from, to int) {
	// TODO: record to in from's adjacency slice.
}

// Neighbors returns the targets of from's outgoing edges, in the order
// the edges were added. A node with no outgoing edges has no neighbors.
func (g *Graph) Neighbors(from int) []int {
	// TODO: implement.
	return nil
}

// BFS returns start plus every node reachable from it, in breadth-first
// order: all nodes one edge away (in neighbor order), then two edges
// away, and so on. Each node appears exactly once, even when the graph
// has cycles.
func (g *Graph) BFS(start int) []int {
	// TODO: implement per LESSON.md — a queue for the frontier, a
	// visited set so cycles terminate; mark nodes when discovered.
	return nil
}

// DFS returns start plus every node reachable from it, in depth-first
// order: follow the first neighbor as deep as possible before
// backtracking, neighbors in the order added. Each node appears exactly
// once, even when the graph has cycles.
func (g *Graph) DFS(start int) []int {
	// TODO: implement — recursion (or an explicit stack) plus a
	// visited set.
	return nil
}

// ShortestPathLen returns the number of edges on the shortest path from
// start to end: 0 when start == end, -1 when end is unreachable.
func (g *Graph) ShortestPathLen(start, end int) int {
	// TODO: implement — BFS, recording each node's distance when it is
	// discovered.
	return 0
}

// Reachable reports whether a directed path leads from from to to.
// Every node can reach itself.
func (g *Graph) Reachable(from, to int) bool {
	// TODO: implement — any traversal answers this.
	return false
}
