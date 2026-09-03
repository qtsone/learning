# Files, Processes & Signals

> `focus.cli.files-processes` · ~3-4h · Stage: Focus: CLI Tooling

## Objectives

By the end of this lesson you can:

- Run external commands with `os/exec`, capturing stdout, stderr and the exit
  code, without invoking a shell.
- Explain why passing user input as `exec.Command` arguments avoids shell
  injection while building a command string does not.
- Handle SIGINT/SIGTERM via `signal.NotifyContext` to cancel in-flight work and
  clean up resources before exiting with the right code.
- Implement an atomic file write (temp file + rename) and explain what can be
  lost with a direct overwrite.
- Use `filepath`, `os.UserHomeDir` and related APIs so paths and file handling
  work on both Unix and Windows.

## Running a program, not a command line

Everything the pack has covered so far happened inside your process. Real tools
spend most of their time asking the operating system for things — run `git`, read
the user's config, replace a file, stop politely on Ctrl-C — and each of those
has a version that works on your laptop and a version that also works on someone
else's, in CI, and when the machine loses power mid-write. Start with children.

```go
cmd := exec.Command("git", "log", "--oneline", "-n", "10")
```

There is no shell here. `exec.Command` takes an executable and a list of
arguments and hands that array to the kernel; `git` receives four arguments and
nothing gets interpreted on the way. If `Name` has no separator in it, os/exec
looks it up on `PATH` for you (`exec.LookPath`), and a failure surfaces when you
start the command, wrapping `exec.ErrNotFound`.

Now the version people write instead, and the reason this objective exists:

```go
// Never do this.
out, err := exec.Command("sh", "-c", "grep "+pattern+" notes.txt").Output()
```

Ask what happens when `pattern` comes from a flag or a config file and its value
is `x; rm -rf ~`. The shell sees a `;` — "end of command, here comes another
one" — and obeys. Every shell feature is live on that string: `;` and `&&` chain,
`|` pipes, `$(…)` substitutes, `>` redirects, `*` globs, `$HOME` expands,
whitespace splits words.

Quoting is not the fix: the rules differ between shells, differ again on
Windows, and one missed edge case is a remote code execution. **The array is the
fix.** `exec.Command("grep", pattern, "notes.txt")` passes `x; rm -rf ~` as one
argument and `grep` searches for that ridiculous string. You will meet this exact
shape again in S4, wherever data has to cross another parser's boundary: the fix
is always separation at the transport layer, never sanitising the data.

Running a script a user *gave* you (a `hooks:` section in their config) through
`exec.Command("sh", "-c", script)` is legitimate — but that is a decision to
execute arbitrary code, and it needs a machine with `sh`. Take it on purpose;
never drift into it.

## stdout, stderr and the exit code

`os/exec` offers three ways to run. `cmd.Output()` returns stdout and tucks
stderr into `ExitError.Stderr`; `cmd.CombinedOutput()` interleaves both into one
buffer, which is convenient and destroys exactly the distinction the previous
lesson was about — once the child's progress bar is mixed into its data you can
never separate them again. `cmd.Run()` returns only an error and leaves the
streams to you, which is what you want:

```go
var stdout, stderr bytes.Buffer
cmd.Stdout, cmd.Stderr = &stdout, &stderr
err := cmd.Run()
```

When `cmd.Stdout` is not an `*os.File`, os/exec makes a pipe and copies from it
in a goroutine. `Wait` (which `Run` calls) waits for those copies before
returning, so reading the buffers afterwards is race-free and reading them
*before* is a data race the detector will catch.

Then read the error, which has exactly three shapes:

- `nil` — the child ran and exited 0.
- `*exec.ExitError` — it ran and exited non-zero. `errors.As` it and ask
  `ExitCode()`. If a signal killed the child, `ExitCode()` is -1.
- anything else (`*exec.Error` wrapping `exec.ErrNotFound`, an `*fs.PathError`
  for a missing `Dir`) — it never ran.

The API decision that follows: **a non-zero exit is data, not a Go error.**
`grep` exits 1 when it finds nothing, `diff` exits 1 when files differ; neither
is a malfunction. Put the status in your result type and reserve the error for
"I could not run this at all".

## The environment you hand down

`cmd.Env` is nil by default, and nil means "inherit ours". Set it and you have
*replaced* the whole environment — a child with no `PATH` and no `HOME` is a
strange place, and the bug reports are baffling. To add a variable, start from
what you have:

```go
cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
```

Duplicate keys resolve last-wins when the command starts (case-insensitively on
Windows), so an appended entry overrides an inherited one.

Two related habits. Secrets belong in the environment, never in arguments:
command lines are visible to every user on the box through `ps` and end up in
shell history — the flags lesson's argument for keeping tokens out of flags. And
to run a child elsewhere, set `cmd.Dir`; `os.Chdir` moves your *whole process*,
which is a race the moment anything else runs concurrently.

## Cancellation is a context, as usual

```go
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
cmd := exec.CommandContext(ctx, "git", "fetch")
```

When the context is done, os/exec kills the child. Everything you learned about
contexts in S3 applies unchanged, but three details are specific to processes:

1. **The error does not tell you why.** A child killed by the context returns an
   `*exec.ExitError` reading `signal: killed`, indistinguishable from any other
   kill — so after `Run` returns you check `ctx.Err()` *first*, and only then the
   exit status. Reverse that order and a timeout is reported as "the command
   failed", sending the user after the wrong bug.
2. **Kill is not polite.** SIGKILL gives the child no chance to remove its lock
   file. If it deserves one, set `cmd.Cancel` to send SIGTERM yourself and let
   `cmd.WaitDelay` be the grace period before the kernel steps in.
3. **Kill reaches the child, not its children.** Grandchildren that inherited the
   output pipes can keep `Wait` blocked after the kill; `cmd.WaitDelay` (Go 1.20)
   bounds that instead of hanging your tool forever. Killing a whole tree means
   putting it in its own process group, which is platform-specific
   (`SysProcAttr.Setpgid` on Unix).

## Testing without external binaries

Tests that shell out are the flakiest code in most repositories, because they
smuggle in assumptions: that `/bin/sleep` exists (not on Windows), that `echo` is
a program (a shell builtin in places), that `python3` is on `PATH` in CI.

The standard fix is the **helper-process idiom**: the child is *this test
binary*, re-executed with a sentinel environment variable telling it to behave
like a small program instead of a test suite.

```go
const helperEnv = "TUTOR_HELPER_MODE"

func TestMain(m *testing.M) {
	if mode := os.Getenv(helperEnv); mode != "" {
		os.Exit(helperProcess(mode, os.Args[1:])) // be the child, then exit
	}
	os.Exit(m.Run())                              // be the test suite
}
```

A test then builds its child from `os.Executable()` — the binary the toolchain
just built — with `Env: []string{helperEnv + "=" + mode}`. `TestMain` runs before
any test, so the child branches away before `m.Run` and never executes one. The
result is a child that exists on every platform, is the same build with the same
race instrumentation, needs no `PATH`, no network and no fixture files, and
behaves identically every run.

Three details worth internalising:

- The sentinel is set **only on the child** (through `Command.Env`). Set it in
  the environment running `go test` and your suite silently becomes a helper.
  The child still inherits everything else, which is how you test inheritance.
- Keep helper modes tiny and deterministic. They are fixtures, not programs.
- To make a helper block until it is killed, sleep — never `select {}`. The
  runtime notices every goroutine is asleep, declares a deadlock and kills the
  process, and your cancellation test then passes for the wrong reason.

The older form, still all over the standard library, re-runs the binary with
`-test.run=TestHelperProcess` plus the sentinel: same idea, more plumbing.

## Signals

A signal is the operating system interrupting your process with a number. Three
matter for a CLI:

| signal | sent by | default action | catchable |
|---|---|---|---|
| SIGINT | Ctrl-C, to the whole foreground process group | terminate | yes |
| SIGTERM | `kill`, systemd, `docker stop`, `kubectl delete` | terminate | yes |
| SIGKILL | `kill -9`, the kernel, a grace period running out | terminate | **no** |

SIGTERM is a request: "wind up, you have a moment". SIGKILL is not a request and
cannot be caught, which is why that moment must be short — Docker gives 10
seconds by default, Kubernetes 30, then SIGKILL. When a signal terminates a
program the shell reports 128 + the signal number: 130 for SIGINT, 143 for
SIGTERM. Following the convention is how a script that ran your tool can tell
"the user pressed Ctrl-C" from "it failed".

128+N describes a process that *was* signalled. A timeout is a different story —
your tool decided to stop waiting — and it has its own convention, borrowed from
`timeout(1)`, which exits **124** when it gave up. That is why the exercise's
`ExitTimeout` is 124 and not 128 plus something: no signal arrived at your
process, so there is no N to add.

Catching them is one line, and it turns a signal into the cancellation you
already know how to handle:

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
```

`stop` unregisters, restoring the default behaviour — so a second Ctrl-C after
your cleanup has started still kills the program, which is what a user leaning on
the key expects. (The older `signal.Notify(ch, …)` with a buffered channel does
the same job, and is what you need to react to a signal *without* cancelling
everything, like SIGHUP for "reload your config".)

Two rules about what happens next:

- **Do no slow work in response to a signal.** The only thing that should happen
  is a context becoming done. Flushing state, finishing an upload, removing temp
  files — all of that belongs in the `defer`s and `<-ctx.Done()` branches that
  were already there, where another timeout can bound it. A handler busy doing
  something slow is a handler that cannot process the next signal.
- **Cleanup only runs if you return.** `os.Exit` on the signal path skips every
  `defer`: the terminal keeps the cursor your progress bar hid, the lock file
  stays. Let cancellation propagate and the stack unwind, then map the outcome to
  an exit code in `main`.

Windows has no signals in the Unix sense: `os.Interrupt` covers Ctrl-C, there is
no `kill(2)`, and service shutdown is another mechanism entirely. That is why the
exercise's signal tests carry a `//go:build unix` line — a platform-specific test
belongs behind a build constraint rather than a runtime skip.

## A file write that survives a crash

`os.WriteFile` opens with `O_TRUNC` and then writes. Between those two steps the
file is empty, and if the process dies there — a crash, a kill, a full disk — it
stays that way. A reader arriving mid-write sees a truncated file. For a cache
that is annoying; for the user's config it is data loss you caused.

The fix is to never touch the target at all. Build the new file beside it and
move it into place in one indivisible step:

1. `os.CreateTemp` **in the same directory as the target**. `os.Rename` is atomic
   only within a filesystem, and `os.TempDir()` is on a different one often
   enough that the rename fails with "invalid cross-device link" on your user's
   machine and never on yours.
2. Write the data, then `Chmod` to the mode you want: `CreateTemp` always creates
   0600, and unlike the mode argument to `os.OpenFile`, `Chmod` is not filtered
   through the umask, so this lands exactly.
3. `Sync`, or the rename can reach the disk before the data does and a crash
   leaves a directory entry pointing at garbage.
4. `Close`, and **check its error** — on a network or full filesystem that is
   where the failed write finally surfaces.
5. `os.Rename` over the target. A reader opens either the old file or the new
   one; there is no third state.
6. `defer os.Remove(tmp)` from the moment the temp file exists, so no failure
   path leaves rubbish next to the user's config. After a successful rename that
   name is gone and removing it is a harmless no-op.

The honest limits: rename replaces the file's *identity*, so the old inode's
ownership, ACLs and hard links do not carry over, and a process holding the old
file open keeps reading the old bytes (usually what you want). Making the rename
itself durable needs an fsync of the *directory* — possible on Unix by opening it
and calling `Sync`, not portably. And on Windows the rename can fail outright if
another process has the target open.

## Paths that work on someone else's OS

`path` and `path/filepath` have nearly the same functions and are not
interchangeable. `path` is always forward slashes: URLs, `io/fs`, archive
entries — anything defined by a spec rather than a disk. `path/filepath` is the
local operating system, with `\` and volume names on Windows. `path.Join` on a
filename is a bug that compiles, passes your tests, and breaks for the first
Windows user.

Never hand-build `$HOME/.config/tool/config.json` either; the standard library
knows where things go on each platform:

| call | Linux | macOS | Windows |
|---|---|---|---|
| `os.UserConfigDir` | `$XDG_CONFIG_HOME` or `~/.config` | `~/Library/Application Support` | `%AppData%` |
| `os.UserCacheDir` | `$XDG_CACHE_HOME` or `~/.cache` | `~/Library/Caches` | `%LocalAppData%` |
| `os.UserHomeDir` | `$HOME` | `$HOME` | `%USERPROFILE%` |

The split is a promise: **config is what the user edits and would be upset to
lose; cache is what you may delete at any time.** Put a cache in the config
directory and their dotfiles repo fills with junk; put config in the cache
directory and a cleanup tool eats their settings. (There is no stdlib call for a
*state* directory — logs, history — so on Unix people follow XDG's
`~/.local/state` by hand.)

Three more traps, all of which the exercise pins down:

- **`filepath.Join` cleans, so it swallows `..`.** `Join(root, "../../etc")`
  cheerfully lands outside `root`. A path from outside your program — a config
  value, an archive entry, an argument — must be *checked*, and
  `filepath.IsLocal` (Go 1.20) is the check: it rejects empty, absolute and
  rooted paths, anything escaping with `..`, and Windows device names like `NUL`.
  It says nothing about symlinks, though — one inside the directory can still
  point out of it, which is what `filepath.EvalSymlinks` (or `os.Root`, in newer
  Go) is for.
- **Case.** APFS and NTFS are usually case-insensitive, ext4 is not: `README` and
  `readme` are one file on your Mac and two on the CI machine.
- **Permission bits** are advisory on Windows, and on Unix the mode you pass to
  `OpenFile` is masked by the process umask. Use `0o600` for anything holding a
  secret and `0o700` for the directory around it.

## Exercise

Open [`exercise/`](exercise/) — a module for `runner`, a tool that runs a child
command with a timeout, forwards its streams and exit code, and caches the last
run. `main.go` is complete: read it first, it is the wiring and the only file
touching `os.Args`, the real streams or the real user directories. Then read the
tests — they are the specification, and `exec_test.go` carries the helper-process
idiom you should understand before writing anything.

Acceptance criteria:

1. `Run` executes a `Command` with no shell involved: each `Args` element reaches
   the child verbatim, and stdout and stderr are captured separately. `Env`
   entries are *added* to the inherited environment and override same-key ones;
   `Dir` sets the child's working directory; `Stdin` feeds it.
2. A child that exits non-zero is not an error: `err` is nil and `Result.ExitCode`
   carries the status, taken from the `*exec.ExitError`.
3. `Run` errors only when the command could not run to completion, and `ExitCode`
   is then -1: an empty `Name` fails with `ErrEmptyCommand` before touching the
   OS, a missing executable with an error wrapping `exec.ErrNotFound`.
4. When `ctx` is done the error wraps `ctx.Err()`, so `errors.Is(err,
   context.DeadlineExceeded)` and `errors.Is(err, context.Canceled)` hold — even
   though the killed child reported an ordinary exit error. An already-cancelled
   context produces no output at all.
5. `RunSteps` runs commands in order, returns the results of the steps that ran
   (including a failing one), and stops at the first failure with a `*StepError`
   carrying `Index`, `Name`, `Result` and `Err`. `Unwrap` exposes the cause,
   `Error()` reads `step 1 (…): exit status 7`, and the context is checked before
   each step, so a cancelled one runs nothing and reports index 0.
6. `WithInterrupt` returns a context derived from its parent and cancelled by
   SIGINT or SIGTERM, plus the `stop` function that unregisters.
7. `ExitCodeFor` maps nil, a cancellation, a deadline and anything else onto
   `ExitOK`, `ExitInterrupted`, `ExitTimeout` and `ExitFailure`, seeing through
   wrapped errors and `*StepError`.
8. `AppDir` joins a validated application name onto whatever the injected lookup
   reports, creates nothing, and wraps a lookup failure. `SafeJoin` cleans `rel`
   and returns the joined path when it stays under `root`. Both reject bad input
   with an error wrapping `ErrUnsafePath` that names the offending path.
9. `WriteFileAtomic` writes through a temp file in the target's directory and a
   rename, applies `perm` exactly, and leaves no temp file behind on success or
   on failure.
10. `go test -race ./...` passes and the code is `gofmt`-clean.

Run `go test -race ./...` from inside `exercise/`; it fails on the starter. When
it is green, build the tool and watch it behave — the exit codes are the
interesting part:

```sh
go build -o runner .
./runner git status;             echo "exit: $?"
./runner nosuchprogram;          echo "exit: $?"
./runner -timeout 200ms sleep 5; echo "exit: $?"   # 124: it gave up
./runner sleep 30                                  # now press Ctrl-C: 130
```

## Further reading

- [pkg.go.dev — os/exec](https://pkg.go.dev/os/exec) — read `Cmd`'s fields top to
  bottom once: `Env`, `Dir`, `Cancel`, `WaitDelay`, the `StdoutPipe` warning.
- [pkg.go.dev — os/signal](https://pkg.go.dev/os/signal) — `NotifyContext` plus
  the "Types of signals" and "Default behavior of signals" notes.
- [pkg.go.dev — path/filepath](https://pkg.go.dev/path/filepath) — `IsLocal`,
  `Clean`, `EvalSymlinks`, and why `Join` is not a security boundary.
- [LWN — Ensuring data reaches disk](https://lwn.net/Articles/457667/) — the
  fsync/rename story in full, and the source of the recipe above.
