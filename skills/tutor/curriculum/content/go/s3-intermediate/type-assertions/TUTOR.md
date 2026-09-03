# Tutor notes — Type Assertions & Switches

## Where the learner is

Third lesson of S3. They understand implicit interface satisfaction, small
interfaces, and embedding from the two previous lessons, and they bring
sentinels + `%w` + `errors.Is` from S1's errors lesson. New here: getting the
concrete type back out (assertions, type switches), custom error *types*,
`errors.As`, and the `Unwrap` mechanics that make chain traversal work.
Generics and concurrency are still ahead — if they ask "couldn't generics do
this?", park it: "next lesson, and the answer is interesting."

## Common misconceptions

- **"The comma-ok form and the single-value form are interchangeable"** — the
  single-value form panics on mismatch. If they can't say *when* it panics and
  why that's sometimes acceptable (asserting an invariant), revisit.
- **"`err.(*ValidationError)` is fine"** — the most important trap in the
  lesson. It only sees the outermost layer; one `%w` wrap anywhere and it
  breaks. `errors.As` exists precisely because wrapping is normal.
- **Passing `ve` instead of `&ve` to `errors.As`** — or declaring
  `var ve ValidationError` (value, not pointer) when the `Error` method is on
  `*ValidationError`. The compile/vet errors confuse people; walk through
  "As needs a place to store the match, and the target type must be the type
  in the chain — which is the *pointer* type here."
- **"`errors.Is` compares messages"** — it compares identity (or defers to an
  `Is` method). The test case with `errors.New("not found")` (same text,
  different value) exists to break this belief.
- **"`case int` catches `int64`"** — type switches match dynamic types
  exactly; Go never blurs numeric types. The `int64` test case targets this.
- **Type-asserting a non-interface value** — `x.(int)` where `x` is already
  concrete doesn't compile; assertions only reveal what interfaces hide.
- **Believing nil is dangerous to assert on** — comma-ok on a nil interface
  is safe (`ok == false`), and `case nil` in a type switch handles it
  explicitly. Only the single-value form panics.

## Grilling points

- "In `Get`, you wrapped with `%w`. Change it to `%v` — which tests fail and
  why? The message is identical!" (Chain severed; `Is`/`As` find nothing.)
- "Why does `errors.As` take `&ve` and not `ve`?" (It writes the match into
  your variable; without the address it has nowhere to store it.)
- "You exported both `ErrNotFound` and `ValidationError`. When would you
  choose a sentinel and when an error type, designing a new package?"
  (Yes/no → sentinel; data to extract → type; types are a bigger API
  commitment.)
- "In `Stringify`, swap the two checks so `fmt.Stringer` is tested first.
  Does any test break? When *would* order matter?" (Not here — `string` has
  no `String()` method; it matters when a concrete case also implements the
  interface case.)
- "`Describe` type-switches over built-ins and that's fine. Write me a
  version of the shape-area switch from the lesson and tell me why that one
  is *not* fine." (Open set of owned types, switch duplicated across core
  logic → wants an `Area()` method.)
- "Where does `Unwrap()` come from in the errors `Get` returns? You never
  wrote that method." (`fmt.Errorf` with `%w` returns a type that has it.)

## Grading rubric

- **A** — All tests pass; `Describe` is a single clean type switch (no
  if/else assertion ladders); `Stringify` uses comma-ok, no panic paths;
  helpers use `errors.Is`/`errors.As` (no string matching, no direct
  assertions on `err`); gofmt-clean; learner can explain why `%w` matters and
  when a type switch is a smell.
- **B** — Tests pass but with rough edges: an assertion ladder where a switch
  belongs, a redundant `err == nil` guard before `errors.Is` (harmless but
  shows uncertainty), or a shaky answer on Is-vs-As.
- **C** — Tests pass only after heavy hinting, or the learner can't explain
  why the direct assertion `err.(*ValidationError)` would fail the
  wrapped-twice test. Time-boxed remediation before advancing.
- **Fail** — Tests failing; helpers compare error strings; or the learner
  cannot articulate what an interface value hides and how to get it back.

## Remediation ladder

1. "Read the failing test name and message aloud — which input, what did it
   expect, what did you return?"
2. For `Describe`/`Stringify`: "What is the *dynamic type* of the value in
   this failing case? Which of your cases (or assertions) should it hit?"
3. For the error helpers: "The error you get from `Get` is a wrapper around a
   wrapper. Which function in the `errors` package walks inward for you —
   and which of `Is`/`As` fits when you need the *fields*?"
4. Sketch the chain on paper with them — `fmt wrapper → ValidationError` —
   and trace what `errors.As(err, &ve)` does at each node; then let them
   type the two-line body themselves.

## After passing

Preview: "You now handle 'one interface, many types' by asking at runtime.
Next lesson — generics — is the compile-time answer to a different question:
same *code* for many types, with the compiler checking it."
