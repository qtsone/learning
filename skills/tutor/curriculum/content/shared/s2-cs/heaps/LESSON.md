# Heaps & Priority Queues

> `shared.cs.heaps` · ~2-3h · Stage: CS Fundamentals

## Objectives

By the end of this lesson you can:

- Explain the heap property and how a complete binary tree is stored compactly
  in an array.
- Trace sift-up and sift-down operations by hand and state why push and pop
  cost O(log n).
- Implement a priority queue using the language's heap facilities (e.g.
  `container/heap`) that passes the provided tests.
- Choose a heap over a sorted slice for a top-k or scheduling problem and
  justify the complexity difference.

## The priority queue problem

A queue (from the stacks & queues lesson) serves items in arrival order:
oldest first. Many systems need a different rule: serve the **most urgent**
first. An emergency room treats the sickest patient, not the earliest one. An
operating system runs the highest-priority task next. A timer wheel fires the
alarm with the soonest deadline. The abstract data type for "items keep
arriving; always hand me the most important one" is the **priority queue**:

- **push(item)** — add an item carrying a priority.
- **pop()** — remove and return the item with the *best* priority. We'll say
  smallest value = best; "largest = best" is the same machine with the
  comparison flipped.

With the structures you already have:

| backing structure | push | peek best | pop best |
|---|---|---|---|
| unsorted slice | O(1) amortized append | O(n) scan | O(n) scan + remove |
| sorted slice (best kept at the end) | O(n) find spot + shift | O(1) | O(1) take from the end |
| **binary heap** | **O(log n)** | **O(1)** | **O(log n)** |

Each slice option makes one side cheap by making the other expensive. That's
fine when the workload is lopsided — but schedulers, simulations, and timer
systems *interleave* pushes and pops endlessly, so the O(n) side gets paid
over and over. The heap charges O(log n) at both doors, guaranteed.

## A weaker promise than the BST

The trees lesson ended with a teaser: a tree that keeps a weaker promise than
the BST's and buys back guaranteed O(log n). Here it is. A **binary min-heap**
is a binary tree obeying two rules:

1. **Shape — completeness.** Every level is full except possibly the last,
   and the last level is filled left to right, no gaps.
2. **Order — the heap property.** Every node's key is ≤ its children's keys.

```
        (2)               a min-heap: every parent ≤ its children
       /   \
     (5)    (3)           note 5 > 3 — siblings are NOT ordered.
    /  \    /             This is not a BST.
  (9)  (6) (8)
```

Compare the invariants. The BST orders *horizontally*: everything left of a
node is smaller than everything right of it, across whole subtrees. The heap
orders only *vertically*: parent ≤ child, and that's all. Siblings are
unordered; the second-smallest key could be either child of the root. You
cannot binary-search a heap — finding an arbitrary key means checking
everything, O(n).

So why accept the weaker promise? Two payoffs:

- The root is *always* the minimum — peek is O(1), no walking.
- The shape rule makes degeneration **impossible**. Your BST collapsed into a
  height-1022 chain when fed sorted input; a heap's shape is fixed by its
  *size alone*. n nodes always occupy ⌊log₂ n⌋ + 1 levels, whatever order the
  values arrived in. The O(log n) is worst-case, not hopeful.

## A tree without pointers

Completeness has a second, beautiful consequence. Read the tree level by
level, left to right, and write the keys into an array:

```
index:    0   1   2   3   4   5
value:  [ 2,  5,  3,  9,  6,  8 ]

              0
            /   \
           1     2        ← the same tree, drawn as indices
          / \   /
         3   4 5
```

Because there are no gaps, position alone encodes the links — no `Left`/
`Right` pointers, just arithmetic (all divisions are integer divisions):

- children of index `i` live at `2i + 1` and `2i + 2`
- the parent of index `i` lives at `(i - 1) / 2`

Check it against the picture: index 1 (key 5) has children at 3 and 4 (keys
9, 6); index 5's parent is (5−1)/2 = 2 (key 3). A "missing" child is simply
an index ≥ length. (Textbooks that start at index 1 use `2i`, `2i+1`, `i/2` —
same idea, shifted by one. We stay 0-based.)

This is why completeness is load-bearing: one hole in the middle and every
index after it would lie. It also means the heap inherits everything you know
about arrays — contiguous memory, cache-friendly walks, amortized O(1)
append — while *behaving* like a tree.

## Push: append, then sift up

To insert, put the new key in the only place the shape rule allows growth —
the end — then repair the order rule by **sifting up**: compare with the
parent, swap while out of order.

```
push(heap, x):
    append x                          # shape stays complete
    i = last index
    while i > 0 and heap[parent(i)] > heap[i]:
        swap heap[i], heap[parent(i)]
        i = parent(i)
```

Trace `push 1` onto `[2, 5, 3, 9, 6, 8]`:

```
append at 6:      [2, 5, 3, 9, 6, 8, 1]
parent(6) = 2:     3 > 1 → swap  →  [2, 5, 1, 9, 6, 8, 3]
parent(2) = 0:     2 > 1 → swap  →  [1, 5, 2, 9, 6, 8, 3]
i = 0:             stop — 1 is the new minimum, at the root
```

Only the new key's ancestors are ever touched. Every swap moves a *smaller*
key into a parent slot, which can't upset that parent's other child — so the
property holds everywhere once the loop stops.

## Pop: the last leaf replaces the root

The minimum sits at index 0, but you can't just delete index 0 — that leaves
a hole at the top, and shifting the whole array over is O(n) and scrambles
the tree anyway. The shape rule says the only slot allowed to *disappear* is
the last one. So: hand back the root, move the **last** element into the root
slot, shrink by one, and repair downward — **sift down**: swap with the
*smaller* child while either child is smaller.

```
pop(heap):
    best = heap[0]
    heap[0] = last element; shrink        # shape restored, order broken at the top
    i = 0
    loop:
        s = smallest of: i, left(i), right(i)     # existing indices only
        if s == i: stop
        swap heap[i], heap[s]; i = s
    return best
```

Trace `pop` on `[2, 5, 3, 9, 6, 8]`:

```
return 2; move 8 up:   [8, 5, 3, 9, 6]
i = 0, children 5, 3:   smallest is 3 (index 2) → swap → [3, 5, 8, 9, 6]
i = 2, left child = index 5 ≥ length:  no children → stop
```

Why the *smaller* child? The promoted key must end up ≤ both children.
Promoting the smaller child guarantees that: it was already ≤ its sibling.
Swap with the larger child once and you've placed a bigger key above a
smaller one — the classic sift-down bug, and the tests will catch it.

Both sifts move one level per step, and a complete tree has ⌊log₂ n⌋ levels:
a million keys is ~20 levels, so a push or pop touches ~20 nodes ever. And a
bonus you've half-met already: push everything, then pop everything, and the
values come out sorted — n pops at O(log n) each. That algorithm has a name,
**heapsort**: O(n log n) guaranteed, like merge sort but in place.

## Heap vs sorted slice: top-k and scheduling

Two workloads where this choice actually shows up:

**Top-k.** "The 10 largest scores in a 10-million-entry stream." Sorting
everything costs O(n log n) time and O(n) memory — and requires having all
the data at once. Instead keep a **min**-heap holding at most k elements:
push each arriving value; when the size exceeds k, pop. The root is always
the *weakest member of the current top-k* — exactly the one to evict. Total:
O(n log k) time, O(k) memory, and it works on an endless stream. Yes,
min-heap to track the *largest* values — the heap's one instant answer is
"who is closest to falling out of the club?"

**Scheduling.** Tasks arrive continuously; between arrivals you repeatedly
run the most urgent one. A sorted slice pays O(n) per arrival to keep itself
sorted; the heap pays O(log n) per arrival *and* O(log n) per dispatch. The
rule of thumb from the sorting lesson still stands — sort once when the data
is static — but the moment inserts and extracts interleave and only the
current best matters, the heap wins.

In Go:

Most languages ship a heap (Python's `heapq`, Java's `PriorityQueue`, C++'s
`std::priority_queue`). Go's is `container/heap`, and its shape surprises
people: it doesn't hand you a heap type. It implements the *algorithms* —
the sifts you just traced — and asks you to supply the storage as five
methods on your own type:

```go
import "container/heap"

type TaskQueue []Task

func (q TaskQueue) Len() int           { return len(q) }
func (q TaskQueue) Less(i, j int) bool { return q[i].Priority < q[j].Priority }
func (q TaskQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }

func (q *TaskQueue) Push(x any) { *q = append(*q, x.(Task)) }
func (q *TaskQueue) Pop() any {
	old := *q
	t := old[len(old)-1]
	*q = old[:len(old)-1]
	return t
}
```

Reading guide:

- `Len`, `Less`, `Swap` let the package inspect and reorder your slice
  without knowing what's in it. `Less` *is* the priority rule: `<` builds a
  min-heap; flip the comparison for a max-heap.
- `any` is Go's "a value of any type", and `x.(Task)` (a *type assertion*)
  converts it back. This is your first brush with Go interfaces — the next
  Go stage opens with them; until then, treat these two lines as the standard
  pattern to copy.
- `Push` and `Pop` use pointer receivers because they change the slice's
  length, and the caller must see that.
- **The trap everyone hits once:** your five methods are plumbing — you never
  call them. You call the *package functions*, which call your methods at the
  right moments:

```go
heap.Init(&q)                  // establish the heap property over existing data
heap.Push(&q, Task{...})       // append via your Push, then sift up
next := heap.Pop(&q).(Task)    // the most urgent task
```

`heap.Pop` runs exactly the pop you traced: swap root to the end, sift down,
then call *your* `Pop` — which is why your method removes and returns the
**last** element even though the caller receives the minimum. Calling
`q.Push(t)` directly would append without sifting and silently break the
heap; the tests exercise everything through the package functions.

## Exercise

Open [`exercise/`](exercise/) — a Go module with two files to
complete and their tests. **Read the tests first**; note how
`minheap_test.go` reaches into the backing array to check the parent/child
rule at exact indices — the array layout is part of the specification.

- `minheap.go` — a from-scratch `MinHeap` of ints (`Push`, `Pop`, `Peek`,
  `siftUp`, `siftDown`) plus `TopK`, built on your heap.
- `taskqueue.go` — a `TaskQueue` implementing the five `container/heap`
  methods, popped in ascending `Priority` order.

Acceptance criteria:

1. `MinHeap`: popping repeatedly returns values in ascending order for any
   push order, duplicates included; `Peek` returns the minimum without
   removing it; `Pop`/`Peek` on an empty heap return `ok == false`; `Len`
   tracks the count.
2. After every tested operation the backing array satisfies
   `data[(i-1)/2] <= data[i]` for all `i ≥ 1` — the white-box invariant test
   checks exactly this.
3. `TopK(nums, k)` returns the k largest values in descending order;
   `k <= 0` yields an empty result, `k >= len(nums)` yields all values.
   Implement it with your `MinHeap` holding at most k elements — the tests
   can't see your approach, but your tutor will ask you to defend O(n log k).
4. `TaskQueue` works under `heap.Init`, `heap.Push`, and `heap.Pop`: draining
   it yields tasks in ascending `Priority`, and the slice obeys the same
   parent/child rule after every operation.
5. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They must FAIL before you start. Suggested order: `siftUp` + `Push` first
(the invariant test starts passing), then `Pop` + `siftDown`, then `Peek`,
`TopK`, and finally the `TaskQueue`.

## Further reading

- [pkg.go.dev — container/heap](https://pkg.go.dev/container/heap) (the
  package docs include a full priority-queue example)
- [Wikipedia — Binary heap](https://en.wikipedia.org/wiki/Binary_heap)
- [Wikipedia — Priority queue](https://en.wikipedia.org/wiki/Priority_queue)
- [Open Data Structures — Heaps (ch. 10, Python edition)](https://opendatastructures.org/ods-python/10_Heaps.html)
