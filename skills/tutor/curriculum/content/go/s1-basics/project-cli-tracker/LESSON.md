# Mini-Project: CLI Tracker

> `go.basics.project-cli-tracker` · ~5-8h · Stage: Programming Basics (Go)

## Objectives

By the end of this project you can:

- Design and implement a small CLI tracker combining structs, slices, maps,
  and methods.
- Persist tracker data to a file and load it back, handling missing or
  corrupt files with wrapped errors.
- Parse command-line arguments to support add/list/complete subcommands.
- Write table-driven tests covering the core tracker logic.
- Structure the project as a module with gofmt-clean code and a clear
  package layout.

## What you're building

This is the stage capstone. No new language features — instead, everything
from this stage meets in one program: a todo tracker you drive from the
terminal, which remembers your tasks between runs:

```sh
$ go run . add buy milk
added #1: buy milk
$ go run . add call grandma
added #2: call grandma
$ go run . list
[ ] #1 buy milk
[ ] #2 call grandma
$ go run . complete 1
completed #1
$ go run . summary
1 open, 1 done
```

The words after a subcommand come from `os.Args` — the slice of strings the
operating system hands your program. `os.Args[0]` is the program's own name;
everything after it is yours to interpret. That is the entire mechanics of a
CLI: a slice of strings in, text out.

Between runs, the tasks live in a plain text file, `tracker.txt`, created in
whatever directory you run the program from. Delete it and the tracker
starts fresh.

## The shape of the project

Unlike every lesson so far, this one hands you almost no implementation —
only types, function signatures, and tests. The layout is two packages, and
the split is the design lesson of the project:

```
project-cli-tracker/
├── go.mod
├── main.go        # package main: load → dispatch → save → print
├── cli.go         # package main: dispatch() turns args into tracker calls
├── cli_test.go
└── tracker/       # package tracker: the actual logic, importable, testable
    ├── tracker.go       # Task, Tracker, Add/List/Complete/Summary
    ├── tracker_test.go
    ├── store.go         # Save and Load
    └── store_test.go
```

Everything that *decides* lives in `tracker` — it never prints and never
touches `os.Args`. Everything that *talks* (arguments in, text out) lives in
`package main`. Remember why hello-world's `Greeting` returned a string
instead of printing it: the same move, one stage later. Because the logic
returns values, the tests can check it without running the binary.

## Order of attack

A project this size is not written top to bottom. Work in passes, keeping
the tests green behind you:

1. **The core** — `tracker/tracker.go`. Run `go test ./tracker` and make the
   tests in `tracker_test.go` pass. Start by designing `Tracker`'s fields:
   the struct is empty on purpose, and the tests only touch the exported
   API, so the internals are yours to choose. You need insertion order and
   ids that never repeat — a slice gets you further than you might think.
2. **Persistence** — `tracker/store.go`, against `store_test.go`.
3. **The CLI** — `cli.go`, against `cli_test.go`.
4. **Wiring** — `main.go`, checked by hand with `go run .`.

Read each test file before writing the code it tests. The tests are the
full specification; this page only summarizes them.

## The file format

`Save` writes one task per line — three fields separated by `|`:

```
1|open|buy milk
2|done|call grandma
```

The rules, which `Load` must enforce:

- The status is exactly `open` or `done`.
- The title is everything after the second `|`, so titles may themselves
  contain `|`. `strings.SplitN(line, "|", 3)` splits on the first two
  separators only — plain `Split` would shred those titles.
- Blank lines are skipped, not errors. (Splitting a file that ends in `\n`
  on `"\n"` gives you a trailing empty string — this rule makes that
  harmless.)
- A missing file is not an error: the first run has nothing to load, so
  `Load` returns an empty tracker. Check with
  `errors.Is(err, os.ErrNotExist)`, not by comparing message text.
- Anything else — wrong field count, non-numeric id, unknown status — is
  corruption, and `Load` fails.

One subtlety worth designing for: after loading a file whose ids are 3 and
7, the next `Add` must hand out 8. Ids never repeat, even across restarts —
"the highest existing id plus one" is the invariant, however you keep it.

## Errors that help

The errors lesson gave you sentinels and `%w`; here they earn their keep.
The `tracker` package declares two:

- `ErrNotFound` — `Complete` with an id nobody has.
- `ErrCorrupt` — `Load` met a line it cannot parse.

Callers must be able to *test* for these with `errors.Is`, and humans must
be able to *read* what happened. Wrapping gives you both at once:

```go
fmt.Errorf("load tracker: line %d: %w", i+1, ErrCorrupt)
```

`errors.Is` walks through the wrapping to find the sentinel; the message
still names the line. The tests check both — an error that satisfies
`errors.Is` but doesn't say which line is broken fails, and so does the
reverse. Notice what `dispatch` must therefore *not* do: swallow a tracker
error and invent its own. Return it (wrapped if you add context), so the
sentinel survives to the caller.

## Your toolbox

Nothing here is new, but you have never needed it all at once. You'll
likely reach for: `strings.TrimSpace`, `strings.Join`, `strings.SplitN`, a
`strings.Builder`, `strconv.Atoi`, `fmt.Sprintf`, `os.ReadFile` /
`os.WriteFile`, `append`, and `errors.Is` / `fmt.Errorf` with `%w`. In the
tests you'll meet `t.TempDir()` — it hands each test a fresh directory and
deletes it afterwards, so file tests never touch your real `tracker.txt`.

## Exercise

Open [`exercise/`](exercise/) and read the three test files first.

Acceptance criteria:

1. `Add` assigns ids starting at 1, one higher than the highest existing id;
   titles are trimmed; an empty or whitespace-only title is an error.
2. `List` returns tasks in insertion order, as a copy — mutating the
   returned slice must not change the tracker.
3. `Complete` marks the task done; completing twice is fine; an unknown id
   returns an error satisfying `errors.Is(err, ErrNotFound)`.
4. `Summary` returns a map that always contains both `"open"` and `"done"`.
5. `Save` and `Load` roundtrip every field through the file format above;
   a missing file loads as an empty tracker; a corrupt file fails with an
   error that wraps `ErrCorrupt` and names the line; blank lines are
   skipped; ids continue after a load.
6. `dispatch` produces the exact outputs shown in `cli_test.go` for `add`,
   `list`, `complete`, and `summary`, errors on bad input, and passes
   tracker errors through.
7. `main` wires it together: load, dispatch on `os.Args[1:]`, save, print;
   errors go to `os.Stderr` with exit code 1. Verify by hand — run the
   whole transcript above, then run it again and confirm the tasks
   survived; then corrupt a line of `tracker.txt` in your editor and watch
   the error name it.
8. Add one table-driven test of your own, with at least three cases, for a
   behavior you choose (more corrupt-file shapes are a good target).
9. `go test ./...` passes and everything is gofmt-formatted.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...          # the full suite
go test ./tracker      # just the core, for pass 1
```

Expect a wall of red at first — that wall is your task list. Work it top
to bottom, one test at a time.

## Further reading

- [Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)
- [pkg.go.dev — os](https://pkg.go.dev/os) (`Args`, `ReadFile`, `WriteFile`)
- [pkg.go.dev — strings](https://pkg.go.dev/strings)
- [go.dev — Organizing a Go module](https://go.dev/doc/modules/layout)
