# Tutor notes — Reflection, unsafe & codegen

## Where the learner is

Tenth lesson of S5. They have written HTTP and gRPC services, driven a
database, and spent four lessons under the hood: scheduler, GC, memory model
and escape analysis, profiling, then advanced testing. They have *used*
reflection all along without naming it — `encoding/json`, `sql.Rows.Scan`,
`fmt`, `reflect.DeepEqual` in tests — and they have used generated code once,
in the gRPC lesson, where `taskboardpb/*.pb.go` was committed for them.

This lesson names all of it and then argues the other way: the interesting
skill here is not writing `reflect` code, it is knowing when not to. Expect
the mechanics to land quickly and the judgement to need pushing.

Two things this lesson deliberately does **not** cover, because they belong
elsewhere: structured logging and instrumentation of the decode path (next
lesson, Observability), and building the whole service (the capstone after
it). If they start bolting `slog` into the generator, park it.

The exercise is deliberately three-part and the parts are independent —
if time runs short, `Decode` plus `fieldSpecs` plus a successful
`go generate` is the load-bearing 80%; `Lookup` can be finished after the
review.

## Common misconceptions

- **"Reflection is slow because it allocates."** Their own benchmark says
  otherwise: both decoders report 0 allocs/op and reflection is still ~7×
  slower. Make them say where the time actually goes (runtime kind switches,
  per-field tag re-parsing, no inlining, no specialization). This is the
  single best moment in the lesson to kill a folk belief with a measurement.
- **"`reflect.ValueOf(cfg)` should let me set fields."** The third law. If
  they hit the panic, celebrate it — it is the clearest teacher here. Ask
  what `ValueOf` received (a copy) and why writing to it could not work.
- **Panicking on bad input.** `reflect` panics readily; a *library* must not
  pass that panic to its caller. `ErrNotStructPointer` exists so `Decode`
  validates first. Watch for a solution that only works because the tests
  never pass a non-pointer — the tests do.
- **"Kind and Type are the same thing."** A named `type Level int` has Kind
  `Int`. The runtime decoder handles it fine; the generator refuses it,
  because generated source must be able to *name* the type. Good learners
  find this asymmetry uncomfortable — it is real, and it is the honest
  limitation of a small generator.
- **"`go test` vets my struct tags."** It does not. The vet pass baked into
  `go test` is a high-confidence subset (`atomic`, `bool`, `printf`, …) that
  excludes `structtag`; only an explicit `go vet ./...` catches a malformed or
  duplicated tag. If they believe the safety net exists, they will trust tags
  they have never checked — which is the whole thesis of this lesson turned
  against them.
- **"`go generate` runs during the build."** It does not. Nothing runs it but
  a human or CI, which is exactly why the output is committed.
- **"Generated code is second-class / I'll just edit it."** Ask what happens
  to their edit on the next regeneration, and what the header line is for.
- **"unsafe means fast."** The compiler already elides `string(b)` copies in
  map lookups, comparisons and ranges. Unsafe conversions win when a *large*
  buffer would be copied — their `Lookup` benchmark shows 64 B/op vs 16 KiB
  at the same allocation count.
- **`uintptr` treated as a pointer.** The GC does not track it and stacks
  move. If they can restate this one clearly, they will avoid the worst
  unsafe bug class for the rest of their career.
- **`reflect.StringHeader`/`SliceHeader` from a blog post.** Deprecated and
  subtly wrong; `unsafe.String`/`Slice`/`StringData`/`SliceData` replaced
  them. Treat their appearance as a signal the learner is copying, not
  reading docs.

## Grilling points

- "Walk me through what `reflect.ValueOf(&cfg).Elem().Field(2)` actually
  holds, and why the same expression without `&` cannot be written to."
- "Your reflect decoder makes zero allocations. Show me one line I could
  change to make it allocate — and say why that line would allocate."
  (`strings.Split` for the options, `fv.Interface()`, `reflect.New`, building
  a `[]string` of keys, boxing a non-pointer into `any`.)
- "I rename `Config.Addr` to `Config.Address` with my editor's refactor tool.
  What breaks, when, and who finds out?"
- "The `Tags []string` experiment in NOTES §2: state the difference in one
  sentence, in terms of *who* sees the failure and *when*."
- "The generator imports the package it generates into. What happens if it
  writes a file that does not compile? How does `render` prevent it, and what
  would you do if it happened anyway?"
- "Why does `render` push every key through `%q` instead of writing it
  directly into the template?"
- "You have `Lookup` returning `strings.Clone(v)`. Delete the `Clone` — which
  test fails, and what would the production symptom have been?"
- "When is `m[string(b)]` an allocation? When is `f(string(b))`?" (Never for
  the map lookup; for the call, only when the string escapes or is long.)
- "Give me a case where you would reach for reflection in your own service
  code — not a library — and defend it."
- "`sqlc` generates row-scanning code; `database/sql` uses reflection in
  `Scan`. Both ship. What does each optimize for?"

## Grading rubric

- **A** — All tests pass under `-race`. `ParseTag` is shared by both decoders
  (not duplicated in the generator); `Decode` validates its target and never
  panics; integer parsing uses the field's own `Bits()`; `FieldError` wraps
  causes so `errors.Is` works. `fieldSpecs` rejects named types with a reason,
  and `conf_gen.go` was produced by `go generate`, not typed. `Lookup` clones
  the returned value and aliases nothing else. `NOTES.md` has their own
  numbers, both experiments run, and a decision table whose justifications
  are about failure timing and cost, not taste. They can explain the
  `UnsafeString` invariant without prompting.
- **B** — Tests pass, but one thing is shallow: tag rules re-implemented in
  the generator instead of calling `ParseTag`, `strconv.Atoi` instead of
  width-aware `ParseInt` (works for `int`, fails the int8 case — if they
  passed it, they did the right thing), or `NOTES.md` numbers recorded
  without the explanation. Judgement is sound but stated vaguely.
- **C** — Tests pass only after heavy hinting; or `conf_gen.go` was
  hand-written and `go generate` never ran successfully; or they cannot say
  what invariant `UnsafeString` imposes on callers. Pass only if remediation
  lands in-session.
- **Fail** — Tests failing; or `unsafe` used by cargo cult ("the TODO said
  so") with no account of the aliasing rule; or `Lookup` returns an aliased
  string and they argue the test is wrong; or they cannot articulate a single
  case where reflection is the wrong tool. Remediate, don't advance.

## Remediation ladder

1. "Read the failing test name aloud. `TestParseTag/Fallback` — what does the
   tag `conf:\",required\"` mean, and what should the key be?" Get `ParseTag`
   green first; everything downstream depends on it, and a wrong key makes
   three other tests lie to you.
2. Now name the concept under *their* failure. `Decode` panicking on a
   `Set…`: "What did you hand to `reflect.ValueOf` — the struct, or its
   address?" `fieldSpecs`: "You already wrote this loop once. Open `Decode`
   next to it — same walk, same `ParseTag`, but instead of *setting* a value
   you are *describing* it." `Lookup`: "Two questions in order. Which parts of
   your function may point into `data` (all of them), and which part outlives
   the call (the return value)?"
3. Now the tool. `Decode`: "Print `rv.Kind()` and `rv.CanSet()` at the top of
   the loop" — let the printout teach the third law — "then find the two calls
   that turn an address into something settable." `fieldSpecs`: "What does the
   template need to know about `Port` in order to write
   `strconv.ParseInt(s, 10, 64)`? Name each thing, then give the spec a field
   for it." `Lookup`: "Re-read `TestLookupResultOutlivesData`. What in
   `strings` hands back a copy that owns its own bytes?"
4. Only if still stuck on the generator: run `go generate ./...` together and
   read the error, or have them add `fmt.Printf("%+v\n", specs)` in `run`
   before rendering. Never hand them `conf_gen.go` — the point of the lesson
   dies with it.

## After passing

Preview: "You have seen the two ways Go code learns about itself, and when
each is worth it. Next lesson turns the microscope outward: observability —
`slog` done properly, metrics, and traces, so a running service can tell you
what it is doing without a profiler attached."
