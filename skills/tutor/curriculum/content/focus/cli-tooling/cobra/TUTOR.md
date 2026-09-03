# Tutor notes — Cobra & Subcommands

## Where the learner is

Second lesson of the CLI focus pack, inserted after S3. They know interfaces,
errors and wrapping, JSON, table-driven tests, and the whole concurrency arc —
but this pack is deliberately single-threaded, so nothing here needs goroutines.
They have shipped two CLIs: the S1 habit tracker and the S3 concurrent link
checker with its `Run(args, stdin, stdout, stderr) int` entry point and its own
`flag.FlagSet`. The previous lesson gave them a precedence chain
(defaults < config file < environment < flags) built on stdlib `flag`.

This is their first third-party dependency taken deliberately. Two things are
genuinely new: a framework that owns the dispatch loop, and a library API large
enough that reading docs beats guessing. They have **not** done S4 — no TDD
lesson, no security, no HTTP servers — so frame testing as "the tests are the
spec", the way S1 did, not as a methodology.

## Common misconceptions

- **"cobra is what Go CLIs use, so use it."** Push back. The registry objective
  is the *choice*, not the framework. A learner who cannot say when `flag` is the
  better answer has not met the objective.
- **Global `rootCmd` + `init()`.** Every tutorial and cobra's own generator does
  this, so expect it. The failure is not stylistic: a `cobra.Command` stores
  parsed flag values, so a shared tree leaks `--format` from one test into the
  next. Have them try it and watch a test go green only when run alone.
- **"Persistent means the value persists."** It means *visible in the subtree*.
  Nothing is stored between runs.
- **Confusing `Flags()`, `LocalFlags()`, `InheritedFlags()`.** Also: `Flags()`
  may not show inherited flags before parsing has happened, which is why the
  scope test uses the other two.
- **`os.Exit`/`log.Fatal`/`cobra.CheckErr` inside a handler.** Kills the test
  binary, skips defers and buffer flushes. `RunE` returns; `main` exits.
- **Expecting `Args: cobra.NoArgs` alone to catch `notes tag bogus`.** It does
  not: cobra returns "needs help" for a non-runnable command *before* validating
  args, so the group command also needs a `RunE` that prints help. This is the
  designed-in surprise of the exercise — let them hit it.
- **`fmt.Println` inside a handler.** Compiles, passes nothing, because the test
  captured `cmd.OutOrStdout()`.
- **Overwriting the environment with a flag default.** Forgetting `Changed` makes
  `NOTES_FORMAT=json` stop working the moment a `--format` default exists.
- **"viper is the standard way to do config."** Ask what it costs before they
  reach for it.

## Grilling points

- "You have `--format` on root and `--tag` on `add`. Which commands can read
  each, and which cobra call would prove it?"
- "Your `RunE` returns an error. Trace it: who receives it, who prints it, what
  is the process exit code, and which two fields decide whether cobra printed
  anything first?"
- "`notes tag bogus` — before you run it, what exit code do you predict, and why
  is the naive answer wrong?"
- "Delete the `Changed` check and predict which test breaks. Then run it."
- "Last lesson insisted on `os.LookupEnv`; `App.env` uses `Getenv` and tests
  `!= ""`. Justify the shortcut here, then name a setting that would break it."
  (Anything with a meaningful empty value — an empty prefix, an empty separator,
  or a variable whose presence alone is the signal.)
- "The test never starts a subprocess. What did the design have to give up
  globals for, to make that possible?"
- "Sell me stdlib `flag` for this exact tool. Now sell me cobra. Where is the
  line?"
- "If you added a config file to this tool tomorrow, would you bring in viper?
  Argue both sides, then commit."
- Stretch: "How would you add a `--verbose` flag that every subcommand honors,
  without touching any subcommand's code?"

## Grading rubric

- **A** — All tests pass. Tree built by constructors taking `*App`; `Args` set on
  every command; grouping commands runnable *and* validated; `SilenceErrors` and
  `SilenceUsage` set with `main` the sole error reporter; output only via
  `cmd.OutOrStdout()`; the store's error returned unwrapped or wrapped with `%w`;
  `Resolve` reads cleanly top to bottom with `Changed` guarding the flag layer.
  They can justify cobra-vs-`flag` and viper-vs-hand-rolled without prompting.
- **B** — Tests pass; design mostly right but with a leak: a package-level
  command or `App`, duplicated rendering in each handler, `%v` instead of `%w`,
  or the flag-scope rules only recited, not explained.
- **C** — Tests pass after heavy hinting, or `Resolve` is a special-cased
  cascade the learner cannot restate as a precedence rule. Pass only if a
  time-boxed remediation lands.
- **Fail** — Tests failing; or `os.Exit`/`fmt.Println` inside handlers; or the
  learner cannot say which command sees which flag. Remediate, do not advance.

## Remediation ladder

1. "Run just the failing test with `-run`. Read the failure message aloud —
   which command, which stream, which exact bytes?"
2. "Which of the four given files does that test touch: the tree in `root.go`, or
   the resolver in `app.go`? Start there and change one thing."
3. For tree problems: "Print `root.Commands()` in a scratch test. Is the command
   even attached? Now check its `Args` and whether it has a `RunE`." For resolver
   problems: "Add a `fmt.Printf` of `f.Changed` and `f.Value.String()` at the top
   of `Resolve` and run the precedence table."
4. Walk one command end to end verbally — `Use`, `Short`, `Args`, flag
   declaration, `RunE` body, where output goes — and have them type it, then
   apply the same shape to the rest themselves. Never paste the whole tree.

## After passing

Preview: "Next: terminal UX — what to print when stdout is a pipe instead of a
terminal, colour that turns itself off, and machine-readable output alongside
human output. You already write to an injected writer; now you make that writer's
context change what you write."
