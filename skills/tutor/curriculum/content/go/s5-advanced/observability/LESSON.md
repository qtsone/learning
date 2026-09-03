# Observability

> `go.advanced.observability` · ~3-4h · Stage: Advanced Go

## Objectives

By the end of this lesson you can:

- Explain the roles of the three observability signals (logs, metrics,
  traces) and which questions each one answers.
- Implement structured logging with `log/slog`: levels, per-request
  attributes, and a handler composed for production JSON output.
- Instrument a service with counters, gauges and histograms rendered in the
  Prometheus text exposition format, choose the right metric type and labels
  for a measurement, and explain label cardinality risks.
- Propagate W3C trace context through an HTTP handler and an outbound call so
  spans join one trace, and map the result onto OpenTelemetry's model.
- Given a symptom (a p99 latency spike, say), say which signal you would
  consult first and how you would drill down.

## Three signals, three questions

Profiling answered "where does this program spend its time on my machine,
right now, under a load I invented". Production asks harder questions, about a
program you cannot attach a debugger to, that broke twenty minutes ago, for
some users. Observability is the property that lets you answer them **from the
system's output alone**, without shipping code to add a print statement.

Three kinds of output carry that weight, and they are not interchangeable:

| Signal | Answers | Shape | Cost driver |
|--------|---------|-------|-------------|
| Logs | "what exactly happened in this one request?" | discrete events, high detail | volume × retention |
| Metrics | "how much, how often, how slow — over time?" | pre-aggregated numbers | number of time series |
| Traces | "where did the time go across services?" | a tree of timed spans | sampling rate |

They overlap, and each one *can* fake the others badly: counting log lines
gives you a rate at a hundred times the storage cost, and no pile of logged
durations gives you a p99 you can alert on. Hence the rule of thumb —
**metrics tell you something is wrong, traces tell you where, logs tell you
why**. Health endpoints get lumped in with all this and shouldn't be: they are
not for humans, but for machines making decisions about your process. They
come last, with their own trap.

## Structured logging with slog

You have used `log/slog` since S4 as "the logger". Now look at its structure,
because production logging is a design problem. A record is a message plus
**attributes** — typed key/value pairs, not text you interpolated:

```go
// Searchable, aggregatable, machine-readable:
logger.Info("order placed", "order_id", id, "items", n)
// Not:
logger.Info(fmt.Sprintf("order %s placed with %d items", id, n))
```

Both lines read the same to you. Only the first lets someone ask "how many
orders had more than 50 items last Tuesday" without a regular expression
against a log file at 3am.

Three parts make up slog: the `Logger` you call, the `Record` it builds, and
the **`Handler`** that decides what happens to it. `slog.NewJSONHandler` emits
one JSON object per line — the right choice for anything a machine collects;
`slog.NewTextHandler` is for humans at a terminal. The handler also holds the
**level**: pass a `*slog.LevelVar` and you can raise verbosity at runtime (a
flag, an env var, an admin endpoint) instead of redeploying to find out what
the service was thinking.

Levels are a budget, not a mood: `Debug` off in production and on when you are
hunting, `Info` for normal operation, `Warn` for "recovered, but a human should
know", `Error` for a failure someone must look at. If nobody looks, they were
warnings.

Every call has a context-carrying form — `InfoContext`, `ErrorContext`,
`LogAttrs` (typed `slog.Attr` values, no `any`-slice conversion). That `ctx`
argument is not decoration: it is the whole trick of the next section.

### Handlers compose

Every `Info`/`Error` call site knowing to attach the request id is not a plan.
The plan is a handler that does it for them. `slog.Handler` is a four-method
interface — `Enabled`, `Handle`, `WithAttrs`, `WithGroup` — so a decorator that
embeds another handler only has to override what it changes:

```go
type ContextHandler struct{ slog.Handler }

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
    if id := RequestIDFrom(ctx); id != "" {
        r.AddAttrs(slog.String("request_id", id))
    }
    return h.Handler.Handle(ctx, r)
}
```

Now every record logged with a request context carries the id, and no call
site changed — the same decorator move as HTTP middleware, on a different
interface.

One trap, and the reason `logger.With(...)` shows up in this lesson's tests:
`WithAttrs` and `WithGroup` return a *handler*, and if you let the embedded
one answer you get back a bare `JSONHandler` — every logger derived with
`With` silently loses its ids. Wrap the result:

```go
func (h *ContextHandler) WithAttrs(a []slog.Attr) slog.Handler {
    return &ContextHandler{Handler: h.Handler.WithAttrs(a)}
}
```

Two more production rules. **Never log secrets or personal data** — tokens,
passwords, card numbers, and depending on your jurisdiction email addresses
and IPs; a log line is copied to five systems within a second of being
written. And **log per request, not per step**: one access line with rich
attributes beats six lines that have to be stitched back together.

## Metrics: cheap numbers over time

A metric is a number the process keeps in memory and a collector reads
periodically — the *pull* model Prometheus popularized: your service exposes
`/metrics`, a scraper fetches it every 15 seconds (the alternative *push*
model, StatsD or OTLP, sends measurements out). Pull has a pleasant property
for this lesson: a metrics endpoint is just a handler that prints text.

Three types cover almost everything:

- **Counter** — only goes up: requests served, errors, bytes. You never read
  its raw value; you ask for its *rate*. That is why a counter that can go
  down is a bug: every rate calculation downstream assumes monotonicity and
  treats a decrease as a process restart.
- **Gauge** — goes up and down: requests in flight, queue depth, goroutines,
  memory in use.
- **Histogram** — counts observations into cumulative buckets, plus a sum and
  a count. `latency_bucket{le="0.5"}` is "how many requests took ≤ 0.5s".

Why not just store the durations? A service handling 50k requests per second
cannot ship 50k numbers per second, but it can ship twelve bucket counts. The
price: quantiles become *estimates* interpolated inside a bucket, so put your
bounds around the latencies in your SLO — and **you cannot average
percentiles**, you aggregate buckets first and compute the quantile after.

Naming conventions are what make a dashboard portable: `snake_case`, a unit
suffix in base units (`_seconds`, `_bytes` — never `_ms`), and `_total` on
counters: `http_request_duration_seconds`, `http_requests_total`.

### Labels, and the cardinality bomb

Labels turn one metric into many time series — one per distinct combination
of label values:

```
http_requests_total{method="GET",route="/items/{id}",status="200"}
```

Series count multiplies: 5 methods × 20 routes × 8 statuses = 800 series.
Fine. Now add `user_id`. With 100k users you just asked your monitoring system
to hold 80 million series, and the usual outcome is that the *monitoring*
system falls over — during the incident, because that is when traffic and
error variety peak. It is the most common way teams break their own
observability.

The rule: **labels must be bounded and known in advance**. Route patterns,
status codes, methods, queue names, versions: yes. User ids, request ids,
paths with ids in them, error messages, URLs with query strings: no. Those
belong in logs and traces, which are stored as events, not as series.

That is why the exercise labels by the *route pattern* `/items/{id}` and not
by `r.URL.Path` — and why the route arrives late: the middleware plants a
mutable box on the context and the router fills it in once it has matched, the
same interception trick as `statusWriter`.

Two framings tell you what to measure: **RED** for request-driven services
(Rate, Errors, Duration) and **USE** for resources (Utilization, Saturation,
Errors). The exercise does RED plus one saturation gauge — the 3am set.

> In production you would import `prometheus/client_golang` instead of writing
> a registry. You write one here — registration plus the exposition pass — so
> that get-or-create, metric families and the text format stop being magic. The
> three metric types come already written: their bodies are arithmetic.

## Traces: the shape of a request

A **span** is one timed operation with a name, a start, a duration and a
parent. A **trace** is the tree of spans that one logical request produced,
possibly across a dozen services. Traces answer the question neither logs nor
metrics can: *of the 900ms this request took, which hop ate 700 of them?*

For spans in different processes to end up in one tree, two ids must travel
with the request. The industry agreed on the **W3C Trace Context** header:

```
traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01
             │  │                                │                └ flags: bit 0 = sampled
             │  │                                └ span id: 8 bytes, the caller's active span
             │  └ trace id: 16 bytes, constant for the whole request
             └ version
```

Your server parses it, treats the incoming span id as the *parent* of the span
it creates, and passes the new span context down. When it calls another
service, it renders its own span context back into a `traceparent` header —
*injection*. That is the entire propagation mechanism, and it is why
`context.Context` is the carrier: the span context has to reach an outbound
call five frames deep without every signature growing a parameter.

Two consequences worth internalizing:

- **A malformed incoming header must not corrupt your trace.** Reject it and
  start a fresh one. Trusting a caller's garbage produces traces that appear
  to link unrelated requests.
- **Sampling is a first-class decision.** The flags byte carries "this trace
  is being recorded" and downstream services inherit it, so head-based
  sampling keeps a trace all-or-nothing rather than full of holes.

Span names follow the cardinality logic of labels: name them after the route
(`GET /items/{id}`), not the path. In the exercise the span starts before the
route is known, so it is renamed on the way out — as real instrumentation does.

> **OpenTelemetry** is the vendor-neutral standard for all of this: an API, an
> SDK and a wire protocol, with `go.opentelemetry.io/otel` as the Go
> implementation. Its model is the one you are building, piece for piece: your
> `SpanContext` is its `SpanContext`; `Tracer.Start` and `Span.End` are its
> `Tracer` and `Span`; `ParseTraceparent` and `Inject` are its *propagator*
> (`propagation.TraceContext`); `Tracer.Finished` stands in for its *exporter*.
> Adopting it costs dependencies and usually a collector, which this lesson
> does not add — but that mapping is what makes its API readable on day one.

## Correlation, and the 3am drill

What makes the three signals a *system* is the ids that join them: the
**request id** identifies one request inside your service (a user can quote it
off an error page), the **trace id** identifies one request across *all* of
them. Put both on every log record — that is what your `ContextHandler` is for.

The drill, in the order you actually do it:

1. **An alert fires** — error rate or p99 latency over the SLO. That alert is
   built on metrics, because metrics are cheap to evaluate every 15 seconds.
2. **Metrics narrow it**: which route, status, instance? Is traffic up (rate),
   or is the same traffic slower (duration)? Are we saturated (in-flight)?
3. **Traces localize it**: pull traces for that route in that window; the
   waterfall says whether the time is yours, the database's, or a dependency's
   dependency's.
4. **Logs explain it**: filter by the trace id of a slow request. Now you are
   reading ten lines, not ten million.
5. **Profiles, if it is in your process**: the S5 profiling lesson takes over
   — `/debug/pprof` on the running service.

Run it backwards ("let me grep the logs") and you will spend the first twenty
minutes of an incident reading text.

## Health versus readiness

Two endpoints, two different questions, and conflating them causes outages:

- **Liveness** (`/healthz`): "is this process fundamentally broken?" A failed
  check gets the process *killed and restarted*, so it must check **nothing
  external**: a probe that pings the database turns a database blip into a
  fleet-wide restart storm whose restarts hammer the recovering database.
- **Readiness** (`/readyz`): "should this instance receive traffic *right
  now*?" A failed readiness check pulls the instance out of the load balancer
  but leaves it running. This one *does* check dependencies: is the migration
  done, is the connection pool up, is the cache warm. It also gives you a
  clean rollout — a new instance takes traffic only when it is ready — and a
  clean shutdown: fail readiness first, let the load balancer drain you, then
  run the graceful shutdown you wrote in the HTTP servers lesson. Make it
  report *which* check failed — "unready" alone turns every rollout into a
  guessing game.

## Exercise

Open [`exercise/`](exercise/) — the observable skeleton of an HTTP service.
`provided.go` and `main.go` are complete (the middleware chain you built in
the HTTP servers lesson, plus the routing table); read them, especially the
order of the chain in `main`. So are the `Counter`, `Gauge` and `Histogram`
types in `metrics.go` — read `Observe` and the `appendTo` methods, they are
the exposition format made concrete. Your work sites are the `TODO`s in
**`logging.go`**, **`metrics.go`**, **`trace.go`** and **`server.go`**. The
four test files are the specification.

Acceptance criteria:

1. `NewLogger(w, level)` writes JSON records to `w`, honours `level` on every
   call (pass a `*slog.LevelVar` and changing it mid-run takes effect), and
   wraps its handler in `ContextHandler`.
2. `ContextHandler` adds `request_id` from the context, and `trace_id` /
   `span_id` when a span is active. Ids that are absent are not logged at all.
   `WithAttrs` and `WithGroup` return a `*ContextHandler`, so loggers derived
   with `With`/`WithGroup` keep the behavior.
3. `Registry` hands back the *same* metric for the same name and label set,
   is safe under concurrent use (including a scrape running while handlers
   write), and panics if a name is reused with a different metric type.
4. `Registry.Write` renders the registry in the Prometheus text format
   documented on it: one `# TYPE` line per family, families sorted by name,
   samples sorted by their rendered label set. Each metric renders its own
   samples; your job is what the registry stores and when it holds its lock.
5. `ParseTraceparent` accepts only well-formed version-`00` headers — four
   fields, 32- and 16-digit lowercase hex ids, neither all-zero — and
   `SpanContext.Traceparent` renders a value it accepts back.
6. `Tracer.Start` continues the trace on the context when there is one (same
   trace id, same sampling decision, parent set to the active span) and starts
   a fresh trace otherwise; the returned context carries the new span.
   `Span.End` records the span with its duration; `Finished` returns them in
   End order. `Inject` writes the active span context into outbound headers,
   and does nothing when there is no span.
7. `Tracing(tracer)` starts one server span per request, continuing a valid
   incoming `traceparent` and ignoring a malformed one, names the span
   `"<METHOD> <route>"` once the route is known, and echoes the trace id on
   the `X-Trace-Id` response header.
8. `Observe(reg, logger)` records `http_requests_total{method,route,status}`,
   `http_request_duration_seconds{route}` (with `DefaultBuckets`) and the
   `http_requests_in_flight` gauge, and logs exactly one `"request"` line per
   request with `method`, `route`, `status`, `duration` — through the request
   context, so the ids from criterion 2 appear. The `route` label is the
   matched pattern (`RouteFrom`), or `"unmatched"`; a panicking handler is
   counted and logged as a 500.
9. `HealthHandler` answers 200 `ok` regardless of dependencies. `Readiness`
   runs its registered checks per request and answers 200
   `{"status":"ok","checks":{…}}` or 503 `{"status":"unready","checks":{…}}`
   with the failing check's error text, as JSON. `MetricsHandler` serves the
   registry as `text/plain; version=0.0.4; charset=utf-8`.
10. `go test -race ./...` passes and the code is `gofmt`-clean.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test -race ./...
```

A sane order: `logging.go` first (small, and everything logs), then
`metrics.go` (registration, then `Write` against `TestRegistryExposition`),
then `trace.go`, then `server.go`, which is mostly gluing the three together.
Finish with the real thing:

```sh
go run .                                      # then, in another terminal:
curl -i localhost:8080/items/7
curl -s localhost:8080/metrics
curl -i localhost:8080/readyz                 # 503 for the first 5 seconds
curl -i localhost:8080/healthz                # 200 the whole time
```

Watch one request produce a JSON access line, a counted series, and a span —
and check that the `trace_id` in the log matches the `X-Trace-Id` header you
got back. That is the lesson.

## Further reading

- [Go blog: Structured Logging with slog](https://go.dev/blog/slog) — the
  handler contract, and why it is shaped this way.
- [Prometheus — Metric types](https://prometheus.io/docs/concepts/metric_types/)
  — counters, gauges and histograms from the people who get paged about them.
- [W3C Trace Context](https://www.w3.org/TR/trace-context/) — the traceparent
  header, field by field.
- [OpenTelemetry Go — Getting started](https://opentelemetry.io/docs/languages/go/getting-started/)
  — the real version of the tracer you just wrote.
