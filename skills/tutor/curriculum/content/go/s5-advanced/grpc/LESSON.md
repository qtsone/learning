# gRPC

> `go.advanced.grpc` · ~3-5h · Stage: Advanced Go

## Objectives

By the end of this lesson you can:

- Define a service in a `.proto` file and generate Go client and server code
  with protoc/buf, explaining what protobuf field numbers guarantee for
  compatibility.
- Implement a unary RPC server and client, propagating deadlines and handling
  status codes with the `status`/`codes` packages.
- Implement a server-streaming RPC and explain when streaming beats repeated
  unary calls.
- Implement a server interceptor for cross-cutting concerns (logging or auth)
  and compare it to HTTP middleware.
- Choose between gRPC and REST for a given service boundary and justify the
  choice in terms of contract, performance, and client ecosystem.

## Why another way for services to talk

In the last two lessons you built HTTP services by hand: you chose paths and
verbs, decoded JSON, validated it, mapped errors to status codes. The
*contract* — what requests exist, what fields they carry — lived in your head
and your docs. A client team has to re-implement all of it and hope they read
the docs the same way you wrote them.

gRPC inverts this. You write the contract first, in a `.proto` file, and a
code generator produces the client and the server halves for you — typed
structs, typed stubs, serialization included. Calling a remote service looks
like calling a Go method (RPC = *remote procedure call*); the paths, verbs,
and JSON plumbing disappear. On the wire it's Protocol Buffers (a compact
binary format) over HTTP/2, which multiplexes many concurrent calls over one
connection. The trade: you can no longer poke it with `curl` and read the
response with your eyes.

## The contract: Protocol Buffers

Here is this lesson's contract, `taskboardpb/taskboard.proto`, abridged:

```proto
syntax = "proto3";

package taskboard.v1;

option go_package = "tutor.local/grpc-tasks/taskboardpb";

service TaskService {
  rpc GetTask(GetTaskRequest) returns (Task);
  rpc ListTasks(ListTasksRequest) returns (stream Task);
}

message Task {
  string id = 1;
  string title = 2;
  Status status = 3;
}
```

A `service` declares RPC methods; a `message` declares a data shape. The
`= 1`, `= 2`, `= 3` are **field numbers**, and they are the heart of
protobuf's compatibility story: on the wire, a message is a sequence of
`(field number, value)` pairs — **names never leave your source code**. That
buys you precise evolution rules:

- Renaming a field is invisible to the wire. Safe.
- Adding a field with a *new* number is safe both ways: old readers skip
  unknown numbers; new readers see the zero value from old writers.
- Changing or reusing a number is silent data corruption: the same bytes
  decode into a different field. Never do it — when you delete a field, mark
  its number `reserved` so nobody can reuse it.

This is the same schema-evolution discipline you applied to JSON APIs in the
REST lesson, but enforced by the format instead of by code review. One more
convention: every enum's zero value is `..._UNSPECIFIED`, because proto3
cannot distinguish "not set" from "set to zero" — the zero value must mean
"nothing chosen", never a real state.

## Generating the Go code

The generator is `protoc` (or `buf`, a friendlier wrapper) plus two Go
plugins:

```sh
# Pin the generators to the versions this module already depends on. With
# @latest you can generate code that needs a newer runtime than go.mod allows.
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.5
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
buf generate    # run in exercise/ — config in buf.gen.yaml
```

The generated files are **committed** in `taskboardpb/` — you only need the
tools above if you change the `.proto`. Skim both files once; you will use:

- `taskboardpb.Task` and friends — message structs with nil-safe getters
  (`task.GetId()` works even when `task` is nil; prefer getters).
- `TaskServiceClient` — the typed client stub, built from a connection with
  `NewTaskServiceClient(conn)`.
- `TaskServiceServer` — the interface your server must satisfy, plus
  `UnimplementedTaskServiceServer`, which you embed so that adding methods to
  the proto later doesn't break your build (they return `Unimplemented`
  instead), and `RegisterTaskServiceServer` to hook it into a `grpc.Server`.

## Unary RPCs and status codes

`GetTask` is a **unary** RPC: one request in, one response out. The server
side is an ordinary method:

```go
func (s *Server) GetTask(ctx context.Context, req *taskboardpb.GetTaskRequest) (*taskboardpb.Task, error)
```

But the error you return crosses a process boundary, so a plain
`errors.New("not found")` collapses to the catch-all code `Unknown` on the
client. gRPC's error vocabulary is the `codes` package — a fixed set that
plays the role HTTP status codes played in your REST lesson (`InvalidArgument`
≈ 400, `Unauthenticated` ≈ 401, `NotFound` ≈ 404, `Internal` ≈ 500) — and you
build errors with the `status` package:

```go
return nil, status.Errorf(codes.NotFound, "no task with id %q", id)
```

On the client, `status.Code(err)` recovers the code (`status.FromError` gives
the whole thing). Just as your REST handlers translated database errors into
HTTP statuses at the boundary, a good gRPC *client* translates codes back
into ordinary Go errors — the exercise has you map `codes.NotFound` to a
sentinel `ErrNotFound` so callers write `errors.Is(err, ErrNotFound)` and
never import `grpc` themselves.

## Deadlines travel with the call

You learned in the HTTP-client lesson that every outbound request needs a
timeout. gRPC builds this into the protocol: the deadline on the client's
`context` is transmitted with the call, and the server's handler `ctx`
carries (approximately) the same deadline — across every hop, the whole chain
shares one time budget, and a server can stop working on a call nobody is
waiting for anymore.

The etiquette your exercise client enforces:

- If the caller's context has no deadline, apply a sensible default
  (`context.WithTimeout` — and `defer cancel()`).
- If the caller already set one, **leave it alone**: `WithTimeout` can only
  shorten a deadline, so wrapping unconditionally silently overrides the
  caller's budget.

## Server streaming

`ListTasks` returns `stream Task`: one request in, many responses out, in
order, over the same call. The server gets a stream instead of a return
value:

```go
func (s *Server) ListTasks(req *taskboardpb.ListTasksRequest, stream taskboardpb.TaskService_ListTasksServer) error {
    // stream.Send(task) for each result; return nil when done.
}
```

The client calls `Recv()` in a loop until it returns `io.EOF`. Streaming
beats N unary calls (or one giant response) when the result set is large or
unbounded: the client processes items as they arrive instead of buffering
everything, memory stays flat on both sides, and the whole transfer is *one*
call — one deadline, one auth check, one cancellation. When a client
disconnects, `Send` starts failing, so return its error and stop producing.
For a handful of small results, a repeated-field unary response is simpler;
streaming earns its complexity with volume. (Client-streaming and
bidirectional RPCs also exist — same machinery, both directions.)

## Interceptors: middleware by another name

In the HTTP-servers lesson you wrote middleware as
`func(next http.Handler) http.Handler`. gRPC's equivalent is the
**interceptor** — a function that runs around every handler:

```go
func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error)
```

Call `handler(ctx, req)` to proceed, or return an error to short-circuit —
exactly the decision your auth middleware made. Differences worth noticing:
an interceptor is registered once on the `grpc.Server`
(`grpc.UnaryInterceptor(...)`) and sees *every* method, with `info` telling
it which; it sees decoded request values, not byte streams; and it is
per-kind — a unary interceptor never runs for streaming RPCs, which need a
separate `grpc.StreamServerInterceptor`. The exercise guards unary calls
only; a production server registers both.

Where does a token live if there are no HTTP headers in your code? In
**metadata** — gRPC's key/value pairs attached to a call (carried as HTTP/2
headers underneath). The client attaches with
`metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer …")`; the
server reads with `metadata.FromIncomingContext(ctx)`.

## Testing without a network: bufconn

Your REST tests used `httptest` so nothing touched a real port. gRPC's
equivalent is `google.golang.org/grpc/test/bufconn`: an in-memory listener.
The test harness in `grpc_test.go` (`startServer`) serves your real
`grpc.Server` on a bufconn and dials it with `grpc.NewClient` and a custom
dialer — full stack, zero network. Read it before you start; you'll reuse
this pattern in the capstone.

## gRPC or REST?

The engineering question is not "which is better" but "what does this
boundary need":

- **Contract.** Many services, many teams, evolving message shapes → gRPC's
  generated, versioned contract prevents drift mechanically. A public API
  documented for strangers → REST + JSON is self-describing and universal.
- **Performance.** High-volume internal traffic → binary encoding and HTTP/2
  multiplexing measurably cut latency and bytes. A form submission → JSON
  parsing is nowhere near your bottleneck.
- **Client ecosystem.** Browsers cannot speak native gRPC (grpc-web needs a
  proxy); `curl` and DevTools speak REST fluently. Polyglot backend fleet →
  generated clients in every language are a gift.

The common production answer is both: REST (or JSON over HTTP) at the public
edge, gRPC between internal services. Your S5 capstone is a REST service; in
a larger system, the services *behind* it would likely speak gRPC.

## Exercise

Open [`exercise/`](exercise/). The contract (`taskboardpb/taskboard.proto`)
and generated code are in place; `grpc_test.go` is the specification. You
implement three files, marked with `TODO`:

- `server.go` — `GetTask` (unary) and `ListTasks` (server-streaming).
- `auth.go` — `AuthUnaryInterceptor`, bearer-token auth as an interceptor.
- `client.go` — `Client.FetchTask`, a client wrapper with auth metadata, a
  default deadline, and error translation.

Acceptance criteria:

1. `GetTask` returns the stored task (id, title, status) for a known id.
2. `GetTask` fails with `codes.InvalidArgument` for an empty id and
   `codes.NotFound` for an unknown one.
3. `ListTasks` streams tasks in insertion order; `STATUS_UNSPECIFIED` means
   no filter, any other status streams only matching tasks.
4. `AuthUnaryInterceptor(token)` rejects calls whose metadata lacks
   `authorization: Bearer <token>` with `codes.Unauthenticated` and passes
   correct calls through untouched.
5. `Client.FetchTask` attaches the bearer token, applies the default timeout
   only when the caller's context has no deadline (never shortening an
   existing one), and returns an error satisfying
   `errors.Is(err, ErrNotFound)` when the server says `NotFound`.
6. `go test -race ./...` passes and the code is `gofmt`-clean.

Run the tests from `exercise/`:

```sh
cd exercise
go test -race ./...
```

They fail until you do the work — read the failure messages; they name the
missing behavior. No test runs the generator, so you do not need
`buf`/`protoc` to pass.

Do this once anyway, ungraded: install the two plugins, add
`string assignee = 4;` to `Task`, run `buf generate`, and read the diff in
`taskboardpb/`. Nothing else breaks — that is the field-number rule holding —
and the regenerated `.pb.go` files are exactly what you would commit next to
the `.proto` so a fresh clone builds with no protoc installed.

## Further reading

- [gRPC — Basics tutorial (Go)](https://grpc.io/docs/languages/go/basics/)
- [Protocol Buffers — proto3 language guide](https://protobuf.dev/programming-guides/proto3/)
- [Protocol Buffers — encoding (why field numbers rule the wire)](https://protobuf.dev/programming-guides/encoding/)
- [gRPC blog — gRPC and deadlines](https://grpc.io/blog/deadlines/)
