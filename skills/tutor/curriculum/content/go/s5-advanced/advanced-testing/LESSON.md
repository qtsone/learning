# Advanced Testing

> `go.advanced.advanced-testing` · ~3-4h · Stage: Advanced Go

## Objectives

By the end of this lesson you can:

- Write a fuzz test with `go test -fuzz`, explain how the corpus and coverage
  guidance work, and turn a found crash into a regression test.
- Implement golden-file tests with an `-update` flag and explain when golden
  files beat inline assertions.
- Separate fast unit tests from slower integration tests (build tags or
  `testing.Short`) and explain the CI implications.
- Write an integration test against a real dependency — here a temporary
  SQLite database — with proper setup and teardown via `TestMain` or
  `t.Cleanup`.
- Choose between a fake, a stub, and a real dependency for a given test and
  justify the trade-off.

## Where example-based testing runs out

Everything you have written since S1 is *example-based*: you pick inputs, you
state the outputs, the test compares. It is the right default — cheap, exact,
readable — but table tests only ever check the rows you thought of, and the S4
debugging lesson's uncomfortable finding was that bugs live in the case nobody
imagined. Four techniques cover what examples cannot:

| Technique | Answers |
|-----------|---------|
| Fuzzing | "what input have I not thought of?" |
| Property tests | "what must be true for *every* input?" |
| Golden files | "did this large output change, and how?" |
| Integration tests | "does it work against the real thing?" |

A theme runs through all four: **you are now writing test infrastructure, not
just tests** — a golden helper, a fixture constructor, a fake clock. That code
lives in `_test.go` files and deserves production-grade care, because a broken
helper silently disarms every test that uses it.

## Fuzzing: the machine writes the inputs

`go test` has a third kind of function next to `TestXxx` and `BenchmarkXxx`:

```go
func FuzzParseLine(f *testing.F) {
	f.Add(`info|api|started`)   // a seed
	f.Fuzz(func(t *testing.T, line string) {
		// the property, checked for every input
	})
}
```

Run normally it behaves like an ordinary test: the body runs once per *seed* —
the `f.Add` values plus every file in `testdata/fuzz/FuzzParseLine/`. Fast,
deterministic, CI-safe.

Run with `-fuzz` it becomes a **coverage-guided mutational fuzzer**. It takes a
seed, mutates it (flip bytes, splice two inputs, insert known-tricky values),
runs your body, and asks: *did that input reach code paths no earlier input
reached?* If so, the mutant is "interesting" and joins the working corpus to be
mutated further. That feedback loop is why fuzzing beats random generation by
orders of magnitude — random bytes never get past a parser's first validation
check, while coverage guidance lets the fuzzer *learn* the input format one
branch at a time.

```sh
go test -fuzz=FuzzParseLine -fuzztime=30s .
```

Two corpora, and the difference matters. The **seed corpus** is committed to
your repository and is part of the test suite. The **generated corpus** lives
in the build cache (`go clean -fuzzcache` empties it), is machine-local, and is
never shared — fuzzing is not reproducible between machines, and that is
normal.

When the body fails or panics, the fuzzer **minimizes** the input — repeatedly
shrinking it while it still fails — and writes the result into
`testdata/fuzz/<FuzzTarget>/`. Commit that file: it is now a permanent
regression test that runs on every plain `go test`. That is the S4 rule ("never
fix a bug without a test that fails first") with the machine writing the test.

The CI implication: **do not put `-fuzz` in your normal test job.** Fuzzing is
unbounded work with a non-deterministic result, and a per-PR job that sometimes
takes 30 seconds and sometimes fails on an unrelated input is a job people
learn to ignore. Run the seed corpus in CI — you get that for free — and fuzz
on a schedule, with a fixed `-fuzztime`, whose findings arrive as committed
corpus files.

## Property-based thinking

A fuzz body cannot say `want == got`: nobody knows the answer for an input the
machine invented. It states a **property** — a claim true for every input.
Four shapes cover most cases:

- **Round trip** — `decode(encode(x)) == x`. The exercise's parser is exactly
  this, and it is the highest-value property in the codebase: it pins encoder
  and decoder to each other.
- **Invariant** — a rule the output always obeys: the delay never exceeds
  `Max`, the sorted slice is sorted, the total equals the sum of the parts.
- **Oracle** — compare against a slower, obviously correct implementation.
- **Metamorphic** — relate two runs: sorting twice changes nothing.

You do not need `-fuzz` to think this way. `TestBackoffProperties` in the
exercise loops over 64 attempts asserting bounds and monotonicity — a property
test in plain Go, and the one that catches the integer overflow no table would
have listed. Write properties first; fuzzing is just an energetic way to supply
inputs to them. One warning: a property that only asserts "it didn't panic" is
weak, because the fuzzer will happily bless a parser that returns nonsense. A
fuzz test is exactly as strong as its property.

## Golden files, and testing a CLI

Some outputs are too big for an inline assertion: a rendered report, a
generated file, an HTTP response body. The golden-file pattern stores the
expected bytes in `testdata/` and compares:

```go
func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name)
	if *updateGolden {                   // -update, declared in a _test.go file
		os.WriteFile(path, got, 0o644)
		return
	}
	want, err := os.ReadFile(path)
	// … compare, and print both sides on mismatch
}
```

Golden files win when the output is **large, structural and changes as a
unit** — you review it as a diff, in the same review as the code change. They
lose when the output is small (an inline `want` is more readable than a file
you have to open) or when only one field matters, because then every unrelated
change becomes a test failure.

`-update` is the pattern's power and its trap. It turns an intended change into
one command instead of hand-edited bytes — and it makes blessing a bug
trivially easy: run it, tests pass, wrong output is now the specification. The
discipline is non-negotiable: **the golden diff is part of the code review.**
Keep golden files in `testdata/` (the Go tool ignores that directory) and keep
them text, so diffs stay readable.

The same machinery is how Go tests command-line tools. The `go` command's own
suite is written as little shell-like scripts — commands, expected stdout,
expected exit status — and the packaged version of that idea,
`github.com/rogpeppe/go-internal/testscript`, is what many Go CLIs use. You
already own the prerequisite: the concurrent link checker in S3 made you write
`Run` with injected streams and its own `flag.NewFlagSet`. Here is why that
shape was worth the trouble — it is what makes a CLI testable at all:

```go
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int
```

`main` becomes `os.Exit(logkit.Run(os.Args[1:], os.Stdin, os.Stdout,
os.Stderr))`, and everything else is a normal in-process test: pass args, feed
a string as stdin, capture two buffers, assert the exit code. Three rules make
it work — parse flags into your own `flag.NewFlagSet` (never the global
`flag.CommandLine`, which cannot be re-parsed between tests), never call
`os.Exit` below `main`, and treat the exit code as part of the contract.

## Testing time without waiting

Tests that sleep are slow; tests that assert on elapsed time are flaky — doubly
so here, where the graded run is `go test -race ./...` and the race detector
stretches every duration by 2-20x. The cure is the S4 one: make the hidden
dependency explicit.

```go
type Clock interface {
	Sleep(ctx context.Context, d time.Duration) error
}
```

Production passes `RealClock`; the test passes a fake that records the
durations it was asked for and returns immediately. The assertion becomes
`slept == []time.Duration{100ms, 200ms, 250ms}` — instant, exact, and a
statement about *intent*. Note what the fake buys beyond speed: no stopwatch
could measure that sequence at all.

Go 1.25 adds `testing/synctest` for the harder case — code with real goroutines
and channel timeouts, where you cannot inject a clock everywhere. It runs a
function in an isolated "bubble" with a fake clock, and advances that clock
instantly whenever every goroutine in the bubble is blocked, so a test of a
30-second timeout finishes in microseconds. The mental model (time moves only
when nothing else can) is worth having; this exercise uses injected clocks,
which work on every version.

## Fast tests, slow tests, and CI

An integration test that opens a database costs orders of magnitude more than a
unit test. Both are worth having; the mistake is making developers pay for both
on every save. Two mechanisms split them:

- **`testing.Short()`** — the test decides at runtime (`if testing.Short() {
  t.Skip(…) }`), and `go test -short ./...` is the fast loop. Everything still
  compiles, always.
- **Build tags** — `//go:build integration` on the file, run with
  `go test -tags=integration ./...`. Stronger separation, and that is also the
  risk: tagged-out code rots silently because nothing type-checks it.

Prefer `testing.Short` unless the excluded code needs dependencies the default
build cannot have. Either way the CI contract is the same and belongs in your
README: **the pull-request job runs everything** — that is the job whose green
light means "safe to merge" — while the short subset is a local convenience. A
team that ships only the fast subset in CI has decorative integration tests.

## Integration tests: setup, teardown, isolation

An integration test touches something real — a database file, a temp directory,
an `httptest` server. Three tools shape them:

- **`t.Cleanup(fn)`** runs teardown when *this* test or subtest ends, LIFO,
  even on `t.Fatal`. It beats `defer` inside a helper, whose `defer` fires when
  the *helper* returns — that is what makes `newTestStore(t)` possible: the
  helper opens the database and registers its own close.
- **`TestMain(m *testing.M)`** wraps the whole package run: setup before
  `m.Run()`, teardown after. Use it for what is genuinely shared and expensive.
  Ordering rule: `os.Exit` does not run defers, so teardown goes *before*
  `os.Exit(code)`, never in a `defer`.
- **`t.TempDir()`** gives one test its own directory and removes it
  automatically. Reach for it first; drop to `TestMain` only when the resource
  must outlive a single test.

What matters more than any of them is **isolation**: a test must not see
another test's data, and must pass alone, in any order, and under `-count=2`.
That means a fresh database per test (or a transaction rolled back at the end),
never a shared fixture mutated in sequence. Order dependence is the commonest
cause of "flaky" suites, and `go test -count=2 -shuffle=on ./...` is how you
find it.

## Fake, stub, or the real thing

S4 gave you the vocabulary (a stub returns canned answers, a fake is a working
lightweight implementation, a mock records calls). The production question is
*when to use each*:

| Dependency | Choice here | Why |
|------------|-------------|-----|
| Time | stub (`fakeClock`) | real time is slow and unassertable; nothing about `time` needs verifying |
| stdin/stdout | real `strings.Reader` / `bytes.Buffer` | the real thing *is* the cheap thing — S3's `io.Writer` paying rent again |
| SQLite store | the real database, in a temp file | the SQL *is* the code under test |
| An HTTP dependency | `httptest.Server` | real server, real serialization, no network |

The rule that generalizes: **double the things whose behavior you do not care
about; keep real the things whose behavior you are testing.** Faking a database
means your tests can never catch a typo in your SQL, a missing index, or a NULL
you failed to handle — precisely the bugs they existed to catch. Faking the
clock costs nothing, because "does `time.Sleep` sleep?" was never in question.

## Reading coverage honestly

```sh
go test -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1     # the headline number
go tool cover -html=cover.out               # the useful view
```

Go measures **statement** coverage: which statements ran. That is weaker than
it sounds. A line covered by a test that asserts nothing counts as covered;
`if a && b` counts as covered when one combination ran; and coverage says
nothing about inputs you never supplied — the gap fuzzing fills.

So use it as a **finder, not a target**. The red lines in the HTML view are a
question list: is this error path untested because it cannot happen, or because
you forgot? Uncovered error branches are the classic finding, and where
incidents live. Mandating a percentage, by contrast, reliably produces tests
written to execute lines rather than check behavior. (`-race` with `-cover`
forces `-covermode=atomic` — correct, but slower: one more reason the full
suite is a CI job, not a save hook.)

## Exercise

Open [`exercise/`](exercise/) — `logkit`, a small log-processing package with
four independent testing problems in it, plus `NOTES.md` for your evidence.
Read the test files first; as always they are the specification, but this time
part of the work *is* test code (two clearly marked helpers).

```sh
cd exercise
go test -race ./...                           # the graded command
go test -short ./...                          # the fast loop: unit tests only
go test -fuzz=FuzzParseLine -fuzztime=30s .   # by hand, never in CI
```

Acceptance criteria:

1. `Format` and `ParseLine` implement the wire format: three `|`-separated
   fields, with `\`, `|`, newline and carriage return escaped as `\\`, `\|`,
   `\n`, `\r`. `ParseLine` rejects — with errors `errors.Is` matches to
   `ErrMalformed` — the wrong number of fields, unknown levels, dangling or
   unknown escapes, and raw newlines.
2. `FuzzParseLine`'s body asserts the properties: a successful parse yields a
   known level, and re-encoding it with `Format` parses back to an equal
   `Event`. You run the fuzzer for at least 30 seconds and record what happened
   in `NOTES.md`; any crasher it finds stays committed under
   `testdata/fuzz/FuzzParseLine/`.
3. `Policy.Backoff` returns capped exponential delays that never overflow, and
   `Retry` calls `fn` at most `MaxAttempts` times, sleeps `Backoff(attempt)`
   between attempts and never after the last one, stops immediately on an error
   wrapping `ErrPermanent`, and on a done context returns an error matching
   both the context error and the last attempt's error. `RealClock` respects
   cancellation.
4. `Report` renders the exact table in its doc comment, and `Run` behaves as
   documented: `-min-level` filtering, `logreport: line <n>: <err>` diagnostics
   on stderr, and exit codes 0 / 1 / 2.
5. `assertGolden` compares against `testdata/golden/<name>` and rewrites it
   under `-update` (a flag you declare in the test file). The committed
   `service-report.txt` must still match at the end — if you regenerate it,
   review the diff.
6. `newTestStore` gives each caller its own database file under the directory
   `TestMain` creates, and registers teardown with `t.Cleanup`. The store opens
   SQLite, applies the schema, rejects unknown levels with `ErrMalformed`, and
   round-trips events through `Insert`, `All` and `CountsByLevel`.
7. `NOTES.md` is filled in: fuzzing evidence, coverage reading, the golden
   `-update` experiment, and the doubles table. Your tutor will ask.
8. `go test -race ./...` passes, `go test -short ./...` skips the integration
   tests, and the code is `gofmt`-clean.

Suggested order: the parser and its fuzz property first (criteria 1-2, they
unlock everything), then the retry loop (3), then the report and CLI (4-5),
then the store (6).

## Further reading

- [go.dev — Go Fuzzing](https://go.dev/doc/security/fuzz/) — the reference:
  supported types, corpus file format, `-fuzztime`, minimization.
- [Go Blog — Fuzzing is Beta Ready](https://go.dev/blog/fuzz-beta) — the design
  and the corpus model, from the team that built it.
- [pkg.go.dev — testing](https://pkg.go.dev/testing) — `TestMain`, `Cleanup`,
  `TempDir`, `Short`, `F`; re-read it now that you use most of it.
- [go.dev — The cover story](https://go.dev/blog/cover) — what the tool
  measures, and what it does not.
