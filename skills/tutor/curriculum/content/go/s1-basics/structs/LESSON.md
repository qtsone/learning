# Structs

> `go.basics.structs` · ~2-3h · Stage: Programming Basics (Go)

## Objectives

By the end of this lesson you can:

- Define structs and initialize them with field-named literals, and explain
  why positional literals are fragile.
- Explain how struct embedding promotes fields and how it differs from a
  named field.
- Predict when two struct values can be compared with `==` and when
  comparison fails to compile.
- Model a small domain (a contact list) using structs with appropriate
  field types.

## Why structs

You can already group *many values of one type*: a slice of strings, a map
from string to int. But real things are made of *different* types that
belong together. A contact has a name (string), an age (int), an email
(string). A map can't hold that — all its values share one type. Keeping
three parallel slices (`names`, `ages`, `emails`) works until one insert
forgets a slice and index 4 means two different people.

A **struct** fixes this: one value with named **fields**, each with its own
type. Where slices and maps are collections of *many like things*, a struct
is *one thing with parts*.

## Defining a struct type

The `type` keyword introduces a new named type. For structs, you list the
fields between braces:

```go
type Contact struct {
	Name  string
	Age   int
	Email string
}
```

Now `Contact` is a type like `string` or `int`: you can declare variables
of it, pass it to functions, return it, put it in a slice (`[]Contact`).

Its zero value follows the rule you know from the variables lesson, applied
per field: `var c Contact` gives you a Contact whose `Name` and `Email` are
`""` and whose `Age` is `0`. No special initialization needed — a
zero-value struct is ready to use.

## Struct literals: name your fields

To build a struct with content, use a **composite literal**. There are two
forms:

```go
c := Contact{Name: "Ada Lovelace", Age: 36, Email: "ada@example.com"}  // field-named
c := Contact{"Ada Lovelace", 36, "ada@example.com"}                    // positional
```

Always prefer the field-named form. The positional form is fragile:

- It breaks silently. Reorder `Name` and `Email` in the type (both strings)
  and every positional literal still compiles — with the values swapped.
- It breaks loudly at the worst time. Add a field to the type and every
  positional literal in the codebase stops compiling.
- It says nothing. `Contact{"Ada", 36, "x@y"}` forces the reader to go look
  up the type; the named form documents itself.

With named fields you can also omit fields — anything you skip gets its
zero value:

```go
c := Contact{Name: "Ada Lovelace"}   // Age: 0, Email: ""
```

## Fields: read, write, copy

You reach fields with a dot, and they are ordinary variables:

```go
c.Age = 37
fmt.Println(c.Name, "is", c.Age)
fmt.Printf("%+v\n", c)   // {Name:Ada Lovelace Age:37 Email:...} — great for debugging
```

One thing to internalize now: **structs are values**. Assigning a struct,
or passing it to a function, copies *all* of its fields:

```go
d := c
d.Age = 99
fmt.Println(c.Age)   // still 37 — d is an independent copy
```

So a function that "changes" a struct it received actually changes its own
copy. The idiom at this point in the roadmap is to return the changed copy
and let the caller keep it. The next lesson (pointers) shows how to share
one struct instead of copying — until then, copies are all you need.

## Embedding: a struct inside a struct

A field can itself be a struct. There are two ways to nest, and they look
almost the same:

```go
type Person struct {
	Name string
	Age  int
}

type Employee struct {
	P      Person   // named field
	Salary int
}

type Contact struct {
	Person          // embedded: just the type, no field name
	Email string
}
```

With the named field you write `e.P.Name`. With **embedding**, Go
**promotes** the inner fields: `c.Name` is shorthand for `c.Person.Name` —
both work, they touch the same field. The embedded version reads as
"a Contact *is built from* a Person, plus an email".

Two things embedding is *not*:

- It is not a merge. `Contact` still has a real field named `Person`
  (the field name is the type name), and literals must say so:

  ```go
  c := Contact{Person: Person{Name: "Ada Lovelace", Age: 36}, Email: "ada@example.com"}
  ```

  `Contact{Name: "Ada"}` does not compile — promotion works for *access*,
  not for literals.

- It is not inheritance. `Contact` is not a `Person` and can't be used
  where a `Person` is expected. It just contains one, with shorter access.
  Embedding grows more powerful later in this stage when types gain
  behavior, not just data.

## Comparing structs with ==

Two struct values of the same type can be compared with `==`, and the
comparison is field by field — *all* fields must match:

```go
a := Contact{Person: Person{Name: "Ada Lovelace", Age: 36}}
b := Contact{Person: Person{Name: "Ada Lovelace", Age: 36}}
fmt.Println(a == b)   // true — same data, even though built separately
```

Note what this means: equality is about *content*, not about being "the
same object".

There is a catch. `==` only compiles if **every field is comparable** —
numbers, strings, booleans, and structs of such fields all are. Slices and
maps are **not** comparable (you met this in the slices lesson: `==` on
slices doesn't compile). One slice field poisons the whole struct:

```go
type Group struct {
	Name    string
	Members []string
}
g1 == g2   // compile error: struct containing []string cannot be compared
```

The good news: it's a *compile-time* error, never a runtime surprise. When
`==` won't compile, you write the comparison yourself, field by field, with
a loop for the slice. And a bonus you'll recognize from the maps lesson:
because comparable structs support `==`, they can be map keys —
`map[Person]string` is legal.

## Exercise

Open [`exercise/`](exercise/) — a Go module for a tiny address book:

- `contacts.go` — the types (`Person`, `Contact` with embedding, `Group`)
  are defined for you; five functions carry `TODO`s.
- `main.go` — a small demo you can `go run .` once things work.
- `contacts_test.go` — the specification. **Read it first.**

Acceptance criteria:

1. `NewContact(name, age, email, phone)` returns a `Contact` with every
   field set, using a field-named literal — `Name` and `Age` live in the
   embedded `Person`.
2. `Rename(c, newName)` returns a copy of `c` with only the name changed;
   the caller's original is untouched.
3. `SameContact(a, b)` reports whether two contacts hold exactly the same
   data — one `==` is all it takes.
4. `FindByEmail(book, email)` returns the matching contact and `true`, or
   the zero `Contact` and `false` when nothing matches (the comma-ok shape
   you know from maps).
5. `SameGroup(a, b)` compares two groups: same name, same members, same
   order. `==` on `Group` won't compile — before writing the loop, try it
   once and read the error.
6. `go test ./...` passes and the code is gofmt-formatted.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...
```

They fail on the starter — make them green, one function at a time.

## Further reading

- [A Tour of Go — Structs](https://go.dev/tour/moretypes/2)
- [Effective Go — Composite literals](https://go.dev/doc/effective_go#composite_literals)
- [Go spec — Struct types](https://go.dev/ref/spec#Struct_types)
- [Go spec — Comparison operators](https://go.dev/ref/spec#Comparison_operators)
