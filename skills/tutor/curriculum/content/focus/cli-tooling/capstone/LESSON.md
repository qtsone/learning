# CLI Capstone

> `focus.cli.capstone` · ~6-10h · Stage: Focus: CLI Tooling

## Objectives

By the end of this lesson you can:

- Design and ship a complete CLI tool combining layered configuration, a cobra
  command tree, and TTY-aware terminal output.
- Handle signals and external processes robustly so the tool exits cleanly with
  meaningful exit codes.
- Test command logic with table tests by injecting `io.Reader`/`io.Writer`
  dependencies instead of touching global state.
- Cut a tagged release with cross-platform binaries and embedded version
  metadata.
- Defend the tool's UX decisions (flag names, error messages, exit codes, help
  text) in a design review.

## The brief

You are building `scout`, a tool that answers two questions about a directory
without leaving the terminal: *what is in it?* and *who has been working on it?*

```
$ scout scan ~/src/myproject
ext         files    lines      bytes
.go            42     6104     148903
.md             7      612      18220
(none)          3       44        901
total          52     6760     168024

$ scout authors ~/src/myproject --limit 2
    91  Ada Lovelace
    34  Grace Hopper

$ scout scan . --json | jq '.exts[0]'
{"ext":".go","files":42,"lines":6104,"bytes":148903}
```

It reads the disk and shells out to `git`; it never touches the network. Every
piece of it is something this pack taught you, and the capstone is the part
nobody can teach: making them agree with each other.

- **A command tree** (cobra) whose commands are constructors, not globals.
- **Layered configuration** — defaults < config file < environment < flags —
  resolved once, into one struct, before any handler does work.
- **Output that behaves** — a table for humans, JSON for scripts, colour only
  when someone is watching, diagnostics on stderr and data on stdout.
- **A child process** (`git log`) run with separate streams, a working
  directory, its exit code interpreted rather than discarded.
- **Cancellation** — one Ctrl-C stops the walk, kills the child, and reports.
- **A version flag** carrying the metadata the linker patched in, ready for the
  release you cut in the distribution lesson.

The tests in `exercise/` are the specification, and they are unusually detailed
on purpose: at this point in the pack, "what exactly should this print?" is a
decision you should be able to read off a test and defend afterwards.

## Where the decisions live

Seven files, and each one has exactly one job:

```
main.go      the process: signals, os.Getenv, UserConfigDir, is stdout a tty
  └─ Execute(ctx, app, args, out, errw) int        ← the seam; tests start here
       └─ NewRootCmd(app)                          cobra tree, one per run
            └─ handler: app.Resolve(cmd) ─▶ Settings
                        Scan / App.Authors         the work (ctx-aware)
                        Renderer                   the bytes
```

`main` is the only file that imports `os/signal` and reads a real environment
variable. Everything below it receives what it needs — a `Getenv` function, a
config directory, a VCS command, a `bool` for "stdout is a terminal" — as data
on `App`. That is the same move you have made in every lesson of this pack, and
here is what it finally buys: **the entire program is reachable from a test with
no subprocess, no `os.Args`, no environment mutation, and no terminal.**

`Execute` is the new piece. `main` shrinks to five lines because `Execute` owns
the three things `main` used to: build the tree, run it, turn the error into an
exit code. It is also the *only* place that prints an error, which is why every
handler can return errors and no handler ever writes to stderr.

## Exit codes are an API

Your S3 link checker returned `int` from `Run` and you probably picked 0 and 1
without much thought. A tool people script against needs a smaller, stabler
contract, and `scout` publishes this one:

| code | meaning | how the shell reads it |
|---|---|---|
| 0 | success | `scout scan && deploy` proceeds |
| 1 | the work failed | a real failure: unreadable directory, git blew up |
| 2 | you called me wrong | bad flag, bad value, missing config file |
| 130 | interrupted, deadline included | 128 + SIGINT, what your shell reports for a killed job |

The distinction that matters is 1 vs 2. A CI job that gets 2 has a broken
invocation and will keep getting 2 until a human edits the command line;
a job that gets 1 might succeed on retry. Collapsing them throws that away.

The fourth row deliberately differs from `runner`, last lesson's tool, which
gave an expired deadline `timeout(1)`'s own 124. `runner` had a `-timeout`
flag, so 124 reported something *it* decided: I gave up on schedule. `scout`
has no such flag, so the only way its context can expire is that something
outside it — `timeout 30 scout scan`, a CI harness, a parent that stopped
waiting — ended the run, and that is the same event as a Ctrl-C wearing a
different hat. Both map to 130, and 124 stays free for the day `scout` grows a
clock that could honestly claim it. Neither table is wrong; a table that
disagrees with the switch underneath it would be.

In Go this is a sentinel plus one classifier. `ErrUsage` is wrapped into every
parse and validation failure, `context.Canceled` and `context.DeadlineExceeded`
arrive on their own from the cancellation machinery, and `exitCode` is a short
switch over `errors.Is`.
Note what is *not* there: no error carries an exit code around with it, and no
handler knows exit codes exist.

One honest gap: cobra reports an unknown subcommand as a plain error with no
hook to classify it, so `scout frobnicate` exits 1 where you might prefer 2.
Matching on the message text would fix it and teach you a habit worth not
forming. Know it as a trade-off you made, not a detail you missed.

## Wrapping two facts into one error

Since Go 1.20, `fmt.Errorf` accepts more than one `%w`:

```go
return Settings{}, fmt.Errorf("%w: %w", ErrUsage, err)
```

The result matches `errors.Is(err, ErrUsage)` *and* `errors.Is(err,
fs.ErrNotExist)`. That is exactly right for "the config file you named is not
there": it is a usage problem, and it is a missing file, and different callers
care about different halves. Use it when both facts are genuinely load-bearing —
not as a reflex, since an error that matches five sentinels tells you nothing.

## Cancellation, end to end

`signal.NotifyContext` (which you met in S3 and again in the files and processes
lesson) sits in `main`, and the context it returns travels to the handlers
through `ExecuteContext`:

```go
err := root.ExecuteContext(ctx)          // in Execute
res, err := Scan(cmd.Context(), dir, …)  // in a handler
```

`cmd.Context()` is how a cobra handler reaches the context `ExecuteContext` was
given (it returns `context.Background()` if nobody set one, so tests that do not
care can ignore it). From there, cancellation is ordinary Go:

- `Scan` checks `ctx.Err()` once per directory entry and returns it. One check
  per file bounds how long a Ctrl-C takes to be noticed, and costs nothing.
- `App.Authors` uses `exec.CommandContext`, so a cancelled context kills `git` —
  and refuses to start it at all if the context is already dead.
- `Execute` sees `context.Canceled` come back, prints `scout: interrupted`, and
  returns 130.

Two details worth stealing. First, **the reason outranks the symptom**: a
cancelled child dies of a signal, so `cmd.Run` hands you an `*exec.ExitError`
saying "killed" — true and useless. Check `ctx.Err()` first and report *that*,
or your users will file bugs about git crashing when they press Ctrl-C. Second,
`stop()` from `NotifyContext` restores the default handler, so a second Ctrl-C
kills the process outright. Graceful shutdown must never trap the user.

## Testing a program that runs programs

`App.Authors` shells out, and the tests still run offline, on any machine, with
no fixtures — using a helper process, as in the previous lesson:

```go
func helperGit(t *testing.T, mode string) []string {
	exe, _ := os.Executable()
	t.Setenv("SCOUT_TEST_HELPER", mode) // set on the parent; scoped to this test
	return []string{exe, "-test.run=TestHelperProcess", "--"}
}
```

The "external command" is *this test binary*, re-executed. `TestHelperProcess`
returns immediately during an ordinary run and only plays git when the sentinel
is set; it writes canned commit authors to stdout, noise to stderr, exits 128
when the test wants a failure, and ends with `os.Exit` so the testing package's
own "PASS" never lands in the parent's captured stdout.

That is the stdlib's older shape — `-test.run=TestHelperProcess`, sentinel
exported by the *parent* — not the `TestMain` + `Command.Env` form the files and
processes lesson taught, and it is forced rather than chosen: `App.Git` is a
command *line*, so argv is all a test can inject; the environment is not part of
what `App` carries. So the warning from that lesson is live here, and what keeps
it safe is `t.Setenv` — it restores the old value when the test ends, and it
refuses to coexist with `t.Parallel` (add one and you get a panic, not a race).
Never export `SCOUT_TEST_HELPER` in the shell you run `go test` from; the whole
suite would become a helper. Give `App` an environment one day, and the sentinel
moves onto `Command.Env` where the previous lesson put it.

Read the modes in `authors_test.go` before you write `Authors` — they *are* the
requirements. And notice what each one pins: that stderr is not parsed as data,
that `cmd.Dir` is honoured, that a non-zero exit becomes an `ErrVCS` error
carrying the code and the child's own words, and that a cancelled run is
reported as cancellation.

The signal test (`signal_unix_test.go`, Unix-only) is the same discipline
applied to signals: send `SIGINT` to your own process, then *block on
`ctx.Done()`* — with a watchdog `time.After` so a broken implementation fails in
seconds instead of hanging. No `time.Sleep` anywhere. A sleeping test is flaky
on a busy machine and slow on an idle one.

## What the tests do not decide

The tests pin observable behaviour: bytes on stdout, exit codes, error
matching. They deliberately leave you the interesting decisions, and each one is
a design-review question:

- **Which flags are persistent?** `--json`, `--color`, `--config` and `--ignore`
  apply to the whole tree; `--top` and `--limit` belong to one command each; and
  `--version` is local to root, so `scout scan --version` is the typo it looks
  like rather than a silent success.
- **Where does `--top` apply?** The tests want the *table* truncated and the
  totals whole-tree, so truncation happens in the handler after the scan — not
  inside `Scan`, which would make the totals lie.
- **What counts as a line?** "Newline-terminated, plus a trailing unterminated
  one" is a choice; `wc -l` makes the other one. Either is defensible;
  disagreeing with yourself between the two is not. (And `filepath.Ext` thinks
  `.gitignore` is an extension, which is how every naive counter gets a bucket
  per dotfile.)
- **How much does the JSON promise?** The human table can be re-columned in a
  patch release. `{"root":…,"files":…,"exts":[…]}` cannot: it is the format
  other people's scripts parse, so it changes the way an API changes.

## Cutting the release

`version.go` is already wired for the distribution lesson: three package-level
`string` vars the linker can patch, and a `BuildInfo` struct so text and JSON
report the same thing.

```sh
go build -ldflags "-X main.version=1.0.0 \
  -X main.commit=$(git rev-parse --short HEAD) \
  -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o scout .
./scout --version
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o dist/scout-linux-arm64 .
```

An unpatched build says `dev`, which is the correct answer for a binary someone
built from a working tree — never fake a version number. What `version.go`
leaves out is the `runtime/debug.ReadBuildInfo` fallback the distribution lesson
made you build, and that is an omission rather than a second opinion: `scout` is
not published at a module path, so nobody can `go install` it and land in the
gap that fallback fills. Publish it and the gap opens the same day — put the
fallback back then, guarded on `version == "dev"`, exactly as you wrote it
there. When you finish the exercise, tag this tool and run it through the
release you set up in the distribution lesson: a `--version` that reports the
tag is the difference between a program and a piece of software.

## Exercise

Open [`exercise/`](exercise/) — a Go module with two complete files and five
work sites. Read the tests first; they are long because the tool is real.

Provided: `main.go` (the process edge and signal wiring), `version.go`, the
`Execute` function in `root.go`, `countFile`, and `NewRenderer`.

Your work:

- `app.go` — `LoadConfigFile`, `configPath`, `Resolve`, `ResolveColor`,
  `exitCode`.
- `scan.go` — `Scan`, `extOf`, `TopExts`.
- `authors.go` — `App.Authors`.
- `render.go` — `Paint`, `RenderScan`, `RenderAuthors`, `RenderVersion`.
- `root.go` — `NewRootCmd` and the three subcommands.

Acceptance criteria:

1. **The tree.** `NewRootCmd` returns a fresh `scout` tree with
   `SilenceErrors`/`SilenceUsage`, persistent `--json`, `--color`, `--config`
   and `--ignore` flags, a root-local `--version`, a local `--top` on `scan`, a
   local `--limit` on `authors`, and a `version` command. `scout` with no
   arguments prints help and succeeds. Every parse error is wrapped in
   `ErrUsage` via `SetFlagErrorFunc`.
2. **Precedence.** `Resolve` collapses defaults < config file < environment <
   flags into `Settings`, reading `SCOUT_TOP`, `SCOUT_LIMIT`, `SCOUT_JSON`,
   `SCOUT_IGNORE` (comma-separated) and `SCOUT_COLOR` through `app.env`. A flag
   that was not typed never overwrites a lower layer; a flag typed at its
   default value does.
3. **The config file.** `--config`, then `SCOUT_CONFIG`, then
   `<ConfigDir>/scout/config.json`. A missing file at the default location is
   fine; one the user named is an `ErrUsage` error that also matches
   `fs.ErrNotExist`. Unknown keys and malformed JSON are errors that name what
   is wrong. Absent keys decode to `nil` and change nothing.
4. **Bad values.** Every rejection is an `ErrUsage` error naming its source:
   `invalid SCOUT_TOP "abc"`, `invalid --top -1`, `invalid color "purple"`,
   `invalid SCOUT_JSON "maybe"`.
5. **Colour.** `ResolveColor` applies, highest priority first: `never`, then
   `always`, then a non-empty `NO_COLOR`, then `TERM=dumb`, then the terminal
   answer — an empty `NO_COLOR` counts as unset. In `--json` mode colour is off
   whatever the policy says.
6. **Scan.** `Scan` totals regular files by lowercased extension, buckets
   dotfiles and extensionless files as `(none)`, prunes ignored directories and
   skips ignored file names, sorts busiest-first with name as the tie-break,
   propagates walk errors, and returns `ctx.Err()` when the context is
   cancelled. A line is newline-terminated, plus a final unterminated one.
7. **Authors.** `App.Authors` runs `App.Git` + `log --format=%an` in `dir` with
   stdout and stderr captured separately, ranks authors busiest-first with name
   as the tie-break, truncates to `limit` (0 means all), and never returns
   `nil`. A non-zero exit is an `ErrVCS` error carrying the exit code and the
   child's stderr; a cancelled run returns an error matching
   `context.Canceled`, not `ErrVCS`.
8. **Output.** `scan` prints the padded table (bold header and total when
   colour is on, the reset before the newline) or the `ScanResult` document;
   `authors` prints `%6d  %s` per author or `{"authors":[…]}`, `[]` when empty;
   `version` prints `scout <version> (commit …, built …, <go>, <platform>)` or
   the `BuildInfo` document. Empty results read `no files` / `no commits`. JSON
   output contains no escape sequences, ever.
9. **Exit codes.** `exitCode` returns 0 for success, 2 for `ErrUsage`, 130 for
   `context.Canceled` *and* `context.DeadlineExceeded`, and 1 for everything
   else — the table above, deadline row included; on any failure
   stdout stays empty and exactly one line goes to stderr, prefixed `scout: `.
   `--version` and `version` print the same thing.
10. `go test -race ./...` passes and the code is `gofmt`-clean.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test -race ./...
```

A sane order: `exitCode` and `ResolveColor` (minutes), then `Scan` and the
renderer, then the tree — the CLI tests start passing in clumps once the tree
exists — then `Resolve`, then `Authors`. When it is green, use it on a real
repository, pipe it into `jq`, press Ctrl-C during a scan of something huge,
and set `NO_COLOR=1`. The tests prove it is correct; only your own eyes prove
it is pleasant.

## Further reading

- [Command Line Interface Guidelines](https://clig.dev/) — read it end to end
  now that you have built one; the arguments will land differently.
- [pkg.go.dev — os/exec](https://pkg.go.dev/os/exec) — `CommandContext`,
  `ExitError`, and the `Cancel`/`WaitDelay` fields for when kill is not enough.
- [pkg.go.dev — os/signal](https://pkg.go.dev/os/signal) — `NotifyContext`, and
  the notes on what is safe to do in a handler.
- [Go Blog — Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)
  — the wrapping model `ErrUsage` and `ErrVCS` rely on.
