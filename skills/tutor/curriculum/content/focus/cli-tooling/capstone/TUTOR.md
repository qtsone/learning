# Tutor notes — CLI Capstone

## Where the learner is

Last lesson of the cli-tooling pack, inserted after S3. They have flags and
precedence, cobra command trees, terminal UX, os/exec plus signals, and
distribution. They have not done S4+ — no TDD lesson, no HTTP servers, no
profiling — so do not send them to material they have not met.

This is a 6-10h build, not an exercise. Expect it across several sessions, and
expect the failure mode to be *breadth*, not difficulty: five test files worth
of red output at once is discouraging even when every individual piece is
familiar. Give them the order from the lesson (`exitCode` and `ResolveColor`
first, `Authors` last) and treat each green clump as a checkpoint worth naming.

The point of the lesson is composition and judgement. Grade the design review at
least as hard as the tests.

## Common misconceptions

- **"The flag's value tells me whether the user set it."** It does not; only
  `Changed` does. The symptom is the environment silently losing to a flag
  sitting at its default. Ask them to predict `SCOUT_TOP=5 scout scan` before
  running it.
- **"A missing config file is an error."** Only if the user named it. This is
  the whole reason `configPath` returns `explicit` — learners who skip that
  return value usually write two loaders instead.
- **"CombinedOutput is easier."** It is, and it merges the child's warnings into
  the data they are about to parse. They just spent a lesson learning why their
  *own* streams are separate.
- **"The child exited 2, so the tool exits 2."** Two different contracts. `git`
  failing is a work failure (1) unless the *user* got the invocation wrong.
- **"Ctrl-C means the child crashed."** The `*exec.ExitError` says killed;
  `ctx.Err()` says why. Any learner who reports "signal: killed" to the user has
  the check in the wrong order.
- **Painting before padding.** `%-10s` on a coloured string counts escape bytes
  as columns. If their coloured table is ragged, this is it, every time.
- **Nil slices in JSON.** `null` where `[]` belongs. Their consumer is `jq`, and
  `jq '.authors[]'` on null is an error.
- **`os.Exit` inside a handler**, usually smuggled in as `log.Fatal` or
  `cobra.CheckErr`. It takes the test binary down with it.

## Grilling points

- "Why does `Execute` exist? What would break if `main` did that work?"
- "Walk me through `SCOUT_TOP=5 scout scan --top 10`. Which layer wins and how
  does the code know?"
- "Somebody wants `scout scan | grep .go` to keep colour. What do they type,
  and where in your chain does it take effect?"
- "Your tool exits 2 for a bad flag and 1 for an unreadable directory. Argue
  for merging them into 1. Now argue against." (Both arguments exist; the point
  is that they have one.)
- "`scout frobnicate` exits 1, not 2. Defend that, then tell me what it would
  cost to change."
- "The test binary re-executes itself as git. Why not just require git in CI,
  or commit a fixture binary?" (Determinism, platform coverage, no build step,
  no network.)
- "A user pipes `scout scan --json` into a script that reads `.exts[0].ext`.
  What are you now allowed to change in a patch release?"
- "Where would per-file concurrency go in `Scan`, and what would it buy you on a
  spinning disk versus an SSD?" (They have the S3 worker pool; the honest answer
  is "measure first", which they cannot do until S5 profiling.)

## Grading rubric

- **A** — All tests pass under `-race`. `main` is thin, `App` carries every
  external dependency, precedence is one readable function, errors are wrapped
  with sentinels and never string-matched, JSON and human output diverge in
  exactly one place. They can defend the flag scopes, the exit-code table, and
  the JSON contract without hedging, and they name at least one thing they would
  do differently.
- **B** — Tests pass; design has seams: colour re-decided in the renderer,
  precedence duplicated per command, error messages that do not name their
  source, or a `Resolve` so long they cannot walk it. Explanation is solid.
- **C** — Tests pass only after heavy hinting, or pass with copied structure
  they cannot justify ("why is `--version` local?" → shrug). Pass only if a
  time-boxed remediation lands; the design review is half of this lesson.
- **Fail** — Tests failing; or globals/`os.Exit` in handlers; or `os.Getenv`
  and `os.Stdout` reached for directly below `main`, which means the injection
  habit never took.

## Remediation ladder

1. "Run one test file at a time: `go test -run TestScan ./...`. Which single
   failure is upstream of the others?"
2. "Read the failure message aloud — these tests say what they wanted and why.
   What is the smallest change that moves that one line?"
3. Point at the shape, not the code: "`Resolve` is four blocks in a fixed order,
   and each one only writes what it has. Which block is missing?" / "`Authors`
   is: build the command, run it, classify the failure three ways, count."
4. Re-derive one worked layer with them — the environment layer of `Resolve`, or
   the `ExitError` branch — then hand the rest back. Never type the tree for
   them; the tree is the lesson.

## After passing

Preview: "That is the cli-tooling pack. You have a tool you could publish
tonight: tag it, run the release you built in the distribution lesson, and hand
someone the binary. Back on the main roadmap, S4 turns this instinct for
testable seams into engineering practice you apply to every project."
