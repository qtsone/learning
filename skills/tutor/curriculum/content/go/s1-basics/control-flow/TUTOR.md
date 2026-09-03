# Tutor notes — Control Flow

## Where the learner is

Third Go lesson. They can declare variables (`var`, `:=`), know the basic
types and zero values, and have filled in a handful of small function
bodies (`Greeting` in hello-world; `ZeroReport`, `Average`, `PriceTag` in
variables-types). This is their first exposure to loops and to
algorithmic thinking — expect off-by-one errors, hand-tracing struggles, and
at least one accidental infinite loop (teach Ctrl-C early, matter-of-factly).
The exercise is seven tiny functions in ascending difficulty; encourage
one-at-a-time greening, running `go test ./...` after each.

## Common misconceptions

- **Parentheses and braces** — writing `if (x > 0)` (gofmt strips parens) or
  omitting braces for one-liners (syntax error, not style).
- **Expecting fallthrough** — C-family learners add `break` to every case or
  fear cases bleed into each other. Go breaks implicitly; `fallthrough` is
  explicit and rare.
- **Hunting for `while`** — they may think Go "lacks" a loop. Show that
  condition-only `for` *is* while.
- **`range n` off-by-one** — believing it yields 1..n. It yields 0..n-1; in
  `Repeat` this doesn't matter (they only count iterations), which is a good
  prompt: "when would it matter?"
- **`break` inside a `switch` inside a loop** — breaks the `switch`, not the
  loop. Classic trap; pairs well with the labels discussion.
- **Short-statement scope** — trying to use `r` from `if r := i % 2; …` after
  the `if` closes and being confused by "undefined: r".
- **Infinite-loop panic** — a wrong condition in `CollatzSteps` (e.g.
  `for n > 1` with a mutation bug, or forgetting to reassign `n`) hangs
  `go test`. Teach: Ctrl-C, then reread what the condition checks and what
  the body actually changes.
- **Dodging the form** — `SumEvens` with `i += 2` and no `continue`, or
  `FirstPowerAbove` as a condition loop. Tests still pass; the criteria (and
  your grading) require the named form — the point is practicing each shape.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go deeper):

- "Walk me through `SumEvens(4)` iteration by iteration — what are `i`, the
  remainder, and `sum` at each step?"
- "In `CountPrimes`, replace `continue candidates` with plain `continue`.
  What happens, and what would the function return then?"
- "Why does Go make you write `fallthrough` explicitly when C falls through
  by default? Which default causes fewer bugs?"
- "After `if r := i % 2; r != 0 { … }`, can the next line use `r`? Why is
  that scoping useful?"
- "Rewrite `Award` as an if/else chain in your head — which version reads
  better, and what tips the balance?"
- "`continue` in a classic `for`: does `i++` still run? What about in your
  condition-only Collatz loop if you `continue`d before changing `n`?"

## Grading rubric

- **A** — All tests pass; every function uses its named form (chain, switch,
  classic+continue, range, condition-only, infinite+break, labeled continue);
  at least one short-statement `if`; gofmt-clean; learner can hand-trace any
  loop and explain what a plain `continue` would do in `CountPrimes`.
- **B** — Tests pass but one form was dodged (e.g. `i += 2` instead of
  `continue`, infinite loop rewritten as a condition loop) or the short-init
  `if` is missing; explanations otherwise solid. Fix the form, then pass.
- **C** — Tests pass only after heavy hinting, or the learner cannot trace
  `SumEvens(4)` by hand. Pass only if time-boxed remediation lands; else
  another iteration with fresh inputs.
- **Fail** — Tests failing, or the labeled continue was pasted without being
  able to say what it skips. Remediate, don't advance.

## Remediation ladder

1. "Read the failing test line aloud — which input, what did it expect, what
   did you return?"
2. "Trace it on paper for the smallest failing input: write down every
   variable after every iteration."
3. Name the moving part: "The middle piece of a classic `for` is checked
   *before* each iteration — when does yours become false?" / "Which loop
   does an unlabeled `continue` belong to?"
4. Sketch the shape verbally — "infinite `for`, an `if` with `break` inside,
   multiply after" — and let them type and rerun the test themselves.

## After passing

Preview: "Next lesson is functions proper — so far you've filled in bodies;
next you'll design signatures, return multiple values, and see why Go
functions return errors instead of throwing them."
