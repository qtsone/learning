# Context

> `go.intermediate.context` · ~2-3h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Implement cancellation propagation with `context.WithCancel` and honor
  `ctx.Done()` in a worker goroutine.
- Choose between `WithTimeout` and `WithDeadline` for a given call and
  explain why `cancel` must always be deferred.
- Explain why context is passed as the first parameter down call chains and
  never stored in structs.
- Explain why context values should carry request-scoped metadata only, not
  function parameters or dependencies.
- Implement a function that distinguishes `context.Canceled` from
  `context.DeadlineExceeded` and reacts appropriately.

## The problem: telling a call tree to stop

You can start goroutines, wire them with channels, wait with a `WaitGroup`,
and guard shared state with a mutex. What you cannot do yet is *call off*
work that is already in flight. Picture a server handling one request: it
calls a database, which calls a connection pool, while a sibling goroutine
fetches from a cache. The client disconnects. Every one of those functions —
across package boundaries you don't own — should stop burning CPU and
holding connections. Who tells them, and how?

You know a mechanism that broadcasts to any number of listeners: closing a
channel (from the Channels lesson — every receiver of a closed channel
unblocks immediately, forever). You could thread a `done chan struct{}`
through every function by hand, and early Go code did exactly that. It
worked, but every library invented its own flavor, and the signals didn't
compose: your `done` channel, the database driver's `quit` channel, and an
HTTP timeout had no common language.

`context.Context` is that hand-rolled done-channel pattern, standardized and
made composable. One interface, four methods:

```go
type Context interface {
	Done() <-chan struct{}            // closed when this work should stop
	Err() error                       // nil until Done is closed, then why
	Deadline() (time.Time, bool)      // when this work must finish, if set
	Value(key any) any                // request-scoped metadata (later section)
}
```

`Done()` is the heart: a channel that is *closed* — never sent on — when the
work should be abandoned. Close-as-broadcast is why it reaches a whole tree
of goroutines at once.

## Deriving contexts

You never construct a context from fields; you *derive* one from a parent.
The root of every tree is `context.Background()` — never canceled, no
deadline, empty. (`context.TODO()` is the same thing spelled "I haven't
wired context through here yet"; it marks refactoring debt.)

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
```

`WithCancel` returns a child context and a `cancel` function. Calling
`cancel()` closes the child's `Done` channel — and the `Done` of every
context derived from *it*. Derivation forms a tree: cancel a node, and the
whole subtree below it stops; siblings and parents are untouched. That tree
shape is what makes context compose across libraries — the HTTP server
cancels the request context, and the database driver three calls down sees
its `Done` close, without either knowing the other exists.

Two more constructors add a time limit:

```go
ctx, cancel := context.WithTimeout(parent, 3*time.Second)         // relative
ctx, cancel := context.WithDeadline(parent, time.Date(...))       // absolute
defer cancel()
```

`WithTimeout(parent, d)` is literally `WithDeadline(parent, time.Now().Add(d))`
— sugar, nothing more. Choosing between them is about which number you
actually have. "This RPC gets 3 seconds" is a *relative* budget: `WithTimeout`.
"The whole batch must finish by 02:00" is an *absolute* instant that stays
meaningful as it propagates through hops: `WithDeadline`. When a deadline
crosses process or function boundaries, absolute wins — each hop re-deriving
"3 seconds from *now*" would quietly extend the budget. A child's deadline
can only tighten the parent's, never loosen it: deriving `WithTimeout(ctx,
time.Hour)` from a context that dies in 2 seconds still dies in 2 seconds.

### Always defer cancel

Every `With*` constructor returns a `cancel` function, and you must call it
on every path — hence `defer cancel()` on the very next line, as a reflex.
Two reasons:

1. **Resources.** A deadline context owns a running timer (the Timers
   section of the Time lesson), and every derived context stays linked into
   its parent's tree until canceled or until the parent dies. Skipping
   `cancel` on the happy path keeps timers and tree nodes alive long after
   the work finished.
2. **Goroutine leaks.** Any goroutine you started that blocks on
   `<-ctx.Done()` stays blocked until someone closes that channel. `cancel`
   is how you guarantee "when this function returns, everything it started
   knows to stop."

Calling `cancel` twice is safe (it's idempotent, via `sync.Once` semantics
you met in the Sync lesson), and canceling after the work succeeded is not
an error — it's the point. `go vet ./...` even ships a `lostcancel` check
that flags paths where the cancel func escapes uncalled.

## Honoring cancellation in a worker

Context is *cooperative*. Canceling does not kill goroutines — nothing in Go
kills goroutines. It closes a channel, and code that never looks at that
channel runs to completion anyway. Honoring cancellation means putting
`ctx.Done()` into your `select`s:

```go
func worker(ctx context.Context, jobs <-chan job, results chan<- result) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case j, ok := <-jobs:
			if !ok {
				return nil // jobs closed: natural end, no error
			}
			select {
			case results <- process(j):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
```

Read the shape carefully — it is this lesson's exercise in miniature:

- The outer `select` races "a job arrived" against "we were told to stop."
  Without it, a canceled worker would block on `<-jobs` forever: a goroutine
  leak the race detector cannot see, because nothing races — it just never
  ends.
- The *send* is guarded too. If nobody is draining `results` anymore
  (perhaps the consumer was the one canceled), an unguarded `results <- r`
  blocks forever. Cancellation must cover every blocking operation, sends
  included.
- On cancellation the worker returns `ctx.Err()` — not `nil`, not an
  invented error. That is the convention the whole ecosystem relies on, and
  the next section shows why.
- When both a job and cancellation are ready, `select` picks randomly
  (Channels lesson). That's acceptable: cancellation is "stop soon," not
  "stop between these two instructions." Don't try to out-engineer it.

## Canceled vs DeadlineExceeded

Once `Done` is closed, `ctx.Err()` returns one of exactly two sentinel
errors, forever:

- `context.Canceled` — someone called `cancel()`. Deliberate: the caller
  gave up, the client disconnected, a sibling already produced the answer.
- `context.DeadlineExceeded` — the clock ran out.

The distinction is operationally real. A deadline expiring is *transient* —
the service was slow, and a retry with a fresh budget may well succeed. A
cancellation is a *decision* — retrying work that somebody explicitly called
off wastes resources at best and duplicates side effects at worst. Retry
logic, metrics, and log severity all branch on which one you got.

By the time the error reaches you it has usually been wrapped — remember
S1's errors lesson: `fmt.Errorf("query users: %w", err)` at every layer. So
you compare with `errors.Is`, never `==`:

```go
switch {
case errors.Is(err, context.DeadlineExceeded):
	// transient: back off and retry
case errors.Is(err, context.Canceled):
	// deliberate: clean up and stop
}
```

This is the payoff of the `return ctx.Err()` convention above: because every
well-behaved function propagates the sentinel (wrapped or not) instead of
swallowing it, the caller at the top can still tell *why* the work stopped.

## First parameter, never stored

The signature convention is absolute:

```go
func FetchUser(ctx context.Context, id string) (*User, error)
```

`ctx` is the first parameter, named `ctx`, on every function in the call
chain that does I/O, blocks, or starts goroutines. Not a global, not a
field — a parameter. The reasoning: a context's lifetime is the lifetime of
*one call tree*. A struct's lifetime is however long the object lives. Store
a request's context in a struct that serves many requests and the mismatch
bites immediately: request 1's context cancels, and requests 2 through N —
which happen to use the same object — die with it. Passing `ctx` explicitly
keeps each call tied to the caller that is actually waiting on it, and makes
the flow of cancellation readable at every call site.

(You'll meet the codified exception when you get to HTTP: `http.Request`
carries its context internally because a request *is* one call tree — and
even there you access it through `r.Context()`, never a field.)

The same logic says: don't pass `nil` for "no context" — pass
`context.Background()` and let the signature stay honest. And functions
receiving a context should assume it can be canceled at any moment, even if
today's only caller passes `Background()`.

## Values: sparingly, and typed

`context.Value` is the method everyone abuses. Its legitimate cargo is
**request-scoped metadata** — data that describes *this request* and rides
along for observability and auth: a request ID, a trace span, the
authenticated user. The test: could the function do its job with the value
missing? A request ID missing means poorer logs — fine, that's metadata. A
database handle missing means the function cannot work — that's a
*dependency*, and it belongs in a parameter or a struct field where the
compiler can see it. Smuggling parameters through `Value` gives you
stringly-typed, invisible, untyped-at-compile-time APIs — everything Go's
type system exists to prevent.

When you do carry a value, the idiom guards against collisions. Keys are
compared with `==`, and every package on the planet might use the string
`"id"` — so you never use a raw string. You define an *unexported* key type
and expose typed accessors:

```go
type requestIDKey struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func RequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey{}).(string)
	return id, ok
}
```

No other package can construct your unexported key, so no other package can
read or clobber your value — collisions are structurally impossible. The
`, ok` type assertion (Type Assertions lesson) keeps a missing value from
panicking. Values flow *down* the tree only: a child sees its ancestors'
values, never its siblings' or children's.

## Exercise

Open [`exercise/`](exercise/) — a Go module with package `ctxkit`: the
context-handling core of a small job-processing client. `ctxkit.go` has four
work sites marked `TODO`; `ctxkit_test.go` is the specification. The tests
coordinate goroutines with channels only — no sleeps — so read them as
worked examples of deterministic concurrent testing.

Acceptance criteria:

1. `Await(ctx, ch)` returns the first value to arrive on `ch`, or — if the
   context is done first — the zero value and `ctx.Err()`.
2. `Square(ctx, jobs, results)` sends `j*j` on `results` for each job, in
   order. When `jobs` is closed it returns `nil`; when the context is done
   it stops promptly — even mid-send — and returns `ctx.Err()`.
3. `Retryable(err)` reports whether the error is worth retrying: `true` for
   a deadline exceeded (transient), `false` for a cancellation (deliberate),
   `nil`, or any other error — and it must see through wrapped errors.
4. `WithRequestID` / `RequestID` round-trip an ID through a context, survive
   derived contexts, report `ok == false` when absent, and are
   collision-proof: a value stored by another package under a look-alike key
   must be invisible to `RequestID`.
5. `go test -race ./...` passes and the code is `gofmt`-clean.

Run the tests from inside `exercise/`, race detector on — from here to the
end of the stage, `-race` is part of the definition of done:

```sh
cd exercise
go test -race ./...
```

They fail before you write code — make them green. Then reread `Square` and
answer for yourself: which *two* blocking operations does cancellation have
to cover, and what would leak if you guarded only the first?

## Further reading

- [pkg.go.dev/context](https://pkg.go.dev/context) — the package docs; the
  opening conventions paragraph is the style guide everyone quotes.
- [Go blog — Go Concurrency Patterns: Context](https://go.dev/blog/context)
  — the original design rationale, worked through a search-server example.
- [Go blog — Contexts and structs](https://go.dev/blog/context-and-structs)
  — the official case for context-as-parameter, including the `http.Request`
  exception.
