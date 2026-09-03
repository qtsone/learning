# Tutor notes — Performance Engineering

## Where the learner is

This is the last lesson of the Go track. They have planned, built, hardened,
operated and contributed upstream; the capstone is a real codebase they have
lived in for weeks. They met the optimization loop in S5 against a program
someone else wrote with two planted bottlenecks. Here the code is theirs, most
of it is fast enough, and they arrive with beliefs about which part is slow.

Two failure modes dominate, and neither is a tooling gap:

- **Optimizing the wrong thing.** They pick the function they remember as
  difficult, or the one that is easy to benchmark, and produce a real 3% win on
  a path nobody waits for.
- **Claiming more than they measured.** A single run each side, no units, no
  input size, "about 10x faster". The number may even be right; the claim is
  still unverifiable, and that is what is being graded.

The one-optimization rule is load-bearing. If they bring five changes, make
them pick the one they can defend to the last decimal and treat the rest as
backlog. Depth here beats breadth every time.

Guidance mode: in `guided`, review the goal and the baseline *before* they
change any code — that is where the lesson is won or lost. In `spartan`, let
them bring the finished `PERF.md` and attack it.

## Common misconceptions

- **"The profile is the goal."** The profile is a hypothesis generator. The
  goal is the number they wrote down before opening pprof.
- **"ns/op is the result."** It is the least portable number they have. Ask for
  allocs/op and B/op alongside, and for the input size each figure belongs to.
- **"Faster on my machine means faster."** Thermal state, `-race`, background
  processes, a laptop on battery. If the delta is not larger than the spread
  between their own repeated runs, there is no delta.
- **"The tests still pass, so behaviour is unchanged."** Those tests were
  written against the code they deleted. Push for a differential test against
  the old implementation, or fuzzing old-vs-new.
- **"It's O(1) now."** Usually it is O(matches) instead of O(n), or a map
  lookup that is O(1) with a large constant. Make them say which variable the
  cost tracks now.
- **"Concurrency is an optimization."** It is a way to use more cores, and it
  buys races, scheduling overhead and new failure modes. Ask what the profile
  said before the goroutines appeared.
- **"There's no trade-off."** Every optimization costs something, if only the
  paragraph a reader now needs to understand the function. "None" means they
  have not looked.
- **"benchstat says `~` but the mean improved."** `~` means no detectable
  difference. This one needs saying out loud, usually twice.

## Grilling points

- "Read me your goal sentence. What is the operation, the input size, the
  number today and the number you wanted — and where did the target come from?"
- "Who waits for this operation, and how often? Convince me it is the one worth
  a week."
- "What did you expect the profile to say before you opened it? Were you
  right?" (The most informative question in the lesson.)
- "Show me the `-list` output. Which line, and what is its share of the total?"
- "Your CPU profile is flat and the wall clock is long. What does that mean and
  what tool answers it?"
- "Take the ratio between your two input sizes, before and after. Algorithmic
  or constant-factor? Say it in one sentence."
- "Under what conditions is your headline number true? What selectivity, what
  size, what hardware?"
- "What did you try that did not work, and how did you know?"
- "Which test would fail if this change altered behaviour? Delete the fast path,
  put the slow one back — does that test still pass?" (It should. The oracle
  test is the one that must not.)
- "What did this cost, and what would make you revert it?"
- "What is the next bottleneck, and how would you confirm it without changing
  anything?"

## Graduation review protocol

Run this once, at the end, 60-90 minutes. It is the last review of the
curriculum and it covers the whole capstone, not just this lesson.

1. **Demo.** They run the system and show the operation they optimized.
2. **The optimization, start to finish** (~25 min): goal, baseline, profile,
   hypothesis, change, comparison, trade-off, correctness. Interrupt for
   evidence at each step — "show me the file" beats "tell me the number".
3. **The system** (~30 min): the eight senior-engineer questions from the
   lesson. Ask about boundaries, numbers, failure modes, known weaknesses, a
   reversed decision, what they would cut, and the next bottleneck.
4. **One adversarial pass** (~15 min): take the other side. Argue the
   optimization was not worth it. Argue the trade-off is unacceptable at 10x
   the size. Argue a package boundary is wrong. What you are grading is whether
   they concede where the evidence is against them and hold where it is not.
5. **Close with the honest summary**: three things this capstone taught them
   that no exercise could have, and what they will build next.

Record the verdict and the three strongest and three weakest answers. This is
the artifact the learner keeps.

## Grading rubric

- **A** — Six harness checks green. The goal was written before the first
  measurement and names an operation someone waits for. The profile picked the
  target and they can say where their intuition was wrong. One change,
  re-measured identically, compared over repeated runs; the win is classified
  correctly and stated with its conditions. The trade-off is named and priced
  with a flip condition. A differential or fuzz test proves behaviour is
  unchanged and they can explain why the existing tests were not enough. In the
  graduation review they answer all eight system questions with numbers and
  concede at least one point to the adversarial pass on evidence.
- **B** — Harness green and the loop was followed, but one link is soft: the
  target came from the profile rather than from a stated goal, the comparison
  is two solid runs rather than ten, or the correctness argument leans on
  pre-existing tests plus one new case. They can defend the change; the
  conditions on the claim need drawing out. Graduation review solid, with one
  or two questions answered in adjectives.
- **C** — A real change with a real number, but the methodology is
  reconstructed after the fact: `PERF.md` written the evening before, evidence
  regenerated to match the story, no memory of what they expected. Or the
  optimization is correct and irrelevant — a measurable win on a cold path.
  Pass only with the weaknesses written down and stated aloud.
- **Fail** — No baseline, or numbers that cannot be reproduced. A claim
  contradicted by their own committed output. Behaviour changed and nobody
  noticed. Or the graduation review shows they cannot explain the system they
  built: boundaries they cannot justify, failure modes they have never
  considered, no weakness they will admit to. Do not graduate them on a green
  harness alone — the harness cannot read the argument, which is the whole
  reason a review exists.

## Remediation ladder

1. **Back to the goal.** "In one sentence: which operation, what size, what
   number now, what number do you want?" Most bad optimizations die here,
   before any code is written.
2. **Make them measure the operation, not the function.** If the benchmark
   targets a convenient internal, help them build the fixture for the real
   entry point. Nothing else in the lesson works until this is right.
3. **Read one profile together.** Take the `-top` output, find the share, then
   `-list` the one function. Say the flat/cum sentence aloud. Stop before
   suggesting a fix — let them form the hypothesis.
4. **Shrink the change.** If they have three changes tangled together, have
   them revert to the baseline commit and reapply exactly one, re-measuring
   after it. The lost afternoon buys a defensible result.
5. **Write the Correctness section first.** If they cannot name a test that
   would catch the change breaking behaviour, that test is the next thing they
   write — before any more tuning. Offer the oracle pattern by shape, not by
   code: "keep the old implementation in the test file; what would you compare?"
6. **Accept a null result.** A learner who measured honestly, found the change
   was not worth it, reverted, and wrote that up has done the lesson. Grade the
   methodology, not the speedup — and say so explicitly, because they will
   assume the opposite.

## After passing

This is the end of the Go track. Close it properly: they planned a system, built
it, hardened it, shipped it, contributed to somebody else's, and tuned their own
with evidence. Preview what is actually next — a project with no curriculum
attached, a language whose runtime works differently, or an open-source issue
harder than the last one — and tell them the loop they just ran is the transfer:
state the question, measure, change one thing, prove it, write down the price.
