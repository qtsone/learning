# Trees & BSTs

> `shared.cs.trees` · ~3-4h · Stage: CS Fundamentals

## Objectives

By the end of this lesson you can:

- Define tree terminology (root, leaf, height, depth, subtree) and explain what
  makes a binary tree a BST.
- Implement BST insert and search that maintain the BST invariant and pass the
  provided tests.
- Implement in-order, pre-order, and post-order traversals and explain what
  ordering each produces.
- Explain why BST operations are O(log n) when balanced but degrade to O(n)
  when the tree becomes a chain.
- Choose between a BST and a hash table for a described lookup workload and
  justify the choice using ordering and complexity arguments.

## From chains to branches

Every structure you have built so far in this stage is *linear*: a linked list
is a chain of nodes where each node points to **one** next node. A **tree**
relaxes exactly one rule — a node may point to *several* next nodes, called its
**children** — with one restriction: every node has exactly one **parent**,
except a single starting node, the **root**, which has none. No cycles, no
sharing: from the root there is exactly one path to every node.

```
            (8)          ← root (depth 0)
           /   \
        (3)     (10)     ← depth 1
       /   \       \
    (1)     (6)     (14) ← depth 2
           /
        (4)              ← depth 3
```

The vocabulary, all of which the exercise tests will use:

- **Root** — the topmost node; the only entry point.
- **Leaf** — a node with no children (`1`, `4`, `14` above).
- **Edge** — a parent→child link.
- **Subtree** — any node plus everything below it is itself a tree. The node
  `3` is the root of the subtree `{3, 1, 6, 4}`. This self-similarity is why
  recursion (from earlier in this stage) fits trees so naturally.
- **Depth** of a node — number of edges from the root down to it.
- **Height** of a tree — number of edges on the *longest* root-to-leaf path.
  The tree above has height 3. A single node has height 0, and the empty tree
  is conventionally height −1 (one less than a single node — the recursion
  below makes this convention pay off).

Depth is measured from the top; height from the bottom. People mix these up
constantly — fix the direction in your head now.

A **binary** tree is a tree where every node has *at most two* children,
labeled **left** and **right**. The labels matter: a node with only a left
child is a different tree than the same node with only a right child.

## The BST invariant

A **binary search tree** (BST) is a binary tree with an ordering rule:

> For **every** node `n`: all keys in `n`'s *left subtree* are less than
> `n`'s key, and all keys in `n`'s *right subtree* are greater.

Read that carefully — it constrains the entire subtree, not just the immediate
children. This tree satisfies "left child < parent < right child" at every
node, yet it is **not** a BST:

```
        (8)
       /   \
    (3)     (10)
       \
        (9)      ← 9 > 8, but it sits in 8's LEFT subtree. Broken.
```

The invariant is what makes the tree *searchable*: standing at any node, one
comparison tells you which entire subtree the key must be in — the other
subtree can be discarded without looking at it.

(This lesson's trees hold unique keys: inserting a key that is already present
changes nothing. Real implementations either do that or store a count/value on
the node.)

## Search and insert: binary search, frozen into a structure

Remember binary search from the searching lesson: compare with the middle,
discard half, repeat. A BST is that same idea turned into a data structure —
each node *is* a stored "middle", and the left/right pointers lead to the two
halves. Searching is one short walk from the root:

```
search(node, key):
    if node is empty:        return false      # ran off the tree
    if key < node.key:       return search(node.left, key)
    if key > node.key:       return search(node.right, key)
    return true                                # found it
```

Insertion is the same walk — you search for the key, and when you fall off the
tree, that empty spot is exactly where the new node belongs:

```
insert(node, key):
    if node is empty:        return new-node(key)
    if key < node.key:       node.left  = insert(node.left, key)
    if key > node.key:       node.right = insert(node.right, key)
    return node                                # equal: already present, no change
```

Note the shape: `insert` *returns* the (possibly new) subtree root, and the
caller re-attaches it — `node.left = insert(node.left, key)`. That one idiom
handles the empty-tree case, the leaf case, and the "already exists" case
without special-casing any of them. Forgetting to re-attach the returned value
is the classic bug: the new node is created and immediately lost.

Every step of search or insert moves one level down, so the cost of both is
**O(h)** where `h` is the tree's height. Keep that phrase in mind — "cost is
the height" — because everything below hinges on what `h` turns out to be.

## Traversals: three ways to visit everything

Search visits one path. A **traversal** visits *every* node. For binary trees
there are three classic depth-first orders, and they differ only in *when the
node itself is visited* relative to its subtrees:

```
in-order(node):                 pre-order(node):              post-order(node):
    if node is empty: return        if node is empty: return      if node is empty: return
    in-order(node.left)             visit(node)                   post-order(node.left)
    visit(node)                     pre-order(node.left)          post-order(node.right)
    in-order(node.right)            pre-order(node.right)         visit(node)
```

On the BST from the first diagram (`8, 3, 10, 1, 6, 14, 4`):

- **In-order** (left, node, right): `1 3 4 6 8 10 14` — **sorted**. On a BST
  this is guaranteed: the invariant says everything left of a node is smaller
  and everything right is larger, so visiting left–node–right emits ascending
  order. In-order traversal is also your best correctness check: if the output
  isn't sorted, the invariant is broken somewhere.
- **Pre-order** (node, left, right): `8 3 1 6 4 10 14` — the root comes first.
  Feeding a pre-order sequence back into `insert`, one key at a time, rebuilds
  the exact same tree — which makes pre-order the natural order for copying or
  serializing a tree.
- **Post-order** (left, right, node): `1 4 6 3 14 10 8` — children before
  parents, root last. This is the order for "destroy" or "sum up" operations,
  where a node's answer depends on its children's answers being ready first.

All three visit each node exactly once: **O(n)** time, always.

## Balance, and the O(n) trap

Search and insert cost O(h). So what is `h`?

Best case, the tree is **balanced** — each level is roughly full before the
next begins. Level 0 holds 1 node, level 1 holds 2, level 2 holds 4… so `n`
nodes fit in about log₂ n levels, and `h ≈ log₂ n`. A balanced BST of a
million keys is about 20 levels deep: search touches ~20 nodes. This is the
same halving you measured in the binary search lesson.

Worst case: insert keys in **sorted order** into an empty BST — `1, 2, 3, 4,
…`. Every new key is larger than everything present, so every insert walks to
the rightmost node and hangs the new node below it:

```
(1)
   \
    (2)
       \
        (3)
           \
            (4)   ← a "tree" in name only: a linked list, h = n−1
```

Nothing broke — the BST invariant holds perfectly — but the shape degenerated
into a chain, `h = n−1`, and every operation is now O(n). Same structure, same
code, same invariant: the *insertion order* alone decides whether you get
O(log n) or O(n). Sorted (or nearly sorted) input is not rare in practice —
timestamps, auto-increment IDs, alphabetized imports — so the trap is real.

The fix is a **self-balancing** BST (AVL and red-black trees are the classic
ones): it does rotations on insert to keep `h` at O(log n) no matter the input
order. The rotation machinery is beyond this lesson, but the takeaway is not:
when a language's standard library offers an "ordered map" or "sorted set"
(C++ `std::map`, Java `TreeMap`), a self-balancing BST is what's inside. The
exercise makes the degradation measurable instead of theoretical: you will
implement `Height` and watch sorted input produce `h = n−1` while a
well-ordered insertion of the *same keys* produces `h ≈ log₂ n`.

## BST vs hash table

You built a hash table earlier in this stage; both it and the BST answer
"do I have this key?". Choosing between them is a real design decision:

| | Hash table | BST (balanced) |
|---|---|---|
| Search / insert / delete | O(1) average | O(log n) |
| Keys kept in order? | No — scattered by the hash | Yes — in-order gives sorted |
| "All keys between A and B"? | Must scan everything: O(n) | Walk the range: O(log n + matches) |
| Min / max key | O(n) scan | Walk far left / far right: O(h) |
| Worst case | O(n) (bad collisions) | O(n) (unbalanced) or O(log n) (self-balancing) |

The deciding question: **does the workload care about key ordering?**

- Only exact lookups — "is this session token valid?", counting word
  frequencies: hash table. O(1) beats O(log n), and ordering buys nothing.
- Range queries, sorted listings, nearest-key — "events between 09:00 and
  17:00", "first 20 names alphabetically after 'Smith'": BST. The hash table
  cannot answer these without visiting every key, because hashing deliberately
  destroys ordering.

In Go:

There is no tree in Go's standard library — `map` is the hash table, and when
ordering is needed Go programmers usually sort a slice. So in the exercise you
build the BST yourself, from the same raw material as your linked list —
a struct and pointers, with `nil` as the empty tree:

```go
type Node struct {
	Key   int
	Left  *Node
	Right *Node
}
```

## Exercise

Open [`exercise/`](exercise/) — a Go module with `bst.go` (six `TODO`
functions on the `Node` type above) and `bst_test.go`. **Read the tests
first**; they are the specification, including a test that measures the height
of a 1023-key tree under two insertion orders.

Acceptance criteria:

1. `Insert(root, key)` returns the root of a valid BST containing `key`;
   inserting an existing key changes nothing. Callers use it as
   `root = Insert(root, key)` starting from `nil`.
2. `Contains(root, key)` reports whether `key` is in the tree, using the BST
   invariant (one root-to-leaf walk — no full traversal).
3. `InOrder`, `PreOrder`, and `PostOrder` return the keys in their respective
   orders; in-order output is ascending for any BST.
4. `Height(root)` returns −1 for the empty tree, 0 for a single node, and the
   longest-path edge count in general — the chain-vs-balanced test must see
   height 1022 vs height 9 for the same 1023 keys.
5. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They must FAIL before you start — make them green, function by function
(`Insert` first; most other tests build trees with it).

## Further reading

- [Wikipedia — Binary search tree](https://en.wikipedia.org/wiki/Binary_search_tree)
- [Wikipedia — Tree traversal](https://en.wikipedia.org/wiki/Tree_traversal)
- [Open Data Structures — Binary trees (ch. 6, Python edition)](https://opendatastructures.org/ods-python/6_Binary_Trees.html)
- [Wikipedia — Self-balancing binary search tree](https://en.wikipedia.org/wiki/Self-balancing_binary_search_tree)
