# Tutor notes — HTTP Servers

## Where the learner is

First lesson of S5, and the first time they own the server side of a
connection. They arrive with everything they need and no practice using it
together: interfaces and embedding (S3), closures, contexts including
`context.WithValue` with a private key type (S3), the full concurrency arc
under `-race` (S3), and HTTP *clients* tested with `httptest` (S4). This
lesson's job is to make them see that `http.Handler` is the only abstraction
in the room — router, middleware, endpoints and server all speak it — and
that a production server is defined as much by its timeouts and its shutdown
as by its routes.

Expect the routing and handler work to go quickly and `Chain`, `statusWriter`
and `Run` to be where the time goes. Those three are the lesson.

## Common misconceptions

- **"`http.Handler` and `http.HandlerFunc` are two kinds of handler."** There
  is one interface and one adapter. Ask them to write `HandlerFunc`'s
  `ServeHTTP` from memory; if they can, the confusion is gone for good.
- **Registering `"/"` for the home page.** `/` is a subtree root that matches
  every path, so `/nope` returns the home page and 404 never happens.
  `TestRoutingEdges` catches it; the fix is `GET /{$}`.
- **Writing their own 405 handling.** With method prefixes the mux does it,
  including the `Allow` header. Similarly, they do not need to check
  `r.Method` inside a handler registered as `"GET /health"`.
- **`Chain` looping forwards.** Produces `B(A(h))` — the reverse of the
  documented contract. `TestChainOrder` names the expected string, so the
  failure is legible; the insight is that *applied last* means *outermost*.
- **Setting headers after `WriteHeader`.** Silently ignored. If they report
  "my header disappears", check the ordering: headers, then status, then
  body.
- **`statusWriter` initialized to zero.** Then a handler that never calls
  `WriteHeader` logs `status: 0`. The implicit-200 rule is the point of that
  subtest.
- **Mutating the request.** `r.Header.Set(...)` or `*r = *r.WithContext(ctx)`
  instead of passing the copy to `next`. It usually works and it is still
  wrong: the caller still holds that pointer. The dedicated subtest exists
  because this habit bites hardest later, in middleware they did not write.
- **Believing `WriteTimeout` bounds a handler's work.** It is a connection
  deadline, not a per-request budget, and it does not stop a running
  goroutine — the handler keeps burning CPU after the client is gone. The
  per-request tool is `r.Context()`.
- **Treating `http.ErrServerClosed` as a failure.** Then every clean shutdown
  exits non-zero and their deploys look broken.
- **Calling `Shutdown` with the same context that was just cancelled.** The
  grace period is instantly expired, so in-flight requests are cut off — the
  shutdown context must be a fresh one with its own deadline.
- **Waiting only on `ctx.Done()` in `Run`.** If `Serve` dies on its own the
  process hangs forever, healthy-looking and serving nothing.

## Grilling points

- "Chain is `Chain(mux, RequestID, Logging, Recover)`. Trace a request that
  panics: which code runs, in what order, and what does the client get?"
  (Then: "Now move `Recover` to the front — what changes, and which is
  right?" Leave it innermost. Inside `Logging`, the panic has already become
  a 500 by the time the logger reads the status, so the request that hurt
  most still produces exactly one log line and it says 500; hoist `Recover`
  outside and panicking requests vanish from your traffic view. The cost is
  real — a panic in `Logging` itself now escapes — which is exactly why
  logging and metrics middleware must stay boring. The capstone chain makes
  this explicit.)
- "Your log lines have an empty `request_id`. Nothing failed, no test broke.
  What did you get wrong?" (Chain order — `RequestID` must be outside
  `Logging`.)
- "You wrote `statusWriter` by embedding `http.ResponseWriter`. A teammate's
  streaming handler calls `w.(http.Flusher).Flush()` and it panics after your
  middleware ships. Why, and what is the one-method fix?" (`Unwrap` +
  `http.NewResponseController`.)
- "An attacker opens 10,000 connections and sends one header byte per minute.
  Which timeout saves you, and what is the cost of setting it too low for a
  service that accepts large uploads?" (`ReadTimeout`; use
  `ReadHeaderTimeout` when bodies are legitimately slow.)
- "Kubernetes sends SIGTERM. Walk me through every step from signal to
  process exit, and tell me what a client with a request in flight sees."
- "Why does `Run` take a `net.Listener` instead of calling
  `ListenAndServe`?" (Testability; same injection instinct as injectable
  clocks — the tests bind `127.0.0.1:0` and never touch a real port.)
- Stretch: "`Recover` swallows every panic. Look up `http.ErrAbortHandler` —
  which one should it *not* swallow, and why?"
- Stretch: "Two goroutines serve two requests through the same `Logging`
  middleware. Which values are shared between them and which are per-request?
  Why is `statusWriter` allocated inside the handler function and not
  outside?"

## Grading rubric

- **A** — All tests green under `-race`; `Chain` loops backwards with a
  one-line reason; `RequestID` passes a copy via `r.WithContext` and reuses
  an inbound id; `statusWriter` seeded to 200; `Run` selects on both the
  serve error and `ctx.Done()`, shuts down with a *fresh* deadline context,
  and drains the serve goroutine; can explain each timeout's failure mode and
  trace the panic path through the chain.
- **B** — Tests green with rough edges: `Chain` implemented by reversing a
  copy of the slice, `Recover` swallowing the panic without logging it,
  shutdown context derived from the cancelled one but tests passing by luck,
  or hazy on why `ErrServerClosed` is success. Explanation broadly right.
- **C** — Green only after the remediation ladder, or they cannot say what
  `HandlerFunc` is, or middleware order is cargo cult. Time-box remediation,
  re-grill, then decide.
- **Fail** — Tests failing, race findings, or `Run` copied without being able
  to name what each `select` case is for. Remediate; this code is the spine
  of the rest of the stage.

## Remediation ladder

1. "Run `go test -race ./... 2>&1 | head -30` and read the first failure
   aloud. Which file's contract does it quote — handlers, middleware, or
   server?"
2. Routing stuck: "Print the pattern strings you registered next to the
   request lines in the test. Which one is the mux matching for `/nope`, and
   what does a pattern ending in `/` mean?" Chain stuck: "Write
   `A(B(handler))` on paper, then write the loop that produces it from
   `[A, B]`. Which end of the slice do you touch first?"
3. Middleware stuck: "A middleware is a function that returns
   `http.HandlerFunc(func(w, r) { ... next.ServeHTTP(w, r) ... })`. Type that
   skeleton for `Recover`. Now: what has to be `defer`red, and where?"
   Logging stuck: "Where does `sw` have to be created — once per middleware,
   or once per request? Then: what number does `status` start at?"
4. `Run` stuck: "Three moving parts: a goroutine running `Serve`, a channel
   with its error, and `ctx.Done()`. Write the `select` with two cases and
   say out loud what each one means. What must happen *after* the select in
   each case?" Only then sketch the shape — `shutdownCtx, cancel :=
   context.WithTimeout(context.Background(), shutdownGrace)` — and let them
   type it, including why the parent is `Background` and not `ctx`.

## After passing

Preview: "Next: REST services. Same server, but the handlers stop being the
whole program — you'll split routing, domain rules, and storage into layers,
and map domain errors to status codes in exactly one place."
