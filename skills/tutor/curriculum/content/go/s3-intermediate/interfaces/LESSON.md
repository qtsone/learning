# Interfaces

> `go.intermediate.interfaces` · ~3-4h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Explain how Go's implicit interface satisfaction differs from nominal
  implements-style typing and what that enables at package boundaries.
- Implement a type satisfying `io.Reader` or `io.Writer` and compose it with
  stdlib functions like `io.Copy`.
- Design a small consumer-side interface (1-2 methods) instead of a large
  provider-side one, and justify the choice.
- Explain why interfaces should be defined where they are consumed, not where
  they are implemented.
- Predict whether a concrete type satisfies a given interface by reasoning
  about its method set.

## The problem: one function, many types

In S1 you wrote functions that take a `*os.File`, or a `[]byte`, or a
`string`. Each one works for exactly that type. Now imagine writing a report
renderer. Sometimes the report goes to a file, sometimes into a buffer a test
can inspect, sometimes over a network connection. The rendering logic is
identical every time — only the destination differs. Copy-pasting three
versions is obviously wrong. What you want to say is:

> "I don't care what you are. I only care that I can *write bytes to you*."

That sentence — behavior required, identity ignored — is what an interface
expresses.

## An interface is a named method set

An interface type lists method signatures and nothing else:

```go
type Writer interface {
	Write(p []byte) (n int, err error)
}
```

A variable of an interface type can hold a value of *any* concrete type whose
method set includes those methods (method sets are the S1 `methods` lesson's
cliffhanger, paid off below). Calling a method on the interface variable calls
the concrete type's method:

```go
var w Writer = os.Stdout  // *os.File has a Write method with this signature
w.Write([]byte("hi"))     // runs (*os.File).Write
```

The check is done by the **compiler**. If `os.Stdout`'s method set didn't
include a matching `Write`, the assignment would not compile. There is no
runtime guessing: interfaces are static contracts with dynamic dispatch.

## Satisfaction is implicit

Notice what the code above *doesn't* say. `os.File` never declares
"I implement Writer". There is no `implements` keyword in Go. A type satisfies
an interface simply by having the right methods — the connection is
**structural** (based on shape), not **nominal** (based on a declared name).
Many languages work the other way: a class must state up front which
interfaces it implements, and only those count.

Implicit satisfaction sounds like a small convenience. At package boundaries
it changes how programs are designed:

- **A type can satisfy interfaces its author never heard of.** `os.File` was
  written long before your code existed, yet it satisfies any one-method
  interface you invent today, as long as the shapes match. No one has to
  coordinate.
- **Dependencies point the right way.** In implements-style languages, the
  implementing package must import the interface to name it — the concrete
  code depends on the abstraction's package forever. In Go, the implementing
  package stays oblivious. Only the *consumer* mentions the interface.
- **Retrofitting is free.** You can carve an interface out of an existing
  concrete API and every existing type that fits starts satisfying it
  immediately, with zero edits to those types.

## The two most important interfaces in Go

Both live in package `io`, and both have exactly one method:

```go
type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}
```

A `Reader` is "a place bytes come from"; a `Writer` is "a place bytes go".
Files, network connections, in-memory buffers, compressors, hashers, HTTP
bodies — a huge slice of the standard library is types satisfying these two
interfaces and functions consuming them. `io.Copy` is the classic consumer:

```go
n, err := io.Copy(dst, src)   // dst is any io.Writer, src is any io.Reader
```

`io.Copy` streams everything from `src` to `dst` without caring what either
one is. Two ready-made types you'll use constantly in tests:

- `strings.NewReader("text")` — an `io.Reader` over a string.
- `bytes.Buffer` — an in-memory `io.Writer` (also a Reader) you can inspect
  with `.String()` afterwards.

Interfaces come with *contracts* — rules the method signature alone can't
express, documented on the interface. For `Writer`:

- `Write` must return the number of bytes from `p` it consumed, and a non-nil
  error if that's less than `len(p)`. Returning `(0, nil)` after "handling"
  the bytes is a contract violation — `io.Copy` will notice the shortfall and
  fail with `io.ErrShortWrite`.
- `Write` **must not modify `p`**, even temporarily. The caller still owns
  that slice; remember from S1 that slices share their backing array.
- For a wrapping writer that transforms data, `n` reports bytes consumed
  *from `p`* — not how many bytes went downstream after transformation.

Read the doc comment on any interface you implement. The compiler checks the
shape; the contract is on you.

## Method sets decide satisfaction

From the S1 methods lesson: every type has a method set, and receivers matter.

- The method set of `T` contains the methods with **value** receivers.
- The method set of `*T` contains methods with value **and** pointer
  receivers.

An assignment to an interface variable compiles only if the method set of the
value you're assigning covers the interface. Concretely, given
`func (w *CountingWriter) Write(p []byte) (int, error)`:

```go
var w CountingWriter
io.Copy(w, src)    // compile error: CountingWriter's method set has no Write
io.Copy(&w, src)   // fine: *CountingWriter's method set includes Write
```

Why the asymmetry? An interface value stores a copy of what you put in it. If
Go called a pointer method on that stored copy, the method would mutate a
value you can never reach again — silently useless. The rule prevents that
class of bug at compile time. Practical consequence: types with pointer
receivers are used as pointers when handed to interfaces.

You can ask the compiler to verify satisfaction up front with a blank-variable
declaration — you'll find this idiom all over real codebases and in this
lesson's tests:

```go
var _ io.Writer = (*CountingWriter)(nil)  // breaks the build if it stops being true
```

One more thing worth knowing early: an interface value is a pair — the
dynamic *type* and the *value*. An interface is `nil` only when **both** are
nil. Store a nil `*CountingWriter` in an `io.Writer` and the interface is
non-nil (it has a type), which surprises people in error handling. File the
fact away; the type-assertions lesson digs into inspecting the pair.

## Small interfaces, defined by the consumer

Two pieces of Go judgment turn interfaces from a feature into a design style.

**Small beats big.** A Go proverb: *"The bigger the interface, the weaker the
abstraction."* `io.Writer` is powerful because almost anything can be one;
a ten-method `Storage` interface can be satisfied by almost nothing, and
every fake you write for tests must stub ten methods to use one. One- and
two-method interfaces are the norm in Go, common enough to have a naming
convention: method name + "er" — `Reader`, `Writer`, `Saver`, `Stringer`.

**The consumer defines it.** In implements-first languages, the package
providing a `Database` also exports a `DatabaseInterface`, and consumers
import it. Go turns that around: the *function that needs the behavior*
declares, in its own package, the minimal interface it requires:

```go
// In the report package — the consumer:
type Saver interface {
	Save(name string, data []byte) error
}

func Archive(s Saver, events []Event) error { … }
```

Why consumer-side?

- The interface documents **what this code actually needs** — one method, not
  the provider's whole API surface. Readers of `Archive` see its true
  requirements at a glance.
- The provider stays a plain concrete type. Thanks to implicit satisfaction
  it fits `Saver` without importing the `report` package — no dependency, no
  coordination, no import cycles.
- Tests become trivial: a five-line fake with one method, defined next to the
  test that uses it. You'll do exactly this in the exercise.

The same idea from the calling side is the proverb **"accept interfaces,
return structs"**: parameters are loose (any `Saver` will do), return values
are concrete (callers get a `*CountingWriter` with all its methods, and can
wrap it in their own interfaces if they wish). Exporting an interface from
the provider "just in case" is premature abstraction — Go code earns its
interfaces at the point of use.

## Exercise

Open [`exercise/`](exercise/) — a package `report` with three work sites and
two test files that are, as always, the specification. Read the tests first.

- `writers.go` — implement `CountingWriter` (an `io.Writer` that counts bytes
  and discards them) and `UpperWriter` (wraps another `io.Writer`,
  uppercasing everything that passes through — `bytes.ToUpper` does the byte
  work).
- `archive.go` — implement `Archive`, a consumer of the one-method `Saver`
  interface defined right above it.

Acceptance criteria:

1. `CountingWriter.Write` counts `len(p)` bytes and returns `(len(p), nil)`;
   `Count` reports the running total; `io.Copy(&w, src)` works — note the
   `&`, and make sure you can explain it.
2. `UpperWriter.Write` forwards the uppercased bytes to its destination
   writer, reports `len(p)` consumed on success, returns the destination's
   error on failure, and never modifies the caller's slice.
3. `Archive` calls `Save` once per event — name `e.Name`, body
   `"<name> <size>"` (e.g. `"build.log 2048"`). An empty slice means zero
   calls and a nil error. On the first `Save` failure it stops and returns an
   error that wraps the cause with `%w` and names the failing event.
4. `go test ./...` passes and the code is `gofmt`-clean.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They fail on the starter — including one failure with `io.ErrShortWrite`
that only makes sense once you've read the `io.Writer` contract above.

## Further reading

- [A Tour of Go — Interfaces](https://go.dev/tour/methods/9)
- [Effective Go — Interfaces and other types](https://go.dev/doc/effective_go#interfaces_and_types)
- [pkg.go.dev/io](https://pkg.go.dev/io) — read the `Reader` and `Writer` contracts in full
- [Go Proverbs](https://go-proverbs.github.io/) — Rob Pike's design aphorisms, several about interfaces
