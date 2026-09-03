# Tutor notes — Stacks & Queues

## Where the learner is

Third lesson of CS Fundamentals. They can analyze loops with Big-O and built
a singly linked list last lesson, so pointers-as-links are fresh. They write
beginner Go comfortably (structs, pointer receivers, comma-ok, table-driven
tests) but have never used generics or interfaces — the concrete `rune`/`int`
element types in the exercise are deliberate; don't let them rabbit-hole on
"how do I make this generic". Recursion has not happened yet: the call stack
may be mentioned as motivation but not exercised.

## Common misconceptions

- **"A stack/queue is a new data structure"** — it's a *discipline* over an
  array or linked list. If they think they must learn a third memory layout,
  reground them: same layouts as last lesson, fewer allowed operations.
- **`q.items = q.items[1:]` is a fine dequeue** — it is O(1) in time, and
  that's exactly why it's seductive. The backing array stays reachable, so a
  long-lived queue pins every element it ever held. Distinguish the two
  failure modes: shifting (slow) vs. reslicing (memory).
- **Forgetting to reset `tail` on drain** — the queue "works" until it's
  emptied once, then `Enqueue` appends to a detached node.
  `TestQueueDrainThenReuse` exists precisely for this; if it fails, point
  them at the dequeue-last-element path, not at `Enqueue`.
- **Peek that pops** — implementing `Peek` as pop-then-push-back, or just
  calling `Pop`. The tests call `Peek` twice and check `Len`.
- **"Amortized O(1) means every push is fast"** — no: *most* pushes are fast
  and the rare doubling copy is paid for by the cheap ones. If they can't
  say this, replay the doubling sequence 1, 2, 4, 8 … total copies < 2n.
- **Counter-based bracket checking** — a single open-minus-close counter
  passes `"()"` and `"((("` style cases but not `"([)]"`. If their
  `Balanced` has no stack in it, run that case with them by hand.

## Grilling points

- "Undo history, print jobs, browser back button, network packet buffer —
  which discipline for each, and what breaks if you swap it?"
- "Why does `([)]` defeat a counter but not a stack? What does the stack
  remember that the counter forgets?"
- "Your `Push` calls `append`. Walk me through the append that triggers a
  grow. Why do we still call `Push` O(1)?"
- "Why does `Queue` carry a `tail` pointer? What would `Enqueue` cost
  without it?" (O(n) walk from head — last lesson's insert-at-tail result.)
- "You dequeue with `items = items[1:]` instead of linked nodes. What
  happens to memory on a queue that runs for a week?"
- "The linked queue never needs a growth copy, yet real libraries often use
  array-backed ring buffers anyway. Why might they?" (Allocation per node,
  cache locality — ties back to Arrays & Linked Lists.)

## Grading rubric

- **A** — All tests pass; `Balanced` uses their `Stack` (not a raw slice or
  counter); `Dequeue` resets `tail` and they can say why unprompted; `Len`
  is O(1) via the counter; they can explain amortized growth and both
  slice-front pitfalls in their own words.
- **B** — Tests pass but with rough edges: `Balanced` re-implements a stack
  inline, or the `tail` reset was found only via the test, or the amortized
  explanation is shaky until nudged.
- **C** — Tests pass only after heavy hinting, or they cannot articulate
  LIFO vs FIFO for a concrete scenario without prompting. Time-boxed
  remediation before advancing.
- **Fail** — Tests failing, or the queue quietly wraps a slice with
  front-reslicing despite the lesson, or they cannot trace `([)]` through
  their own `Balanced`. Remediate, don't advance.

## Remediation ladder

1. "Read the failing test name and message aloud. Which operation, and what
   did it expect?"
2. Stack stuck: "Where is the top of a slice-backed stack? Which slice
   expression gives you everything *except* the last element?"
3. Queue stuck: "Draw three boxes and two arrows for head and tail. Now
   dequeue twice on paper — what must each pointer be after the last box
   leaves?"
4. `Balanced` stuck: walk `"([)]"` together character by character, saying
   push/pop out loud — then let them turn the walk into code.

## After passing

Preview: "Next up, hash tables — the structure behind Go's `map`. You'll
build one and find out what a 'collision' costs."
