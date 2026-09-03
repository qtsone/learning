# Tutor notes — Recursion

## Where the learner is

Fifth lesson of S2. They can classify growth with Big-O and have implemented a
linked list, a stack, a queue, and a basic hash table — all in Go at S1 level.
Recursion as a deliberate technique is brand new, and this is the first time
the call stack becomes visible machinery rather than background noise. The
stacks-and-queues lesson is the anchor: keep pointing back at "the stack you
built is the stack the runtime uses". Tracing on paper (objective five) is a
skill they must demonstrate, not just nod along to.

## Common misconceptions

- **"Recursion is magic"** — it's ordinary function calls tracked by an
  ordinary stack. Deflate the mystique by drawing frames.
- **Frames sharing variables** — believing all levels see one `n`. Have them
  label each frame's own `n` in a trace; this one error explains most broken
  recursive code.
- **Trying to mentally unroll every level** — instead of trusting the
  recursive call on smaller input. Teach the leap of faith plus the
  three-point checklist (base case exists, reachable, progress every call).
- **Base case as decoration** — "works on my input" hides an unreachable base
  case. `Factorial(-1)` with the solution's code recurses forever; use it.
- **Expecting Go to rescue deep recursion** — Go has no tail-call
  optimization, and a blown stack is a fatal error, not a recoverable panic.
- **Reversing bytes instead of runes** — `s[1:]` on a string slices bytes; the
  "héllo"/"日本語" test rows exist precisely to catch this (S1 strings-runes).
- **`SumIterative` that secretly recurses** — calling `Sum` or recursing
  anyway. The deep-tree test lowers the stack cap, so the *entire test binary*
  dies with `fatal error: stack overflow` and no tidy failure list. That
  alarming output is the designed feedback; help them read the crash message
  and connect it to the lesson.

## Grilling points

Beyond quiz.json:

- "Draw the stack for `Factorial(3)` at its deepest point. Now unwind it,
  return value by return value." (Must distinguish winding from unwinding.)
- "What happens with `Factorial(-1)` as you wrote it? Where exactly does the
  checklist break?" (Base case exists but is unreachable from −1.)
- "Your `reverseRunes` swaps the ends and recurses inward — why is the base
  case `len < 2` and not `len == 0`?"
- "What are the time and space complexities of your `Sum`? What counts as
  space here?" (The call stack counts: O(depth) frames even with no
  allocations.)
- "`SumIterative` visits nodes in a different order than `Sum` — does the
  total care? Can you think of a task where order *would* matter?" (Seeds the
  trees lesson's traversals; do not teach them now.)
- "Why is a slice-as-stack immune to the limit that kills recursion?" (Heap
  allocation, grows by `append`, bounded by RAM.)

## Grading rubric

- **A** — All tests pass, including the deep-chain test. `Factorial`,
  `Reverse`, and `Sum` are genuinely recursive with clean, correct base
  cases; `SumIterative` uses an explicit stack with no recursion. Learner
  traces `Factorial(4)` or `Reverse("abc")` fluently on paper and articulates
  when they would pick iteration over recursion.
- **B** — Tests pass, but the hand trace is shaky (winding/unwinding
  confused), or `SumIterative` works but they can't explain why it survives
  the deep test; minor gofmt/style issues.
- **C** — Tests pass only after heavy hints, or rune handling in `Reverse` is
  accidental, or base-case reasoning collapses on a fresh problem. Time-boxed
  remediation before advancing.
- **Fail** — `Factorial`/`Reverse`/`Sum` solved with loops (dodges the
  lesson), `SumIterative` recursing (deep test crashing the run), or the
  learner cannot identify base and recursive cases in a problem they haven't
  seen. Remediate, don't advance.

The tests cannot detect that the recursive functions actually recurse — only
you can. Read the code before grading.

## Remediation ladder

1. "Say the problem in this shape: 'the answer for THIS input is <one step>
   combined with the answer for a SMALLER input'. What's the smallest input?"
2. "Write only the base case and return early. Run the tests — which cases
   pass now, and what does that tell you?"
3. "For the recursive case: assume the function already works for anything
   smaller — you're allowed to just call it. What single step remains for
   you to write?"
4. Trace the failing input together on paper, frame by frame, until they spot
   the frame where their code's answer diverges from the hand trace. Let them
   type the fix.

## After passing

Preview: "Next: sorting — merge sort and quicksort are this lesson's
divide-and-conquer recursion turned into the workhorses behind every standard
library's sort."
