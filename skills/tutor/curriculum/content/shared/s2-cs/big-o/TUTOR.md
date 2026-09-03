# Tutor notes — Big-O: Time & Space

## Where the learner is

First lesson of the CS stage, straight off the S1 capstone. They write
beginner Go fluently (slices, maps, table-driven tests) but have *zero*
algorithmics vocabulary — "Big-O", "logarithm", "auxiliary space" are all new
words. Expect math anxiety around log: defuse it immediately with "log n is
just the number of halvings from n down to 1"; no other math is needed. This
lesson is the measuring stick for the whole stage — time spent making it
solid pays off eleven more times.

## Common misconceptions

- **"Big-O measures speed in seconds."** It measures growth of step count.
  If they say "O(n) means fast", reframe: it means *doubling the input
  doubles the work* — on any machine.
- **"O(1) is always faster than O(n)."** Only eventually. Constants matter
  at small n; Big-O says who wins as n grows.
- **"Dropping constants means constants never matter."** They matter in
  practice, just not to the growth *class*. Modeling choice, not physics.
- **"Nested loops = O(n²)."** `FirstTen` in Part A is the trap: a
  constant-bounded inner loop keeps it linear. Both loops must scale with n
  for the multiplication to bite.
- **"Two loops in a row = O(2n), which is worse than O(n)."** `SumTwice` is
  the other trap: sequential loops add, and 2n is still O(n).
- **"The early return makes HasPairSum faster than O(n²)."** Big-O rates the
  worst case; the no-match input runs every pair.
- **Counting the input as space.** Auxiliary space is only what the function
  allocates on top; a caller-passed slice is free.
- **"Maps are free."** Map operations are roughly O(1) *on faith* until the
  Hash Tables lesson — but they allocate, which is exactly why
  `HasDuplicateSorted` forbids them.
- **`xs[i:j]` allocates a new array.** It shares the backing array (S1
  slices lesson); good moment to reinforce.

## Grilling points

Ask in the learner's own words (quiz.json has the core set; these dig into
their exercise):

- "Your `HasDuplicate` uses a map. What exactly did you pay for the speed,
  and where does the test catch someone who won't pay?"
- "Why isn't `FirstTen` quadratic? Now change the 10 to `len(xs)/2` — what
  is it then, and why?" (n × n/2 → O(n²); constants drop, variables don't.)
- "The linear-time tests use a wall-clock guard. Why is that a smell test
  for O(n) rather than a proof?" (Timing measures one n on one machine;
  Big-O is about growth across all n.)
- "`HasDuplicateSorted` gets O(n) time *and* O(1) space — what did the
  sortedness buy, and what would sorting first have cost?" (Preview only —
  sorting's price tag is a later lesson.)
- "When would you deliberately ship the O(n²) pair-comparison duplicate
  check?" (Tiny n, zero allocations, simplicity; there must be a reason,
  stated out loud.)
- "Rank O(n log n) against O(n) and O(n²) at n = one million — roughly how
  many steps each?"

## Grading rubric

- **A** — All tests pass; Part A answers came from counting iterations (ask
  them to walk `HalvingPerItem`); Part B is clean single-pass code
  (`HasDuplicateSorted` compares neighbors, `CountCommon` deletes or
  otherwise dedupes correctly); learner articulates the time-space trade-off
  between the two duplicate checks unprompted.
- **B** — Tests pass but a Part A trap needed a nudge, or Part B works with
  minor awkwardness (e.g. `CountCommon` builds a second map to dedupe —
  correct, still linear, worth discussing); trade-off explained when asked.
- **C** — Tests pass only after heavy hinting, or the learner can't explain
  *why* the map version is O(n) or what auxiliary space means. Time-boxed
  remediation before advancing; this vocabulary is load-bearing for the
  whole stage.
- **Fail** — Complexities filled by trial-and-error against the test, or
  Part B copied without being able to trace a run. Reset and remediate;
  don't advance.

## Remediation ladder

1. "Pick n = 8 and trace the function on paper. How many times does the
   innermost line actually run?"
2. "Now double n to 16. Did the count double, quadruple, or grow by one?
   That answer *is* the complexity class."
3. Point them back to the four loop rules in the lesson (single = n,
   sequential = add, nested = multiply, halving = log) and ask which rule
   fits the function in front of them.
4. Walk one function or one Part B solution shape together verbally — "one
   pass; before looking at each value, ask: have I seen it before?" — then
   have them write the code and do the remaining ones alone.

## After passing

Preview: "Next you meet your first real data structures — arrays and linked
lists — and Big-O becomes the tool you use to compare them honestly."
