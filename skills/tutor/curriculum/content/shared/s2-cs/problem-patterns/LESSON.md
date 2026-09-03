# Problem-Solving Patterns

> `shared.cs.problem-patterns` · ~4-6h · Stage: CS Fundamentals

## Objectives

By the end of this lesson you can:

- Implement the two-pointers technique to solve pair-sum and in-place array
  problems in O(n) that pass the provided tests.
- Implement the sliding-window technique for fixed and variable window sizes
  on subarray/substring problems.
- Recognize which pattern (two pointers, sliding window, hashing, stack,
  BFS/DFS, DP) fits a fresh problem statement and justify the choice.
- Solve the mixed problem set capstone by combining data structures and
  patterns from across the stage within the stated complexity bounds.

## The toolbox is full — now pick the right tool

Eleven lessons ago you could barely analyze a loop. Now you can build a hash
table, sort and search, walk trees and graphs, and cache overlapping
subproblems. This capstone adds two last techniques that working programmers
reach for constantly — two pointers and sliding windows — and then trains the
harder skill: real problems never arrive labeled "use BFS here". You get a
sentence of English and must recognize the shape yourself.

Recognition is not guessing. Every choice is defended with the same
three-part argument you have practiced all stage: what **state** do I keep,
what **invariant** does it maintain, and what does that buy me in **Big-O**
terms? Keep that trio in mind — your tutor will ask for it, per problem.

## Two pointers, flavor one: converging

Problem: in a **sorted** array, find two elements that sum to a target.

Brute force checks every pair — the nested loop from the Big-O lesson, O(n²).
The hash-tables lesson does it in O(n) time but O(n) space: scan once, asking
the map "have I seen `target − x`?". Sorted input licenses something better:
one index at each end, walking toward each other.

```
pair_sum(nums, target):            # nums sorted ascending
    lo = 0, hi = last index
    while lo < hi:
        sum = nums[lo] + nums[hi]
        if sum == target: return (lo, hi)
        if sum <  target: lo = lo + 1
        else:             hi = hi - 1
    return "no pair"
```

Each step permanently discards one element. Why is that safe? Say
`sum < target`. `nums[hi]` is the **largest** element still in play, so if
`nums[lo]` falls short paired with it, `nums[lo]` falls short paired with
anything — discard `lo` forever. The `sum > target` case is the mirror image:
`nums[lo]` is the smallest partner available, so `nums[hi]` overshoots with
everyone. One discard per step, n elements: **O(n) time, O(1) space** — and
the whole argument rests on sortedness. Shuffle the input and the discard
logic collapses; that is the trade you weigh against the hash-map version.

## Two pointers, flavor two: reader and writer

The converging flavor closes in from both ends. The second flavor sends two
indexes the *same* direction at different speeds — most often to rewrite an
array **in place**. Classic: remove duplicates from a sorted array.

```
remove_duplicates(nums):           # sorted, so equal values are adjacent
    if nums is empty: return 0
    w = 1                          # write index
    for r from 1 to last index:    # read index
        if nums[r] != nums[w-1]:   # a value not yet kept
            nums[w] = nums[r]
            w = w + 1
    return w                       # the first w slots are the answer
```

The invariant that makes this correct: at every step, `nums[0..w)` is the
finished, deduplicated output so far, and `r ≥ w` — the reader stays ahead of
the writer, so nothing is overwritten before it has been read. One pass, no
second array: O(n) time, O(1) extra space.

> **In Go:** remember from the arrays-and-linked-lists lesson that a slice is
> a small header pointing at a backing array. Writing `nums[w] = …` inside
> the function writes to the *caller's* backing array — that is what "in
> place" means here. The convention is to return the new length `k` and let
> the caller keep working with `nums[:k]`.

## Sliding window, fixed size

Problem: given daily sales, find the best 7-day stretch. Any question about
the best **contiguous run of exactly k elements** is this shape.

The naive version sums each window from scratch: n windows × k additions =
O(n·k). But adjacent windows overlap in all but two elements — so don't
rebuild the sum, *slide* it:

```
max_window_sum(nums, k):
    sum  = nums[0] + … + nums[k-1]         # first window, computed once
    best = sum
    for i from k to last index:
        sum  = sum + nums[i] - nums[i-k]   # one enters, one leaves
        best = max(best, sum)
    return best
```

Constant work per step, so O(n) no matter how large k is. This
delta-update trick — pay only for what changed — shows up far beyond
arrays; it is worth recognizing on sight.

## Sliding window, variable size

Now the window's size is part of the question: *the longest substring with no
repeated character*. Grow the window rightward one element at a time; when
the new element breaks the "all distinct" rule, move the left edge just far
enough to restore it. The bookkeeping — "where did I last see this element?"
— is a job for the hash map you built earlier in the stage:

```
longest_unique(items):
    last = empty map               # element -> index of most recent sighting
    left = 0, best = 0
    for right from 0 to last index:
        x = items[right]
        if x in last and last[x] >= left:
            left = last[x] + 1     # jump past the previous sighting
        last[x] = right
        best = max(best, right - left + 1)
    return best
```

Two subtleties earn full marks. First, the `last[x] >= left` guard: the map
may remember a sighting from *before* the current window — jumping back to it
would silently widen the window (the tests catch exactly this). Second, the
cost: "two nested-looking movements" tempts people to say O(n²), but `left`
only ever moves right. Across the whole run each index enters the window
once and is passed by `left` at most once — about 2n steps total. That style
of argument — total work across all iterations, not worst case per iteration
— is called **amortized** analysis, and you already used it when you analyzed
append's occasional array-doubling in the arrays lesson.

> **In Go:** substring problems mean runes, not bytes — the strings-runes
> lesson applies in full. Convert once with `[]rune(s)` and index that, and
> keep the sightings in a `map[rune]int`.

## A field guide to the patterns

The cue is almost always in the problem statement. This table is the lesson:

| The problem says… | Reach for | Because |
|---|---|---|
| sorted input; find a pair / meet a target | two pointers (converging) | order lets every comparison discard an element for good |
| rewrite or compact an array, "in place", O(1) space | two pointers (reader/writer) | prefix invariant, one pass, no second array |
| best/longest/shortest **contiguous** subarray or substring | sliding window | each element enters and leaves the window once |
| "have I seen this?", counts, frequencies | hash map / set | O(1) average lookup |
| nested things, matching openers/closers, most-recent-first | stack | LIFO mirrors nesting |
| grid or network: what's reachable, connected regions, fewest steps | BFS / DFS | BFS explores in layers (fewest steps); DFS floods a region |
| best result from a sequence of take-or-skip choices; count the ways | dynamic programming | overlapping subproblems, solved once |
| repeatedly need the current min/max, top-k | heap | the extreme is always at the root |
| sorted input; "is X present / where does it belong" | binary search | halve the space per probe |

Practice reading a fresh statement against it: *"longest stretch of days
with no repeated visitor"*. "Stretch of days" — contiguous, so a window.
"Longest" — variable size. "No repeated" — distinctness, so a hash set does
the bookkeeping. Pattern chosen, bound predicted (O(n)), and you have not
written a line of code yet. That is the order of operations from now on:
**pattern first, invariant second, code last.**

When two patterns both seem to fit, the tie-breaker is usually one word.
"Contiguous" points at windows; "any subset" or "count the ways" points at
DP; "fewest moves" points at BFS. Say your justification out loud — if it
does not mention an invariant or a bound, it is a hunch, not a choice.

## Exercise

Open [`exercise/`](exercise/) — a ready Go module with two problem files:

- `patterns.go` — the two new techniques, guided: `PairSum`,
  `RemoveDuplicates` (two pointers), `MaxWindowSum`, `LongestUniqueRun`
  (sliding window).
- `capstone.go` — the mixed set, **patterns deliberately not named**:
  `BalancedBrackets`, `CountIslands`, `MaxNonAdjacentSum`. Before coding
  each one, write down which pattern you chose and why — your tutor will ask
  for exactly that justification.
- `patterns_test.go`, `capstone_test.go` — the specification. **Read them
  first.**

The stated complexity bounds are part of the spec, and the tests enforce
them the way the Big-O lesson did: some feed inputs in the hundreds of
thousands under a generous time guard, others count memory allocations. A
right-answer-wrong-growth solution fails with a message telling you which
direction to think.

Acceptance criteria:

1. `PairSum(nums, target)` on an ascending sorted slice returns indexes
   `i < j` with `nums[i]+nums[j] == target` and `true`, or `(0, 0, false)`
   when no pair exists — in O(n) time and O(1) extra space (no map: the
   tests time a 2-million-element input and count allocations).
2. `RemoveDuplicates(nums)` compacts a sorted slice in place and returns
   `k`; the first `k` elements are the distinct values in order — O(n) time,
   zero allocations.
3. `MaxWindowSum(nums, k)` returns the best k-window sum and `true`, or
   `(0, false)` when `k <= 0` or `k > len(nums)` — O(n) regardless of k.
4. `LongestUniqueRun(s)` returns the length **in runes** of the longest run
   of distinct runes — O(n); `"día"` has answer 3, not 4.
5. `BalancedBrackets(s)` checks `()`, `[]`, `{}` nesting; other runes are
   ignored; interleavings like `([)]` are rejected.
6. `CountIslands(grid)` counts 4-directionally connected groups of `'#'` in
   a grid of `'#'` and `'.'`; it may mark the grid in place as it works.
7. `MaxNonAdjacentSum(nums)` returns the best sum of non-negative values
   with no two adjacent picks — O(n) time, O(1) extra space (the tests run a
   million elements and count allocations).
8. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They fail before you write code. If a run dies with
`fatal error: stack overflow`, that *is* feedback — you met this message in
the recursion lesson, and it means unbounded recursion on a huge input.

## Further reading

- [Competitive Programmer's Handbook — ch. 8, two pointers & sliding window](https://cses.fi/book/book.pdf)
- [USACO Guide — Two Pointers](https://usaco.guide/silver/two-pointers)
- [Go blog — Slices: usage and internals](https://go.dev/blog/slices)
