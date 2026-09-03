# Tutor notes — Runtime & Scheduler

## Where the learner is

Mid-S5. They have shipped HTTP servers, REST services, gRPC, and
production-style database code, and they carry the full S3 concurrency arc
(goroutines, channels, sync, context, patterns, race detector) plus S3
tooling's pprof basics. What they have *never* done is look under `go f()`.
This lesson is discussion-verified: the win condition is a correct,
defensible mental model, not code. Grade from `NOTES.md` and, crucially, from
stories told aloud — every objective is a narrative ("trace what happens
when…"), so the protocol below is retell-based, not quiz-based.

Discussion protocol: have their `NOTES.md` and lab output open. (1) Pick one
part and have them re-run it live, narrating the output. (2) Require the
blocking-syscall story and the preemption story told end to end without
notes — these two are the non-negotiable core. (3) Probe transfer with the
Part 6 diagnosis and at least two grilling points below. (4) Numbers in
NOTES.md are machine-specific; never grade against expected digits, only
against explanations of shape. If a lab misbehaved on their machine (e.g.
Windows/WSL quirks with the pipe labs), the model still gates: they must be
able to say what *should* happen and why.

## Common misconceptions

- **"GOMAXPROCS is the max number of goroutines."** The most common one.
  It's the number of Ps — simultaneous *execution* slots. Part 2 exists to
  kill this: 8 goroutines all progressed on 1 P. If it survives the lab,
  re-anchor with concurrency-is-structure vs parallelism-is-execution.
- **"A goroutine is a lightweight thread."** Acceptable as a slogan, fatal as
  a model — it hides the M:N mapping and predicts wrong answers for Part 3.
  A goroutine is a runtime-managed stack + state that *borrows* threads.
- **"time.Sleep / channel waits tie up a thread."** Thread-pool intuition
  from other ecosystems. Parked Gs cost memory only; blocklab's sleep mode
  shows a flat thread count. Related: believing 10k idle connections need
  10k threads — the netpoller is the missing concept.
- **"Network I/O blocks a thread like any syscall."** The inverse error.
  Sockets go through the netpoller (G parks, M freed); raw syscalls and cgo
  trap the M. If they can't articulate this asymmetry, Part 3 question 3
  (os.Pipe vs syscall.Pipe) is the remediation.
- **"Raising GOMAXPROCS fixes thread explosions / more Ps = faster."**
  Threads created for syscall hand-off have nothing to do with the P count;
  and extra Ps beyond cores buy contention, not speed. Probe with the Part 6
  scenario: "would GOMAXPROCS=64 help?" (No — runqueues were already empty.)
- **"Preemption prevents data races" / "before 1.14 Go couldn't multitask."**
  Preemption is about fairness and GC progress, not memory safety — the race
  rules from S3 are unchanged. And cooperative preemption at call sites
  worked fine pre-1.14 for any code that called functions; only call-free
  loops starved the scheduler.
- **"Goroutines are free, so unbounded spawning is fine."** The lesson's
  closing trap. The G is ~2-4 KiB, but what it *holds* (buffers, DB conns
  from their S5 pool work) and what it *blocks* (threads, via syscalls) is
  the real budget. Accept "cheap" only when paired with "…and bounded, or
  parked on the netpoller."

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go
deeper):

- "Tell the blocking-syscall story with the letters: G8 calls read(2) on an
  un-pollable fd. Narrate M, P, sysmon, and the return path. Where does G8 go
  if no P is free when the read returns?" (Global queue; its M parks. This
  story is objective 1 — require it fluent.)
- "Why does P exist? Rebuild the argument for why Gs and Ms alone weren't
  enough." (Bound on parallelism + lock-free local run queues + per-P caches;
  the pre-1.1 global-mutex design collapsed under cores.)
- "Main spawns 32 spinners, so they start near one P. Thirty seconds later
  every P is busy. Walk me through the steal: who steals, when, how much?"
  (An M whose P has an empty local queue, after checking global queue and
  netpoller, takes half a victim P's queue.)
- "Your preemptlab run 2 hung. Same binary, GOMAXPROCS=2 — what happens and
  why?" (It finishes: the spinner monopolizes one P, main runs on the other.
  Tests whether they reason from the model or memorized the demo.)
- Part 2 question 3: "Your 8 spinners never touch a channel or a sleep, yet
  they took turns on one P. Which mechanism, and how would you prove it?"
  (Asynchronous preemption. `spinlab`'s loop is deliberately call-free — an
  atomic load and an atomic add are instructions, not calls — so there is no
  prologue for the runtime to poison; sysmon signals the thread after ~10ms.
  The proof is Part 4: that same shape of loop hangs under
  `asyncpreemptoff=1`. If they answer "cooperative preemption", ask them to
  point at the function call.)
- "Why is a call-free loop immune to *cooperative* preemption specifically?
  What exactly does the runtime poison, and where is it checked?" (The stack
  bound checked in function prologues; no calls, no checks. Async preemption
  bypasses it with a signal.)
- "The costlab depth-0 number is ~2-4 KiB, not exactly 2048. Give two
  reasons." (G descriptor and bookkeeping beyond the stack; adaptive initial
  stack sizing; measurement noise via StackSys granularity — any two.)
- "Your S5 HTTP server takes a spike of 50k concurrent requests. Each handler
  does a SELECT through your sqlite pool. Where does the bound on real work
  actually come from, and are 50k handler goroutines a problem?" (The DB
  pool bounds concurrency; 50k parked Gs ≈ 100-200 MB stacks — fine-ish, but
  they should mention memory and the option of limiting in-flight requests.)
- Part 6 diagnosis follow-ups: "Why do near-zero runqueues rule out CPU
  starvation?" — "Name the two fixes' trade-offs explicitly." (Semaphore
  around the blocking calls: bounds threads, adds queuing latency; replacing
  the blocking API with a pollable/pure-Go one: removes the mechanism, costs
  migration risk. Raising the 10k limit is a bigger blast radius, not a fix —
  push back if offered as one.)

## Grading rubric

- **A** — Both core stories (syscall hand-off, preemption evolution) told
  fluently without notes; schedtrace line decoded field by field; Part 6
  diagnosis names the mechanism, plausible culprits (cgo library, DNS
  resolver change, file I/O, un-pollable fds), concrete evidence (execution
  trace syscall view, pprof threadcreate profile, bisecting the upgrade), and
  two fixes with trade-offs; the bounded-vs-unbounded goroutine argument is
  made from held resources and blocked threads, not "cheap"; transfer
  questions (GOMAXPROCS=2 variant, os.Pipe question) answered by reasoning,
  not recall.
- **B** — All six parts done honestly and the model is sound, but one story
  needs a nudge (e.g. forgets sysmon or where the returning G goes), or one
  trace observation is superficial, or a Part 6 fix lacks its trade-off.
  Solid after a single follow-up.
- **C** — Labs run but narration reads back the README; can define G, M, P
  but the syscall story falls apart mid-telling, or GOMAXPROCS-bounds-
  goroutines resurfaces under pressure. Pass only if live remediation lands:
  re-tell the syscall story with the ladder below, then re-answer the Part 6
  mechanism question cold.
- **Fail** — Labs not actually run (NOTES paraphrases the lesson, no pasted
  output), or the core model is absent: goroutines equated with threads,
  sleep believed to hold a thread, preemption explained as a safety feature.
  Redo Parts 1, 3, and 4 together at the terminal, then re-discuss.

## Remediation ladder

1. "Point at your own schedtrace line. Which field is threads, which is Ps?
   They differ — so what must some of those threads be doing, if only
   gomaxprocs of them can run Go code?"
2. Draw the invariant: G-on-M-holding-P. "Now G8 enters read(2) and the
   kernel won't give the thread back. Which letter is stuck? Can the P do
   anything while attached to it? So what *must* the runtime do with the P?"
3. For preemption: "In run 2, list every goroutine and what it's waiting
   for. Main's timer fired — what does main still need that it can never
   get? Now add the 10ms signal from run 1 — what changes?"
4. Re-run blocklab in both modes side by side and have them read the
   threads= field aloud every second, narrating the divergence as it happens
   — then immediately retry the full story unprompted.

## After passing

Preview: "Next lesson stays inside the runtime: the garbage collector — how
Go reclaims memory concurrently, what GOGC and GOMEMLIMIT tune, and how
allocation behavior shows up in your programs."
