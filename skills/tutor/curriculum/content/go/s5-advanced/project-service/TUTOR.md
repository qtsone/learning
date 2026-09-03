# Tutor notes — Mini-Project: Production Service

## Where the learner is

Last lesson of S5, and the last Go-only lesson before the shared systems
stage. They have every technique this project needs — 1.22 routing,
middleware, layered packages, `database/sql` with migrations and a bounded
pool, `log/slog`, metrics and health endpoints, fake clocks, table tests,
graceful shutdown — each learned separately against a small example. Nothing
here is new; everything here is *assembly*, and assembly is where their
mental model gets tested.

Expect this to take two or three sittings. Push them to work in the suggested
order (domain → storage → middleware → api) and to commit after each package
goes green; a capstone attempted as one giant edit is how people lose a
weekend. In `guided` mode, read `api_test.go` with them before they write a
line: the suite is the specification and half the design questions answer
themselves once they have read it.

Two files ship complete that a learner might expect to write: `metrics.go`
(the registry and exposition they built in observability) and `Run` (the
serve-and-drain loop from http-servers). Re-typing either proves nothing a
capstone should be spending its hours on. The requirement moved rather than
vanished: criterion 9 is now the *wiring* — pattern-not-path labels,
`Instrument` outside `Auth`, probes uninstrumented — plus the NOTES.md
defence of the metric design, and criterion 10 is the NOTES.md argument for
the timeout ordering. If they hand in green tests and an empty section 6 or
7, they have not done criteria 9 and 10.

This is a **review** lesson as much as a coding one. `NOTES.md` and the quiz
are not optional extras — objective 5 is "defend the architecture", and a
green test suite does not demonstrate it.

## Common misconceptions

- **"Layers are ceremony for a service this small."** Push back with the
  concrete question from NOTES.md: which packages change when storage moves
  to Postgres? Then ask what a handler that talks to `*sql.DB` directly does
  to the test suite.
- **"Liveness and readiness are the same endpoint."** The costly version of
  this belief is a liveness probe that pings the database: the database
  hiccups, every replica reports unhealthy, the orchestrator kills all of
  them, and a recoverable dependency outage becomes a total one.
- **"The timeout middleware stops the handler."** It does not, and cannot.
  `http.TimeoutHandler` answers the *client*; the handler goroutine runs
  until it checks its context. If they cannot explain this, they will write
  timeouts that leak goroutines forever.
- **"401 vs 403."** 401 is "I do not know who you are"; 403 is "I know, and
  no". With bearer-token-only auth there is no 403 case yet.
- **Path as a metric label.** The most common real-world outage in this
  lesson's subject matter. Ask what `/tasks/{id}` costs with a million tasks.
- **"Constant-time comparison is paranoia."** Ask them to describe how they
  would recover a token from a service that uses `==`, out loud. If the
  answer is vague, the habit will not stick.
- **Logging the Authorization header** "just while debugging". Logs get
  shipped, copied, and kept for a year. There is no "just while debugging".
- **"The tests are slow because SQLite is slow."** They are slow because the
  race detector is on and each test opens its own database. Do not let them
  "optimize" by sharing one database between tests — isolation is why the
  suite is trustworthy.
- **Idempotence confusion** — they often implement `done → done` as a write
  with a fresh timestamp. The test catches it; make sure they can say *why*
  a no-op must not churn `updated_at`.

## Grilling points

Ask these in the learner's own words; `quiz.json` has the core set, these go
deeper.

- "Walk me through a `PATCH /tasks/7` from socket to SQL and back. Name every
  layer it crosses and what each one adds."
- "Reverse two middlewares in the chain and tell me what breaks. Now do it in
  the code and check whether you were right."
- "Your service is returning 500s. Which of your three signals — access log,
  metrics, `/readyz` — do you look at first, and what does each one tell you
  that the others don't?"
- "Someone leaks a token in a screenshot. What happens next, minute by
  minute?"
- "Why is `Instrument` outside `Auth`? What would the graph look like if it
  were inside, during a credential rotation gone wrong?"
- "Your `Timeout` is 5s and `WriteTimeout` is 10s and `shutdownGrace` is 15s.
  Justify that ordering. What breaks if you reverse it?"
- "You measured the index. What would have to be true for you to *remove*
  it?" (Write-heavy workload, tiny table, memory pressure.)
- "Where would a trace id have told you something the request id cannot?"
  (Across process boundaries — which this service does not have yet.)

## Grading rubric

- **A** — All tests pass on the first honest run of `go test -race ./...`.
  `NOTES.md` is filled in with real benchmark numbers and a defensible
  argument. Layer boundaries are clean (no `net/http` in `task`, no SQL
  outside `sqlitestore`, error mapping in exactly one function). They can
  narrate the middleware order and its consequences without notes, and can
  state both what the token strategy protects against and what it does not.
- **B** — Tests pass; the structure holds, but one boundary is smudged
  (validation duplicated in a handler, a status code decided outside
  `respondError`) or `NOTES.md` is thin — numbers present, reasoning
  shallow. Explanation solid on the mechanics, hand-wavy on the trade-offs.
- **C** — Tests pass only after substantial hinting, or the design is
  copied-in shape without understanding: they cannot say why `Auth` sits
  inside `Routes`, or why the route label is a pattern. Pass only if a
  focused remediation conversation lands the two or three ideas they missed.
- **Fail** — Tests failing; or storage errors leaking to clients; or the
  learner cannot explain their own error mapping or auth check. This is the
  stage capstone: do not advance a learner who cannot defend it. Better to
  spend another session than to start S6 on sand.

## Remediation ladder

1. "Which package's tests are failing? Run just that one and read the first
   failure message aloud — these tests were written to tell you what they
   wanted."
2. "You have built each of these pieces before. Which lesson had this exact
   problem in miniature?" (Point at the sibling lesson, not the answer:
   rest-services for envelopes and mapping, databases for transactions,
   http-servers for the middleware shapes, security for the constant-time
   compare. For the metrics wiring, point at the provided `metrics.go` and
   ask what `Instrument` needs to be given.)
3. Narrow to one behavior and reason about it together — for example: "What
   does `sw.status` hold if the handler never calls `WriteHeader`? Where does
   that default get set?"
4. If they are stuck on structure rather than syntax, sketch the shape on
   paper — the four cases of `respondError`, or the nesting of `Instrument`
   around `Auth` around the handler — and let them type it. Never paste the
   solution file; this is the lesson where typing it is the point.

## After passing

Celebrate properly: this is a service they could actually operate. Then
preview: "S5 was Go at production depth. S6 steps back from the language
entirely and asks how systems are designed — the stage where the questions
stop having compiler-checkable answers."
