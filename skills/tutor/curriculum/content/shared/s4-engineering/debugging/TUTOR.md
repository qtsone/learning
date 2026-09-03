# Tutor notes — Debugging

## Where the learner is

Fourth lesson of the engineering stage, after clean code, code organization,
and TDD. They write intermediate Go, read test failures fluently, know git
from S0, binary search from S2, and met heisenbugs and the race detector in
S3 — bisect and the "breakpoints reshape timing" point both have hooks to
hang on. Two firsts here: first time driving a debugger, and first time
fixing code they didn't write. The exercise inverts their habits — the code
exists, the tests are the evidence — so grade the *method* (the log, the
delve session, the trail) as heavily as the green tests. A learner who fixes
all three bugs by staring hard has not passed this lesson's actual bar.

## Common misconceptions

- **"Debugging = try changes until green."** The tell: a filled-in log with
  hypotheses that were obviously written after the fix. Ask for one ruled-out
  hypothesis per bug; shotgun debuggers have none.
- **"The bug is where the symptom is."** `Median` returns correct values —
  the fault is what it does to its *caller's* slice. If they hunted inside
  the median arithmetic, that's the teachable moment.
- **"Print debugging is unprofessional / the debugger is always better."**
  Both are instruments; the choosing table in the lesson is the point. Probe
  with scenarios until they answer with reasons, not loyalty.
- **"delve needs special build flags."** `dlv test` handles compilation
  itself (it disables optimizations for you); no `-gcflags` incantation
  needed at this level.
- **"git bisect needs to understand the code."** It's pure binary search over
  good→bad; it needs a known-good point and a verdict per step, nothing else.
  Tie it to S2's binary search if it feels magic.
- **Fixing the evidence instead of the code** — editing wants in
  `ledger_test.go`. Criterion 1 forbids it explicitly; treat it as a fail.
- **"Empty-slice behavior is a fourth bug."** `Lowest`/`Median` document that
  callers guarantee non-empty input; the solution's panic on empty is the
  contract, not a fault. Redirect them to the failing tests.

## Grilling points

Ask in the learner's own words (quiz.json has the core set; these dig into
their exercise):

- "Walk me through your `Total` trail: what was your minimal failing input,
  and what did the lengths that *passed* have in common?" (Multiples of four;
  the pattern names the bug.)
- "Show me the delve session: which breakpoint, and what did `print lowest`
  show before the loop's first iteration? Why was that already the answer?"
- "In `TestLowest`, the refund case passed while all-purchases failed. What
  did that pattern rule out before you read a single line of `ledger.go`?"
  (The comparison and loop work; something favors values below zero — points
  straight at the initial value.)
- "`TestMedian` was all green, yet `Median` had a bug. What's the general
  lesson about trusting a function's return value?" (Contracts include side
  effects; correct output ≠ correct behavior.)
- "Your `Total` fix: plain loop or clamped batches — defend the choice, and
  what would make you pick the other?" (Plain loop is simpler and summing
  needs no batches; the clamp is the answer if batching were a real
  constraint of the importer.)
- "Twelve commits, one bad — how many bisect verdicts did the lab need, and
  why that number?" (~log₂ 12 ≈ 3-4; halving, same as S2.)
- Quickfire scenarios: value goes wrong on iteration 8,412 of 10,000; test
  fails only on the CI runner; struct three calls deep holds nonsense; bug
  appeared some time in the last sixty commits. One instrument each, one
  reason each.

## Grading rubric

- **A** — All tests green with `ledger_test.go` untouched; all three fixes
  minimal and idiomatic (`Lowest` seeds from `xs[0]`, `Median` clones before
  sorting, `Total` either de-batched or clamped — bonus if they argue the
  simplification); log shows a real trail per bug including at least one
  ruled-out hypothesis; narrates the delve session confidently; choosing-the-
  instrument scenarios answered with reasons.
- **B** — Tests green, fixes correct but one is heavier than needed (e.g.
  `Median` rebuilt rather than clone-and-sort), or the log is complete but
  thin — hypotheses read post-hoc, minimization skipped for one bug. Delve
  was used but narration is shaky.
- **C** — Tests green but the method is absent: log reconstructed afterwards,
  no ruled-out hypotheses, delve never actually driven, fixes found by
  guess-and-run. The lesson's objective is the method — remediate (pick one
  bug, redo the trail live together) before advancing.
- **Fail** — Tests failing, `ledger_test.go` edited to pass, or the learner
  cannot explain a fix's root cause in one sentence. Reset and remediate.

## Remediation ladder

1. "Read the test output again, slowly. Which cases pass? Say the pattern out
   loud — what does a passing case prove?"
2. "Shrink it: does a five-element slice still fail `Total`? Four? One? What
   is special about the sizes that pass?"
3. "Set a breakpoint on the first line of the failing function (`dlv test`,
   then `break ledger.Lowest`, `continue`, `next`). Before each `next`,
   predict what `print lowest` will show — where does reality first disagree
   with you?"
4. Per-bug nudge, letting them type the fix: `Lowest` — "what is `lowest`
   before the loop begins, and can any purchase ever be below it?"; `Total` —
   "when `i` reaches the last partial batch, is `i+batchSize <= len(xs)`
   still true?"; `Median` — "print the caller's slice before and after the
   call — who reordered it, and what could `Median` sort instead?"

## After passing

Preview: "Next lesson your programs stop forgetting everything at exit — the
relational model and SQL, with a real (embedded) database under your tests."
