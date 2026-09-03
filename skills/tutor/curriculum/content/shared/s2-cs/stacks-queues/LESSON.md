# Stacks & Queues

> `shared.cs.stacks-queues` · ~2-3h · Stage: CS Fundamentals

## Objectives

By the end of this lesson you can:

- Explain LIFO and FIFO ordering and identify which discipline a described
  problem requires.
- Implement a stack and a queue with push/pop (enqueue/dequeue) and peek
  operations that pass the provided tests.
- Compare slice-backed and linked-node implementations, including the
  amortized cost of growth and the pitfall of dequeuing from a slice's front.
- Apply a stack to a classic use case such as balanced-parentheses checking.
- Name real systems where queues appear (task scheduling, BFS, buffering) and
  explain why FIFO fits each.

## Less power, more meaning

Arrays and linked lists (last lesson) let you touch any element at any time.
Stacks and queues deliberately take that power away: you may only add and
remove at the ends, in one fixed pattern. The restriction *is* the feature —
when a collection can be used in only one way, every piece of code that
touches it tells you the exact order things will happen in, and a whole class
of ordering bugs becomes impossible to write.

Neither one is a new memory layout. Both are **disciplines** — a rule about
which end you may touch — imposed on the structures you already know.

## Stacks: last in, first out

A stack works like a stack of plates: you add to the top and take from the
top. The last element in is the first element out — **LIFO**.

```
push(D)                        pop() → D
   │                              ▲
   ▼                              │
 ┌───┐                          ┌───┐
 │ D │ ← top                    │ D │
 ├───┤                          ├───┤
 │ C │                          │ C │
 ├───┤                          ├───┤
 │ B │                          │ B │
 ├───┤                          ├───┤
 │ A │ ← bottom (oldest)        │ A │
 └───┘                          └───┘
```

The classic operations, each O(1):

| operation | meaning                              |
|-----------|--------------------------------------|
| push(x)   | place x on top                       |
| pop()     | remove and return the top element    |
| peek()    | return the top element, don't remove |
| len()     | how many elements are held           |

You already run on a stack. When `main` calls `parse`, which calls
`readFile`, the runtime pushes a record for each call and pops it on return:
the most recently started call always finishes first, so nested calls unwind
in reverse order. The Recursion lesson later in this stage leans hard on this
picture.

Stack-shaped problems share a smell: **the most recent thing matters first**,
or things nest and must be closed in reverse order of opening. Undo history
(undo the *last* edit), a browser's back button (return to the *previous*
page), matching nested brackets — all stacks.

## Queues: first in, first out

A queue is the line at a bakery: you join at the back, you're served at the
front. The first element in is the first out — **FIFO**.

```
enqueue(D)                                     dequeue() → A
    │                                              ▲
    ▼                                              │
  back ┌───┬───┬───┬───┐ front
   ──▶ │ D │ C │ B │ A │ ──▶
       └───┴───┴───┴───┘
```

| operation  | meaning                                |
|------------|----------------------------------------|
| enqueue(x) | add x at the back                      |
| dequeue()  | remove and return the front element    |
| peek()     | return the front element, don't remove |
| len()      | how many elements are held             |

Queues appear wherever work *arrives* at a different rate than it can be
*handled*, and arrival order must be respected:

- **Task scheduling** — print jobs, background jobs, CPU run queues. "First
  submitted, first served" is the fairness contract; a queue is that contract
  as a data structure.
- **Buffering** — keystrokes, network packets, audio samples. A fast producer
  and a slow consumer meet in the middle; the buffer must release data in the
  order it arrived or the stream becomes garbage.
- **Breadth-first search** — the graph traversal you will meet in the Graphs
  lesson explores everything one step away before anything two steps away.
  "Process in the order discovered" is exactly what a queue does.

## Which discipline does a problem need?

Ask one question: *when I take an element out, should it be the newest or the
oldest?*

- Newest first — undoing, backtracking, unwinding nesting → **stack**.
- Oldest first — fairness, arrival order, spreading work over time →
  **queue**.

Print jobs must come out in the order sent: queue. Ctrl+Z must revert the
latest change, not the first one ever made: stack. If you can say *why* one
order is correct and the other is a bug, you've identified the discipline.

## Case study: balanced brackets

Is every bracket in `([]{})` closed in the right order? This is the classic
stack application, and it shows why "count openers minus closers" is not
enough. Pseudocode:

```
for each character c in the text:
    if c is an opener ( [ {:
        push c
    if c is a closer ) ] }:
        if the stack is empty:            # closer with nothing open
            return false
        if pop() does not pair with c:    # wrong closer for the last opener
            return false
return stack is empty                     # anything left open?
```

Walk `([)]`: push `(`, push `[`, then `)` arrives — pop gives `[`, which does
not pair with `)`: unbalanced. A counter would say "two open, two closed,
fine" because it forgets *which* opener came most recently. The stack
remembers, in exactly the order that matters: last opened must be first
closed. Nesting is LIFO by nature.

## Implementing a stack

A dynamic array is the natural backing store: push appends at the end, pop
shrinks from the end, and the end of an array is O(1) to touch. When the
array is full, push allocates a bigger one (typically double) and copies
everything over — an O(n) moment. Spread that rare copy over the many cheap
pushes that preceded it and each push averages out to O(1). That averaged
figure is called **amortized** O(1): individual operations may occasionally
be expensive, but any sequence of n pushes costs O(n) total.

In Go: a slice plus `append` is already 90% of a stack — this is why Go has
no stack type in the standard library.

```go
s.items = append(s.items, r)          // push
top := s.items[len(s.items)-1]        // read the top
s.items = s.items[:len(s.items)-1]    // shrink by one: pop's second half
```

For the empty case, follow the comma-ok convention you know from map lookups:
`Pop` returns `(value, ok)` and reports `ok = false` instead of panicking on
an empty stack.

## Implementing a queue — and the trap

A queue needs O(1) at *both* ends, and here the array bites back. Enqueue at
the back is amortized O(1), same as push. Dequeue at the front gives you two
bad options:

1. **Shift** every remaining element one slot left: O(n) per dequeue.
   Draining an n-element queue costs O(n²) — 100,000 items means on the
   order of five billion moves.
2. **Advance a front index** and leave the slot behind: O(1) time, but the
   abandoned slots are dead weight. The array only grows, and a long-lived
   queue drags its entire history behind it.

In Go: option 2 is the tempting one-liner `q.items = q.items[1:]`. It reslices
in O(1), but the backing array is not freed while any slice still points into
it — dequeued elements stay reachable, so a busy queue quietly pins memory it
will never read again.

The clean fix you can build today is **linked nodes** with two pointers, one
at each end:

```
 head                          tail
   │                             │
   ▼                             ▼
 ┌───┐      ┌───┐      ┌───┐
 │ A │ ───▶ │ B │ ───▶ │ C │ ───▶ nil

 dequeue:  head = head.next               (drop A)
 enqueue:  tail.next = new; tail = new    (attach D)
```

Both operations follow one pointer: true O(1), no amortization, and each
dequeued node becomes garbage the collector can actually reclaim. The
trade-off is the one from last lesson: one allocation per element, and
pointer-chasing instead of cache-friendly contiguous memory. (Production
libraries often square the circle with a **ring buffer** — an array whose
front and back indices wrap around.)

Watch the edges: an empty queue has `head` and `tail` both nil, and dequeuing
the *last* element must reset `tail` too — forget that and the next enqueue
appends to a ghost node. The tests check exactly this.

## Exercise

Open [`exercise/`](exercise/) — a Go module with three work sites,
each marked `TODO`, and the tests that decide when you're done:

- `stack.go` — `Stack`, a LIFO collection of runes backed by a slice:
  `Push`, `Pop`, `Peek`, `Len`.
- `queue.go` — `Queue`, a FIFO collection of ints backed by linked nodes
  with `head`/`tail` pointers: `Enqueue`, `Dequeue`, `Peek`, `Len`.
- `brackets.go` — `Balanced(s string) bool`, the bracket checker, built *on
  your own `Stack`*.

Acceptance criteria:

1. `Stack` is LIFO: after pushing `a`, `b`, `c`, pops return `c`, `b`, `a`.
   `Pop`/`Peek` on an empty stack return `ok = false`; `Peek` never removes;
   `Len` tracks pushes and pops.
2. `Queue` is FIFO: after enqueuing 1, 2, 3, dequeues return 1, 2, 3.
   `Dequeue`/`Peek` on an empty queue return `ok = false`; `Peek` never
   removes; `Len` tracks both operations.
3. A queue that has been drained to empty accepts new elements correctly
   (the `tail` reset from the theory).
4. The queue survives 100,000 enqueues followed by 100,000 dequeues in
   moderate time — O(1) per operation. A front-shifting implementation will
   crawl here.
5. `Balanced` accepts `""`, `"()"`, `"([]{})"`, `"{[()]}"`, and text with
   non-bracket characters like `"a(b)c"`; it rejects `"([)]"`, `"((("`,
   `"())"`, and `")("`.
6. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They fail before you start — read the failures first; they are the
specification.

## Further reading

- [Go blog — Go Slices: usage and internals](https://go.dev/blog/slices-intro)
- [Go wiki — SliceTricks](https://go.dev/wiki/SliceTricks)
- [Wikipedia — Stack (abstract data type)](https://en.wikipedia.org/wiki/Stack_(abstract_data_type))
- [Wikipedia — Queue (abstract data type)](https://en.wikipedia.org/wiki/Queue_(abstract_data_type))
