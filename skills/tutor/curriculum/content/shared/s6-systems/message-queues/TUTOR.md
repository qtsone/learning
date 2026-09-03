# Tutor notes — Message Queues

## Where the learner is

Mid-S6, straight after caching. They have just built a concurrent structure
with an injected clock, so the mechanics here (mutex, fake clock, no sleeps)
are familiar and the novelty is entirely conceptual: this is the first lesson
where **two processes fail independently** and correctness has to survive
that. From the Go path they bring channels, `-race`, and retry/idempotency
instincts from http-clients; from api-design they bring idempotency keys.
The `202 Accepted` + status-URL shape is introduced here, not recalled —
don't quiz it as prior knowledge. Expect the code to come easily and the
semantics to be where they actually struggle.

The load-bearing idea to protect: *the broker cannot tell "crashed before the
work" from "crashed after the work", so someone must absorb duplicates.* If
they leave with only that, the lesson worked.

## Common misconceptions

- **"Exactly-once delivery is a broker feature."** The marketing claim they
  will meet in the wild. Force the argument: the ack can always be lost after
  the side effect committed, so the broker's only two options are redeliver
  (duplicate) or not (loss). What exists is exactly-once *processing*, built
  at the consumer. Kafka's "exactly-once" is transactional read-process-write
  *within Kafka* — mention only if they raise it, and note it does not extend
  to charging a credit card.
- **"A Go channel is a queue."** Very common on this path. A channel receive
  consumes the value: no ack, no redelivery, no visibility timeout, no DLQ,
  and it dies with the process. Ask what happens to in-flight work when the
  worker goroutine panics.
- **Acking before doing the work.** Sounds like it prevents duplicates; it
  converts the system to at-most-once and silently drops work on crash. Ask
  which failure they just chose.
- **"Retries fix the dual-write problem."** They do not: the crash lands
  *between* the two writes, so there is no process alive to retry. Only
  collapsing the two writes into one transaction removes the window.
- **Recording the dedupe ID before processing, or on failure.** Turns the
  first failed attempt into a permanent skip — the retry they designed for
  becomes a no-op and the work never happens. The tests catch it; make sure
  they can say *why* it is wrong, not just that the test went red.
- **Dedupe state in a process-local map is production-ready.** In the
  exercise it is one value on purpose; in reality replicas do not share
  memory. It must be a durable store shared by all consumers — ideally
  committed in the same transaction as the side effect.
- **"Queues preserve order."** Redelivery reorders by construction. Anything
  needing per-key order needs partitioning, not a plain queue.
- **DLQ as a trash can.** Deleting exhausted messages is at-most-once through
  the back door. The DLQ's value is alerting and replay.
- **"Nack and DeadLetter are the same button."** `Nack` says "this attempt
  failed" and still spends the retry budget; `DeadLetter` says "no attempt
  will succeed" and parks the message on attempt one. Nacking a message that
  can never parse just runs the treadmill 3-5 times faster.
- **Bigger visibility timeout / more retries = more reliable.** Both trade:
  long timeouts stall recovery from real crashes, unbounded retries burn
  capacity on a message that will never succeed.

## Grilling points

- "Your consumer charges a card, then crashes before acking. Walk me through
  what the broker knows, what it does, and what the customer sees — with and
  without your dedupe record."
- "Where exactly in your code is the at-least-once guarantee? Point at the
  line." (The ack after processing; move it before and name what changed.)
- "You set the visibility timeout to 5 seconds and a job occasionally takes
  30. What happens? Now set it to 10 minutes and the pod gets OOM-killed —
  what happens?"
- "Why is a `sync.Mutex` enough for your broker, and what would you need
  instead if the broker were a separate process?" (Durable storage plus
  atomic claim of a message — the lock stops being a language primitive.)
- "Your relay crashes between publishing and marking the row published. What
  does the consumer see, and which piece of your design saves you?"
- "Your pipeline consumer is keyed on `EventID` and your redelivery consumer
  on `ByMessageID`. Swap the first one to `ByMessageID` — predict which test
  breaks and why before you run it." (The relay-crash test: two distinct
  broker messages carry one business event, so only the event ID collapses
  them.)
- "A message has failed 3 times with 'invalid JSON' and another 3 times with
  'payment gateway timeout'. Should both have the same retry policy — and
  which broker verb does your consumer call for each?" (`DeadLetter` for the
  first, `Nack` for the second.)
- "Your DLQ has 400 messages this morning. What do you actually do — in
  order?" (Alert existed → inspect one → classify bug vs data → fix → replay,
  and replay hits the idempotent consumer, so duplicates are fine.)
- "Which of these belongs on a queue and which stays synchronous: sending a
  receipt email, checking a password at login, resizing an avatar, reserving
  the last ticket in stock?"
- Estimation habit (keep S6's spine): "Producers burst to 50k orders/minute,
  the consumer handles 500/s. How deep does the queue get, and how long until
  it drains?" Expect stated assumptions and a back-of-envelope number, not
  precision.

## Grading rubric

- **A** — All tests pass under `-race`. The broker keeps ready / in-flight /
  dead as distinct states with the sweep centralized (one `retire` rule
  shared by expiry and `Nack`, which `DeadLetter` deliberately bypasses);
  `DeadLetters` returns a copy; errors wrap `ErrUnknownID` with context;
  `Handle` records the dedupe key only after the processor returns nil;
  `ExecTx` mutates the store only after `fn` succeeds.
  Learner explains unprompted why exactly-once delivery is impossible, why
  the outbox and the idempotent consumer are two halves of one design, and
  what their in-memory dedupe map would have to become in production.
- **B** — Tests pass, with a wart they can't fully defend: expiry and `Nack`
  duplicating the attempts rule instead of sharing it, `DeadLetters` handing
  back the internal slice, lock held around a callback, or `Attempts`
  incremented in the wrong place and patched until green. Semantics solid;
  the outbox explained as a recipe more than as an argument.
- **C** — Tests pass only after heavy hinting, or the design was
  reverse-engineered from failure messages: they can state the rules but not
  derive them from the crash-between-work-and-ack story. Time-box
  remediation on the delivery-semantics table and the dual-write crash
  window before passing.
- **Fail** — Race failures or failing tests; or the learner believes a broker
  can deliver exactly once; or they cannot say why a failed attempt must not
  be recorded as done. Do not advance: distributed-systems, reliability and
  the capstone all assume this vocabulary.

## Remediation ladder

1. "Run just the failing test with `-run`. Read it aloud: what did it do to
   the clock, and what did it expect the broker to say?"
2. "Name the three places a message can be at any instant, and list every
   event that moves it from one to another. That table *is* the broker."
   (ready → in-flight on `Receive`; in-flight → gone on `Ack`; in-flight →
   ready or dead on `Nack`/expiry; in-flight → dead on `DeadLetter`.)
3. For redelivery: "Your broker has no goroutines, so who notices a deadline
   passed? When is the only moment anyone asks?" (The next `Receive` — sweep
   first, then deliver.) For the consumer: "Trace a message whose first
   attempt errors. At which line did you write it into `done`, and what does
   the *second* delivery then do?"
4. Sketch the shape verbally — for the broker, `Receive` = lock, sweep
   expired via the shared retire rule, pop head, bump `Attempts`, record
   deadline; for the outbox, `ExecTx` = stage into a `Tx`, run `fn`, return
   early on error, otherwise apply writes and append events — and let them
   type it. Do not open the solution files.

## After passing

Preview: "You just made one broker survive a dying consumer. Next lesson
zooms out to the whole system: replication, partitioning, CAP/PACELC, and the
failure modes that make distributed state hard — concepts, no implementation."
