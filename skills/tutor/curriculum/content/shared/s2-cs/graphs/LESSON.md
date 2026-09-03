# Graphs

> `shared.cs.graphs` · ~3-4h · Stage: CS Fundamentals

## Objectives

By the end of this lesson you can:

- Model a problem as a graph, choosing directed vs undirected and weighted vs
  unweighted representations.
- Compare adjacency-list and adjacency-matrix representations by memory use
  and edge-lookup cost.
- Implement BFS and DFS traversals that pass the provided tests, tracking
  visited nodes to handle cycles.
- Explain why BFS finds shortest paths in unweighted graphs while DFS does
  not.
- Apply graph traversal to a practical problem such as connectivity or
  reachability checking.

## Trees with every restriction removed

A tree bought its clean structure with three rules: one root, one parent per
node, no cycles. A **graph** drops all three. It is just a set of **nodes**
(also called *vertices*) and a set of **edges**, where an edge connects two
nodes. Any node may connect to any number of others, connections may loop
back around, and the whole thing may fall apart into disconnected islands:

```
    (1)───(2)       (6)
     |   /  |
     |  /   |
    (3)    (4)───(5)
```

There is no root and no "up" — `1, 2, 3` even form a cycle — and `6` is not
connected to anything. That freedom is exactly why graphs model so much of
the real world:

- road maps — intersections are nodes, roads are edges;
- social networks — people are nodes, "follows" or "friends" are edges;
- package dependencies — packages are nodes, imports are edges;
- the web — pages are nodes, links are edges.

Two numbers describe a graph's size: **V**, the number of nodes, and **E**,
the number of edges. Every cost in this lesson is stated in terms of them.

## Two modeling decisions

Turning a problem into a graph forces two choices, and the tests of your
model are questions like "does the connection have a direction?" and "do
connections have a cost?".

**Directed or undirected?** An edge is *directed* when it only goes one way:
"page A links to page B" says nothing about B linking back. It is
*undirected* when the relationship is symmetric: a friendship, a two-way
road. The two are not different machinery — an undirected edge is simply
**two directed edges**, one in each direction. Implementations, including
yours in the exercise, usually store directed edges and model undirected
graphs by adding both directions.

**Weighted or unweighted?** An edge is *weighted* when it carries a cost —
kilometers, milliseconds, price. Unweighted means only the connection itself
matters, and "distance" between nodes means *number of edges*. This lesson
is entirely unweighted; shortest paths under weights need a different
algorithm (Dijkstra's is the classic one), which is beyond this lesson —
what matters now is recognizing *when* the unweighted tools stop applying.

Try it on "git commit history": commits are nodes, each commit points to its
parent — directed (history flows one way), unweighted (a link is a link).
On "flight planner minimizing price": airports, directed edges per flight,
weighted by fare.

## Storing a graph: adjacency list vs adjacency matrix

The graph itself is an abstract idea; you have to pick a memory layout. The
two classic ones store the directed graph `1→2, 1→3, 2→4, 3→4` like this:

```
adjacency list                adjacency matrix

1 → [2, 3]                        1  2  3  4
2 → [4]                       1 [ 0  1  1  0 ]
3 → [4]                       2 [ 0  0  0  1 ]
4 → []                        3 [ 0  0  0  1 ]
                              4 [ 0  0  0  0 ]
```

An **adjacency list** keeps, per node, the list of nodes its edges point to
— a map from node to its neighbors. An **adjacency matrix** keeps a V×V grid
where cell `[u][v]` is 1 when the edge `u→v` exists. They trade off exactly
the way arrays and linked structures traded off earlier in this stage:

| | Adjacency list | Adjacency matrix |
|---|---|---|
| Memory | O(V + E) | O(V²) |
| "Is there an edge u→v?" | O(deg(u)) — scan u's list | O(1) — index the grid |
| "All neighbors of u" | O(deg(u)) | O(V) — scan a whole row |
| Add an edge | O(1) — append | O(1) — set a cell |

(`deg(u)`, the *degree*, is how many edges leave `u`.)

The decider is **density**. Most real graphs are *sparse*: E is nowhere near
the V² maximum. A social network with a million users where each follows a
few hundred people has E ≈ 10⁸ — but the matrix would allocate 10¹² cells,
almost all zero, before storing a single follow. The list stores only the
edges that exist. Reach for a matrix only when the graph is small or genuinely
*dense* (E close to V²), or when O(1) edge tests dominate everything else.
Everywhere else — and in this exercise — the adjacency list is the default.

## BFS: explore in ripples

**Breadth-first search** starts at a node and explores in rings: first the
start, then everything one edge away, then everything two edges away, and so
on — a ripple spreading from a stone dropped in water. The machinery is two
structures you built earlier in this stage:

- a **queue** holding the frontier — nodes discovered but not yet explored.
  FIFO order is what makes the ripple: nodes discovered earlier (closer)
  are explored before nodes discovered later (farther).
- a **visited set** — every node ever discovered. Membership must be cheap;
  a hash set is the usual choice.

```
bfs(graph, start):
    visited ← {start}
    order   ← [start]
    queue   ← [start]
    while queue is not empty:
        node ← dequeue(queue)
        for each neighbor of node:
            if neighbor not in visited:
                add neighbor to visited
                append neighbor to order
                enqueue(queue, neighbor)
    return order
```

Mark a node visited *when it is discovered* (enqueued), not when it is
dequeued — otherwise the same node can sit in the queue several times.

In the tree lessons you traversed without any visited set, and nothing went
wrong. That was not luck — it was the tree's guarantees working for you:
one parent, no cycles, so no node could ever be reached twice. Graphs revoke
those guarantees, and the visited set is what replaces them. Drop it and two
things go wrong, in order of severity: a node reachable along two routes
(say `1→2→4` and `1→3→4`) gets visited twice; and a cycle like `1→2→3→1`
sends the traversal around forever — the queue never drains. **On graphs,
tracking visited nodes is not an optimization; it is what makes the
traversal terminate.**

## Why BFS finds shortest paths

In an unweighted graph, the shortest path between two nodes is the one with
the fewest edges — and BFS finds it without ever comparing path lengths.

The argument: BFS explores in distance order. It discovers every node at
distance 1 before any node at distance 2, every node at distance 2 before
any at distance 3, and so on — the queue enforces it, because a node at
distance d is only discovered from a node at distance d−1, which entered the
queue earlier. So the *first* time BFS reaches a node, it got there in the
fewest possible edges. For a shorter route to be missed, a closer node would
have had to be discovered *after* a farther one — which the queue order
makes impossible. Record each node's distance when you discover it
(`dist[neighbor] = dist[node] + 1`) and you have shortest-path lengths for
free.

This argument leans on every edge counting the same. With weights it
collapses — two hops of cost 1 beat one hop of cost 10 — which is exactly
when you graduate to Dijkstra's algorithm.

## DFS: dive and backtrack

**Depth-first search** commits: from the start node, follow the first edge,
then the first edge from *that* node, as deep as possible; only when stuck
does it back up and try the next option. Recursion — the call stack doing
the backtracking for you — expresses it naturally:

```
dfs(graph, start):
    visited ← {}
    order   ← []
    walk(node):
        if node in visited: return
        add node to visited
        append node to order
        for each neighbor of node:
            walk(neighbor)
    walk(start)
    return order
```

(An explicit stack works too — the stacks lesson's LIFO discipline instead
of FIFO — but then neighbors pop in reverse push order; push them reversed
if the visit order matters.)

DFS needs the visited set for the same reasons BFS does — without it, a
cycle means infinite recursion until the call stack overflows.

BFS and DFS visit the **same set** of nodes — everything reachable from the
start — in different **orders**. And the order is the whole story: DFS's
order encodes *a* route, not the shortest one. If node 5 is reachable both
by a four-edge path and a two-edge path, DFS happily reports whichever its
first-edge choices stumble into. Use DFS when only reachability matters —
"can I get there at all?", "which nodes form this connected island?" — and
BFS when *how far* matters.

Both traversals, on an adjacency list, cost **O(V + E)**: every node enters
the queue or stack at most once (the visited set guarantees it), and every
edge is examined at most once, when its source node is explored. On a
matrix the same traversals cost O(V²) — finding one node's neighbors means
scanning a whole row.

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

## Exercise

Open [`exercise/`](exercise/) — a Go module with `graph.go` (the
`Graph` type above and five `TODO` methods) and `graph_test.go`. **Read the
tests first**; they are the specification, including the cycle graphs that
never terminate without visited tracking and a long-route/short-route graph
that BFS must get right.

Acceptance criteria:

1. `AddEdge(from, to)` stores the directed edge `from → to`;
   `Neighbors(from)` returns the targets of `from`'s outgoing edges in the
   order added. A node with no outgoing edges (or absent from the graph) has
   no neighbors. Undirected graphs are modeled by adding both directions.
2. `BFS(start)` returns `start` plus every node reachable from it, in
   breadth-first order — all nodes at distance 1 (in neighbor order), then
   distance 2, and so on. Every node appears exactly once, and traversal
   terminates on graphs with cycles.
3. `DFS(start)` returns the same set in depth-first order: follow the first
   neighbor as deep as possible before backtracking, neighbors in the order
   added. Same visited guarantees as BFS.
4. `ShortestPathLen(start, end)` returns the number of edges on the shortest
   path: `0` when `start == end`, `-1` when `end` is unreachable, and the
   short route's length even when a longer route was added first.
5. `Reachable(from, to)` reports whether a directed path leads from `from`
   to `to`; every node can reach itself.
6. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They must FAIL before you start. Suggested order: `AddEdge`/`Neighbors`
first (everything builds on them), then `BFS`, then `DFS`, then the two
applications.

## Further reading

- [Wikipedia — Graph (abstract data type)](https://en.wikipedia.org/wiki/Graph_(abstract_data_type))
- [Wikipedia — Breadth-first search](https://en.wikipedia.org/wiki/Breadth-first_search)
- [Wikipedia — Depth-first search](https://en.wikipedia.org/wiki/Depth-first_search)
- [Open Data Structures — Graphs (ch. 12, Python edition)](https://opendatastructures.org/ods-python/12_Graphs.html)
