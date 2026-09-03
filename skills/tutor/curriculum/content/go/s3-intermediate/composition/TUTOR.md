# Tutor notes — Composition & Embedding

## Where the learner is

Second lesson of S3, straight after interfaces (implicit satisfaction, small
consumer-side interfaces, io.Reader/Writer). They saw a one-paragraph
embedding teaser in S1 structs and know method sets from S1 methods, sentinel
errors and `errors.Is` from S1 errors. Many learners arrive with an
inheritance mental model from other languages — this lesson exists to replace
it, not just to teach syntax. Expect the "no virtual dispatch" point to need
the most repetition.

## Common misconceptions

- **Embedding = inheritance** — they expect `*TaggedRecorder` to be usable
  where `*Recorder` is expected. It isn't; substitutability comes only from
  interfaces (the test's `Sink` makes this concrete).
- **Expecting virtual dispatch** — believing that if an embedded type's
  method calls a sibling method, an outer shadow of that sibling runs. It
  never does: the promoted method's receiver is the inner value, which cannot
  see the outer type. Use the Animal/Dog example from LESSON.md.
- **Re-implementing instead of delegating** — a `Record` that appends to its
  own slice (or a re-written `Events`) passes the prefix test but fails
  `TestInnerRecorderStillReachable` because there are now two histories.
- **Infinite recursion** — writing `t.Record(...)` inside the shadow instead
  of `t.Recorder.Record(...)`; the stack overflow surprises them. Good
  moment to cement that shadowing is name lookup, not dispatch.
- **Writing a Get on ReadOnly** — hand-delegating all three Store methods.
  It works, but it means promotion never clicked; have them delete `Get` and
  watch the tests still pass.
- **Nil embedded interface** — thinking the compiler protects them.
  `ReadOnly{}` compiles; `ReadOnly{}.Get("x")` panics. Ask them why the
  compiler is satisfied (method set is complete) but runtime isn't (nil
  interface value).
- **Returning the bare sentinel without context** — `return ErrReadOnly`
  passes `errors.Is` but drops the S1 habit of wrapping with context; nudge,
  don't fail them for it.

## Grilling points

- "Walk me through what the compiler does with `tr.Record("login")` before
  you added your method, and after." (Promotion shorthand; depth rule.)
- "In the Animal/Dog example, why does `Describe` print `I am animal`? What
  would Java print, and why is Go's answer a feature?" (Fragile base class.)
- "Your `Record` calls `t.Recorder.Record`. What happens if you call
  `t.Record` instead, and why?"
- "Does a *value* of type `TaggedRecorder` satisfy `Sink`? Reason from method
  sets." (No — your `Record` has a pointer receiver, and the promoted one
  comes from an embedded value with pointer-receiver methods, so only
  `*TaggedRecorder` satisfies it.)
- "`ReadOnly` embeds the `Store` interface. Where does `Get` come from at
  runtime? What happens if the field is nil?"
- "You need a cache type that uses a `MemStore` internally but must never
  expose `Delete`. Embed or named field? Defend it." (Named field —
  embedding would promote `Delete` into the public API.)
- "When would you still pick embedding over a named field, given the risks
  you just listed?" (Wrapper must satisfy the wrapped interface; promoted
  methods genuinely belong to the outer type's contract.)

## Grading rubric

- **A** — All tests pass; `TaggedRecorder.Record` delegates via
  `t.Recorder.Record` with no duplicated state; `ReadOnly` has no
  hand-written `Get`; write rejections wrap `ErrReadOnly` with context;
  gofmt-clean; learner correctly predicts the Animal/Dog output and gives a
  coherent embed-vs-field rule.
- **B** — Tests pass but with rough edges: bare `return ErrReadOnly` without
  wrapping, or a redundant delegating `Get` on `ReadOnly`, or the
  embed-vs-field justification is shaky though the mechanics are solid.
- **C** — Tests pass only after heavy hinting (e.g. tutor had to point at the
  delegation call), or explanation shows promotion is still magic — they
  can't say where `ReadOnly.Get` comes from. Time-boxed remediation before
  advancing.
- **Fail** — Tests failing, or a "solution" that re-implements `Events`/all
  three Store methods and can't explain why the embedded alternative exists,
  or learner still asserts `*TaggedRecorder` can be passed as `*Recorder`.
  Remediate, don't advance.

## Remediation ladder

1. "Run the tests and read the first failure aloud. The events came out
   without a tag — which `Record` ran, and why?"
2. "The depth rule: a method on the outer type shadows the promoted one. What
   method could you declare on `*TaggedRecorder` so your code runs instead?"
3. "Inside your new method you need the original behavior too. The embedded
   `Recorder` is still a field — how do you call *its* `Record` by name?"
4. For the store half, mirror it: "Which two methods must stop passing
   through? Declare just those on `ReadOnly` and return an error built from
   `ErrReadOnly` — remember `%w` from the errors lesson." Let them type
   every line.

## After passing

Preview: "Next lesson: type assertions and type switches — how to safely ask
an interface value what concrete thing it's holding, and how `errors.Is` and
`errors.As` use exactly that machinery."
