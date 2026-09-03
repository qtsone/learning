# Your Dev Environment

> `shared.foundations.dev-environment` · ~2-3h · Stage: Foundations

## Objectives

By the end of this lesson you can:

- Install the language toolchain for your chosen track and verify the
  installation from the terminal.
- Explain what environment variables are, set one persistently in your shell
  profile, and verify it with `echo`.
- Explain why the toolchain's binary directory must be on `PATH` and fix a
  setup where it is not.
- Organize a project directory with a sensible layout and initialize it under
  version control.
- Verify the whole setup end to end by creating, building, and running a
  minimal program.

## From lessons to a workshop

Everything in this stage so far gave you ideas and tools: what a program is,
how to drive the terminal, an editor, git. This last lesson assembles them
into a **development environment** — a machine set up so that going from "I
have an idea" to "a program ran" takes seconds, not an afternoon of fighting
your computer. You will set it up once, verify every piece, and then trust it.

## What a toolchain is

Remember the first lesson of this stage: source code is text for humans, and
something must translate it before the CPU can run it. A **toolchain** is the
set of programs that does that job and everything around it — a compiler or
interpreter at the core, plus helpers: a build tool, a test runner, a
formatter, a way to download libraries. Languages ship these together so they
work as one kit, usually behind a single command.

In Go: the entire toolchain lives behind one command named `go`. You hand it a
subcommand for each job — `go version`, `go build`, `go run`, and more you
will meet in the next stage. Install that one command and you have everything.

## Installing the toolchain

Two sane routes exist on macOS and Linux, for almost any language:

- **The official installer** from the language's website. Most current
  version, works everywhere, sometimes needs one manual `PATH` step.
- **A package manager** — Homebrew on macOS (`brew install …`), your distro's
  on Linux (`apt`, `dnf`, …). One command, easy upgrades, but distro versions
  can lag badly behind.

Whichever route you take, the test is the same: open a **new** terminal and
ask the tool for its version. New terminal, because installers change startup
configuration that your already-open windows never re-read.

In Go:

- macOS: `brew install go`, or download the `.pkg` installer from
  [go.dev/dl](https://go.dev/dl/) and click through it.
- Linux: prefer the official tarball (distro packages are often old). Follow
  the three steps at [go.dev/doc/install](https://go.dev/doc/install) — they
  unpack Go into `/usr/local/go` and add its `bin` directory to `PATH`.

Then verify:

```sh
go version
# go version go1.25.1 darwin/arm64
```

Read the output: the tool's name, its version, then your operating system and
CPU type. If you see a version line, the toolchain is installed *and*
reachable. If you see `command not found`, keep reading — this lesson teaches
you to fix exactly that.

## Environment variables

When the shell starts a program, it hands the program a set of named text
values called **environment variables**. They are how your terminal session
carries settings: `HOME` is your home directory, `SHELL` is your shell,
`PATH` you already know from the terminal lesson. Any program can read them,
which makes them the standard way to configure tools without editing files.

Look at them yourself:

```sh
echo "$HOME"      # print one variable — $NAME means "the value of NAME"
printenv          # print them all
```

You can create your own for the current session:

```sh
export PROJECTS_DIR="$HOME/projects"
echo "$PROJECTS_DIR"
```

`export` means "make this visible to every program I start from here". But
here is the catch: this variable dies with the terminal window. Close it, open
a new one, `echo "$PROJECTS_DIR"` — empty. Session variables are sticky notes,
not engravings.

To make one permanent, put the `export` line in your **shell profile**: a
plain text file your shell reads every time a new terminal opens. Which file
depends on your shell — check with `echo "$SHELL"`:

- `zsh` (macOS default): `~/.zshrc`
- `bash` (most Linux defaults): `~/.bashrc`

Open the file in your editor (create it if missing), add the `export` line,
save. Newly opened terminals will have the variable; existing ones will not —
when in doubt, open a fresh terminal. That "edit profile, open new terminal,
verify with echo" loop is one you will repeat for years.

## PATH and the toolchain

From the terminal lesson: `PATH` is an environment variable holding a list of
directories, separated by `:`, that the shell searches — in order — when you
type a command name. `command not found` almost never means "not installed";
it usually means "installed somewhere `PATH` doesn't mention".

Two directories matter for a toolchain:

1. **Where the toolchain itself lives.** Installers usually wire this up for
   you — that is why `go version` works right after installing.
2. **Where tools you install later land.** Toolchains can download and install
   extra developer tools, and those go to a *different* directory that is
   often *not* on `PATH` out of the box. This is the classic trap: the
   language works, then some add-on tool is "not found".

Diagnose with two commands:

```sh
command -v go              # which file actually runs when you type "go"?
echo "$PATH" | tr ':' '\n' # PATH as a readable list, one directory per line
```

The fix is always the same shape — append the missing directory to `PATH` in
your shell profile:

```sh
export PATH="$PATH:/the/missing/directory"
```

`$PATH:` keeps everything already there and adds one more place to look.

In Go: ask Go where that second directory is with `go env GOPATH` — installed
tools land in the `bin` folder inside it (usually `~/go/bin`). The editor
lesson introduced language servers; Go's is called `gopls`, and when you
install the Go extension in VS Code it offers to install `gopls` — into
exactly that folder. So add this line to your shell profile now:

```sh
export PATH="$PATH:$HOME/go/bin"
```

## A home for your projects

You will create many projects. Decide where they live *once*, and every
future "where did I put that?" disappears. The convention is simple:

- One directory for all projects, e.g. `~/projects`.
- One subdirectory per project, named in lowercase without spaces: `hello`,
  `budget-tracker`.
- Inside each project, from day one: a `README.md` saying what this is, a
  `.gitignore` for files git should never track (remember the git lesson —
  build outputs don't belong in history), and your source files.
- `git init` immediately, and commit early. A repository from minute one
  costs nothing and means every experiment is undoable.

In Go: one more day-one file. Every Go project starts by running
`go mod init <name>` inside the project directory, which creates a file
called `go.mod` — the project's ID card for the toolchain. What it contains
becomes clear in the next stage; for now, treat "new Go project" as: make the
directory, `go mod init`, `git init`.

## The smoke test

Plumbers fill new pipes with smoke before water: if smoke leaks, water would
have. Your version: the smallest possible program, pushed through the whole
chain — editor writes it, toolchain builds it, the binary runs, git records
it. If all of that works, your environment works, and any future failure is
in your code, not your setup.

You will not fully understand the program you are about to type, and that is
fine — it is a test pattern, not a lesson. The next stage explains every line.

## Exercise

Open [`exercise/`](exercise/) — it contains one file, `check.sh`, a script that
inspects your machine and tells you exactly what is left to fix. Run it now,
watch it fail, and work through the failures top to bottom. It is safe to
re-run as many times as you like:

```sh
bash ./check.sh
```

Acceptance criteria:

1. `go version` prints a version — the Go toolchain is installed and on
   `PATH`.
2. Go's tool directory (`go env GOPATH` + `/bin`) is on `PATH`, via a line in
   your shell profile.
3. An environment variable `PROJECTS_DIR` is set persistently in your shell
   profile and points at a real directory — your projects home.
4. A project `$PROJECTS_DIR/hello` exists containing `main.go`, `go.mod`,
   `README.md`, and `.gitignore` (ignoring the built binary, `hello`).
5. The project is a git repository with at least one commit that includes
   `main.go`.
6. Inside the project, `go build` succeeds, and the resulting program prints
   exactly: `Hello from my dev environment!`

For criterion 4, create the module with `go mod init tutor.local/hello` and
type this into `main.go` exactly:

```go
package main

import "fmt"

func main() {
	fmt.Println("Hello from my dev environment!")
}
```

Build and run it from inside the project directory:

```sh
go build -o hello .
./hello
```

When `check.sh` reports everything green in a **freshly opened** terminal —
proving your profile changes stick — you are done.

## Further reading

- [Download and install Go](https://go.dev/doc/install) — the official
  installation steps for every platform.
- [Go in Visual Studio Code](https://code.visualstudio.com/docs/languages/go)
  — the extension that wires up `gopls` for you.
- [Pro Git — Getting a Git Repository](https://git-scm.com/book/en/v2/Git-Basics-Getting-a-Git-Repository)
  — a recap of `git init` and first commits.
- [github/gitignore](https://github.com/github/gitignore) — community
  `.gitignore` templates for many languages and tools.
