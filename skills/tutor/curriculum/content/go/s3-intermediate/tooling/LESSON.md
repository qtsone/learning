# Go Tooling

> `go.intermediate.tooling` · ~2-3h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Explain what classes of bugs `go vet` and staticcheck catch that the
  compiler does not, with concrete examples.
- Implement a golangci-lint configuration for a project and interpret its
  findings.
- Explain module version selection: semantic versioning, minimal version
  selection, and what `go.sum` guarantees.
- Explain what pprof profiles exist (CPU, heap) and when you would reach for
  each.
- Choose appropriate `go` commands (`mod tidy`, `work`, `list`, `generate`)
  for given maintenance scenarios and justify them.

## The compiler's blind spots

You have leaned on the Go compiler since S1: wrong type, missing return,
unused local variable — it refuses to build. That safety net is real, but it
only covers what the *language* forbids. Plenty of code is perfectly legal Go
and still wrong:

```go
fmt.Sprintf("user %s has id %s", u.ID, u.Name) // %s with an int — compiles
func (c SafeCounter) Value() int { … }          // copies the mutex — compiles
ctx, _ := context.WithTimeout(parent, 2*time.Second) // leaks the context — compiles
Name string `json: "name"`                      // broken tag — it's just a string
```

Each of these type-checks. `Sprintf` takes `...any`, so any argument fits any
verb. Copying a struct — mutex included — is legal copy semantics, even
though a copied lock guards nothing (remember the sync lesson: a `Mutex` must
never be copied after first use). Nothing in the language *requires* you to
call a cancel function. And a struct tag is an arbitrary string; `encoding/json`
silently ignores one it can't parse, so that field quietly stops renaming.

Catching this class of bug is the job of **static analysis**: programs that
read your program and check facts the compiler doesn't. Go's culture leans on
them hard — they run in editors, in pre-commit hooks, and in CI, and a
finding is treated much like a failing test.

## go vet: the analyzer in the box

`go vet` ships with the toolchain — no install, no config:

```sh
go vet ./...
```

It runs a curated set of checks with a strict admission rule: near-zero false
positives. If vet flags something, it is almost certainly a bug. All four
examples above are vet findings — `printf` (verb/argument mismatch),
`copylocks` (a `sync.Mutex` passed or received by value), `lostcancel` (a
discarded cancel function), `structtag` (malformed tags).

You have already met vet without noticing: `go test` runs a subset of vet
checks before your tests, which is why a bad `Printf` can fail `go test`
with a vet error rather than a test failure.

## staticcheck: the second opinion

[Staticcheck](https://staticcheck.dev/) is the community's standard deeper
analyzer. Install it like any Go tool:

```sh
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

Every finding carries a check code you can look up at
[staticcheck.dev/docs/checks](https://staticcheck.dev/docs/checks/). The
families tell you what kind of problem you're looking at:

- **SA — correctness.** Real bugs vet doesn't cover: a value assigned and
  never read (`SA4006`), a deprecated API like `strings.Title` (`SA1019`),
  impossible comparisons, misused standard-library calls.
- **S — simplifications.** Legal but baroque code: `if x == true`,
  `for _ = range xs`, hand-rolled `strings.Contains`.
- **ST — style.** Naming and doc-comment conventions beyond gofmt's reach.
- **U — unused.** Unexported functions and fields nothing references. The
  compiler rejects an unused *local variable* but says nothing about an
  unused *function* — dead code accumulates silently without this check.

Where vet optimizes for "never wrong", staticcheck optimizes for "thorough".
The two overlap very little; serious projects run both.

## golangci-lint: one runner for all of them

Real projects run many linters — vet, staticcheck, errcheck (unchecked
`error` returns), and dozens more. Running each by hand doesn't scale, so the
ecosystem standardized on [golangci-lint](https://golangci-lint.run/): a
single binary that runs many linters in parallel, caches results, and reads
one config file at the repo root. Install it via Homebrew
(`brew install golangci-lint`) or their install script, then:

```sh
golangci-lint run
```

Configuration lives in `.golangci.yml`. A deliberate, minimal one:

```yaml
version: "2"
linters:
  default: none
  enable:
    - govet
    - staticcheck
    - errcheck
    - unused
```

`default: none` plus an explicit `enable` list means the team *chose* every
linter — findings stay signal, not noise. Each output line ends with the
linter that produced it:

```
lintlab/lintlab.go:42:9: Error return value of `f.Close` is not checked (errcheck)
```

Read it like a compiler error — file, line, column — then the message, then
the linter to consult for docs. Sometimes a finding is a false alarm in
context; the escape hatch is a targeted comment, and discipline demands the
linter name *and* a reason:

```go
defer f.Close() //nolint:errcheck // read-only file; close error is uninteresting
```

A naked `//nolint` suppresses every linter forever and tells the next reader
nothing. Treat each one as a tiny code-review debt.

## pprof: measuring instead of guessing

In S2 you timed algorithms to see Big-O in the wild. **pprof** is the grown-up
version: a profiler built into the toolchain that answers "where does this
program actually spend its resources?" Two profiles matter today:

- **CPU profile** — while the program runs, it is sampled ~100 times per
  second: which function is executing right now? Reach for it when the
  program is *slow* and you need to know where the time goes.
- **Heap profile** — a sampled census of memory allocations: which call
  sites allocated the bytes currently live (or allocated in total)? Reach for
  it when memory *grows*, or when garbage-collection pressure makes
  everything a little slow because you allocate too much.

The cheapest way to feed pprof is a **benchmark** — a test-file function the
tooling runs in a calibrated loop:

```go
func BenchmarkJoinNaive(b *testing.B) {
	for range b.N {
		JoinNaive(corpus)
	}
}
```

`go test -bench` picks `b.N` large enough for stable timing and reports
ns/op; `-benchmem` adds allocations per op. Add `-cpuprofile` /
`-memprofile` and the run writes profile files you open with:

```sh
go tool pprof -top cpu.out
```

`-top` ranks functions by **flat** time (spent in the function itself) versus
**cum** (cumulative — including everything it called). A function with huge
cum and tiny flat is a manager delegating work; big flat means the work
happens right there. (Long-running servers expose the same profiles over HTTP
via `net/http/pprof` — file that away for the capstone and beyond.)

The rule this lesson plants: **never optimize without a profile.** Guesses
about hot spots are famously wrong; the profile is the evidence.

## Modules, deeper

Since S1 you've typed `go mod init` and moved on. Time to understand the
machinery. Modules are versioned with **semantic versioning**:
`vMAJOR.MINOR.PATCH`. Patch = fixes, minor = compatible additions, major =
breaking changes — and a breaking major release becomes a *different import
path* (`…/v2`), so two majors can coexist in one build.

When your dependencies disagree — you require `sampler v1.3.0`, but `quote`
requires `sampler v1.3.1` — Go resolves it with **minimal version
selection (MVS)**: for each module, take the *maximum of the minimum versions
anyone requires*. Here that's v1.3.1: the lowest version satisfying
everybody. Note what Go does *not* do: it never silently grabs the newest
release (v1.99.0 may exist; nobody asked for it). Builds stay reproducible
from `go.mod` alone, upgrades happen only when you run `go get`, and the
algorithm is simple enough to explain in a sentence — no dependency solver,
no surprise resolutions.

So what is `go.sum` for, if `go.mod` already pins versions? **Integrity, not
selection.** It records cryptographic hashes of every module version you've
used. On every later download — your machine, a teammate's, CI — Go
recomputes the hash and refuses to build if the bits differ. If someone
republishes `sampler v1.3.1` with altered contents (or a proxy tampers with
it), the build fails loudly instead of running attacker-modified code. New
modules are additionally checked against Go's public checksum database.
`go.sum` is not a lockfile — `go.mod` + MVS already decide versions — it is
tamper evidence. Commit both files, always.

## The maintenance toolbox

Four commands cover most day-to-day module maintenance. Know which scenario
calls for which:

| You need to… | Reach for |
|---|---|
| Sync `go.mod`/`go.sum` with reality — drop requirements nothing imports, add ones your code now uses | `go mod tidy` |
| Hack on two of your modules side by side, with one using your local checkout of the other | `go work init` (workspaces — no `go.mod` edits to remember to revert) |
| Ask the build questions: which version of X is in use? which packages exist under ./…? | `go list -m all`, `go list ./...` |
| Re-run code generators declared in `//go:generate` comments (mocks, stringers, protobufs) | `go generate ./...` |

Two more you'll use in the exercise: `go mod why <module>` explains the
import chain that drags a dependency in, and `go mod graph` dumps the whole
requirement graph.

## Exercise

Open [`exercise/`](exercise/) — a guided tour, not a build-from-scratch task.
`README.md` has six parts: you run the tools against a small module with
planted bugs, fix what they find, profile a slow function, and take a modules
field trip in a scratch directory. Record everything in `NOTES.md`; the
discussion with your tutor is the verification.

Acceptance criteria:

1. `go vet ./...` findings for `vetlab/` are recorded in `NOTES.md` — for
   each: what is wrong, why the compiler allowed it — then fixed, and
   `go vet ./...` exits clean.
2. staticcheck's findings for `lintlab/` are recorded with their check codes
   and fixed; `staticcheck ./...` exits clean.
3. You wrote `.golangci.yml` yourself, `golangci-lint run` caught the
   unchecked errors in `WriteReport` (which vet and staticcheck missed), and
   after your fix the run is clean.
4. You benchmarked both `Join` functions, captured CPU and heap profiles of
   the naive one, and `NOTES.md` names the function dominating each profile
   and explains *why* the builder version wins.
5. The modules field trip is complete: `NOTES.md` answers the MVS question
   with reasoning, and describes what happened when you tampered with
   `go.sum`.
6. Every scenario in part 6 has a chosen command and a one-line
   justification.

Run everything from inside `exercise/`; the README gives exact commands per
part.

## Further reading

- [go.dev — cmd/vet documentation](https://pkg.go.dev/cmd/vet)
- [staticcheck.dev — checks index](https://staticcheck.dev/docs/checks/)
- [golangci-lint — documentation](https://golangci-lint.run/)
- [Go blog — Profiling Go programs](https://go.dev/blog/pprof)
- [Go modules reference — Minimal version selection](https://go.dev/ref/mod#minimal-version-selection)
