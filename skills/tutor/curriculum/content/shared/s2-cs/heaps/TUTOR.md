# Tutor notes — Heaps & Priority Queues

## Where the learner is

Ninth lesson of the CS stage. They have built linked lists, stacks, queues, a
hash table, binary search, sorting algorithms, and a BST — including the
1023-key test where sorted input degraded the BST to a chain. Lean on that
memory: the heap's completeness rule is the answer to it. This lesson is also
their first brush with `container/heap`, which means first contact with
interface satisfaction and `any`; keep it at "five methods, the package calls
them" and resist teaching S3 interfaces early. Graphs come next — don't reach
forward.

## Common misconceptions

- **"The heap array is sorted"** — only parent ≤ child holds. In
  `[2, 5, 3, 9, 6, 8]`, 5 > 3 at indices 1 and 2 and that's fine: siblings
  are unordered. If they expect sorted output from `h.data`, revisit the
  invariant.
- **Heap-the-structure vs heap-the-memory-region** — the allocator's "heap"
  shares nothing but the name. Clear this up the moment it surfaces.
- **"Heaps can search"** — finding an arbitrary key is O(n). The heap's weak
  vertical-only promise can't discard a subtree the way the BST's horizontal
  promise can.
- **Pop via `data = data[1:]`** — drops the minimum but scrambles every
  parent/child relationship (all indices shift). Swap-with-last is the
  load-bearing move; the invariant test catches this immediately.
- **Sift-down swaps with the larger (or first-found) child** — must promote
  the *smallest* of the two, else a bigger key lands above the remaining
  child. `TestArrayLayoutInvariant` exists to catch exactly this.
- **1-based formulas in a 0-based slice** — textbooks say `2i`/`2i+1`/`i/2`;
  we need `2i+1`/`2i+2`/`(i-1)/2`. Off-by-one here fails everything at once.
- **Calling `q.Push`/`q.Pop` directly** instead of `heap.Push(&q, …)` /
  `heap.Pop(&q)` — appends without sifting. Related: value receivers on
  Push/Pop, so the append is lost (`TestSchedulerScenario`'s Len message
  points at this), or a `Pop` method that removes `q[0]` instead of the last
  element.
- **TopK with the wrong heap** — a max-heap of all n values works but is
  O(n) memory; the point is a min-heap bounded at k, whose root is the
  weakest current member. Expect "why a MIN-heap for the LARGEST values?" —
  that inversion is the teaching moment.

## Grilling points

- "Your heap holds a million values. How many levels? What's the worst case
  a push can climb — and why could your BST degrade to n but this can't?"
- "In `[2, 5, 3, 9, 6, 8]`, where can the second-smallest value live? Point
  at the indices." (1 or 2 only — children of the root.)
- "Trace `Pop` on paper for `[2, 5, 3, 9, 6, 8]` — every swap, every index."
- "Why does the *last* element replace the root, instead of promoting the
  smaller child upward all the way down?" (Cascading promotion leaves the
  hole at some leaf that usually isn't the last slot — completeness breaks.)
- "`heap.Pop(&q)` hands you the most urgent task, but your `Pop` method
  returns the last element. Reconcile those two facts."
- "Top-10 of a 10-million-entry stream: full sort vs a sorted 10-slot slice
  vs a 10-slot min-heap — time, memory, and what changes when the stream
  never ends?"
- "Pop everything out of your MinHeap and collect the values. What algorithm
  did you just run, and what's its total cost?" (Heapsort, O(n log n).)

## Grading rubric

- **A** — All tests pass; sift loops derive the index formulas rather than
  cargo-culting them (learner can recompute children/parent from the level
  picture); `Pop` uses swap-with-last; `TopK` keeps at most k elements and
  they can defend O(n log k) and the min-heap-for-largest inversion; the
  five `TaskQueue` methods are minimal with pointer receivers only where
  needed; gofmt-clean.
- **B** — Tests pass but with rough edges: a `TopK` that heaps all n values
  (or sorts) without seeing the memory cost, sift-down written with special
  cases for one-child nodes instead of the smallest-of-three shape, or a
  correct `TaskQueue` they can only partly explain (e.g. can't say who calls
  `Swap`).
- **C** — Tests pass only after heavy hints, or the learner cannot trace a
  push or pop on paper without running the code. Pass only if remediation
  lands; otherwise iterate.
- **Fail** — Tests failing, or `container/heap` methods pasted from the docs
  without being able to explain the `heap.Pop` last-element contract.
  Remediate, don't advance.

## Remediation ladder

1. "Draw `[2, 5, 3, 9, 6, 8]` as a tree on paper — one level per row. Now
   compute the children of index 1 and the parent of index 5 with the
   formulas, and check them against your drawing."
2. "Push 1 by hand: which index does it land on, which indices does it visit
   on the way up, where does it stop? Now single-step your `siftUp` and find
   where it disagrees."
3. "Removing the root leaves a hole at the top. Which slot is the *only* one
   the shape rule allows to disappear? So which element must move, and to
   where — and which direction do you repair?"
4. "Open the `container/heap` docs example next to your `taskqueue.go`. Map
   each of the five methods to a line of yours, then trace `heap.Pop`'s three
   steps (swap to end, sift down, call your Pop) on a three-task queue." Let
   them fix their own code after each step.

## After passing

Preview: "Next: graphs — drop the 'exactly one parent' rule entirely, let
anything link to anything, and learn BFS/DFS, the two ways to explore what a
traversal can no longer cover."
