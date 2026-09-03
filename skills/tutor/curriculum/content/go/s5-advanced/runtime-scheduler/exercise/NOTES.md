# My notes — fill every slot, then bring this to the discussion

## Part 1 — schedtrace census

- My pasted SCHED line:

```
(paste one line here)
```

- `gomaxprocs` means:
- `idleprocs` means:
- `threads` means, and why it exceeds gomaxprocs:
- `runqueue` (the bare number) means:
- The numbers in brackets mean:
- Where the ~24 runnable-but-not-running goroutines live:
- The mechanism that spread main's goroutines across all Ps:

## Part 2 — GOMAXPROCS

### Numbers

- `GOMAXPROCS=1` — total iterations / min / max:
- default — total iterations / min / max:

### Answers

1. How throughput scaled with P count, and why:
2. Did all 8 goroutines progress under `GOMAXPROCS=1`; what GOMAXPROCS bounds
   and what it doesn't:
3. What made never-blocking spinners take turns on one P:

## Part 3 — Threads are not goroutines

### Numbers

- `-mode sleep` stable `threads=`:
- `-mode syscall` stable `threads=`:

### Answers

1. What parks a sleeping goroutine, and where it waits:
2. The blocking-syscall story for one goroutine (M → sysmon → P → new
   threads), in order:
3. Why `os.Pipe` would have kept threads flat:
4. The limit `-g 20000` would hit:

## Part 4 — Preemption

### What each run did

- Run 1 (default):
- Run 2 (`asyncpreemptoff=1`):
- Run 3 (`asyncpreemptoff=1 -call`):

### Answers

1. Run 1's mechanism, its delivery, and the ~time before it fires:
2. Why run 2 hangs; life before Go 1.14; why a stuck loop endangered GC:
3. What the function call in run 3 puts in the loop; why `//go:noinline`:

## Part 5 — Goroutine cost

### Numbers

- bytes/goroutine at depth 0:
- bytes/goroutine at depth 100:

### Answers

1. Depth-0 number vs the ~2 KiB starting stack; what else a G needs:
2. What `-depth` changed; how stacks grow; what happens to pointers:
3. The case *for* goroutine-per-request in an HTTP server:
4. One workload where I'd bound goroutine creation, and the S3 pattern I'd
   use (arguing from what each goroutine holds or blocks):

## Part 6 — Tracer and diagnosis

### Trace observations

1. Busy PROC rows in phase 1, what the count equals, what slice boundaries
   are:
2. How the syscall-blocked goroutine renders in phase 2, vs the sleepers:
3. Phase 3's shape; the cost the tracer shows that a CPU profile hides:

### Diagnosis: the thread explosion

1. What near-zero run queues rule out, and why:
2. The only mechanism that grows `threads` unbounded; two culprits an
   upgrade could introduce:
3. Evidence I would collect to find the call site:
4. Fix 1 — what it changes, what it trades away:
5. Fix 2 — what it changes, what it trades away:
