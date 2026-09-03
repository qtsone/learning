# Tutor notes — Background Jobs

## Where the learner is

Sixth lesson of the web-services focus pack, after authentication,
authorization, realtime, hardening and GraphQL. They have S5's production
server (middleware, graceful shutdown, `log/slog`), S5's `database/sql` over
`modernc.org/sqlite`, S5's fake clocks and S4's SQL and TDD. They have *not*
done S6 systems design, so avoid CAP/consensus vocabulary — "leader election"
appears in the scheduling section and is explained in place as "one replica
holds a lock row".

The lesson's intellectual move is a shift from *within one process, one
request* to *across processes, across crashes*. Everything they have built so
far fails by returning an error to somebody who is waiting. Here the failure
mode is silence: work that vanished, or work that happened twice, and nobody
is watching. If they leave able to say "delivery is at least once, so the
consumer must be idempotent, and the ledger commits with the effect", the
lesson landed.

Watch for two shapes. The first is the learner who thinks a queue is a data
structure — they will design an elegant in-memory channel and miss that the
whole point is a row that survives the process. The second is the learner who
believes exactly-once is achievable with enough care; do not let that one
slide, it is the root of most real-world duplicate charges.

## Common misconceptions

- **"`go doWork()` is a background job."** Ask what happens on deploy, on
  panic, on a traffic spike, and what `r.Context()` is doing to the goroutine
  the moment the handler returns. The table at the top of the lesson is the
  answer, but make them build it themselves.
- **"Commit, then publish — the window is tiny."** Tiny and constant. Ask how
  many orders a day, times how many deploys, and whether they would notice.
  Then ask which of the two orderings loses jobs and which invents them
  (publish-first invents phantom jobs).
- **"Exactly-once delivery."** There is a gap between "the effect happened" and
  "the queue knows", and nothing closes it across two systems. What is
  achievable is exactly-once *effect*, by making the effect idempotent — and
  when the effect is in your database, by committing it with its marker.
- **"The dedup marker should be written before/outside the handler's
  transaction."** Then a failed attempt leaves a marker and the retry skips the
  work. Silent data loss that looks like success. The test
  `TestMarkProcessedRollsBackWithItsTransaction` is exactly this.
- **"A random UUID is a fine job id."** It deduplicates deliveries, not work.
  A producer that retried its own request gets two rows and the customer gets
  two e-mails. Natural keys (`"send-confirmation:"+orderID`) turn that into a
  primary-key conflict.
- **"SELECT then UPDATE is fine, the window is microseconds."** Two workers,
  one row, one race. Push until they reach a single conditional statement (or
  `FOR UPDATE SKIP LOCKED`).
- **"Increment attempts when the handler returns an error."** A job that kills
  the worker never returns. Claim-time increment is what bounds a poison job.
- **"Retry forever; giving up loses work."** Dead-lettering does not lose the
  work, it stops burning the system on it and hands it to a human. Deleting
  the row would lose it.
- **"Jitter is a micro-optimisation."** It is the difference between one
  outage and a self-sustaining retry storm. Ask what happens at 09:00 when a
  provider comes back and ten thousand jobs with identical schedules retry
  together.
- **"Graceful shutdown is what stops jobs being lost."** The lease is. Graceful
  shutdown avoids the redelivery; a `kill -9` is still survivable.
- **"`time.Ticker` gives me cron."** On one replica. Ask what three replicas
  do, and what happens on the night the single replica is being restarted.
- **"Sleep in the test until the lease expires."** The clock is injected for
  precisely this. A sleeping test is slow, flaky under `-race`, and asserts
  wall-clock behaviour that is not the contract.

## Grilling points

- "Draw the timeline where the two-step version loses a job, and the one where
  it invents one. Which is worse to debug, and why?"
- "Your worker dies mid-job, hard. Walk me through what happens next, minute by
  minute, and name the column that makes it happen."
- "Your lease is 30 seconds and a handler takes 45. What goes wrong, and what
  are your options?" (Redelivery of a live job → idempotency saves you; extend
  the lease from inside; raise the lease; split the job.)
- "The customer got two confirmation e-mails. Give me three distinct ways that
  could have happened." (Producer retried the enqueue; lease expired under a
  slow handler; effect committed and the ack was lost.)
- "Why is the processed-jobs insert inside the handler's transaction and not
  before it? What breaks if you move it out?"
- "Your handler charges a card through Stripe. The transaction trick is not
  available. What do you do instead?" (Their idempotency key = your job id;
  otherwise choose duplicate-or-loss explicitly.)
- "Your queue has 40,000 ready jobs and four workers. Which number do you put
  on the dashboard, and why not queue depth alone?" (Age of the oldest ready
  job — depth without arrival rate says nothing about lateness.)
- "Somebody sets `max_attempts` to 1000 'so nothing is ever lost'. Argue them
  down."
- "You cancel the pool's context and pass that same context to the handler.
  What does the failure look like in production?" (Half-written jobs, retry
  storms on every deploy.)
- "Two replicas, one nightly digest. Give me two designs that send it once."
- "When would you move this off SQLite/Postgres onto a real broker? Give me a
  number or a property, not a vibe."

## Grading rubric

- **A** — All tests pass under `-race`. Claim is a single conditional
  statement; `MarkProcessed`, the handler's effect and `Complete` share one
  transaction, and the rollback path is deliberate; `ProcessOne` uses
  `context.WithoutCancel` for the claimed job only; `errors.Is` distinguishes
  `ErrNoJob` and `ErrPermanent`; backoff reads the injected source. The
  learner explains at-least-once and the dedup ledger unprompted, and can name
  what the lease costs when it is mis-sized.
- **B** — Tests pass; the design is sound but the seams are rough: the
  processed marker committed separately from the effect, `Complete` on the
  pool instead of the transaction, attempts incremented on failure rather than
  on claim, or a `strings.Contains` where an error sentinel exists.
  Explanations mostly right with one misconception still live.
- **C** — Tests pass only after substantial hinting, or the learner treats the
  transaction boundaries as incantation and cannot say what rolls back with
  what. Pass only if a time-boxed remediation lands; otherwise iterate.
- **Fail** — Tests failing, or the learner still believes commit-then-publish
  is safe, or that a queue gives exactly-once delivery. Both are load-bearing
  for the capstone; remediate rather than advance.

## Remediation ladder

1. "Run one failing test with `-run` and read the message aloud — what did it
   expect, what did it get?" The failure text names the concept every time.
2. Move to the state, not the code: "Sketch the `jobs` row before the claim,
   after the claim, and after the crash. Which column is different, and who
   changes it back?"
3. Name the tool without the shape: "One statement — `UPDATE … WHERE id = (SELECT
   … LIMIT 1) RETURNING …`"; "`INSERT … ON CONFLICT DO NOTHING` plus
   `RowsAffected` tells you whether you were first"; "`context.WithoutCancel`
   drops cancellation but keeps the values."
4. Walk one path verbally end to end — claim, begin, mark, handle, complete,
   commit, and where each failure branches — and let them type it. Only write
   code beside them if step 3 stalls twice.

If they are stuck on the pool rather than the store, have them make
`ProcessOne` green first: every worker test except the last two drives it
directly, and `Run` is then a loop around a function that already works.

## After passing

Preview: "Next is API performance — caching, pagination and the N+1 problem
from the other side. Some of it is the same move you just made: the fastest
request is the one that does less work while somebody waits."
