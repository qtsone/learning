# Tutor notes — Variables & Types

## Where the learner is

Second Go lesson, first-time programmer. They can write and run a
`package main` program, call `fmt.Println`, and make tests green — but a
*variable* is their first real abstraction: a name that refers to a stored
value which can change over time. Give that idea room; everything in this
stage builds on it. `Printf` verbs and conversions are brand new too, so
expect syntax fumbling — that's fine, the concepts are what matter.

## Common misconceptions

- **`=` vs `:=` confusion** — using `:=` to reassign (creates a shadowing
  error or "no new variables on left side of :=") or `=` for a first
  declaration ("undefined: x"). Anchor it: `:=` creates, `=` updates.
- **"An unassigned variable is empty/undefined"** — `var count int` holds a
  real, usable `0`. If they hardcoded `ZeroReport`'s string instead of
  declaring variables, they've dodged the objective — send them back.
- **Expecting `7 / 2` to be `3.5`** — integer division truncates. Most
  calculators and many languages taught them otherwise.
- **Expecting implicit conversion** — writing `float64(sum / count)` or
  `sum / float64(count)` and being surprised by the error or the truncated
  result. The conversion must happen to *both* operands *before* dividing.
- **Trying `:=` at package level** — it only works inside functions; the
  const block and any package-level state need `var`/`const`.
- **`"1" + "2"` is `"3"`** — operator meaning follows the type;
  concatenation, not arithmetic.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go deeper):

- "In `ZeroReport`, change `var price float64` to `price := 0`. The test
  still passes — so what changed, and why doesn't the test notice?" (Type is
  now `int`; `%v` prints `0` either way. Tests check values, not types —
  a healthy early lesson about what tests can and can't see.)
- "Predict the output of `float64(7 / 2)` versus `float64(7) / float64(2)`,
  then explain the difference step by step."
- "Why did you use `:=` for `dollars` but `var` inside `ZeroReport`? Could
  you swap them? What would you lose?"
- "What is the type of `Monday`? What's the difference if the block were
  `const Monday int = 1` — when would the typed version refuse to compile
  where the untyped one works?"
- "Why does Go refuse to compile an unused variable instead of just
  warning you?"

## Grading rubric

- **A** — All tests pass; `ZeroReport` genuinely declares four zero-valued
  `var`s (no hardcoded string, no explicit `= 0`); `Average` converts both
  operands before dividing; const block uses one `iota` expression; gofmt-
  clean; learner explains var-vs-`:=` and the integer-division trap unaided.
- **B** — Tests pass but with rough edges: an explicit `= 0` in `ZeroReport`,
  a redundant conversion, or `iota` written on every line; explanation mostly
  solid with minor prompting.
- **C** — Tests pass only after heavy hinting, or the learner can't predict
  `7 / 2` or say what a zero value is when asked cold. Pass only if
  time-boxed remediation lands; otherwise another iteration.
- **Fail** — Tests failing, `ZeroReport` hardcoded *and* learner can't
  explain zero values, or the solution is copied without understanding.
  Remediate, don't advance.

## Remediation ladder

1. "Read the failing test message aloud — it names the exact want/got and
   hints at the fix. Which word in it don't you recognize?"
2. For `Average`: "What is `7 / 2` when both sides are `int`? So what must be
   true about both sides *before* the division happens?"
3. For `ZeroReport`: "Declare `var count int` and immediately
   `fmt.Println(count)` in `main`. What prints? Nobody assigned that — where
   did it come from?"
4. For the const block: walk through `iota` line by line on paper — iota is
   0, 1, 2 … per line, and a bare name repeats the previous expression —
   then let them type the fix themselves.

## After passing

Preview: "Next lesson your programs learn to make decisions and repeat
themselves — `if`, `switch`, and Go's single loop keyword, `for`. The `bool`
type you just met is about to earn its keep."
