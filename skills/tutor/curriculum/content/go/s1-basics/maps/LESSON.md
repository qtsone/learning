# Maps

> `go.basics.maps` · ~2-3h · Stage: Programming Basics (Go)

## Objectives

By the end of this lesson you can:

- Create, read, update, and delete map entries, including initialization with `make` and literals.
- Use the comma-ok idiom to distinguish a missing key from a zero value.
- Explain why map iteration order is randomized and how to iterate keys in a stable order.
- Implement a set using `map[T]struct{}` and justify the empty-struct value type.

## From positions to names

A slice answers "what is at position 3?". A map answers "what is stored under
the name `"flour"`?". A map associates **keys** with **values**, and its type
spells out both: `map[string]int` maps strings to ints.

Any type you can compare with `==` can be a key — strings, numbers, booleans.
Slices cannot (there is no `==` for slices, as you saw last lesson), so you
cannot key a map by a slice. Values can be anything.

## Creating a map

Two ways, mirroring what you know from slices:

```go
scores := make(map[string]int)                    // empty, ready to use
pantry := map[string]int{"flour": 2, "eggs": 12}  // literal, contents known up front
```

There is a trap in the third way. A `var`-declared map is **nil**:

```go
var m map[string]int   // nil — no storage allocated
n := m["flour"]        // reading is fine: n is 0
m["flour"] = 1         // PANIC: assignment to entry in nil map
```

Reading a nil map behaves like reading an empty one, but *writing* crashes the
program at runtime — the compiler cannot catch it. Rule of thumb: if you will
ever write to a map, create it with `make` or a literal first.

## Read, update, delete

```go
pantry["flour"] = 3       // create or update — same syntax
n := pantry["eggs"]       // read: 12
pantry["eggs"]++          // read, add one, store back
delete(pantry, "flour")   // remove the entry entirely
len(pantry)               // number of entries
```

Reading a key that is not there does **not** error — it returns the zero value
of the value type (`0` for `int`, `""` for `string`). That makes counting
delightfully short: `counts[word]++` works even the first time you see a word,
because the missing read yields `0` and you store `1`.

Note that `delete` and setting a value to `0` are different: `delete` removes
the key from the map, while `pantry["eggs"] = 0` keeps an entry whose value
happens to be zero. The next section is about telling those two apart.

## Missing, or just zero? The comma-ok idiom

If `pantry["milk"]` returns `0`, is the pantry out of milk, or has it never
stocked milk at all? A plain read cannot tell you. The indexed read has a
second form that can:

```go
n, ok := pantry["milk"]
```

`ok` is `true` if the key exists and `false` if it does not — regardless of the
value. This is the **comma-ok idiom**, and you will use it constantly, often
folded into an `if` the way you learned in control flow:

```go
if n, ok := pantry["milk"]; ok {
	fmt.Println("we have", n)
}
```

## Maps are shared, not copied

When you pass a map to a function, the function sees — and can change — the
caller's data:

```go
func restock(p map[string]int) { p["flour"] += 10 }

restock(pantry)   // pantry itself now has 10 more flour
```

A map value is a small handle pointing at shared storage; passing it copies the
handle, not the entries. So functions can update a map without returning it.
(The machinery behind this arrives in the pointers lesson — for now, remember
the behavior.) One consequence: maps cannot be compared with `==` (except to
`nil`); the standard library's `maps` package provides `maps.Equal`, which the
exercise tests use.

## Iteration order is deliberately random

`range` works on maps like it does on slices, yielding key and value:

```go
for item, n := range pantry {
	fmt.Println(item, n)
}
```

Run that twice and the lines can come out in *different orders*. That is not a
bug: a map organizes entries for fast lookup, not in any meaningful sequence,
and the Go runtime deliberately randomizes iteration order so that no program
accidentally comes to depend on it. A hidden order-dependency would break
mysteriously later; randomization makes it break immediately, while you are
watching.

When you need a stable order, make one: collect the keys into a slice, sort
it, then range over the slice. The standard library's `slices` package
(imported like any other) does the sorting:

```go
items := make([]string, 0, len(pantry))
for item := range pantry {          // key-only form of range
	items = append(items, item)
}
slices.Sort(items)                  // ascending order, in place
for _, item := range items {
	fmt.Println(item, pantry[item])
}
```

## Maps as sets

Sometimes you only care *whether* something is present — not any value
attached to it. The idiomatic Go set is a map whose value type is `struct{}`,
the **empty struct**: a type with no fields that occupies zero bytes. (Structs
with actual fields are the whole next lesson; here we only use the empty one.)

```go
seen := make(map[string]struct{})
seen["flour"] = struct{}{}          // struct{}{} is the type's only value
_, ok := seen["flour"]              // ok == true: it's a member
```

Why not `map[string]bool`? It works, but it can store `false` — a member that
claims not to be one — and every reader must wonder whether that third state
is meaningful. `struct{}` says exactly what you mean: the value carries no
information; membership of the key is the whole point.

## Exercise

Open [`exercise/`](exercise/) — a Go module with package `pantry` in two
files (no `main.go`: this is a library package, tested directly, like the
ones you built in the packages lesson):

- `pantry.go` — six small functions with `TODO`s for you.
- `pantry_test.go` — the specification. **Read it first.**

Acceptance criteria:

1. `Count(words)` returns a map from each word to how many times it appears.
2. `Describe(pantry, item)` returns `"not stocked"` when the pantry has no
   entry for the item, `"out of stock"` when the entry exists with count 0,
   and `"3 in stock"` (with the real count) otherwise.
3. `Take(pantry, item, n)` subtracts `n` when there is enough stock and
   returns `true`; taking the last unit deletes the entry entirely.
4. `Take` returns `false` and leaves the pantry untouched when the item is
   missing or stock is insufficient.
5. `SortedItems(pantry)` returns all item names in alphabetical order.
6. `NewSet(items)` returns a `map[string]struct{}` in which duplicates
   collapse; `Has(set, item)` reports membership.
7. `go test ./...` passes, and the code is `gofmt`-formatted.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...
```

They fail on the starter — make them green, one function at a time.

## Further reading

- [Go blog — Go maps in action](https://go.dev/blog/maps)
- [A Tour of Go — Maps](https://go.dev/tour/moretypes/19)
- [Effective Go — Maps](https://go.dev/doc/effective_go#maps)
- [pkg.go.dev — the maps package](https://pkg.go.dev/maps)
