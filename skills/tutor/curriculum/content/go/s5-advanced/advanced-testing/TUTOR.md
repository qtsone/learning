# Tutor notes — Advanced Testing

## Where the learner is

Late S5. They have written table-driven tests since S1, done red-green-refactor
and the doubles taxonomy in S4, and spent this stage under the hood (scheduler,
GC, memory model, profiling). They already know `httptest`, `errors.Is/As`,
`t.Run`, and the discipline of "the test file is the spec". What is new here is
*owning* the test infrastructure: writing a golden helper, a fixture
constructor, a fake clock, and a property that a machine will attack.

This is also the lesson where the exercise deliberately asks them to write code
inside `_test.go` files. Expect friction ("am I allowed to edit the tests?") —
yes, the two TODO helpers, and only those.

The exercise is four small problems, not one big one. If they stall, move them
to the next part rather than letting the parser eat the session; the parts are
independent.

## Common misconceptions

- **"Fuzzing runs in CI."** It does not, and the seed corpus does. Make sure
  they can say which command runs which corpus. A learner who put
  `-fuzz` in a CI job has missed the point about unbounded, non-deterministic
  work.
- **"The fuzzer generates random input."** Coverage-guided mutation is the
  whole trick; random bytes would never get past field-count validation. Ask
  what the `new interesting:` counter in the output means.
- **"No panic" is a property.** The weakest possible one. Push until they can
  state the round-trip property in words.
- **Golden files as a rubber stamp.** The classic failure: run `-update`,
  green, ship. Ask what the diff looked like — if they never looked, that is
  the finding.
- **`-update` declared in a non-test file.** Then the production binary grows
  a `-update` flag. Small, but it shows they have not internalized what
  `_test.go` means.
- **Testing time by sleeping.** "I slept 350ms and checked elapsed > 300ms"
  fails on a loaded CI box and under `-race`. The fake clock is not an
  optimization, it is what makes the assertion possible at all.
- **Shared fixtures.** A single package-level `*Store` reused by every test
  passes until someone runs `-shuffle=on`. `TestStoreFixturesAreIsolated`
  exists to catch it; make sure they understand *why* it caught it.
- **`defer` inside a helper.** `newTestStore` cannot `defer store.Close()` —
  it would close before the test body runs. This is the single most instructive
  bug in the exercise; let them hit it.
- **Coverage as a target.** If they say "we should get to 90%", ask what a
  test that raises coverage without asserting anything is worth.

## Grilling points

- "Your fuzz body returns early when `ParseLine` errors. Why is that not
  cheating?" (Rejecting input is allowed; the property is about *accepted*
  input. A parser that rejects everything would fail the deterministic tests.)
- "The fuzzer found nothing in 30 seconds. Does that mean the parser is
  correct?" (No — it means no input it tried violated the property you wrote.
  Coverage of the property, not proof.)
- "Which of your golden files would you delete, and why?" (Small outputs; the
  inline `Report` table test is more readable than a file.)
- "`newTestStore` uses `TestMain`'s directory. When would `t.TempDir()` be the
  better call?" (Almost always, unless the resource is expensive to create and
  genuinely shared — a container, a schema load. Good learners argue for
  `t.TempDir()` here; agree with them, then ask what `TestMain` is *for*.)
- "Why is the SQLite store real but the clock faked? Give me the rule."
- "Your CI runs `-short` on every push and the full suite nightly. What breaks
  first?" (Integration failures found a day late, on a merged commit; the PR
  job must be the complete one.)
- "How would you test `Run` if it called `os.Exit` directly?" (You couldn't —
  it kills the test binary. That is why the exit code is a return value.)
- "`-count=2 -shuffle=on` — what would that catch that a normal run doesn't?"

## Grading rubric

- **A** — All criteria pass. The fuzz property is stated in full (level
  validity *and* round trip), and they ran the fuzzer and can describe the
  corpus behavior. `assertGolden` prints both sides on mismatch; `-update`
  guarded and understood as a review obligation. `newTestStore` uses
  `t.Cleanup` and gives each caller its own file; they can explain why `defer`
  would not work. `NOTES.md` is real evidence — a coverage number *and* a
  named uncovered line they judged. They can articulate the fake/real rule
  without prompting.
- **B** — Criteria pass, but with one soft spot: a weak fuzz property, a
  golden helper whose failure message shows only "mismatch", or `NOTES.md`
  answered thinly. Explanation is otherwise sound.
- **C** — Green only after heavy hinting, or the test-infrastructure code is
  copied without understanding (cannot say what `t.Cleanup` guarantees, or why
  `-update` is dangerous). Re-teach the two helpers and re-check.
- **Fail** — Tests failing; or `-update` used to bless output they never
  looked at; or the integration tests "pass" because the helper was weakened
  (e.g. `skipIfShort` made unconditional, or `newTestStore` returning a shared
  store). Remediate, do not advance.

## Remediation ladder

1. "Run `go test -race ./... 2>&1 | head -30`. Which of the four parts is
   failing first? Work that one only." (Scope control is half the battle in a
   four-part exercise.)
2. Now name the concept under *their* part. Parser: "Take the input
   `info|a\|b|c` and walk it byte by byte — what does `strings.Split` do to
   it, and which of those pipes is actually a separator?" Retry: "Write down,
   on paper, the sequence of calls and sleeps for `MaxAttempts: 3` where every
   attempt fails. How many sleeps? Now count them in your loop." Golden/CLI:
   "What is `assertGolden` responsible for, in one sentence — and what does
   `-update` change about that responsibility? It is not 'make the test
   pass'." Store: "`defer store.Close()` runs when `newTestStore` *returns* —
   and the test body has not run yet. What do you want to hang the cleanup off
   instead?"
3. Now the tool. Parser: "You need your own scan, not `strings.Split`: walk
   the bytes and treat `\` as 'the next byte is data, whatever it is'. What
   happens at the end of the input when a `\` has no next byte?" Retry:
   "`time.Sleep` cannot be canceled. Which two channels do you `select` on to
   wait for a duration *or* the context, whichever lands first?" Golden/CLI:
   "Four lines, in order — build the path, read the file, compare, and one
   branch for `-update`. Write them in that order." If still stuck there, show
   the `flag.Bool` declaration only; they finish the body. Store: "`t.Cleanup`
   takes a function that runs when the *test* ends. Rewrite the `defer` as
   `t.Cleanup(func() { … })` and run again."
4. Only if still stuck: walk the `sql.Open` → `Ping` → schema sequence
   verbally and let them type it, or make them state the fuzz property in
   words ("format, then parse, and you get back what you started with") before
   they write a line of it. Never write the fuzz body for them — the property
   *is* the lesson.

## After passing

Preview: "Next you take the other route into a program's internals — reflection
and `unsafe`, plus code generation — and, more importantly, learn when *not*
to."
