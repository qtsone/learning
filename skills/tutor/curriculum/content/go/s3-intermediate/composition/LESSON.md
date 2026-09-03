# Composition & Embedding

> `go.intermediate.composition` · ~2-3h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Implement struct embedding to reuse behavior and explain how method promotion
  resolves shadowed names.
- Embed an interface in a struct to override a subset of its methods (partial
  wrapping).
- Explain why Go favors composition over inheritance and what problems
  classical inheritance causes that embedding avoids.
- Choose between embedding and an explicit named field for a given design and
  justify the trade-off.

## Go left inheritance out — on purpose

In class-based languages, reuse usually means *inheritance*: `Dog extends
Animal`, and `Dog` gets everything `Animal` has. It looks convenient and ages
badly:

- **Fragile base class** — a change deep in the base silently changes the
  behavior of every subclass, including ones in other codebases.
- **Premature taxonomy** — you must decide the *is-a* hierarchy up front
  ("is a `Square` a `Rectangle`?"), and reorganizing it later breaks everyone.
- **Tight coupling** — a subclass depends on its parent's internals: which
  methods call which, what may be overridden, what must not be.
- **The diamond problem** — two parents provide the same method; which one
  wins? Languages need special rules just to answer that.

Go refuses the whole game. There is no `extends`. Instead it splits the two
jobs inheritance was doing into two independent tools:

- **Composition** — build a bigger type out of smaller ones (*has-a*).
  Embedding, this lesson, makes that convenient.
- **Interfaces** — make different types interchangeable (*behaves-like-a*).
  That was last lesson.

Keep that split in mind throughout: embedding reuses *behavior*; interfaces
express *substitutability*. Embedding never gives you the second one.

## Struct embedding and method promotion

You glimpsed embedding in S1's structs lesson; here is the full story. Declare
a field with a type but no name, and the field is **embedded**:

```go
// Recorder collects events in order.
type Recorder struct {
	events []string
}

func (r *Recorder) Record(event string) { r.events = append(r.events, event) }
func (r *Recorder) Events() []string    { return append([]string(nil), r.events...) }

type TaggedRecorder struct {
	Recorder // embedded — no field name
	Tag      string
}
```

The embedded field's methods (and fields) are **promoted** to the outer type:

```go
tr := &TaggedRecorder{Tag: "auth"}
tr.Record("login")        // shorthand for tr.Recorder.Record("login")
fmt.Println(tr.Events())  // shorthand for tr.Recorder.Events()
```

Nothing magic happened: `Recorder` is still an ordinary field whose name is
its type name — `tr.Recorder` reaches it explicitly. Promotion is compiler
shorthand for delegation, not a parent-child relationship.

Promotion also feeds method sets (remember the methods lesson): promoted
methods join the outer type's method set, so `*TaggedRecorder` satisfies any
interface `*Recorder` satisfies — without you writing a line.

## Shadowing: who wins when names collide

Name lookup follows a **depth rule**: the shallowest name wins. A method you
declare on the outer type sits at depth zero, so it *shadows* the promoted
one:

```go
func (t *TaggedRecorder) Record(event string) {
	t.Recorder.Record("[" + t.Tag + "] " + event)
}
```

Now `tr.Record("login")` runs *your* method, and the promoted one is only
reachable explicitly as `tr.Recorder.Record`. That explicit call is the
standard delegation move: decorate the input, then hand off to the embedded
type so its state stays the single source of truth. (Careful: calling
`t.Record` inside this method would recurse forever — delegate through the
field name.)

Two embedded types at the *same* depth with the same method name are
ambiguous. That is a compile error — but only at the point of use; the type
declaration itself is legal. You resolve it the same way: name the field
explicitly.

## Embedding is not inheritance

Two differences bite people coming from class-based languages:

**1. No subtyping.** A function taking `*Recorder` will not accept a
`*TaggedRecorder`. There is no is-a relationship. When you need
interchangeability, define a small consumer-side interface — both types
satisfy it and the caller stops caring which one it got.

**2. No virtual dispatch.** Predict this program's output:

```go
type Animal struct{}

func (Animal) Name() string       { return "animal" }
func (a Animal) Describe() string { return "I am " + a.Name() }

type Dog struct{ Animal }

func (Dog) Name() string { return "dog" }

func main() {
	fmt.Println(Dog{}.Describe())
}
```

It prints `I am animal`. When `Describe` runs, its receiver is the embedded
`Animal` — it has no idea a `Dog` exists, so its call to `Name` is
`Animal.Name`, always. In an inheritance language the subclass override would
"win" from inside the base class. In Go the embedded type can never see the
outer one.

This is not a limitation to route around; it is the point. The fragile-base-
class problem exists precisely *because* a base class's behavior can be
rewired from below. An embedded Go type behaves identically whether embedded
or not, so you can reason about it in isolation — and so can its author.

## Embedding an interface in a struct: partial wrapping

Structs can embed interfaces too, and this unlocks a pattern you will meet
constantly in real code. Say last lesson's advice produced a small store
interface:

```go
type Store interface {
	Get(key string) (string, error)
	Put(key, value string) error
	Delete(key string) error
}
```

You want a wrapper that changes *one* behavior — reject writes — without
touching the rest. Embed the interface and override only what changes:

```go
type ReadOnly struct {
	Store
}

func (r ReadOnly) Put(key, value string) error { return ErrReadOnly }
```

`ReadOnly` satisfies `Store`: `Put` is yours, `Get` and `Delete` are promoted
straight through to whatever implementation you wrapped. Without embedding
you would hand-write three delegating methods; with a 20-method interface,
twenty. This is how logging, metrics, and retry decorators wrap interfaces
while overriding a subset — and how test stubs implement only the one method
a test actually calls.

One trap: the embedded `Store` is an interface *value*, and its zero value is
`nil`. `ReadOnly{}` compiles fine — the method set is complete as far as the
compiler cares — but calling a promoted method panics at runtime with a nil
dereference. The compiler checks the shape; filling the field is on you.
Wrappers should take the inner value in a constructor so a nil can't sneak
in.

## Embedding or a named field?

Embedding is an API decision, not a keystroke saver. Every promoted method
becomes part of your type's public contract — callers will use them, and
removing the embedded field later is a breaking change.

- **Embed** when the promoted methods *should* be part of the outer type's
  API — you would happily document "TaggedRecorder records events and can
  report them". Wrappers that must satisfy the same interface as the thing
  they wrap are the canonical case.
- **Named field** when the inner type is an implementation detail. You keep
  the API surface small, callers can't reach behavior you never promised, and
  there are no shadowing or ambiguity surprises. Delegating one or two
  methods by hand is cheap.

A useful test: would you write "callers may call the inner type's methods
through me" in the doc comment? If saying it out loud feels wrong, use a
named field. When in doubt, start with the named field — you can always
promote later; un-promoting breaks callers.

## Exercise

Open [`exercise/`](exercise/) — a Go module with both halves of the lesson:

- `recorder.go` — a complete `Recorder` base type. Do not modify it.
- `tagged.go` — declares `TaggedRecorder`, which embeds `Recorder`. Your
  work: give it a `Record` method that shadows the promoted one.
- `store.go` — the `Store` interface, a complete `MemStore` implementation
  (do not modify), the `ErrReadOnly` sentinel, and a `ReadOnly` wrapper that
  embeds `Store`. Your work: reject the writes.
- `tagged_test.go`, `store_test.go` — the specification. Read them first.

Acceptance criteria:

1. `TaggedRecorder` declares its own `Record` that shadows the promoted one:
   with `Tag: "auth"`, `Record("login")` stores `[auth] login`.
2. Your `Record` delegates to the embedded `Recorder` — you do not touch
   `events` directly and you do not re-implement `Events`. Mixing
   `tr.Record(…)` and `tr.Recorder.Record(…)` interleaves into one history.
3. `ReadOnly.Put` and `ReadOnly.Delete` return an error that
   `errors.Is(err, ErrReadOnly)` reports true, and the wrapped store is left
   untouched. The tests accept a bare `ErrReadOnly`; wrapping it with
   context, as in the S1 errors lesson, is what code review expects of an
   A-grade solution.
4. `ReadOnly.Get` keeps working even though you never wrote it — promotion
   from the embedded interface does the delegation.
5. `go test ./...` passes and the code is gofmt-clean.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

The starter compiles and fails the tests — the first failure even shows you
promotion in action: the un-shadowed `Record` stores events without a tag.

## Further reading

- [Effective Go — Embedding](https://go.dev/doc/effective_go#embedding)
- [Go FAQ — Why is there no type inheritance?](https://go.dev/doc/faq#inheritance)
- [Go spec — Struct types (embedded fields)](https://go.dev/ref/spec#Struct_types)
