# Web Services Capstone

> `focus.web.capstone` · ~8-12h · Stage: Focus: Web Services

## Objectives

By the end of this lesson you can:

- Extend a production service with authentication and middleware-enforced
  authorization on every protected route.
- Implement a realtime feature integrated with the service's existing domain
  events.
- Implement a background-job flow with retries and an idempotent consumer for a
  slow operation.
- Harden the extended service with rate limiting, strict validation and security
  headers, and demonstrate each with tests.
- Defend the design in review: justify the auth model, the realtime transport
  and the job-queue choices against the alternatives.

## Where the bugs actually are

Seven lessons, seven working pieces: a password verifier, a rule table, a hub, a
token bucket, a dataloader, an outbox, a keyset cursor. Each one was correct in
isolation, and none of them is where a real service goes wrong.

Real services go wrong at the **seams**. The authorization rule that governs
`GET /tasks` and not the stream that carries the same rows. The event announcing
a row that then failed to commit. The listing that is correctly scoped and
correctly cached — into a shared cache, where the next user gets it. The role
change that lands in the database while three sessions carry the old privilege
around for another twelve hours.

Every one of those is a bug you cannot find by testing a component, because every
component is behaving exactly as designed. They appear only where two designs
meet, and that is what this capstone is: one service, eight hours, and the parts
of the pack that only exist in between the parts.

## The service

A task board. Members own tasks; auditors read everything and change nothing;
admins do everything. It is small on purpose — one domain object, six routes —
so that nothing you spend time on is CRUD you have written before.

```
POST   /login                 mint a session          (provided)
POST   /logout                destroy it              (provided)
GET    /tasks?limit=&cursor=  the caller's feed
POST   /tasks                 create, enqueue, announce
GET    /tasks/{id}            one task
PATCH  /tasks/{id}            change state, announce
DELETE /tasks/{id}            exists; nothing grants it
POST   /users/{id}/role       admin only, with consequences
GET    /events                text/event-stream
```

The primitives are provided, finished and commented: the bcrypt hasher and
session store, the hub with its bounded buffers and eviction, the token bucket,
the strict JSON decoder, the SQLite-backed store and queue, the worker pool. You
have built each of them. What is missing is every line where two of them have to
agree.

## Seam 1: identity that refuses nobody

Authentication answers *who is this*; authorization answers *may they*. Keeping
them apart has a concrete consequence for the middleware you are about to write:
**`Authenticate` rejects nobody.** It reads the cookie, looks up the session,
attaches a `Subject`, and lets an anonymous request through untouched. Every
refusal — the 401 for the anonymous, the 403 for the merely unwelcome — comes
out of the authorization layer, in one place, with one audit line.

That leaves the chain order as a real decision, and it is not the one from the
hardening lesson unchanged. That lesson put the rate limiter near the outside so
a refusal costs a map lookup. This service wants to limit **per account**,
because an address is a poor identity: NAT hides thousands of people behind one,
a phone changes networks mid-session, and IPv6 hands one attacker more addresses
than you have memory. But you cannot key on an account before something has
established which account it is.

So the limiter moves inside authentication and pays for a session lookup on a
request it is about to refuse — a map read, next to the body read and JSON
decode it still refuses to do. Anonymous callers fall back to the address, which
is exactly right for the one anonymous endpoint that costs real money: a login,
where each attempt is a deliberately slow password hash.

Meanwhile the security headers stay outermost, because the responses that most
need them are the ones no handler ever produced. One middleware from that lesson
is missing entirely: `Timeout`, because `http.TimeoutHandler` buffers the
response and cannot flush, so `GET /events` behind it would die the same way it
would behind a global `WriteTimeout` — per-request budgets here live on the
context inside the handlers that need them.

## Seam 2: one policy, four call sites

The authorization lesson's rule was one decision function and two enforcement
points. This service has four call sites, and the extra two are the interesting
ones:

| Where | What it can decide | Why it exists |
|---|---|---|
| Route gate (middleware) | role vs action | a denied request never enters a handler |
| Object check (handler) | ownership, object in hand | the middleware has not loaded the row |
| Listing scope (query) | which rows to select at all | a filter can forget a row; a `WHERE` cannot |
| Stream filter (SSE loop) | which events to write | the hub has no idea who may see what |

That is still *one* enforcement point in the sense that matters: one function
interprets the rules and every call site funnels through it. What is distributed
is the invocation, because only the handler has the object and only the stream
has the event.

The trap is the third and fourth rows. Both are tempting to write as `if role ==
auditor { everything }`, and both are then a second copy of your access-control
model — one that will not move when the rule table moves. Six months later
somebody narrows the auditor role in the table, ships, and the listing keeps
handing out every row because the listing never read the table. So the scope is
*derived*: ask the policy whether this subject may read a task somebody else
owns, and let the answer decide between "everything" and "your own".

The stream is the same question one layer further out, and it is the leak nobody
looks for. Nothing about `hub.Publish` suggests authorization. The hub is a
mechanism: it fans events out and stays out of policy, deliberately, because a
hub that consulted a policy would be a second copy of one. Every event carries
the one attribute the policy needs — the owner — and each stream filters for its
own subscriber. Skip it and you have built the unscoped listing again, arriving
over a pipe with no status code, no test, and nobody watching.

`Event.Frame` comes across from the realtime lesson unchanged, validation
included, and it is worth saying why since nothing here can trip it. Every field
this service publishes is server-generated: ids are counters, `Data` is
`json.Marshal` output, and marshalling escapes the control characters that
matter. The guard is for the event you have not written yet — the one carrying a
task title, a comment, an error message. A bare `\r` ends a line in
text/event-stream, so an unvalidated field is how a user forges an `event:` or
an `id:` on somebody else's connection. Dropping a security check because
today's callers cannot reach it is how it is missing on the day they can.

## Seam 3: the write that must not split

`POST /tasks` does three things: write the row, queue the notification, tell the
connected clients. Two of them belong in one transaction and the third must not
be.

The outbox rule you already know: the task and its job commit together, because
there is no ordering of "write the row" and "publish the job" that survives a
crash when they are two systems — so they are not two systems. The job is a row
in the same database, on the same transaction.

The new half is the announcement, and it is the mirror image. A transaction can
be rolled back; a line that has appeared on somebody's screen cannot. So the
publish happens **after** the commit, never inside it:

```go
if err := tx.Commit(); err != nil { … }
s.hub.Publish(taskEvent(EventTaskCreated, task))
```

Get that backwards and a failed commit leaves every connected client showing a
task that does not exist, with no event coming to correct it. The same rule is
why the worker pool — not the job handler — publishes what a job produces: the
handler cannot know whether its own transaction is about to commit.

There is an honest gap in the other direction. A crash between the commit and
the publish loses the announcement: the row exists and nobody was told. That is
the trade you are making, and it is the right one, because "told about something
that did not happen" is unrecoverable and "not told about something that did" is
fixed by the next refresh — or by the reconnect that replays from `Last-Event-ID`.
Naming which failure you chose is the deliverable, not avoiding failure.

## Seam 4: at-least-once, in front of a live screen

Delivery is at least once. It always will be, because "do the work" and "record
that the work is done" are two operations and a crash can land between them. The
consumer therefore has to be idempotent — and in a service with a live stream,
idempotent means two things now:

1. the effect happens once (one notification row, marker and effect on the same
   transaction, both rolled back together when anything fails);
2. **the announcement happens once**. A redelivery that skips the database write
   and still publishes has told the user twice, which is exactly the bug the
   dedup ledger existed to prevent, relocated to a place your row counts do not
   look.

The second one is why the gate stays exactly where the background-jobs lesson
put it: in `Pool.run`, not in the handlers. The pool claims the job id in the
ledger on the transaction *before* it dispatches, so a duplicate never reaches a
handler at all — it writes no row, and it returns no events for the pool to
publish. Both halves out of one call, for every kind of job that will ever
exist.

The alternative is the tempting one, so name what it costs. Let each handler
mark its own job and each kind can decide for itself what a duplicate means —
but then a new kind is idempotent only if whoever wrote it remembered, and
forgetting is invisible until a user is told something twice. It is the same
argument as the route table you are about to write, where naming an action on
every route is what makes omitting a gate impossible: a rule each author has to
remember is a rule you have not enforced.

## Seam 5: caching a response that depends on who asked

The feed is a perfect caching candidate: read-mostly, expensive relative to its
size, requested constantly. It is also scoped by identity, which makes the naive
`Cache-Control: public, max-age=30` from the performance lesson a data breach
with good latency.

Two headers carry the whole decision:

- **`Cache-Control: private, no-cache`** — one client may hold it; no shared
  cache may. (`no-cache` means "revalidate before reuse", not "do not store" —
  which is what makes the ETag worth having.)
- **`Vary: Cookie`** — this body depends on the session, and a cache that does
  not know that will hand one user another user's tasks.

The ETag still pays: a member polling every ten seconds gets a 304 with no body,
and your handler still ran the query — cheap, not free. Caching *after*
authorization is the rule; a cache in front of an access-control decision is a
cache of somebody else's data.

## Revocation: when the answer changes under a live session

`POST /users/{id}/role` is four lines of code and one of the more interesting
decisions in the service. The role changes in the database — and every session
that user already has is still carrying the old privilege, because a session is
a lookup key and nothing about it re-asks who they are.

The auth lesson's rule was that a privilege change rotates the session. Here the
change is applied by *somebody else*, so rotation is not available: you cannot
hand a new cookie to a browser that is not making this request. The available
answer is stronger and simpler — destroy the target's sessions. One re-login for
a user whose permissions just changed, and no window in which an id captured
under one privilege is an id under another.

Note what this buys, and that a JWT could not have bought it: an immediate,
service-wide "no" that costs one map delete.

"Immediate" has a catch, and it is the last seam in the service. Deleting the
session stops the *next request* — and an open `/events` connection is not a
next request. It made one, hours ago, and has been living off the `Subject` the
middleware attached to it ever since. Destroy the sessions of a demoted auditor
and their browser tab keeps receiving everybody's tasks over a pipe that was
authorized under the old answer, for as long as they leave it open.

So the stream re-resolves rather than captures: it keeps the session id and asks
`subjectFor` again on every event, and on every heartbeat, so an idle connection
dies too. A session that is gone ends the response — the only refusal left, since
the status line went out with the first flush. The cost is a map read and one row
per event, which is what an immediate "no" is worth. Miss it, and the answer to
"when exactly does their open tab stop being an admin?" is "when they close it".

## What the tests will not tell you

`DESIGN.md` in the exercise is part of the deliverable. Six questions, a
paragraph each, in your own words: why sessions, why this chain order, why one
policy with four call sites, why SSE, why a queue in your database, and what you
would do next.

Write them as you go. A decision you cannot write down is usually one you have
not made, and "the tests pass" is an answer to a question nobody asked in a
design review.

## What this capstone leaves out

Named so you know they are missing, not forgotten: **fan-out across replicas**
(this hub is one process, so a client connected to instance A never sees an
event published on B — the answer is a shared broker or sticky routing); CSRF
tokens for the state-changing routes; multi-factor and passkeys; a limiter
shared between instances; metrics and alerting on dead-letter rate and queue
depth; and the whole of deployment, which the containers pack takes.

## Exercise

Open [`exercise/`](exercise/) — one `board` package over SQLite. Read
`server.go` first: it is the map of the service.

Provided, and worth reading before you write anything: `db.go`, `store.go`,
`queue.go` (queue and worker pool), `auth.go` (hasher, sessions, subject),
`login.go`, `hub.go`, `edge.go` (limiter, decoder, headers, ETag, timeouts),
`respond.go`, `wire.go` (config, wiring, and the janitor that keeps the two
in-memory maps from growing forever), `clock.go`.

Yours, in the order the tests will make sense in:

```
policy.go      the rule table, and the scope derived from it
middleware.go  Authenticate, Require, Enforce, RateLimit, the key
server.go      the route table's gates, and the chain order
tasks.go       list, read, create, update, role change
jobs.go        the notify handler, and which failure it calls permanent
stream.go      the authorized event stream
DESIGN.md      six answers
```

The starter compiles and runs. It is also the version of this service that a
hurried team ships: no gate, no scope, no filter, no queue, no announcement.
Every hole is a failing test.

Acceptance criteria:

1. `DefaultRules` says exactly this: members, auditors and admins may list;
   all three may read, members only their own; members and admins create;
   members and admins update, again only their own; only admins set roles;
   **nothing grants `task:delete`**, so the route that exists is refused for
   everyone, admins included, and the row survives.
2. `Policy.ScopeFor` derives the readable scope from the rule table, not from a
   role literal: change the table and the scope changes with it.
3. `Authenticate` attaches a `Subject` and rejects nobody; a cookie naming a
   session that no longer exists is cleared on the way past. Session expiry is
   enforced server-side against the injected clock.
4. `Require(action)` refuses before the handler runs: **401 `unauthenticated`**
   for the anonymous, **403 `forbidden`** for the denied, and no response body
   ever contains a decision reason.
5. `Enforce` completes the decision on the loaded object. A member may not read
   or update another member's task; an auditor may read any and update none; an
   admin passes by exemption; a denied request leaves storage byte-for-byte
   unchanged and publishes nothing. A task that does not exist is 404. The
   object check comes **before** the body is decoded: a denied `PATCH` never
   reads a byte of it.
6. `GET /tasks` returns only the rows the caller may read, restricted **in the
   query**, and pages by keyset: `?limit=` defaults, clamps to `MaxPageLimit`
   and 400s on nonsense; `?cursor=` 400s on anything unreadable; `next_cursor`
   appears only when a further page exists; no row is skipped or repeated.
7. That response carries an `ETag`, a `Cache-Control` that no shared cache may
   act on, and a `Vary` naming `Cookie`; `If-None-Match` gets **304** with no
   body and the same tag; a new task invalidates it.
8. `POST /tasks` decodes strictly (unknown field 400 naming the field, blank or
   over-long title 422, oversized body 413, wrong media type 415, a second JSON
   value 400) and takes ownership from the session — a body cannot set it.
9. `POST /tasks` writes the task and enqueues `notify:<task-id>` in **one**
   transaction: if either fails, neither row exists, the response is 500, and no
   event reaches a subscriber. On success it publishes `task.created` **after**
   the commit; `PATCH` publishes `task.updated` the same way.
10. `NotifyHandler.Handle` writes one notification and returns the
    `task.notified` event for the pool to publish after the commit — so a
    redelivered job, which the pool's marker keeps out of the handler entirely,
    writes no second row and announces nothing. A payload naming a task that
    does not exist dead-letters immediately (`ErrPermanent`) with the cause
    recorded, one attempt spent, no notification and **no processed marker left
    behind**.
11. `GET /events` streams `text/event-stream` with `Cache-Control: no-cache` and
    `X-Accel-Buffering: no`, subscribes before writing, flushes every frame,
    advertises `retry:`, and answers 503 when the hub is closed. It keeps
    `NewHTTPServer`'s `WriteTimeout` — the hardening lesson's four timeouts stay
    four — by resetting its own deadline before every frame
    (`StreamFrameTimeout`) instead of the route going unbounded.
12. The stream writes only the events its subscriber may read — live and
    replayed alike — replays from `Last-Event-ID`, sends `ResyncEvent` when the
    backlog cannot reach that far, heartbeats on every tick of a ticker from
    the injected `Clock`, and unsubscribes on every exit path. The identity is
    re-resolved per event, not captured: a logout or a role change **ends the
    target's open streams**, on the next event or the next heartbeat.
13. Rate limiting keys on the account when there is one and the client address
    otherwise: two accounts never share a bucket, a refusal is **429** with a
    whole-second `Retry-After` of at least 1, and the request body is never
    read. The security headers reach every response, including that one. Both
    in-memory maps are swept: `Sweep` evicts buckets idle for at least the
    refill window and sessions past expiry, and `StartJanitor` runs it off the
    injected `Clock` — a service that never starts it hands anyone with a range
    of addresses a memory-exhaustion vector.
14. `POST /users/{id}/role` is admin-only and destroys every session belonging
    to the target — their old cookie no longer authenticates, their open event
    streams end, the actor's session still works — and a denied attempt changes
    no role.
15. `DESIGN.md` is answered, `go test -race ./...` is green, and the code is
    `gofmt`-formatted.

```sh
cd exercise
go test -race ./...
```

Nothing in the suite sleeps, and nothing in your code may either: every rule
about time here — session expiry, lease expiry, refill, heartbeats — is "compare
a stored value against `Clock.Now()`". If you reach for `time.Sleep` to make a
test pass, the design is wrong, not the test.

Suggested order: `policy.go` and `middleware.go` and the gates in `server.go`
first (most of the suite is behind them), then `tasks.go`, then `jobs.go`, then
`stream.go`.

## Further reading

- [OWASP Top 10 — A01:2021 Broken Access Control](https://owasp.org/Top10/A01_2021-Broken_Access_Control/)
- [OWASP — Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [microservices.io — Transactional outbox](https://microservices.io/patterns/data/transactional-outbox.html)
- [RFC 9111 §4.1, §5.2 — Vary and Cache-Control directives](https://www.rfc-editor.org/rfc/rfc9111#name-calculating-cache-keys-with)
- [WHATWG HTML — Server-sent events](https://html.spec.whatwg.org/multipage/server-sent-events.html)
- [pkg.go.dev — net/http.ResponseController](https://pkg.go.dev/net/http#ResponseController)
