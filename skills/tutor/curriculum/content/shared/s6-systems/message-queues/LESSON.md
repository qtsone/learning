# Message Queues

> `shared.systems.message-queues` · ~3h · Stage: Systems & Design

## Objectives

By the end of this lesson you can:

- Explain when to move work to async processing and what latency, complexity,
  and consistency costs the queue introduces.
- Contrast at-most-once, at-least-once, and exactly-once delivery semantics
  and explain why exactly-once end-to-end requires idempotent consumers.
- Implement an idempotent consumer that survives redelivery without
  duplicating side effects.
- Explain the dual-write problem and implement the transactional outbox
  pattern to publish events atomically with state changes.
- Design dead-letter and retry policies for poison messages.

## Getting off the request path

Everything you have built so far answers inside the request: the caller waits,
you do the work, you respond. That model has a ceiling. Some work is too slow
to make a user wait for (transcoding a video, generating a report). Some work
spikes far above what your downstream can absorb (a flash sale placing 50,000
orders in a minute against a payment provider that handles 500 per second —
your capacity estimates from the design-intro lesson tell you this ends badly).
And some work simply doesn't belong to the request: the order is placed;
sending the confirmation email is *someone else's job, later*.

A **message queue** is the buffer you put between the two: the producer writes
a message describing the work and returns immediately; a consumer picks it up
when it has capacity. The queue absorbs bursts, smooths load to the consumer's
pace, and decouples the two sides' deploys, outages, and scaling.

None of that is free, and you should be able to name the bill:

- **Latency.** The work is no longer done when the response returns — it is
  done *eventually*. "Your export is ready" becomes an email, not a response
  body.
- **Consistency.** The caller gets "accepted", not "done". Read-your-writes is
  gone: the user who placed the order may refresh and not see its side effects
  yet. Producer and consumer now agree only eventually, and your UI and API
  design must say so honestly. This pattern's HTTP face is a `202 Accepted`
  carrying the URL of a status resource the caller can poll: the response
  says "I have the work, here is where to watch it finish" instead of
  pretending it is already done.
- **Complexity.** You now run a broker: one more system to deploy, monitor,
  and capacity-plan, and a whole new failure vocabulary — redelivery,
  duplicates, poison messages — which is most of this lesson.

So the design rule: queue work that is slow, bursty, or independent of the
response; keep work synchronous when the caller genuinely needs the answer
now. "We added a queue" is not an upgrade by itself — it is a trade.

## The shape of a queue

The moving parts are the same in every system (SQS, RabbitMQ, Pub/Sub, and —
with a twist — Kafka):

- **Producers** publish messages to the broker and move on.
- The **broker** stores messages durably and hands them to consumers. A plain
  *queue* delivers each message to one consumer; a *topic* (pub/sub) copies it
  to every subscriber. This lesson works the queue case.
- **Consumers** receive a message, do the work, and then **acknowledge** (ack)
  it — "done, delete it". The ack is the heart of the whole design.

Between receive and ack, the message is **in flight**: still owned by the
broker, but invisible to other consumers. If the ack never comes — the
consumer crashed, hung, or lost its network — the broker waits out a
**visibility timeout** and then makes the message deliverable again, to
anyone. That redelivery loop is what makes a queue more than a buffer: work
survives the death of the worker doing it.

Two consequences to internalize now:

- The visibility timeout is a *guess* about how long processing takes. Too
  short, and the broker redelivers a message that a healthy consumer is still
  working on — instant duplicate. Too long, and a crashed consumer's messages
  sit invisible for minutes before anyone retries them.
- Redelivery reorders. A message that failed and came back joins behind
  messages published after it. Queues promise "roughly FIFO", not FIFO;
  designs that need strict per-key ordering need a log-structured broker like
  Kafka (a partitioned, replayable log with consumer offsets — recognize the
  name; we don't need its machinery here).

In Go: a buffered channel looks like a queue and is not one. A value received
from a channel is *gone* — there is no ack, so a consumer that crashes after
`<-ch` silently loses the work, and there is no redelivery, no visibility
timeout, no dead-lettering. Channels move work between goroutines inside one
process; a message queue moves work between *processes that fail
independently*. The exercise makes you build the difference.

## Delivery semantics: pick which failure you keep

Here is the uncomfortable core of the topic. A consumer receives a message,
processes it (charges the card, sends the email), and acks. Now crash the
consumer at the worst moment — after processing, before the ack arrives. The
broker sees only a missing ack. It cannot distinguish "crashed before doing
the work" from "crashed after" — the information doesn't exist on its side.
It must choose blind, and the choice defines the semantics:

| Semantics | Broker's rule | Failure cost |
|---------------|--------------------------------------|--------------------------|
| At-most-once | never redeliver (fire and forget) | work can be **lost** |
| At-least-once | redeliver until acked | work can be **duplicated** |
| Exactly-once | — | not a broker setting |

At-most-once is fine when a lost message is cheaper than a duplicate — a
metrics sample, a presence ping. For anything that matters, you want
at-least-once: never lose the order, accept that it may arrive twice.

Exactly-once *delivery*, end to end, is not something a broker can sell you,
no matter what the marketing page says: the ack can always be lost after the
work is done, so the broker must either redeliver (duplicate) or not (loss).
What you *can* build is exactly-once **processing**: let the transport
duplicate, and make the duplicates harmless at the consumer. That is the
next section, and it is the single most useful idea in this lesson.

## Idempotent consumers

You met idempotency twice already: safe-to-retry methods in the http-clients
lesson, and idempotency keys in api-design. Here it becomes a *requirement*:
under at-least-once delivery, your processor **will** be called twice with the
same message someday. An **idempotent consumer** is one where that's fine —
processing a message twice leaves the world exactly as processing it once.

Two ways to get there:

- **Naturally idempotent operations.** "Set order 42's status to `shipped`"
  can run five times; "increment the shipped counter" cannot. When you control
  the message schema, prefer absolute statements ("the state is X") over
  deltas ("change by X") — the operation dedupes itself.
- **A dedupe record.** For side effects that can't be replayed (charge a
  card, send an email), remember what you've already done: before processing,
  check a *processed-messages* store for this message's ID; after processing,
  record it. On a duplicate, skip the work and ack anyway — the ack is
  honest, because the work *is* done.

Two sharp edges the exercise will press on:

- Record the ID only on **success**. A processing attempt that failed must
  not be remembered as done, or the retry you wanted becomes a skip and the
  message dies having done nothing.
- The dedupe store must survive the consumer and be shared by its replicas —
  in practice a database table, not a variable in one process's memory. And
  ideally the "record the ID" write commits **in the same transaction as the
  side effect**, so "did the work" and "remembered doing it" can't disagree.
  In the same transaction... that move has a name, and it's next.

## The dual-write problem and the outbox

Now flip to the producer side, where a subtler bug lives. The order service
must do two things when an order is placed: write the order to its database,
and publish an `order-placed` event to the queue. Two different systems, no
shared transaction. Whatever order you pick, the crash between the two steps
lies to someone:

- **DB first, then publish** — crash after the commit and the order exists
  but no event was ever sent. The email service, the analytics pipeline, the
  warehouse: none of them will ever hear about this order.
- **Publish first, then DB** — crash after the publish and downstream systems
  process an order that, as far as your database is concerned, never
  happened.

This is the **dual-write problem**, and sprinkling retries on it does not fix
it — the crash can always land between the two writes. The fix is to stop
doing two writes. The **transactional outbox** pattern:

1. In the *same database transaction* as the state change, insert the event
   into an `outbox` table. One system, one commit: the order and its event
   now exist atomically or not at all.
2. A separate **relay** process polls the outbox for unpublished rows,
   publishes each to the broker, and marks it published.

The relay is deliberately boring: read, publish, mark, repeat. Note what
happens when *it* crashes between publish and mark — the event is published
again on restart. The outbox gives you atomicity at the cost of duplicates,
which is exactly the deal you already accepted: the pipeline is at-least-once
end to end, and the idempotent consumer you just built absorbs it. One detail
matters: relay duplicates are *distinct broker messages* with the same
business event inside — same event, brand-new message ID — so you dedupe on an
event ID carried in the payload, not on the broker's message ID. That means the
consumer has to be *told* which key identifies the work; the exercise wires
exactly that. The two patterns are halves of one design.

## Poison messages, retries, and the dead-letter queue

Redelivery assumes failures are transient. Some aren't. A message with a
malformed payload, or one that reliably crashes the consumer, will fail on
every attempt, forever. That is a **poison message**, and under redelivery it
becomes a treadmill: deliver, crash, time out, redeliver — burning consumer
capacity and, if ordering funnels work behind it, blocking everything else.

The policy that contains it has three parts:

- **Bounded attempts.** Count deliveries per message; after N failures, stop
  redelivering. N is small — 3 to 5 — because attempt six teaches you nothing
  attempt five didn't (the same lesson as retry caps in http-clients, and
  backoff between attempts applies here too).
- **A dead-letter queue (DLQ).** Exhausted messages aren't deleted — they are
  moved to a parking-lot queue, out of the consumers' way but preserved with
  their content and delivery history. Deleting them silently is at-most-once
  through the back door.
- **Someone watching.** A non-empty DLQ is an alert (your observability work
  in S5 is where it lands): each message there is work the system promised
  and didn't do. The payoff of parking rather than deleting is **replay** —
  fix the consumer bug, then move the dead letters back through the queue.

Distinguish failure kinds when you can, because the consumer knows things the
broker cannot: "downstream timed out" deserves the full retry budget, while a
payload that will never parse deserves zero. Say which out loud rather than
going quiet and letting the visibility timeout expire — silence costs the whole
timeout and tells the broker nothing:

- **Nack** — "this attempt failed." The message retires immediately: back to
  the ready queue if attempts remain, to the DLQ if that was the last one.
- **Dead-letter** — "no attempt will ever succeed." Park it now, whatever the
  attempt count says. Without this verb, a permanently-invalid message still
  rides its full 3-5 attempts, which is the treadmill in miniature.

## Exercise

Open [`exercise/`](exercise/) — a Go module where you build the whole
story end to end, in three files: `broker.go` (a tiny broker with acks,
visibility timeouts, redelivery, and a DLQ), `consumer.go` (the idempotent
consumer), and `outbox.go` (a simulated transactional store with an outbox
relay). The tests are the specification — read them first. Time is fully
injected via the `Clock` in `clock.go`: tests advance a fake clock, nothing
sleeps, and the broker is passive — no goroutines, no timers; all redelivery
bookkeeping happens inside `Receive` when it is called. The wire format is
handed to you — `Event.Wire()` writes `"<id>|<payload>"`, `EventID` reads the
id back out — so the decision left to you is which key each consumer dedupes
on.

Acceptance criteria:

1. `Publish` assigns IDs `"m1"`, `"m2"`, … in publish order; `Receive`
   delivers ready messages FIFO, with `Attempts` counting deliveries
   (first delivery = 1), and reports `ok == false` when nothing is
   deliverable.
2. A received message is in flight — invisible to further `Receive` calls —
   until it is acked, nacked, dead-lettered, or its deadline (receive time +
   `Visibility`) passes.
3. `Receive` first sweeps expired in-flight messages, oldest delivery first:
   a message with attempts remaining rejoins the **back** of the ready queue;
   one that has used `MaxAttempts` deliveries moves to the dead-letter queue.
4. `Ack` removes the message for good; `Nack` retires it immediately (requeue
   or dead-letter by the same attempts rule, no waiting for the deadline);
   `DeadLetter` parks it at once whatever its attempt count — the "this will
   never succeed" verb. All three return an error satisfying
   `errors.Is(err, ErrUnknownID)` when the id is not in flight.
5. `DeadLetters` returns dead-lettered messages in arrival order, and the
   broker is safe under concurrent use (`go test -race` stays quiet).
6. `IdempotentConsumer.Handle` runs the processor at most once per **dedupe
   key** — `key(msg)`, falling back to `ByMessageID` when the key is nil:
   duplicates return nil without re-running it, and a processor *error*
   surfaces without being recorded — a later redelivery retries it.
7. `ExecTx` commits staged `Set` writes and `Emit`ted events together, or —
   when the transaction function returns an error — commits neither and
   returns that error; `Get` sees only committed state.
8. `Relay` publishes each committed event to the broker exactly once per
   store, in commit order, as `Event.Wire()` — the event ID travels with the
   payload — and returns how many it published.
9. End to end, a relay that publishes and then crashes before marking the row
   sends the same event as two broker messages; a consumer keyed on `EventID`
   runs the side effect once, while `ByMessageID` would run it twice.
10. `go test -race ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test -race ./...
```

Build it in test order: broker first (publish/receive, then ack, then
redelivery, then the DLQ), consumer second, outbox last — the final tests
wire all three into the at-least-once pipeline this lesson is about.

## Further reading

- [AWS SQS — visibility timeout](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html)
- [microservices.io — Transactional outbox](https://microservices.io/patterns/data/transactional-outbox.html)
- [Tyler Treat — You Cannot Have Exactly-Once Delivery](https://bravenewgeek.com/you-cannot-have-exactly-once-delivery/)
- [RabbitMQ — Reliability guide](https://www.rabbitmq.com/docs/reliability)
