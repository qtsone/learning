# Arrays & Slices

> `go.basics.arrays-slices` · ~3-4h · Stage: Programming Basics (Go)

## Objectives

By the end of this lesson you can:

- Explain the difference between an array and a slice, including the slice header (pointer, len, cap).
- Predict the length and capacity of a slice after slicing and `append` operations.
- Explain why `append` sometimes returns a new backing array and why its result must be reassigned.
- Use `copy` to duplicate a slice and explain when two slices alias the same backing array.
- Implement common slice manipulations (insert, remove, filter) without index bugs.

## Many values, one name

Every variable you've declared so far held exactly one value. Real programs deal
in collections: the lines of a file, the scores of a game, the arguments of a
command. Go's foundation for "a row of values" is the **array**, and the tool
you'll actually use every day is built on top of it: the **slice**.

## Arrays: fixed length, copied whole

An array holds a fixed number of elements of one type:

```go
var week [7]string      // 7 strings, each "" (zero values, as always)
week[0] = "Monday"      // index with [ ], counting from 0
fmt.Println(len(week))  // 7
```

Two things make arrays rigid. First, the length is part of the type:
`[7]string` and `[3]string` are different types, so a function taking one
cannot accept the other. Second, arrays are plain values — assigning one
copies **all** its elements:

```go
a := [3]int{1, 2, 3}
b := a             // full copy of all three elements
b[0] = 99
fmt.Println(a[0])  // 1 — a is untouched
```

Fixed length and copy-everything semantics are why you'll rarely declare an
array yourself. But every slice you use has an array underneath, so you need
this picture in your head.

## Slices: a window onto an array

Drop the length from the brackets and you get a slice:

```go
nums := []int{3, 1, 4}    // slice literal — Go builds the array for you
nums = append(nums, 1)    // grows: 3 1 4 1
```

A slice is *not* the data. It's a small descriptor — the **slice header** —
with three fields:

- a **pointer** to an element of an underlying array (an address in memory
  saying "the elements start here" — pointers get their own lesson later this
  stage; that working definition is enough for now),
- **len** — how many elements the slice can see (`len(s)`),
- **cap** — how many elements the underlying array has room for, counting
  from the pointer onward (`cap(s)`).

`make` builds a slice with a chosen length and capacity:

```go
s := make([]int, 3)     // len 3, cap 3, elements all 0
t := make([]int, 0, 8)  // len 0, cap 8 — empty, with room to grow
```

The zero value of a slice type is `nil`: `var s []int` has len 0 and no
backing array yet — and, as you'll see below, it is perfectly ready to grow.

## Slicing: `s[lo:hi]`

The slicing expression takes a sub-window. It is **half-open**: it includes
index `lo` and excludes `hi`, so the result's length is `hi - lo`. That
convention is what keeps index arithmetic sane: `s[:i]` and `s[i:]` split a
slice into two parts that never overlap, and `s[0:len(s)]` is the whole thing.
Either bound can be omitted: `s[:2]`, `s[2:]`, `s[:]`.

Crucially, slicing does **not** copy anything. The new header points into the
*same* backing array:

```go
s := []int{10, 20, 30, 40, 50}
mid := s[1:4]      // len 3 (20 30 40), cap 4
mid[0] = 99
fmt.Println(s[1])  // 99 — same backing array!
```

When two slices share a backing array like this, they **alias** each other:
a write through one is visible through the other. The capacity arithmetic:
`cap(mid)` is `cap(s) - 1 = 4`, because capacity always runs from the
pointer to the *end of the backing array*, not to `hi`.

## `append`, capacity, and the reassignment rule

`append` adds elements at the end and returns the resulting slice. Two cases:

- **Room left** (`len < cap`): it writes into the existing backing array and
  returns a header with a bigger `len`.
- **Full** (`len == cap`): it allocates a new, bigger array, copies every
  element over, appends there, and returns a header pointing at the **new**
  memory. The old array is left behind.

You cannot know from the call site which case happened. So there is one rule,
always: **reassign the result to the same variable**.

```go
s = append(s, v)   // always this shape
```

Writing `t := append(s, v)` and then using both `t` and `s` is a trap: whether
they alias depends on whether the append fit in capacity — a bug that appears
only for certain lengths. Watch the growth happen:

```go
s := make([]int, 0, 2)
s = append(s, 1)   // len 1, cap 2 — same array
s = append(s, 2)   // len 2, cap 2 — same array, now full
s = append(s, 3)   // len 3, cap 4 — NEW array, old elements copied
```

(The exact growth factor is an implementation detail; "roughly doubles" is the
right mental model.) And because `append` handles allocation itself, it works
on a nil slice too — this filter shape is idiomatic Go:

```go
var out []int
for _, v := range s {
    if v > 10 {
        out = append(out, v)
    }
}
```

That loop uses `for … range`, the loop form made for collections: each pass,
`range` hands you the index and the element (`for i, v := range s`). Use `_`
to discard the index, exactly like discarding any other value.

## An independent duplicate: `copy`

Assigning a slice copies only the three-field header, never the elements —
`dst := src` gives you two windows onto one array. When you need a genuinely
independent duplicate, allocate and copy:

```go
dst := make([]int, len(src))
copy(dst, src)   // copies min(len(dst), len(src)) elements
```

`copy` never grows the destination: copying into an empty slice copies
nothing, silently. Size `dst` first, always.

One more thing slices *can't* do: `==` does not compile on slices (the only
legal comparison is to `nil`). Should it compare the headers or the
elements? Go refuses to guess. To ask whether two slices hold the same
contents, check the lengths first, then loop and compare element by element
— exactly what the exercise's tests do in their hand-written `equal` helper.
Later lessons lean on this fact.

## Exercise

Open [`exercise/`](exercise/) — a Go module with three files:

- `main.go` — a small demo; `go run .` to eyeball your functions' output.
- `slicelab.go` — four functions marked `TODO` for you to implement.
- `slicelab_test.go` — the spec. **Read it**: it snapshots each input before
  the call and checks it afterwards, so mutation bugs have nowhere to hide.

House rule: none of the functions may modify the slice it receives. Build and
return a fresh slice instead.

Acceptance criteria:

1. `Clone(s)` returns a slice with the same elements as `s` that shares no
   backing array with it — changing one never changes the other.
2. `Insert(s, i, v)` returns a new slice with `v` inserted at index `i`
   (`0 ≤ i ≤ len(s)`); inserting at `len(s)` appends to the end.
3. `Remove(s, i)` returns a new slice with the element at index `i` removed
   (`0 ≤ i < len(s)`).
4. `KeepAbove(s, limit)` returns the elements of `s` strictly greater than
   `limit`, in their original order.
5. No function modifies its input, `go test ./...` passes, and the code is
   `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They fail before you write any code — make them green, one function at a time.

## Further reading

- [A Tour of Go — Slices](https://go.dev/tour/moretypes/7)
- [Go blog — Go Slices: usage and internals](https://go.dev/blog/slices-intro)
- [Go blog — Arrays, slices (and strings): the mechanics of `append`](https://go.dev/blog/slices)
- [Effective Go — Slices](https://go.dev/doc/effective_go#slices)
