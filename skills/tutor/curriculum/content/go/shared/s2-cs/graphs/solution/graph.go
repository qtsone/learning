// Package graphs implements a directed graph as an adjacency list,
// with breadth-first and depth-first traversal.
package graphs

import "slices"

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
	g.adj[from] = append(g.adj[from], to)
}

// Neighbors returns the targets of from's outgoing edges, in the order
// the edges were added. A node with no outgoing edges has no neighbors.
func (g *Graph) Neighbors(from int) []int {
	return g.adj[from]
}

// BFS returns start plus every node reachable from it, in breadth-first
// order: all nodes one edge away (in neighbor order), then two edges
// away, and so on. Each node appears exactly once, even when the graph
// has cycles.
func (g *Graph) BFS(start int) []int {
	visited := map[int]bool{start: true}
	order := []int{start}
	queue := []int{start}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for _, nb := range g.adj[node] {
			if visited[nb] {
				continue
			}
			visited[nb] = true
			order = append(order, nb)
			queue = append(queue, nb)
		}
	}
	return order
}

// DFS returns start plus every node reachable from it, in depth-first
// order: follow the first neighbor as deep as possible before
// backtracking, neighbors in the order added. Each node appears exactly
// once, even when the graph has cycles.
func (g *Graph) DFS(start int) []int {
	visited := make(map[int]bool)
	var order []int
	var walk func(node int)
	walk = func(node int) {
		if visited[node] {
			return
		}
		visited[node] = true
		order = append(order, node)
		for _, nb := range g.adj[node] {
			walk(nb)
		}
	}
	walk(start)
	return order
}

// ShortestPathLen returns the number of edges on the shortest path from
// start to end: 0 when start == end, -1 when end is unreachable.
func (g *Graph) ShortestPathLen(start, end int) int {
	dist := map[int]int{start: 0}
	queue := []int{start}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		if node == end {
			return dist[node]
		}
		for _, nb := range g.adj[node] {
			if _, seen := dist[nb]; !seen {
				dist[nb] = dist[node] + 1
				queue = append(queue, nb)
			}
		}
	}
	return -1
}

// Reachable reports whether a directed path leads from from to to.
// Every node can reach itself.
func (g *Graph) Reachable(from, to int) bool {
	return slices.Contains(g.BFS(from), to)
}
