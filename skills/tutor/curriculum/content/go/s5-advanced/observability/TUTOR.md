# Tutor notes — Observability

## Where the learner is

Eleventh lesson of S5. They can build an HTTP service with middleware, layers,
a database and tests, and they have spent three lessons inside the runtime and
one on profiling. What they have never done is operate a service they cannot
attach a debugger to. This lesson is the bridge from "it passes tests" to "I
can find out what it did last Tuesday", and it is the last piece of vocabulary
before the stage capstone, which expects a middleware chain, health endpoints
and a metrics endpoint as a matter of course.

The exercise is four independent problems (slog handler, metrics registry,
tracer, HTTP wiring) that only meet in the final correlation test. Encourage
them to finish and green one file at a time rather than sketching all four.
In `metrics.go` the three metric types come written — the work there is
registration and `Write` — so if they are running short on time, the order in
the lesson (logging → metrics → trace → server) protects the most transferable
material.

## Common misconceptions

- **"Structured logging means my messages are JSON."** It means the *fields*
  are data. `slog.Info(fmt.Sprintf(...))` in a JSON handler is still a string
  nobody can query. Watch for interpolated messages in their solution.
- **"Log levels are severity vibes."** Push for the operational definition:
  Error means a human should look. If everything is an error, nothing is.
- **`WithAttrs` returning the inner handler.** The single most common failure
  here, and the tests name it. Ask them what `logger.With("component","api")`
  returns and where the ids went.
- **"A counter is just a number I can also decrement."** Rates assume
  monotonicity; a decrease reads as a restart. Same family: reading a
  counter's absolute value on a dashboard instead of its rate.
- **"More labels, more insight."** The cardinality bomb. Make them do the
  multiplication out loud for `user_id`. Related: labelling by `r.URL.Path`
  because it was already in the request — the exercise's route holder exists
  precisely to make the right thing easy.
- **Averages and percentiles.** "The average latency is fine" is compatible
  with 1% of users timing out; and averaging per-instance p99s is not the
  fleet p99. If they are shaky here, have them reason about the bucket counts
  their own histogram produces.
- **"The trace id is the request id."** Related but different scopes: request
  id is yours, trace id spans services. Both belong on the log line.
- **Liveness that checks the database.** The restart storm. Ask what happens
  to fifty instances when the database has a ten-second blip.
- **"Tests are green so the observability works."** It is *their own* logs and
  metrics; nothing external validates them. The manual `go run .` pass at the
  end is not optional — a service whose metrics endpoint 500s in production
  fails silently.

## Grilling points

- "Your p99 latency alert fires. Walk me through the next ten minutes." (Want:
  metrics to narrow route/instance/rate-vs-duration, traces to localize the
  hop, logs filtered by trace id, pprof last.)
- "Why did you not just log the duration of every request and compute the p99
  from that?" (Volume, retention cost, query latency; histogram is
  pre-aggregation. Then: what did you *lose*? Bucket-boundary precision.)
- "A colleague adds `customer_id` as a metric label to debug one customer.
  What do you say in review, and what do you suggest instead?" (Cardinality;
  put it on the log line and the span.)
- "Your service returns 500s. Which of your three signals told you first, and
  which one told you why?"
- "Show me the line where a trace survives a network hop." (`Inject`.) "Now
  delete it — what breaks, and what does the downstream trace look like?"
  (Two disconnected traces, no parent link.)
- "Why does `Tracing` rename the span at the end instead of naming it right?"
  (The route is only known after the mux matched — same reason the route label
  needs the holder.)
- "Your readiness check fails for 30 seconds during a deploy. What does the
  load balancer do, what do users see, and what would the same failure in
  liveness have done?"
- "Point at each piece you wrote and give me OpenTelemetry's name for it."
  (`SpanContext` → `SpanContext`, `Start`/`End` → `Tracer`/`Span`,
  `ParseTraceparent` + `Inject` → a propagator, `Finished` → an exporter.)
  Then: "what would you replace with a library at work, and what would you
  keep?" (Registry → client_golang, tracer → OpenTelemetry; keep the handler
  decorator, the middleware order, the route holder, the health/ready split.)
- Stretch: "Where would you put a `traceparent` on an outbound database call?"
  (No standard header; spans still nest via context — the propagation story
  differs per protocol.)

## Grading rubric

- **A** — All tests pass under `-race`. `Registry.Write` snapshots under the
  lock and renders outside it; the metric types are chosen correctly and the
  label sets are bounded; `ContextHandler` re-wraps in both `WithAttrs` and
  `WithGroup`; `Tracing` rejects a malformed traceparent without breaking the
  trace; the access log is one line per request. They can run the 3am drill
  unprompted and defend liveness-vs-readiness with a concrete failure mode.
- **B** — Tests pass, but with a wart: rendering while holding the registry
  lock, `Finished` handing back the live slice, an empty `request_id` logged
  as `""`, or metric names that ignore the unit/`_total` conventions. The
  explanations are right; the craft needs one more pass.
- **C** — Tests pass only after heavy hinting, or an objective is parroted:
  cardinality named but not explained, or the drill answered as "check the
  logs". Time-box remediation; if the cardinality and health/readiness answers
  do not land, iterate rather than advance.
- **Fail** — Tests failing, `-race` complaining, or the correlation test green
  without them being able to say which id joins which signal.

## Remediation ladder

1. "Read the failing assertion's message — it names the concept, not just the
   value. Which of the four files does it live in?"
2. Narrow with `go test -race -run TestRegistryExposition` (or
   `TestContextHandler…`, `TestParseTraceparent`, `TestSignalsCorrelate`) so
   they work one problem at a time.
3. Targeted Socratic nudges:
   - Exposition mismatch: "diff the two blocks the test printed — is it order,
     spacing, or the bucket values? Buckets are cumulative."
   - Ids missing after `With`: "print `%T` of `logger.Handler()` before and
     after `With`."
   - No spans recorded: "who calls `End`, and does the deferred rename run
     before it?"
   - Route label wrong: "when does the mux know the route, relative to when
     your middleware reads it?"
4. Draw the picture with them — chain order on paper, or the traceparent field
   by field — and let them type the fix. Only walk a full implementation for
   `Registry.Write`, and only after they have the data structure right.

## After passing

Preview: "Next is the stage capstone: a production HTTP service that has to
have all of this in it from the first commit — layered packages, a real
database, graceful shutdown, and the middleware chain you just wired,
observability included."
