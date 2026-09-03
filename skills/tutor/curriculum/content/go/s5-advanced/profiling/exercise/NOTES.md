# Profiling notes

Fill this in as you work — it is your evidence, and the tutor will ask
for it. Numbers without a profile line attached don't count.

## 1. Baseline

Paste the relevant `benchstat`-formatted baseline (from
`go test -bench=. -run='^$' -count=10 | tee before.txt`):

```
(paste before.txt highlights here)
```

## 2. Profile evidence

From `go tool pprof -top cpu.out` on the unmodified code: which
functions dominate, and how much? Paste the top 5 lines.

```
(paste pprof top lines here)
```

Which line did `go tool pprof -list TopUsers cpu.out` blame inside
`TopUsers`? What did the heap profile (`-sample_index=alloc_space`)
blame in `Render`?

> your answer:

## 3. Hypothesis

Before editing: what change do you expect to help, and roughly how much?

> your answer:

## 4. After

Paste `benchstat before.txt after.txt` output:

```
(paste benchstat comparison here)
```

Did the measurement match your hypothesis? Anything surprising?

> your answer:
