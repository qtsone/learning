# Tutor notes — Arrays & Linked Lists

## Where the learner is

Fresh off S1 Go (structs, pointers, methods, table-driven tests) and the
Big-O lesson — they can name growth classes but have never *built* a data
structure. This is their first time using pointers to link records together,
which is a bigger conceptual jump than it looks: in S1 pointers pointed *at*
things; here pointers *are* the structure. Recursion hasn't been taught yet —
expect and accept iterative solutions only. Budget most of the session for
the Delete edge cases; they're where the learning happens.

## Common misconceptions

- **"Linked lists insert in O(1), so they're faster"** — dropped caveat: O(1)
  only *given a node reference*. Finding the position is O(n), so
  search-then-insert is O(n) total. Press until they say the caveat unprompted.
- **Stale tail after deleting the last node** — the classic bug, and
  `TestAppendAfterDeletingTail` exists to catch it: Append then attaches to
  an unlinked node and the value silently vanishes. If they hit it, make them
  draw the pointers before touching code.
- **Prepend on an empty list forgets the tail** — mirror-image bug, caught by
  `TestPrependOnEmptyThenAppend`.
- **"Slice append is O(n) because it copies"** — conflates the rare growth
  copy with the common in-place write. Revisit doubling and amortized O(1).
- **"Same Big-O, same speed"** — an O(n) list scan and O(n) slice scan differ
  ~10-100× in practice because of cache misses on pointer chases. Big-O
  compares growth, not constants.
- **Trying to delete with only `cur`** — they discover a singly linked node
  can't unlink itself. That frustration is the lesson: let them feel it, then
  name the prev/cur two-pointer walk (and mention doubly linked lists as the
  structural fix, without detouring into building one).
- **Length drift** — forgetting `length` updates in one branch of Delete.
  `TestDelete` checks Len in every case, so the failure localizes it.

## Grilling points

- "Draw memory for `[10, 20, 30]` as an array and as a linked list. Where
  does each value live?"
- "Why does `xs[3]` and `xs[3000000]` cost the same on a slice? Walk me
  through the arithmetic."
- "Your Delete carries a `prev` reference. Why can't the current node alone
  do the job? What would a doubly linked list change?"
- "Both structures scan in O(n). Why does the slice usually win by a wide
  margin anyway?" (Cache lines, prefetching, pointer chasing.)
- "150,000 appends: what's the total cost with your tail pointer, and what
  would it be without one?" (O(n) vs O(n²) — tie back to the Big-O lesson.)
- "A workload does thousands of inserts at the front and one full read at the
  end. Slice or list? Now flip it: one build, millions of random index reads."
- "Why does Go's standard library ship `container/list` yet almost nobody
  uses it?"

## Grading rubric

- **A** — All tests pass; Append/Prepend/Len are O(1) with correct
  empty-list handling; Delete is a single prev/cur walk that handles all four
  edge cases without special-cased duplication; gofmt-clean; learner can draw
  the pointer moves for "delete the tail" unprompted and states the
  node-reference caveat when praising list insertion.
- **B** — Tests pass but with avoidable clutter (e.g. separate code paths for
  head/middle/tail deletes, redundant nil checks) or one edge case fixed only
  after a test pointed at it; explanations solid on arrays vs lists, shakier
  on why tail bookkeeping exists.
- **C** — Tests pass only after the remediation ladder reached hint 3+, or
  the learner can't explain why Delete needs `prev` / why the tail moved.
  Re-derive the pointer diagrams together before advancing.
- **Fail** — Tests failing, an O(n) Append that walks the list (fast-test
  failure ignored), or a pasted solution they can't walk through node by
  node. Remediate and re-attempt.

## Remediation ladder

1. "Run the failing test alone (`go test -run TestDelete/tail`). Read the
   got/want values aloud — which node is still reachable that shouldn't be?"
2. "Draw three boxes with arrows for `1 → 2 → 3`, plus head and tail arrows.
   Now perform your code's Delete(3) on the drawing, one pointer write at a
   time. Where does `tail` point when you're done?"
3. "In your walk you have `cur` on the match. What extra reference do you
   need to rewire around it, and how do you keep it during the walk?"
4. Talk through the solution shape — prev starts empty, walk both together;
   on match: empty prev means new head, else `prev.next = cur.next`; if cur
   is tail, tail becomes prev — but they type every line themselves.

## After passing

Preview: "Next you build stacks and queues — thin discipline layers (last-in
first-out, first-in first-out) over exactly the slice and list machinery you
just wrote."
