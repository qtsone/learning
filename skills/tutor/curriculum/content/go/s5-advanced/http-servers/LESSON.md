# HTTP Servers

> `go.advanced.http-servers` · ~3-4h · Stage: Advanced Go

## Objectives

By the end of this lesson you can:

- Implement an HTTP server with `net/http` using `http.Handler` and
  `http.HandlerFunc`, and explain the difference between the two.
- Use Go 1.22+ ServeMux patterns (method prefixes and path wildcards) to
  route requests and extract path values.
- Implement composable middleware (logging, recovery, request ID) as
  `func(http.Handler) http.Handler` and explain the order in which a
  middleware chain executes.
- Configure server timeouts (`ReadTimeout`, `WriteTimeout`, `IdleTimeout`)
  and explain what failure mode each one prevents.
- Implement graceful shutdown with `server.Shutdown` and a signal-aware
  context.

## The other side of the socket

In S4 you wrote HTTP clients and learned the rules of talking across a
network: timeouts everywhere, bodies closed, statuses treated as data. This
stage flips the connection around. Now you are the machine that must answer —
to clients you don't control, some slow, some broken, some hostile, all of
them at once.

Go's answer lives in the standard library. The `net/http` server is not a
toy you outgrow and replace with a "real" framework — it *is* the production
server behind a large share of the Go services running today. Frameworks
mostly add routing sugar and middleware conventions on top; by the end of
this lesson you will have built both yourself and will know exactly what you
would be importing.

One fact shapes everything that follows: **the server handles each request
on its own goroutine**. Two requests arriving together are two goroutines
running your code at the same time. Your whole S3 concurrency arc applies:
shared state needs synchronization, and the race detector referees. That is
why every exercise in this stage runs under `go test -race`.

## One interface: http.Handler

The entire server API hangs off a single, one-method interface:

```go
type Handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
}
```

Read it as a contract: "given a parsed request `r`, write the response into
`w`." Everything — your endpoints, your router, your middleware, the server
itself — speaks this interface and nothing else. That uniformity is the
lesson's secret theme: because everything is a `Handler`, everything
composes.

A struct makes a natural handler when state must travel with it — a version
string, a database handle, a logger:

```go
type versionHandler struct {
	version string
}

func (h versionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, h.version)
}
```

But most endpoints need no struct, and writing a type per endpoint gets old
fast. Enter `http.HandlerFunc`:

```go
type HandlerFunc func(http.ResponseWriter, *http.Request)

func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}
```

Look closely: it is a *named function type with a method* — a trick you met
in S3. `ServeHTTP` just calls the function itself. So `HandlerFunc` is an
**adapter**: it converts any ordinary function with the right signature into
something that satisfies `http.Handler`. There is no "second kind of
handler" — there is one interface, and a convenience for reaching it from a
plain function (closures included, which is how handlers capture
dependencies without a struct).

Two rules about `ResponseWriter` will save you real debugging time:

- Order is fixed: set headers, then `WriteHeader(status)`, then write the
  body. Headers changed after `WriteHeader` are silently ignored — they have
  already been sent down the wire.
- If you never call `WriteHeader`, the first `Write` sends `200 OK` for you.
  Most success paths simply write the body and let the 200 happen.

## Routing: the Go 1.22 ServeMux

`http.ServeMux` is the standard router: it matches an incoming request
against registered patterns and dispatches to the winning handler. Since Go
1.22 the patterns carry real routing power:

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /health", healthHandler)
mux.HandleFunc("GET /greet/{name}", greetHandler)
mux.Handle("GET /version", versionHandler{version: "1.0.0"})
```

- **Method prefix** — `"GET /health"` matches only GET requests. A POST to a
  path that only registers GET gets `405 Method Not Allowed` from the mux
  automatically, with a correct `Allow` header. You write no code for this.
- **Wildcards** — `{name}` matches exactly one non-empty path segment. Inside
  the handler you read it with `r.PathValue("name")`. A trailing
  `{rest...}` wildcard matches everything to the end of the path.
- **`{$}`** — the pattern `"/"` matches *every* path (it is a subtree root),
  which is almost never what you want for a home page. `"GET /{$}"` matches
  only `/` itself, so unknown paths correctly fall through to `404`.
- **Precedence** — when several patterns match, the most specific one wins;
  registration order is irrelevant. Two patterns that overlap with no winner
  panic at registration, so conflicts surface at startup, not in production.

The mux is itself a `Handler` (of course), so plugging it into a server is
just assignment.

## Middleware: handlers wrapping handlers

Logging, panic recovery, request IDs, auth — none of these belong inside
individual endpoints, and all of them apply to every request. The Go idiom
is a function that takes a handler and returns a new handler wrapped in
extra behavior:

```go
type Middleware func(http.Handler) http.Handler

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ...before next...
		next.ServeHTTP(w, r)
		// ...after next...
	})
}
```

This is the decorator pattern built from parts you already own: a closure
capturing `next`, adapted through `HandlerFunc`. Middleware that needs
configuration takes it and *returns* a `Middleware` — `Logging(logger)` in
the exercise.

Wrapping nests like an onion. `A(B(handler))` means: A's "before" code, then
B's, then the handler, then B's "after" code, then A's — the first wrapper
applied last is the outermost layer, and the outermost layer runs first and
finishes last. A `Chain` helper keeps the reading order sane:

```go
h := Chain(mux, RequestID, Logging(logger), Recover)
```

Contract: the first middleware listed is outermost. To build that, loop over
the slice **backwards**, wrapping as you go — the last one listed hugs the
handler tightest.

Middleware also needs a way to hand facts *down* the chain. Headers are the
wire's vocabulary, not your program's, and you must not mutate the
`*http.Request` you were handed — the caller still holds that pointer. The
right channel is the one S3 built for exactly this job: the request context.
`r.WithContext(ctx)` returns a shallow *copy* of the request carrying a new
context, and you pass the copy to `next`:

```go
ctx := WithRequestID(r.Context(), id)
next.ServeHTTP(w, r.WithContext(ctx))
```

`WithRequestID`/`RequestIDFrom` hide an unexported key type, exactly as in
your context lesson. Now ordering stops being trivia: `Chain(mux, RequestID,
Logging(logger))` puts the ID on the context *before* the logger reads it, so
every log line carries a `request_id`. Swap the two and the field comes out
empty — same code, same tests on each piece, silently worse telemetry.

One practical wrinkle: a logging middleware wants the response status, but
`ResponseWriter` has no `Status()` method — the code flows one way, out. The
fix is interface interception, using embedding from S3: wrap the writer in
your own type, override `WriteHeader` to record the code before delegating,
and hand *that* to `next`. Remember the implicit-200 rule: if the handler
never calls `WriteHeader`, your wrapper must report 200.

Wrapping a `ResponseWriter` has a cost worth knowing about: the real writer
may also implement `http.Flusher` or `http.Hijacker`, and a wrapper that only
embeds the interface hides those from the handler. Since Go 1.20 the fix is
mechanical — give your wrapper an `Unwrap() http.ResponseWriter` method, and
`http.NewResponseController(w)` finds its way back to the original. One
method, no interface archaeology.

## Timeouts: the server side of the bargain

The S4 client lesson called a client without timeouts a scheduled production
incident. The server default is the same trap: every timeout on
`http.Server` is zero, and zero means *wait forever*. A server that waits
forever is a server whose connections — and the goroutines and file
descriptors behind them — are handed out to whoever wants to hold them:

```go
srv := &http.Server{
	Addr:         addr,
	Handler:      h,
	ReadTimeout:  5 * time.Second,
	WriteTimeout: 10 * time.Second,
	IdleTimeout:  120 * time.Second,
}
```

Each one closes a specific hole:

- **`ReadTimeout`** caps reading the whole request, headers and body. It is
  your defense against slow senders — including the deliberate kind
  (slowloris attacks drip one header byte at a time to pin thousands of
  connections). Its little sibling `ReadHeaderTimeout` caps only the header
  phase, useful when you must accept legitimately slow uploads.
- **`WriteTimeout`** caps producing and writing the response. It bounds
  the damage from a hung handler and from clients that read one byte per
  second: past the deadline, the connection is cut.
- **`IdleTimeout`** caps how long a keep-alive connection may sit between
  requests. Keep-alive is a performance win (your client lesson's connection
  pool is the other end of it), but without an idle cap every client that
  ever connected can park a socket on you until you run out of descriptors.

These are blunt, whole-connection backstops, not per-request budgets — the
per-request tool is the request's context, which you will wire into handlers
in the coming lessons. Set the backstops anyway. Always.

## Graceful shutdown

Your service will be restarted constantly — deploys, scale-downs, node
drains. The difference between a clean deploy and a spray of user-facing
errors is what your process does with the requests *in flight* when it is
told to stop.

`srv.Shutdown(ctx)` does exactly the right things, in order: stop accepting
new connections, close idle ones, then wait for active requests to finish —
up to the deadline on the `ctx` you pass. Meanwhile the blocked
`srv.Serve(ln)` call returns `http.ErrServerClosed`. Read that error's name
again: it is not a failure, it is the confirmation that shutdown happened —
"the server is closed, as you asked." Treat it as success; treat every
other `Serve` error as real.

Who decides it's time to stop? The operating system, via signals: Ctrl-C
sends `os.Interrupt`, and process managers send `SIGTERM` before killing.
Since you already think in contexts, Go hands you the bridge:

```go
ctx, stop := signal.NotifyContext(context.Background(),
	os.Interrupt, syscall.SIGTERM)
defer stop()
```

`ctx` is cancelled the moment a signal lands, and cancellation of a context
is something the whole program already knows how to react to. The run loop
becomes: serve in a goroutine, wait for `ctx.Done()` (or for `Serve` to fail
on its own), call `Shutdown` with a grace deadline, and drain the `Serve`
error.

One last design move, and it is the same move as injectable time in S4:
write `Run(ctx, srv, ln)` to take a `net.Listener` instead of calling
`ListenAndServe`. Production passes a listener on the real port; tests pass
one on `127.0.0.1:0` — a free port the OS picks, no network beyond loopback,
no collisions. Separating *listen* from *serve* is what makes a server
testable at all.

## Exercise

Open [`exercise/`](exercise/) — a small "greeter" service with production
bones: `handlers.go` (routes), `middleware.go` (the chain), `server.go`
(timeouts and lifecycle), and a `main.go` that wires them together.
`handlers_test.go`, `middleware_test.go` and `server_test.go` are the
specification — read them first.

Acceptance criteria:

1. `NewMux(version)` routes with 1.22 patterns: `GET /{$}` → `greeter
   service`, `GET /health` → `ok`, `GET /version` → the version string
   (served by the `versionHandler` struct implementing `http.Handler`
   directly), and `GET /greet/{name}` → `Hello, <name>!` via
   `r.PathValue`.
2. Unknown paths get 404 (register the home page as `{$}`, not `/`); a
   wrong method gets 405 — both from the mux, no extra code.
3. `Chain(h, mws...)` applies middleware so the first one listed is
   outermost: `Chain(h, A, B)` runs A-in, B-in, handler, B-out, A-out.
4. `RequestID` reuses the incoming `X-Request-ID` header or generates one
   (`math/rand/v2`), puts it on the request **context** so inner handlers
   read it with `RequestIDFrom`, echoes it on the response header, and gives
   distinct requests distinct IDs. It must not modify the request it was
   handed — pass a copy made with `r.WithContext`.
5. `Logging(logger)` logs one line per request through the given
   `*slog.Logger` with `method`, `path`, `status`, and `request_id`
   attributes — `status` is 200 when the handler never calls `WriteHeader`
   (that's your `statusWriter`), and `request_id` is non-empty whenever
   `RequestID` sits outside `Logging` in the chain.
6. `Recover` turns a handler panic into a `500` response; the panic must not
   escape.
7. `NewServer(addr, h)` sets `Addr`, `Handler`, `ReadTimeout` 5s,
   `WriteTimeout` 10s, `IdleTimeout` 120s.
8. `Run(ctx, srv, ln)` serves on `ln`; when `ctx` is cancelled it stops
   accepting new connections, lets in-flight requests finish, and returns
   `nil` on a clean shutdown (`http.ErrServerClosed` is success). If `Serve`
   fails on its own — a dead listener, say — `Run` returns that error instead
   of blocking on a cancellation that will never come.
9. `main` uses a signal-aware context. Manual check: `go run .`, curl a few
   endpoints, hit Ctrl-C — you should see `shutdown complete`, not a killed
   process.
10. `go test -race ./...` passes and the code is `gofmt`-clean.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test -race ./...
```

They fail on the starter. A sane order: routes first, then `Chain`, then the
middlewares one by one, then `NewServer`, then `Run` — each has its own test
you can target with `go test -race -run TestChainOrder` and friends. The
`Run` tests bind `127.0.0.1:0`, so they exercise a real server on a real
socket without ever leaving your machine or picking a port that could
already be taken.

## Further reading

- [Go blog — Routing Enhancements for Go 1.22](https://go.dev/blog/routing-enhancements)
- [pkg.go.dev — net/http (Handler, ServeMux, Server)](https://pkg.go.dev/net/http)
- [Cloudflare — The complete guide to Go net/http timeouts](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/)
- [pkg.go.dev — net/http Server.Shutdown](https://pkg.go.dev/net/http#Server.Shutdown)
