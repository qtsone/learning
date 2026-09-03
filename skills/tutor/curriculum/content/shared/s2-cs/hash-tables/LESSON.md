# Hash Tables

> `shared.cs.hash-tables` · ~3-4h · Stage: CS Fundamentals

## Objectives

By the end of this lesson you can:

- Explain how a hash function maps keys to buckets and what properties make a
  hash function good for table use.
- Describe collision handling by separate chaining and open addressing, and
  compare their trade-offs.
- Explain load factor and why exceeding a threshold triggers resizing,
  including the cost of rehashing.
- Implement a basic hash table with put, get, and delete that handles
  collisions and passes the provided tests.
- Explain why hash table operations are O(1) on average but O(n) in the worst
  case, and what drives the degradation.

## The problem: looking things up by key

Arrays gave you one superpower: given an *index*, jump straight to the element
in O(1), because the address is just arithmetic (`base + index × size` — the
arrays lesson). But real programs rarely look things up by position. They look
things up by *key*: a username, a word, a product code.

With only the structures you have so far, key lookup means scanning — walk an
array or a linked list, compare every key, O(n). For a table of a million
users, that is a million comparisons per lookup. Too slow.

The hash table's idea is almost cheeky: **compute the index from the key
itself**. If some function could turn `"alice"` into `7`, you could store
Alice's data in slot 7 of an array and find it again in O(1) — no scanning.
That function is a *hash function*, the array slots are called *buckets*, and
the whole arrangement is a *hash table*.

You have already been using one for weeks: Go's built-in `map` is a hash
table. This lesson opens the lid.

## Hash functions: from key to bucket

A hash function takes a key (ultimately: its bytes) and produces a
fixed-size integer, the *hash*. The table then folds that big integer into a
bucket index:

```text
index = hash(key) mod number_of_buckets
```

For table use, a hash function must be:

1. **Deterministic** — the same key always produces the same hash. Otherwise
   you could store under one index and search under another, and the table
   would simply lose data.
2. **Uniform** — hashes spread evenly across the whole integer range, so keys
   spread evenly across buckets. Similar keys (`"user1"`, `"user2"`) should
   land in unrelated buckets.
3. **Fast** — it runs on *every* operation. A hash function that is slow makes
   the whole table slow, no matter how clever the rest is.

A classic that hits all three is **FNV-1a**. It is a few lines: start from a
large constant (the *offset basis*), then for each byte of the key, XOR the
byte in and multiply by a second constant (the *prime*):

```text
hash := offset_basis
for each byte b of key:
    hash := hash XOR b
    hash := hash × prime        (wrapping on overflow)
return hash
```

The XOR mixes each byte into the state; the multiply smears that change across
all the bits, so `"cat"` and `"cab"` end up with wildly different hashes. You
will implement exactly this in the exercise.

One clarification, because the word "hash" is overloaded: these are *not*
cryptographic hashes. SHA-256 exists to be irreversible and
collision-resistant against attackers, and pays for it in speed. Table hashes
only need to be fast and well-spread — different job, different tool.

## Collisions are inevitable

A hash function maps an *infinite* set of possible keys onto a *finite* set of
buckets, so two different keys will sometimes land in the same bucket. That is
not a bug you can fix — with more distinct keys than buckets it is a
mathematical certainty (the pigeonhole principle). A hash table is therefore
defined as much by its *collision strategy* as by its hash function. Two
families dominate.

### Separate chaining

Each bucket holds not one entry but a *linked list* of entries — you built
this structure two lessons ago. Insert: hash to a bucket, walk its chain; if
the key exists, update it, otherwise link a new node in. Lookup and delete:
hash to the bucket, walk the chain comparing keys.

Collisions just make some chains longer. As long as chains stay short (a
handful of entries), every operation is "hash + a few comparisons" — O(1) in
practice.

### Open addressing

Each bucket holds at most one entry, and a collision means: *probe* for
another slot in the same array. The simplest scheme, linear probing, tries
`index + 1`, `index + 2`, … (wrapping around) until it finds a free slot.
Lookup follows the same probe sequence until it finds the key or an
empty slot.

Deletion is the famous trap: if you just empty a slot, you punch a hole in
somebody else's probe sequence, and lookups that should succeed stop early at
the hole. Real implementations leave a *tombstone* — a "something was deleted
here, keep probing" marker.

### Trade-offs

| | Separate chaining | Open addressing |
|---|---|---|
| Extra memory | pointer per entry, nodes on the heap | none — one flat array |
| Cache behavior | chasing pointers, scattered in memory | scanning adjacent slots — cache-friendly (the locality argument from the arrays lesson) |
| Tolerates high load | degrades gently as chains grow | degrades sharply; probes cluster as the array fills |
| Deletion | unlink a node — easy | tombstones — subtle |
| Implementation | simpler | trickier to get right |

Chaining is the classic teaching implementation and what you will build.
High-performance tables (including Go's own `map`) tend toward open
addressing for the cache behavior, at the price of significant complexity.

## Load factor and resizing

Whatever the strategy, a table's health is measured by its **load factor**:

```text
load_factor = stored_entries / number_of_buckets
```

At load factor 0.5, buckets average half an entry each; chains are short,
probes are rare. As it climbs toward 1.0 (and past it, for chaining), chains
lengthen, and every operation slows. So practical tables set a threshold —
0.75 is a common choice for chaining — and when an insert would push the load
factor past it, they **resize**: allocate a bucket array twice as large and
re-insert every entry.

Why re-insert everything, rather than copy the old array? Because the bucket
index is `hash mod number_of_buckets` — change the bucket count and every
key's index changes with it. This full pass is called **rehashing** and costs
O(n). That sounds alarming for a structure advertising O(1), but doubling
means resizes become rarer as the table grows — most inserts are cheap, and
averaged over any long sequence the cost per insert is still O(1). This is the
same amortized-doubling argument you saw for Go's `append` growing a slice;
the Big-O lesson called it amortized analysis.

## O(1) on average — O(n) in the worst case

Now you can state precisely what "maps are fast" means:

- **Average case O(1):** with a uniform hash and the load factor held under
  the threshold, expected chain length is a small constant, so put, get, and
  delete each cost "one hash + a few comparisons" regardless of n.
- **Worst case O(n):** if every key lands in the *same* bucket, the table is a
  disguised linked list, and every operation scans it.

What drives an actual table toward the worst case:

1. **A bad hash function** — e.g. hashing only the first character: all keys
   starting with `"s"` pile into one chain.
2. **Never resizing** — even a perfect hash cannot help when a fixed 8-bucket
   table holds 10,000 entries: chains average 1,250 either way.
3. **Adversarial keys** — an attacker who knows your hash function can craft
   thousands of keys that all collide, turning your O(1) table into O(n) and
   your server into a space heater (a real attack, "hash flooding"). This is
   why serious runtimes, Go included, mix a random per-process seed into their
   hash so key-to-bucket mapping is unpredictable from outside.

In Go:

The `map` you have used since S1 is exactly this machinery, hidden behind
syntax. The comma-ok idiom is `Get`, the `delete` builtin is `Delete`, and map
keys must be comparable because the runtime must confirm "same key" after
hashing. Even iteration order being deliberately randomized is this lesson:
bucket order is an artifact of hashing, not something meaningful, and Go
randomizes it so programs cannot accidentally depend on it. The standard
library also ships FNV as [`hash/fnv`](https://pkg.go.dev/hash/fnv) — after
the exercise you can check your implementation against it.

## Exercise

Open [`exercise/`](exercise/). You will build a separate-chaining
hash table mapping `string` keys to `int` values — from scratch: **the
built-in `map` must not appear anywhere in `hashtable.go`**. The struct
layout, constants, and function signatures are given (the tests inspect the
`buckets` and `size` fields, so keep them); your work sites are marked `TODO`:

- `fnv1a` — the hash function, exactly the FNV-1a recipe above (the constants
  are provided; remember from strings-runes that indexing a string yields
  bytes, which is what you want here).
- `Put`, `Get`, `Delete` — the chain-walking logic.
- Resizing inside `Put` — double the buckets and rehash when the load factor
  would pass `maxLoadFactor`.

Acceptance criteria:

1. `fnv1a` matches the published FNV-1a test vectors (the tests carry them).
2. `Put` then `Get` round-trips values; `Get` on a missing key returns
   `(0, false)`; a stored value of `0` returns `(0, true)` — missing and
   zero are different things.
3. `Put` on an existing key replaces the value in place — `Len()` does not
   grow.
4. `Delete` removes the key (including from the *middle* of a chain without
   losing its neighbors), reports whether it was present, and decrements
   `Len()`.
5. After many inserts the table has resized: more buckets than it started
   with, load factor ≤ 0.75, and every key still retrievable afterwards —
   rehashing must not lose entries.
6. Chains stay short: after 1,000 inserts the longest chain is a few entries,
   not hundreds — this is your O(1) average made observable.
7. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They fail on the starter — that is the spec. Read the failure messages; they
tell you which behavior is missing. Work in the order listed: hash first (its
test is independent), then put/get, then delete, then resizing.

## Further reading

- [Go blog — Go maps in action](https://go.dev/blog/maps)
- [Go blog — Faster Go maps with Swiss Tables](https://go.dev/blog/swisstable)
  — how Go's own map applies these trade-offs
- [Open Data Structures — Hash Tables (ch. 5, Python edition)](https://opendatastructures.org/ods-python/5_Hash_Tables.html)
- [FNV hash — the canonical reference](http://www.isthe.com/chongo/tech/comp/fnv/)
