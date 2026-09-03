# Big-O: Time & Space

> `shared.cs.big-o` · ~2-3h · Stage: CS Fundamentals

## Objectives

By the end of this lesson you can:

- Explain what Big-O notation measures and why constant factors and
  lower-order terms are dropped.
- Rank the common growth classes (O(1), O(log n), O(n), O(n log n), O(n²),
  O(2ⁿ)) and give a concrete example algorithm for each.
- Derive the time complexity of code with single, nested, and sequential loops
  by counting operations.
- Analyze the space complexity of a function, distinguishing auxiliary space
  from input space.
- Choose between a time-optimal and a space-optimal approach for a given
  problem and justify the trade-off.

## Why we count steps, not seconds

You finished S1 able to make a program *work*. This stage is about making it
work *at scale* — and the first tool is a way to talk about cost.

Seconds are a terrible unit for comparing algorithms. The same code runs at
different speeds on your laptop, a CI runner, and a phone; a compiler upgrade
can shave 20% off overnight. None of that changes the *algorithm*. So instead
of timing code, we count **basic operations** — comparisons, additions,
assignments — as a function of the **input size**, written *n*. The question
Big-O answers is not "how fast is this?" but:

> **When the input grows, how does the work grow?**

An algorithm that does `3n + 20` operations and one that does `n` operations
are, for this question, the same: double the input and both do roughly double
the work. An algorithm that does `n²` operations is fundamentally different:
double the input and it does *four times* the work. That difference in *shape*
is what survives hardware changes, and it is what Big-O captures.

## What Big-O says, precisely enough

Big-O describes the **growth rate of the worst case**, keeping only the term
that dominates for large inputs:

- **Drop lower-order terms.** `n² + 5n + 40` is O(n²). By the time n is
  1,000, the `n²` term is 1,000,000 and the `5n + 40` is noise — the biggest
  term decides the shape of the curve.
- **Drop constant factors.** `3n` and `n` are both O(n). The constant depends
  on hardware and implementation details Big-O deliberately ignores; the
  *growth* is identical — double the input, double the work.
- **Worst case, by default.** A search that can exit early still counts as
  its worst run (the target is last, or absent). You plan bridges for the
  heaviest truck, not the average one.

One honest caveat before you internalize this: dropping constants is a
modeling choice, not a claim that constants never matter. For small inputs, a
"slow" O(n²) algorithm with a tiny constant can beat a "fast" O(n) one with a
big constant. Big-O tells you who wins *eventually*, as n grows — and inputs
in real systems have a habit of growing.

## The growth classes to know cold

Ranked from best to worst. The third column is roughly how many steps each
takes when n = 1,000,000:

| Class | Name | Steps at n = 10⁶ | Example |
|-------|------|------------------|---------|
| O(1) | constant | 1 | read the first element of an array |
| O(log n) | logarithmic | ~20 | binary search a sorted list (later this stage) |
| O(n) | linear | 10⁶ | sum a list; find the max |
| O(n log n) | linearithmic | ~2 × 10⁷ | good general-purpose sorting (later this stage) |
| O(n²) | quadratic | 10¹² | compare every pair of elements |
| O(2ⁿ) | exponential | astronomically large | try every subset of n items |

Intuitions worth keeping:

- **O(log n)** means "each step discards half of what's left." Halving
  1,000,000 hits 1 in about 20 steps — that's why looking up a word in a
  physical dictionary is fast: open in the middle, discard half, repeat. You
  will build the code version (binary search) in the Searching lesson.
- **O(n log n)** is the price of the good sorting algorithms you'll implement
  in the Sorting lesson — for large inputs it behaves much closer to O(n)
  than to O(n²).
- **O(n²)** is the signature of "for every element, look at every element."
  Fine at n = 100 (10,000 steps). Ruinous at n = 1,000,000.
- **O(2ⁿ)** — every extra element *doubles* the work, because each element
  can be in or out of a subset. At n = 50 the universe's patience runs out.
  These problems need cleverness (a taste comes in the Dynamic Programming
  lesson) or approximation.

## Reading complexity off loops

You don't derive Big-O with math tricks; you count how many times the work
inside the loops runs. Four rules cover almost everything you'll meet:

**1. A single loop over the input is O(n).**

```text
for each x in list:      # runs n times
    total = total + x    # O(1) work each time  → O(n)
```

**2. Sequential loops add — and addition keeps only the biggest term.**

```text
for each x in list: ...   # n steps
for each x in list: ...   # n more steps
```

That's `n + n = 2n`, and 2n is O(n). One pass or two passes — same class.
A linear loop followed by a quadratic one is `n + n²` = O(n²).

**3. Nested loops multiply.**

```text
for i from 0 to n-1:         # n times
    for j from i+1 to n-1:   # up to n times each
        compare list[i], list[j]
```

Inner work runs about n × n/2 = n²/2 times → drop the ½ → O(n²). Watch the
bounds, though: if the inner loop runs a *constant* number of times (say,
always at most 10), the multiplication is n × 10 — still O(n). Nesting alone
doesn't make code quadratic; *both* loops must scale with the input.

**4. A loop that halves its range each step is O(log n).**

```text
while n > 1:
    n = n / 2      # 1,000,000 → 500,000 → … → 1 in ~20 steps
```

And combinations compose: a halving loop *inside* a full loop over the input
is n × log n → O(n log n).

In Go, rule 3's shape looks like this — the pair-comparison loop you have
already written by instinct:

```go
for i := 0; i < len(xs); i++ {
	for j := i + 1; j < len(xs); j++ {
		if xs[i] == xs[j] {
			return true // early exit — but the worst case still visits every pair
		}
	}
}
```

The `return true` does not change the class: Big-O rates the worst case, and
the worst case (no duplicates) runs the full n²/2 comparisons.

## Space complexity

The same notation measures memory. One convention matters: **the input
doesn't count**. You were handed the list; what you're charged for is the
**auxiliary space** — the extra memory *your* algorithm allocates on top.

- A function that scans a list keeping one running total uses O(1) auxiliary
  space — a couple of variables, regardless of n.
- A function that builds a lookup table with one entry per element uses O(n)
  auxiliary space.
- A function that builds a table of every *pair* would use O(n²) — usually a
  design smell.

In Go: `make(map[int]bool)` and `append` that grows a slice allocate memory
proportional to what you put in them — that's auxiliary space. Indexing
`xs[i]` or slicing `xs[2:5]` allocates nothing: a slice expression shares the
backing array (remember the S1 slices lesson). The Go test helper
`testing.AllocsPerRun` counts allocations, which makes "O(1) auxiliary
space" something a test can actually check — the exercise uses it.

## The time-space trade-off

Time and space are a budget you shift, not two scores you max out. The
classic example — and your exercise — is duplicate detection:

- **Time-optimal:** one pass, remembering every value seen so far in a
  lookup table. O(n) time, O(n) auxiliary space. In Go the table is a map —
  and you may take "map operations cost roughly O(1)" on faith until the
  Hash Tables lesson, where you build one and see why.
- **Space-optimal (general case):** compare every pair. O(n²) time, O(1)
  space. Slow, but allocates nothing.
- **The structure bonus:** if the input is *already sorted*, equal values
  sit next to each other — one pass comparing neighbors gives O(n) time
  *and* O(1) space. Knowing something about your data can beat both generic
  options. (What sorting costs is the Sorting lesson's business.)

How to choose: on a server with memory to spare, buy speed with space. On a
tiny device, or when n is small enough that n² is cheap, the frugal version
wins. There is no universal answer — the skill is naming what each option
costs and picking deliberately. You'll be asked to defend a choice like this
in review; "it felt faster" is not a defense, "O(n) time for O(n) space, and
n here is millions" is.

## Exercise

Open [`exercise/`](exercise/) — a Go module with two parts.

**Part A — analyze** (`analyze.go`): eight small functions are already
written; your job is to *read* them and record each one's time complexity in
the `Complexities` map, using the provided constants (`O1`, `OLogN`, `ON`,
`ONLogN`, `ON2`, `O2N`). Count loop iterations; don't guess from the shape —
two of them are deliberate traps.

**Part B — implement** (`implement.go`): three functions with a required
complexity each:

- `HasDuplicate(xs []int) bool` — true if any value occurs more than once.
  Required: O(n) time; O(n) auxiliary space is the price, pay it.
- `HasDuplicateSorted(xs []int) bool` — same question, but the input is
  sorted ascending. Required: O(n) time **and** O(1) auxiliary space — no
  maps, no new slices.
- `CountCommon(xs, ys []int) int` — how many *distinct* values appear in
  both slices. Required: O(len(xs) + len(ys)) time.

The tests check behavior *and* complexity: large-input tests fail with a
clear message if your code behaves quadratically, and an allocation test
fails if `HasDuplicateSorted` allocates.

Acceptance criteria:

1. Every entry in `Complexities` is one of the six constants and matches the
   function's actual growth class.
2. `HasDuplicate` passes its correctness table and finishes the 300,000-element
   worst case comfortably inside the test's generous time guard.
3. `HasDuplicateSorted` passes its correctness table and
   `testing.AllocsPerRun` reports **zero** allocations.
4. `CountCommon` counts each shared value once (duplicates inside one slice
   don't inflate the count) and stays linear on the large disjoint inputs.
5. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They FAIL on the starter — read the failure messages; they tell you which
part they're probing.

## Further reading

- [Khan Academy — Asymptotic notation](https://www.khanacademy.org/computing/computer-science/algorithms/asymptotic-notation/a/asymptotic-notation)
- [Big-O Cheat Sheet](https://www.bigocheatsheet.com/) — growth classes of the structures and algorithms you'll build this stage
- [pkg.go.dev — testing.AllocsPerRun](https://pkg.go.dev/testing#AllocsPerRun) — the helper the exercise uses to test space behavior
