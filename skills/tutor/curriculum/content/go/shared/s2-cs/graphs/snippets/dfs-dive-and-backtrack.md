In Go:

An adjacency list is a map from node to neighbor slice, wrapped in a struct;
the queue is the slice-based queue from the stacks & queues lesson, and the
visited set is a `map[int]bool`:

```go
type Graph struct {
	adj map[int][]int
}

func (g *Graph) AddEdge(from, to int) {
	g.adj[from] = append(g.adj[from], to)
}

queue := []int{start}
node := queue[0]      // peek the front
queue = queue[1:]     // dequeue
```

Yes, `queue = queue[1:]` is the front-advance move the stacks & queues lesson
warned about — it is fine here because the queue lives only for one traversal
and is garbage the moment BFS returns; a long-lived queue needs the care from
that lesson.

Note what stays deterministic: iterating a Go map is randomized, but no
function in the exercise iterates the whole map — traversals only ever walk
one node's neighbor *slice*, which preserves the order edges were added. That
is why the tests can assert exact BFS and DFS orders.
