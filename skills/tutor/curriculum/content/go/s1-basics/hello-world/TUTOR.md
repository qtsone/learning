# Tutor notes — Hello, Go

## Where the learner is

First contact with Go, straight after S0. They have a working toolchain and git
but have never written a compiled program. Everything is new; in `guided` mode
narrate generously and celebrate the first successful run.

## Common misconceptions

- **"go run executes the source directly"** — it compiles first, every time.
  If they believe Go is interpreted, revisit the S0 compiler/interpreter model.
- **Editing but running stale binaries** — running `./hello` after changing
  code without rebuilding. `go run .` avoids this while learning.
- **Println vs Printf confusion** from copying random snippets — keep them on
  `Println` here.
- **Capitalization** — `fmt.println` doesn't exist; Go is case-sensitive and
  exported names are capitalized (seed for the packages lesson; don't deep-dive
  yet).
- **Missing `package main` or wrong folder** — "expected 'package', found …"
  or "no Go files in …" errors usually mean they're running from the wrong
  directory.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go deeper):

- "Walk me through what happens, step by step, when you type `go run .`."
- "Why does `Greeting` *return* a string instead of printing it? What did that
  buy us in the tests?" (Separation makes it testable — plant the seed.)
- "Delete the `import "fmt"` line and compile. What happens and why?"
  (Go refuses unused/missing imports — strictness is a feature.)
- "What's the difference between an error from the compiler and a failing
  test?"

## Grading rubric

- **A** — All tests pass; `main.go` calls `Greeting` (no duplicated string
  logic); code is gofmt-clean; learner can explain every line aloud.
- **B** — Tests pass but with duplication (e.g. greeting rebuilt in `main`) or
  formatting misses; explanation mostly solid.
- **C** — Tests pass only after heavy hinting, or explanation reveals the
  program is magic to them. Pass only if time-boxed remediation lands; else
  another iteration.
- **Fail** — Tests failing, or solution copied without being able to explain
  `package`/`import`/`func main`. Remediate, don't advance.

## Remediation ladder

1. "Read the failing test output aloud — what did it expect, what did it get?"
2. "In `greet.go`, what does the function return right now, always?"
3. "You need two behaviors: empty name and non-empty name. What Go keyword
   chooses between two paths?" (They saw `if` conceptually in S0.)
4. Walk through the solution shape verbally — `if name == "" { … }` then build
   the string — but let them type it.

## After passing

Preview: "Next lesson gives names to values — variables and Go's type system.
The `string` you returned here is your first type."
