# Design review

A reference set of answers. A learner's version does not have to match this one
— it has to be *theirs*, specific to the code they wrote, and it has to name the
alternative it rejected.

## 1. Sessions, not tokens

The clients are browsers talking to this service and nothing else, and the
property that matters is revocation: a role change here destroys the target's
sessions with one map delete, and the next request is anonymous — including the
next event on an open stream, which re-resolves its subject and ends rather than
serving out the old privilege for the rest of the connection. With a
stateless JWT the same change is invisible until the token expires, so I would
be choosing between a long window of stale privilege and a token so short-lived
that I have rebuilt sessions with extra steps. Sessions also give the streaming
endpoint its authentication for free, since `EventSource` cannot set headers and
carries the cookie.

What would change my mind: another service having to verify identity without
calling me (service-to-service, or a mobile client plus a partner API), or
identity that is not mine to hold — at which point OIDC issues the identity, I
validate the `id_token` with a maintained library, and I still mint my own
session from it.

## 2. The chain order

`SecurityHeaders → Authenticate → RateLimit → Require(action) → handler`.

`SecurityHeaders` is outermost so every response carries it, including the 401s,
403s and 429s no handler produced — headers set inside the chain are missing
from exactly the responses an attacker generates most. `Authenticate` sits above
`RateLimit` because an account is a far better limiter key than an address (NAT
puts thousands of people behind one, and a single user can move between many),
and you cannot key on an account before you know which one it is. The route gate
is innermost and per route, so a denial costs no handler work and no route can
be registered without naming an action.

What it makes worse: a request that ends in a 429 still pays for a session
lookup. That is a map read plus one indexed row, against the body read and JSON
decode the limiter still refuses to do — but if the session store ever became a
network hop, I would revisit it and put a cheap address-keyed limiter outside
authentication as well.

## 3. One policy, four call sites

`Check` is the only function that interprets a rule. The route gate, the object
check, `ScopeFor` and the stream filter all call it; what is distributed is the
*invocation*, because only the handler has the object and only the stream has
the event. The rule itself is never duplicated, so a change to the table reaches
all four at once.

Writing "auditors see everything" in the listing would break that: the day
auditors lose their ownership exemption, three call sites change and the listing
silently keeps handing out every row. That is why `ScopeFor` asks the policy
about a task owned by somebody else instead of testing the role.

The one a new teammate forgets first is the stream filter — nothing about
`hub.Publish` suggests authorization, and the leak shows up in a UI event rather
than an HTTP response, where nobody is looking.

## 4. SSE, not WebSockets

Everything flowing is server → client: created, updated, notified. SSE is plain
HTTP, so the session cookie, the middleware chain, the logging and the security
headers all work unchanged, and reconnection with `Last-Event-ID` is the
browser's job rather than mine. A WebSocket would add a dependency, a second
authentication story (no preflight, so the `Origin` check is mine to write), a
reader/writer goroutine discipline per connection, and proxies that must
understand upgrades — for a channel the client never writes to.

I would switch when the client becomes a peer: typing indicators, collaborative
editing, anything where a round trip per message over the same connection
matters.

A client that falls behind is evicted: its buffer is bounded, `Publish` never
blocks, and the eviction closes its channel. That is humane only because
reconnection is automatic and ids make replay possible — the client learns it
fell behind instead of quietly missing rows, and when the backlog cannot reach
back it gets a `resync` event rather than a partial replay that would be a
silent lie.

## 5. A queue in the database

The task insert and the job insert are one commit, so there is no window in
which a task exists with no work queued, and no window in which a worker can
claim a job for a task that was never written. With a broker there is no such
commit: whichever write goes first, a crash after it leaves the system wrong,
and no ordering fixes it.

I would move to a broker when I need fan-out to independent consumers,
retention and replay, per-key ordering, or more polling throughput than this
database should be spending on a queue. At that point the outbox stays and a
relay publishes from it — which is why the consumer must tolerate duplicates
either way.

The job id is `notify:<task-id>` because it names the *work*: a producer that
retries its own request gets a primary-key conflict instead of a second
notification, and the consumer's dedup ledger keys on something meaningful. A
UUID would identify the delivery and deduplicate nothing.

The pool publishes after the commit because a transaction can be rolled back and
a line on somebody's screen cannot. Publishing inside the handler would announce
work that may still fail — and the handler cannot know whether its own
transaction is about to commit.

## 6. What you would do next

CSRF tokens for the state-changing routes (`SameSite=Lax` is a mitigation, not a
cure); a bounded fan-out across replicas, since this hub is one process and a
client connected to instance A never sees an event published on B; and metrics
plus alerting on the dead-letter rate and queue depth by state, because a
background job that stops working is currently silent.

First: the fan-out, because it is the one that is wrong the moment there are two
instances, and it is the one users notice as "the page just stops updating".
