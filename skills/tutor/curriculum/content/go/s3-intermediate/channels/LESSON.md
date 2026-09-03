# Channels

> `go.intermediate.channels` · ~3-5h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Explain the blocking semantics of unbuffered versus buffered channels and
  predict when a given send or receive blocks.
- Implement a pipeline of goroutines connected by channels, closed correctly
  so all stages terminate.
- Implement `select` to multiplex multiple channels, including a `default`
  case and a cancellation channel.
- Explain the rules of closing channels: who closes, why receiving from a
  closed channel yields zero values, and why sending to one panics.
- Predict the behavior of channel operations on nil channels and explain how
  that is exploited in `select` loops.

## From waiting to communicating

The goroutines lesson left you with a deliberate gap. You can start goroutines
and wait for them with a `WaitGroup`, but the moment they need to hand results
back, you had to write into shared variables — and the race detector showed
you how badly unsynchronized sharing goes. Channels close that gap: a channel
is a typed conduit that *moves a value from one goroutine to another and
synchronizes the two in the same operation*. The Go proverb: "Don't
communicate by sharing memory; share memory by communicating."

```go
ch := make(chan int)      // a channel carrying ints

go func() { ch <- 42 }()  // send: the value goes in, following the arrow

v := <-ch                 // receive: the value comes out, following the arrow
```

The arrow always points the way data flows. It also appears in *types*:

```go
func produce(out chan<- int) { … }   // produce may only send on out
func consume(in <-chan int)  { … }   // consume may only receive from in
```

A plain `chan int` converts implicitly to either restricted form. Returning
`<-chan int` from a constructor means callers *cannot* send on it or close it
— the compiler enforces your ownership rules. Every pipeline stage you write
below uses this.

## When does an operation block?

This is the heart of the lesson. `make(chan int)` creates an **unbuffered**
channel: a send blocks until another goroutine is ready to receive, and a
receive blocks until another goroutine sends. The two goroutines meet, the
value passes directly between them, and both proceed — a rendezvous. That
makes an unbuffered send a *synchronization point*, not just data transfer:
when `ch <- v` returns, you know the receiver has the value.

`make(chan int, 3)` creates a **buffered** channel: a queue of capacity 3.
A send blocks only while the buffer is full; a receive blocks only while it
is empty. Three sends succeed with no receiver anywhere; the fourth blocks.
A buffered send synchronizes less: it proves the value entered the queue,
not that anyone took it out.

Blocking is not a flaw to engineer away — it is how channels coordinate work
without locks. But it does mean you can deadlock yourself in one line:

```go
func main() {
	ch := make(chan int)
	ch <- 1              // blocks forever: the receive below can never run
	fmt.Println(<-ch)
}
```

```text
fatal error: all goroutines are asleep - deadlock!
```

`main` is one goroutine executing top to bottom. The send waits for a
receiver, but the only receive in the program is *after* the send in the same
goroutine. The runtime notices every goroutine is stuck and crashes with the
error above — learn to read it; you will produce it at least once in the
exercise. The fix is a second goroutine (as in the snippet further up), not a
buffer: `make(chan int, 1)` makes this toy compile-and-run, but in real code
a buffer just delays a deadlock until the buffer fills. Default to unbuffered
channels; add a buffer only when you can name the reason (a known burst size,
decoupling a fast producer from a slow consumer) — never as a deadlock "fix".

## Closing: the end-of-stream signal

A sender tells receivers "no more values are coming" with `close(ch)`. The
rules are few and strict:

- **Only the sender closes.** A receiver closing would make the sender's next
  send panic. When several goroutines send on one channel, none of them can
  safely close it — coordinating that is next-stage material; in this lesson
  every channel has exactly one sending goroutine, which closes it.
- **Receiving from a closed channel never blocks.** It first drains any
  buffered leftovers, then yields the element type's zero value, forever.
- **The comma-ok form tells you which you got.** `v, ok := <-ch` — `ok` is
  `false` only when the channel is closed *and* empty. Without it you cannot
  distinguish a genuine `0` from "closed".
- **`range` does the loop for you.** `for v := range ch` receives until the
  channel is closed and drained, then exits. This is why closing matters: a
  `range` over a channel nobody closes blocks forever.
- **Sending on a closed channel panics.** So does closing an already-closed
  channel. Close is a one-shot, one-owner operation.

One myth to kill early: you do *not* need to close every channel. Channels
are garbage collected whether or not they were closed — `close` is not
cleanup, it is a broadcast signal, and you send it only when receivers need
it (a `range` downstream, or a cancellation, below).

## Pipelines

Chain the pieces and you get the pipeline: stages connected by channels, each
stage a goroutine that receives from upstream, transforms, and sends
downstream. Here is a source and a stage:

```go
func gen(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			out <- n
		}
	}()
	return out
}

func double(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			out <- 2 * v
		}
	}()
	return out
}

for v := range double(gen(1, 2, 3)) {
	fmt.Println(v) // 2, 4, 6
}
```

Study the shape — you will reproduce it from memory for years:

- Each stage **makes and owns its output channel**; it is the only sender,
  so it is the one that closes — `defer close(out)` first thing in the
  goroutine, so the signal fires however the goroutine exits.
- The sends happen **inside the goroutine**. Move `out <- n` outside it and
  `gen` blocks on the first send — the caller hasn't received yet, because
  `gen` never returned. (This is the one-line deadlock again, in disguise.)
- **Termination cascades.** `gen` closes → `double`'s `range` ends → its
  `defer` closes → `main`'s `range` ends. Delete any one `close` and every
  stage downstream of it blocks forever — a *goroutine leak*. The runtime's
  deadlock detector only fires when **all** goroutines are stuck, so in a
  real program a leak is silent. The exercise tests guard every wait with a
  timeout so a missing close fails loudly instead of hanging your test run.

## select: waiting on several channels at once

A receive waits on one channel. `select` waits on many operations and runs
whichever becomes ready first:

```go
select {
case v := <-results:
	fmt.Println("got", v)
case err := <-errs:
	fmt.Println("failed:", err)
}
```

If no case is ready, `select` blocks. If **several** are ready, it picks one
**uniformly at random** — not top to bottom. That is deliberate (it prevents
one busy channel from starving the others), and it means you must never
encode priority as case order.

Adding a `default` case changes the contract: if nothing is ready, `default`
runs immediately and `select` never blocks. That is how you write a
*non-blocking* channel operation:

```go
select {
case v := <-ch:
	use(v)
default:
	// nothing ready right now; do not wait
}
```

Use `default` for a genuine "try once" — a loop spinning on a `select` with
`default` burns a CPU core polling; if you catch yourself doing that, you
wanted a blocking `select` after all.

### Cancellation: the done channel

`select`'s most important job in this arc is making a goroutine *stoppable*.
The idiom: give it a channel whose only purpose is to be closed.

```go
func worker(done <-chan struct{}, out chan<- int) {
	for n := 1; ; n++ {
		select {
		case out <- n:
		case <-done:
			return
		}
	}
}
```

While `done` is open, its case is never ready and the worker keeps sending.
The moment someone runs `close(done)`, the `<-done` case becomes ready —
remember, receiving from a closed channel never blocks — and the worker
returns. Closing as *broadcast* is the superpower here: one `close` releases
every goroutine selecting on `done`, which no single send could do. The
element type `struct{}` says "no data, pure signal". The context lesson two
steps ahead standardizes exactly this mechanism.

## nil channels: the off switch

A channel variable you never `make` is `nil`, and *every* send or receive on
it blocks forever. Outside `select` that is a bug you will meet — a forgotten
`make` produces a program that hangs, not an error message.

Inside `select`, it is a tool. A case on a nil channel is never ready, so it
is effectively **disabled** — and you can disable it *at runtime* by
assigning `nil` to the channel variable. This is how you merge two channels
until both are exhausted:

```go
case v, ok := <-a:
	if !ok {
		a = nil   // a is done: its case goes dormant for good
		continue
	}
	out <- v
```

Keep looping while `a != nil || b != nil`, with a case like this for each
input. Why not just leave the exhausted channel in place? Because a closed
channel is the exact opposite of nil: **always ready**, yielding zero values.
Left in the `select`, the loop spins on it — burning CPU or, worse,
forwarding garbage zeros downstream. Flip the channel from "closed" to `nil`
and its case falls silent.

Here is the whole rulebook on one card:

| Operation | nil channel | open channel | closed channel |
|-----------|-------------|--------------|----------------|
| `ch <- v` | blocks forever | blocks per buffering rules | **panics** |
| `<-ch` | blocks forever | blocks until a value arrives | never blocks: leftovers, then zero values, `ok == false` |
| `close(ch)` | **panics** | succeeds | **panics** |

If you can reproduce this table and say *why* each cell is what it is, you
own the material.

## Channels and the race detector

Channel operations are synchronization: a send *happens before* the
corresponding receive completes, so passing a value — even a pointer —
through a channel is a safe handoff, provided the sender stops using it
afterwards. Transfer ownership, don't share it. The goroutines lesson's
standing rules apply: run every test with `-race`, and no `time.Sleep` as
synchronization, ever — if a test only passes with a sleep in it, the
synchronization is wrong.

## Exercise

Open [`exercise/`](exercise/) — a module with `channels.go` (your work sites,
marked `TODO`) and `channels_test.go`. Read the tests first: every wait in
them is guarded by a generous timeout, so the classic channel mistakes
(forgot the goroutine, forgot the close) fail with a message naming the
mistake instead of hanging your terminal.

You will build six functions:

1. **`Generate(nums ...int) <-chan int`** — a pipeline source: emits each
   value in order, then closes.
2. **`Square(in <-chan int) <-chan int`** — a pipeline stage: squares each
   value, closing its output when the input is exhausted.
3. **`Sum(in <-chan int) int`** — a pipeline sink: totals the stream and
   returns when it ends.
4. **`TryRecv(ch <-chan int) (int, bool)`** — a non-blocking receive built
   on `select` with `default`.
5. **`Counter(done <-chan struct{}) <-chan int`** — emits 1, 2, 3, … until
   `done` is closed, then closes its output.
6. **`MergeTwo(a, b <-chan int) <-chan int`** — merges two channels with a
   `select` loop, disabling each input via `nil` as it closes.

Acceptance criteria:

1. `Generate(1, 2, 3)` returns immediately, emits 1, 2, 3, then closes;
   `Generate()` closes without emitting.
2. `Square` squares every value and closes its output when its input closes.
3. The composed pipeline `Sum(Square(Generate(1, 2, 3)))` returns 14 and
   terminates.
4. `TryRecv` never blocks: a ready value yields `(v, true)`; an empty, nil,
   or closed-and-drained channel yields `(0, false)`; a buffered value in a
   closed channel is still delivered as `(v, true)`. No special-casing nil —
   one `select` covers everything.
5. `Counter` keeps counting while `done` is open and closes its output once
   `done` is closed, so consumers ranging over it terminate.
6. `MergeTwo` delivers every value from both inputs (any order) and closes
   its output only after *both* inputs close. It must survive a producer
   that alternates between the two channels — draining one input completely
   before touching the other deadlocks; the tests check this.
7. `go test -race ./...` passes and the code is `gofmt`-clean. No
   `time.Sleep` anywhere in your solution.

Run the tests from inside `exercise/`, always with the race detector on:

```sh
cd exercise
go test -race ./...
```

They fail on the starter — make them green.

## Further reading

- [Go Blog — Go Concurrency Patterns: Pipelines and cancellation](https://go.dev/blog/pipelines)
  — this lesson's patterns, extended to fan-out/fan-in; read after passing.
- [Effective Go — Channels](https://go.dev/doc/effective_go#channels)
- [Go Blog — Share Memory By Communicating](https://go.dev/blog/codelab-share)
- [Go Spec — Select statements](https://go.dev/ref/spec#Select_statements) —
  the precise rules, including the random choice among ready cases.
