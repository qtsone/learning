# Tutor notes — Concurrency Patterns

## Where the learner is

End of the concurrency arc: they've done goroutines/WaitGroup, channels and
`select`, `Mutex`/`Once`/`atomic`, and context, each in isolation. This
lesson is the synthesis — the first time they orchestrate several primitives
in one design. Expect the individual pieces to be solid but the *choreography*
(who closes what, who waits for whom) to be new and fragile. They have not
seen `errgroup` or `x/sync` before; the exercise builds a mini-errgroup from
scratch precisely so the real library isn't magic. Insist on `-race` for
every run and treat a hanging test as a first-class learning moment: have
them Ctrl+C and read the goroutine dump before you explain anything.

## Common misconceptions

- **"Close the results channel from a worker"** — panics another worker's
  send. The closer-goroutine (`wg.Wait()` then `close`) exists because only
  the WaitGroup knows when *all* senders are done.
- **Feeding and collecting from the same goroutine** — the pool deadlocks
  once workers fill up: feeder blocked on `jobsCh <-`, workers blocked on
  `results <-`, collector never starts. The feeder must be its own goroutine.
- **"A goroutine per job with a WaitGroup *is* a worker pool"** — it's
  unbounded fan-out; the bounds test catches it (max in-flight > workers).
  Ask where the backpressure went.
- **"Cancellation stops goroutines"** — `cancel()` is a request; `Wait` must
  still wait for cancelled siblings to return. Learners often skip the
  WaitGroup in `Group` because "they're cancelled anyway".
- **Racy first-error capture** — `if g.err == nil { g.err = err }` from
  multiple goroutines is a data race; `-race` flags it. `sync.Once` (or a
  mutex) is the point of the exercise.
- **Semaphore direction flipped** — Acquire receives and Release sends on an
  *empty* buffered channel: Acquire blocks forever on a fresh semaphore. The
  buffer-is-the-count model fixes the mental picture: tokens *in* the
  channel are held slots.
- **Send on closed channel in `Submit`** — checking `closed` without a mutex,
  or setting the flag and closing under different locks. The panic is
  intermittent, which is the lesson: "it passed once" means nothing.
- **`Shutdown` waits unconditionally** — bare `wg.Wait()` with no select:
  the deadline test hangs 2s then fails. They know the wrap-wait-in-a-channel
  move from the context lesson; nudge them to reuse it.
- **Sleep-based fixes** — any `time.Sleep` in their solution code is
  synchronization by luck. Make them name the channel/WaitGroup edge that
  should replace it.

## Grilling points

- "Walk me through the pool's shutdown cascade: I close `jobsCh` — what
  happens, in order, until `RunPool` returns?" (range ends → `wg.Done` →
  closer closes `results` → collector's range ends.)
- "Why does the closer goroutine exist? What breaks if worker 3 closes
  `results` when it finishes?"
- "In your `Group`, thread A and thread B error at the same instant. What
  guarantees exactly one of them wins, and what happens to the loser's
  error?" (`sync.Once`; dropped, like real errgroup.)
- "Why does `Wait` cancel the context even on success?" (The derived
  context's resources — vet's `lostcancel` catches the leak in ordinary
  context code; same hygiene here.)
- "You have a web server, one goroutine per request, and a database that
  melts past 10 concurrent queries. Pool or semaphore, and why?" (Semaphore:
  the goroutines already exist; you need a ceiling, not a queue-and-workers
  restructure.)
- "Your `Shutdown` hits the deadline and returns `ctx.Err()`. What is the
  stuck worker doing now? Can you kill it?" (Still stuck; no — goroutines
  can only be asked via context. Segue: what would a context-aware `fn`
  change?)
- "Where is the backpressure in your Queue? Who feels it, and when?"
- (A-level) "Every worker is permanently stuck in `fn`, and one `Submit` is
  blocked mid-handoff. Now I call `Shutdown` with a deadline. Does it
  return `ctx.Err()`?" (No — in the send-under-mutex design that Submit
  holds the mutex, so Shutdown blocks *before* its deadline select. The
  deadline guards slow drains, not a wedged fn. Full credit is naming the
  envelope, not patching it: fixing it means restructuring intake — e.g.
  select on a quit channel instead of sending under the lock — with its own
  trade-offs. The point: every concurrent design has scenarios its
  guarantees exclude; an expert states them unprompted.)
  (`Submit` blocks on the unbuffered send until a worker frees up.)
- "Why is sending under the mutex in `Submit` load-bearing rather than a
  style choice?" (It excludes `Shutdown`'s close while a send is in flight —
  drop the lock before the send and the panic window reopens.)

## Grading rubric

- **A** — All tests pass under `-race`; pool has the four-role shape with
  channel-close cascade (no counters, no sleeps); `Group` uses
  `Once`+`WaitGroup`+`cancel` and cancels in `Wait`; semaphore is the bare
  buffered channel with a `select`/`default` TryAcquire; `Queue` guards flag
  and close under one mutex and selects done vs `ctx.Done()`. Learner
  answers the pool-vs-semaphore and backpressure questions crisply and can
  narrate the shutdown cascade unprompted.
- **B** — Tests pass under `-race` but with roughness: a buffered results
  channel sized to len(jobs) doing quiet double duty, a mutex where `Once`
  would say it better, or a correct-but-hesitant explanation of why the
  closer goroutine exists. Design judgment (pattern choice) mostly sound.
- **C** — Tests pass only after heavy hinting, or pass while the learner
  can't explain who closes each channel and why, or can't say where the
  backpressure lives. Pass only if remediation lands in-session; otherwise
  iterate.
- **Fail** — Tests failing or racing; a `time.Sleep` doing synchronization
  work; send-on-closed panic still possible in `Submit`; or the learner
  cannot distinguish "cancel requested" from "goroutine stopped". Remediate,
  don't advance.

## Remediation ladder

1. "Run `go test -race -run TestRunPool -v`. If it hangs, Ctrl+C and read
   the dump: which goroutine is parked on which line? Every deadlock names
   its own cause."
2. "List your channels. For each: who sends, who closes, and how does every
   receiver learn it's over? There should be no channel you can't answer
   for." (This question alone fixes most pool and queue bugs.)
3. "Order of operations in shutdown: door, drain, deadline. Which of the
   three is your failing test about? Now — which primitive owns that step:
   the mutex, the closed channel, or the select?"
4. Walk one piece verbally at the shape level — e.g. for `Group.Go`:
   "Add before the goroutine starts; inside: defer Done, run fn, and on
   error a `once.Do` that records and cancels" — then have them type it and
   re-run `-race` before touching the next piece.

## After passing

Preview: "Next is a guided tour of the toolbox — vet, staticcheck,
golangci-lint, pprof — and then the stage capstone: a concurrent tool where
these patterns stop being exercises and start being the design."
