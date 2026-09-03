# Tutor notes — Dynamic Programming Intro

## Where the learner is

Eleventh lesson of the CS stage — only the problem-patterns capstone remains.
Recursion (base cases, call-stack model), hash tables, Big-O growth classes,
and divide-and-conquer via merge sort are all in hand; DP is where they
combine. This is the most conceptually loaded lesson of the stage: the coding
is short, the *thinking* (state/transition/base cases) is the deliverable.
Expect the greedy trap and the "check before, store after" cache discipline to
be the two friction points. Do not reach into 2D DP, LCS, or knapsack — the
exercise is deliberately 1D.

## Common misconceptions

- **"DP means filling 2D grids"** — DP is one idea: never solve the same
  subproblem twice. The cache/table is bookkeeping; the grid shape is
  incidental. If they parrot "make a table", push for the *why*.
- **Memoization missing one of its two moves** — checking the cache but never
  storing, or storing but never checking. Both still return correct answers,
  just exponentially slowly; `TestFibMemoComputesEachSubproblemOnce` and the
  guard exist precisely for this. Ask which line is the check and which the
  store.
- **"Any recursion gets faster with a cache"** — merge sort's subproblems are
  disjoint; caching them buys nothing. Overlap is a property of the problem,
  not of recursion. This distinction is objective 1 — grill it.
- **"Memoized fib is O(1)"** — it is O(n): n+1 subproblems at O(1) each. The
  drop is exponential→linear, not exponential→constant.
- **Greedy assumed correct for coins** — US denominations hide the flaw; the
  `{1,3,4}`/6 case exposes it. If they ask "why not just take the biggest
  coin?", have them run greedy on that case by hand.
- **Sentinel overflow in `MinCoins`** — `math.MaxInt` as "impossible" wraps
  negative on `+1` and then wins every min, producing garbage answers on
  unreachable amounts. The `amount+1` sentinel is the taught fix; the
  `{2}`, amount-7 case flushes this out (the large-amount test never reads
  the sentinel — its coin set includes a 1, so every cell is reachable).
- **Tabulation fill-order taken on faith** — they copy the `for i := 2` loop
  without seeing that it exists to satisfy dependencies. Ask what breaks if
  the loop runs backwards.
- **Call-count confusion in `FibNaive`** — forgetting the `+1` for the
  current call or not counting base cases. The doc comment pins the
  convention; exact-count tests catch it immediately.

## Grilling points

- "Merge sort recurses; naive fib recurses. Why does a cache transform one
  and do nothing for the other?" (Overlapping subproblems present vs absent.)
- "Your `FibMemo(90)` computed 91 subproblems. Where did the other
  ~10¹⁸ calls go? Walk me through one cache hit." (Objective 2 head-on.)
- "State, transition, base cases — give me all three for climbing a staircase
  where you may take 1, 2, or 3 steps and I ask in how many ways you reach
  step n." (New problem, same class: state = step, transition = sum of the
  previous three, bases 0/1/2. Objective 5.)
- "Why could `FibConstSpace` drop the table to two variables, but `MinCoins`
  must keep the whole `best` slice?" (Transition lookback: bounded k=2 vs
  reaching back by arbitrary coin values.)
- "You need fib of five million. Which of your four fib functions do you call,
  and what kills each of the others?" (Naive: time; memo: recursion depth /
  stack; tab: fine but O(n) space; const-space: the answer. Overflow caveat
  earns bonus credit.)
- "In `MinCoins`, why is `amount+1` a safe 'impossible' marker? What exactly
  goes wrong with max-int?" (No real answer exceeds `amount` coins; max-int
  wraps negative on +1.)
- "Point at the line in your `MinCoins` that encodes optimal substructure."
  (`best[a-c] + 1` — reusing an optimal sub-answer.)

## Grading rubric

- **A** — All tests pass. `FibMemo` threads the cache through a helper (map +
  pointer counter or equivalent — no package-level state); `FibConstSpace`
  genuinely holds two values (no slice/map/recursion); `MinCoins` has the
  state/transition/base-cases comment and a wrap-safe sentinel. Learner
  explains the exponential→linear drop with the call-tree picture, states
  when they would pick tabulation over memoization, and can produce
  state/transition/base for an unseen staircase-style problem. gofmt-clean.
- **B** — Tests pass with rough edges: `FibConstSpace` quietly allocates or
  delegates to `FibTab`, sentinel is max-int with a hand-waved "it worked",
  package-level memo map shared across calls, or the two DP signals are
  recited but the merge-sort contrast falls apart under questioning.
- **C** — Tests pass only after heavy hints, or the learner cannot define the
  state for a new problem without being walked through it — they memorized
  the fib code rather than the recipe. Pass only if time-boxed remediation
  lands; otherwise iterate.
- **Fail** — Tests failing, or a working solution the learner cannot explain:
  can't point at the cache check vs the cache store, or can't say why the
  memoized version is fast. Remediate, don't advance.

## Remediation ladder

1. "Draw the call tree for `fib(5)` on paper — every call, arrows down.
   Circle every node that appears more than once. How many *distinct* values
   of n are there in the whole tree?"
2. "A cache needs exactly two moves: look before you compute, save after you
   compute. Read your `FibMemo` line by line — which of the two is missing or
   in the wrong place?"
3. "For `FibTab`: which cells does `table[i]` read? So which cells must be
   full before you reach i? What loop direction guarantees that?"
4. "For `MinCoins`, forget code. Coins {1,3,4}, amount 6. Fill cells 0
   through 6 by hand, saying each choice out loud: 'best way to make 5 is…'.
   Now find the line of the table walk in LESSON.md and check yourself." Then
   let them translate the walk into the loop — don't dictate it.

## After passing

Preview: "Next is the stage capstone — a mixed problem set with no labels
telling you which tool fits. Pattern recognition is the skill: some problems
will want the structures you built, some the algorithms, and at least one
will want today's recipe."
