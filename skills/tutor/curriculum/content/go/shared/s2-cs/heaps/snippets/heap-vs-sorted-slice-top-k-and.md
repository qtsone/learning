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
