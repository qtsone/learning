# Type Assertions & Switches

> `go.intermediate.type-assertions` · ~2-3h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Implement type assertions using the comma-ok form and explain when the
  single-value form panics.
- Implement a type switch that handles multiple concrete types behind one
  interface.
- Choose between `errors.Is` and `errors.As` for a given error-inspection need
  and justify the choice.
- Explain why `errors.Is`/`errors.As` traverse wrapped error chains and how
  `%w` wrapping enables that.
- Explain why heavy type-switching can signal a design that wants a new
  interface method instead.

## The concrete type is still in there

From the interfaces lesson: an interface value is a pair — the **dynamic
type** and the value itself. Assigning a `*os.File` to an `io.Reader` variable
doesn't transform the file; it wraps it. The `*os.File` is still in there,
you've just agreed to talk to it through a narrower doorway.

Most of the time that's exactly what you want — code that depends only on
behavior. But three situations force you to look *behind* the interface:

- **Boundaries.** Values arrive typed as `any` (the empty interface —
  literally an interface with zero methods, so everything satisfies it):
  decoded config, generic containers, `fmt`-style APIs.
- **Optional behavior.** You hold an `io.Reader`; does it *also* know how to
  close? The extra ability isn't in the type you were given.
- **Error inspection.** `error` is just an interface with one method. "What
  exactly failed?" means asking what's behind it.

Go gives you two tools for asking: assertions and type switches.

## Type assertions

A type assertion takes an interface value and a type, and asks "is this what's
really in there?"

```go
var r io.Reader = os.Stdin

f := r.(*os.File)        // single-value form
```

If the dynamic type is `*os.File`, you get the file back with its full type —
methods and all. If it's anything else, the program **panics**:

```
panic: interface conversion: io.Reader is *strings.Reader, not *os.File
```

That's the same all-or-nothing shape as indexing a slice out of range: no
error value, just a crash. So the single-value form is only right when the
dynamic type is an *invariant* — something your code guarantees, where being
wrong is a bug you want loud. Everywhere else, use the **comma-ok form**:

```go
f, ok := r.(*os.File)
if !ok {
    // r holds something else; f is the zero value (nil here)
}
```

This mirrors the map-lookup and channel idioms you already know: `ok` reports
success, and nothing panics — not even when `r` is nil.

Two details worth pinning down:

- Assertions only apply to *interface* values. `x.(int)` where `x` is already
  an `int` doesn't compile — there's nothing hidden to reveal.
- You can assert to another **interface** type, not just a concrete one. Then
  the question changes from "is it exactly this type?" to "does its type
  implement this interface?":

```go
if s, ok := v.(fmt.Stringer); ok {
    return s.String()      // v's type has a String() method — use it
}
```

This is *behavior discovery*, and the standard library leans on it heavily:
`fmt` checks whether your value is a `Stringer`, `io.Copy` checks whether the
reader is a `WriterTo` to skip its own buffer. You'll meet more of these in
the io lesson later this stage.

## Type switches

Asking about types one comma-ok at a time gets clumsy. A **type switch**
handles several possibilities in one structure:

```go
func Describe(v any) string {
    switch v := v.(type) {
    case nil:
        return "nothing"
    case string:
        return fmt.Sprintf("text %q", v)   // v is a string in this case
    case int:
        return fmt.Sprintf("number %d", v) // v is an int in this case
    default:
        return fmt.Sprintf("unexpected type %T", v)
    }
}
```

The magic is in `v := v.(type)`, legal only in a switch header: within each
`case`, `v` has *that case's type*. No second assertion needed. Notes:

- `case nil` matches an interface holding nothing at all — the one place you
  can "assert nil" safely.
- Cases match the dynamic type **exactly**: an `int64` does not match
  `case int`. Go never blurs numeric types, and a type switch is no exception.
- A case may list several types (`case int, int64:`) — but then `v` keeps the
  interface type inside that case, because Go can't know which one matched.
- Interface types can appear as cases. Order matters then: cases are tried
  top-to-bottom, and a concrete type that also implements an interface case
  will stop at whichever comes first.
- `%T` is the format verb that prints a value's concrete type name — the
  debugging companion of everything in this lesson.

## Errors are interface values too

Here's where this lesson stops being abstract. In S1 you learned sentinels,
`%w`, and `errors.Is`. Now you have the vocabulary for what was really going
on: `error` is an interface, every error you've ever returned had a concrete
type behind it, and error inspection is type inspection.

Sentinels answer "did *this specific thing* happen?" But some failures carry
**data** — which field was invalid, which path failed. For those you define an
error *type*:

```go
type ValidationError struct {
    Field  string
    Reason string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("invalid %s: %s", e.Field, e.Reason)
}
```

Any type with an `Error() string` method satisfies `error` — implicit
satisfaction, nothing to declare. The real standard library does exactly this:
`os.Open` failures are `*fs.PathError` values carrying `Op`, `Path`, and the
underlying cause.

So to get the fields back, just type-assert? Tempting — and wrong:

```go
ve, ok := err.(*ValidationError)   // fragile!
```

The moment any layer wraps the error — `fmt.Errorf("get %q: %w", key, err)` —
the dynamic type of the *outer* error is fmt's internal wrapper type, not
`*ValidationError`. The assertion sees only the outermost layer, `ok` is
false, and your inspection silently breaks the day someone adds context.

## How %w builds a chain, and how Is/As walk it

`%w` does two things: it formats the wrapped error into the message (like
`%v`), *and* it makes the resulting error carry an `Unwrap() error` method
that returns the original. Wrap twice and you have a linked list of errors —
each node adds context, each `Unwrap` steps inward. (You built linked lists
from scratch in S2; this is one, made of errors.)

`errors.Is` and `errors.As` both walk that list so your check doesn't care how
many layers of context were added between the failure and you:

- **`errors.Is(err, target)`** walks the chain asking "is any node *this
  value*?" — identity comparison, made for sentinels. It's why
  `errors.Is(err, os.ErrNotExist)` works even though `os` actually returns a
  `*fs.PathError` that merely wraps the sentinel.
- **`errors.As(err, &target)`** walks the chain asking "is any node of *this
  type*?" — and on a match, copies that node into `target` so you can read
  its fields:

```go
var ve *ValidationError
if errors.As(err, &ve) {
    log.Printf("bad field: %s", ve.Field)   // data recovered through any wrapping
}
```

You pass a *pointer* to your variable (`&ve`) because `As` needs somewhere to
store the match. Declare the variable, pass its address, use it only when
`As` returns true.

Choosing between them is about what you need from the failure:

| You need | Use | Typical target |
|---|---|---|
| "Did X happen?" — a yes/no on a known failure | `errors.Is` | sentinel value (`ErrNotFound`) |
| Data off the failure — fields, structure | `errors.As` | error type (`*ValidationError`) |

And a rule from the other side of the API: if callers only ever need yes/no,
export a sentinel; export an error type only when there's data worth
extracting. Types are a bigger commitment — their fields become API.

## When a type switch is a design smell

`fmt` type-switches over `int`, `string`, `bool`… and that's fine: the set of
built-in types is closed, and the switch lives in one place at a boundary.

The smell is a type switch over **your own types** repeated through core
logic:

```go
switch s := shape.(type) {
case Circle:    area = math.Pi * s.R * s.R
case Rect:      area = s.W * s.H
}
```

Add `Triangle` and you must find and extend every such switch in the codebase
— miss one and it silently falls through. The switch is the compiler telling
you the behavior belongs *on the types*: give the interface an `Area()`
method, let each shape implement its own case, and new types plug in without
touching existing code. That's the same open-for-extension instinct as the
composition lesson.

Rule of thumb: type-switch at boundaries over types you don't control
(decoding, formatting, error inspection). Inside your core logic, over types
you *do* control, reach for a method first.

## Exercise

Open [`exercise/`](exercise/) — package `inspect`, two work files:

- `describe.go` — `Describe` (a type switch) and `Stringify` (comma-ok
  assertions).
- `store.go` — a tiny key-value store with a sentinel, a validation error
  type, and the inspection helpers callers would actually use.

Read the tests first — they are the specification.

Acceptance criteria:

1. `Describe(v)` returns, by dynamic type: `nothing` for nil; `text "hi"` for
   a string (quoted with `%q`); `number 42` for an int; `boolean true` for a
   bool; `list of 3 items` for a `[]string` (its length); and
   `unexpected type float64`-style (`%T`) for anything else — including
   `int64`, which is *not* `int`.
2. `Stringify(v)` returns `(s, true)` for a plain string, `(v.String(), true)`
   for any `fmt.Stringer`, and `("", false)` otherwise — nil included, and it
   must never panic.
3. `(*ValidationError).Error()` returns `invalid <Field>: <Reason>`.
4. `Get(key)`: an empty key returns a `*ValidationError` (field `key`, reason
   `must not be empty`) wrapped so the full message is exactly
   `get: invalid key: must not be empty`; a missing key returns `ErrNotFound`
   wrapped so the message is exactly `get "ghost": not found`; a present key
   returns its value and a nil error. Wrap with `%w`, not `%v`.
5. `IsNotFound(err)` is true when `ErrNotFound` is anywhere in the chain, and
   false for nil and for a *different* error that merely has the same text.
6. `InvalidField(err)` returns `(field, true)` when a `*ValidationError` is
   anywhere in the chain, `("", false)` otherwise.
7. `go test ./...` passes and the code is `gofmt`-clean.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They FAIL on the starter — make them green.

## Further reading

- [A Tour of Go — Type assertions](https://go.dev/tour/methods/15) and
  [Type switches](https://go.dev/tour/methods/16)
- [Go blog — Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)
- [Effective Go — Interface conversions and type assertions](https://go.dev/doc/effective_go#interface_conversions)
- [pkg.go.dev/errors](https://pkg.go.dev/errors)
