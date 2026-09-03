# Tutor notes — Trees & BSTs

## Where the learner is

Eighth lesson of the CS stage. They have implemented linked lists, stacks,
queues, a hash table, binary search, and recursive algorithms — so pointers,
`nil`-as-empty, and base-case thinking are established, and this lesson leans
on all three. Trees are their first non-linear structure; expect the "return
the subtree root and re-attach it" idiom in `Insert` to be the main new mental
move. Heaps and graphs come later — don't reach forward.

## Common misconceptions

- **BST invariant read as parent/child only** — "left child < parent < right
  child" instead of *entire subtrees*. Show the broken tree from LESSON.md
  (9 under 8's left subtree via 3's right child) and ask them to search for 9.
- **Height vs depth swapped** — depth is measured from the root down to a
  node; height from a node down to the deepest leaf. Also the conventions:
  empty tree −1, single node 0.
- **Forgetting to re-attach in `Insert`** — calling `Insert(root.Left, key)`
  and dropping the return value, or never handling `root == nil`. Symptom:
  every test that uses `build` fails with an empty tree.
- **"BSTs are O(log n)"** stated unconditionally — the bound is O(h), and h is
  O(log n) only when balanced. The 1023-key test exists to break this belief.
- **In-order confused with insertion order** — expecting `InOrder` to replay
  the sequence of inserts rather than sorted order.
- **Traversal slice bugs** — appending the node key at the wrong position
  (pre/post swapped), or in Go, aliasing surprises when appending recursive
  results; `slices.Equal` in the tests catches order, not aliasing, so the
  simple copy-append shape in the solution is fine.

## Grilling points

- "Point at any node in your tree and tell me its depth and its height. Now
  the tree's height." (Terminology, both directions.)
- "Walk me through `Insert(root, 6)` on the 5/3/8/1/4/7/9 tree, call by call.
  Where does the new node attach, and who re-attaches whom?"
- "Why does `Contains` never need to look at both subtrees? Which lesson from
  this stage is that the same trick as?" (Binary search: discard half.)
- "Your in-order traversal comes out unsorted. What does that tell you —
  a traversal bug, an insert bug, or can't you tell?" (Either; but if
  pre/post are consistent with in-order's tree shape, suspect insert.)
- "The chain test inserts 0..1022 in order and gets height 1022. What input
  arriving sorted could realistically happen in production, and what would you
  use instead of your BST there?" (Timestamps/IDs; a self-balancing tree —
  they don't need to know rotations, just that the fix exists.)
- "A service needs 'all user IDs between X and Y' *and* 'does ID exist?'.
  Hash table, BST, or both? Justify with costs." (Objective 5 head-on.)

## Grading rubric

- **A** — All tests pass; `Insert` uses the return-and-reattach shape (or a
  correct iterative equivalent); `Contains` walks one path, no full traversal;
  learner explains the invariant as a subtree property, derives why in-order
  is sorted, and argues chain-vs-balanced heights with the O(h) framing;
  gofmt-clean.
- **B** — Tests pass but with rough edges: a traversal built by collecting
  into a package-level slice, an O(n) `Contains` via `InOrder`+scan they can't
  defend, or hand-wavy balance explanation ("it gets slow") without h = n−1
  vs h ≈ log n.
- **C** — Tests pass only after heavy hints, or the learner cannot predict
  what pre/post-order print for a small tree without running the code. Pass
  only if remediation lands; otherwise iterate.
- **Fail** — Tests failing, or `Insert` copied without being able to explain
  why `root.Left = Insert(root.Left, key)` re-attaches. Remediate.

## Remediation ladder

1. "Draw the tree after `build(5, 3, 8)` on paper — boxes and arrows. Now do
   what your code does, line by line, for `Insert` of the 3. Where does the
   returned node go?"
2. "What are the two cases every recursive tree function needs? What is the
   base case for an empty tree in `Insert`? What must it *return*?"
3. "For traversals: the three functions are the same three lines in different
   orders. Write in-order first, get `TestInOrderIsSorted` green, then derive
   the other two by moving one line."
4. "For `Height`: in words — the height of a node is one more than the taller
   of its children's heights. What must the empty tree return so that a single
   node computes to 0?" (Walk them to −1, let them type it.)

## After passing

Preview: "Next: heaps — a tree that keeps only a *weaker* promise than the
BST's, and buys back guaranteed O(log n) plus instant access to the minimum."
