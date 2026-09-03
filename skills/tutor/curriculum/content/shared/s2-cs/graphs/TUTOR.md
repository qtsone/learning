# Tutor notes — Graphs

## Where the learner is

Tenth lesson of the CS stage. Everything BFS/DFS needs is already in their
hands: the slice-backed queue (stacks & queues), O(1) membership sets (hash
tables), recursion and the call stack, and tree traversals. Frame graphs as
"trees with every restriction removed" and traversal as "tree traversal that
must now defend itself against cycles". The two genuinely new mental moves
are the visited set as a *termination* requirement (not an optimization) and
BFS discovery order encoding distance. Dynamic programming is next — don't
reach forward into Dijkstra, topological sort, or weighted graphs beyond
naming them.

## Common misconceptions

- **No visited set** — works on tree-shaped graphs, loops forever on cycles.
  The diamond test (`TestBFSVisitsEachNodeOnce`) fails loudly with a
  duplicated 4 *before* the cycle tests can hang; steer them to fix that one
  first. If a run does hang, that is the cycle tests — ^C and ask what the
  queue contains over time.
- **Marking visited at dequeue instead of enqueue** — output is still
  correct here (the visited check on pop skips duplicates), but the queue
  holds repeats. Not a failing offense; worth a "what's in your queue on the
  diamond graph?" conversation.
- **"BFS and DFS are basically the same"** — same reachable *set*, different
  *order*, and only BFS's order encodes distance. The short-route test makes
  this concrete: DFS following first-added edges reports 4 hops where the
  answer is 2.
- **Undirected edge added one way only** — then `Reachable(3, 1)` is false
  and they blame the traversal. Undirected = both directions, every time.
- **Reaching for an adjacency matrix by default** — have them price the
  memory of a matrix for a million-node, few-edges-per-node graph (10¹²
  cells) versus the list's O(V + E).
- **"Go map order is random, so traversal order is untestable"** — only
  iteration over the *map* is randomized; no function here does that.
  Traversals walk per-node neighbor *slices*, which preserve append order.
- **`ShortestPathLen` via DFS** — returns the length of whatever path DFS
  stumbles into first. If they wrote it that way and it fails, ask which
  traversal's order means "fewest edges first" and why.

## Grilling points

- "Model these as graphs — nodes, edges, directed?, weighted?: a flight-price
  planner; Go package imports; the git commit DAG; 'mutual friends' on a
  social network." (Objective 1 head-on; expect direction and weight choices
  *justified*, not recited.)
- "Your graph has 10,000 nodes, each touching about 5 others. Memory for the
  list vs the matrix? Cost of 'is there an edge u→v' in each? When would you
  still pick the matrix?"
- "Convince me the first time BFS reaches a node is via the fewest edges.
  What would have to be true of the queue for a shorter route to be missed?"
- "Delete your visited check mentally and run `1→2, 2→3, 3→1` — narrate the
  queue. Now do the same for DFS — what actually crashes, and when?"
  (Queue grows forever vs call-stack overflow.)
- "Why did your tree traversals last week need no visited set?" (One parent,
  no cycles — the tree's guarantees did the job.)
- "`Reachable` uses BFS. Would DFS be equally correct? Would it be equally
  correct inside `ShortestPathLen`? Why the difference?"
- "What is the cost of `BFS` on the adjacency list, in V and E? Why is each
  edge examined at most once?"

## Grading rubric

- **A** — All tests pass; BFS is queue + visited marked at discovery; DFS is
  recursion or a correctly-reversed explicit stack; `ShortestPathLen`
  derives distances from BFS levels (no path enumeration); `Reachable`
  reuses a traversal rather than duplicating one; learner can model a novel
  problem as a graph, argue list vs matrix with V/E numbers, and give the
  level-order argument for BFS shortness; gofmt-clean.
- **B** — Tests pass with rough edges: traversal logic copy-pasted between
  methods instead of reused, distances recomputed clumsily but correctly, or
  the shortest-path explanation is "BFS just finds it" until prompted.
- **C** — Tests pass only after heavy hints, or the learner cannot trace the
  BFS queue on the diamond graph on paper, or cannot say why the visited set
  exists beyond "the test wanted it". Pass only if remediation lands.
- **Fail** — Tests failing, or a working visited set the learner cannot
  explain the necessity of. Remediate, don't advance.

## Remediation ladder

1. "Draw the diamond graph (1→2, 1→3, 2→4, 3→4). Play BFS on paper: show me
   the queue and the visited set after every step. Where would 4 enter the
   queue twice?"
2. "Which structure from this stage gives you 'first discovered, first
   explored'? Which gives 'have I seen this node?' in O(1)? Those two are
   the whole algorithm."
3. "For `ShortestPathLen`: when you discover a neighbor, what do you already
   know about the node you discovered it *from*? Write that as one line of
   arithmetic."
4. Talk through the BFS skeleton aloud — seed queue and visited with start;
   pop; for each unseen neighbor mark, record, push — and let them type it.
   Then DFS is the same visited set with the call stack doing the queue's
   job.

## After passing

Preview: "Next: dynamic programming — you've been trading structure for
speed all stage; now you'll trade *memory* for speed by remembering answers
to subproblems you'd otherwise recompute."
