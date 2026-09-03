# Runtime & Scheduler

> `go.advanced.runtime-scheduler` · ~2-3h · Stage: Advanced Go

## Objectives

By the end of this lesson you can:

- Explain the GMP model (goroutines, OS threads, processors) and trace what
  happens when a goroutine makes a blocking syscall.
- Explain how work stealing distributes goroutines across Ps and why
  GOMAXPROCS bounds parallelism but not concurrency.
- Describe how cooperative and asynchronous preemption work, and why a tight
  CPU loop no longer starves the scheduler since Go 1.14.
- Estimate the cost of a goroutine (initial stack size, growth) and argue
  when spawning per-request goroutines is or is not appropriate.
- Diagnose a scheduler-related symptom (e.g. latency spikes from thread
  blocking) by reasoning from the GMP model.

## Why Go ships its own scheduler

Since S3 you have written `go f()` hundreds of times and trusted that it's
cheap. Time to see why. The operating system already has a scheduler — it
juggles *threads* — so why does Go bring its own?

Because OS threads are expensive in exactly the ways servers can't afford. A
thread reserves megabytes of stack, and switching between threads means a trip
into the kernel: save registers, switch page tables' worth of context, come
back. Tens of thousands of threads is a machine-wide event. Tens of thousands
of concurrent requests is a Tuesday.

Go's answer is **M:N scheduling**: multiplex a huge number of cheap user-space
goroutines onto a small number of OS threads, and do the switching in user
space, in nanoseconds, without asking the kernel. The component that does this
is the runtime scheduler, and its vocabulary has three letters you need.

## The GMP model

- **G — goroutine.** What `go f()` creates: a tiny stack (a couple of KiB), a
  program counter, and scheduling state. The thing you have millions of.
- **M — machine.** An OS thread, created and owned by the runtime. The thing
  that actually executes instructions.
- **P — processor.** A *scheduling token*, not a piece of hardware. A P holds
  a **local run queue** of runnable Gs plus per-P caches (like the memory
  allocator's). An M must hold a P to execute Go code. There are exactly
  `GOMAXPROCS` Ps.

The invariant to memorize: **running Go code = one G on one M holding one P.**
Everything else in this lesson falls out of it.

```
 GOMAXPROCS = 2

   P0 ──runq──▶ [G7][G2][G9]        P1 ──runq──▶ [G4]
   │                                │
   M0 ◀─ executing G1               M1 ◀─ executing G3

   global run queue: [G12][G13]     netpoller: G5, G6 (waiting on sockets)
   parked Ms: M2, M3                blocked in syscall: M4 (with G8, no P)
```

Why does P exist at all — why not just Gs and Ms? Two reasons. First, it
bounds parallelism cleanly: at most `GOMAXPROCS` Ms can run Go code
simultaneously, no matter how many threads exist. Second, it kills lock
contention: because a P is owned by one M at a time, pushing and popping its
local run queue needs no lock. A single global queue guarded by a mutex was
exactly the design Go abandoned in 2012 — it melted under many cores.

## Finding work: stealing keeps everyone busy

When an M's current G blocks or finishes, the M asks its P for the next G:

1. Pop from the P's **local run queue** — the fast, lock-free path.
2. Occasionally (every 61st schedule) check the **global run queue** anyway,
   so neglected Gs there can't starve behind an always-full local queue.
3. Local queue empty? Check the global queue, then ask the **netpoller**
   whether network I/O completed and made some G runnable.
4. Still nothing? **Steal half** the run queue of another, randomly chosen P.

Step 4 is work stealing, and it's why you never place goroutines "on" a
processor: you just make Gs runnable, and idle Ps pull work toward
themselves. A burst of goroutines spawned by one P redistributes across all
Ps in a few scheduling ticks — you will watch this happen in the exercise.

This also explains the phrase you must be able to defend: **GOMAXPROCS bounds
parallelism, not concurrency.** With `GOMAXPROCS=1` you can still have 100,000
goroutines in flight — sleeping, waiting on channels, queued, interleaving.
Concurrency is *structure* (how many things are in progress); parallelism is
*execution* (how many run at the same instant), and only the latter is capped
by the P count. The default is the machine's CPU core count — but note
*which* count: historically the host's, cgroup limits ignored, so a Go
process in a 2-CPU container cheerfully set `GOMAXPROCS` to the host's 64 and
spent its life throttled, which is why the `automaxprocs` library exists.
Go 1.25 taught the runtime to read the container CPU limit itself, but only
for modules whose `go` directive is 1.25 or later — this exercise's `go 1.22`
module still gets plain `NumCPU`. The env var `GOMAXPROCS=n` overrides all of
it, and exists mostly so you can experiment — and you will.

## When a goroutine blocks: three very different fates

"Blocked" is one word for three mechanically different situations. Telling
them apart is the single most useful production skill in this lesson.

**1. Blocked on the runtime: channels, mutexes, timers.** `<-ch`,
`mu.Lock()`, `time.Sleep` — the runtime itself implements the wait. The G is
parked in a wait list, costing only its memory; the M immediately picks up
another G. A million sleeping goroutines need roughly zero threads. This is
why `time.Sleep` in a goroutine is harmless where sleeping a thread would be
a catastrophe.

**2. Blocked on network I/O.** Sockets would be thread-killers if each read
parked a thread. Instead the runtime routes all network fds through the
**netpoller** — an event loop over the kernel's readiness API (`epoll` on
Linux, `kqueue` on macOS). A G reading a socket with no data is parked and
its fd registered; the M moves on. When the kernel reports the fd ready, the
G goes back on a run queue. Result: 10,000 idle connections ≈ 10,000 parked
Gs ≈ a handful of threads. This is the mechanism that makes
goroutine-per-connection servers (your S5 HTTP work) viable at all.

**3. Blocked in a syscall or cgo call.** Here the runtime loses control: the
thread itself enters the kernel and won't come back until the call returns.
The M is trapped. The runtime's monitor thread (**sysmon**, which runs
without needing a P) notices the M stuck in a syscall and **hands its P to
another M** — waking a parked one or creating a fresh thread — so the other
Gs keep running. When the syscall finally returns, that M tries to reacquire
a P; if none is free, its G joins the global queue and the M parks.

Trace the story yourself, out loud — it's the first objective: G8 calls
`read(2)` on something the netpoller can't handle → M4 enters the kernel with
G8 → sysmon detaches P from M4 → P moves to M5 (woken or created) → other Gs
run on schedule → `read` returns → M4 wants a P back → none free → G8 to the
global queue, M4 parks.

The corollary: **each simultaneously blocking syscall or cgo call consumes a
whole OS thread.** File I/O on most setups, some cgo database drivers, DNS
via the cgo resolver, any C library call. A few dozen is fine. A few thousand
means thousands of threads — memory pressure, kernel scheduler load, and
eventually the runtime's default 10,000-thread limit, which crashes the
process. You will manufacture exactly this pathology, safely, in the lab.

## Preemption: how a G loses the CPU

Parking on channels and sockets is voluntary. What about a goroutine that
never blocks — a tight numeric loop? Someone has to take the P back.

**Cooperative preemption** came first. Every Go function's prologue contains
a stack-bound check (it exists for stack growth, below). The scheduler
abuses it: to preempt a G, it poisons the G's stack bound so the *next
function call* fails the check and detours into the runtime, which reschedules.
Cheap and safe — but it needs the G to call a function. A loop like
`for { x++ }` contains no calls, so before Go 1.14 it could hold a P
*forever*. Worse than unfair: garbage collection begins with a
stop-the-world phase that must pause every G, so one call-free loop could
stall GC — and with it the entire program.

**Asynchronous preemption** (Go 1.14) closed the hole. Sysmon spots a G that
has run for ~10ms and sends its thread a signal (`SIGURG` on Unix). The
signal handler suspends the G mid-instruction, no cooperation required. Your
mental model becomes: any goroutine can lose its P at nearly any instruction,
roughly every 10ms — scheduling is fair, GC can always start, and a stuck
loop is a CPU bill rather than an outage. In the lab you'll switch async
preemption off (`GODEBUG=asyncpreemptoff=1`) and watch a two-line program
travel back to 2013.

One thing preemption is *not*: a memory-safety mechanism. The interleavings
it creates were always allowed — your race-detector discipline from S3 is
what keeps them correct.

## What a goroutine costs

A new goroutine starts with a stack of about **2 KiB** (the runtime adapts
the exact figure to your program's behavior) plus a small descriptor. Compare
a thread's megabytes-plus-kernel-objects. That is three orders of magnitude,
and it is the entire economic basis of Go's concurrency style.

Small starting stacks only work because stacks can grow. The same prologue
check from the preemption story detects an about-to-overflow stack; the
runtime then allocates a bigger one (doubling), *copies* the old stack over,
rewrites pointers into it, and continues — invisible to you, up to a default
cap of 1 GB. Stacks that grew for a deep excursion can shrink again at GC.
The practical consequence: deeply recursive code makes goroutines cost tens
of KiB instead of two, which matters when you have a million of them.

So is goroutine-per-request right? Usually, yes — that's the design the
runtime is built for, and `net/http` does it for you. But "goroutines are
cheap" is not "goroutines are free", and the goroutine itself is rarely the
real cost. The honest questions are: what does each goroutine *hold*
(buffers, DB connections, file handles — remember your S5 pool limits), does
its work **block a thread** (syscalls, cgo — see above), and is the number of
them **bounded by anything** (a flood of spawns is a memory and scheduling
problem no matter how cheap each one is)? Unbounded fan-out gets a bound —
the semaphore-channel and worker-pool patterns you built in S3. Bounded work
over parked-on-I/O goroutines scales to numbers that would sound like typos
in a thread-based system.

## Watching the scheduler work

Three instruments, all built in — the exercise is built around them:

- **`GOMAXPROCS=n ./prog`** — set the P count for one run. Your experimental
  variable.
- **`GODEBUG=schedtrace=500 ./prog`** — the runtime prints a scheduler
  census to stderr every 500ms: P count, idle Ps, thread count, spinning
  threads, global queue length, and each P's local queue length. One line of
  it, read correctly, answers "is this program starved, blocked, or idle?"
- **The execution tracer** — `runtime/trace` records every scheduling event:
  which G ran on which P when, syscalls, GC, network wakeups.
  `go tool trace trace.out` opens a timeline UI in your browser. Where
  pprof (S3) tells you *where CPU time went*, the tracer tells you *why
  goroutines waited* — it is the ground truth for latency mysteries.

## Exercise

Open [`exercise/`](exercise/) — a guided lab, not a build task. `README.md`
walks you through six parts: five small programs to run and observe
(`spinlab`, `blocklab`, `preemptlab`, `costlab`, `tracelab`) plus a written
diagnosis. Record every observation and answer in `NOTES.md`; the discussion
with your tutor is the verification — there are no automated tests.

Acceptance criteria:

1. One `schedtrace` line from `spinlab` is pasted into `NOTES.md` and decoded
   field by field, in your own words, including what the per-P bracket
   numbers are.
2. The `GOMAXPROCS=1` vs default comparison is recorded with iteration
   counts, and `NOTES.md` states precisely what changed (parallelism) and
   what did not (concurrency — all goroutines progressed).
3. `blocklab`'s thread counts for `-mode sleep` vs `-mode syscall` are
   recorded, with the GMP explanation of the difference and the full
   blocking-syscall story (M, P hand-off, sysmon) written out.
4. All three `preemptlab` runs (default; `asyncpreemptoff=1`;
   `asyncpreemptoff=1 -call`) are recorded with an explanation of *which
   preemption mechanism* each run demonstrates.
5. `costlab`'s measured bytes-per-goroutine at two depths is recorded, plus
   your per-request-goroutine argument: one situation where you'd spawn
   freely and one where you'd bound, with reasons.
6. The Part 6 trace observations and the written diagnosis of the
   thread-explosion scenario are complete — the diagnosis must name the
   mechanism, the evidence you'd collect, and two distinct fixes.

Run everything from inside `exercise/`; the README gives exact commands per
part. Numbers will vary run to run and machine to machine — that's expected;
you are explaining shapes, not reproducing digits.

## Further reading

- [runtime package docs — GODEBUG and environment variables](https://pkg.go.dev/runtime#hdr-Environment_Variables)
- [Scalable Go Scheduler Design Doc (Dmitry Vyukov, 2012)](https://golang.org/s/go11sched)
- [Proposal: non-cooperative goroutine preemption](https://go.googlesource.com/proposal/+/master/design/24543-non-cooperative-preemption.md)
- [Go blog — More powerful Go execution traces](https://go.dev/blog/execution-traces-2024)
