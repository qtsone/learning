# Terminal UX

> `focus.cli.terminal-ux` · ~3-4h · Stage: Focus: CLI Tooling

## Objectives

By the end of this lesson you can:

- Detect whether output is a TTY and degrade gracefully — no colors or progress
  animations when piped or redirected.
- Render styled ANSI output while honoring the NO_COLOR convention and a
  `--no-color` flag.
- Implement an interactive prompt with input validation and a sensible default
  value.
- Implement a progress indicator for a long-running operation that writes to
  stderr, leaving stdout clean for data.
- Explain what a TUI framework's model-update-view loop (Bubble Tea and
  friends) adds over line-based output, and identify when it is overkill.

## Two streams, two audiences

Your program has two output streams and they are not interchangeable:

- **stdout carries data** — the thing the user asked for. It is what flows into
  the next command in a pipeline.
- **stderr carries diagnostics** — progress, warnings, errors, prompts.
  Everything *about* the run rather than *of* the run.

That split is the whole reason `tool | jq .` and `tool > out.txt 2> log.txt`
work. Put a progress bar on stdout and you have corrupted your user's data.
Put the results on stderr and the pipeline downstream gets nothing.

The rule of thumb: *if a script would want to parse it, it goes on stdout;
if a human would want to read it while the program runs, it goes on stderr.*
The third channel is the exit code — 0 for success, non-zero for failure —
which is how `&&` and CI decide what happens next.

You already have the shape for this. Your S3 link checker was entered through
`Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int`, and in the
Cobra lesson you fed commands with `SetOut`/`SetErr`. This lesson is about what
you *write* into those two writers.

## Never reach for os.Stdout

Once a package writes to `os.Stdout` directly, the only way to test its output
is to hijack a global, and the only way to reuse it elsewhere is not at all.
Take writers as parameters and let exactly one function — `main` — know that
the real ones come from `os`:

```go
func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
```

`io.Discard` then becomes your off switch: passing it instead of `os.Stderr`
silences a component without a single `if quiet` inside it.

## Is anyone watching?

A terminal renders escape sequences; a file, a pipe, and a CI log do not. So
before styling anything you have to ask whether the stream on the other end is
a terminal. The stdlib answer is a file-mode check:

```go
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
```

Two honest caveats: this asks "is it a character device?", which is true for
terminals *and* for `/dev/null`; and on Windows the console needs extra work
before it interprets escapes at all. The precise answer is an ioctl, and
`golang.org/x/term.IsTerminal(int(f.Fd()))` is the tiny, well-maintained
dependency that does it properly on every platform.

Whichever you use, the important design move is the same: **detection happens
once, at the edge, and everything below it receives a plain `bool`.** That is
what makes the rest of the program testable — a test can say "pretend this is a
terminal" without owning a terminal. Note also that stdin, stdout, and stderr
are redirected independently, so you may need three answers: colour the results
by whether *stdout* is a terminal, draw progress by whether *stderr* is, and
prompt by whether *stdin* is.

## ANSI in one page

Styling is done with escape sequences the terminal interprets instead of
printing. They start with the escape character `\x1b` (ESC), then `[` — the
pair is called CSI, Control Sequence Introducer — then parameters, then a
letter that says what to do. For colour and style the letter is `m` (SGR,
Select Graphic Rendition):

```go
const (
	Reset = "\x1b[0m"   // turn every attribute off
	Bold  = "\x1b[1m"
	Dim   = "\x1b[2m"
	Red   = "\x1b[31m"  // 30-37 foreground, 40-47 background,
	Green = "\x1b[32m"  // 90-97 the bright variants
)

fmt.Println(Red + Bold + "error" + Reset + ": file not found")
```

There are also 256-colour (`\x1b[38;5;<n>m`) and 24-bit
(`\x1b[38;2;<r>;<g>;<b>m`) forms, but the basic palette is what users have
themed and what every terminal renders — reach past it rarely.

Three rules save you most of the pain:

1. **Always reset.** An unclosed attribute leaks into the user's shell prompt.
2. **Escape sequences have no width.** `\x1b[32mok\x1b[0m` is 11 bytes and two
   columns. `fmt.Sprintf("%-4s", painted)` pads by the length of the string it
   is handed — counting the nine invisible escape characters as if they
   occupied columns — and destroys your alignment: pad by the visible text,
   then paint.
3. **Cursor control is not colour.** `\r` (carriage return) and `\x1b[K`
   (erase to end of line) move and clear rather than decorate. `NO_COLOR` has
   no say over them, but a non-terminal stream must not see them either.

## The colour decision is policy

Whether to emit those bytes is a decision with several inputs, and — exactly as
in the flags-and-configuration lesson — the value of a precedence chain is that
it is *written down* and testable:

1. `--color=never` / `--no-color` → off. The user said no.
2. `--color=always` → on. The user said yes, even into a pipe (this is how
   `tool --color=always | less -R` works).
3. `NO_COLOR` set to a non-empty value → off. This is a cross-tool convention
   (see no-color.org): users set it once and every well-behaved program obeys.
   An explicit flag on *this* invocation is more specific than an environment
   default, which is why it sits above.
4. `TERM=dumb` → off. The terminal has told you it cannot render escapes.
5. Otherwise → on only if the stream is a terminal.

Two practical notes. Take the environment as a `func(string) (string, bool)`
parameter rather than calling `os.LookupEnv` deep in the code: the policy then
tests as a pure function. And keep colour *out* of machine-readable output
unconditionally — a `--json` run must never emit an escape byte, whatever the
flags say.

## Quiet and verbose

Verbosity is a level, not a pile of booleans:

```go
const (
	LevelQuiet Level = iota
	LevelNormal
	LevelVerbose
)
```

The gate applies to the *diagnostic* stream only. **Data** is the program's
product, so quiet mode never suppresses it, and **errors** are never suppressed
either — a silent failure is a bug report you receive next quarter. What the
level gates is **info** (what is happening, normal and above) and **debug**
(how it is happening, only when asked). Two flags, `-q` and `-v`, cover almost
every real tool; resist inventing five levels nobody can remember.

## Progress that behaves

A progress line is a single line rewritten in place: return the cursor to
column 0 with `\r`, print the new status, and erase whatever the previous,
longer status left behind with `\x1b[K`:

```go
fmt.Fprintf(w, "\r%s: %d/%d (%d%%)\x1b[K", label, n, total, n*100/total)
```

It goes to **stderr**, so that `tool > results.txt` still shows progress and
`tool | wc -l` still counts the right thing.

When the stream is not a terminal, the animation is not just useless but
actively harmful: a CI log ends up with ten thousand identical lines full of
control characters. Degrade to nothing during the run and one plain summary
line at the end. That is the general shape of graceful degradation — *the
information survives, the decoration does not*.

Keep the reporter driven by explicit `Update` calls from the work loop rather
than by a background timer. A timer means a goroutine, wall-clock flakiness,
and a test that cannot assert bytes; explicit calls mean the same input always
produces the same output. (Real tools throttle redraws — "at most every 100ms"
— but that belongs outside the piece you are testing.) Two details worth
knowing: a status line longer than the terminal wraps and breaks the redraw, so
truncate it; and a program killed mid-draw can leave a hidden cursor behind,
which is signal handling — the next lesson in this pack.

## Prompts

An interactive prompt is the same discipline again: write the question to
stderr, read the answer from an injected reader, and treat "there is no
terminal" as a first-class case.

- Show the default in brackets — `Region [eu-west-1]: ` — and let an empty
  answer take it. If the user has to type the default, it is not a default.
- Validate, and on rejection say what was wrong and ask again. Cap the retries
  so a broken loop terminates.
- **Never block on a prompt when stdin is not a terminal.** Use the default, or
  fail with an error naming the flag that supplies the value. A tool that hangs
  in CI waiting for an answer nobody can give is the classic version of this
  bug; `--yes` exists for exactly this reason.
- Create the `bufio.Scanner` (or `bufio.Reader`) **once**, in the constructor.
  A fresh one per question keeps whatever it buffered past the newline, and
  your second question silently eats the answer to the third.
- For secrets, don't echo: `golang.org/x/term.ReadPassword` turns echo off and
  restores it. Rolling that yourself with raw terminal modes is not worth it.

## Machine-readable output

Every tool worth scripting has a `--json` mode, and the reason is a contract:

> The human format is not an API. The machine format is.

You may realign columns, add colour, or reword a summary in a patch release.
The JSON shape is something other people's scripts depend on, so it changes
only the way an API changes.

Practical rules, all of them exercised by this lesson's tests:

- JSON goes on **stdout, alone**. Diagnostics stay on stderr, so
  `tool --json 2>/dev/null | jq` never sees a stray log line.
- Emit a **document**, not a bare array: `{"results": [...]}` can grow a
  `"version"` or `"summary"` field later; `[...]` cannot.
- Beware `null`. A nil slice marshals to `null`, and every consumer that loops
  over the array breaks on it. Encode an empty slice.
- Never style it. See the previous section.
- For a stream of events, one JSON object per line (NDJSON) lets consumers
  process results as they arrive instead of waiting for the closing brace.

## When a TUI is the wrong answer

Bubble Tea and its ecosystem (Lip Gloss for styling, Bubbles for widgets)
implement the Elm architecture: your program is a `Model`, an `Update(msg)`
that returns a new model plus commands, and a `View()` that renders the whole
screen as a string. The framework owns the event loop, the alternate screen
buffer, raw input, and repainting — which is genuinely hard to hand-roll, and
the right call for a long-lived interactive session: a file browser, a log
explorer, a git client.

It is the wrong answer for most CLI tools, and the reasons are the same ones
this lesson has been about. A full-screen app is not pipeable, not greppable,
not scriptable, and awkward in CI, over a poor ssh link, in a dumb terminal, or
with a screen reader. It is also markedly harder to test than a function that
writes bytes to an `io.Writer`. If the interaction is "run, do work, report",
line-based output with good degradation serves users better and costs less.
Reach for a TUI when the user's job is to *stay* in your program, not to pass
through it.

## Exercise

Open [`exercise/`](exercise/) — a module with four work files and their tests.
`main.go` is complete and shows the wiring: it is the only file that touches
`os`, and it hands everything below three writers and three booleans.

Read the tests first; they pin the exact bytes.

- `color.go` — `ParseColorMode` and `ResolveColor`, the precedence chain.
- `render.go` — `Renderer`: `Paint`, `Out`, `Info`, `Debug`, `Errorf`,
  `Results`.
- `progress.go` — `Progress`: `Update`, `Done`.
- `prompt.go` — `Prompter.Ask`.

Acceptance criteria:

1. `ParseColorMode` accepts `"auto"`, `"always"` and `"never"`, and returns an
   error naming the bad value otherwise.
2. `ResolveColor` implements the documented chain: `never` off, `always` on
   (beating `NO_COLOR`), non-empty `NO_COLOR` off, `TERM=dumb` off, otherwise
   the terminal answer. `NO_COLOR` set to the empty string is not a veto.
3. `Paint` wraps text in the given codes plus one `Reset` when colour is on,
   returns it unchanged when colour is off, and never emits a lone `Reset`.
   The provided `main.go` folds its `--no-color` shorthand into the same
   chain by mapping it to `never` — read that wiring and be ready to say why
   the shorthand exists alongside the three-valued `--color`.
4. `Out` writes data to stdout at every level; `Info` writes to stderr at
   normal and above; `Debug` only at verbose; `Errorf` writes to stderr at
   every level with a `Red`+`Bold` `error:` prefix.
5. `Results` in human mode writes one line per result, `ok` green and `fail`
   red, with the status padded to four visible columns so coloured and plain
   output align identically.
6. `Results` in JSON mode writes one compact `Report` line on stdout with no
   escape sequences even when colour is on, and encodes an empty result set as
   `{"results":[]}`. Both modes report write errors.
7. `Progress` on a terminal redraws `\r<status>\x1b[K` per `Update` and leaves
   a `done` summary line; off a terminal it writes nothing until `Done`, then
   one plain summary with no `\r` and no escapes. An unknown total (0) reports
   a count instead of a percentage.
8. `Ask` prints `question [default]: ` (or `question: `), trims the answer,
   takes the default on an empty one, re-asks with an `error: …` line on
   validation failure up to `MaxPromptAttempts` before returning
   `ErrTooManyAttempts`, returns the default without touching the streams when
   there is no terminal, and returns `ErrNotInteractive` when there is neither
   terminal nor default. Consecutive questions read consecutive lines.
9. `go test ./...` passes and the code is `gofmt`-clean.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...
```

They fail on the starter. When they are green, build the demo and look at it
with your own eyes — the difference between the two runs is the lesson:

```sh
go build -o scan . && ./scan          # a terminal: colour and a live line
./scan | cat                          # a pipe: plain, quiet, parseable
./scan --json | cat
NO_COLOR=1 ./scan
```

## Further reading

- [no-color.org](https://no-color.org/) — the convention, in about ten lines.
- [Command Line Interface Guidelines](https://clig.dev/) — the best single
  write-up of CLI conventions; the output and interactivity sections cover
  exactly this lesson's ground.
- [pkg.go.dev — golang.org/x/term](https://pkg.go.dev/golang.org/x/term) —
  `IsTerminal`, `GetSize`, `ReadPassword`.
- [XTerm control sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)
  — the reference for what those escape codes actually mean.
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — read the tutorial
  to see the model-update-view loop, then decide honestly whether your tool
  needs one.
