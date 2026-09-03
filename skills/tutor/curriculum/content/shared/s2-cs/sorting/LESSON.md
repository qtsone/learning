# Sorting

> `shared.cs.sorting` · ~3-4h · Stage: CS Fundamentals

## Objectives

By the end of this lesson you can:

- Implement insertion sort and merge sort that pass the provided tests, and
  explain each algorithm's time and space complexity.
- Explain how quicksort partitions and why its worst case is O(n²) despite an
  O(n log n) average.
- Define sorting stability and give an example where an unstable sort produces
  a wrong-looking result.
- Use the standard library's sort facilities with a custom comparison to sort
  structs by multiple keys.
- Choose an appropriate sort for a given input size and ordering requirement
  and justify the choice.

## Why ordering matters

Unordered data answers only one question cheaply: "give me everything." Ordered
data answers many: what's the smallest, the largest, the median, the
duplicates sitting next to each other? The next lesson shows the biggest prize
of all — searching sorted data in O(log n). Sorting is the toll you pay once so
that every later question gets cheaper.

Sorting is also the classic playground for algorithm analysis. You already
have the two tools you need: Big-O from the start of this stage, and recursion
from the previous lesson. Every sort below is judged the same way — how many
element *comparisons and moves* does it make as n grows, and how much extra
memory does it need beyond the input?

## Insertion sort: how humans sort cards

Pick up cards one at a time and slide each one left until it sits in the right
spot among the cards already in your hand. That's insertion sort:

```text
for i from 1 to n-1:
    key = a[i]                 # the card you just picked up
    j = i - 1
    while j >= 0 and a[j] > key:
        a[j+1] = a[j]          # shift bigger cards one step right
        j = j - 1
    a[j+1] = key               # drop the card into the gap
```

The invariant: before iteration i, the prefix `a[0..i-1]` is sorted. Each
iteration grows the sorted prefix by one.

Complexity:

- **Worst case O(n²)** — input in reverse order: every card must travel all
  the way left, so the inner loop runs 1 + 2 + … + (n-1) ≈ n²/2 times. You met
  this triangle-sum in the Big-O lesson.
- **Best case O(n)** — input already sorted: the inner `while` condition is
  false immediately, every time. One comparison per element and done. This
  makes insertion sort *adaptive*: nearly-sorted input is nearly free.
- **Space O(1)** — it shuffles elements inside the array it was given.
  In-place, no allocation.
- **Stable** — the shift condition is strictly `a[j] > key`, so an equal
  element is never jumped over. Equal elements keep their original order
  (more on why that matters below).

O(n²) sounds disqualifying, but the constant factors are tiny — no recursion,
no allocation, cache-friendly sequential access. For small n (a few dozen
elements) insertion sort beats every fancy algorithm, which is why production
sorts use it as their finishing move on small ranges.

## Merge sort: divide, conquer, merge

Recursion's favorite trick from last lesson — solve a big problem by solving
two half-size copies of it — turns sorting into:

```text
merge_sort(a):
    if length(a) <= 1: return a          # base case: trivially sorted
    left  = merge_sort(first half of a)
    right = merge_sort(second half of a)
    return merge(left, right)
```

All the real work happens in `merge`: combine two *already sorted* lists into
one sorted list. Walk both with a finger each, repeatedly taking the smaller
front element:

```text
merge(a, b):
    out = empty list
    while both a and b have elements left:
        if front(a) <= front(b): move front(a) to out
        else:                    move front(b) to out
    append whatever remains of a, then of b
    return out
```

Merge looks at each element once, so merging two lists totalling n elements is
O(n).

Complexity:

- **Time O(n log n) — always.** Halving n until you reach 1 takes log₂ n
  levels of recursion (the same halving argument from the Big-O lesson), and
  each level's merges touch all n elements once: n work × log n levels. No
  bad input exists; reverse-sorted, all-equal, adversarial — all O(n log n).
- **Space O(n)** — merge builds output lists, so you pay a linear amount of
  extra memory. That's the trade: guaranteed time, bought with space.
- **Stable** — as long as `merge` takes from the *left* list on ties
  (`<=`, not `<`). Flip that one comparison and stability silently breaks.

## Quicksort: partition around a pivot

Quicksort also divides and conquers, but does its work *before* recursing
instead of after. Pick one element — the **pivot** — and **partition**:
rearrange the array so everything less than the pivot is to its left and
everything greater is to its right. The pivot is now in its final sorted
position. Recurse on the two sides:

```text
quick_sort(a, lo, hi):
    if lo >= hi: return
    p = partition(a, lo, hi)     # place pivot, return its index
    quick_sort(a, lo, p-1)
    quick_sort(a, p+1, hi)
```

Partitioning a range of n elements is one O(n) sweep. There's no merge step —
after partitioning, the two sides never need to interact again.

So why isn't this simply "merge sort but in-place"? Because the split point is
not chosen — it's wherever the pivot happens to land:

- **Good pivot** (near the median): two roughly equal halves, log n levels,
  **O(n log n)** — the average case, and in practice faster than merge sort
  thanks to in-place partitioning and cache friendliness.
- **Bad pivot** (smallest or largest element): one side is empty, the other
  has n-1 elements. The "halving" degrades to shrinking by one — n levels of
  O(n) partitions: **O(n²)**. With a naive "always pick the first element"
  pivot, the killer input is embarrassingly common: an *already sorted*
  array.

Real implementations defend against this by picking pivots at random or by
median-of-three sampling, making the worst case astronomically unlikely — but
never impossible. That "average O(n log n), worst O(n²)" asterisk is exactly
the kind of thing you're expected to know about a tool you use.

Standard quicksort is **in-place (O(log n) stack space)** but **not stable** —
partitioning swaps elements across long distances and equal elements can leap
over each other.

## Stability: when equal isn't identical

A sort is **stable** if elements that compare equal keep their original
relative order.

For plain numbers, who cares — one 7 is as good as another. Stability matters
when you sort *records* by one field and the rest of the record tags along.

Example: a support ticket queue, already ordered by arrival time. You sort it
by customer name to group each customer's tickets:

- **Stable sort**: within each customer, tickets remain in arrival order.
  Alice's ticket from Monday still precedes her ticket from Wednesday.
- **Unstable sort**: tickets for the same customer come out in arbitrary
  order. Wednesday before Monday. Nothing is "wrong" by the comparison you
  asked for — all Alice tickets compare equal — but the result looks shuffled
  and any downstream logic that relied on arrival order is now corrupted.

Two ways to get trustworthy order for equal elements: use a stable sort (the
previous order survives), or make the comparison itself break ties (compare by
name, *then* by arrival time), after which no two elements compare equal and
stability is moot. You'll do both in the exercise.

Scorecard so far: insertion sort and merge sort are stable; quicksort is not.

## Use the standard library (but know what's inside)

You implement sorts in this lesson to *own* the reasoning — in real code you
call the standard library. Every mainstream language ships a tuned hybrid:
typically a hardened quicksort variant or a merge-sort derivative, switching
to insertion sort on small ranges, with defenses against the quadratic worst
case. It's faster and better tested than anything you'd write in an
afternoon. What the library *cannot* decide for you: what "less than" means
for your data, and whether you need stability. That's your job, expressed as
a comparison function.

In Go:

```go
import (
	"cmp"
	"slices"
)

nums := []int{3, 1, 2}
slices.Sort(nums) // ascending, for ordered types (numbers, strings)

type Player struct {
	Name  string
	Score int
}

// Custom order: score descending, ties broken by name ascending.
slices.SortFunc(players, func(a, b Player) int {
	if c := cmp.Compare(b.Score, a.Score); c != 0 { // b before a: descending
		return c
	}
	return cmp.Compare(a.Name, b.Name)
})
```

The comparison function returns negative when `a` should come first, positive
when `b` should, zero when they're equal — and `cmp.Compare` builds that
three-way answer for any ordered type. Multi-key ordering is just "compare the
first key; on a tie, fall through to the next."

`slices.SortFunc` is *not* stable. When equal elements must keep their input
order, reach for `slices.SortStableFunc` — same signature, stability
guaranteed, slightly slower. (You may also meet the older `sort` package in
existing code; `slices` is the modern surface.)

## Choosing a sort

| Situation | Reach for | Why |
|---|---|---|
| Everyday sorting in a program | stdlib sort | Tuned hybrid, battle-tested; just supply the comparison |
| Equal elements must keep input order | stable sort (stdlib stable variant, or merge sort) | Plain quicksort-family sorts shuffle equals |
| Tiny input (n ≲ 30) or nearly sorted | insertion sort | O(n) on nearly-sorted; lowest constants, in-place |
| Guaranteed O(n log n), adversarial input | merge sort | No quadratic worst case, at the price of O(n) space |
| Memory is the constraint | in-place sort (insertion, quicksort family) | Merge sort's O(n) buffer may not fit |

The interview-and-code-review answer is rarely "the fastest algorithm" — it's
the one whose guarantees match the requirement you were given.

## Exercise

Open [`exercise/`](exercise/) — a Go module with `sorting.go` (your
work sites, marked `TODO`) and `sorting_test.go` (the specification — read it
first). You'll implement the two classics, then use the standard library the
way you would at work.

Acceptance criteria:

1. `InsertionSort(nums []int)` sorts ascending, **in place**. On an
   already-sorted input it must run in linear time — the tests time it on a
   large sorted slice, so a non-adaptive O(n²)-always approach fails.
2. `Merge(a, b []int) []int` merges two sorted slices into one sorted slice,
   taking from `a` on ties.
3. `MergeSort(nums []int) []int` returns a **new** sorted slice and leaves its
   input untouched. The tests run it on 500,000 elements with a generous
   deadline — O(n log n) passes with ease, O(n²) does not.
4. `RankPlayers(players []Player)` sorts in place by `Score` descending, ties
   by `Name` ascending, using `slices.SortFunc` with a custom comparison.
5. `SortByScore(players []Player)` sorts in place by `Score` descending
   **only**, and players with equal scores keep their original order — the
   tests check stability on 1,000 players, so pick your stdlib function
   accordingly.
6. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They fail on the starter — make them green. If a timing test fails, don't
reach for micro-optimizations: it's telling you your algorithm is in the
wrong complexity class.

## Further reading

- [pkg.go.dev/slices](https://pkg.go.dev/slices) — `Sort`, `SortFunc`,
  `SortStableFunc` and friends
- [pkg.go.dev/cmp](https://pkg.go.dev/cmp) — `cmp.Compare` and the ordering
  contract comparison functions follow
- [VisuAlgo — sorting visualizations](https://visualgo.net/en/sorting) — watch
  insertion, merge, and quick sort move real elements; the O(n²) vs O(n log n)
  difference is visceral at n = 50
