# Concurrency Patterns

> `go.intermediate.concurrency-patterns` · ~4-6h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Implement a bounded worker pool that fans work out over a channel and
  collects results without leaking goroutines.
- Implement coordinated error handling and cancellation across goroutines
  with errgroup (and explain what it does under the hood).
- Implement a semaphore with a buffered channel (or `x/sync/semaphore`) to
  limit concurrency, and explain when to prefer it over a worker pool.
- Implement graceful shutdown: stop accepting work, drain in-flight work,
  and exit within a deadline.
- Choose an appropriate concurrency pattern for a described workload and
  justify the choice in terms of backpressure and failure handling.

## From primitives to patterns

You now own every concurrency primitive Go has: goroutines and `WaitGroup`,
channels and `select`, `Mutex`/`Once`/`atomic`, and `context`. This lesson is
where they click together into the handful of shapes that real Go programs
are actually made of. None of these patterns is a library feature — each is a
short arrangement of primitives you already know, and each earns its place by
answering two questions every concurrent design must answer:

- **Backpressure** — when the work arrives faster than you can do it, who
  slows down, and how? (Unbounded concurrency isn't speed; it's an
  out-of-memory error on a delay.)
- **Failure handling** — when one piece of work fails, what happens to the
  others: keep going, or stop everything?

Keep the race detector on for this entire lesson. A concurrency pattern you
haven't run under `-race` is a rumor, not a fact.

## The worker pool

The problem: you have many independent jobs and you want *at most N* of them
running at once. The shape has four roles, each a few lines:

```go
jobsCh := make(chan int)
results := make(chan Result)
var wg sync.WaitGroup

wg.Add(workers)
for range workers {                 // N workers, started once
	go func() {
		defer wg.Done()
		for j := range jobsCh {     // pull until the channel closes
			results <- Result{Job: j, Val: fn(j)}
		}
	}()
}

go func() {                         // feeder: hand out the jobs
	for _, j := range jobs {
		jobsCh <- j
	}
	close(jobsCh)                   // no more work — workers will drain and exit
}()

go func() {                         // closer: end the results stream
	wg.Wait()
	close(results)
}()

for r := range results {            // collector: the only reader
	out = append(out, r)
}
```

Read the choreography, because every line is load-bearing:

- The **feeder** runs in its own goroutine so the collector can start
  receiving immediately — feeder and collector in the same goroutine is a
  classic deadlock (the feeder blocks sending job 3 because no worker is
  free, workers block sending results because nobody collects).
- `close(jobsCh)` is how workers learn there is no more work. Each worker's
  `for range` loop ends, its `wg.Done()` runs. You learned in the channels
  lesson that the *sender* closes — here the feeder is the only sender, so
  it's the only place `close` can safely live.
- The **closer** exists because `close(results)` must happen after *all*
  workers finish sending, and only `wg.Wait()` knows when that is. Closing
  from any single worker would panic another worker's send.
- Results arrive in **completion order**, not submission order. If order
  matters, carry the job (or its index) inside the result, as `Result` does.

The pool is *bounded*: exactly N goroutines, started once, reused for every
job. And the backpressure is built in: `jobsCh` is unbuffered, so when all
workers are busy the feeder blocks. Slow consumers slow producers — the
system degrades by waiting, not by exploding.

### Goroutine leaks

A goroutine leaks when it blocks forever on a channel nobody will ever
service again. The function that spawned it returns; the goroutine stays,
holding its stack and everything it references, invisible until you have ten
thousand of them. In pool code, the classic leaks are:

- Forgetting `close(jobsCh)` — every worker blocks forever in `for range`.
- Forgetting the closer — the collector blocks forever in `for range
  results`, and your function never returns (in a server, that's a leaked
  goroutine per request).
- A worker sending on `results` after the collector stopped reading early.

The discipline that prevents all three: **for every channel, know who closes
it and know why every send eventually finds a receive.** The exercise tests
count goroutines before and after your pool runs — a leak fails the test.

## errgroup: shared fate for a batch of goroutines

The pool treats jobs as independent: one job failing is a result, not a
crisis. Plenty of workloads are the opposite — fetch five resources that a
page needs, and if any fetch fails, the other four are wasted work that
should stop *now*. The tool for "these goroutines succeed or fail together"
is `errgroup.Group` from `golang.org/x/sync/errgroup`, and it looks like a
`WaitGroup` that grew up:

```go
g, ctx := errgroup.WithContext(ctx)
for _, url := range urls {
	g.Go(func() error {
		return fetch(ctx, url)      // honors cancellation, like you learned
	})
}
if err := g.Wait(); err != nil {    // first non-nil error, after ALL finish
	return err
}
```

Its contract, precisely:

1. `Go` runs the function in a tracked goroutine — no manual `Add`/`Done`.
2. The **first** non-nil error is remembered; later errors are dropped.
3. That first error **cancels the derived `ctx`**, so every sibling that
   respects context (as all your code does since the context lesson) stops
   early instead of finishing pointlessly.
4. `Wait` blocks until *every* function has returned — even after an error —
   then returns the first error. No goroutine outlives the group.

Point 4 is the quiet one that matters: cancellation is a request, not a
kill. The group still waits for cancelled siblings to notice and return,
which is why nothing leaks. (The real errgroup also has `SetLimit` to cap
concurrent `Go` calls — a pool and a group in one.)

In the exercise you build a miniature errgroup yourself — `WaitGroup` +
`sync.Once` + `context.CancelFunc` is the entire trick — both to see through
the magic and to keep the exercise module dependency-free. In production
code, reach for the real one.

## The semaphore: a limit without a pool

Sometimes you don't want a pool — the goroutines already exist. A server
gives you one goroutine per request for free; you just need to stop more
than 10 of them hitting the database at once. What you want is a *counting
semaphore*: N tokens, take one to proceed, give it back when done. A
buffered channel is exactly that:

```go
type Semaphore struct{ slots chan struct{} }

func NewSemaphore(n int) *Semaphore {
	return &Semaphore{slots: make(chan struct{}, n)}
}

func (s *Semaphore) Acquire() { s.slots <- struct{}{} } // blocks when full
func (s *Semaphore) Release() { <-s.slots }
```

The buffer *is* the count: while fewer than `n` tokens are in the channel,
`Acquire`'s send proceeds; at `n` it blocks until someone releases. A
non-blocking variant falls out of `select` with `default` — the same move
you learned in the channels lesson:

```go
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.slots <- struct{}{}:
		return true
	default:
		return false                // full — caller can shed load instead of waiting
	}
}
```

(`golang.org/x/sync/semaphore` is the grown-up version — weighted costs and
context-aware acquire — same idea.)

**Pool or semaphore?** Both cap concurrency at N; they differ in *where the
goroutines come from*:

- A **worker pool** owns its goroutines: N long-lived workers pulling from a
  queue. Choose it for a stream of homogeneous jobs in a long-running
  service — you pay the goroutine cost once and the jobs channel gives you a
  natural queue with backpressure.
- A **semaphore** caps goroutines it doesn't own. Choose it when the
  concurrency already exists (a goroutine per request, a burst of
  heterogeneous tasks) and you only need a ceiling around one contended
  resource — no restructuring, and `TryAcquire` gives you load-shedding,
  which a pool's queue can't express.

## Graceful shutdown

Every long-running service eventually receives "please stop." The naive
options are both wrong: exiting immediately drops in-flight work; waiting
unconditionally can hang forever on one stuck job. Graceful shutdown is a
three-step contract:

1. **Stop accepting.** New work is refused from this moment on.
2. **Drain.** Work already accepted is finished.
3. **Bounded wait.** If draining outlasts a deadline, give up and report it.

Steps 1 and 2 reuse the pool's machinery: close the jobs channel and the
workers drain it and exit. But there's a trap — in the pool above, the feeder
was the *single* sender, so closing was trivially safe. A server's `Submit`
can be called from many goroutines, and **sending on a closed channel
panics**. So intake needs a gate: a mutex-guarded `closed` flag that `Submit`
checks before sending and `Shutdown` sets before closing. Flag and channel
close under the same mutex, or a `Submit` that squeezed past the check sends
into a closing channel.

Step 3 is a `select` between "drained" and "out of time" — and since
`WaitGroup.Wait` isn't a channel, you wrap it in one:

```go
done := make(chan struct{})
go func() {
	wg.Wait()
	close(done)
}()
select {
case <-done:
	return nil                      // drained cleanly
case <-ctx.Done():
	return ctx.Err()                // deadline hit: report, let the caller decide
}
```

You've seen this wrap-a-wait-in-a-channel move in the context lesson; here
it completes the shutdown story. Note what a timeout does *not* do: it
doesn't kill the workers — goroutines can't be killed, only asked. It stops
*waiting* and tells the caller the truth. This is the shape behind
`http.Server.Shutdown(ctx)`, which you'll meet in the advanced stage.

## Choosing a pattern

The interview question — and the real-life design review — is never "write a
worker pool"; it's "here's a workload, what do you reach for?" Answer in
terms of backpressure and failure handling:

| Workload | Reach for | Because |
|----------|-----------|---------|
| Long-lived service chewing a stream of similar jobs | Worker pool | Fixed goroutine cost; the jobs channel queues and applies backpressure; one job failing is just a result |
| A batch of related tasks that stand or fall together | errgroup | First failure cancels the siblings; one `Wait`, one error, nothing leaks |
| Existing goroutines (one per request) contending for a scarce resource | Semaphore | A ceiling without restructuring; `TryAcquire` sheds load under pressure |
| "Please stop" arriving at any of the above | Close intake → drain → bounded wait | Accepted work completes, stuck work can't hang you past the deadline |

When you justify a choice, say who blocks when the system is saturated
(that's your backpressure story) and what one failure does to the rest
(that's your failure story). If you can't answer both, the design isn't done.

## Exercise

Open [`exercise/`](exercise/) — a module with four small files, one per
pattern: `pool.go`, `group.go`, `semaphore.go`, `queue.go`, each with `TODO`
work sites and a matching test file. Read the tests first: they check the
choreography, not just the answers — goroutine leaks, concurrency ceilings,
cancellation, and drain-versus-deadline all have dedicated tests.

You will build four pieces:

1. **`RunPool(workers int, jobs []int, fn func(int) int) []Result`** — a
   bounded worker pool: exactly `workers` goroutines fan the jobs out over a
   channel and a collector gathers one `Result` per job.
2. **`WithContext(ctx) (*Group, context.Context)`** — a mini-errgroup:
   `Go(fn)` tracks goroutines, the first error cancels the derived context,
   `Wait` blocks for all and returns that first error.
3. **`NewSemaphore(n int)`** — a counting semaphore over a buffered channel
   with `Acquire`, `Release`, and non-blocking `TryAcquire`.
4. **`NewQueue(workers int, fn func(int))`** — a job queue with graceful
   shutdown: `Submit` refuses work once shutdown begins, `Shutdown(ctx)`
   drains in-flight jobs but returns `ctx.Err()` if the deadline expires
   first.

Acceptance criteria:

1. `RunPool` returns exactly one `Result` per job with `Val = fn(Job)`, runs
   at most `workers` calls to `fn` concurrently (and at least two, when
   `workers` permits — the pool must actually be parallel), and leaks no
   goroutines after it returns.
2. `Group`: all functions run; `Wait` returns `nil` when all succeed and the
   *first* error otherwise; that first error cancels the group context while
   siblings are still running; `Wait` returns only after every function has
   finished, and the context is cancelled by the time `Wait` returns.
3. `Semaphore`: `TryAcquire` succeeds exactly `n` times on a fresh semaphore
   and fails when full; `Acquire` blocks at the limit and proceeds after
   `Release`; at most `n` holders exist at any moment.
4. `Queue`: every job submitted before shutdown is processed exactly once;
   `Shutdown(ctx)` returns `nil` after a clean drain; with an
   already-expired context and a stuck job it returns `ctx.Err()`; `Submit`
   returns `false` after shutdown begins — even when the drain timed out.
5. `go test -race ./...` passes and the code is `gofmt`-clean.

Run the tests from inside `exercise/` — with the race detector, always:

```sh
cd exercise
go test -race -timeout 30s ./...
```

They fail on the starter — make them green. Two notes for the road: a test
run that *hangs* means you've built a deadlock — press Ctrl+C and read the
goroutine dump, it names every stuck line. And no sleeps as synchronization:
if you're tempted to `time.Sleep` your way past a failure, the choreography
is wrong — fix the channel or the `WaitGroup`, not the clock.

## Further reading

- [Go Blog — Go Concurrency Patterns: Pipelines and cancellation](https://go.dev/blog/pipelines)
  — the canonical essay on fan-out/fan-in, closing, and leak-free pipelines.
- [pkg.go.dev — golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup)
  — read the `WithContext` and `SetLimit` docs; compare with your mini version.
- [pkg.go.dev — golang.org/x/sync/semaphore](https://pkg.go.dev/golang.org/x/sync/semaphore)
  — the weighted, context-aware big sibling of your buffered channel.
- [Go Blog — Share Memory By Communicating](https://go.dev/blog/codelab-share)
  — the philosophy underneath every pattern in this lesson.
