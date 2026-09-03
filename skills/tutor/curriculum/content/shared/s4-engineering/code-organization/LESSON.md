# Code Organization

> `shared.eng.code-organization` · ~2-3h · Stage: Engineering Practice

## Objectives

By the end of this lesson you can:

- Lay out a small multi-package project and justify what belongs in each
  package.
- Explain dependency direction and why stable, low-level packages must not
  import volatile, high-level ones.
- Detect an import cycle or wrong-direction dependency in a given project and
  restructure to remove it.
- Choose between one flat package and multiple packages for a given codebase
  size and justify the trade-off.
- Refactor a tangled single-file program into modules with clear boundaries
  while keeping its tests green.

## One level up from clean code

The Clean Code lesson worked at the scale of a name and a function. This
lesson moves one level up: given a program made of clean functions, *where
should each one live?* The question sounds cosmetic. It isn't. Organization
decides two things that dominate the life of a codebase:

- **Where a reader looks.** "The validation rules are in the expense module"
  is a map. "It's all in main, somewhere" is an archaeology dig.
- **What a change touches.** Good boundaries mean a change to the report
  format cannot break validation, because the two share no code and the
  compiler enforces it. Bad boundaries mean every change is a game of
  minesweeper.

Every language has some unit of organization above the function — a module, a
package, a namespace. The design questions are the same everywhere; only the
keyword changes.

In Go: the unit is the **package** — one directory, one package, as you saw
in S1's packages lesson. (Careful with words: Go also has *modules*, the
versioned unit with a `go.mod`. When this lesson says "module" in portable
theory, the Go realization is *package*.)

## What belongs together

A good module passes one test: **it has a single reason to change**, and you
can name it. "This package changes when the validation rules change." "This
one changes when we redesign the output format." That is cohesion — the same
property Clean Code demanded of functions, at a bigger scale.

Two consequences follow:

- **Name the module for what it provides, not what it contains.** `expense`,
  `report`, `parser` are jobs. `utils`, `helpers`, `common` are junk drawers:
  they name a *shape* ("small functions") instead of a *purpose*, so
  everything drifts into them and soon every part of the program depends on
  them — and through them, indirectly, on each other.
- **Small surface, hidden insides.** A module should export the few things
  callers need and keep the rest private. Every exported name is a promise
  you must keep; every hidden one is a decision you may still revise.

In Go: the surface is exactly the capitalized identifiers, and package names
read at the call site — `report.Summary(...)` explains itself, which is why
`report.ReportSummary` would be a bad name (stutter) and `utils.Format` a
useless one. Go also gives hiding teeth one level up: anything under a
directory named `internal/` is importable only within your module.

## Dependency direction

Draw your modules as boxes and every import as an arrow. The shape of that
picture is the architecture. For the exercise project — an expense reporter —
it should look like this:

```
            main  (wiring)
           /  |  \
          v   v   v
    report  ledger ──▶ expense
                        (core)
```

Two ends of every arrow differ in kind:

- **Low-level, stable** modules sit at the bottom: core data types, rules
  that define the problem itself. `expense` — what an expense *is* and when
  it is valid. Many things depend on it; it rarely changes.
- **High-level, volatile** modules sit at the top: policy, presentation,
  wiring. `main` and `report` — today the output is text sorted by category;
  next month it's JSON. Nothing should depend on them.

The rule: **arrows point from volatile toward stable — never the other way.**
Why? An import is inherited exposure: if stable `expense` imported volatile
`report`, then every change to the output format could recompile, break, or
re-test the core type that half the program sits on. The module everything
depends on would churn as fast as the module nothing should depend on. And
practically: you could no longer use or test `expense` without dragging the
whole presentation layer along.

A useful instinct: the further down a module sits, the more boring it should
be. If your most-imported module is also your most-edited file, the arrows
are lying somewhere.

## Import cycles

The degenerate case is an arrow pointing both ways: A imports B and B imports
A (directly, or via a chain). Now neither can be built, understood, or tested
without the other — they are one module wearing two names, with the
complexity of two. Some languages tolerate cycles at runtime and reward you
with initialization-order bugs.

In Go: the compiler simply refuses — `import cycle not allowed`. That error
is not the compiler being pedantic; it is a design smell made loud.

Three standard ways to break a cycle (or a wrong-direction edge — same
disease, milder form):

1. **Move the shared piece down.** If A and B both need `FormatCents`, it
   belongs in a third module below both — or in whichever of the two is
   genuinely lower.
2. **Pass data instead of importing.** The low module returns plain values
   (a map, a struct); the high module consumes them. The exercise's
   `report.Summary` takes a `map[string]int` — it needs numbers, not a
   `Ledger`, so it doesn't even have to import `ledger`.
3. **Invert with an interface.** When a low module genuinely must *call*
   upward (storage that notifies the UI), it defines an interface for what
   it needs and the high module implements it. The call goes up; the import
   still points down. You met this in S3: `io.Writer` is exactly this trick
   — `fmt` calls your type without importing your package.

Detecting the disease is mechanical: list each module's imports and check
every edge against the picture you *meant* to have. The exercise automates
this with a test that parses import lists — architecture as a failing test,
not a wiki page nobody reads.

## One package or many?

Splitting is not free, so "more packages = better organized" is false. Every
boundary costs you:

- a name and an import at every use site, and a hop for every reader;
- a **frozen surface** — exported names are commitments, and moving code
  across packages later is a breaking change for callers;
- boundary guesses made early, when you know the problem least.

So: **start flat.** One package is the right answer for a small program, and
for any codebase whose seams you cannot confidently name yet. Split when a
seam announces itself:

- a cluster of code has its own reason to change (format vs. rules);
- you want to *hide* a decision behind a small surface;
- you want the compiler to *enforce* a dependency rule, not just document it;
- two features keep colliding in the same files for unrelated reasons.

The exercise program is honestly near the borderline — small enough to stay
flat, tangled enough to show every mechanic of splitting. Real projects reach
this fork constantly; what matters is choosing deliberately and being able to
say why.

In Go: the standard library is the model — `strings`, `sort`, `net/http`:
each package one job, named for what it provides. And Go's culture leans
flat: a 500-line package is normal; ten 50-line packages are a smell.

## Refactoring with the tests green

Reorganizing must not change behavior — and "must not" is worthless without
proof. The proof is a test suite that stays green while the structure churns.
The discipline:

1. **Small moves.** Relocate one function or type at a time; compile and
   test after each move. Never "big bang" a reorganization.
2. **Let the compiler drive.** Move a piece, then chase the errors it causes
   — each one is a call site to update. When it compiles again, run the
   tests.
3. **Bottom-up.** Extract the module that depends on nothing first, then the
   one that depends only on it, and so on up to the wiring. Top-down strands
   you with pieces whose dependencies don't exist yet.
4. **Same behavior, new address.** Resist improving logic mid-move. One
   commit moves code, another changes it — a reviewer can verify each; a
   mixed one, neither.

## Exercise

Open [`exercise/`](exercise/) — a Go module containing a working expense
reporter written as one tangled blob in `main.go` (run it: `go run .`), plus
three package skeletons that give the concerns a proper home:

- `expense/` — the core record and its validation rules. Bottom of the
  project: imports nothing from this module.
- `ledger/` — stores validated expenses, computes totals. May import
  `expense` only.
- `report/` — turns computed numbers into text. Depends on plain data
  passed in, not on the packages that produce it.

Tests in each package specify the behavior to move there. Two tests at the
module root check the *architecture*: `TestDependencyDirection` parses every
package's imports against exactly the arrows drawn above (one starter
dependency already points the wrong way — find where that code truly
belongs), and `TestMainWiring` checks that `main` imports all three packages.
That second test is only the mechanical half of "thin wiring" — a compiler
can see imports, not intent — so whether the tangled original is truly gone
is settled by criterion 5 and your tutor's review, not by the test.

Acceptance criteria:

1. `expense.New` trims surrounding whitespace, rejects blank descriptions
   and categories and non-positive amounts with the package's sentinel
   errors (matchable via `errors.Is`), and otherwise returns the populated
   `Expense`.
2. `ledger.Ledger` works from its zero value: `Add` stores, `Len` counts,
   `Total` sums all amounts, `TotalByCategory` sums per category.
3. `report.FormatCents` renders cents as dollars (`350` → `"$3.50"`, `5` →
   `"$0.05"`); `report.Summary` emits one `category: amount` line per
   category in alphabetical order, then a `total:` line, each
   newline-terminated.
4. `TestDependencyDirection` passes: the wrong-way edge in the starter is
   gone and no new one appears.
5. `main.go` is wiring only: build each expense with `expense.New`, report
   invalid ones on stderr and skip them, `Add` the rest to a
   `ledger.Ledger`, print `report.Summary` — and delete the tangled
   original. `go run .` prints the same report as before the refactor.
6. `go test ./...` passes and everything is `gofmt`-formatted.

Run the checks from inside the exercise folder:

```sh
cd exercise
go test ./...
```

They fail before you start. Work bottom-up — `expense`, then `ledger`, then
`report`, then `main` — and re-run the tests after every move.

## Further reading

- [Go blog — Package names](https://go.dev/blog/package-names)
- [go.dev — Organizing a Go module](https://go.dev/doc/modules/layout)
- [The Clean Architecture — Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Go Wiki — Code Review Comments](https://go.dev/wiki/CodeReviewComments#package-names)
