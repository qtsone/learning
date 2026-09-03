# Tutor notes — gRPC

## Where the learner is

Third lesson of S5. They have just built HTTP servers and REST services with
the 1.22 mux, middleware chains, and httptest-driven tests, and they know
contexts, errors.Is/As, and the full S3 concurrency arc. This is their first
contract-first system and their first generated code — expect friction from
"where did all these types come from" more than from the Go itself. The
generated code in `taskboardpb/` is committed; they never need protoc/buf to
pass, only to experiment.

## Common misconceptions

- **"Field names matter on the wire"** — only numbers do. If they can't say
  why renaming is safe but renumbering is corruption, replay the encoding
  model: messages are (number, value) pairs, names are compile-time only.
- **Returning plain errors from handlers** — `errors.New`/`fmt.Errorf`
  crosses the wire as `codes.Unknown`. Handlers must speak `status.Error`
  with a deliberate code; it's the same discipline as choosing an HTTP status
  in the REST lesson.
- **Checking errors with string matching or `==`** — the wire strips wrapping;
  `status.Code(err)` / `status.FromError` on the client, and their own
  wrapper translates to `ErrNotFound` so *callers* are back in errors.Is
  land. If they compare `err == ErrNotFound` in their head, remind them the
  test demands `errors.Is` because the solution wraps with `%w`.
- **Wrapping the context in `WithTimeout` unconditionally** — the most common
  client bug; `TestFetchTask_KeepsCallerDeadline` catches it. The insight:
  `WithTimeout` can only shorten, so the default must be applied *only when
  no deadline exists*.
- **Believing the unary interceptor guards streams too** — it doesn't;
  interceptors are per-kind. The exercise's `ListTasks` is deliberately
  unauthenticated; ask them whether that's a hole and how they'd plug it
  (`grpc.StreamInterceptor` + a stream interceptor).
- **Forgetting `defer cancel()`** after `WithTimeout` — `go vet` flags it;
  a leaked context is the kind of slow leak S5 keeps returning to.
- **Editing generated `.pb.go` files** — anything they hand-edit there is
  overwritten by the next `buf generate`; changes go in the `.proto`.

## Grilling points

- "Your service is live. A teammate renames `title` to `summary` and a field
  `priority = 3` is added, while an old client keeps running. What breaks,
  what doesn't, and why?" (Nothing breaks; then: what if they'd *reused*
  number 3 from a deleted field?)
- "Walk me through what the server does when a client's deadline expires
  mid-`ListTasks`. Where does the handler find out?" (`stream.Send` errors /
  stream context cancels — connect to S3's cancellation habits.)
- "Why does `FetchTask` translate `codes.NotFound` into `ErrNotFound` instead
  of letting callers use `status.Code`?" (Callers shouldn't import grpc;
  transport-agnostic call sites; same boundary-translation they did for
  database errors in S4.)
- "Compare `AuthUnaryInterceptor` line-by-line with the auth middleware from
  your HTTP lessons. What's the analog of `next.ServeHTTP`? Of the request
  headers?" (`handler(ctx, req)`; metadata.)
- "When would you *not* stream — what does `ListTasks` cost as a stream that
  a repeated field in a unary response wouldn't?" (Per-message framing,
  client Recv loop complexity; for small bounded sets unary is simpler.)
- "Your capstone service needs a public API for a mobile app and calls an
  internal pricing service 200 times per request. Which boundary gets gRPC,
  which gets REST, and why?"

## Grading rubric

- **A** — All tests green under `-race`; handler errors use precise codes;
  `FetchTask` checks for an existing deadline before applying the default and
  defers cancel; NotFound wrapped with `%w`; interceptor short-circuits
  without calling the handler; can explain field-number compatibility and
  argue gRPC-vs-REST for a concrete boundary both ways.
- **B** — Tests green but with rough edges: `status.Error(codes.NotFound, ...)`
  with unhelpful messages, sentinel returned unwrapped (`return nil,
  ErrNotFound` — passes errors.Is but loses the id), or shaky on *why* the
  caller-deadline test exists; explanation of streaming trade-offs mostly
  there.
- **C** — Tests pass only after the remediation ladder, or they can't explain
  what the interceptor replaced from their HTTP middleware, or they think
  generated code is hand-maintained. Time-box remediation; re-grill before
  passing.
- **Fail** — Tests failing, `-race` findings, or the deadline/metadata logic
  pasted without being able to trace what the server sees. Remediate from the
  bufconn harness upward.

## Remediation ladder

1. "Run `go test -race ./... 2>&1 | head -30` and read the first failure
   aloud. Which file's contract does it quote — server, auth, or client?"
2. Server stuck: "Forget gRPC. Write plain Go: given `s.tasks` and an id,
   return the task or 'not found'. Now, which `codes` value does each return
   path map to?" Streaming stuck: "What does `stream.Send` replace from the
   unary signature — where did the return value go?"
3. Interceptor stuck: "Print the metadata you get from
   `metadata.FromIncomingContext` inside the interceptor and run the auth
   test. What key arrives, and what exact string do you need it to equal?"
4. Client stuck: "Three requirements, three lines of the contract comment.
   Do them in order: append metadata → check `ctx.Deadline()` before
   `WithTimeout` → `status.Code(err) == codes.NotFound`. Which one does the
   failing test name?" Only then sketch the shape (`if _, ok :=
   ctx.Deadline(); !ok { ... }`) and let them type it.

## After passing

Preview: "Next: databases in production — pooling, migrations, and
transactions. Your capstone service will sit exactly where this lesson's
server does, but with SQLite behind it instead of a slice."
