# Pointers

> `go.basics.pointers` · ~3-4h · Stage: Programming Basics (Go)

## Objectives

By the end of this lesson you can:

- Explain what `&` and `*` do and trace what a pointer holds versus what it
  points to.
- Explain why Go is pass-by-value and demonstrate how a pointer parameter lets
  a function mutate its argument.
- Predict when dereferencing causes a nil pointer panic and guard against it.
- Choose between passing a value and passing a pointer and justify the choice
  (mutation, size, nil-ability).

## Every variable lives somewhere

When you write `x := 42`, the machine sets aside a small piece of memory and
stores 42 in it. That piece of memory has an **address** — a number that says
*where* the box is, not what's in it. Most of the time you don't care: the name
`x` is all you need. Pointers are for the moments when you need the *where*.

A **pointer** is a value that holds the address of another value. You get one
with `&`, the *address-of* operator:

```go
x := 42
p := &x          // p now holds the address of x
fmt.Println(p)   // something like 0xc000014088 — an address, not 42
fmt.Println(*p)  // 42 — follow the pointer and read what's there
```

The type of `p` is `*int` — read it as "pointer to int". A `*string` points at
a string, a `*Player` at a `Player`, and so on. Pointers are typed: a `*int`
can never point at a string, and the compiler enforces that.

Keep two things separate in your head, because they are separate:

- what `p` **holds** — an address (`0xc000014088`)
- what `p` **points to** — the int 42 sitting at that address

`*p` is the *dereference* operator: "go to the address in `p` and give me the
value there". It works for writing too:

```go
*p = 100
fmt.Println(x) // 100
```

There is only one box. `x` and `*p` are two names for the same memory, so
writing through one is visible through the other. That's the entire trick —
everything else in this lesson is consequences.

One notational wart to accept early: the `*` symbol has two jobs. In a *type*
(`*int`) it means "pointer to". In an *expression* (`*p`) it means
"dereference". Same character, different positions, different meanings — like
how `=` and `==` look related but do different things.

## Go passes copies

Here is a function that tries to change its argument and fails:

```go
func bump(n int) {
	n = n + 1 // changes the copy; the caller never sees it
}

x := 5
bump(x)
fmt.Println(x) // still 5
```

When you call `bump(x)`, Go copies the value 5 into the parameter `n`. Inside
the function, `n` is a brand-new variable in a brand-new box. Incrementing it
touches only the copy. This is **pass-by-value**, and Go does it for *every*
argument, always — ints, strings, structs, everything. You saw the same rule in
the structs lesson: assigning a struct copies all its fields.

If the caller should see the change, don't pass the value — pass its address:

```go
func bump(n *int) {
	*n = *n + 1 // follow the pointer, change the shared box
}

x := 5
bump(&x)
fmt.Println(x) // 6
```

Note what did *not* change: Go still copied the argument. It copied the
*pointer* — the callee's `n` is a copy of the address. But a copy of an address
points at the same box, so writing through it mutates the caller's `x`.
Pass-by-value is never suspended; pointers just make the copied value a
*location* instead of the data itself.

Multiple assignment works through dereferences too, which gives you the
classic swap:

```go
func swap(a, b *int) {
	*a, *b = *b, *a
}
```

Try to write `swap` without pointers — you can't. The function would swap two
copies and the caller would never know.

This also retroactively explains the arrays-slices lesson: a function *can*
change a slice's elements because the slice header it receives — a copy — 
contains a pointer to the backing array. Maps behave similarly. You've been
using pointers all along; now they have a name.

## nil: the pointer to nowhere

Like every type in Go, pointers have a zero value. For pointers it is `nil`:
"points at nothing". A declared-but-unassigned pointer is nil, and nil is what
you get when there's no sensible address to hand out.

Dereferencing nil is asking Go to read from nowhere, and the program crashes:

```go
var p *int
fmt.Println(*p)
```

```
panic: runtime error: invalid memory address or nil pointer dereference
```

A **panic** is a runtime crash — the compiler cannot catch this for you,
because whether `p` is nil depends on what happened while the program ran.
This is one of the most common crashes in real Go code, so build the reflex
now: *before* dereferencing a pointer that might be nil, guard it:

```go
if p != nil {
	fmt.Println(*p)
}
```

Nil isn't only a hazard — it's also a feature. A `*int` can express "no value
at all", which a plain `int` cannot: is `0` a real measurement or just the
zero value? With a `*int`, nil means *absent* and a non-nil pointer to 0 means
*measured zero*. This is the same missing-versus-zero problem the comma-ok
idiom solved for maps, answered with a different tool.

## Pointers and structs

Pointers earn their keep with structs. Two idioms to learn:

**Taking the address of a struct literal.** You can create a struct and get a
pointer to it in one expression:

```go
p := &Player{Name: "Gopher", HP: 100, MaxHP: 100} // p is a *Player
```

**Field access auto-dereferences.** Strictly, reaching a field through a
pointer would be `(*p).Name`. Go finds that so tedious it does it for you:

```go
p.Name = "Gogh" // shorthand for (*p).Name = "Gogh"
```

So code that works with `*Player` reads exactly like code that works with
`Player` — the dot does the right thing either way.

Finally: returning a pointer to a local variable is **safe** in Go.

```go
func NewPlayer(name string) *Player {
	pl := Player{Name: name, HP: 100, MaxHP: 100}
	return &pl // fine — pl stays alive as long as someone points at it
}
```

If you've heard horror stories from C about returning addresses of locals,
relax: the Go compiler notices that `pl`'s address escapes the function and
keeps the value alive; the garbage collector reclaims it when nothing points
at it anymore. You never manage any of that. Constructor functions shaped
exactly like `NewPlayer` are everywhere in real Go code.

## Value or pointer?

For every function you write, you now have a choice: take `Player` or
`*Player`? Decide with three questions:

1. **Mutation** — must the function change the caller's value? Then a pointer
   is the only option; a value parameter edits a copy.
2. **Size** — is the struct large and passed around constantly? A pointer
   copies one address instead of every field. (Don't over-apply this: copying
   small structs is cheap, and clarity beats micro-optimization.)
3. **Nil-ability** — do you need "absent" as a legitimate state? Only a
   pointer can be nil.

If none of the three apply, prefer the value. Values can't be nil, can't be
mutated behind your back from across the program, and are easier to reason
about. And you rarely need a pointer *to* a slice or map — they already
behave reference-like for their contents.

Whichever you choose, be consistent for a given type across your codebase.
The next lesson, methods, builds directly on this exact choice.

## Exercise

Open [`exercise/`](exercise/) — a Go module with the package `player`:

- `player.go` — a `Player` struct and four functions with `TODO`s.
- `player_test.go` — the specification. **Read it first**, then make it green.

Acceptance criteria:

1. `Swap(&a, &b)` exchanges the values of `a` and `b`.
2. `ValueOr(p, fallback)` returns `*p` when `p` is non-nil, and `fallback`
   when `p` is nil — it must never panic.
3. `Heal(&p, n)` raises `p.HP` by `n` but never above `p.MaxHP`.
   `Heal(nil, n)` does nothing — healing nobody must not panic.
4. `NewPlayer(name, maxHP)` returns a non-nil `*Player` at full health
   (`HP == MaxHP`), and every call returns a fresh, independent `Player` —
   mutating one must not affect another.
5. `go test ./...` passes, and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They FAIL on the starter code — that's your worklist. If a test *panics* with
`nil pointer dereference`, that's not a broken test: it's criterion 2 or 3
telling you a guard is missing.

## Further reading

- [A Tour of Go — Pointers](https://go.dev/tour/moretypes/1)
- [The Go Programming Language Specification — Address operators](https://go.dev/ref/spec#Address_operators)
- [Dave Cheney — Understand Go pointers in less than 800 words](https://dave.cheney.net/2017/04/26/understand-go-pointers-in-less-than-800-words-or-your-money-back)
