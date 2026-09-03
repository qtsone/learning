# Test-Driven Development

> `shared.eng.tdd` · ~3-4h · Stage: Engineering Practice

## Objectives

By the end of this lesson you can:

- Implement a feature strictly via red-green-refactor, showing a failing test
  before each production change.
- Explain what each red-green-refactor phase is for and what skipping it
  costs.
- Choose between a fake, a stub, and a mock for a given dependency and
  justify the choice.
- Redesign a hard-to-test function (hidden I/O or globals) into a testable
  shape by injecting its dependencies.
- Explain how TDD pressures API design, with an example where writing the
  test first changed the interface.

## You already write tests — this is about *when*

Since S1 you have written table-driven tests and treated the test file as the
specification. Every exercise so far handed you failing tests and asked you to
turn them green. Test-driven development is that same experience with one
change: **you** write the failing test, one small behavior at a time, before
you write the code that passes it. The loop has three phases and takes
minutes, not hours:

```text
    ┌────────────── RED ──────────────┐
    │ write ONE small failing test    │
    │ run it — watch it fail          │
    └───────────────┬─────────────────┘
                    ▼
    ┌───────────── GREEN ─────────────┐
    │ simplest change that passes —   │
    │ nothing speculative             │
    └───────────────┬─────────────────┘
                    ▼
    ┌──────────── REFACTOR ───────────┐
    │ improve names, remove           │
    │ duplication; tests stay green   │
    └───────────────┬─────────────────┘
                    └──▶ next RED
```

The output of many loops is not just working code — it is a test suite where
every test has *earned its place* by failing once, and a design that grew
under constant pressure to stay usable.

## What each phase is for — and what skipping it costs

**Red exists to prove the test can fail.** A test you never saw fail is an
assumption, not evidence. Maybe it asserts nothing; maybe it exercises the
wrong function; maybe a typo makes it vacuously true. Teams discover these
"tests that cannot fail" years later, guarding nothing, when the bug they
should have caught ships anyway. Watching the red — and *reading the failure
message* to check it is the failure you expected — is the only moment you
ever test the test. Skip red, and you may be accumulating decoration.

**Green exists to keep you honest about scope.** Write the simplest code that
passes — even code that feels naively direct. The discipline sounds silly
("just return the constant?!") but it has a serious point: any code beyond
what a test demanded is *untested* code, and speculative generality is how
small functions grow the extra parameters and half-implemented options you
learned to distrust in the clean-code lesson. If more behavior is needed,
that is the next red test's job. Skip the discipline, and your "tested" code
quietly contains branches no test has ever executed.

**Refactor exists because green makes it safe.** With passing tests as a net,
you can rename, extract, and de-duplicate without fear — run the tests, and
mistakes surface in seconds. This is where design quality actually happens;
red and green deliberately postpone it. Skip refactor, and each green pass
piles code on top of the last one's shortcuts. The suite keeps passing while
the design rots — debt with excellent test coverage.

One more rule the loop implies: **never write production code without a
failing test demanding it, and never fix a bug without first writing the test
that reproduces it.** The bug got through because a test was missing; fixing
the code first and back-filling the test means the test never fails, so it
never proves it would have caught anything.

## Test doubles: stub, fake, mock

Real dependencies make bad test partners: clocks move, networks flake,
payment gateways charge money. A **test double** stands in for the real thing.
The names matter because they describe *different verification strategies*,
and picking the wrong one produces brittle tests:

| Double | What it does | You assert on |
|--------|--------------|---------------|
| **Stub** | returns canned answers ("now is noon", "card declined") | the *result* your code produced from those answers |
| **Fake** | a real, working, lightweight implementation (in-memory store instead of a database) | the *result*, same as with the real thing |
| **Mock / spy** | records the calls made to it (a spy records; a mock also asserts expectations) | the *interaction* — that the right calls happened |

Stubs and fakes support **state-based** tests: arrange inputs, act, assert on
outputs. Mocks and spies support **interaction-based** tests: assert that
your code *called* the dependency correctly. Prefer state-based when you have
a choice — a test that says "the total is 42" survives any internal
rewrite, while a test that says "it called Get, then Put, in that order"
breaks the moment you refactor, even when behavior is unchanged. Reach for a
mock or spy only when the interaction *is* the behavior: sending a
notification, charging a card, writing an audit record. There is no return
value to inspect — the call itself is the observable outcome.

In Go: you rarely need a mocking framework. Declare a small interface where
the dependency is *consumed* (remember S3 — interfaces belong to the
consumer), and hand-roll the double in the test file:

```go
type stubClock struct{ now time.Time }

func (c stubClock) Now() time.Time { return c.now }

type spyNotifier struct{ messages []string }

func (s *spyNotifier) Notify(m string) error {
	s.messages = append(s.messages, m)
	return nil
}
```

Ten lines, no dependencies, and the double's behavior is in plain sight
inside the test that uses it. This is idiomatic Go testing, and it only works
because the interfaces are small — one more reason to keep them that way.

## Designing for testability

Some code is easy to test: values in, values out. Some is nearly impossible:

```text
function sendOverdue(tasks):
    for each task:
        if task.due < CURRENT_TIME():     # hidden input
            PRINT("overdue: " + task.title)   # hidden output
```

Two dependencies are *hidden inside* the function: the clock (a global,
ambient input — the answer changes with the wall clock) and the terminal (a
side effect no test can read back). The test cannot control what the function
sees or observe what it did. The fix is always the same move — **make hidden
dependencies explicit parameters**, then substitute doubles in tests:

```text
function sendOverdue(tasks, clock, notifier):
    for each task:
        if task.due < clock.now():
            notifier.notify("overdue: " + task.title)
```

Now the test injects a stub clock ("it is exactly noon") and a spy notifier,
and can assert both the boundary behavior (due *at* noon — overdue or not?)
and the exact messages. Notice that the untestable version could never pin
down that boundary at all — the wall clock never cooperates. This is the same
dependency-direction idea from the code-organization lesson, applied in the
small: the function depends on an abstraction it is handed, not on a global
it reaches for. Hidden time, hidden randomness, hidden environment variables,
hidden files, printing directly — the injection cure is identical for each.

In Go the injected dependencies are interface (or func) fields, and the
zero-cost fake for anything that writes text is `io.Writer` — pass
`os.Stdout` in production and a `bytes.Buffer` in tests, exactly the S3 io
philosophy paying rent.

## TDD as design pressure

Here is the underrated effect: writing the test first makes you the **first
consumer of your own API** before any implementation exists to defend. In the
test you must construct the thing, call it, and check the result — so
awkwardness surfaces immediately. Can't build the object without six
arguments you don't have? Can't check the result because the function prints
instead of returning? Need three doubles just to call one method? The test is
telling you the *interface* is wrong, at the one moment it is still free to
change.

That is exactly the redesign above. Test-first, `sendOverdue` grew a clock
and notifier parameter and a return value — not for testability as decoration
but because the first consumer (the test) could not use the old shape at all.
Code written implementation-first tends to grow hidden dependencies because
nothing pushes back; code written test-first *cannot*, because the test feels
the pain instantly. A widely shared observation follows: hard-to-write tests
are rarely a testing problem — they are a design smell announcing itself
early, while it is still cheap to fix.

TDD is a discipline, not a religion. For exploratory spikes where you don't
yet know what you're building, write throwaway code first, learn, then delete
it and TDD the real thing. But for code you intend to keep, the loop earns
its keep — as you're about to feel.

## Exercise

Open [`exercise/`](exercise/) — a Go module with three parts. This lesson
grades the *process* as much as the result: keep a short log (or a git commit
per phase) showing red before green — your tutor will ask for it.

**Part A — the rhythm** (`calculator.go`): implement
`Add(input string) (int, error)` against `calculator_test.go`, which is
written as six ordered increments. Treat each increment as a red you would
have written yourself: make exactly one more test pass with the simplest
change, then refactor before moving on. Resist implementing ahead of the
tests — that is the whole exercise.

**Part B — doubles and testability** (`reminder.go`): `legacy.go` contains
`LegacySendOverdue`, the untestable "before" (hidden clock, prints to the
terminal — do not modify it). Implement `Reminder.SendOverdue`, the injected
redesign, against `reminder_test.go`. Read the test file first: it defines a
`stubClock` and a `spyNotifier` — know which double is which and why each
was chosen before you write a line.

**Part C — red first, for real** (`split.go`): `SplitEvenly` has a bug that
silently loses money. Write `split_test.go` yourself, watch your test fail on
the buggy code, and only then fix the implementation. Test first — writing
the fix before the failing test forfeits the point of the part.

Acceptance criteria:

1. `Add` passes all six increments, including rejecting negatives with the
   exact error `negative numbers not allowed: -2, -4` for input `"1,-2,3,-4"`.
2. `SendOverdue` notifies only tasks strictly overdue at the injected clock's
   time, in input order, with messages `overdue: <title>`.
3. On a notifier failure, `SendOverdue` stops, returns the count delivered so
   far, and returns an error that `errors.Is` can match to the notifier's.
4. Your own `split_test.go` fails on the starter `SplitEvenly` (keep the
   evidence), and after your fix the shares always sum to the total with no
   two shares differing by more than one cent, larger shares first.
5. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

Parts A and B FAIL on the starter — work them red by red.

## Further reading

- [Kent Beck — Canon TDD](https://tidyfirst.substack.com/p/canon-tdd) — the inventor's own correction of what the loop is and isn't
- [Martin Fowler — Mocks Aren't Stubs](https://martinfowler.com/articles/mocksArentStubs.html) — the doubles taxonomy and state vs interaction testing
- [Learn Go with Tests](https://quii.gitbook.io/learn-go-with-tests) — a whole Go curriculum taught test-first; skim a chapter to see the rhythm at length
- [Martin Fowler — Test Double](https://martinfowler.com/bliki/TestDouble.html) — the five-name glossary on one page
