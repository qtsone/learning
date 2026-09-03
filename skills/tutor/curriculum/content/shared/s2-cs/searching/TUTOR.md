# Tutor notes — Searching

## Where the learner is

Seventh lesson of S2. They have Big-O vocabulary, know why arrays give O(1)
indexing and linked lists don't, have built a hash table, and just finished
sorting — so "sorted input costs O(n log n)" is fresh. From S1 they write Go
slices and table-driven tests comfortably. Binary search will *look* trivial
to them; the lesson's real content is invariant discipline (half-open
intervals, termination arguments) and the boundary/lower-bound reframing.
Expect the first/last-occurrence variants to be where they actually struggle.

## Common misconceptions

- **"Binary search kind of works on unsorted data"** — it can return a right
  answer by luck, and luck is not correctness. Have them trace
  `[5, 1, 4]`, target 1: the first comparison discards the half holding it.
- **Interval-convention mixing** — half-open `[lo, hi)` with closed-interval
  updates (or vice versa). Symptoms: infinite loop (`lo = mid` when the
  interval has one element) or skipped boundary elements (`hi = mid - 1`
  under half-open). The fix is never a patch; it's picking one convention and
  restating the invariant.
- **"A hung test run means the test is broken"** — it means their loop isn't
  shrinking the interval. Point them at `go test -timeout 10s ./...` and the
  termination argument in LESSON.md.
- **"Overflow can't happen in Go, so `lo + (hi-lo)/2` is pointless"** —
  correct about 64-bit `int` and slice sizes; the habit still matters on
  32-bit targets and in every other language (the nine-year JDK bug). Accept
  `(lo+hi)/2` in their code but make sure they can articulate the risk.
- **First occurrence via "find any match, walk left"** — right answers,
  O(n) worst case. The all-equal million-element test catches it *if* they
  count the walk's probes honestly; if their probe count looks too good for
  the code shape, read the code, not the number.
- **Probe miscounting** — forgetting to count, or double-counting one
  `xs[mid]` read compared twice. Spec: one probe per index inspected.
- **"O(log n) means binary search is always the right choice"** — ignores
  the sorting precondition, its O(n log n) price, and small-n constants.

## Grilling points

- "Why does your loop terminate? Convince me it *cannot* run forever."
  (mid is inside a non-empty interval; both updates strictly shrink it.)
- "State your loop invariant — what is true about `[lo, hi)` at the top of
  every iteration?"
- "Four billion sorted records: how many probes? Same records in a sorted
  linked list?" (~32; then O(n) — random access is a precondition.)
- "You get one search over unsorted data. Sort-then-binary or linear? What if
  it's a thousand searches?" (n log n vs n; the sort amortizes.)
- "You already hold the data in your S2 hash table with O(1) lookup. Name a
  query that still needs sorted data and binary search." (first ≥ x, range,
  nearest — order questions.)
- "How would you find the first index where a yes/no condition flips from no
  to yes, with no target value in sight?" (The predicate/boundary view;
  connect to `sort.Search` and their lower-bound code.)

## Grading rubric

- **A** — All tests pass; one coherent interval convention with the invariant
  stated in their own words; `FirstOccurrence`/`LastOccurrence` are genuine
  boundary searches (shared or mirrored lower-bound logic, no sideways
  walking); probes counted honestly; can do the "million → ~20 probes"
  arithmetic unprompted.
- **B** — Tests pass, but the convention wobbled and was patched into
  correctness, or first/last are two blobs of near-duplicate code they can't
  relate to lower bound, or probe accounting deviates slightly from spec.
  Explanation of sortedness and halving is solid.
- **C** — Tests pass only after heavy hinting, or they cannot explain why
  sorted input is required, or the probe budget was met by adjusting the
  counter rather than the algorithm. Time-boxed remediation before advancing.
- **Fail** — Tests failing, an unresolved infinite loop, or the halving
  argument is clearly not understood (e.g. can't say why half is discarded).

## Remediation ladder

1. "Play the guessing game: I picked a number from 1 to 100; you may only ask
   'is it bigger than …?'. Play three rounds aloud — you already know this
   algorithm. Now map your moves onto lo, hi, and mid."
2. "Write `[lo, hi)` on paper for `[1, 3, 5, 7, 9]`, target 7. After each
   comparison, which half is *proven* target-free, and what are the new lo
   and hi exactly? Is mid still a candidate after the comparison?"
3. "Your run hangs: print lo, hi, mid each iteration for `[1, 3, 5]`, target
   5. Which assignment leaves the interval the same size?" (`lo = mid`.)
4. For first/last: "Stop searching for the *value*. Under each index of
   `[1, 2, 2, 2, 3]` write T or F for `xs[i] >= 2`. Binary-search for where F
   turns to T — when the interval empties, where is lo standing?" Let them
   type the code themselves.

## After passing

Preview: "Next: trees. A binary search tree is this lesson frozen into a data
structure — every node comparison throws away half the tree, no sorted slice
required."
