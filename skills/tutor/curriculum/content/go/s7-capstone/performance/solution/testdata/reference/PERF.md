# PERF — listing notes by tag

## Question — what is slow, and how slow is acceptable?

`notes list --tag=urgent` is the operation a user waits for: it runs on every
invocation of the CLI, and everything else the program does is bounded by it.
The store holds up to 10,000 notes (NFR-2 in the PRD) and a typical tag matches
about one note in ten.

At 10,000 notes, `store.Memory.List("urgent")` cost **352,637 ns/op** and
allocated **802,816 B/op**. The goal for this pass: get the tagged listing
under 100,000 ns/op and under 200 KB per call at that size, without changing
what it returns or what it costs to add a note. One axis — latency of the
tagged read — with memory tracked because the two turned out to be the same
problem.

## Method — how it was measured

`BenchmarkListByTag` in `internal/store/bench_test.go`, with sub-benchmarks at
n=100 and n=10,000. The sweep is deliberate: the ratio between the two sizes
says whether a change moved the curve down or changed its shape.
`BenchmarkListAll` is the control — the index cannot help an unfiltered
listing, so that is where cost moved sideways would show up.

The store is seeded so that "work" matches every note and "urgent" matches one
in ten, which is the selectivity the CLI sees. Setup happens before
`b.ResetTimer()`, the result goes to a package-level sink so the call cannot be
optimized away, and `b.ReportAllocs()` is on because allocs/op and B/op are the
numbers that survive being run on someone else's hardware.

Ten runs of each, `go test -bench='BenchmarkList' -run='^$' -count=10
./internal/store`, no `-race` (its 2-20x instrumentation distorts the ratios),
on an idle Apple M1 Pro, darwin/arm64, Go 1.22. Raw output is committed as
`docs/perf/bench-before.txt` and `docs/perf/bench-after.txt`; the figures below
are medians of those ten samples. benchstat was not installed on this machine,
so the comparison has ranges instead of p-values — the min/max columns in the
raw files are what stands in for them, and they do not overlap.

## Evidence — what the profiles said

The CPU profile's `-top` view was useless on its own: half the samples sat in
`runtime.kevent` and `runtime.pthread_cond_wait`, which is the scheduler and
the collector, not a function anyone can fix. That is itself the finding — the
program was spending its time on garbage, so the heap profile was the one to
read.

`docs/perf/heap-top-before.txt` is unambiguous: **99.88% of all bytes
allocated** come from `store.(*Memory).List` — 23.34 GB over the benchmark run.
The line-level CPU view in `docs/perf/cpu-list-before.txt` shows where the rest
goes inside the function: 480 ms in the `m.byID[id]` map lookup and 380 ms in
`n.HasTag(tag)`, both executed once per note in the store, and `HasTag` calls
`strings.ToLower` on the query for every comparison.

So both costs had the same shape: work proportional to the number of notes
that *exist*, for an answer proportional to the number of notes that *match*.
The over-allocation was the same mistake in another dress —
`make([]note.Note, 0, len(m.order))` sizes the result for the whole store.

## Change — index the tag lookup

`Memory` now keeps `byTag map[string][]string`, tag to note ids in insertion
order, maintained by `Add` inside the same critical section that writes `byID`
and `order`. `List` reads the index when the tag is non-empty and falls back to
`order` when it is not, so the returned slice is allocated at the size of the
matches rather than the size of the store.

The hypothesis was narrow enough to be wrong in one place: if the profile was
right, the per-note map lookup, the per-note `HasTag`, and the oversized slice
all disappear together, and the unfiltered path does not move at all. One
change, one re-measure. Normalising the query tag moved into
`note.NormaliseTag`, so the index keys and `HasTag` cannot drift apart.

## Result — before and after

Medians of ten runs; `n` is the number of notes in the store.

| benchmark | before | after | change |
|---|---|---|---|
| `ListByTag/n=100` | 2,820 ns/op, 8,192 B/op, 1 allocs/op | 249 ns/op, 896 B/op, 1 allocs/op | -91.2% time, -89.1% bytes |
| `ListByTag/n=10000` | 352,637 ns/op, 802,816 B/op, 1 allocs/op | 35,920 ns/op, 81,920 B/op, 1 allocs/op | -89.8% time, -89.8% bytes |
| `ListAll/n=10000` (control) | 278,714 ns/op, 802,821 B/op | 279,651 ns/op, 802,821 B/op | +0.3%, within run-to-run spread |

Both goals are met: 35,920 ns/op against a target of 100,000, and 81,920 B/op
against 200 KB.

Read the ratio honestly. This is not a complexity-class win in the usual sense:
`List` was O(n) in the store size and is now O(m) in the number of matches, and
both are linear. What changed is *which* input the cost tracks. At the CLI's
10% selectivity that is a 10x constant; at 0.1% it would be nearer 1000x, and
for a tag that matches everything it is nothing at all — which is exactly what
the unchanged control row shows. Quoting "10x faster" without the selectivity
it was measured at would be a claim nobody could check.

## Trade-off — what the index costs

An invariant that now has to hold in two places: every write path touching
`byID` must touch `byTag` in the same critical section, or a listing silently
misses notes. Today there is one write path (`Add`); a future `Delete` or
`Retag` is where this bites, and the cross-check test below is the thing that
would catch it.

Memory: one extra string header per (note, tag) pair, 16 bytes each — about
11,000 pairs, so roughly 180 KB held at 10,000 notes, against ~700 KB not
allocated on every tagged listing. `Add` does a little more work per note, not
measured here because it is not on the path anyone waits for; if the project
ever becomes write-heavy that assumption is the first thing to re-measure.

The price is accepted because the sizes in the PRD make the read path the one
that matters and the write path rare. The flip condition is explicit: if
`Delete` arrives and index maintenance stops being a two-line addition, the
index moves behind its own type with its own tests, or it goes away.

## Correctness — proving behaviour did not change

`TestListByTagMatchesLinearScan` is the proof: it builds 500 notes with
pseudo-random tag combinations from a fixed seed, then compares `List(tag)`
against `linearScan`, the pre-index implementation kept in the test file as an
oracle. It runs every tag, two queries that need normalising (`"WORK"`,
`"  Urgent  "`), and one tag nobody used — the cases where an index and a scan
are most likely to disagree.

`TestListFiltersAndKeepsOrder` still pins insertion order and the empty-tag
case, `TestAddRejects` pins duplicate and missing ids, and `TestConcurrentUse`
exercises the new write path under `-race`, since `Add` now mutates two maps
under the same lock.
