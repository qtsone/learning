# Recursion

> `shared.cs.recursion` · ~2-3h · Stage: CS Fundamentals

## Objectives

By the end of this lesson you can:

- Explain how the call stack tracks recursive calls and why a missing or wrong
  base case causes a stack overflow.
- Identify the base case and the recursive case in a problem statement before
  writing any code.
- Implement recursive solutions to classic problems — factorial, reversing,
  tree-shaped input — that pass the provided tests.
- Convert a recursive solution to an iterative one and explain when iteration
  is preferable.
- Trace a recursive execution by hand, drawing the stack frames for a small
  input.

## A function that calls itself

Recursion is a function solving a problem by calling *itself* on a smaller
version of the same problem. That sounds circular, but it isn't — as long as
the input shrinks toward a case so small the function can answer without
calling itself again.

The classic warm-up is factorial: `n!` is the product `n × (n−1) × … × 1`,
and mathematicians define it in two lines:

```
0! = 1
n! = n × (n−1)!        for n > 0
```

That definition is *already recursive* — factorial is defined in terms of a
smaller factorial. The code just mirrors it:

```
factorial(n):
    if n == 0:                       # base case
        return 1
    return n * factorial(n - 1)      # recursive case
```

There is no magic here. `factorial` calling `factorial` works exactly like
`factorial` calling any other function. The machinery that makes it work is
something you built by hand one lesson ago.

## The stack you already built

In the stacks-and-queues lesson you implemented a stack: push on top, pop from
the top, LIFO. Your language runtime uses precisely that structure — the
**call stack** — to track every function call in your program:

- **Call** a function → push a **frame**: its parameters, its local
  variables, and where to resume when it returns.
- **Return** from it → pop the frame and continue where the caller left off.

Recursion is nothing more special than the same function having several frames
on the stack at once. Crucially, **each frame has its own private copies** of
the parameters and locals. Five frames of `factorial` means five independent
`n`s.

Here is the stack at the deepest moment of `factorial(3)`:

```
│ factorial(0)  n=0 │ ← top: base case, about to return 1
│ factorial(1)  n=1 │   waiting for factorial(0)
│ factorial(2)  n=2 │   waiting for factorial(1)
│ factorial(3)  n=3 │   waiting for factorial(2)
│ main              │
```

Then the stack **unwinds**: `factorial(0)` returns 1, so `factorial(1)`
computes `1 × 1` and returns 1, so `factorial(2)` returns `2 × 1 = 2`, so
`factorial(3)` returns `3 × 2 = 6`. Every recursive execution has these two
phases: *winding* (calls pile up) and *unwinding* (results combine on the way
back down).

## Base case and recursive case

Before writing any recursive code, answer two questions **from the problem
statement**:

1. **Base case** — what is the smallest input I can answer immediately,
   with no further calls? (Empty list → sum is 0. Zero or one characters →
   already reversed. `n == 0` → factorial is 1.)
2. **Recursive case** — how do I shrink the input one step toward the base
   case, *trusting the recursive call* to handle the rest? (Sum of a list =
   first element + sum of the remaining list.)

That word *trusting* matters. The most common beginner trap is trying to
mentally unroll every level at once. Don't. Assume the recursive call already
works for smaller input — your only job is the one step that connects the
smaller answer to the current one. This "leap of faith" is legitimate because
of a checklist you can verify mechanically:

- a base case exists,
- it is reachable for every valid input,
- every recursive call makes progress toward it.

If all three hold, induction guarantees the whole thing works — the same way
`n! = n × (n−1)!` is a complete definition.

## When it goes wrong: stack overflow

Break any item on that checklist and the frames never stop piling up. The
call stack is finite, so the program dies — a **stack overflow**.

Consider `factorial(-1)` with the code above: the base case is `n == 0`, but
from −1 the recursive case goes to −2, −3, … — the base case exists but is
*unreachable*, so the recursion winds forever until the stack limit kills it.

Even *correct* recursion can overflow: one frame per level means the input's
depth is your stack depth. A recursive walk over a million-element linked
list needs a million live frames at once.

> **In Go:** each goroutine's stack starts tiny and grows on demand, but only
> up to a hard cap (about 1 GB on 64-bit machines by default). Blowing it is
> not a recoverable panic — the program dies with:
>
> ```
> runtime: goroutine stack exceeds 1000000000-byte limit
> fatal error: stack overflow
> ```
>
> You will meet this exact message in the exercise, on purpose.

## Tracing by hand

Tracing recursion on paper is a skill, not busywork — it is how you debug
recursion for the rest of your career. Two formats work; use whichever clicks.

**Substitution**, expanding until only arithmetic remains (here for
`sum(n) = n + sum(n−1)`, base `sum(0) = 0`):

```
sum(3)
= 3 + sum(2)
= 3 + (2 + sum(1))
= 3 + (2 + (1 + sum(0)))
= 3 + (2 + (1 + 0))
= 6
```

**Stack drawing**, one box per frame with its own variables, like the
`factorial(3)` picture above — push boxes while winding, then annotate each
with its return value while unwinding.

Do it now, on paper, for `factorial(4)`: draw all five frames and the value
each one returns. Your tutor will ask you to do this from memory — it's
objective five, and it's the difference between using recursion and hoping.

## Recursion follows the shape of the data

Remember the arrays-and-linked-lists lesson: a linked list is *itself* defined
recursively — a list is either empty, or a node followed by a list. Data
defined recursively practically begs for recursive code: the base case handles
"empty", the recursive case handles "a node plus the rest".

The same goes for anything **tree-shaped** — data that nests inside itself to
unknown depth: folders containing folders, comments with replies, a company
org chart. A node with children:

```
sum_tree(node):
    if node is missing:              # base case
        return 0
    total = node.value
    for each child in node.children:
        total += sum_tree(child)     # one recursive call per child
    return total
```

A plain loop handles *flat* sequences beautifully, but nesting of unknown
depth is where loops get awkward and recursion reads like the data's own
definition. Later in this stage, a whole lesson digs into trees as a data
structure — this lesson only needs the shape.

## Two calls per level: the cost

Some definitions recurse *twice*: `fib(n) = fib(n−1) + fib(n−2)` (base cases
`fib(0) = 0`, `fib(1) = 1`). Elegant — and a trap. Sketch the calls for
`fib(5)`:

```
fib(5)
├── fib(4)
│   ├── fib(3)
│   │   ├── fib(2) …
│   │   └── fib(1)
│   └── fib(2) …
└── fib(3)
    ├── fib(2) …
    └── fib(1)
```

`fib(3)` is computed twice, `fib(2)` three times, and it snowballs: the call
tree roughly doubles per level, so the running time grows exponentially —
O(2ⁿ) in the growth classes from the Big-O lesson. `fib(50)` this way is over
a *trillion* calls. Recursion made the code match the definition, but the
definition repeats work. The dynamic-programming lesson at the end of this
stage teaches the fix; for now, just learn to *see* the repeated subtrees.

## Recursion vs iteration

Every recursive solution can be rewritten iteratively:

- **Simple shrink-by-one recursion** becomes a plain loop with an
  accumulator. Factorial needs no stack at all:

  > **In Go:**
  >
  > ```go
  > func factorial(n int) int {
  > 	result := 1
  > 	for i := 2; i <= n; i++ {
  > 		result *= i
  > 	}
  > 	return result
  > }
  > ```

- **The general case** becomes a loop plus an **explicit stack** — the data
  structure from last lesson, now managed by you: push the work you would
  have recursed into, loop while the stack is non-empty. Your explicit stack
  lives in ordinary memory (the heap), so its depth is bounded by RAM, not by
  the call-stack limit.

When is iteration preferable?

- **Unbounded or user-controlled depth** — recursing over input a user can
  make arbitrarily deep is a crash waiting to happen.
- **Flat, linear data** — a loop over a slice or list is simpler and cheaper
  than frames.
- **No tail-call optimization** — some languages reuse the current frame when
  the recursive call is the very last act. If yours doesn't, "tail recursion"
  still costs one frame per level.

> **In Go:** Go does *not* perform tail-call optimization — every recursive
> call consumes a frame, tail position or not. That's why Go code iterates
> over linear structures and typically saves recursion for nested ones.

When is recursion preferable? Naturally nested data (this lesson's trees),
and divide-and-conquer algorithms — the sorting lesson up next runs on
exactly that idea.

## Exercise

Open [`exercise/`](exercise/) — a ready Go module with:

- `recursion.go` — a `Node` type and four functions with `TODO`s:
  `Factorial`, `Reverse`, `Sum` (all recursive), and `SumIterative`
  (a loop with an explicit stack).
- `recursion_test.go` — the specification. **Read it first**, including the
  comment on the deep-tree test.

Acceptance criteria:

1. `Factorial(n)` returns n! for n ≥ 0, recursively; `Factorial(0)` is 1.
2. `Reverse` reverses a string *by runes*, recursively:
   `Reverse("héllo")` is `"olléh"` (remember S1 strings-runes: slicing a
   string slices bytes, so convert to `[]rune` first).
3. `Sum` recursively totals every `Value` in a tree of `Node`s;
   `Sum(nil)` is 0.
4. `SumIterative` returns the same totals with **no recursion** — a loop and
   a slice used as a stack — and survives the million-node deep-chain test,
   which lowers the stack limit so a recursive version dies with
   `fatal error: stack overflow`. If your whole test run aborts with that
   message, it *is* your feedback: `SumIterative` is still recursing.
5. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They fail before you write code — make them green, then trace `Factorial(4)`
on paper for the quiz.

## Further reading

- [SICP §1.2 — Procedures and the Processes They Generate](https://sarabander.github.io/sicp/html/1_002e2.xhtml)
- [Khan Academy — Recursive algorithms](https://www.khanacademy.org/computing/computer-science/algorithms/recursive-algorithms/a/recursion)
- [Wikipedia — Call stack](https://en.wikipedia.org/wiki/Call_stack)
