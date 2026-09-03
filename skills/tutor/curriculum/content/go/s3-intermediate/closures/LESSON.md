# Closures & First-Class Functions

> `go.intermediate.closures` · ~2-3h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Implement a closure that captures and mutates enclosing state, and explain
  that variables are captured by reference.
- Implement the functional options pattern for configuring a constructor.
- Explain the loop-variable capture pitfall and how Go 1.22's per-iteration
  loop variables changed it.
- Explain where functional style helps in Go and where its limits (no method
  chaining culture, verbosity) argue for plain loops.

## Functions are values

You have called functions since S1 and written methods since the structs
lesson. What you haven't done yet is treat a function the way you treat an
`int`: store it in a variable, put it in a slice, pass it to another function,
return it from one. In Go, functions are ordinary values with ordinary types:

```go
func double(n int) int { return n * 2 }

var f func(int) int   // a variable whose type is "function from int to int"
f = double
fmt.Println(f(21))    // 42
```

`func(int) int` is a type, exactly like `[]string` or `map[string]int` is a
type. Two functions with the same parameter and result types are assignable to
the same variable. And like any other type, function types can appear as
parameters:

```go
func apply(n int, f func(int) int) int { return f(n) }
```

You don't need a named function to call `apply` — you can write one inline, a
**function literal** (often called an anonymous function):

```go
apply(10, func(n int) int { return n + 1 })   // 11
```

This is the shape you'll use constantly with the standard library. Sorting a
slice of structs, for example, takes a comparison function:

```go
slices.SortFunc(people, func(a, b Person) int {
	return cmp.Compare(a.Age, b.Age)
})
```

You wrote sorting algorithms from scratch in S2; `slices.SortFunc` is the same
idea with the "which comes first?" decision handed in as a value.

## Closures: functions that remember

A function literal can use variables from the function that surrounds it. When
it does, it becomes a **closure**: the function plus the variables it
*captured* travel together, even after the surrounding function has returned.

```go
func counter() func() int {
	count := 0
	return func() int {
		count++
		return count
	}
}

next := counter()
fmt.Println(next()) // 1
fmt.Println(next()) // 2
fmt.Println(next()) // 3
```

Pause on what just happened. `counter` returned — its local variable `count`
should be gone, by the mental model you've had since S1 ("locals live until
the function returns"). But the returned closure still uses `count`, so Go
keeps the variable alive for as long as the closure exists. The compiler
notices the capture and moves the variable to the heap — this is the escape
analysis you met when returning pointers to locals in the pointers lesson. A
closure capturing a local and a function returning `&local` are the same
mechanism wearing different clothes.

Each *call* to `counter()` creates a fresh `count`. Two counters never share:

```go
a := counter()
b := counter()
a(); a()
fmt.Println(a()) // 3
fmt.Println(b()) // 1 — b has its own count
```

## Captured by reference, not by value

The single most important fact about Go closures: they capture the
**variable**, not a snapshot of its value at capture time.

```go
x := 10
show := func() { fmt.Println(x) }
x = 99
show() // 99, not 10
```

The closure holds a reference to `x` itself. Whatever `x` is when the closure
*runs* is what it sees. This cuts both ways: a closure can also *write*
through that reference, and every other closure sharing the variable sees the
write:

```go
count := 0
inc := func() { count++ }
reset := func() { count = 0 }
inc(); inc()
reset()
inc()
fmt.Println(count) // 1 — both closures act on the same count
```

Two or more closures over the same variable are shared mutable state — a tiny
object, in effect, with the captured variables as fields and the closures as
methods. That framing ("a closure is a struct with one method") is worth
keeping: when you find yourself capturing four variables across three
closures, a named struct with methods usually says the same thing more
clearly. Later in this stage, when goroutines arrive, "closures share the
variable" becomes a safety question and the race detector will hold you to it.

## The loop-variable pitfall — and Go 1.22's fix

For its first thirteen years, Go had a famous trap. In a loop, `for i := …`
declared **one** variable `i` for the whole loop, updated each iteration. So
every closure created inside the loop captured the *same* `i` — and by the
time the closures ran, `i` held its final value:

```go
// Go 1.21 and earlier:
var prints []func()
for i := 0; i < 3; i++ {
	prints = append(prints, func() { fmt.Println(i) })
}
for _, p := range prints {
	p() // printed: 3 3 3 — every closure captured the same i
}
```

This follows directly from capture-by-reference — the closures did exactly
what closures do — but it surprised nearly everyone, and the standard fix
(`i := i` inside the loop body, shadowing the loop variable with a fresh copy)
looked like a typo to newcomers.

Go 1.22 changed the language: **each iteration now gets a fresh loop
variable**. The snippet above prints `0 1 2` today. The same applies to
`for _, v := range …` — each iteration's `v` is a new variable, so capturing
it is safe. Your modules declare `go 1.22`, so you get the new semantics; the
per-module declaration is also how Go could change this without breaking old
code (a module declaring `go 1.21` keeps the old behavior).

Why learn a fixed bug? Three reasons. You will read pre-1.22 code and older
answers online where `i := i` appears — now you know it was load-bearing, not
noise. You may work in codebases whose `go.mod` still declares an older
version. And the *reasoning* — "which variable did this closure capture, and
who else writes to it?" — remains exactly the question you'll ask when
goroutines enter the picture later in this stage.

## The functional options pattern

Here is the most idiomatic use of closures in Go APIs. Suppose you're
constructing a server with three knobs — host, port, banner — where most
callers want the defaults. Your instinct from earlier lessons might be a
config struct:

```go
srv := NewServer(ServerConfig{Host: "example.internal", Port: 9090})
```

Workable, but flawed: did the caller *choose* port 9090, or leave `Port` as
the zero value 0 meaning "default"? You can't tell a deliberate zero from an
omitted field, so defaults get awkward. Piling up constructors
(`NewServer`, `NewServerWithPort`, `NewServerWithHostAndPort`, …) is worse.

The **functional options** pattern solves this with closures. An option is a
function that edits the server under construction:

```go
type Option func(*Server)

func WithPort(port int) Option {
	return func(s *Server) { s.port = port }
}

func NewServer(opts ...Option) *Server {
	s := &Server{host: "localhost", port: 8080} // defaults, in one place
	for _, opt := range opts {
		opt(s)
	}
	return s
}
```

Call sites read like a sentence, and every knob is unmistakably deliberate:

```go
srv := NewServer(WithHost("example.internal"), WithPort(9090))
```

Look at `WithPort` closely — it's a function that *returns* a closure. The
returned `func(*Server)` captures `port` and carries it until `NewServer`
applies it. Each piece of the pattern is something you've now seen: a function
type (`Option`), a closure capturing an argument (`WithPort`), and functions
passed as values (variadic `opts`).

Why the pattern earns its keep:

- **Defaults live in one place**, inside the constructor, and callers only
  mention what they change.
- **Growth without breakage** — adding `WithTLS(…)` next year changes no
  existing call site, and the `Server` fields stay unexported. This is the
  "return structs, keep them opaque" judgment from the interfaces lesson.
- **Options are values** — they can be built conditionally, stored in slices,
  shared as presets.

The cost is boilerplate: one `WithX` per knob. For a constructor with one or
two required parameters, plain arguments are still better. Reach for options
when the *optional* configuration surface is real — you'll recognize the
pattern all over the ecosystem.

## Where functional style helps — and where it doesn't

Some languages lean hard on chaining higher-order functions:
`users.filter(active).map(name).take(10)`. Go deliberately doesn't. There's no
method-chaining culture, function literals are wordy (`func(u User) bool {
return u.Active }` versus a one-character lambda elsewhere), and until
generics arrived there was no way to even write a general `Map`. The idiomatic
Go answer to "transform this slice" is usually a `for` loop — plain, fast,
debuggable, and instantly readable by any Go programmer.

So when *does* a function-as-value earn its place? When the function is the
**varying part of an otherwise fixed algorithm**:

- **Comparators and predicates** the stdlib asks for: `slices.SortFunc`,
  `slices.ContainsFunc`, `strings.Map`.
- **Configuration** — the options pattern you just learned.
- **Deferred cleanup** — `defer func() { … }()` closing over what to clean.
- **Callbacks with state** — e.g. `http.HandlerFunc`s that close over their
  dependencies; you'll build these when you reach the stdlib's io and, later,
  web territory.

A useful test: if the closure fits on one line and slots into an API that
wants a function, use it. If you're building a pipeline of `Map`/`Filter`
calls each allocating an intermediate slice, or a closure has grown a body
long enough to need its own tests — write the loop, or name the function. In
the exercise you'll implement a generic `Filter` (your generics lesson makes
it a five-liner) precisely so you can judge both sides from experience.

## Exercise

Open [`exercise/`](exercise/) — a Go module with three files to complete and
their tests:

- `counter.go` — `Counter()` returns two closures, `inc` and `reset`, sharing
  one captured count.
- `funcs.go` — `MakeAdders` builds closures in a loop; `Filter` is a generic
  higher-order function.
- `server.go` — the functional options pattern: `NewServer`, `WithHost`,
  `WithPort`, `WithBanner`. The types and defaults are specified in the doc
  comments; the bodies are yours.

Acceptance criteria:

1. `Counter()` returns `inc` and `reset` closing over the same variable:
   `inc()` returns 1, 2, 3, …; after `reset()` the next `inc()` returns 1;
   each call to `Counter()` gets independent state.
2. `MakeAdders(offsets)` returns one function per offset; the function at
   index `i` adds `offsets[i]` to its argument. Each closure captures its own
   offset.
3. `Filter(s, keep)` returns the elements of `s` for which `keep` returns
   true, in their original order, for a slice of any element type.
4. `NewServer()` with no options yields host `"localhost"`, port `8080`,
   banner `"ready"`; options override defaults; when options conflict, the
   last one wins (they apply in order).
5. `go test ./...` passes and the code is `gofmt`-clean.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...
```

They fail on the starter — read the failures first; they are the
specification. While you work, keep asking the lesson's central question of
every function literal you write: *which variables am I capturing, and who
else can touch them?*

## Further reading

- [A Tour of Go — Function closures](https://go.dev/tour/moretypes/25)
- [Go blog — Fixing For Loops in Go 1.22](https://go.dev/blog/loopvar-preview)
- [Dave Cheney — Functional options for friendly APIs](https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis)
- [Rob Pike — Self-referential functions and the design of options](https://commandcenter.blogspot.com/2014/01/self-referential-functions-and-design.html)
