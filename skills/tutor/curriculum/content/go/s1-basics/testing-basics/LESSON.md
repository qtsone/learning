# Testing Basics

> `go.basics.testing-basics` · ~3-4h · Stage: Programming Basics (Go)

## Objectives

By the end of this lesson you can:

- Write a test function following the `TestXxx(t *testing.T)` convention and
  run it with `go test`.
- Implement a table-driven test and explain why Go favors this pattern.
- Use `t.Run` to create named subtests and run a single case with
  `go test -run`.
- Measure coverage with `go test -cover` and explain what coverage does — and
  does not — guarantee.

## Switching sides

Every exercise in this stage has ended the same way: `go test ./...`, red
until you made it green. For a dozen lessons you have been *reading* tests as
specifications someone else wrote. Today you switch sides and write them.

Why programmers write tests at all:

- **Proof that stays.** Checking with `go run .` and your eyes works once,
  then evaporates. A test is that same check written down — it re-runs in
  milliseconds, every time, forever.
- **Courage to change.** Once tests cover a function, you can rewrite its
  insides and know within seconds whether you broke a promise. Without tests,
  every change is a small act of hope.
- **An executable specification.** You've felt this from the reader's side:
  the test file told you exactly what "done" meant. Now you get to *define*
  done.

## Anatomy of a test

Tests live next to the code they test, in files whose names end in
`_test.go`, inside the same package. `go build` ignores these files;
`go test` compiles them together with the package and runs what it finds.

A test is a function with an exact shape:

```go
func TestLongest(t *testing.T) {
	got := Longest([]string{"go", "gopher", "tea"})
	if got != "gopher" {
		t.Errorf("Longest(%v) = %q, want %q", []string{"go", "gopher", "tea"}, got, "gopher")
	}
}
```

`go test` recognizes a test by convention, not registration:

- The file ends in `_test.go`.
- The function's name starts with `Test` followed by a *non-lowercase*
  letter: `TestLongest` is a test; `Testlongest` is silently ignored — no
  error, it just never runs. Check `go test -v` actually lists your test.
- It takes exactly one parameter, `t *testing.T` — your handle for reporting
  failures.

Notice what's *not* there: no assert functions, no framework. This is
deliberate. You test Go with Go — call the function, compare with `if`,
report with `t.Errorf`. The message convention you've seen all stage is
`got X, want Y`, with the inputs included, so a failure reads like a small
bug report: what was called, what came back, what should have.

Two ways to report, and the difference matters:

- `t.Errorf(...)` — mark the test failed and *keep going*, so one run can
  report several mismatches.
- `t.Fatalf(...)` — mark it failed and *stop this test function now*. Use it
  when continuing makes no sense: if you needed a value and got an error
  instead, every later check would just be noise.

A test *passes* by not failing: reach the end of the function without calling
either, and it's green. Run `go test` for a summary, `go test -v` to see
every test by name.

## Table-driven tests

The test above checks one input. Real functions need many: the normal case,
the empty case, ties, boundaries. Copy-pasting the call-and-compare block
once per input buries the logic in repetition — and Go's answer is to store
the *cases* as data and keep the *checking* in one place. You know both
halves already: a slice, of structs.

```go
func TestCountVowels(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"simple word", "gopher", 2},
		{"no vowels", "rhythm", 0},
		{"empty string", "", 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CountVowels(c.in); got != c.want {
				t.Errorf("CountVowels(%q) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}
```

One new thing here: the struct type has no name. It's an **anonymous struct**,
declared right where it's used, because no other code will ever need this
type. Everything else is the field-named literals and `range` loop you've
been writing for weeks.

Why Go favors this pattern enough that it has a name:

- **Adding a case is one line of data**, not a new function. When you think
  "what about the empty string?", the cost of answering is nearly zero.
- **The checking logic exists once.** Improve the failure message, or fix a
  comparison, in one place.
- **The table reads as a specification.** A reviewer scans inputs and
  expected outputs as a list, without re-reading any logic.

## Subtests with t.Run

The loop above wraps each case in `t.Run(name, func(t *testing.T) {...})`.
Each call creates a named **subtest**, and that buys you three things:

- **Failures name the guilty case**: `--- FAIL: TestCountVowels/no_vowels`
  tells you *which row* of the table broke. (Spaces in names become
  underscores in the output.)
- **Cases are isolated**: `t.Fatalf` inside a subtest stops only that case —
  the loop continues to the next row. Without `t.Run`, a `Fatalf` would
  abandon the whole table.
- **You can run a single case**:

```sh
go test -run 'TestCountVowels/no_vowels' -v
```

`-run` takes a slash-separated pattern: the part before `/` matches test
function names, the part after matches subtest names. When one stubborn case
is failing, rerunning just that case keeps the loop tight.

## Coverage — executed is not verified

How much of the package do your tests actually reach?

```sh
go test -cover                     # coverage: 87.5% of statements
go test -coverprofile=cover.out    # record it, then view line by line:
go tool cover -html=cover.out
```

The HTML view paints every executed line green and every missed line red —
red lines are code no test ever ran, which usually means a branch you forgot
(an error path, an empty-input fallback).

Now the caveat that makes this an objective, not a victory lap: coverage
proves your tests *executed* a line, not that they *verified* it. Consider:

```go
func Double(n int) int {
	return n * n // wrong: that's squaring
}

func TestDouble(t *testing.T) {
	if got := Double(2); got != 4 {
		t.Errorf("Double(2) = %d, want 4", got)
	}
}
```

100% coverage. Green. Wrong for almost every input — `2` happens to be the
one value where doubling and squaring agree. Coverage finds code you forgot
to *reach*; only well-chosen cases find code that's *wrong*. Choose inputs
where the right answer and a plausible wrong answer differ: empty inputs,
a single element, ties, numbers that don't divide evenly, error paths.

## Exercise

Open [`exercise/`](exercise/) — a Go module with the roles reversed:

- `stats.go` — three small finished functions (`Longest`, `CountVowels`,
  `Average`). Their doc comments are the contract, and the contract is
  honest — but **one of the three implementations has a real bug**.
- `stats_test.go` — three stub tests that fail with `TODO`. Replace each
  stub with a real test; the comments say what each must cover.

Acceptance criteria:

1. `TestLongest` follows the `TestXxx(t *testing.T)` convention — written
   plainly, no table — and checks a normal slice, a tie (earliest wins), and
   an empty slice, reporting failures with got *and* want.
2. `TestCountVowels` is table-driven: a slice of anonymous case structs and
   one named `t.Run` subtest per case, covering at least a simple word, a
   word with no vowels, the empty string, and mixed upper/lower case.
3. `TestAverage` is table-driven and also checks errors: at least a slice
   that divides evenly, a slice whose true average is *not* a whole number,
   and the empty slice, which must return an error (and no error anywhere
   else).
4. Your tests catch the planted bug. Fix `stats.go` so the implementation
   honors its doc comment — fix the code, never the test.
5. You've run a single subtest with `go test -run 'TestName/case_name'` and
   checked `go test -cover` — the solution reaches 100% of `stats.go`. Be
   ready to explain why that number alone wouldn't have caught the bug.
6. `go test ./...` passes and the code is gofmt-formatted.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test -v
```

The starter fails all three tests with `TODO` messages — replace the stubs
one at a time, watching each function's tests go green before moving on.

## Further reading

- [go.dev — Add a test (Getting started tutorial)](https://go.dev/doc/tutorial/add-a-test)
- [pkg.go.dev — testing package](https://pkg.go.dev/testing)
- [Go wiki — Table-driven tests](https://go.dev/wiki/TableDrivenTests)
- [Go blog — The cover story](https://go.dev/blog/cover)
