# Capstone: Core Build

> `go.capstone.build-core` · ~20-30h · Stage: Expert Capstone (Go)

## Objectives

By the end of this lesson you can:

- Implement the core functionality defined in your PRD, delivering each planned
  milestone with passing tests.
- Structure the codebase with clear package boundaries and dependency
  direction, and justify the layout in review.
- Apply idiomatic Go throughout: error wrapping, context propagation, small
  interfaces, and concurrency only where it earns its cost.
- Maintain a test suite that covers the core logic and runs green via
  `go test ./...` at every milestone.
- Explain during milestone reviews which ADRs held up, which you revised, and
  why.

## What a harness can grade, and what it cannot

Every exercise until now shipped tests that knew the answer. This one cannot:
the project is yours, the features are yours, and nobody wrote the spec but
you. What ships instead is a **conformance harness** — a module that finds your
project, runs its own toolchain, and checks the properties that hold for any
finished piece of Go, whatever it does.

So the harness asks: does it build, does it vet clean, does your suite pass
with the race detector on, does that suite actually reach the code, is there
real package structure, is there unfinished work left lying around, can a
stranger run it, did you deliver the milestones you planned. Those are the
questions a reviewer asks before they even look at your features.

Your features are graded too — by conversation, at each milestone review. That
half is the harder half, and it is the half the rest of this lesson prepares
you for.

## The walking skeleton

The first thing you build is not the interesting thing. It is the thinnest
slice that goes end to end: entry point → one real package → one real
behaviour → one real test → a command you can run and watch produce output.
For a CLI that might be "parse one flag, call one function, print one line".
For a service, "one route, one handler, one store call, one row".

This is a **walking skeleton**: it walks, badly, but it walks. Building it
first buys three things.

- **Integration risk arrives on day one**, when it is cheap. The parts that
  refuse to fit — the driver that needs cgo, the library whose API is nothing
  like the docs, the config that has to be read before logging starts — are
  found now, not in the last week.
- **Feedback loops exist from the start.** `go test ./...`, `go run`, and the
  harness all work on day one, so every later change is measured against a
  green baseline instead of a hope.
- **You are always demoable.** At any hour of the 20-30, you can show someone a
  program that runs. Projects that die in the middle almost always died with
  nothing runnable to show.

The skeleton is not a prototype you throw away. It is milestone one, and it
stays.

## Milestone discipline

You wrote milestones in the planning lesson. The rule that makes them worth
having: **every milestone leaves the project runnable and green.** Not "green
at the end". Green at each box you tick.

That rule has teeth. It forbids the six-day refactor that leaves `main` broken
in the middle. It forbids "I'll write the tests after". It forces you to slice
by *behaviour a user could observe* rather than by layer — "the store package"
is not a milestone anyone can run; "notes survive a restart" is.

Practical shape for 20-30 hours: eight to twelve working sessions, four to six
milestones, so one to three sessions each. If a milestone is eating its third
session, it was two milestones. Split it — cut the piece that is already
working out of the front and tick that.

Tick a box only when `go test -race ./...` is green *and* the behaviour is
reachable by running the program. A ticked box that isn't true is worse than an
unticked one: it is a lie you will believe next week.

## Commit early, commit small

Commit at every green point, not at every "done" point. A commit is a save
point in an experiment, and you will want to run experiments: rip out the
interface, try the other data structure, invert the dependency. Those are safe
only when the suite is green and the tree is clean, because then `git diff`
shows you exactly what you did and `git restore` undoes exactly that.

Two habits worth building here:

- **Never end a session on red.** If you must stop mid-change, commit on a
  branch with a message that says what is broken and what you were about to
  try. Future-you starts by reading, not by archaeology.
- **The message says why.** "fix bug" is noise. "store: return ErrNotFound
  instead of nil so callers can branch" is the reason you will search for in
  three weeks.

## Working the long middle without stalling

Twenty hours is long enough for the project to lose momentum. The stalls all
look different and are all the same thing — the next action stopped being
obvious.

- **Analysis paralysis.** Two designs look equal, so you compare them for a
  day. If they are equal, they are equal: pick the one that is easier to
  *reverse*, write an ADR saying you flipped a coin and why the coin was fair,
  move.
- **The refactor spiral.** A small change reveals a bad boundary, fixing it
  reveals another. Time-box it: land the smallest fix that makes the current
  milestone possible, and put the rest in the parking lot as a risk with a
  named trigger.
- **Yak shaving.** Making your editor perfect, writing a code generator to save
  twenty lines, building the tool that would build the thing. Fun, and it does
  not deliver a milestone.
- **The blank-page stall.** You do not know how to start the next piece. Write
  the test first — not for purity, but because a test forces you to name the
  function, its inputs, and what "worked" means, and that is the whole design
  decision.

Two mechanics that keep it moving: a **two-hour rule** — if a single task has
not moved in two focused hours, shrink it, spike it in a throwaway file, or ask
for a review — and a **next-action note**: the last thing you do each session is
write one line saying what you will do first next session. It costs a minute
and saves the twenty it takes to reload context.

## When to stop gold-plating

The definition of done for a milestone is its acceptance criteria, written
before you started. Not "as good as I can make it". Ship, then improve later
with evidence — that is what the rest of this stage is for.

Signals you have crossed from finishing into polishing:

- **Configurability nobody asked for.** Every option is a branch, a test case,
  and a documentation line, forever. Hard-code it until a second caller exists.
- **Abstraction with one implementation.** An interface with one implementer
  and no test double is a rename with extra steps. The rule of three: on the
  third occurrence, abstract. Not the first.
- **Generic where concrete works.** You know generics from S3. A concrete type
  you can read beats a type parameter you have to decode, until the second type
  actually arrives.
- **Work that contradicts a non-goal.** You wrote non-goals in the PRD
  precisely so that future-you, mid-build and enthusiastic, would have to argue
  with past-you in writing. Non-goals win by default; change one deliberately
  and record it in an ADR.

The opposite failure is real too. Stopping early is not discipline if what you
shipped panics on empty input. "Done" means the behaviour holds for the inputs
the PRD promised, including the ugly ones.

## Keeping tests honest while the design moves

The design *will* move under you — that is what a 20-hour build is. Tests
written against the old shape are the main reason people stop refactoring, so
write them so they survive.

- **Test at package boundaries, not at private helpers.** Exported behaviour is
  the contract; internals are the part you want free to change. A test that
  breaks every time you rename something unexported is testing the wrong thing.
- **Delete tests that describe a design you deleted.** Deleting a test is not
  losing coverage if the behaviour it described no longer exists. Keeping it is
  how suites rot into archaeology.
- **State the behaviour in the name.** `TestListFiltersAndKeepsOrder` tells the
  next reader what the code promises. `TestList2` tells them nothing.
- **No sleeping tests.** `time.Sleep` in a test is a race condition with good
  manners. You have channels, `sync.WaitGroup`, and injected clocks from S3 and
  S5; use them.
- **Keep the race detector on.** Run `go test -race ./...` as your normal
  command, not as a ceremony before review. A data race found the day you wrote
  it takes minutes; found three milestones later it takes an evening.

**Coverage is a floor, not a goal.** The harness requires 60% statement
coverage across the module. Below that, a suite demonstrably does not exercise
the code — whole packages are unvisited, and "the tests pass" means almost
nothing. Above it, the number stops carrying information: 95% coverage of
getters and 40% of your decision logic is a worse suite than the reverse.
Sixty is chosen to be clearable by an honest suite that covers the core and
leaves `main` and thin plumbing uncovered, and to be *un*clearable by a suite
that only tests the easy package. The moment you find yourself writing a test
to move the number rather than to pin down behaviour, you have turned a floor
into a target and the measure has stopped being one.

One measurement detail worth knowing, because the default will lie to you. By
default `go test -cover` credits a statement only to the package whose own
tests ran it: a test in `cmd/` that drives your whole application leaves every
package it exercised reported at 0%. That would punish exactly the tests worth
writing, so the harness measures with `-coverpkg=./...`, which credits every
package the run actually reached. Use the same flag when you check the number
yourself — otherwise your figure and the harness's will disagree and you will
chase the wrong gap.

## Package boundaries and dependency direction

The layout question you answered for the service in S5, now for a codebase
nobody handed you.

- **`main` is a composition root.** It reads config, constructs things, hands
  each its dependencies, and starts the run loop. It contains no rules, because
  everything in `main` is code your tests cannot reach.
- **Dependencies point inward, toward the domain.** The package that holds your
  types and rules imports none of your other packages. Storage and transport
  import it. Nothing imports them except `main`.
- **Interfaces are declared by the consumer**, next to the code that calls
  them, and stay small. An interface with eight methods is a class; an
  interface with two is a seam.
- **`internal/` for anything you are not promising to strangers.** The compiler
  enforces it, which makes it a real boundary rather than a naming convention.
- **No `util`, `common`, or `helpers`.** A package named for its lack of a
  subject collects everything and depends on everything. Name packages after
  what they *are*: `note`, `store`, `httpapi`, `config`.

The harness checks a deliberately weak proxy for all of this: at least two
packages outside `main`, holding at least 60% of your non-test Go lines. That
cannot tell a good boundary from a bad one — only your review can — but it does
catch the single failure mode that dooms projects this size, which is 2,000
lines in `package main`.

## Idiomatic Go, applied to a project you own

Nothing new here; the point is that it now has to hold across a whole codebase
rather than one exercise.

- **Errors carry context and stay matchable.** Wrap with `%w` and say what you
  were doing (`fmt.Errorf("load config %s: %w", path, err)`). Define sentinel
  errors at package boundaries so callers can use `errors.Is`. Handle an error
  or return it — never both. `main` is where errors become exit codes.
- **Context flows down, never sideways.** First parameter, passed to everything
  that can block, never stored in a struct. If your project has no cancellable
  work, say so in review rather than sprinkling `ctx` for decoration.
- **Concurrency only where it earns its cost.** Before every goroutine, answer
  three questions: what does it own, who waits for it, and how does it stop?
  If a sequential version is fast enough for the sizes in your PRD, sequential
  is the right answer and "we measured and it did not need it" is a strong
  review answer.
- **Zero values that work, and small honest types.** A struct usable without a
  constructor, a `Store` you can swap for a fake in one line, no field that has
  to be set "before you call anything".

## ADRs meet reality

You wrote at least three ADRs in the planning lesson. Half of them are about to
be wrong, which is the normal outcome and the reason the format exists.

When a decision changes, do not edit history: add an ADR that supersedes the
old one, set the old one's status to `superseded by ADR-00N`, and write down
what you learned that changed your mind. When a decision holds under pressure,
add a line to it saying so — "M3 review: held; the interface stayed at three
methods". In review you will be asked which ADRs held, which you revised, and
why, and "I don't remember" is the one answer that costs you.

## Exercise

Build the core of your capstone. The harness in [`exercise/`](exercise/) grades
whatever it finds at your project directory, resolved in this order:

1. `TUTOR_CAPSTONE_DIR` in the environment;
2. the first line of `exercise/capstone.path` (relative paths resolve against
   the exercise directory);
3. `projects/capstone` at your workspace root — the convention from the
   planning lesson, which the harness finds by walking up from its exercise
   directory.

If none of them is a directory containing a `go.mod`, every check fails with a
message telling you how to fix it. Run the harness on day one, on the walking
skeleton, so you can see the eight failures you are working towards clearing.
Grading runs with `GOPROXY=off`, so if your project has dependencies, run
`go mod download` in it once.

Acceptance criteria — the first eight are exactly what the harness checks:

1. **It builds.** `go build ./...` in your project exits zero.
2. **It vets clean.** `go vet ./...` exits zero, with no findings suppressed.
3. **Your suite passes under the race detector.** `go test -race ./...` in your
   project exits zero. A `DATA RACE` report is a failure, not a flake.
4. **Coverage clears the floor.** Total statement coverage across the module,
   measured by that same run with `-covermode=atomic -coverpkg=./...
   -coverprofile`, is at least **60.0%** as reported by `go tool cover -func`.
   `-coverpkg` means a test anywhere credits every package it reaches, so
   integration tests through `cmd/` count for the packages underneath.
5. **Real package structure.** At least **two** packages other than `main`
   (directories with non-test `.go` files), and at least **60%** of the
   module's non-test Go lines live outside `package main`.
6. **No unfinished work in non-test code.** No `TODO` or `FIXME` in any
   comment; no `panic("…")` whose message contains "not implemented",
   "unimplemented", "implement me" or "todo"; no commented-out code — a
   non-doc comment block whose text parses as Go statements or declarations.
   Test files are exempt, and doc comments may contain example code.
7. **A README a stranger can start from.** `README.md` at the project root, at
   least 200 bytes, containing at least one fenced code block showing how to
   build and run it.
8. **Milestones delivered.** `MILESTONES.md` at the project root with at least
   three checklist items (`- [ ]` / `- [x]`), every one of them ticked.

The rest is graded in review, at each milestone rather than at the end:

9. **The milestones are the ones you planned**, or the changes are explained.
   Scope you cut is a decision; scope that quietly vanished is not.
10. **The layout defends itself.** You can draw the import graph, say which
    package may import which, and name the interface that would let you swap
    your storage.
11. **Idiomatic Go throughout**: wrapped errors, context propagation, small
    consumer-declared interfaces, and a defensible answer for every goroutine
    (what it owns, who waits, how it stops).
12. **The tests are honest.** You can point at the tests that pin your core
    behaviour, and name a test you deleted because the design moved.
13. **ADR status is current** — at least one revised or explicitly upheld ADR,
    with the reason written down.

Run the harness from the exercise directory:

```sh
cd exercise
go test ./...                 # all eight checks
go test -run Coverage -v .    # coverage number on its own
```

And run your own project's checks constantly, from inside it:

```sh
go test -race ./...
gofmt -l .

# the same total the harness reads — per-package percentages are not it
go test -coverpkg=./... -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1
```

## Further reading

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Modules Reference — organizing a module](https://go.dev/ref/mod)
- [Go Blog — Package names](https://go.dev/blog/package-names)
- [Go Blog — Error handling and Go](https://go.dev/blog/error-handling-and-go)
