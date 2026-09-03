# Garbage collection notes

Fill this in as you work — it is your evidence, and the tutor will ask for
it. Exact numbers differ per machine; the *shapes* and the explanations are
what count.

## Part 1 — the pacer

Run each command and paste one representative `gc` line, then fill the table.

```sh
GODEBUG=gctrace=1 go run ./cmd/gcdemo
GODEBUG=gctrace=1 GOGC=50 go run ./cmd/gcdemo
GODEBUG=gctrace=1 GOGC=400 go run ./cmd/gcdemo
GODEBUG=gctrace=1 GOGC=off GOMEMLIMIT=100MiB go run ./cmd/gcdemo
```

| setting | GC cycles | heap goal | live after sweep | heap at exit |
|---------|-----------|-----------|------------------|--------------|
| `GOGC=50` | | | | |
| default (`GOGC=100`) | | | | |
| `GOGC=400` | | | | |
| `GOGC=off GOMEMLIMIT=100MiB` | | | | |

One `gc` line, decoded field by field in your own words:

```
(paste one gc line here)
```

> your decoding:

The live set barely moves across all four runs, but the heap goal changes a
lot. Why? Write the pacer's formula and check it against your numbers.

> your answer:

Which run spent the most CPU on garbage collection, and which held the most
memory? State the trade-off in one sentence.

> your answer:

`GOGC=off` disables the pacer entirely — so why did the fourth run still
collect, and more often than the default? What is `GOMEMLIMIT` doing, and
what would happen if the live set grew to 95 MiB under that same setting?

> your answer:

## Part 2 — the hot path

Baseline, before any edits (run without `-race`):

```sh
go test -bench=. -benchmem -run='^$'
```

```
(paste the three Benchmark lines here)
```

For each function, name the specific mistake and the allocation it causes per
event or per call:

> `FormatEvents`:
>
> `EventIDs`:
>
> `WriteEvents`:

After your fixes:

```
(paste the three Benchmark lines here)
```

| benchmark | allocs/op before | allocs/op after | B/op before | B/op after |
|-----------|------------------|-----------------|-------------|------------|
| `FormatEvents` | | | | |
| `EventIDs` | | | | |
| `WriteEvents` | | | | |

`WriteEvents` should end up at or near zero allocations per call in steady
state, while `FormatEvents` stays at one no matter how many events it renders.
Explain both numbers — why can one of them reach zero and the other cannot?

> your answer:

What would break if `WriteEvents` used a single package-level
`bytes.Buffer` instead of a pool? Which test catches it, and what does the
race detector say?

> your answer:
