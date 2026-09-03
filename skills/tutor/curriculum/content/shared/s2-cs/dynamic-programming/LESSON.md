# Dynamic Programming Intro

> `shared.cs.dynamic-programming` · ~4-5h · Stage: CS Fundamentals

## Objectives

By the end of this lesson you can:

- Identify overlapping subproblems and optimal substructure as the signals
  that a problem admits dynamic programming.
- Implement a top-down memoized solution to a classic problem and show the
  complexity drop from exponential to polynomial.
- Implement a bottom-up tabulated solution to the same class of problem and
  pass the provided tests.
- Compare memoization and tabulation in terms of code shape, stack usage, and
  space optimization opportunities.
- Define the state, transition, and base cases for a new DP problem before
  coding it.

## When beautiful recursion goes wrong

The Fibonacci sequence starts `0, 1` and every later number is the sum of the
two before it: `0, 1, 1, 2, 3, 5, 8, 13, …`. The definition is a recurrence,
so the recursion lesson's recipe — base case plus self-call on smaller input —
translates directly:

```
fib(n):
    if n < 2: return n
    return fib(n-1) + fib(n-2)
```

Correct, three lines, and a performance disaster. Draw the calls for `fib(5)`:

```
                        fib(5)
                 /                \
            fib(4)                fib(3)
           /      \              /      \
       fib(3)     fib(2)     fib(2)     fib(1)
      /     \     /    \     /    \
  fib(2)  fib(1) …      …   …      …
```

`fib(3)` is computed twice, `fib(2)` three times, `fib(1)` five times — and
those repeat counts are themselves Fibonacci numbers, so the waste compounds.
Each `+1` to `n` multiplies the total work by about 1.6: exponential growth,
the worst class on the Big-O ladder from the start of this stage. Concretely,
`fib(30)` takes 2,692,537 calls; `fib(90)` would take about 9 × 10¹⁸ — years
of compute for a number that fits in a machine word.

Now the absurdity: between `fib(0)` and `fib(90)` there are only 91 *distinct*
subproblems. The recursion does years of work computing 91 different answers,
because it keeps re-deriving answers it already produced and threw away.

Dynamic programming is the fix, and it is one idea: **never solve the same
subproblem twice**. Everything else in this lesson is bookkeeping around that
idea.

## The two signals

Not every problem benefits. DP applies when you can spot both of these:

**Overlapping subproblems** — breaking the problem down reaches the *same*
subproblem many times, like `fib(2)` above. Contrast merge sort from the
sorting lesson: it also recurses, but the two halves it splits into are
disjoint — no subarray is ever sorted twice, so there is nothing to reuse and
caching would buy nothing. That family is divide-and-conquer. Same recursive
shape, opposite reuse profile.

**Optimal substructure** — the best answer to the whole problem is built from
best answers to subproblems. For `fib` this is trivial (there is only one
answer). It matters for optimization problems: the *fewest* coins that make
63¢ necessarily contain the fewest coins that make 63¢ minus the last coin —
if a cheaper way to make that remainder existed, you could swap it in and beat
an answer that was supposedly optimal.

Both signals present → cache subproblem answers and reuse them. There are two
mechanical ways to do that, and you will implement both.

## Top-down: memoization

Keep the recursion exactly as you derived it; add a cache in front:

```
memo = empty map                    # subproblem → answer

fib(n):
    if n < 2:        return n
    if n in memo:    return memo[n]         # check BEFORE computing
    memo[n] = fib(n-1) + fib(n-2)           # store AFTER computing
    return memo[n]
```

Two moves, easy to fumble separately: *check before computing* (or the cache
is never used) and *store after computing* (or the cache is never filled).
Miss either and you silently fall back to exponential time — the code still
returns correct answers, which is exactly why this bug survives casual
testing. The exercise tests count your computations to catch it.

The cache is a hash table — the structure you built earlier in this stage —
so check and store are O(1). Each distinct subproblem is now computed at most
once: `fib(n)` computes `n+1` subproblems, each combining two O(1) lookups.
Exponential has become **O(n)** time with O(n) space for the cache. That is
the whole trick: `fib(90)` drops from ~10¹⁸ calls to 91 computations.

One cost remains from the recursion lesson's call-stack model: the *first*
call still recurses all the way down before anything is cached, so the stack
grows to depth n. For `fib(90)` that is nothing; for n in the millions it can
overflow the stack. Remember this — it is the main argument for the second
technique.

## Bottom-up: tabulation

Memoization starts at the goal and works down. Tabulation flips the
direction: start at the base cases and iterate *up*, filling a table where
entry `i` holds the answer to subproblem `i`:

```
fib(n):
    table = array of size n+1
    table[0] = 0
    table[1] = 1
    for i from 2 to n:
        table[i] = table[i-1] + table[i-2]
    return table[n]
```

Same subproblems, same answers, same O(n) time and space — but a plain loop.
No recursion means no call stack to overflow, and usually less overhead per
subproblem. The price is that *you* must order the work so that everything an
entry depends on is already filled when you reach it. Here that is easy —
`i-1` and `i-2` come before `i` — but choosing the fill order is the part of
tabulation that requires actual thought in harder problems.

## Shrinking the table

Look at the loop body: computing `table[i]` reads only `table[i-1]` and
`table[i-2]`. Everything older is dead weight — so don't store it:

```
fib(n):
    if n < 2: return n
    prev2, prev1 = 0, 1                 # fib(0), fib(1)
    repeat n-1 times:
        prev2, prev1 = prev1, prev2 + prev1
    return prev1
```

O(n) time, **O(1)** space. The general rule: when the transition only reaches
back a bounded number of steps `k`, you only ever need the last `k` values.

Notice this is a tabulation-only trick. Because you control the fill order,
you know exactly when a value becomes garbage. A memoized version cannot
easily discard entries — the recursion revisits them in an order you don't
control.

Not every table shrinks. In the coin problem below, the transition reaches
back by each coin's denomination — arbitrarily far — so the whole table must
stay live. Whether the table can shrink is a property of the transition, and
you can read it off before writing any code.

## Memoization vs tabulation

|                      | Memoization (top-down)              | Tabulation (bottom-up)            |
|----------------------|-------------------------------------|-----------------------------------|
| Code shape           | recursive function + cache; mirrors the recurrence | loop filling a table |
| Evaluation order     | on demand, driven by the recursion  | chosen by you, smallest first     |
| Subproblems touched  | only the ones actually reached      | typically all of them             |
| Call stack           | grows with recursion depth — can overflow | none — it's a loop          |
| Space optimization   | hard; the cache keeps everything    | often easy — keep the last k values |

A sane default: *derive* top-down, because the memoized code is a
line-for-line copy of how you reasoned about the problem. Switch to bottom-up
when the recursion depth threatens the stack, when you would compute nearly
all subproblems anyway, or when you want the shrunken-table space win.

## The recipe: state, transition, base cases

Every DP solution you will ever write answers three questions. Answer them in
writing *before* coding — if you cannot state them, no amount of code will
rescue you:

1. **State** — what identifies one subproblem? For `fib` it is a single
   integer `i`: "the i-th Fibonacci number". The state is your cache key and
   your table index, and the number of distinct states bounds your running
   time.
2. **Transition** — how is one state's answer built from smaller states'
   answers? `table[i] = table[i-1] + table[i-2]`. This is the recurrence.
3. **Base cases** — which smallest states get direct answers, no recursion?
   `table[0] = 0`, `table[1] = 1`. Exactly the base cases the recursion
   lesson taught you to write first.

With those three on paper, memoized and tabulated versions are both
mechanical translations. Practice on a genuinely new problem:

### Worked example: fewest coins

Given coin denominations and a target amount, make the amount *exactly* with
as few coins as possible; report impossible if no combination works.

First instinct is usually greedy — take the largest coin that fits, repeat.
Try denominations `{1, 3, 4}` and amount 6: greedy takes 4, then 1, then 1 —
three coins. But `3 + 3` does it in two. Greedy commits to the big coin and
never reconsiders; the optimum required skipping it. You need to *consider
every coin as the possible last coin*, and that is exactly a recurrence:

- **State**: `best(a)` = fewest coins that make amount `a`.
- **Transition**: the last coin is some denomination `c ≤ a`; before adding
  it you had an optimal way to make `a − c` (optimal substructure). So
  `best(a) = 1 + min over every coin c ≤ a of best(a − c)`.
- **Base cases**: `best(0) = 0` — zero coins make zero. States with no valid
  move, or only impossible predecessors, are impossible (conceptually ∞; in
  code, a sentinel).

Overlap check: `best(6)` needs `best(5)`, `best(3)`, `best(2)`; `best(5)`
needs `best(4)`, `best(2)`, `best(1)` — `best(2)` already repeats. Both
signals present. Tabulating for `{1, 3, 4}`, amount 6, filling left to right:

```
a:        0   1   2   3   4   5   6
best[a]:  0   1   2   1   1   2   2
```

The last cell: via coin 1, `best[5] + 1 = 3`; via coin 3, `best[3] + 1 = 2`;
via coin 4, `best[2] + 1 = 3`. Minimum: 2. Each of the `amount + 1` cells
tries every coin, so the cost is **O(amount × #coins)** — polynomial, versus
the exponential tree of "try every sequence of coins".

In Go:

The memo cache is a `map[int]int` with the comma-ok idiom from the maps
lesson; the table is a slice from `make`. A recursive helper can carry the
cache and a counter explicitly — no globals needed:

```go
func fibMemo(n int, memo map[int]int, computed *int) int {
	if v, ok := memo[n]; ok {
		return v
	}
	*computed++
	// … compute, store in memo[n], return …
}
```

Two Go-specific traps for the exercise:

- **Overflow is silent.** Go's `int` is 64-bit on your machine; `fib(90)`
  fits, `fib(93)` wraps around into garbage with no error. The tests stop at
  90 deliberately.
- **Pick a safe "impossible" sentinel.** In `MinCoins`, marking impossible
  cells with `math.MaxInt` breaks the moment you compute `best[a-c] + 1` on
  one: max-int plus one wraps to a huge *negative* number that then wins every
  `min`. Use `amount + 1` instead — no real answer can use more than `amount`
  coins (the smallest coin is at least 1), so it is unreachable-large yet
  arithmetic-safe. Translate it to `-1` only when returning.

## Exercise

Open [`exercise/`](exercise/) — a Go module with `dp.go` (five `TODO`
functions) and `dp_test.go`. **Read the tests first**: they check answers
*and* count your work — call counts for the naive version, computed-subproblem
counts for the memoized one.

Acceptance criteria:

1. `FibNaive(n)` returns `fib(n)` and the total number of calls made — this
   call plus every recursive call, base cases included. The counts are exact:
   n=30 must report 2,692,537 calls. Feel the exponential before you kill it.
2. `FibMemo(n)` returns `fib(n)` and how many subproblems it actually
   computed (cache misses). n=90 must return 2880067194370816120, and the
   computed count must stay linear (at most 2n+2) — both are impossible
   without a working cache.
3. `FibTab(n)` computes the same values bottom-up with a table and no
   recursion, n=90 included.
4. `FibConstSpace(n)` computes the same values in O(1) extra space: no slice,
   no map, no recursion — two carried values. (Tests verify the values; your
   tutor verifies the space claim by reading the code.)
5. `MinCoins(coins, amount)` tabulates bottom-up: 0 for amount 0, −1 when the
   amount cannot be made, and it must beat greedy on the trap case
   (`{1,3,4}`, 6 → 2).
6. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They must FAIL before you write anything. Work in order — `FibNaive`,
`FibMemo`, `FibTab`, `FibConstSpace`, `MinCoins` — and for `MinCoins`, write
the state, transition, and base cases as a comment *before* the code. That
habit is the lesson.

## Further reading

- [Jeff Erickson — *Algorithms*, ch. 3: Dynamic Programming (free PDF)](https://jeffe.cs.illinois.edu/teaching/algorithms/book/03-dynprog.pdf)
- [Wikipedia — Dynamic programming](https://en.wikipedia.org/wiki/Dynamic_programming)
- [Wikipedia — Memoization](https://en.wikipedia.org/wiki/Memoization)
- [MIT OCW 6.006 — Dynamic Programming lectures](https://ocw.mit.edu/courses/6-006-introduction-to-algorithms-spring-2020/)
