# Tutor notes — Code Organization

## Where the learner is

Second lesson of S4, straight after Clean Code. They write idiomatic
intermediate Go (interfaces, generics, errors with `errors.Is`, io, testing,
concurrency) and have built two capstones, but everything so far was
organized *for* them — this is the first lesson where structure itself is the
material. They know Go packages mechanically from S1 (`packages-modules`);
what's new is deciding boundaries and defending them. The architecture-test
idea (`layout_test.go`) will be novel — point at it as a real technique, not
exercise magic.

## Common misconceptions

- **"More packages = better organized."** Package count is not structure;
  every boundary freezes a surface and adds a hop. Push them to defend *each*
  split with a reason to change.
- **`utils`/`helpers`/`common` as legitimate homes.** These name a shape, not
  a purpose, and become dependency magnets. Ask what job the code does; the
  job is the package name.
- **Go module vs. package confusion.** One `go.mod` = one module; the
  organizational unit here is the package. The lesson flags it; re-flag it if
  they say "module" for `expense`.
- **Dependency direction confused with call direction.** They may think "low
  must never *call* high." Imports must point down, but calls can go up via
  an interface the low package defines (`io.Writer` from S3). This is the
  subtlest point in the lesson.
- **Fixing the ledger→report edge by moving `FormatCents` into `ledger`.**
  That removes the arrow but relocates presentation into the domain — wrong
  home, same disease. The right move is deleting `Describe` (its behavior
  already lives in `Summary`); moving it to `report` as a function taking the
  total is acceptable if they argue for it.
- **"The cycle rule is Go being restrictive."** Other languages allow cycles
  and pay in init-order bugs and untestable pairs. The compile error is early
  detection, not limitation.
- **Refactoring top-down.** Starting with `main` strands them on packages
  that don't exist yet; the lesson says bottom-up — if they flounder, check
  the order they picked.

## Grilling points

- "Why must `expense` import nothing from this module? Try making it import
  `ledger` and read what the compiler says." (Cycle: ledger→expense already
  exists. Have them connect the error to the diagram.)
- "`Describe` lived in `ledger` and used `report.FormatCents`. List every
  place that behavior could live, and argue for the one you chose."
- "`report.Summary` takes a `map[string]int`, not a `*ledger.Ledger`. What
  does that buy? What would `report` importing `ledger` cost?" (No import
  edge, report testable with a literal map, reusable for any totals source.)
- "TestDependencyDirection is a test that reads source code. Where would a
  rule like this live in a real team — and what happens to an architecture
  rule that exists only in a wiki?"
- "When would you keep this whole program one flat package? What signal
  would make you split it later?" (Objective 4 — demand a concrete signal,
  not vibes.)
- "Storage needs to notify the UI when data changes. Imports point down —
  how does the call go up?" (Interface defined by the low package; S3
  `io.Writer` analogy.)
- "What let you delete the tangled main with a straight face?" (Green tests
  = behavior proof; same stdout before and after.)

## Grading rubric

- **A** — All tests pass; `Describe` gone with a correct justification of
  where the behavior went; tangled code fully removed from `main.go`
  (wiring only, no duplicated validation/formatting); work proceeded
  bottom-up with tests run along the way; learner articulates the dependency
  rule, one way to break a cycle, and a defensible flat-vs-split position.
- **B** — Tests pass but with residue: dead tangled helpers left in `main`,
  formatting logic duplicated instead of moved, or `FormatCents` relocated
  into `ledger`/`expense` with weak justification; explanation of direction
  mostly right but wobbly on *why* (exposure/churn), or on imports-vs-calls.
- **C** — Tests pass only after heavy hinting, or the arch test was
  appeased by deleting `Describe` without being able to say why the edge was
  wrong; cannot name a reason to change per package. Pass only if time-boxed
  remediation lands; otherwise iterate.
- **Fail** — Tests failing; or the direction rule weakened to pass (they
  edited `layout_test.go`/`allowedInternal` — treat editing the spec as an
  automatic fail and a teaching moment); or solution reproduced without
  being able to walk the dependency diagram.

## Remediation ladder

1. "Run `go test ./...` and read the first failure top to bottom. Which
   package is it in, and what exactly does it expect?"
2. "Draw four boxes — main, report, ledger, expense — and one arrow per
   `import` currently in the code. Compare with the diagram in LESSON.md:
   which arrow is extra?"
3. "`Describe` produces text for humans. Which package's doc comment claims
   that job? Then look at what `Summary` already prints — what does that
   tell you about `Describe`?"
4. "Go bottom-up: finish `expense.New` alone until `go test ./expense`
   passes, then `ledger`, then `report`, and only then rewrite `main` as:
   New → skip invalid → Add → print Summary."

## After passing

Preview: "You just leaned on tests to reorganize without fear. Next —
Test-Driven Development: write the test first and let it drive the design,
so the safety net exists before the code does."
