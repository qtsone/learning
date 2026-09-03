# Tutor notes — Test-Driven Development

## Where the learner is

Post S0-S3 plus clean-code and code-organization: an idiomatic intermediate
Go programmer who has *consumed* table-driven tests since S1 (every exercise
here hands them failing tests) but has likely never driven design with tests
they wrote themselves. They know interfaces, consumer-side interface
declaration, `io.Writer`, `errors.Is/As` and `%w` — Part B deliberately
cashes in all of that. The new material is the discipline (red before green,
minimal green, refactor under green) and the doubles vocabulary.

Process matters as much as output in this lesson. Ask early for their
red-green evidence for Parts A and C: a commit per phase, or a pasted log of
a failing run followed by the passing one. If they arrive with only finished
code, probe hard — a perfect suite written code-first misses the objective.

## Common misconceptions

- **"TDD means writing lots of tests"** — it means writing *one failing test
  at a time, first*. Test-after with high coverage is fine work but a
  different practice; the design pressure and the test-the-test moment are
  exactly what test-after lacks.
- **"I don't need to run the red — I know it fails"** — the red run is the
  only time the test itself is tested. Have them recall a test that passed
  for the wrong reason (wrong function, vacuous assert) or show them one.
- **"Mock" as the word for every double** — the industry sloppily says mock
  for everything; this lesson grades the distinction. `stubClock` is a stub
  (canned answer, no recording), `spyNotifier` is a spy (records
  interactions). If they call both mocks, drill the table.
- **Mocking everything by reflex** — interaction tests on pure logic couple
  tests to implementation and shatter on refactor. State-based first; spy
  only where the call *is* the behavior (the notifier).
- **"Simplest thing that passes" read as permission for dumb code forever**
  — the refactor phase is where quality happens; green minimalism only
  postpones design until it is safe, it doesn't cancel it.
- **Interface-for-everything after Part B** — injection is the cure for
  *hidden* dependencies (time, output, network), not a mandate to wrap pure
  logic like `SplitEvenly` in interfaces. Clean-code's YAGNI instinct applies.
- **Implementing Part A ahead of the increments** — writing the full parser
  at increment 2 skips the rhythm the part exists to teach. If the log shows
  one giant green, have them narrate what each increment forced.

## Grilling points

- "Show me your Part C red — the actual failing output. What exactly did the
  failure message say, and why is that message worth polishing?"
- "In `reminder_test.go`, which double is `stubClock` and which is
  `spyNotifier`? Why does the notifier need a spy when the clock doesn't?"
- "Why does the spy record messages instead of `SendOverdue` returning the
  messages it sent? What makes the interaction the thing worth asserting?"
- "What precisely made `LegacySendOverdue` untestable? Two answers expected:
  hidden input, hidden output — and one injected replacement for each."
- "The boundary test: a task due *exactly* now is not overdue. Could you have
  pinned that behavior down against the legacy version? Why not?"
- "Walk me through one full cycle from Part A — which increment, what was
  red, what was the simplest green, what did you refactor?"
- "Where did writing (or reading) the test first change an interface in this
  exercise? What would you have written implementation-first?"
- "When would you *not* TDD?" (Spikes, throwaway exploration — then delete
  and TDD the keeper.)

## Grading rubric

- **A** — All tests pass including their own `split_test.go` with a
  remainder case and a sum check; red evidence exists for Part C (and
  ideally per-increment for Part A); doubles named correctly with the
  state-vs-interaction reasoning; can articulate what each phase is for and
  point to where the injected design beat the legacy shape.
- **B** — Tests pass and process was mostly followed, but `split_test.go` is
  thin (single case, no sum property), or the doubles taxonomy is shaky
  ("they're all mocks"), or Part A was implemented in one or two big greens
  with a plausible retrospective narration.
- **C** — Tests pass but code-first throughout with no red evidence, or
  `SendOverdue` needed heavy hints (calling `time.Now()` inside despite the
  injected clock is the telltale). Pass only after a time-boxed live cycle:
  make them do one genuine red-green-refactor on a small change while you
  watch; else another iteration.
- **Fail** — Tests failing; or `split_test.go` missing/asserting nothing; or
  the learner cannot explain why a test must fail first. Remediate, don't
  advance.

## Remediation ladder

1. "Read the first failing test top to bottom and tell me what behavior it
   specifies — inputs, expected output, and what the failure message says."
2. Part A stuck: "Increment N is red. What is the *least* code — however
   crude — that makes exactly that assertion pass? Write that, nothing more."
3. Part B stuck: "Inside `SendOverdue`, where must 'now' come from? Not
   `time.Now()` — which field was injected for exactly this? Now do the same
   substitution for the printing."
4. Part C stuck: "Call `SplitEvenly(100, 3)` in a test and assert the shares
   sum to 100. Run it. Read the numbers it printed — where did the cent go?"
   Then let them fix the remainder loop themselves.

## After passing

Preview: "Tests catch the bugs you predicted. Next lesson is Debugging — a
scientific method for the ones you didn't, and the failing test you write at
the end of every hunt is this lesson's discipline paying off."
