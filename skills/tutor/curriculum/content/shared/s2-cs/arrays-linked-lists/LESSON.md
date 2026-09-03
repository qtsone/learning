# Arrays & Linked Lists

> `shared.cs.arrays-linked-lists` · ~3-4h · Stage: CS Fundamentals

## Objectives

By the end of this lesson you can:

- Explain how contiguous memory gives arrays O(1) indexing and cache-friendly
  iteration, and what linked nodes trade away to get O(1) insertion given a
  node reference.
- Implement a singly linked list with append, prepend, delete, and search
  operations that pass the provided tests.
- State the Big-O cost of index, search, insert-at-head, and insert-at-tail
  for arrays/slices versus linked lists.
- Choose between a slice and a linked list for a described workload and
  justify the choice with access-pattern and memory arguments.

## Memory is a street of numbered boxes

Picture your program's memory as one enormous street of same-sized boxes,
each with a numeric address: box 1000, box 1001, box 1002, … Every value your
program keeps lives in some run of boxes. All data structures are answers to
one question: *where do we put the values, and how do we find them again?*

An **array** gives the simplest possible answer: put the elements next to
each other, in one contiguous run.

```text
index:       0      1      2      3
         ┌──────┬──────┬──────┬──────┐
         │  10  │  20  │  30  │  40  │
         └──────┴──────┴──────┴──────┘
address:  1000   1008   1016   1024     (8 bytes per element)
```

Contiguity buys you a superpower: the address of any element is pure
arithmetic.

```text
address(i) = base_address + i × element_size
```

To fetch element 3, the computer computes `1000 + 3 × 8 = 1024` and reads one
box. To fetch element 3,000,000 it does the *same two operations*. That is
why array indexing is O(1) — the cost does not grow with the array's size or
with which index you ask for. No walking, no searching: multiply, add, read.

## Why iterating an array is even faster than it looks

Last lesson you learned to count steps. Hardware adds a twist: not all steps
cost the same. Main memory is slow compared to the CPU, so the CPU keeps
recently-used memory in small fast **caches** — and it doesn't fetch one box
at a time, it fetches a whole *cache line* (typically 64 bytes) of neighbors
in one go.

When you loop over an array front to back, element `i+1` is almost always
already in cache because it came along for the ride when `i` was fetched.
The hardware even notices the straight-line pattern and prefetches ahead.
The result: an O(n) scan over an array runs close to the memory system's top
speed. Remember this — Big-O counts steps, but the *constant factor* per step
can differ by 100× depending on whether each step hits cache or main memory.

## What contiguity costs

The same "everything is packed together" property that makes indexing free
makes *editing* expensive:

- **Insert at the front**: every existing element must shift one slot right
  to make room — O(n) moves.
- **Delete from the middle**: everything after the gap shifts left — O(n).
- **Grow past capacity**: an array's run of boxes has fixed size; the boxes
  after it may belong to something else. To grow, you allocate a bigger run
  elsewhere and copy everything over — O(n).

Dynamic arrays (Go's slices, Python's lists, Java's ArrayList) soften the
last cost with a trick: when they run out of room they don't grow by one,
they roughly **double** the capacity. Most appends then land in spare
capacity and cost O(1); the occasional copy is O(n), but doubling makes
copies so rare that the *average* append is still O(1). This "expensive
sometimes, cheap on average" accounting is called **amortized** O(1) — you'll
see the word again in this stage.

**In Go:** this is exactly the `len`/`cap` machinery from S1. A slice is a
view onto a backing array; `append` writes into spare capacity when it can
and allocates a bigger backing array when it can't:

```go
xs := make([]int, 0, 4)
for i := 0; i < 10; i++ {
	xs = append(xs, i)
	fmt.Println(len(xs), cap(xs)) // watch cap jump: 4, 8, 16…
}
```

That is also why S1 warned you to keep the result of `append`: after a
growth step, the slice points at a *new* backing array.

## Linked lists: trade the street for a treasure hunt

A **linked list** answers the where-do-values-live question the opposite
way: put each element wherever there happens to be room, and make each one
carry a signpost to the next. Each element lives in a **node** — a little
record holding the value and a reference (a pointer, in Go terms) to the
next node. The list itself just remembers where the first node — the
**head** — is.

```text
head ──▶ ┌────┬───┐     ┌────┬───┐     ┌────┬───┐
         │ 10 │ ●─┼───▶ │ 20 │ ●─┼───▶ │ 30 │ ∅ │
         └────┴───┘     └────┴───┘     └────┴───┘
```

The last node's next-reference is empty (`∅` — `nil` in Go), which is how a
traversal knows to stop:

```text
walk(list):
    current ← list.head
    while current is not empty:
        visit current.value
        current ← current.next
```

Now the trade-offs flip:

- **Index is O(n)**: there is no address arithmetic. To reach element 3 you
  must follow 3 signposts. Element 3,000,000 costs 3,000,000 hops.
- **Insert at the head is O(1)**: make a node, point it at the old head,
  call it the new head. Two pointer writes, no shifting, regardless of size.
- **Insert after a node you already hold is O(1)**: rewire two references.
  This is the linked list's real superpower — *given a node reference*,
  splicing in or out is constant time. The catch: *finding* that node in the
  first place is O(n) unless something else handed it to you.
- **Iteration is cache-hostile**: nodes are scattered across memory, so each
  hop is a potential cache miss, and the hardware can't prefetch a pointer
  chase. An O(n) list scan and an O(n) array scan share a growth rate but
  not a speed.

There is also a memory cost: every node pays for a next-pointer (and
allocation overhead) on top of the value. A list of small values can spend
more memory on plumbing than on data.

## Appending without walking: the tail pointer

Naively, appending to a linked list means walking to the end first — O(n)
per append, O(n²) to build a list of n elements. The fix is bookkeeping:
keep a second reference, the **tail**, always pointing at the last node.

```text
append(list, v):
    n ← new node(v)
    if list is empty:  list.head ← n
    else:              list.tail.next ← n
    list.tail ← n
```

Now append is O(1). The price of bookkeeping is that *every* operation that
can touch the last node must keep `tail` truthful — as you're about to
discover in the exercise, deleting the last node is where this bites. The
same trick applies to length: keep a counter, update it on every insert and
delete, and `length` becomes O(1) instead of an O(n) count.

## Deleting from a singly linked list

To unlink a node you must rewire the reference *pointing at it* — and in a
singly linked list, a node doesn't know who points at it. So deletion walks
with two references: the current node and the one just before it.

```text
delete(list, v):                        # remove first node holding v
    prev ← empty
    current ← list.head
    while current is not empty:
        if current.value = v:
            if prev is empty:  list.head ← current.next    # v was the head
            else:              prev.next ← current.next
            fix tail and length if needed
            report removed
        prev ← current
        current ← current.next
    report not found
```

Every edge case here is a classic interview trap: deleting the head (there
is no prev), deleting the tail (tail must move back to prev), deleting the
only element (both head and tail become empty), deleting a value that isn't
there. Your tests cover all four.

## The scorecard

| operation | array / slice | singly linked list |
|---|---|---|
| index `xs[i]` | O(1) | O(n) |
| search (unsorted) | O(n) | O(n) |
| insert at head | O(n) — shift everything | O(1) |
| insert at tail | amortized O(1) | O(1) with tail pointer, else O(n) |
| insert/delete given a node reference | O(n) — shift | O(1) |
| memory per element | value only, tightly packed | value + pointer + allocation |
| cache behavior on scan | excellent | poor |

## Choosing: slice or list?

Ask two questions about the workload:

1. **How is it accessed?** Random indexing or iterate-heavy → array/slice,
   no contest. Almost all work at the ends with the middle untouched, or
   long-lived node references that get spliced in and out → a list earns
   its keep.
2. **How does it change shape?** Mostly append-and-read → slice. Constant
   insertion/removal at the *front*, or splicing sequences together in O(1)
   → list.

And one honest caveat: because of caches, slices beat lists in practice far
more often than the table suggests. Even "insert at the front repeatedly" —
the list's showcase — is often fine as a slice until n gets large, because
one contiguous memcpy is fast and pointer-chasing is not. Measure before
you reach for the exotic structure; default to the slice.

**In Go:** the standard library ships a doubly linked list
(`container/list`), yet you'll rarely see it in production Go — slices win
almost every benchmark that doesn't specifically need O(1) splicing. You are
building a list from scratch here not because Go needs another one, but
because pointer-linked nodes are the atom that stacks, queues, trees, and
graphs — the rest of this stage — are built from.

## Exercise

Open [`exercise/`](exercise/) — a Go module with two files:

- `list.go` — a singly linked list of `int`s with the types and `New` given,
  and six methods marked `// TODO:` for you to implement.
- `list_test.go` — the specification. **Read it first**; note how
  `TestDelete` probes the edge cases and what `TestManyAppendsStayFast`
  implies about your `Append`.

Acceptance criteria:

1. `New()` returns an empty list: `Len()` is 0, `Values()` is empty,
   `Contains` reports false, `Delete` reports false.
2. `Append` adds to the back, `Prepend` adds to the front, `Values` returns
   the values front-to-back, and `Len` stays accurate through any mix of
   operations.
3. `Contains(v)` reports whether `v` is anywhere in the list.
4. `Delete(v)` removes only the *first* occurrence of `v` and reports
   whether it removed anything — handling head, middle, tail, and
   only-element cases. After deleting the tail, `Append` must still extend
   the list correctly.
5. `Append` and `Len` are O(1): the test performs 150,000 appends and fails
   if they take too long. Keep a tail pointer and a length counter — do not
   walk the list.
6. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They fail on the starter — make them green, one test at a time, in file
order.

## Further reading

- [Go blog — Go Slices: usage and internals](https://go.dev/blog/slices-intro)
- [Go blog — Arrays, slices: the mechanics of append](https://go.dev/blog/slices)
- [Interactive latency numbers](https://colin-scott.github.io/personal_website/research/interactive_latency.html) — see for yourself why a cache hit and a memory fetch are different worlds
- [pkg.go.dev — container/list](https://pkg.go.dev/container/list) — the standard library's doubly linked list, and how rarely you need it
