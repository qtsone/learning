# Tutor notes — Generics

## Where the learner is

Fourth lesson of S3. They have interfaces (implicit satisfaction, io.Reader/
Writer, small interfaces), composition/embedding, and type assertions plus
`errors.Is/As` behind them, and they implemented S2's data structures with
concrete types — so they have personally felt the copy-paste pain generics
solve. Closures are the *next* lesson: the exercise passes function literals
(fine, they know function values from S1), but don't detour into capture
semantics yet. Concurrency is entirely unknown — no goroutine examples.

## Common misconceptions

- **"Generics replace interfaces."** They complement. Interfaces abstract
  varying *behavior* at runtime; generics abstract varying *types* over
  identical code at compile time. If this is shaky, the whole lesson is —
  drill the io.Reader vs IndexOf contrast until it lands.
- **Tilde blindness** — believing `float64` in a union admits `Celsius`.
  It doesn't; unions without `~` list exact types only. The lesson's
  delete-the-tilde experiment exists to make the compiler teach this.
- **"`any` lets me do anything."** Backwards: as a constraint, `any`
  guarantees nothing, so the body can do almost nothing with `T` (no `==`,
  no `+`, no method calls). The constraint must grant every operation used.
- **Constraint vs interface type confusion** — trying `var n Number` or
  passing `Number` as a parameter type. Union interfaces are constraint-only.
- **Expecting inference everywhere** — surprise at `cannot infer T` when
  assigning `Map` to a variable or when a type parameter appears only in a
  return type. Inference draws only on call arguments.
- **"Why not `[]fmt.Stringer`?"** — for DescribeAll. If they can't answer,
  revisit: `[]point` does not convert to `[]fmt.Stringer`; slices are not
  covariant, elements would have to be copied one by one.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go deeper):

- "Sum's body uses `+=`. Walk me through why the constraint can't be `any`
  or `comparable` — what exactly does each constraint permit?"
- "Do the tilde experiment live: remove `~` before `float64`, run the tests,
  and read me the error. What is the compiler saying?"
- "DescribeAll takes `[T fmt.Stringer]`. When would a plain
  `func([]fmt.Stringer) []string` actually be the better design?" (When
  elements genuinely have mixed concrete types — a heterogeneous slice.)
- "In `Filter`, the element type is a type parameter but the predicate is a
  plain function value. Why is `keep` not an interface or another type
  parameter?" (The varying behavior is already captured by a function value;
  only the element type varies structurally.)
- "You wrote IndexOf in S2 for ints. Where does today's version live in the
  standard library, and what should you use in real code?" (`slices.Index`;
  the stdlib.)
- "Show me a function from your own S2 code that you would NOT make generic,
  and defend that."

## Grading rubric

- **A** — All tests pass; bodies are idiomatic (range loops, `append`, no
  index-write gymnastics); can explain the tilde, why `comparable` gates
  `==`, and why `Map[int, int]` needs explicit instantiation; gives a
  coherent generics-vs-interfaces rule with an example each way.
- **B** — Tests pass but with rough edges (e.g. `make([]U, len(s))` plus
  index writes mixed with append confusion, or DescribeAll duplicating
  Map's loop — fine, but ask why); explains constraints correctly but
  wobbles on inference or the `[]point` vs `[]fmt.Stringer` question.
- **C** — Tests pass only after heavy hinting, or explanation shows they
  pattern-matched the syntax without grasping that constraints are
  compile-time contracts. Time-boxed remediation on constraints; re-grill
  before advancing.
- **Fail** — Tests failing, or cannot explain what `[T comparable]` means in
  a signature they just wrote. Remediate from the "Type parameters" section;
  don't advance.

## Remediation ladder

1. "Read the failing test name and message aloud. Which function, which
   input, what did it expect?"
2. "Forget generics for a second: write `IndexOf` for `[]int` only, on
   paper. Now, what in that code actually depends on `int`?" (Nothing but
   the signature — that's the whole trick.)
3. "For Sum: what operation does the body need? Which of the three
   constraint kinds from the lesson can grant an operator rather than a
   method?"
4. Walk one function to completion together — `Filter`: range, `if keep(v)`,
   `append` — narrating each type. Then they do the remaining bodies alone.

## After passing

Preview: "Next lesson: the function values you kept passing to Map and
Filter become the subject — closures, what they capture, and the functional
options pattern."
