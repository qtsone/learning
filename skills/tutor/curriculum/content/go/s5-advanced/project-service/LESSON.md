# Mini-Project: Production Service

> `go.advanced.project-service` · ~8-12h · Stage: Advanced Go

## Objectives

By the end of this lesson you can:

- Build a complete HTTP service that combines routing, layered structure,
  authentication, and a real database with migrations.
- Secure the service with authentication middleware, and explain the chosen
  token strategy and its threat model.
- Achieve meaningful test coverage across unit and integration levels,
  including an httptest-driven API test suite.
- Instrument the service with structured logs, Prometheus-format metrics, and
  health endpoints, and demonstrate graceful shutdown.
- Defend the architecture in review: layer boundaries, error-handling
  strategy, and at least one measured performance decision.

## What a capstone is for

Nothing in this lesson is new. Every technique — the 1.22 mux, middleware
chains, layered packages, migrations, pooled connections, `log/slog`,
histograms, fake clocks, graceful shutdown — arrived in an earlier lesson,
attached to a small example that fit on a screen. That is the honest way to
teach a technique and a dishonest way to represent a service.

Real services are not hard because any one piece is hard. They are hard
because the pieces have to agree: the middleware order has to match what the
logs promise, the error taxonomy has to match the status codes, the
readiness probe has to match what "ready" means for the storage you chose,
and all of it has to survive being restarted mid-request. This lesson is
where you assemble the pieces and find out where they rub.

The deliverable is `taskd`: a small JSON API over a SQLite-backed task list,
built the way you would build something you have to carry a pager for.

## The shape of the service

```
cmd/taskd     composition root: config, wiring, run loop        (provided)
   │
httpapi       HTTP edge: routes, middleware, envelopes, metrics
   │
 task         domain: types, rules, error vocabulary, Store interface
   ▲
sqlitestore   task.Store over modernc.org/sqlite + migrations
```

The arrows are the same ones the REST lesson drew, and they still point at
the domain: `httpapi` imports `task`, `sqlitestore` imports `task`, and
`task` imports neither. Only `main` knows all three exist. Two consequences
you should be able to state in review:

- **`task` has no idea it is on the web.** No `net/http` import, no status
  codes, no JSON decisions beyond struct tags. Its tests need no httptest.
- **`sqlitestore` is replaceable.** It satisfies `task.Store`, an interface
  declared by the consumer. Swapping in Postgres is a new package and one
  line in `main`.

The one addition since the REST lesson: `task.Store` methods all take a
`context.Context`. A request that is cancelled — client hung up, timeout
fired, process shutting down — must be able to stop work in the driver, not
just at the handler. Contexts only work if every layer passes them down.

## main is a composition root

`cmd/taskd/main.go` reads configuration, constructs each layer, hands each
one its dependencies, and starts the run loop. It contains no rules. That is
deliberate: everything in `main` is the code your tests cannot reach, so the
less of it there is, the better. Configuration comes from the environment
(`TASKD_ADDR`, `TASKD_DB`, `TASKD_TOKENS`) and the service refuses to start
without credentials configured — a default token in source is a default token
in production.

## The chain, and why the order is the design

The middleware chain in `main` is short and every position is an argument:

```go
handler := httpapi.Chain(api.Routes(httpapi.Auth(tokens)),
	httpapi.RequestID,             // outermost
	httpapi.AccessLog(logger),
	httpapi.Recover(logger),
	httpapi.Timeout(requestTimeout), // innermost
)
```

- **`RequestID` first** so everything inside it — every log line, every
  panic report — can carry the same correlation id, and so the id is echoed
  on the response before anything else writes a header.
- **`AccessLog` outside `Recover`** so a panicking request still produces
  exactly one access line, and that line says `status=500`. Flip the two and
  panics vanish from your traffic view: the request that hurt the most is
  the one you cannot see.
- **`Timeout` innermost** so it bounds the handler rather than the logging
  around it. `http.TimeoutHandler` runs the handler on its own goroutine
  with a deadline on the request context; when the deadline wins, the client
  gets a 503 and the handler's later writes are dropped. Note what it does
  *not* do: it cannot kill the goroutine. Nothing in Go can. The handler
  keeps running until it notices `ctx.Done()` — which is why every layer
  takes a context.
- **`Auth` is not in this chain at all.** It is applied inside `Routes`, to
  the task endpoints only, because health probes and metric scrapes have no
  credentials and must not need any.

That last decision has a cost you should be able to name: because `Auth`
runs inside the router, the access log — which runs outside it — cannot see
which client made the request. Fixing that is a design change, not a bug fix.

## Bearer tokens: what they buy and what they don't

The service authenticates with static bearer tokens, one per client, loaded
from the environment. Each request must carry `Authorization: Bearer <token>`;
anything else gets `401` with a `WWW-Authenticate` header and no hint about
*why* it failed.

The comparison is worth reading closely:

```go
digest := sha256.Sum256([]byte(presented))
if subtle.ConstantTimeCompare(digest[:], c.digest[:]) == 1 { … }
```

Two habits from the security lesson, both load-bearing. `==` on strings
returns as soon as it finds a differing byte, so an attacker who can time
your responses can recover a token byte by byte; `subtle.ConstantTimeCompare`
always looks at everything. And comparing *digests* rather than the raw
tokens makes both sides a fixed 32 bytes, so the comparison cannot leak the
token's length either.

Be equally clear about what this does not do. A bearer token is a shared
secret: whoever holds it is the client. It has no expiry, no scopes, and no
revocation beyond editing the environment and restarting. On plaintext HTTP
it is visible to every proxy on the path, so TLS is not optional. It goes in
a header, never in a URL — query strings end up in access logs, browser
history, and `Referer` headers. And it never appears in your own logs
either: the access-log middleware logs method, path, status, duration, and
request id, and deliberately nothing else. Sessions, JWTs, and OAuth all
trade this simplicity for expiry and scopes; you are not shipping to the
public internet, so simple and honest wins.

## Observability: three pillars, two of them here

The observability lesson framed logs, metrics, and traces. This service ships
the first two and tells the truth about the third.

**Logs** are structured `log/slog` records. One access line per request, with
stable attribute names (`method`, `path`, `status`, `duration_ms`,
`request_id`) — stable because dashboards and alerts are written against
them. Errors log the real cause at `ERROR` with the same request id; the
client gets `{"error":{"message":"internal error"}}` and nothing else.

**Metrics** are hand-rolled counters exposed at `/metrics` in the Prometheus
text format: `http_requests_total` by method, route and status; a duration
histogram; an in-flight gauge. That is RED (rate, errors, duration) plus a
saturation signal, and it is enough to answer "is it broken and how badly"
without opening a log. One rule matters more than the rest: **the `route`
label is the mux pattern, not the path.** `/tasks/{id}` is one time series;
`/tasks/1`, `/tasks/2`, … is one series per task, and that is how people take
their metrics backend down. Production services use the official client
library, which handles registries, label validation, and exposition edge
cases; writing the exposition by hand once — which you did in the
observability lesson — is how you stop treating it as magic. Once is enough,
so the registry ships with the skeleton here. The part that was never in the
formatter is still yours: which metrics exist, which labels they carry, where
the bucket bounds sit, and what the instrumentation wraps.

**Traces** are the pillar this service does not have. OpenTelemetry is the
standard answer, and it needs a collector to send spans to — out of scope
here. The `request_id` threaded through the context is the poor relative of a
trace id, and correlating a log line to a metric spike by hand is exactly the
work a tracing system automates.

Health and readiness are two questions, not one:

- **`/healthz`** — "is this process alive?" It touches nothing. A liveness
  probe that fails when the database blinks gets your process killed for
  someone else's outage.
- **`/readyz`** — "can this process serve traffic right now?" It pings the
  database. A failing readiness probe takes the instance out of the load
  balancer and leaves it running, which is what you want while a dependency
  recovers.

Neither requires credentials, and neither is counted in the request metrics:
probe traffic is not application traffic, and burying your signal under it is
self-inflicted.

## Exercise

Open [`exercise/`](exercise/) — a four-package skeleton with the plumbing in
place and the decisions left to you. Provided complete (read them, they are
part of the lesson): `task/task.go`, `sqlitestore/migrate.go`,
`cmd/taskd/main.go`, and in `httpapi` the `Chain`/`RequestID` middleware, the
response envelope helpers, `metrics.go`, and `server.go`.

Those last two are code you have already written — the registry and text
exposition in observability, the serve-and-drain loop in http-servers. Typing
them a second time would teach you nothing; defending them, which criteria 9
and 10 ask for, still does. Your work sites are the `TODO`s in
`task/service.go`, `sqlitestore/store.go`, `httpapi/api.go`, and
`httpapi/middleware.go`.

The test files are the specification. Read them before you write anything —
especially `httpapi/api_test.go`, which drives the whole service end to end
over a real SQLite database.

Acceptance criteria:

1. **Domain rules** (`task/service.go`, tested with no httptest). Titles are
   trimmed, then validated: empty → `required`, over 200 *runes* →
   `must be at most 200 characters`. New tasks are always `open` and carry
   one clock reading in both `CreatedAt` and `UpdatedAt`, in UTC. `List`
   sorts by id and never returns a nil slice. Invalid status filters and
   targets produce `{"status": "must be \"open\" or \"done\""}`.
2. **Lifecycle.** Setting the status a task already has succeeds and writes
   nothing. `done → open` returns `task.ErrAlreadyDone`. Storage errors
   travel out unchanged.
3. **Storage** (`sqlitestore/store.go`). Every query uses `?` placeholders.
   `sql.ErrNoRows` never escapes the package — callers see
   `task.ErrNotFound`. Timestamps round-trip without losing precision. The
   status filter runs in SQL, not in Go. `SetStatus` writes and reads back
   inside one transaction, and reports `ErrNotFound` when no row matched.
4. **Routing.** Go 1.22 patterns: `POST /tasks` (201), `GET /tasks` (200),
   `GET /tasks/{id}` (200), `PATCH /tasks/{id}` (200), `DELETE /tasks/{id}`
   (204, no body), plus `GET /healthz`, `GET /readyz`, `GET /metrics`.
   Unknown paths (404) and wrong methods (405) are the mux's job — write no
   code for them.
5. **Payloads.** Success is `{"data": …}` with `Content-Type:
   application/json`; failure is `{"error": {"message": …, "fields": …}}`.
   An empty listing is exactly `{"data":[]}`. `GET /tasks?status=open`
   filters.
6. **Error mapping** happens in exactly one function: validation → 400
   `validation failed` + fields, `ErrNotFound` → 404 `task not found`,
   `ErrAlreadyDone` → 409 `task is already done`, malformed JSON → 400
   `invalid JSON`, non-integer id → 400 `invalid id`, anything else → 500
   with exactly `{"error":{"message":"internal error"}}` while the real
   error goes to the log.
7. **Authentication.** `Auth(tokens)` accepts `Bearer <token>` for a
   configured client (scheme case-insensitive), puts the client's name on the
   request context, and answers everything else with 401, a
   `WWW-Authenticate` header, and `unauthorized`. An empty token map
   authenticates nobody. Only the task routes are wrapped; `/healthz`,
   `/readyz`, and `/metrics` are open.
8. **Middleware.** `AccessLog` emits exactly one line per request with
   `method`, `path`, `status` (200 when the handler never called
   `WriteHeader`), `duration_ms`, and `request_id`, and never logs the
   token. `Recover` turns a panic into a 500 in the standard envelope and
   logs it at `ERROR` with the request id. `Timeout(d)` answers a slow
   handler with 503 and the timeout envelope.
9. **Instrumentation.** The registry renders itself; deciding what it sees is
   your job. Every task route goes through `Metrics.Instrument` with the mux
   *pattern* as the `route` label, and `Instrument` wraps `Auth` rather than
   the other way round, so a flood of rejected credentials shows up as
   traffic instead of as silence. `/healthz`, `/readyz` and `/metrics` are
   not instrumented at all. In NOTES.md, defend the metric names, the label
   sets, and the bucket bounds you kept.
10. **Lifecycle.** `Run` and the server's timeouts are provided. Run
    `server_test.go`, watch a request drain through a shutdown, then argue
    the design in NOTES.md: what a SIGTERM does to a request already in a
    handler, and why `requestTimeout < WriteTimeout < shutdownGrace`.
    Demonstrate it for real with the Ctrl-C below.
11. **`NOTES.md` is filled in**, including the benchmark numbers for
    criterion 12. It is the agenda for the review conversation.
12. **A measured decision.** Run `BenchmarkListByStatus` with and without the
    v2 index on a fresh database, record both numbers, and be ready to argue
    for the result you got — including the cost of the index.
13. `go test -race ./...` is green and the code is `gofmt`-formatted.

Run everything from the module root:

```sh
cd exercise
go test -race ./...
```

Suggested order: `task/service.go` first (pure rules, fastest feedback), then
`sqlitestore`, then `httpapi/middleware.go`, then `api.go` — where the
routing, the error mapping, and the instrumentation wiring all land at once.
When it is green, run it for real:

```sh
TASKD_TOKENS=cli:local-dev-token go run ./cmd/taskd
curl -H 'Authorization: Bearer local-dev-token' -d '{"title":"first"}' \
     localhost:8080/tasks
curl localhost:8080/metrics
```

Then press Ctrl-C with a request in flight and watch it finish.

## Further reading

- [pkg.go.dev — net/http (Server, ServeMux, TimeoutHandler)](https://pkg.go.dev/net/http)
- [pkg.go.dev — log/slog](https://pkg.go.dev/log/slog)
- [Prometheus — exposition formats and metric naming](https://prometheus.io/docs/instrumenting/exposition_formats/)
- [Google SRE Book — Monitoring Distributed Systems](https://sre.google/sre-book/monitoring-distributed-systems/)
