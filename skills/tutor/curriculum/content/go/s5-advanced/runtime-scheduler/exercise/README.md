# Exercise — Watch the scheduler work

A guided lab: you run five small programs, observe the runtime with its own
instruments, and explain what you see. Record every answer in `NOTES.md` — it
has a slot for each — then walk your tutor through it. There are no automated
tests; the discussion is the verification.

Ground rules:

- Work from inside this directory. The labs assume Linux or macOS (two use
  the raw `pipe(2)` syscall).
- Numbers vary between runs and machines. You are explaining *shapes* —
  which counter grows, which stays flat, and why — never reproducing digits.
- Build binaries first, then run the binaries. `go run` would work, but the
  `go` command is itself a Go program: under `GODEBUG=schedtrace` it prints
  its *own* scheduler lines and pollutes your observations.

```sh
go build -o bin/ ./...
```

## Part 1 — Read the machine: schedtrace

Run more spinners than you have cores, with the census printing twice a
second:

```sh
GODEBUG=schedtrace=500 ./bin/spinlab -g 32 -for 5s
```

Each stderr line looks like:

```
SCHED 2004ms: gomaxprocs=8 idleprocs=0 threads=13 spinningthreads=0
              needspinning=0 idlethreads=4 runqueue=9 [3 2 4 2 3 3 2 4]
```

Paste one line into `NOTES.md` and decode it: what are `gomaxprocs`,
`idleprocs`, `threads`, `runqueue`, and the numbers in brackets? Why is
`threads` bigger than `gomaxprocs` even though only `gomaxprocs` goroutines
can run at once? With 32 runnable Gs and all queues busy, where are the ~24
that aren't running right now?

All 32 goroutines were spawned by *one* goroutine (main) — so they all
started life near one P. The brackets show them spread across every P. Name
the mechanism that did that.

## Part 2 — GOMAXPROCS: parallelism vs concurrency

Same program, two runs, same duration:

```sh
GOMAXPROCS=1 ./bin/spinlab -g 8 -for 5s
./bin/spinlab -g 8 -for 5s
```

Record total iterations and the min/max per-goroutine spread for both runs.
Then answer in `NOTES.md`:

1. Roughly how did total throughput scale with the P count, and why?
2. With `GOMAXPROCS=1`, did all 8 goroutines make progress? What does that
   tell you about what GOMAXPROCS bounds — and what it doesn't?
3. The 8 spinners interleaved on one P even though none of them ever blocks
   on a channel or sleep. What made them take turns? (You'll test your answer
   in Part 4.)

## Part 3 — Threads are not goroutines

Park 64 goroutines two different ways and watch the `threads=` field:

```sh
GODEBUG=schedtrace=1000 ./bin/blocklab -g 64 -mode sleep -for 10s
GODEBUG=schedtrace=1000 ./bin/blocklab -g 64 -mode syscall -for 10s
```

Record the stable `threads=` count for each mode. Then, in `NOTES.md`:

1. 64 sleeping goroutines cost almost no threads. Which runtime mechanism
   parks them, and where is each goroutine "stored" while it waits?
2. 64 goroutines blocked in `read(2)` cost ~64 extra threads. Write out the
   full story for *one* of them, in order: what happens to its M, who notices,
   what happens to its P, and where the new threads come from.
3. Read `blockInRead` in `blocklab/main.go`. If it had used `os.Pipe` instead
   of raw `syscall.Pipe`/`syscall.Read`, the thread count would have stayed
   flat. Why? (Which runtime component handles `os.Pipe` fds?)
4. Extrapolate: this program at `-g 20000` would not just be slow — it would
   crash. What limit does it hit? (Try it if you're curious — it's your
   machine — but reason it out first.)

## Part 4 — Preemption: taking the P back

Three runs of the same two-goroutine program — a call-free spinner vs main,
on a single P:

```sh
./bin/preemptlab
GODEBUG=asyncpreemptoff=1 ./bin/preemptlab        # hangs — Ctrl-C to kill it
GODEBUG=asyncpreemptoff=1 ./bin/preemptlab -call
```

Record what each run did. Then, in `NOTES.md`:

1. Run 1 finishes. Which mechanism preempted the spinner, how is it
   delivered, and roughly how long can a G run before it fires?
2. Run 2 hangs even though main's 100ms sleep expired long ago. Walk through
   why main can never run again — and connect it to life before Go 1.14.
   Why was this worse than an unfair scheduler (think GC)?
3. Run 3 turns preemption *back on* without the signal — by changing the loop
   body. Read `tick` in `preemptlab/main.go`: what does the function call put
   inside the loop that the runtime can use? Why must it be `//go:noinline`?

## Part 5 — The price of a goroutine

Measure stack memory per parked goroutine, then grow the stacks:

```sh
./bin/costlab -g 100000
./bin/costlab -g 100000 -depth 100
```

Record bytes-per-goroutine for both. Then, in `NOTES.md`:

1. How does the depth-0 number compare to the ~2 KiB starting stack from the
   lesson? What else besides the stack does a G need?
2. What did `-depth 100` change, and what does the runtime do when a
   goroutine outgrows its stack? What happens to pointers into the old stack?
3. The program compares against threads at 1 MiB apiece. Give the two-line
   argument for goroutine-per-request in a typical HTTP server (your S5
   servers do exactly this).
4. Now argue the other side: describe one concrete workload where you would
   *bound* goroutine creation, and which S3 pattern you'd reach for. "They're
   cheap" is not allowed as an argument in either direction — talk about what
   each goroutine holds or blocks.

## Part 6 — The execution tracer, then a diagnosis

Capture and open a trace:

```sh
./bin/tracelab
go tool trace trace.out
```

Your browser opens the trace UI; pick "View trace by proc". Find the three
labeled phases (they're regions under the "workload" task; the phase
boundaries are also obvious from the shapes). Record in `NOTES.md`:

1. Phase 1 runs 8 CPU-bound goroutines. How many PROC rows are busy at any
   instant, and what does that number equal? What do the color-block
   boundaries within one row represent?
2. Phase 2: find the goroutine blocked in a syscall (~150ms). How does the
   tracer render it, and what are the four sleepers doing in comparison?
3. Phase 3 moves 50k values through unbuffered channels. Describe the shape —
   why so many tiny slices? What cost is the tracer making visible that
   pprof's CPU profile (S3) would largely hide?

Finally, the diagnosis — no tools, just the model. Write your answer in
`NOTES.md`:

> A production Go service (8 cores, `GOMAXPROCS=8`) normally runs ~40 OS
> threads. After a dependency upgrade, p99 latency spikes appear.
> `schedtrace` samples show `threads=` climbing through 1,800 while
> `runqueue` and the per-P queues stay near zero. Days later an instance
> crashes: `runtime: program exceeds 10000-thread limit`.

1. Near-zero run queues rule out one whole family of causes. Which, and why?
2. What is the only mechanism in the GMP model that makes `threads` grow
   without bound? Name two concrete culprits a dependency upgrade could have
   introduced.
3. What evidence would you collect to find the culprit call site?
4. Propose two distinct fixes. For each: what does it change about the
   threads, and what does it trade away?

Done? Bring `NOTES.md` to the discussion — your tutor will ask you to tell
several of these stories with the lab output in front of you.
