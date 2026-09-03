# Reflection, unsafe & codegen

> `go.advanced.reflection-unsafe` · ~3-4h · Stage: Advanced Go

## Objectives

By the end of this lesson you can:

- Use the `reflect` package to walk an arbitrary struct's fields and tags, the
  way `encoding/json` does internally.
- Explain what reflection costs — performance, and the compile-time safety you
  give up — and identify code where it is and is not justified.
- Explain what guarantees `unsafe.Pointer` breaks, and recognize the few
  legitimate patterns (zero-copy conversions) versus the cargo-culted ones.
- Wire up a `go:generate` code generator and explain why generated code is
  often the better answer than reflection in Go.
- Given a problem, choose between generics, reflection, and code generation,
  and defend the choice.

## Three ways to write code that doesn't know its types

Go is statically typed, and most of the time that is the whole point. But
some code genuinely has to work on types it has never seen: a config loader,
a JSON decoder, a row scanner, a test helper. Go gives you three tools, and
they differ in *when* the type decision happens:

| Tool | Decision time | Failure lands | Cost |
|---|---|---|---|
| Generics | Compile time, per instantiation | Your build | None at runtime |
| Reflection | Runtime, per call | Production | Real; measurable |
| Code generation | Generate time, then compile time | Your laptop | A build step |

Generics you already have (S3). They solve the case where *your caller* knows
the type and you don't need to. They cannot solve "look at whatever struct
this is and find the fields tagged `conf`" — a type parameter gives you no way
to ask a type about its fields. That question needs reflection or codegen, and
this lesson builds the same decoder both ways so you can compare them with
numbers instead of opinions.

## What reflection actually is

An interface value is a pair: a **type** and a **value**. That is not a
metaphor — it is the runtime representation you have been using since S3's
interface lesson. `reflect` is simply the package that lets you read that pair
at runtime and act on it. Rob Pike's three laws of reflection say it best:
interface value to reflection object (`reflect.TypeOf`, `reflect.ValueOf`);
reflection object back to interface value (`v.Interface()`); and **to modify a
reflection object, the value must be settable**, which requires that it came
from a pointer you dereferenced with `Elem()`.

The third law is where everyone gets stuck once. `reflect.ValueOf(cfg)` gives
you a reflect.Value holding a *copy* of `cfg`; writing to it could not
possibly affect the original, so `reflect` refuses (it panics on `Set…`
rather than silently doing nothing). Pass `&cfg`, call `.Elem()`, and now you
hold something addressable that `SetString` can write through. This is exactly
why every decoder in the standard library — `json.Unmarshal`, `sql.Rows.Scan`
— takes a pointer and returns an error when you hand it anything else.

Two more distinctions worth pinning down. **Type vs Kind**: `Type` is the
specific named type (`time.Duration`, `conf.Config`), `Kind` the underlying
category (`Int64`, `Struct`, `Slice`). You switch on `Kind` to decide *how* to
convert and check `Type` when the specific type matters — note that a
`type Level int` has Kind `Int` and a `Type` whose `PkgPath()` is non-empty,
which comes back in the codegen half. And **exported vs not**: `reflect` can
read unexported fields but never set them (`f.IsExported()` is how you skip
them); reflection does not grant access the language denies you.

## Struct tags are a string convention, nothing more

You met tags in S3's JSON lesson — `json:"name,omitempty"`. Now you build the
other side of one.

```go
type Config struct {
	Addr string `conf:"ADDR,required"`
}
```

That backtick string is stored in the type's metadata and handed to you as
`reflect.StructField.Tag`, a `reflect.StructTag` with `Get` and `Lookup`. The
convention — space-separated `key:"value"` pairs — is shared by `encoding/json`,
`encoding/xml` and every ORM you will meet, but **the compiler does not check
any of it**. `conf:"ADDR"` mistyped as `conf :"ADDR"` is not an error, it is a
tag that silently does nothing. Rename `Addr` to `Address` and the tag still
says `ADDR`; rename the *key* and nothing anywhere breaks, because there are no
references — just a string compared at runtime.

`go vet`'s `structtag` check catches malformed syntax and duplicate keys — but
you must run `go vet ./...` yourself: the vet pass `go test` runs is a
high-confidence subset that does not include `structtag`. It cannot catch a
tag that is well-formed and wrong. This is the first real cost of reflection:
**you have moved a class of error from compile time to runtime**, and only for
inputs that actually exercise the field.

## Walking a struct

The whole recipe fits in a screen:

```go
rv := reflect.ValueOf(dst)
if rv.Kind() != reflect.Pointer || rv.IsNil() || rv.Elem().Kind() != reflect.Struct {
	return ErrNotStructPointer      // fail loudly; never panic on caller input
}
sv := rv.Elem()                         // settable, because it came from a pointer
st := sv.Type()

for i := range st.NumField() {
	f := st.Field(i)                // StructField: Name, Type, Tag
	key, ok := f.Tag.Lookup("conf")
	if !ok || !f.IsExported() {
		continue
	}
	if fv := sv.Field(i); fv.Kind() == reflect.String {
		fv.SetString(src[key])  // …and a case per supported kind
	}
}
```

Note the shape: `Type` for the field metadata, `Value` for the storage, index
`i` linking them. (For embedded structs, `reflect.VisibleFields(t)` flattens
promoted fields; the exercise sticks to flat structs.)

Note also what happens with a field type you don't handle: nothing, until
someone sets that key. A `[]string` field with a `conf` tag is a bug that
sits dormant in your binary until the day an operator adds `TAGS=a,b` to the
environment. Hold that thought.

## What reflection costs

- **Speed.** Every field access is a runtime lookup, a bounds check and a kind
  switch, and nothing inlines: the compiler cannot see through `reflect.Value`
  to specialize anything. You will measure a 5-10× gap on this decoder.
- **Allocations — sometimes.** This is the myth worth killing. Reflection does
  not inherently allocate; `fv.SetString` writes straight into the struct. It
  allocates when you *box*: `v.Interface()`, `reflect.New`, collecting results,
  passing values as `any`. The exercise's reflect decoder allocates nothing per
  call and is still much slower. Measure before you narrate.
- **Compile-time safety.** Tags are unchecked strings and kinds are checked at
  runtime, so mistakes surface in production, on the inputs that reach them.
- **Tooling blindness.** Rename a field and the tag does not follow. "Find
  usages" on a reflected-on field returns nothing. Dead-code elimination cannot
  prove such a field unused, so reflection-heavy programs strip poorly.
- **Readability.** `reflect` code is written once and read fifty times by
  people who are debugging something else.

So when *is* it justified? When the type is genuinely unknown until runtime and
the boundary is real: **libraries at the edge of the type system**
(`encoding/json`, `fmt`, `text/template`, `database/sql`'s `Scan` — all four
decode into *your* types, which their authors could never see); **test and tool
code** (`reflect.DeepEqual`, diff helpers, fixtures) where generality matters
and cost does not; **once, at startup**, where a cost paid once is not a cost;
and **inside a generator**, which is the next section.

And when is it not? Any hot path where the set of types is known and finite —
which, in your own service, is nearly always.

## go:generate: reflection that runs once

```go
//go:generate go run ./cmd/confgen -type Config -out conf_gen.go
```

Three things to understand about that line:

1. It is **a comment**. `go build` and `go test` ignore it completely; only
   `go generate ./...` scans for it, and only because you typed that command.
   It then runs an arbitrary program with the package directory as its working
   directory — no dependency graph, no caching, no ordering beyond file order.
2. **You commit the output.** Anyone who clones the repo gets a package that
   builds with no generator, no protoc and no network. You relied on this in
   the gRPC lesson: `taskboardpb/*.pb.go` was committed, and you never
   installed `protoc` to run those tests.
3. Generated files start with a line matching
   `^// Code generated .* DO NOT EDIT\.$` — the convention every Go tool,
   linter and diff viewer keys on. Write it exactly, including the period.

Committed output can drift from its source; the standard fix is a CI step that
regenerates and fails if anything changed
(`go generate ./... && git diff --exit-code`).

What do you get for the build step? A generator reports unsupported types **at
generate time**, for every field, whether or not a key is ever set — the
dormant bug becomes an error on your laptop. Its output is ordinary Go: it
inlines, it appears in profiles and stack traces under its own name, and you
can read it in review. And it costs nothing at runtime, because the type
decisions were made once, by you.

The costs are equally real: another tool in the build, generated code in diffs,
a regeneration step people forget, and a generator that is itself code someone
maintains. `stringer`, `protoc-gen-go`, `mockgen` and `sqlc` all pay it because
the alternative — reflection, or hand-written boilerplate that drifts — is
worse at their scale.

The generator in this exercise uses **reflection to write the code that
replaces reflection**: it walks `Config` with the same `ParseTag` the runtime
decoder uses, turns each field into a small spec of plain strings, and renders
Go source from a `text/template`. That is the pattern in miniature —
reflection at generate time is free, because "generate time" happens once, on
a developer's machine.

Two habits from that generator worth stealing. **Quote everything you emit**
(`{{printf "%q" .Key}}`): `text/template` knows nothing about Go syntax, and a
key containing a quote would otherwise write a syntax error into your package.
And **run `go/format` on the output, refusing to write if it fails**: a
generator that leaves a broken file behind takes down the package it lives in
— including, since this generator imports that package, its own ability to
run. If that ever happens, restore the previous generated file (or a stub with
the right method signature), then fix the generator and regenerate.

## unsafe: the escape hatch, and its rules

`unsafe` is how you tell the compiler "trust me, I know the memory layout".
Three facts frame everything else:

- **It is not covered by the Go 1 compatibility promise.** Code using
  `unsafe` may break on any release, and periodically does.
- **`unsafe.Pointer` is a pointer the GC understands; `uintptr` is just a
  number it does not.** Store a pointer as a `uintptr` and the garbage
  collector will neither keep the object alive nor update the number if the
  object moves — and stacks *do* move when they grow. The bug appears months
  later, under load, as corruption.
- **Only a fixed list of conversion patterns is valid**, enumerated in the
  package docs. The ones you are likely to meet: reinterpreting a pointer,
  `(*T1)(unsafe.Pointer(p))`, when the two types have equivalent layout;
  `uintptr(unsafe.Pointer(p))` for *inspection only*, such as printing an
  address; pointer arithmetic, `unsafe.Pointer(uintptr(p) + off)`, valid only
  as a **single expression**, because splitting it across statements leaves a
  bare `uintptr` the GC can invalidate in between; and the `syscall.Syscall`
  and `reflect.Value.Pointer` conversions, likewise only in the expression
  that produces them.

Since Go 1.17/1.20 you rarely need any of these by hand, because the good cases
got helpers: `unsafe.Slice`, `unsafe.String`, `unsafe.SliceData`,
`unsafe.StringData`. They replaced the old `reflect.SliceHeader`/`StringHeader`
hacks that fill older blog posts — those built headers the GC could not see and
are deprecated. **If a snippet you find online declares a `StringHeader`, it is
out of date.**

You are not without checks. `go vet` has an `unsafeptr` analyzer for the
uintptr rules, and the race detector you run on every test also enables the
runtime's `checkptr` instrumentation, which catches invalid or misaligned
pointer arithmetic as it happens.

### The one pattern worth knowing

Zero-copy conversion between `[]byte` and `string`:

```go
func UnsafeString(b []byte) string { return unsafe.String(unsafe.SliceData(b), len(b)) }
func UnsafeBytes(s string) []byte  { return unsafe.Slice(unsafe.StringData(s), len(s)) }
```

`string(b)` copies, because strings are immutable and Go will not let a
mutable slice alias one. `UnsafeString` skips the copy and hands you a string
that shares the array — so **the caller must guarantee `b` is never modified
while that string is reachable**. Break that and you get silently wrong
behaviour: a map key that hashes to one bucket and compares as something else,
a cached string that mutates behind your back. `UnsafeBytes` is worse: string
data may live in read-only memory, so writing through it can fault the
process, and literals are shared, so a "successful" write can corrupt
unrelated code. Read-only, always.

### The cargo cult

Most `[]byte`→`string` unsafe tricks you will see in the wild buy nothing,
because the compiler already elides the copy in the common cases:

```go
m[string(b)]          // map lookup: no allocation
if string(b) == key   // comparison: no allocation
for _, r := range string(b)   // range: no allocation
```

A short conversion whose result does not escape also lands in a stack buffer,
not the heap. What is left, and genuinely wins, is aliasing a **large** buffer
you are only scanning — which is what the exercise measures: same allocation
count, 16 KiB per operation versus 64 bytes.

The rule to leave with: `unsafe` needs a benchmark, a comment stating the
invariant, and a reviewer who agrees. Without all three it is someone else's
outage.

## Exercise

Open [`exercise/`](exercise/) — module `tutor.local/reflection-unsafe`, one
package `conf` plus the generator in `cmd/confgen`:

- `config.go` — the `Config` type, its tags, and the `//go:generate` line.
- `decode.go` — `ParseTag` and the reflect-based `Decode`. **Your code.**
- `unsafeconv.go` — `UnsafeString`, `UnsafeBytes`, `Lookup`. **Your code.**
- `cmd/confgen/specs.go` — `fieldSpecs`, the generator's brain. **Your code.**
  `main.go` and `render.go` (flags, template, formatting) are given.
- `conf_gen.go` — a placeholder that `go generate` replaces.
- `NOTES.md` — benchmark numbers, two experiments, and a decision table.

Acceptance criteria:

1. `ParseTag` implements the tag rules exactly: unexported fields, untagged
   fields and `conf:"-"` are skipped; an empty key falls back to the Go field
   name; unknown options are ignored; `required` is recognized.
2. `Decode(src, dst)` fills a struct through a pointer. A dst that is not a
   non-nil pointer to a struct returns `ErrNotStructPointer` — an error, never
   a panic. Absent keys leave their field untouched, so a pre-filled struct
   acts as defaults. Supported kinds: string, bool, and the signed integers,
   parsed with the field's own bit width so an `int8` rejects `9000`.
3. Every field failure returns a `*FieldError` naming the Go field *and* the
   key, wrapping the cause so `errors.Is(err, ErrMissing)` and
   `errors.Is(err, strconv.ErrSyntax)` work.
4. `UnsafeString` and `UnsafeBytes` use the `unsafe` helpers and copy nothing;
   `UnsafeString(nil)` returns `""` without panicking.
5. `Lookup` scans an env-style blob without copying it, and returns a value
   that survives the caller overwriting the buffer. Both halves are tested —
   read `TestLookupResultOutlivesData` before you start.
6. `fieldSpecs` produces one spec per decodable field, in field order, with
   `GoType` and `BitSize` set for integers, and rejects any type the template
   cannot spell — including named types — with a `*conf.FieldError` wrapping
   `conf.ErrUnsupportedType`.
7. `go generate ./...` regenerates `conf_gen.go`, and the generated
   `(*Config).DecodeMap` is behaviourally identical to `Decode`: same values,
   same `*FieldError` field and key, same `errors.Is` answers. Do not
   hand-write that file; the header says why.
8. `go test -race ./...` passes, `gofmt` is clean, and `NOTES.md` is filled in
   with your own numbers.

From inside `exercise/`:

```sh
go test -race ./...             # the gate
go generate ./...               # regenerate conf_gen.go after fieldSpecs works
go test -bench=. -benchmem      # read these, nothing gates on them
go vet ./...
```

The starter compiles and fails. A good order is `ParseTag` → `Decode` →
`unsafeconv.go` → `fieldSpecs` → `go generate`.

## Further reading

- [The Laws of Reflection](https://go.dev/blog/laws-of-reflection) — the
  canonical explanation of the interface pair and settability.
- [pkg.go.dev — unsafe](https://pkg.go.dev/unsafe) — read the "valid uses of
  Pointer" list in full, plus `String`, `Slice`, `StringData`, `SliceData`.
- [Generating code](https://go.dev/blog/generate) — the `go:generate` design
  and why it is deliberately dumb.
- [pkg.go.dev — reflect](https://pkg.go.dev/reflect) — skim `StructField`,
  `StructTag`, `VisibleFields` and `TypeFor`.
