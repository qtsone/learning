# Tutor notes — Hash Tables

## Where the learner is

Fourth lesson of S2. They can analyze simple loops with Big-O, have
implemented a linked list (chaining should feel like a reuse of that lesson,
not a new idea), and built stacks/queues. From S1 they use Go's `map` daily —
comma-ok, `delete`, iteration — without knowing what is inside. The payoff
moment of this lesson is "the map I've been using is *this*"; steer toward it.
Recursion, trees, and later structures have NOT happened yet — keep examples
to arrays, lists, and this table.

## Common misconceptions

- **"A good hash function avoids collisions."** It cannot — pigeonhole. It
  only spreads them evenly. If they say "collision-free", probe with: "you
  have infinite possible keys and 8 buckets; now what?"
- **"Hash = cryptography."** Conflating FNV with SHA-256. Table hashes are
  fast and well-spread, not secure; using SHA-256 in a hash table is just
  slow, not wrong-answers wrong.
- **"index = hash(key)"** — forgetting the `mod buckets` fold, then being
  confused why resizing forces rehashing. If resizing confuses them, this is
  almost always the missing link.
- **"Resizing is copying the bucket array."** No — every index is
  `hash mod bucket_count`, so every entry must be re-inserted. The
  `TestResizeKeepsEveryKey` failure mode ("key present before resize, gone
  after") is this misconception in code form.
- **"O(1) always."** They quote average-case as a law. Push: what input makes
  it O(n)? (Everything in one chain — bad hash, no resize, or crafted keys.)
- **Put appends without checking for the key** — duplicates in the chain;
  `Len` drifts and `TestPutReplacesExistingKey` catches it.
- **Delete drops the whole tail** (`bucket = e.next` regardless of position)
  — `TestDeleteFromMiddleOfChain` exists precisely for this; it is the same
  unlink bug from the linked-list lesson.
- **FNV order swapped** (multiply before XOR is FNV-1, not FNV-1a) or ranging
  over the string (runes) instead of indexing (bytes) — both produce wrong
  vectors in `TestFNV1a` for multi-byte input.

## Grilling points

- "Walk me through `Get("alice")` end to end: bytes, hash, index, chain."
- "Why must a hash function be deterministic? What actually breaks if it
  isn't?" (Data becomes unfindable — stored under one index, sought at
  another.)
- "Your table has 8 buckets and 10,000 entries with a *perfect* hash. What is
  a lookup's cost, and why?" (≈1,250 per chain — load factor beats hash
  quality; resizing is not optional.)
- "Why does deleting from an open-addressing table need tombstones while your
  chaining table doesn't?" (Probe sequences must not be broken by holes.)
- "Resizing is O(n). Defend calling Put O(1) anyway." (Amortized doubling —
  same argument as slice `append`.)
- "Why does Go randomize map iteration order — what would people do if it
  didn't?" (Depend on an accident of hashing; and per-process seeding also
  defends against hash flooding.)

## Grading rubric

- **A** — All tests pass; no built-in `map` anywhere in the implementation;
  update-in-place and resize both correct; can explain the mod-fold, why
  rehashing re-inserts, and the average-vs-worst-case story unprompted;
  gofmt-clean.
- **B** — Tests pass but with rough edges: resize threshold checked after
  insert, clumsy-but-correct delete, or explanation wobbles on *why*
  rehashing is needed (parrots "you re-insert" without the mod argument).
- **C** — Tests pass only after heavy hinting, or they smuggled a `map` into
  the implementation, or cannot trace a lookup end to end. Time-boxed
  remediation before advancing.
- **Fail** — Tests failing, or the collision/resize code is copied and they
  cannot explain what a chain is or when resizing fires. Remediate from the
  linked-list lesson if the unlink logic is the blocker.

## Remediation ladder

1. "Run the tests and read the first failure aloud. Which acceptance
   criterion is it, and which function owns it?"
2. Hash wrong: "The recipe is XOR the byte, *then* multiply — check your
   order. And does `key[i]` give you a byte or a rune?" Put/Get wrong:
   "Two steps: which bucket, then which entry in the chain. Which step is
   your code missing?"
3. Delete wrong: "Draw a 3-node chain and cross out the middle one. What must
   the first node's `next` point to now? That is the pointer your code has to
   rewrite — you did this exact move in the linked-list lesson."
4. Resize wrong: "A key hashed to bucket 5 of 8. You now have 16 buckets —
   which bucket does the *same* hash put it in? So what must happen to every
   entry when the count changes?" Then let them write the re-insert loop.

## After passing

Preview: "Next is recursion — functions that call themselves. It is the
thinking tool behind the trees and graph algorithms coming later in this
stage, and the call stack behaves exactly like the stack you built."
