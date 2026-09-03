# Tutor notes — Problem-Solving Patterns

## Where the learner is

End of S2 — the stage capstone. They have implemented every structure and
algorithm in the stage, but always with the pattern named for them in the
lesson title. This is the first time they choose the tool themselves. Expect
solid Go mechanics and hesitation at the blank page; coach the recognition
step (pattern → invariant → bound), not syntax. The complexity probes make
"right answer, wrong growth" a first-class failure — treat probe failures as
teaching moments, not test flakiness.

## Common misconceptions

- **Two pointers on unsorted input** — the discard argument collapses
  without order. Ask for a concrete counterexample (e.g. `[1, 9, 2]`,
  target 3: converging pointers walk past the answer).
- **Recomputing window sums** — O(n·k) "works" on the small cases, then the
  timed test fails. Some learners assume the probe is broken; point out the
  message names the growth class on purpose.
- **The `abba` bug** — moving `left` *backwards* because the map holds a
  stale index left of the window. The "window must not reopen" test exists
  for exactly this; have them trace `abba` by hand.
- **Bytes vs runes** — indexing the string directly makes `"día"` return 4.
  Send them back to S1 strings-runes, not forward.
- **Counters for brackets** — one counter per kind passes `"([)]"`. Ordering
  information is gone; only LIFO keeps it (quiz stretch question).
- **Islands without visited-marking** — infinite loops or double counting.
  Also diagonal adjacency: the "diagonals do not connect" case is there to
  settle the argument.
- **Greedy for the non-adjacent sum** — "take every other element" fails
  `[3, 2, 7, 10]` (answer 13 takes 3 and 10, skipping two in a row);
  "always take the max first" reasoning also collapses under the
  take-or-skip recurrence. If they hand-wave greedy, run them through that
  case.
- **Allocation probes feel arbitrary** — explain that `AllocsPerRun`
  counting a map or fresh slice is the *space* half of the stated bound;
  O(1) extra space is spec, not style.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go
deeper):

- "Walk me through the proof that PairSum never skips a valid pair when it
  decrements `hi`."
- "The variable window's left edge moves too. Why is the whole thing O(n)
  and not O(n²)? Where else this stage did we make an amortized argument?"
  (Append's array doubling.)
- For each capstone function: "Name the pattern *before* you explain the
  code. What invariant does your state maintain? What bound did the tests
  enforce, and how?"
- "Which single word in a problem statement separates sliding window from
  DP?" (Contiguous vs. any-subset/count-the-ways.)
- "CountIslands mutates its input. When is that acceptable, and what would
  you change if the caller needed the grid intact?" (Document it, or pay
  O(rows·cols) for a visited set — a space trade-off they should name.)

## Grading rubric

- **A** — All tests including the time and allocation probes pass; for every
  function the learner names pattern, invariant, and Big-O unprompted; the
  capstone choices come with a rejected alternative ("hash map also works
  for PairSum but costs O(n) space").
- **B** — Everything passes, but a probe was fixed only by following its
  failure message verbatim, or justifications are correct only when
  prompted per function.
- **C** — Correctness tests pass but a probe needed the tutor to walk the
  shape, or pattern names are recited without the invariant argument. Pass
  only if time-boxed remediation lands; otherwise another iteration with a
  fresh problem.
- **Fail** — Probes failing (right answers, wrong growth) or patterns
  misidentified across the board. The stage goal is recognition; do not
  advance on memorized code.

## Remediation ladder

1. "Read the failing probe's message aloud — it names the growth class and
   points a direction. What is it telling you to stop doing?"
2. "How many times does your code touch each element? Count it for a
   10-element input, then extrapolate."
3. Point to the specific LESSON.md section: the discard proof, the slide
   step, the `last[x] >= left` guard, or the field-guide table row for the
   capstone function in question.
4. Whiteboard the invariant together — draw the window over `abba` or the
   stack for `([)]` — then let them translate the picture into code alone.

## After passing

Preview: "S2 is done — back to Go proper. S3 opens with interfaces, the
feature the entire standard library is designed around."
