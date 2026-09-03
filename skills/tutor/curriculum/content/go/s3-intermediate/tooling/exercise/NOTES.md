# My notes — fill every slot, then bring this to the discussion

## Part 1 — go vet findings (vetlab/)

### Finding 1

- vet check / message:
- What is actually wrong:
- Why the compiler accepted it:

### Finding 2

- vet check / message:
- What is actually wrong:
- Why the compiler accepted it:

### Finding 3

- vet check / message:
- What is actually wrong:
- Why the compiler accepted it:

### Finding 4

- vet check / message:
- What is actually wrong:
- Why the compiler accepted it:

### Earlier-lesson connections

- Which two findings tie back to the sync and context lessons, and what would
  go wrong at runtime:

## Part 2 — staticcheck findings (lintlab/)

### Finding 1

- Check code / family (SA, S, ST, U):
- What it caught:

### Finding 2

- Check code / family:
- What it caught:

### Finding 3

- Check code / family:
- What it caught:

### Finding 4

- Check code / family:
- What it caught:

### Compiler question

- Why does the compiler reject an unused local variable but tolerate an
  unused function:

## Part 3 — golangci-lint

- My `.golangci.yml` (paste it):
- The two `WriteReport` findings (exact lines, with linter name):
- Which linter caught them, and the production failure mode if unfixed:
- How I fixed them:

## Part 4 — pprof

### Benchmark numbers

- JoinNaive — ns/op, B/op, allocs/op:
- JoinBuilder — ns/op, B/op, allocs/op:

### Profiles

- Top functions in the CPU profile, and what they do:
- What the heap profile (alloc_space) shows:
- Why JoinNaive behaves this way (what `out += …` forces each iteration):
- Asymptotic cost of naive concatenation in a loop (S2 hat on):

### CPU vs heap

- In one sentence each: when would I capture a CPU profile, and when a heap
  profile:

## Part 5 — Modules field trip

1. What `go get rsc.io/quote@v1.5.2` brought in (`go list -m all`):
2. The `go mod why -m golang.org/x/text` chain, in my own words:
3. Why Go did not pick the newest sampler version — the rule at work:
4. MVS question — v1.3.0 required by a dependency, v1.3.1 by me; which wins
   and why:
5. What happened when I tampered with `go.sum`, and the attack this defeats:

## Part 6 — Choose the command

1. Stale requirement after deleting an import:
2. App building against local library checkout:
3. "Which version of x/text do we actually use?":
4. `//go:generate stringer` after constants changed:
5. "missing go.sum entry" in CI:
