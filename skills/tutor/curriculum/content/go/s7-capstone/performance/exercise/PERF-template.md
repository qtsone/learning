# PERF — <the operation you optimized>

Copy this file into your project root as `PERF.md`, replace every `TODO:` line
with your own writing, and keep the headings (the harness matches them loosely,
so `## Result — before and after` is fine, but `## Numbers I Got` is not).

Each section has to be at least a short paragraph. Write it for a reviewer who
was not there and does not trust you yet.

## Question

TODO: Name the operation, the input size, the number it costs today and the
number you wanted. One of latency, throughput or memory — pick one, because a
goal with three axes has none. Say why this operation is the one that matters:
who waits for it, how often, and what it blocks.

## Method

TODO: How you measured. The benchmark function and where it lives, the inputs
and why they resemble real ones, `-count=` and the machine, and what you did to
keep the machine quiet. Anything you know is unrepresentative goes here too —
that admission is what makes the rest believable.

## Evidence

TODO: What the profile said, in your words: the dominant cost, its share, and
whether it was flat or cumulative. Then point at the committed artifact by
path, e.g. `docs/perf/cpu-top.txt`, or paste the pprof output here. A profile
that only ever existed in your terminal is not evidence.

## Change

TODO: What you changed, in one paragraph a reviewer can follow without the
diff. Say which hypothesis it tested and why it should have helped. If you
tried something first that did not work, say so and say how you knew.

## Result

TODO: Before and after, as numbers with units, from repeated runs — benchstat
output pasted here is ideal. Include allocs/op and B/op alongside ns/op: they
are the figures that survive being run on someone else's hardware. State the
input size each number belongs to.

## Trade-off

TODO: What the speed cost — code that is harder to read, memory held longer,
an invariant that now has to be maintained in two places, a dependency, a
cache that can go stale. Then why that price is acceptable at the sizes in your
PRD, and the condition under which you would pay it back.

## Correctness

TODO: Name the test functions that would fail if this change altered
behaviour — `TestSomething`, by identifier, in backticks. At least one of them
should compare the fast path against the obvious one over generated input,
rather than re-asserting the answers you already believed.
