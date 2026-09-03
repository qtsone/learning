# Generics

> `go.intermediate.generics` · ~3-4h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Implement a generic function with type parameters and a constraint
  interface (e.g. a Map/Filter over slices).
- Explain the difference between the `comparable` constraint, constraints
  built from type sets (`~int | ~string`), and method-based constraints.
- Choose between a generic function and an interface-based design for a given
  problem and justify the trade-off.
- Explain why type inference sometimes fails and when explicit instantiation
  is required.
- Identify code where generics add complexity without benefit and argue for
  the simpler non-generic form.

## The problem: one algorithm, many types

In S2 you wrote linear search over a slice of ints. Suppose you now need it
for strings. The algorithm is identical — walk the slice, compare, return the
index — only the type changes. Before Go 1.18 you had two options, and you
have already felt the cost of both:

1. **Copy-paste** — `IndexOfInt`, `IndexOfString`, `IndexOfFloat64`, …
   Three copies of one idea, three places for a bug to hide.
2. **`any` plus type assertions** — one function taking `[]any`, checked at
   runtime. You know from the type-assertions lesson what that trades away:
   the compiler stops helping you, and mistakes surface as runtime panics or
   `ok == false` branches instead of compile errors.

Generics give you the third option: write the algorithm once, keep full
compile-time type checking.

## Type parameters

A generic function declares **type parameters** in square brackets, before
the ordinary parameters:

```go
func IndexOf[T comparable](s []T, target T) int {
	for i, v := range s {
		if v == target {
			return i
		}
	}
	return -1
}
```

Read the signature inside out:

- `[T comparable]` — "this function is parameterized over a type `T`, and
  `T` must satisfy the constraint `comparable`."
- `s []T, target T` — the slice's elements and the target are the *same*
  type, whatever it turns out to be. That relationship is the point: with
  `[]any` you could accidentally search a `[]int` for a `string`; here the
  compiler rejects it.

At each call site the compiler picks a concrete type for `T` — this is
called **instantiation** — and type-checks the whole call as if you had
written the specialized version by hand:

```go
IndexOf([]int{3, 1, 4}, 4)          // T = int
IndexOf([]string{"a", "b"}, "b")    // T = string
IndexOf([]int{3, 1, 4}, "x")        // compile error: type mismatch
```

Everything is settled at compile time. There are no runtime type assertions
hiding inside, nothing to panic.

## Constraints are interfaces

A **constraint** describes what the function is allowed to *do* with `T`.
Constraints are written as interfaces, but they are checked at compile time
against the type argument, not at runtime against a value.

The loosest constraint is `any` (an alias for `interface{}`): it promises
nothing, so inside the function you can only move values of type `T` around
— assign them, put them in slices, pass them along. You cannot compare them,
add them, or call methods on them. That is exactly enough for something like
`Map`:

```go
func Map[T, U any](s []T, f func(T) U) []U
```

`IndexOf` needs more: it uses `==`. The built-in constraint `comparable`
permits `==` and `!=` — the same property Go requires of map keys, which is
why `comparable` is also the constraint you'll use for generic map helpers.

The rule to internalize: **the constraint must grant every operation the
body performs.** If you write `v == target` under `[T any]`, the compiler
refuses — `any` never promised equality.

## Type sets and the tilde

What about `Sum`, which needs `+`? No method expresses "supports `+`", so
constraints can instead list types, as a union:

```go
type Number interface {
	~int | ~int64 | ~float64
}

func Sum[T Number](s []T) T {
	var total T
	for _, v := range s {
		total += v
	}
	return total
}
```

`Number`'s **type set** is "any type whose underlying type is `int`,
`int64`, or `float64`". The body may then use any operation that *all* types
in the set support — arithmetic, in this case.

The tilde is load-bearing. You define types on top of basic types all the
time — remember methods in S1, where you hung behavior off defined types:

```go
type Celsius float64
```

`Celsius` is a distinct type; it is *not* `float64`. A union written
`int | int64 | float64` (no tildes) admits only those exact three types, so
`Sum[Celsius]` would not compile. `~float64` means "float64, or anything
*defined on* float64" — which is almost always what you want. When in doubt,
write the tilde.

Two notes so you're not surprised later:

- An interface containing a union can only be used as a constraint. You
  cannot declare `var n Number` — it is not a runtime interface type.
- You do not have to invent common constraints yourself: the standard
  library's `cmp` package (Go 1.21+) ships `cmp.Ordered`, the constraint for
  everything supporting `<`, which is what generic min/max/sort helpers use.

## Method-based constraints

Any ordinary interface also works as a constraint. Then the type argument
must have the methods, and the body may call them:

```go
func DescribeAll[T fmt.Stringer](s []T) []string
```

Wait — you know interfaces. Why not just `func DescribeAll(s []fmt.Stringer)
[]string`? Because of a fact you met in the interfaces lesson from the other
direction: a `[]point` is **not** a `[]fmt.Stringer`, even when `point`
implements `Stringer`. Slice types don't convert element-wise; the caller
would have to copy every element into a new `[]fmt.Stringer` first. With the
generic version, `T = point` and the caller passes their `[]point` as-is.

This is the honest difference between the two designs:

- Interface parameter: the function receives *values that can differ in
  concrete type* from one element to the next; dispatch happens at runtime.
- Generic with a method constraint: *all* elements share one concrete type,
  fixed at compile time; you keep the caller's slice type intact.

## Type inference and explicit instantiation

You rarely write the square brackets at call sites, because the compiler
**infers** type arguments from the function arguments: in
`IndexOf([]string{"a"}, "a")`, the `[]string` forces `T = string`.

Inference works *from the arguments you pass*. It fails when there is
nothing to infer from:

```go
double := Map[int, int]        // no call, no arguments — you must instantiate
double([]int{1, 2}, func(v int) int { return v * 2 })
```

Assigning a generic function to a variable (or passing it as an argument)
without calling it gives the compiler zero material to work with, so you
supply the type arguments explicitly. The same applies when a type parameter
appears only in the return type — `func Zero[T any]() T` can only ever be
called as `Zero[int]()`, because no argument mentions `T`.

When you get an error like `cannot infer T`, don't fight it: either add the
explicit instantiation or restructure so the type shows up in a parameter.

## Generics or interfaces?

You now have two tools that both say "this code works with more than one
type". They are not competitors; they answer different questions.

- **Reach for an interface when the *behavior* varies.** `io.Reader` works
  because each implementation reads differently — a file, a network socket,
  a string in memory. The caller doesn't know or care which it gets at
  runtime. This is the "accept interfaces" half of the idiom you learned,
  and generics change nothing about it.
- **Reach for generics when the *type* varies but the code is identical.**
  `IndexOf` does exactly the same thing for ints and strings; there is no
  per-type behavior to abstract. Containers and slice/map helpers are the
  home turf: note that both constraints (`comparable`, `Number`) express
  *operations*, not behavior.

A useful smell test: if you find yourself writing a type switch inside a
generic function to do something different per type, you wanted an interface
(or separate functions). If you find yourself copying a function to change
only the types in its signature, you wanted generics.

## When generics make code worse

Generics are the newest hammer in your bag, so this deserves its own
section: **most Go functions should not be generic.**

Signs a generic is not earning its keep:

- **One instantiation.** If `T` is only ever `User`, delete the type
  parameter and write `User`. Abstraction you don't use is pure cost.
- **The constraint names one type.** `[T ~string]` with no second type in
  sight is a defined-type ceremony, not generality.
- **It's generic because it could be, not because callers need it.** Write
  the concrete version first. Generalize when the *second or third* caller
  with a different type actually appears — the duplication will tell you,
  and by then you'll know the right constraint.
- **The signature became harder to read than the duplication it removed.**
  A reviewer should understand the signature in one pass.

Also know what already exists: the standard library's `slices` and `maps`
packages (Go 1.21+) provide `slices.Index`, `slices.Contains`,
`slices.SortFunc`, and friends. In this exercise you re-implement a few of
them to learn the mechanics; in production code, use the stdlib.

## Exercise

Open [`exercise/`](exercise/) — a Go module with package `sliceops`:

- `sliceops.go` — five function stubs and the `Number` constraint. The
  signatures and constraint are final; your work is the bodies, marked
  `TODO`.
- `sliceops_test.go` — the specification. **Read it first**: note the
  defined types `Celsius` and `point` it uses, and the explicit
  instantiation `Map[int, int]`.

Acceptance criteria:

1. `Map(s, f)` returns a new slice with `f` applied to each element, in
   order, for any element and result types.
2. `Filter(s, keep)` returns the elements for which `keep` is true,
   preserving order.
3. `IndexOf(s, target)` returns the index of the first occurrence, or `-1`
   if absent, for any comparable element type.
4. `Sum(s)` totals a slice of any `Number` type — including defined types
   like `Celsius`.
5. `DescribeAll(s)` returns each element's `String()`, in order.
6. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...
```

They fail before you write the bodies — make them green.

Then one experiment (not graded by tests, but expect to discuss it): delete
the `~` before `float64` in the `Number` constraint and run `go test` again.
Read the compiler error slowly — it tells you precisely what the tilde was
doing. Put it back.

## Further reading

- [Go blog — An Introduction To Generics](https://go.dev/blog/intro-generics)
- [Go blog — When To Use Generics](https://go.dev/blog/when-generics)
- [Tutorial: Getting started with generics](https://go.dev/doc/tutorial/generics)
- [pkg.go.dev — the `slices` package](https://pkg.go.dev/slices)
