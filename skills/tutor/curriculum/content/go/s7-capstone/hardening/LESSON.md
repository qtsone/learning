# Capstone: Hardening

> `go.capstone.hardening` · ~8-12h · Stage: Expert Capstone (Go)

## Objectives

By the end of this lesson you can:

- Deepen your test suite with integration tests, fuzz targets at the input
  boundaries, and concurrent paths that stay clean under the race detector.
- Run a security pass over your own code: input validation, secret handling,
  dependency vulnerabilities with `govulncheck`, and a written record of what
  you found and what you fixed.
- Profile your project under a load you consider realistic and name its top CPU
  and allocation hotspots.
- Fix at least one defect the pass turned up, and pin it with a regression test
  that fails against the old code.
- Justify what you chose *not* to harden, and say out loud what risk you
  accepted by choosing that.

## The standard changes

Until now the question was "does it work". The milestones are ticked, the suite
is green, the thing runs. That is the standard for a demo.

The standard now is: **does it hold** — when the input is hostile, when the
peer never answers, when the caller hangs up mid-request, when a dependency
turns out to have had a hole in it since last March. Nothing in this lesson
adds a feature.

Two habits carry it. You go looking for trouble on purpose rather than waiting
for it to arrive with a user attached; and everything you find gets written down
— the fix, the test that pins it, the risks you decided to live with. An
undocumented decision to accept a risk is indistinguishable from not having
noticed it.

## What "done" means for tests

You cleared a 60% coverage floor last lesson and you already know why it is a
floor: it counts lines the tests *executed*, not behaviours they *checked*. A
test that calls a function and asserts nothing scores identically to one that
pins the contract.

The ceiling is a different question, and the tool for thinking about it is
**mutation thinking**. Pick a line and imagine changing it: flip a `<` to `<=`,
delete an early return, swap `&&` for `||`, return `nil` instead of the error.
Now ask: *which test goes red?* If the answer is "none", that line is executed
by your suite but not defended by it. No mutation-testing tool required —
asking the question over your ten most important lines finds real gaps in an
hour.

Where the gaps usually are: **error paths** (every function that can fail has a
caller that handles the failure, and that handler is usually untested — most of
"the tests passed but production burned" lives here); **boundaries** (zero, one,
many, exactly the limit, one past it, empty, nil); **concurrent interleavings**
(one goroutine works, two contending do not); and **the seams between
packages**, where both unit suites pass and the two halves still disagree about
who trims the whitespace. That last one is what an integration test is for:
exercise a real user-visible operation through the real wiring, with only the
outside world faked.

The harness therefore requires assertions on failure, not only on success:
`errors.Is`, `if err == nil { t.Fatalf(…) }`, a `wantErr` column. Four across
two files is a deliberately low bar — it catches the suite that never once asked
what happens when things go wrong, and stays out of the way of one that has.

## Write the failure case first

The discipline that makes the rest of this cheap: **when you find a defect, the
first thing you write is the test that reproduces it.** Not the fix. The test.

You saw this in S4's debugging lesson as a rule about bugs. It buys three
things at once: it proves you understood the defect, because a fix aimed at the
wrong cause is much harder to spot than a test that fails for the wrong reason;
it gives "is it fixed?" an answer that is not a matter of opinion; and it leaves
behind the only artifact that stops the bug returning during a refactor.

Name it for the behaviour it defends — `TestParseRejectsPathTraversal`, not
`TestBug17` — and keep it beside that package's other tests rather than in a
`regressions_test.go` dumping ground. Regressions are not a category of test;
they are tests that happen to have been written in anger.

## Fuzzing your own boundaries

S5 taught the mechanics: `FuzzXxx(f *testing.F)`, `f.Add` seeds, coverage-guided
mutation under `-fuzz`, minimised crashers written into `testdata/fuzz/<Target>/`.
New here is choosing the target in a codebase nobody designed for the exercise.

Fuzz where **bytes you did not write become a value you trust**: a parser or
decoder, a function that turns a string into a path, query, command or URL,
anything hand-written over a wire format, anything computing a size or an index
from input. If your project reads JSON with `encoding/json` and does nothing
else with the bytes, fuzz the *validation* that turns the decoded struct into
your domain type — that is your code, and that is where your bugs are.

Then write a property that must hold for **every** input, not for the inputs
you thought of:

- *It never panics.* The weakest property and often the first to fall — index
  out of range, nil map write, slice bounds from an untrusted length.
- *Accepted output satisfies the invariants.* If the parse returned no error,
  the id matches the id rules, the title is within limits, the tags are sorted
  and unique. This is the property that catches validation you thought you had.
- *Round trips are identity*: `parse(render(x))` equals `x`. And *two paths
  agree*: the fast path and the obvious path return the same answer.

Commit the corpus. The **seed corpus** on disk (`testdata/fuzz/FuzzX/`) runs on
every plain `go test`, on every machine, forever; the generated corpus lives in
your build cache and is yours alone. When the fuzzer finds a crasher it writes
the minimised input into the seed corpus — commit that file in the same change
as the fix and you have a regression test the machine wrote for you. Entries can
also be hand-written: one line of `go test fuzz v1`, then one typed literal per
parameter. Run the target for real at least once, naming the package it lives
in:

```sh
go test -fuzz=FuzzParseLine -fuzztime=60s ./internal/note
```

`-fuzz` accepts a **single package** — `./...` fails with "cannot use -fuzz
flag with multiple packages" — and that restriction says what the tool is for.
Fuzzing is aimed at one boundary at a time, not sprayed across a suite; the
suite-wide run is the plain `go test ./...` that replays your committed corpus.
Never put `-fuzz` in an automated job either — unbounded work with a
nondeterministic result is a job people learn to ignore.

## The self-review pass

Now read your own code the way an attacker would, using S4's map. The OWASP
Top 10 is written for web applications; the categories below are the ones that
reach a project of this size, whatever shape it is. Secret failures are the
fourth, and get the section after this one.

**Injection — never concatenate into an interpreter.** An interpreter is
anything that parses a string and acts on it: SQL, a shell, a template, a
regexp, an HTTP URL, a file path. Parameterised queries, `exec.Command` with
separate arguments (never `sh -c`), `filepath.Join` plus an explicit check that
the result is still inside the directory you meant. The path case is the one
capstones fail: an identifier that came from outside and ends up in a filename
is a directory traversal waiting for `../`.

**Broken access control and unsafe defaults.** Who is allowed to do this, and
what happens if the answer is "whoever can reach the port"? A single-user CLI
has a legitimate answer — "anyone who can run it already has my shell" — but it
has to be an answer you wrote down, not one you never asked. Same for the
listening address: `localhost` and `0.0.0.0` are very different decisions.

**Resource exhaustion.** Every input has a size and every size needs a limit:
the length of a line, the number of elements in a decoded list, the size of a
response body you read from a peer, the capacity of a slice you allocate from a
number in the input. Without a limit, the input decides how much memory your
program uses. `io.LimitReader`, an explicit `MaxBytes` constant, and
`bufio.Scanner.Buffer` are the tools; `make([]T, n)` with `n` from the wire is
the anti-pattern.

Do the pass with a written checklist in a fixed order: every input surface, then
every place a value crosses into an interpreter, then every place a secret could
be printed, then every unbounded allocation. Reading "generally" finds nothing;
walking a list finds things.

## Timeouts, and the context you were given

Two rules the harness checks mechanically, because both are invisible in review
and fatal in production.

**Every outbound HTTP client has a `Timeout`.** `http.DefaultClient` has none,
and neither do `http.Get`, `http.Post` and friends, which use it. A client with
no timeout promises to wait forever for a peer that may never answer, pinning
the goroutine, the connection and everything they hold. `Client.Timeout` bounds
the *whole* exchange — connect, TLS handshake, request, response headers and
body — which is why per-phase transport timeouts are not a substitute: each can
pass while the call still hangs. Put it in the literal, where it is visible:

```go
var client = &http.Client{Timeout: 10 * time.Second}
```

**A function handed a `context.Context` never creates a new root.** Calling
`context.Background()` inside a function that already has a `ctx` throws away
the caller's deadline, their cancellation, and everything above them: the user
hangs up, and your work carries on against a database nobody is waiting for.
Pass `ctx` down. When you genuinely want work to outlive the request — flushing
a buffer, finishing an audit write — that is `context.WithoutCancel(ctx)` plus a
timeout of its own, which says the intent out loud. Roots belong in `main`, or
in the `run` function `main` calls, where the program's lifetime actually
starts. `context.TODO()` belongs nowhere in finished code — it is a marker
saying "someone must decide which context goes here", and shipping it means
nobody did. (The previous lesson's hygiene scan will not catch it for you: it
reads comments and `panic` strings, not call expressions. This lesson's
harness does.)

The same instinct applies beyond HTTP, where no check can see it: a database
call, a channel receive, a `select` with no `ctx.Done()` branch. Anything that
can block should have an answer to "for how long, at most?"

## Secrets, and the blast radius of a leak

From S4: secrets come from the environment or a secret manager, never from
source, and never from a flag — flags are visible in the process list of every
user on the box. Two things worth re-deriving against your own project.

**A secret is anything that grants access**, which is broader than "password":
API tokens, webhook URLs whose token sits in the query string *or in a path
segment*, database DSNs, signed cookies, private keys, and the config file
holding them. Enumerate yours.

**Leaks happen through errors and logs, not source control.** The classic shape
is wrapping an error that quotes the input — `url.Parse` puts the whole URL in
its error text, and now your token is in every log that catches it. Validate at
the boundary and produce *your* error, redacting as you go; then grep your own
logging calls for anything that logs a whole request, config struct, or error
chain from a client. And if you committed a secret before you knew better,
rotating it is the fix — deleting the line is not, because the object is still
in history and on every clone.

## Dependency hygiene

`govulncheck` is the Go team's vulnerability scanner, and its distinguishing
feature is that it reports *reachability*: not "you depend on a module with an
advisory" but "a path exists from your code to the vulnerable symbol". That
makes its output short enough to actually read.

```sh
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

Run it by hand and read every finding — the harness cannot run it for you,
because it needs the vulnerability database and grading is offline. For each
finding: is the vulnerable symbol reachable, does a fixed version exist, does
upgrading break you? A finding you decided not to act on is fine, written down;
unread output is not. Re-run it after every toolchain upgrade — the standard
library gets advisories too — and after changes to your own code, because
reachability depends on the call graph you just moved.

While you are here: `go mod tidy`, then read what is left. Every direct
dependency is a decision — maintained, licence, size of the tree it drags in.
Your policy goes in the security document, and "standard library only" is a
perfectly good one to have to defend.

## Reconnaissance with pprof

You know the profiling loop from S5. Here you use only the first half of it:
**measure, and write down what you find.** Do not optimise yet — a later lesson
in this stage does that properly, and optimising before you have a suite you
trust is how you make something fast and wrong.

Drive your project with a load you can defend as realistic — the sizes from
your PRD, not ten items — and capture a CPU and an allocation profile:

```sh
go test -run='^$' -bench=. -cpuprofile cpu.out -memprofile mem.out ./internal/store
go tool pprof -top -nodecount=15 cpu.out
go tool pprof -top -nodecount=15 -sample_index=alloc_space mem.out
```

`-run='^$'` is the idiomatic way to say "no tests": a pattern that matches no
name, so only the benchmarks run and the profile is not diluted by test setup.
The profile flags, like `-fuzz`, take **one package at a time** — `./...` fails
with "cannot use -cpuprofile flag with multiple packages". That is not in your
way: profile the layer you believe owns the time, read the top entries, and if
they point somewhere else, profile that package next. Top-down, one layer at a
time, is the discipline anyway.

For a service, `net/http/pprof` on an internal port gives you the same from a
running process. Write down the top few entries of each and ask the only
question that matters at this stage: *did I expect that?* All
`runtime.mallocgc` and `runtime.growslice` says you are allocating in a loop;
encoding or reflection says you are paying for convenience; your own domain code
is the healthy case. Keep the numbers and the profile files — you will want the
"before" side of that comparison later.

## Writing it down: the security document

The deliverable of the pass is `SECURITY.md` (or `THREAT-MODEL.md`) at your
project root, with six sections, written for someone who has to operate, extend
or attack your project without you in the room.

- **Trust boundaries** — every place data crosses from something you do not
  control into something you do, and what is checked at the crossing.
- **Inputs and validation** — each input surface (arguments, files, stdin,
  network, environment), the validation it gets and the limits it enforces. A
  table beats prose here.
- **Secrets handling** — what counts as a secret here, where it comes from, and
  what stops it reaching logs, error strings or the repository.
- **Dependency policy** — the rule you apply before adding a module, and the
  `govulncheck` run you read: when, against what, and what it said.
- **Findings and fixes** — what the pass turned up and what you did, each with a
  `Regression test: TestName` line. The harness checks one of those exists.
- **Accepted risks** — below.

Length is not the point: four honest sentences per section beat two pages of
security theatre.

## What you chose not to harden

You cannot fix everything, and pretending otherwise is how the whole document
becomes fiction. The senior move is to be explicit: *this is the risk, this is
why I am carrying it, this is what would change my mind.*

A good accepted risk names three things — the exposure ("webhook responses are
not authenticated"), the reason ("one wrong exit code is the entire blast
radius; a trust store is a week"), and the trigger that reopens it ("the first
time this runs unattended"). Compare that with "we didn't have time", which
tells a reader nothing and future-you less.

Two failure modes: **accepting a risk you never measured** — you have to know
the blast radius before you can call it small — and **accepting a risk that
contradicts your own PRD**. If requirement F3 promises multi-user access and
your accepted risks say authentication is out of scope, one of the two documents
is lying, and you get asked which in review.

## Exercise

Harden your capstone. The harness in [`exercise/`](exercise/) grades whatever it
finds at your project directory, resolved in the usual order:
`TUTOR_CAPSTONE_DIR`, then the first line of `exercise/capstone.path`, then
`projects/capstone` at your workspace root, which the harness finds by walking
up. Grading runs with `GOPROXY=off`, so run `go mod download` in your project
once if it has dependencies.

Acceptance criteria — the first nine are exactly what the harness checks:

1. **The suite still passes under the race detector.** `go test -race ./...` in
   your project exits zero.
2. **It vets clean.** `go vet ./...` exits zero, with nothing suppressed.
3. **At least one fuzz target with a committed seed corpus** in
   `testdata/fuzz/<Target>/` next to the test that declares it.
4. **Error paths are tested**: at least four assertions on failure
   (`errors.Is`/`errors.As`, `if err == nil { t.Fatalf(…) }`, or a `wantErr`
   table column) across at least two test files.
5. **Hostile input is tested.** If your project takes input from outside itself,
   at least one test is named for the hostile case (`…Malformed…`, `…Hostile…`,
   `…Traversal…`, `…Oversize…`), or three or more fuzz corpus entries are
   committed. Projects with no external input surface skip this check.
6. **Every outbound HTTP client has a `Timeout` in its literal**, and no
   non-test code uses `http.DefaultClient` or the `http.Get`/`Post` helpers.
   Projects making no HTTP calls pass trivially.
7. **Context discipline**: no `context.TODO()` in non-test code, and no
   `context.Background()` inside a function that already receives a
   `context.Context`.
8. **A security document** — `SECURITY.md` or `THREAT-MODEL.md`, at the root or
   in `docs/` — with all six sections above, each carrying real prose (the
   harness enforces that as a floor of 120 bytes of body per section: enough
   to rule out a heading with one word under it, and no trouble at all for a
   section that says something), and the dependency section naming
   `govulncheck`.
9. **Every fix you pin is pinned by a test that exists**: each
   `Regression test: TestName` line in that document must name a test present
   in your project — all of them, not just the first.

The rest is graded in review:

10. **You can demonstrate the fix**: `git stash` it, run the regression test,
    show it red against the old code.
11. **Mutation thinking, applied**: pick a line of your core logic, name the
    change that would break it, and name the test that would catch it.
12. **An integration test exists** that exercises one real user-visible
    operation through the real wiring, with only the outside world faked.
13. **The self-review pass was systematic**: you can produce the checklist, and
    say which categories did not apply to your project and why.
14. **The profiles are captured and read**: top CPU and top allocation entries
    written down, with a sentence on which surprised you. No optimisation yet.
15. **The accepted risks are defensible**: exposure, reason, and the trigger
    that reopens each one — and none of them contradicts your PRD.

Run the harness from the exercise directory, and your own checks constantly
from inside your project:

```sh
cd exercise && go test ./...          # all nine checks
```

```sh
go test -race ./...                   # in your project
go test -fuzz=FuzzYourTarget -fuzztime=60s ./internal/yourpkg   # one package
go vet ./...
govulncheck ./...
```

## Further reading

- [Go Blog — Fuzzing is beta ready](https://go.dev/blog/fuzz-beta) and the
  [fuzzing tutorial](https://go.dev/doc/tutorial/fuzz) — targets, corpora,
  and how minimisation works.
- [Go Blog — Govulncheck](https://go.dev/blog/govulncheck) — what reachability
  analysis buys you over a plain dependency scanner.
- [pkg.go.dev — net/http.Client](https://pkg.go.dev/net/http#Client) — read the
  `Timeout` field's documentation in full; it says exactly what it bounds.
- [OWASP Top 10](https://owasp.org/www-project-top-ten/) — the map from S4,
  now applied to code you wrote.
