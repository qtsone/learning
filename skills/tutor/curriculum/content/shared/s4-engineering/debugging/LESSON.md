# Debugging

> `shared.eng.debugging` · ~2-3h · Stage: Engineering Practice

## Objectives

By the end of this lesson you can:

- Apply the scientific method to a bug: state a hypothesis, design a minimal
  experiment, and record what the result rules out.
- Use a debugger (e.g. delve) to set breakpoints, step through execution, and
  inspect variables to locate a fault.
- Bisect to isolate a regression, both over commits (git bisect) and over
  input size or code paths.
- Explain when print/log debugging beats a debugger and vice versa, with a
  concrete scenario for each.
- Diagnose and fix a seeded bug in a provided program, documenting the
  reproduce-minimize-hypothesize trail.

## Debugging is a method, not a mood

Everyone's first debugging technique is the same: stare at the code, change
something that feels wrong, run it again, repeat until 2 a.m. It sometimes
works, which is exactly why it's dangerous — it teaches you nothing, doesn't
scale to code you didn't write, and the fix you stumble into often papers over
the fault instead of removing it.

Professionals replace luck with a loop borrowed from science:

1. **Reproduce** — find a way to trigger the bug on demand, deterministically.
2. **Minimize** — shrink the trigger until nothing removable remains.
3. **Hypothesize** — state, in one sentence, a cause that would explain the
   evidence.
4. **Experiment** — design the *smallest* test of that hypothesis, predict the
   outcome, then run it.
5. **Conclude** — the result confirms the hypothesis or rules it out. Write
   down which. Go to 3 until the cause is cornered.
6. **Fix and prove** — make the change, and keep a test that fails without it.

The discipline lives in step 4: an experiment is only worth running if some
outcome would *rule something out*. "Let me try changing this and see what
happens" is not an experiment — there's no hypothesis, so no result can teach
you anything. And step 5's writing-down matters more than it looks: a log of
ruled-out causes is what stops you from testing the same guess three times at
1 a.m., and it's what turns "I fixed it but I don't know how" into knowledge
you can reuse and hand to a teammate.

You already own the best reproduction tool there is: from the TDD lesson, a
failing test *is* a reproduction — deterministic, self-checking, and it stays
behind as the proof in step 6.

## Step zero: read what the bug already told you

Before forming any hypothesis, collect the free evidence:

- **Error messages and stack traces** point at the crime scene — read them top
  to bottom, and find the first frame that's in *your* code.
- **Which cases fail — and which pass.** A test suite where "refund" passes
  but "all purchases" fails has just handed you a huge clue: whatever the
  fault is, it must be something a refund hides. The *pattern* of passes and
  failures is evidence; most people only read the failures.
- **What changed.** Bugs rarely appear spontaneously; if it worked yesterday,
  the diff since yesterday is the prime suspect list.

In Go: a panic prints the full goroutine stack; scan for the first line in
your package. Run one test in isolation with `go test -run TestName`, and see
every subtest verdict with `go test -run TestName -v`.

## Reproduce, then minimize

A bug you can't reproduce can't be debugged — only worried about. So first
make the failure boring: same input, same failure, every time. Flaky
reproductions usually mean a hidden input — time, randomness, iteration order,
concurrency — and finding *which* is itself a hypothesis to test.

Then shrink. If a 103-element input fails, does a 50-element one? 10? 5? Each
halving discards half the suspects, the same way binary search discarded half
the array in S2 — you are bisecting over input size. Minimization pays twice:
experiments on a 5-element input take seconds and fit in your head, and the
minimal case often *names* the bug. If every input whose length is a multiple
of four passes and everything else fails, you barely need a debugger anymore —
what in the code cares about fours?

The same halving works on code paths: disable half the pipeline (return early,
feed a canned value) and ask "is the bug still there?". Whichever half keeps
the bug owns it; recurse. Researchers call the automated version *delta
debugging*; done by hand it's just disciplined halving.

## Print debugging

The oldest technique: make the program narrate itself, run it, read the story.

It shines when the question is about *evolution over time or volume*: which
iteration goes wrong out of ten thousand, what order things actually happened
in, what a value looked like on every request since noon. A printed trace of
10,000 lines is scannable in seconds; stopping at a breakpoint 10,000 times is
not. It's also the only instrument that works everywhere: a CI runner you
can't attach to, a colleague's machine, production (where the "prints" have
grown up into structured logs). And in concurrent code it's often the *least
bad* option — pausing one goroutine at a breakpoint reshapes the timing you're
trying to observe, the same heisenbug effect you met around the race detector
in S3.

Discipline keeps it honest: label every print with where it came from
(`total after batch 3 = 12.5`, not `12.5`), print one thing per line so you
can diff two runs, and delete the prints once the bug is dead — or promote the
genuinely useful ones to real log statements.

In Go: `fmt.Printf` with the `%#v` verb prints a value in Go syntax —
`[]float64{1.5, 2}` — which distinguishes `nil` from empty and string from
number at a glance. Inside tests prefer `t.Logf`, which stays silent until a
test fails or you pass `-v`, so diagnostic output never pollutes a green run.
For narration that deserves to survive the debugging session, use `log/slog`
with key-value pairs — a later stage builds full observability on top of it.

## Interactive debuggers

A debugger runs your program under a microscope: pause it at a chosen line (a
**breakpoint**), then advance in slow motion — **step over** a line, **step
into** the function it calls, **continue** to the next breakpoint — and at
every pause inspect any variable in scope. A **conditional breakpoint** pauses
only when an expression is true (`i == 4999`), which rescues the
ten-thousandth-iteration case without ten thousand clicks.

A debugger shines when the question is about *rich state at one moment*: what
exactly is in this struct three calls deep, which branch does this unfamiliar
code actually take, where does reality first diverge from what I believe the
code does. You get to interrogate a live program without editing it, which
also makes it the best tool for exploring code you've never read — pause at
the entry point and walk.

In Go the debugger is **delve**. Install it once, then run your test suite
under it:

```sh
go install github.com/go-delve/delve/cmd/dlv@latest
cd exercise
dlv test                      # compile the tests, attach the debugger
```

A session looks like this — the commands to know:

```text
(dlv) break ledger.Lowest         # breakpoint at a function
(dlv) break ledger.go:24          #   …or at a file:line
(dlv) continue                    # run until a breakpoint hits
(dlv) next                        # step over: run this line, stop at the next
(dlv) step                        # step into the function being called
(dlv) print lowest                # show one variable
(dlv) locals                      # show every local in scope
(dlv) break ledger.go:30 if i == 8  # conditional breakpoint
(dlv) continue                    # …until the next hit; Ctrl-D or quit to leave
```

Your editor wraps the same engine: the VS Code Go extension from your S0
dev-environment lesson drives delve behind its "Debug Test" links, with
breakpoints as clickable margins. Learn the CLI once anyway — it works over
ssh, in containers, everywhere the mouse can't reach.

## Bisecting over history

Minimizing input was binary search over size; **git bisect** is binary search
over your commit history. If the tests pass at some old commit and fail at
HEAD, the culprit is one of the commits in between — and you can find it in
log₂(n) checkouts instead of n:

```sh
git bisect start
git bisect bad                # HEAD is broken
git bisect good v1.4          # this old commit/tag was fine
# git now checks out the midpoint; run your tests, then verdict:
git bisect good               #   …or: git bisect bad
# repeat until git prints the first bad commit, then:
git bisect reset              # back to where you started
```

Sixty commits need about six verdicts. Better yet, when a command can deliver
the verdict — and your test suite can — git will drive the whole hunt alone:

```sh
git bisect start HEAD v1.4
git bisect run go test ./...     # git bisects on the exit code; zero = good
```

Two habits make bisect cheap: small, single-purpose commits (each commit in
the suspect range should be one idea — the S0 git lessons preached this, now
you know why), and a test suite fast enough to run sixty times without dread.
Note what bisect does *not* need: any understanding of the code. It's pure
binary search over "good → bad", which is why it finds bugs in codebases
you've never read.

## Choosing the instrument

The tools compose — a typical hunt bisects to a commit, minimizes the input,
then steps through the small case — but each has a home ground:

| Situation | Reach for | Why |
|---|---|---|
| Wrong value somewhere in 10,000 iterations | print / conditional breakpoint | a trace scans in seconds; unconditional stepping doesn't |
| Small repro, wrong answer, cause unclear | debugger | watch state diverge from your expectation, line by line |
| "It worked last week", many commits since | git bisect | log₂(n) verdicts, zero code understanding needed |
| Only fails in CI / on another machine | print (logs) | nothing to attach a debugger to |
| Concurrency-flavored, timing-sensitive | race detector + logs | breakpoints reshape the timing under study (S3) |
| Unfamiliar codebase, "what does this even do" | debugger | pause at the entry and walk, no edits needed |

If you remember one sentence: **print debugging answers "what happened, over
time?"; a debugger answers "what is true, right now?"; bisection answers
"where did it start?"**.

## Exercise

Open [`exercise/`](exercise/) — a Go module with a twist: for the first time
in this curriculum, the code is already written. `ledger.go` computes summary
statistics for a day of expense amounts, its three functions are fully
implemented, and **each contains exactly one seeded bug**. Your job is not to
rewrite them — it's to run the loop from this lesson on each one and make the
smallest fix that removes the fault.

The files:

- `ledger.go` — the accused. Read it *after* reading the test failures, not
  before: evidence first, suspects second.
- `ledger_test.go` — the specification. Run it, and mine the pattern of
  passes and failures for clues.
- `debugging-log.md` — your lab notebook, one section per bug. Fill it in *as
  you work*, not after: symptom, minimal reproduction, hypotheses with what
  each experiment ruled out, root cause, fix. Your tutor reviews this log
  with the same weight as the code.
- `bisect-lab.sh` — optional but recommended: builds a throwaway 12-commit
  repo (far away from this exercise) with a hidden regression, for practicing
  `git bisect` for real. Run `bash bisect-lab.sh` and follow its printout.

Use at least two instruments across the three bugs: delve on at least one
(`Lowest` is a good candidate — break, step, watch the variable that should
change and doesn't), and prints or `t.Logf` where a trace serves better. Be
ready to narrate your delve session to your tutor.

Acceptance criteria:

1. `go test ./...` passes, with **no changes to `ledger_test.go`** — you fix
   the code, never the evidence.
2. Each fix is minimal and idiomatic: the fault is removed, the function's
   documented contract is unchanged, and nothing is rewritten wholesale.
3. `debugging-log.md` has a completed section per bug: the failing test and
   its message, a minimal reproduction, at least one hypothesis with the
   experiment that confirmed or ruled it out, the root cause in one sentence,
   and the fix.
4. You used delve on at least one bug and can walk your tutor through the
   session: which breakpoint, which stepping commands, what you inspected.
5. The code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They FAIL right now — three functions, three stories. Start with the pattern
of which cases pass.

## Further reading

- [Delve CLI documentation](https://github.com/go-delve/delve/tree/master/Documentation/cli) — the full command set behind `dlv test`
- [git-scm.com — git bisect](https://git-scm.com/docs/git-bisect) — including `bisect run`, skip, and replaying a bisect
- [The Debugging Book — Andreas Zeller](https://www.debuggingbook.org/) — the scientific method and delta debugging, with executable chapters
- [Julia Evans — a debugging manifesto](https://jvns.ca/blog/2022/12/08/a-debugging-manifesto/) — the mindset, distilled
