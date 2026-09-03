# Tutor notes — Closures & First-Class Functions

## Where the learner is

Fifth lesson of S3, right after generics. They can write interfaces,
embedding, type switches, and simple generic functions, and they've built data
structures and algorithms from scratch in S2 — so `Filter` is mechanically
easy for them; the *new* material is capture semantics and the options
pattern. They have used function literals only incidentally (maybe in
`t.Run`). No concurrency yet: keep every discussion of shared mutable state
single-threaded, and defer race questions with "hold that thought for the
goroutines lesson."

## Common misconceptions

- **"Closures capture a snapshot of the value"** — the big one. They capture
  the variable itself; mutations after capture are visible, and writes from
  the closure are visible outside. The `x := 10; …; x = 99` demo in the
  lesson settles it — have them predict the output before running.
- **"The captured local dies when the outer function returns"** — S1's
  stack-frame model overapplied. Connect to escape analysis: same mechanism
  as returning `&local` in the S1 pointers lesson.
- **"The loop-variable pitfall still exists"** (or the reverse: never existed)
  — learners who read old blog posts expect `3 3 3`; learners who only know
  1.22 don't understand why old code says `i := i`. Both need the history:
  one variable per loop before 1.22, one per iteration since.
- **"Options pattern is just a fancy config struct"** — the differentiator is
  zero-value ambiguity (deliberate 0 vs omitted field) plus growth without
  breaking call sites. If they can't articulate that, they've memorized the
  shape without the why.
- **Returning `Option` that runs immediately** — writing `WithPort` to mutate
  a package-level server or trying `WithPort(s, 9090)`. The two-stage shape
  (function returning a function) is genuinely new; slow down there.
- **`reset` declared with its own `count`** — shadowing the shared variable
  inside one closure, breaking the sharing the tests check.

## Grilling points

- "In `Counter`, where does `count` live after `Counter` returns? Walk me
  through why it isn't destroyed." (Escape to heap; closure keeps it alive.)
- "Your `inc` and `reset` — prove to me they touch the same variable, not two
  copies. What in your code guarantees that?"
- "Rewrite `MakeAdders` mentally for Go 1.21. What would the tests print, and
  what one line fixes it?" (`off := off` — and why that shadow works.)
- "Why does `NewServer` take `...Option` instead of a `ServerConfig` struct?
  Give me the zero-value argument, not just 'it reads nicer'."
- "When would you *not* use `Filter`? Show me a case where the plain loop is
  the better call." (Chained transforms allocating intermediates, complex
  bodies, early exit wanting `break`.)
- "A closure is 'a struct with one method.' Unpack that — what are the
  fields? When would you actually reach for the struct instead?"

## Grading rubric

- **A** — All tests pass; `Counter` uses one shared variable (no globals, no
  struct detour); `MakeAdders` closes over the per-iteration variable
  naturally; options are closures capturing their argument; learner explains
  capture-by-reference, the 1.22 change, and the zero-value rationale for
  options unprompted; code gofmt-clean.
- **B** — Tests pass but with rough edges: needless indirection (e.g. a
  struct inside `Counter`, options appending to a slice then applied twice),
  or the 1.22 history is fuzzy while capture semantics are solid.
- **C** — Tests pass only after hints from the remediation ladder, or the
  learner can't predict what a two-closure sharing example does without
  running it. Pass only if a re-explanation in their own words lands.
- **Fail** — Tests failing; or `Counter` uses package-level state (breaks
  independence — and the test catches it); or the options pattern was copied
  in without being able to trace what `WithPort(9090)` returns. Remediate.

## Remediation ladder

1. "Run the tests and read the first failure aloud. Which function, which
   expectation?"
2. For `Counter`: "Where must `count` be declared so that *both* returned
   functions can see it — inside `inc`, inside `reset`, or somewhere both can
   reach?"
3. For options: "`WithPort(9090)` is a call that finishes immediately. What
   value does it hand back, and who calls that value later?" Trace one option
   end-to-end on paper: create, pass to `NewServer`, applied in the loop.
4. Sketch the shape without the body: `func WithPort(port int) Option {
   return func(s *Server) { /* what single line goes here? */ } }` — let them
   fill the line and mirror it for the other options themselves.

## After passing

Preview: "Next up is the io philosophy — `io.Reader`/`io.Writer` and
composing small pieces. You'll see interfaces and closures working together
the way the stdlib intends."
