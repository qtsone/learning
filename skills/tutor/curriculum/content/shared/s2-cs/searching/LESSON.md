# Searching

> `shared.cs.searching` · ~1-2h · Stage: CS Fundamentals

## Objectives

By the end of this lesson you can:

- Explain why binary search requires sorted input and how halving the search
  space yields O(log n).
- Implement binary search correctly, avoiding the classic off-by-one and
  midpoint-overflow mistakes.
- Compare linear and binary search costs and identify the input sizes and
  sortedness conditions where each wins.
- Adapt binary search to variant problems such as finding the first or last
  occurrence of a value.

## The baseline: linear search

The simplest way to find something in a collection is to look at every element
until you hit it:

```
linear-search(xs, target):
    for i from 0 to length(xs) - 1:
        if xs[i] == target: return i
    return -1
```

Linear search makes no demands. The data can be in any order, in an array or a
linked list, and it finds the *first* match for free. Its cost profile, in the
vocabulary of the Big-O lesson: best case O(1) (first element), worst and
average case O(n) — a miss examines every element. For three elements that is
nothing. For a billion, it is a billion comparisons, every single lookup.

Keep linear search in your pocket. It is not the "dumb" option; it is the
*general* option, and later in this lesson you'll see real situations where it
beats the clever one.

## What sortedness buys you

Think of a guessing game: I pick a number between 1 and 100, you ask "is it
bigger than 50?" One answer eliminates *half* the possibilities — not because
you're lucky, but because the numbers are ordered. That is the entire idea of
binary search.

Compare one element in the *middle* of a sorted array against the target:

- If it equals the target — done.
- If it is *less* than the target, the target can't be anywhere to its left:
  everything left is even smaller. Discard that half.
- If it is *greater*, discard the right half.

One comparison, half the candidates gone. Repeat on the surviving half until
you find the target or run out of candidates.

Both preconditions matter, and each earlier lesson supplied one:

- **Sorted order** (the sorting lesson): the discard step is a *proof* —
  "everything left of a smaller element is also smaller" is only true when the
  data is sorted. On unsorted data the discarded half may contain the target,
  and the algorithm confidently returns a wrong answer.
- **Random access** (the arrays & linked lists lesson): "jump to the middle"
  must cost O(1). An array gives you that; a linked list makes you walk to the
  middle, which costs the very O(n) you were trying to avoid.

## Binary search, carefully

Binary search is famously easy to get *almost* right. The discipline that
makes it actually right is choosing an interval convention and stating an
invariant. We use the **half-open interval** `[lo, hi)`: `lo` is included,
`hi` is excluded — the same convention as slicing, where `xs[lo:hi]` has
length `hi - lo`.

```
binary-search(xs, target):            # xs sorted ascending
    lo ← 0
    hi ← length(xs)                   # candidates live in [lo, hi)
    while lo < hi:                    # interval not empty
        mid ← lo + (hi - lo) / 2      # integer division
        if xs[mid] == target: return mid
        else if xs[mid] < target: lo ← mid + 1
        else:                         hi ← mid
    return -1
```

The invariant: *at the top of every iteration, if the target is in the array
at all, its index is inside `[lo, hi)`.* Every line preserves it:

- `xs[mid] < target` → the target is strictly right of `mid`, so the new
  interval is `[mid+1, hi)`. Not `[mid, hi)` — `mid` is ruled out, and keeping
  it is the classic **off-by-one** bug: when the interval shrinks to one
  element, `mid == lo`, and `lo ← mid` makes no progress. Infinite loop.
- `xs[mid] > target` → the target is strictly left, new interval `[lo, mid)`.
  Because `hi` is *excluded*, `hi ← mid` already rules `mid` out. Writing
  `hi ← mid - 1` here is the mirror-image off-by-one: it silently discards an
  index that was never examined.

Termination is now an argument, not a hope: `mid` always lies inside a
non-empty `[lo, hi)`, and both updates produce a strictly smaller interval,
so the loop must reach `lo == hi` — the empty interval — if it never finds
the target.

The midpoint line hides the other classic bug. The obvious `(lo + hi) / 2`
first computes `lo + hi`, and in a language with fixed-size integers that sum
can overflow for large arrays even though the *result* would fit — this
exact bug sat in Java's standard library binary search for nine years.
`lo + (hi - lo) / 2` computes the same value and cannot overflow, since
`hi - lo` never exceeds the array length.

**In Go:** `int` is 64 bits on mainstream platforms, so a slice large enough
to overflow `lo + hi` won't fit in memory — but write the safe form anyway.
It costs nothing, it is correct in every language and on 32-bit targets, and
reviewers recognize it as the hand of someone who knows the history.

```go
for lo < hi {
	mid := lo + (hi-lo)/2
	switch {
	case xs[mid] == target:
		return mid
	case xs[mid] < target:
		lo = mid + 1
	default:
		hi = mid
	}
}
```

## How fast is halving?

Each comparison halves the candidates: n → n/2 → n/4 → … → 1. The number of
halvings until one candidate remains is log₂ n — the logarithmic growth class
from the Big-O lesson:

| elements n    | comparisons (~log₂ n) |
|---------------|-----------------------|
| 1,000         | ~10                   |
| 1,000,000     | ~20                   |
| 1,000,000,000 | ~30                   |

Read that table again. *Doubling* the data adds *one* comparison. A million
elements need about twenty looks — which is why the exercise tests don't just
check your answers; they count how many elements you examined.

## Linear vs binary: who wins when

O(log n) beats O(n), so binary search always wins? No — the comparison has
preconditions and constant factors:

- **Unsorted data, one lookup**: sorting first costs O(n log n) (the sorting
  lesson), which is *more* than one O(n) linear scan. Linear wins.
- **Unsorted data, many lookups**: sort once for O(n log n), then every lookup
  is O(log n). The sort pays for itself after a handful of searches.
- **Tiny n**: below a few dozen elements, both are effectively instant and
  linear search's simplicity — no sortedness contract to maintain, no
  boundary bugs to write — often wins on engineering grounds. Big-O describes
  growth, not small-input speed.
- **Sorted linked list**: binary search is *useless* — no O(1) jump to the
  middle. Linear scan regardless.
- **Hash table already in hand**: exact-match lookups are O(1) (the hash
  tables lesson), beating O(log n). But a hash table scatters keys and
  destroys order: it cannot answer "the first element ≥ x", "everything
  between a and b", or "the closest value". Those are *order* questions, and
  sorted-array-plus-binary-search answers them in O(log n).

## Variants: first and last occurrence

The plain loop returns as soon as it hits *an* equal element. With duplicates
— say `[1, 2, 2, 2, 3]`, target 2 — `mid` may land anywhere in the run, and
which occurrence you get is an accident of the interval sizes. Often the
question is sharper: the *first* occurrence, or the *last*.

The tempting fix — binary-search any match, then walk left one element at a
time — reintroduces O(n): imagine a million-element slice that is *all*
target. The tests in this lesson use exactly that slice, so the walk will not
sneak through.

The real fix changes what you search *for*. Stop looking for the target;
look for a **boundary**. Ask of every index the yes/no question
"is `xs[i] >= target`?" On sorted data the answers form two clean blocks:

```
xs:      1  2  2  2  3      target = 2
>= 2 ?   F  T  T  T  T
            ^ first occurrence = the F→T boundary
```

The answers never go back to F once they turn T — the question is
*monotone* — so you can binary-search for the flip point. Don't return on
equality; keep halving until the interval is empty, and `lo` lands exactly on
the boundary:

```
lower-bound(xs, target):              # first index with xs[i] >= target
    lo ← 0; hi ← length(xs)
    while lo < hi:
        mid ← lo + (hi - lo) / 2
        if xs[mid] < target: lo ← mid + 1
        else:                hi ← mid
    return lo                         # may be length(xs): everything is smaller
```

First occurrence = lower-bound, then check that the index is in range and
actually holds the target. Last occurrence mirrors it: find the first index
with `xs[i] > target` (flip the `<` to `<=`); the answer, if it exists, sits
one to the left. Trace both on `[1, 2, 2, 2, 3]` on paper before you code.

The boundary view is the version of binary search you will use most in
practice: any monotone yes/no question over an ordered range — "first commit
where the build breaks", "smallest capacity that fits the load" — is
searchable this way, no explicit target value required.

**In Go:** the standard library ships both flavors, and both return the
*boundary*, not "any match":

```go
i, found := slices.BinarySearch(xs, target)      // i = lower bound; found = presence
j := sort.Search(len(xs), func(i int) bool {     // first i where the predicate
	return xs[i] >= target                       // flips to true
})
```

In real code you reach for these. In this exercise you build them from
scratch — owning the invariant is the point.

## Exercise

Open [`exercise/`](exercise/) — a Go module with `search.go` (four
`TODO`s) and `search_test.go` (the specification — read it first).

Every function returns two values: the `index` it found (or -1) and `probes`
— how many elements of the slice it examined. Count one probe each time you
inspect an element at some index. The tests check *what* you found **and**
*how much work it took*: a correct answer found the slow way still fails.

Acceptance criteria:

1. `LinearSearch` returns the index of the **first** match or -1, works on
   unsorted input, and stops at the first match: a hit at index i costs
   exactly i+1 probes, a miss costs len(xs) probes.
2. `BinarySearch` returns the index of *some* occurrence of the target in a
   sorted slice, or -1. The empty slice returns (-1, 0).
3. `FirstOccurrence` and `LastOccurrence` return the first/last index of the
   target, or -1.
4. On a ~million-element slice, `BinarySearch`, `FirstOccurrence`, and
   `LastOccurrence` each stay within the tests' probe budget (64 — honest
   halving needs about 21).
5. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They fail on the starter — make them green. One warning: the classic
off-by-one bug loops forever, which looks like a *hung* test run. If nothing
prints for seconds, interrupt it (Ctrl-C) and run
`go test -timeout 10s ./...` instead — a shrinking-interval argument, not
patience, is the fix.

## Further reading

- [pkg.go.dev — slices.BinarySearch](https://pkg.go.dev/slices#BinarySearch)
- [pkg.go.dev — sort.Search](https://pkg.go.dev/sort#Search) — the predicate
  form, with the boundary-search examples
- [Google Research — Nearly All Binary Searches and Mergesorts are Broken](https://research.google/blog/extra-extra-read-all-about-it-nearly-all-binary-searches-and-mergesorts-are-broken/)
  — the story of the overflow bug that hid in Java's stdlib for nine years
