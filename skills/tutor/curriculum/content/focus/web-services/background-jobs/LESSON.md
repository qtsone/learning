# Background Jobs

> `focus.web.background-jobs` · ~4-5h · Stage: Focus: Web Services

## Objectives

By the end of this lesson you can:

- Explain why long-running work must move out of the request path, and what a
  job queue adds over firing a bare goroutine.
- Implement a producer and a worker-pool consumer in Go that process jobs from
  a queue, with graceful shutdown.
- Implement retry with exponential backoff and jitter, and route permanently
  failing jobs to a dead-letter destination.
- Explain at-least-once delivery and implement an idempotent consumer that is
  safe under duplicate deliveries.
- Explain the transactional outbox pattern, and why writing to the database and
  enqueueing in one transaction prevents lost and phantom jobs.

## The request path is a budget

A handler has a budget. The client is waiting, your `WriteTimeout` and
`TimeoutHandler` deadline from the hardening lesson are counting, and every
goroutine parked inside a handler is a connection you cannot serve anyone else
with. Charging the checkout, writing the order and returning 201 fits. Sending
the confirmation e-mail does not: it depends on a third party that is sometimes
slow and sometimes down, and its failure is not something to explain to a
customer whose payment already succeeded. So the work moves, and the first
attempt is always the same:

```go
func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	// ... write the order ...
	go s.sendConfirmation(order)   // fire and forget
	respond(w, http.StatusCreated, order)
}
```

Count what that line gives away:

| A bare goroutine | A job queue |
|---|---|
| dies with the process — a deploy, an OOM kill, a panic elsewhere | the job outlives the process; it is a row |
| one failed send is gone | retried on a schedule you chose |
| no record it ever existed | a row you can query, count and alert on |
| unbounded: a traffic spike is an unbounded number of in-flight sends | bounded by the number of workers |
| no backpressure on the dependency | the queue *is* the backpressure |
| `r.Context()` is cancelled the moment you respond, so anything using it stops | the job carries its own context |

That last one bites hardest and quietly: the request context is cancelled when
the handler returns, so a goroutine that passes it to an HTTP client or a
database call is cancelled seconds after you fired it, and handing it
`context.Background()` fixes the cancellation while keeping every other problem.
A queue is not a big idea — it is *durability plus a retry policy plus a
concurrency limit*, and you already have the durable part: a database.

## Two writes, one crash

Here is the bug this lesson is really about.

```go
tx.Exec("INSERT INTO orders ...")
tx.Commit()                       // committed
queue.Publish(confirmationJob)    // ← the process dies here
```

The order exists and no e-mail will ever be sent. Nothing failed. Nothing
retried. Nothing logged. You find out from the customer.

Swap the order and it is not better:

```go
queue.Publish(confirmationJob)    // published
tx.Commit()                       // ← fails: constraint violation, disk, crash
```

Now a worker picks up a job for an order that does not exist — a **phantom
job**. It will fail, retry, fail again on its whole budget, and dead-letter.

There is no safe ordering of "write to the database" and "write to the broker":
they are two systems and you cannot commit to both at once. (Distributed
two-phase commit exists; nobody sane uses it for this.) The fix is to stop
trying — **write the job into the same database, in the same transaction, as the
data that justifies it.**

```go
tx, _ := db.BeginTx(ctx, nil)
defer tx.Rollback()
tx.ExecContext(ctx, "INSERT INTO orders ...")
store.Enqueue(ctx, tx, confirmationJob)   // same tx
return tx.Commit()                        // both, or neither
```

That table is the **outbox**, and this is the transactional outbox pattern. One
commit, no gap to crash into. If the transaction rolls back, the job goes with
it, so a phantom job is not possible either.

When your jobs must reach a broker (Kafka, SQS, RabbitMQ) rather than your own
workers, one extra piece appears: a **relay** — a small loop that reads
unpublished outbox rows, publishes them, and marks them published. It can crash
mid-publish and republish on restart, which is why the *consumer* must tolerate
duplicates; more on that below. When the queue lives in the same database as
your data, as it does here, the outbox and the queue are the same table and the
relay is just the worker — a real simplification, not a shortcut, and the reason
"just use your database" carries a service further than people expect.

## The queue is a table

```sql
CREATE TABLE jobs (
	id           TEXT PRIMARY KEY,   -- chosen by the producer; the dedup key
	kind         TEXT NOT NULL,      -- which handler runs it
	payload      TEXT NOT NULL,      -- what it needs, not a pointer to state
	state        TEXT NOT NULL,      -- ready | running | done | dead
	worker       TEXT NOT NULL,      -- who holds the current claim
	attempts     INTEGER NOT NULL,
	max_attempts INTEGER NOT NULL,
	run_at       INTEGER NOT NULL,   -- invisible before this: delay and backoff
	lease_until  INTEGER NOT NULL,   -- a claim expires; a crash does not hide a job
	last_error   TEXT NOT NULL
);
```

Four columns carry the whole design: `state` and `run_at` decide what is visible,
`lease_until` decides what a crash costs, `attempts` against `max_attempts`
decides when you stop. Two notes on `payload`: keep it **self-contained enough to
run** but small — the order id, not the rendered e-mail — and treat it as a
schema you version, because the job you enqueue today may be executed by the
*next* deployment's code, which is an API compatibility problem with a longer
window and no client you can call.

**Is a database queue good enough?** Up to a few hundred jobs a second, on a
database that is not already your bottleneck, yes — and you get transactions,
one operational thing to run, and SQL to inspect it with. You reach for a
dedicated broker when you need fan-out to many independent consumers, retention
and replay, throughput past what your primary database should spend on polling,
or ordering guarantees per key. Those are the reasons; "queues belong in Redis"
is not one. The exercise builds it over `modernc.org/sqlite` — the pure Go
SQLite from S5, no cgo, no server — because that makes the outbox transaction
real. Everything else is standard library.

## Claiming: one statement, or two workers run one job

The naive consumer does this:

```go
job := db.QueryRow("SELECT ... WHERE state='ready' LIMIT 1")   // worker A and B
db.Exec("UPDATE jobs SET state='running' WHERE id=?", job.ID)  // both of them
```

Between those two statements another worker reads the same row. Now the job runs
twice, concurrently, and the second copy is not a duplicate you can detect later
— it is a race. Finding a job and taking ownership must be **one statement**:

```sql
UPDATE jobs
   SET state = 'running', attempts = attempts + 1, lease_until = ?, worker = ?
 WHERE id = (SELECT id FROM jobs
              WHERE (state = 'ready'   AND run_at      <= ?)
                 OR (state = 'running' AND lease_until <= ?)
              ORDER BY run_at LIMIT 1)
RETURNING id, kind, payload, attempts, max_attempts;
```

In Postgres the inner select gets `FOR UPDATE SKIP LOCKED`, the same idea with a
lock: skip rows another worker holds instead of queueing behind them.

Note where `attempts` is incremented: **at claim time**, not when a handler
returns an error. A job that crashes the worker never returns anything, so if
only clean failures counted, a payload that segfaults your image resizer would
be redelivered forever — taking the queue down with it every time.

## Leases: the price of a dead worker

The second `OR` in that query is the whole reason this design survives machines
dying. A claim is a **lease**: I own this job until `lease_until`. When a worker
crashes, nothing tells the queue — no error, no disconnect it would notice — the
lease simply expires and the job becomes claimable again. Brokers call the same
idea a *visibility timeout*. Sizing it is a real trade-off with no safe default:

- **Too short** and a healthy-but-slow job is redelivered while still running,
  so it runs twice concurrently — the case your idempotency must survive.
- **Too long** and a crashed worker's job sits invisible for that long: on a
  30-minute lease, a deploy that kills a worker mid-job delays it by half an
  hour.

Size it against the p99 duration of your slowest handler for that queue, and for
genuinely long jobs extend the lease from inside the handler (a heartbeat)
rather than setting one long enough for the worst case. One honest gap remains:
the queue cannot tell "crashed" from "slow", so after a lease expires the
original worker may still be alive and about to finish. Systems that must not
tolerate that add **fencing** — the claim hands out a token, and writes carry it
so a stale owner's write is rejected. Naming it is enough here.

## At-least-once, and why "exactly-once" is a sales word

Look for the moment a job is finished. The handler did its work, and now the
worker updates the row to `done`. Between those two things is a gap, and a crash
in that gap means the work happened and the queue does not know. It redelivers.
You can make the gap smaller; you cannot make it disappear, because "do the
work" and "record that the work is done" are two operations, and if the work
happens in another system there is no transaction spanning both. So delivery is
**at least once**, always, and the interesting question moves to the consumer:
*what happens when this job runs twice?*

An **idempotent consumer** is one where running the same job twice has the same
effect as running it once. Three ways to get there, in order of preference:

1. **The effect is naturally idempotent.** `UPDATE orders SET status='paid'
   WHERE id=?` is safe at any multiplicity. Design for this when you can.
2. **The effect is in your database, so join its transaction.** Insert a row
   into a dedup ledger keyed on the job id *in the same transaction as the
   effect*:

   ```go
   tx, _ := db.BeginTx(ctx, nil)
   first, _ := store.MarkProcessed(ctx, tx, job.ID)  // INSERT ... ON CONFLICT DO NOTHING
   if first {
       handler.Handle(ctx, tx, job)                  // the effect, same tx
   }
   store.Complete(ctx, tx, job.ID)
   tx.Commit()                                       // all of it, or none of it
   ```

   A redelivery of committed work gets `first == false`, skips the effect and
   finishes the job. A failure rolls back the marker with the effect, so the
   retry is a real retry — easy to get backwards, and it silently drops work.
3. **The effect is somebody else's system.** You cannot join their transaction,
   so you send *them* a key: Stripe's `Idempotency-Key`, an `INSERT ... ON
   CONFLICT` at the far end, a message id the mail provider deduplicates — the
   job id is that key. Where the third party offers nothing, you choose between
   a possible duplicate and a possible loss, deliberately and per job kind.

Two practical notes. The dedup ledger grows forever unless you prune it; prune
by age, keeping entries longer than any job could be retried. And dedup is only
as good as the id: `"send-confirmation:"+orderID` identifies the *work*, so a
producer that retried its own request enqueues a primary-key conflict instead of
a second e-mail, where a random UUID identifies the *delivery* and deduplicates
nothing.

## Retries: backoff, and why jitter is not optional

A transient failure is worth retrying: an SMTP timeout, a 503, a deadlock. The
policy has three parts. **Exponential backoff** — wait `base * 2^(attempt-1)`,
capped: 1s, 2s, 4s, 8s … up to a ceiling, because retrying a struggling
dependency at the rate that just failed is how you keep it down.

**Jitter.** When one outage fails a thousand jobs at once, an identical backoff
schedule retries all thousand at the same instant — and again at the next
boundary, and the next. The retries themselves become the outage. Randomising
each delay spreads them: *full jitter* is `random(0, d)`, the best spread, where
a retry can return almost immediately; *equal jitter* is `d/2 + random(0, d/2)`,
which keeps a floor under the delay and is what the exercise implements. Jitter
is randomness, and randomness inside code you want to test is a coin flip you
cannot re-run — so inject the source (a `func(n int64) int64` here) as you
inject the clock, and tests assert the policy instead of sampling it.

**A budget.** `max_attempts` is a policy decision per job kind, not a constant
of nature: five for an e-mail, one for a charge you cannot safely repeat, more
for something cheap and important. And distinguish the failure no retry can fix
— a payload that will never parse, a record that was deleted, a 400 from an API
— from the transient kind. In Go that distinction is an error value: wrap
`ErrPermanent`, check with `errors.Is`, and stop immediately instead of burning
the budget and the wait.

## Dead letters

When the budget runs out, the job stops being the queue's problem and becomes a
person's. It moves to a **dead-letter** state (or table, or queue) and is not
retried. Three things make that useful:

- **Keep the cause.** `last_error` and `attempts` are what somebody reads at
  09:00. A dead-lettered job with no error text is a mystery.
- **Alert on the count, not on each job.** Dead letters arriving is normal; the
  *rate* changing is the signal. From S5's observability lesson: a counter by
  kind, and queue depth by state.
- **Make replay possible.** After the bug is fixed, "set these back to ready"
  should be one statement — which it is, when the queue is a table.

Deleting failed jobs instead of dead-lettering them is the same mistake as
swallowing an error: the system loses work quietly, which is what moving the
work out of the request path was meant to prevent.

## Graceful shutdown

A worker is killed on every deploy. What you want is what S5's HTTP server did
for connections: **stop accepting, finish what you have.**

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
pool.Run(ctx)   // returns when ctx is done and every in-flight job is finished
```

The subtlety is the context. Cancelling it must stop *claiming*, but a job
already claimed should finish: if it inherits the cancellation, its database
calls fail halfway and a clean shutdown becomes a pile of retries. Go 1.21 added
exactly the tool:

```go
// Claiming honours ctx; the claimed job does not.
job, err := p.Store.Claim(ctx, worker, p.Lease)
...
return p.run(context.WithoutCancel(ctx), job)
```

`context.WithoutCancel` returns a context carrying the same values with the
cancellation dropped. In production you would bound the drain (a
`context.WithTimeout` over that, sized under your orchestrator's grace period)
so a stuck handler cannot hold the deploy forever. And keep the whole thing in
proportion: graceful shutdown is *politeness*, and the lease is what makes
correctness independent of it. If the process is hard-killed mid-job, the lease
expires and the job is redelivered — which is why you can run this on
preemptible machines and sleep at night.

## Scheduling: later, and repeatedly

**Delayed** jobs are already free: `run_at` in the future, and the claim query
ignores them until then — retry backoff is the same mechanism. **Periodic** jobs
are where a well-run service usually acquires its first distributed-systems bug,
because `time.Ticker` in the process works perfectly on one replica and silently
misbehaves on three: three nightly digests, or none on the night the replica
holding the ticker was restarting. Three honest answers, named so you know them
— an **external scheduler** (your orchestrator's cron) that only *enqueues* a
job; a **leader** elected through the database, a lock row or a `schedules` row
per task claimed with the same conditional update the queue already uses; and
**idempotent scheduling**, deriving the job id from the tick
(`"daily-digest:2024-03-01"`) so three replicas produce one row and two
primary-key conflicts. The shape generalises: when replicas must agree on "who
does this", either make agreement unnecessary or push it into the one component
that already serialises everything.

## Exercise

Open [`exercise/`](exercise/) — one `jobs` package over SQLite. `db.go` (schema
and DSN), `clock.go`, `job.go` (types and errors) and `query.go` (`Get`,
`CountByState`, `scanJob`) are provided; read them first. You complete four:

```
store.go     Enqueue, Claim, Complete, Retry, DeadLetter, MarkProcessed
backoff.go   Delay — exponential, capped, equal jitter, injected randomness
worker.go    ProcessOne, run, fail, Run — the worker pool
outbox.go    CreateOrder — the business write and its job, one transaction
```

The `_test.go` files are the specification; `support_test.go` holds the fake
clock and the counting handler the suite uses instead of measuring anything.

Acceptance criteria:

1. `Enqueue` inserts on the **caller's transaction**, rejects a job with no ID
   or Kind, defaults `RunAt` to `Clock.Now()` and `MaxAttempts` to
   `DefaultMaxAttempts`, and starts the job `ready` with zero attempts.
2. `Claim` takes one job in a single statement, sets state, worker and
   `lease_until = now + lease`, increments `attempts`, and returns `ErrNoJob`
   when nothing is claimable. It never hands the same job to two workers under
   `-race`.
3. `Claim` ignores a job whose `run_at` is in the future, prefers the oldest
   due job, and redelivers a `running` job whose lease has expired — with the
   crashed attempt counted.
4. `Complete` runs on the caller's transaction so a handler's writes and the
   queue update commit together; `Retry` returns a job to `ready` at a given
   `run_at` with the cause recorded and the lease cleared; `DeadLetter` parks
   it in `dead`. Updating a job that is not there is an error, not a no-op.
5. `MarkProcessed` returns true exactly once per job id, and rolls back with
   its transaction — a marker must never survive an effect that did not.
6. `Backoff.Delay` doubles from `Base`, caps at `Max`, treats attempts below 1
   as the first, and returns a value in `[d/2, d]` drawn from the injected
   source.
7. `ProcessOne` claims a job and drives it to exactly one terminal outcome:
   done, retried at `now + Delay(attempts)`, or dead-lettered. An empty queue
   is `(false, nil)`, not an error.
8. A handler error rolls back everything it wrote, including the processed
   marker; the job dead-letters when the cause wraps `ErrPermanent` or the
   attempt budget is spent, and is otherwise retried. A job whose `Kind` has no
   registered handler dead-letters without running anything.
9. A redelivery of work that already committed does **not** run the handler
   again, and still finishes the job.
10. `Run` starts `Workers` goroutines, drains the queue, and returns only after
    ctx is cancelled **and** every in-flight job has reached a terminal state —
    with no lease left behind. A cancelled pool must not abandon the job it
    already holds (`context.WithoutCancel`).
11. `CreateOrder` writes the order and enqueues its job in one transaction:
    if either fails, neither row exists.
12. `go test -race ./...` is green and the code is `gofmt`-formatted.

```sh
cd exercise
go test -race ./...
```

There is not a single `time.Sleep` in the suite, and there must not be one in
your code: every rule about time — lease expiry, delayed jobs, backoff — is
"compare a stored timestamp against `Clock.Now()`", so tests advance a fake
clock instead of waiting.

Suggested order: `store.go` first (most of the suite depends on it), then
`backoff.go`, then `worker.go`, and `outbox.go` whenever you like.

## Further reading

- [microservices.io — Transactional outbox](https://microservices.io/patterns/data/transactional-outbox.html)
- [AWS Architecture Blog — Exponential backoff and jitter](https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/)
- [pkg.go.dev — context.WithoutCancel](https://pkg.go.dev/context#WithoutCancel)
- [PostgreSQL docs — SELECT … FOR UPDATE SKIP LOCKED](https://www.postgresql.org/docs/current/sql-select.html#SQL-FOR-UPDATE-SHARE)
- [AWS SQS — Visibility timeout](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html)
