# Tutor notes — Your Dev Environment

## Where the learner is

Last lesson of S0. They can drive a terminal, edit files in VS Code, and use
basic git (init/add/commit, branches, remotes), but they have never written or
built a program — the Go snippet in this lesson is the first code they will
ever type, and it is deliberately presented as an untaught test pattern. The
payoff of this lesson is confidence: a machine they trust before S1 starts.
Expect the most friction around shell profiles and PATH, not around Go.

## Common misconceptions

- **"I set the variable, but it's gone"** — they ran `export` in a terminal
  and expect it to persist, or edited the profile and expect already-open
  terminals to pick it up. Drill the model: profile is read at terminal
  *startup*; session exports die with the window.
- **Wrong profile file** — bash user editing `~/.zshrc` or vice versa; or
  macOS users following Linux tutorials into `~/.bashrc`. First question when
  a variable "doesn't stick": `echo $SHELL`.
- **"command not found means it's not installed"** — usually it means "not on
  PATH". Teach the reflex: `command -v <tool>` and reading `PATH` before
  reinstalling anything.
- **Typing the prompt literally** — beginners paste `$ go version` including
  the `$`. If errors look bizarre, check for this.
- **"go works, so everything Go-related works"** — the toolchain's own
  directory and `$(go env GOPATH)/bin` are different places; gopls and other
  `go install`-ed tools live in the second. This is acceptance criterion 2's
  whole point.
- **Editing `PATH` destructively** — writing `export PATH="/new/dir"` and
  wiping the list. If suddenly `ls` breaks, this is why; the fix is
  `export PATH="$PATH:/new/dir"` and a new terminal.
- **Committing the binary** — `hello` (the built program) ends up in git.
  Reconnect to the S0 git lesson: build outputs are regenerable, `.gitignore`
  exists for exactly this.

## Grilling points

- "Close this terminal, open a new one. Will `PROJECTS_DIR` still be set?
  Walk me through *why*, step by step." (Profile file read at startup.)
- "What exactly happens when you type `go` and press enter? How does the
  shell find it?" (PATH search, in order, first match wins.)
- "Why did check.sh insist the export line lives in a profile file instead of
  just being set right now?" (Persistence vs session.)
- "`go version` works but VS Code says it can't find `gopls`. Where do you
  look?" (`go env GOPATH`/bin on PATH — and the editor may need a restart.)
- "Why is the built `hello` binary in `.gitignore` but `main.go` is not?"
  (Source is the project; binaries are regenerable outputs.)
- "If `./hello` prints the wrong text after you edit `main.go`, what did you
  forget?" (Rebuild — seeds the compile-then-run model for S1.)

## Grading rubric

- **A** — `check.sh` fully green in a *fresh* terminal; profile lines are
  correct and minimal; learner can explain the PATH search, the
  session-vs-profile distinction, and why `$(go env GOPATH)/bin` matters,
  all in their own words.
- **B** — All checks green, but one concept is shaky under grilling (e.g.
  can't say why a new terminal was needed) or the profile contains cargo-cult
  duplicate lines. Correct the gap in conversation before moving on.
- **C** — Checks green only after the tutor dictated exact lines, or learner
  cannot explain what `export` did. Re-run the remediation ladder on the
  weakest objective; pass only if the re-explanation lands.
- **Fail** — `check.sh` still failing, or green only in the tutor-driven
  session (learner cannot reproduce in a fresh terminal). Do not advance:
  S1 assumes this machine works.

## Remediation ladder

1. "Run `bash ./check.sh` and read the first FAIL line aloud — what does the
   `fix:` line tell you to do?"
2. "Which shell are you running? (`echo $SHELL`) So which file does your shell
   read when a terminal opens?"
3. "Print your PATH one directory per line (`echo "$PATH" | tr ':' '\n'`). Is
   the directory from the FAIL message anywhere in that list?"
4. Dictate the exact `export` line for *their* shell profile, have them type
   it, open a new terminal, and re-run `check.sh` — then make them explain
   back why it now passes.

## After passing

Preview: "Your machine is now a workshop. Next stage you write your first real
Go program — and this time, you'll understand every line of it."
