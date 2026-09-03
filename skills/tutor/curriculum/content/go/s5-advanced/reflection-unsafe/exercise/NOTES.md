# Reflection, unsafe and codegen — notes

Fill this in as you work; the tutor reviews it with you. The numbers must be
yours, from your machine.

## 1. What reflection costs

Once both decoders work, run:

```sh
go test -bench=Decode -benchmem -count=5
```

Paste the lines for `BenchmarkDecodeReflect` and `BenchmarkDecodeGenerated`:

```
(paste)
```

- ns/op, reflect vs generated: ______ vs ______ (roughly ____×)
- allocs/op, reflect vs generated: ______ vs ______

The allocation columns probably surprised you. Reflection is not automatically
an allocation story — this decoder writes through `reflect.Value` without
boxing anything. So where does the time go? Name two costs that are *not*
allocations (look at what your `Decode` does per field, per call, that the
generated code does not do at all):

- Now make the reflect decoder allocate on purpose: change one line so it
  parses the tag options with `strings.Split` instead of `strings.Cut`, or
  reads a field back with `fv.Interface()`. Re-run the benchmark and record
  what changed. Undo it afterwards.

- Is a 5-10× win on this function a reason to generate code, on its own?
  What would you need to know about the program first?

## 2. Where the failure lands

Add a field to `Config` that neither decoder supports:

```go
Tags []string `conf:"TAGS"`
```

Now run both:

```sh
go generate ./...                       # generate time — on your machine
go test -run TestDecodeUnsupportedType  # run time — wherever this ships
```

- What did `go generate` print, and did it write a file?

- What does the *runtime* decoder do with the same field when `TAGS` is
  absent from the source? When it is present?

- One sentence: which failure would you rather explain to an on-call
  engineer, and why?

(The generator's own tests fail while that field exists — they expect the
real `Config`. That is the experiment working, not a broken exercise.)

Remove the field again and regenerate before moving on.

## 3. unsafe

- Which `unsafe` helpers did `UnsafeString` and `UnsafeBytes` use, and why
  are they preferable to hand-built `reflect.StringHeader` / `SliceHeader`
  conversions?

- State the invariant a caller of `UnsafeString` must keep. What in your
  `Lookup` guarantees it?

- `Lookup` returns a copy. Give the concrete bug that would appear if it
  returned an aliased string instead, and say which of your tests catches it.

- Run `go vet ./...`. Then note: `go test -race` also enables the runtime's
  `checkptr` checks. What class of mistake does that catch that `go vet`
  cannot?

## 4. The decision table

For each problem, pick **generics**, **reflection**, **code generation** — or
"none of these", when the honest answer is ordinary Go — and justify it in
one line. Disagreeing with the obvious answer is fine if the reasoning holds.

| Problem | Choice | Why |
|---|---|---|
| A `Max[T cmp.Ordered](a, b T) T` helper used all over the codebase | | |
| A library that decodes arbitrary JSON into structs its users define | | |
| Adding `String()` to 30 enum constants in your own package | | |
| Mapping rows of 12 known tables to structs, in a hot request path | | |
| A test helper that reports the first differing field of two values | | |
| Reading one `[]byte` request body as a string for parsing, no retention | | |

## 5. One paragraph

If a teammate opens a pull request that adds `reflect` to a hot path in this
service, what three questions do you ask in review?
