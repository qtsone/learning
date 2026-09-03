# Tutor notes — Web Services Capstone

## Where the learner is

Last lesson of the web-services pack, and the first time they assemble it.
Individually they can hash a password, write a rule table, run a hub, bound a
request, batch a query, drain a queue and page a feed. What they have never done
is make those agree with each other under one `main`.

Expect the failures to cluster at the seams rather than inside any component:
the stream that is not filtered, the event published before the commit, the
listing scoped in Go instead of in SQL, the role change that leaves live
sessions intact. None of those look like bugs while you are writing them; each
one is a component behaving exactly as designed.

Two personalities show up here. The **fast one** gets the suite green in five
hours and cannot say why the limiter sits inside authentication — for them the
grading weight is `DESIGN.md` and the grilling, not the tests. The **stuck one**
tries to hold all seven lessons in their head at once and freezes; for them,
insist on the suggested order (policy → middleware → gates → tasks → jobs →
stream) and treat each green test as territory taken.

This is also the lesson where "I asked an assistant and it wrote the handler" is
most likely and least survivable. Every design question below is answerable only
by someone who made the decision.

## Common misconceptions

- **"The hub should check permissions."** It should not: it would be a second
  copy of the policy, and it would have to know what every event means.
  Mechanism fans out; policy filters per subscriber.
- **"The route gate covers the stream."** It covers *reaching* the stream. Which
  events go down it is a per-event decision the gate cannot make.
- **"Filtering rows after the query is the same thing."** Same answer, worse
  failure mode: a filter can forget a row, and it loads data the caller may not
  see into the process that is about to serialise it.
- **"ScopeFor is just: auditors and admins see everything."** That sentence is
  the bug. It is a copy of the rule table that will not change when the table
  does. Make them explain what happens when the auditor role is narrowed.
- **"Publish inside the transaction so it is atomic."** There is no transaction
  that spans a database and a socket. Publishing before the commit announces
  work that may never exist, and an event cannot be rolled back.
- **"Rate limiting must be outermost."** It must be outside anything expensive.
  Which side of authentication it sits on is a trade they must be able to argue,
  not a rule.
- **"At-least-once is a queue bug."** It is the contract. The consumer's
  idempotency is the design, and in this service it covers the *announcement*
  as well as the row.
- **"The dedup marker can go in before the work."** Only in the same
  transaction — which is exactly what `Pool.run` does, one statement before it
  dispatches. A marker written outside it survives a failed effect and turns the
  next retry into a silent skip.
- **"Each handler should mark its own job."** Then dedup is a rule every future
  job kind has to remember, and forgetting it is invisible until a user is told
  twice. The pool marks once, for all kinds, and the skipped handler is also why
  a duplicate publishes nothing.
- **"304 means the handler was skipped."** It means the body was. The query and
  the authorization still ran; the saving is bandwidth and client rendering.
- **"`Cache-Control: private` is enough."** Without `Vary: Cookie`, a cache that
  keys on the URL alone can still serve one session's page to another.
- **"A role change takes effect immediately because it is in the database."**
  Only for sessions that have not been minted yet. Live sessions carry the old
  answer until something ends them.
- **"Destroying the session revokes everything."** It revokes the next
  *request*. An open stream is not one: it made its request hours ago and lives
  off the `Subject` the middleware attached then. Capturing that subject is the
  seam bug of this lesson — the fix is to re-resolve from the session id per
  event and per heartbeat.
- **"Frame's validation is dead code here."** Today, yes: every field is
  server-generated. It guards the event they add next year from a title, and a
  bare `\r` is enough to inject a field into somebody's connection.
- **"Sessions are the safe default, so tokens are wrong."** The pack's argument
  is about revocation and deployment shape, not about safety. Push back on
  cargo cult in either direction.

## Grilling points

Ask in their own words (quiz.json has the core set; these go deeper):

- "Walk me through a `POST /tasks` from the first byte on the wire to the event
  on another user's screen. Name every layer it passes and what each could
  refuse."
- "Move the rate limiter outside `Authenticate`. What breaks, what improves, and
  what would make you actually do it?"
- "I am an auditor. Show me the four places in your code that decide what I can
  see, and prove they cannot disagree."
- "Your commit succeeds and the process is killed before `Publish`. What does a
  connected client see, and how does it recover? Now the other order: what would
  a client see then?"
- "A worker is redelivered a job whose effect already committed. Trace the exact
  path. What does the owner's browser receive?"
- "Somebody puts a CDN in front of this service. Which of your responses is now
  dangerous, and which header saves you?"
- "The CEO's account is compromised and you demote them at 03:00. When exactly
  does their open tab stop being an admin — and what would the answer have been
  with a 12-hour JWT?" (Two answers wanted: the open `/events` connection ends
  on its next event or next heartbeat, whichever lands first, because the
  stream re-resolves; and the JWT is an admin until it expires.)
- "Somebody moves the `DecodeJSON` in `handlePatchTask` back above the load and
  the `Enforce`. Every test but one still passes. What did they just sell, and
  to whom?"
- "One member opens the stream in seven tabs on HTTP/1.1. What happens, and why
  is that not your bug to fix in Go?"
- "You have two instances behind a load balancer tomorrow. Name everything in
  this service that becomes wrong, in order of how quickly a user notices."
- "Delete the `notify:` prefix and use a UUID for job ids. What stops working,
  and when would you find out?"

## Grading rubric

- **A** — Suite green. The four policy call sites all funnel through `Check`,
  and `ScopeFor` is derived rather than written out per role. The chain order is
  argued, including what it costs. Publish-after-commit is deliberate and they
  can state the failure it accepts. They can say why the marker shares the
  handler's transaction, and why the pool holds it rather than each handler —
  including that this is what makes a duplicate announce nothing. Caching
  headers are explained as an authorization question.
  Role change destroys sessions, the target's open streams end with them, and
  they can contrast both with rotation and with a JWT. `DESIGN.md` reads like an
  engineer's, not a summary of the lesson.
- **B** — Suite green; one seam understood mechanically rather than in
  principle — typically the stream filter ("the test wanted it") or `Vary`.
  Minor smells: policy consulted twice per request, an event built in two
  places, a denial reason leaking into a log line at the wrong level.
- **C** — Green only after heavy hinting, or `DESIGN.md` restates the lesson
  instead of defending choices. Pass only if a focused remediation lands on the
  two non-negotiables: **no unauthorized event ever leaves the stream**, and
  **nothing is announced that is not committed**.
- **Fail** — Red suite; or green by way of a shortcut that guts the point: the
  scope hard-coded per role, `Enforce` returning true, the stream filtered by a
  string compare on the event name, dedup done outside the transaction, sessions
  left alive after a role change, the stream's subject captured once at connect
  time, or an inability to explain any single one of the four call sites.
  Remediate; this is the lesson the pack is graded on.

## Remediation ladder

1. "Read the failing test's name aloud. Which two lessons does it sit between?"
   Nearly every failure here is a seam, and naming the pair usually locates it.
2. Narrow to the layer, not the fix: "is this a policy question, a transaction
   question, or an ordering question?" Then point at the one file.
3. Name the rule, never the code. "Every event carries the attribute the policy
   needs — what would you ask the policy, and with what?" Or: "something is
   announced that might not exist. Which two lines are in the wrong order?" Or:
   "a redelivery is supposed to be free. What does yours currently cost the
   user?"
4. For the genuinely stuck, walk the shape verbally — "subscribe, headers,
   flush, retry frame, replay filtered, then a select over context, subscriber
   and ticker" — and have them type it. Never paste `solution/`. If they have
   seen it, require a from-scratch re-implementation of that file plus an
   explanation of every decision in it.

For a learner who is green but hollow, skip the ladder: go to `DESIGN.md` and
make them defend one answer at a time. Rewriting a weak answer teaches more here
than another passing test.

## After passing

Preview: "You have shipped a service that authenticates, authorizes,
streams, queues and refuses — and you can defend every one of those choices. The
pack ends here. What it never faced is a second instance, a real deployment or a
pager: the containers pack takes the first two, and the expert stage takes the
service as a whole into operations."
