# Tutor notes — Sorting

## Where the learner is

Mid-S2. They can analyze loops with Big-O, implemented linked lists, stacks,
queues, and a hash table, and just finished recursion — merge sort is their
first *payoff* for recursion, so lean on that connection. They write Go
comfortably at S1 level (slices, structs, table-driven tests) but have never
used `slices.SortFunc` or `cmp.Compare`, and function-values-as-arguments will
feel new; treat the comparison function as "a rule you hand to the library,"
not as a functional-programming lesson (closures get proper treatment in S3).
Searching, trees, and heaps are still ahead — don't reference binary search or
heapsort mechanics.

## Common misconceptions

- **"O(n²) means insertion sort is always slower."** No — for tiny or
  nearly-sorted inputs its low constants and O(n) best case win. If they miss
  this, they'll also miss why stdlib hybrids finish with insertion sort.
- **"Merge sort splits are the hard part."** The split is trivial; *merge* is
  where sorting actually happens. Learners who don't internalize this write a
  merge that re-scans or re-sorts, silently going quadratic — the 500k-element
  test is designed to expose exactly that.
- **"Quicksort is just merge sort in place."** The structural difference:
  quicksort does its work *before* recursing (partition), merge sort *after*
  (merge), and quicksort's split point is uncontrolled — that's the whole
  worst-case story.
- **"Randomized pivots eliminate the O(n²) worst case."** They make it
  vanishingly unlikely, not impossible.
- **"Stability is about correctness of the sort."** An unstable sort is
  perfectly correct by the comparison given; stability only matters for how
  *equal* elements land. Watch for hand-waving here.
- **`InsertionSort` shifting with `>=` instead of `>`** — still sorts ints
  correctly (tests pass) but breaks stability and does needless work; worth a
  grilling question even though the int tests can't catch it.
- **Modifying the input in `MergeSort`** — usually from sorting `nums` in
  place then returning it; the input-untouched test catches this.
- **Using `slices.SortFunc` in `SortByScore`** — the at-scale stability test
  fails; the fix (StableFunc) should come from *them* after re-reading the
  stability section, not from you.

## Grilling points

- "Your merge sort passed 500,000 elements in milliseconds. Walk me through
  why it's O(n log n) — where does the log come from, where does the n come
  from?"
- "Insertion sort was fast on the big sorted slice. What input makes it
  crawl, and exactly how many shifts happen there?" (Reverse order; ~n²/2.)
- "Explain partitioning as if to a colleague. Now feed plain quicksort an
  already-sorted array with first-element pivots — trace what happens to the
  recursion depth."
- "Your `Merge` uses `<=`. Change it to `<` — what still passes, and what
  property just broke?" (All int tests pass; stability of merge sort dies.)
- "In `RankPlayers` you compare `b.Score` against `a.Score`. Why that order?
  What happens if you flip every comparison?"
- "Why does `SortByScore` need `SortStableFunc` but `RankPlayers` doesn't?"
  (Multi-key comparison leaves no ties for stability to matter on — if they
  see this unprompted, that's A-grade understanding.)
- "You need to sort a 10-million-row log file already 99% in arrival order,
  and ties must keep arrival order. Which sort and why?"

## Grading rubric

- **A** — All tests pass; `InsertionSort` shifts (no swap-bubbling), `Merge`
  is a single two-finger pass taking from `a` on ties, `MergeSort` is clean
  recursion; stdlib functions chosen deliberately (can say *why* StableFunc);
  explains all three algorithms' time/space and the quicksort worst case
  without prompting.
- **B** — Tests pass; implementation works but shows rough edges (merge with
  repeated `append`+re-slice, clone-then-sort-in-place mergesort, `>=` shift
  condition) or one wobble in the theory (e.g. hazy on why merge sort is
  O(n log n) *always*).
- **C** — Tests pass only after the remediation ladder, or stability/worst
  case can only be recited, not applied to a scenario. Pass only if a
  time-boxed re-explanation lands; otherwise iterate.
- **Fail** — Timing tests fail (wrong complexity class) and they can't
  diagnose why, or the stdlib section was copied without being able to
  explain what the comparison function returns.

## Remediation ladder

1. "Read the failing test name and message aloud — is it complaining about
   *wrong order*, *modified input*, *too slow*, or *shuffled equals*? Each is
   a different bug."
2. For `InsertionSort`: "Sort 4 playing cards in your hand, narrating every
   move. Now map each move to a line of code — which move is the inner loop?"
3. For `Merge`/`MergeSort`: "Take `[1,4,6]` and `[2,3,5]` on paper, one
   finger on each front. Write the output list one element at a time. Which
   finger moved and why?" Then: "Your `MergeSort` — what does it return when
   the slice has one element? Zero?"
4. For the timing tests: "Your code is *correct* but in the wrong complexity
   class. Find the loop-inside-a-loop (or the merge that re-scans) and count
   its work the way the Big-O lesson taught." For stability: point them back
   to the LESSON.md stability section and the two `slices` functions — let
   them pick.

## After passing

Preview: "Sorted data was the toll — next lesson collects the prize:
searching it in O(log n) by halving, the same shrinking trick merge sort
used, aimed at *finding* instead of ordering."
