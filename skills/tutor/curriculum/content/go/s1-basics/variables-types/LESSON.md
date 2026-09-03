# Variables & Types

> `go.basics.variables-types` · ~2-3h · Stage: Programming Basics (Go)

## Objectives

By the end of this lesson you can:

- Choose between `var` declarations and `:=` short assignment and justify the choice in context.
- State the zero value for each basic type (numbers, string, bool) and explain why Go guarantees them.
- Explain what makes Go statically typed and predict when a type conversion is required.
- Declare typed and untyped constants and use `iota` to define a sequence of related constants.

## Naming values

In the last lesson every value lived inside a function call: `Println("Hello")`
used the string once and it was gone. A **variable** gives a value a name so
you can keep it, reuse it, and change it:

```go
var age int = 30
fmt.Println(age)      // 30
age = 31              // a birthday: same variable, new value
fmt.Println(age)      // 31
```

Read `var age int = 30` as: "declare a variable named `age`, of type `int`
(a whole number), starting at 30." After that, plain `=` **assigns** a new
value to the existing variable. Declaring creates the name; assigning changes
what it holds. You declare a name once, then assign as often as you like.

Go can usually see the type from the value, so you may omit it — and inside a
function there is an even shorter form, `:=`, which declares *and* assigns in
one step:

```go
var age = 30     // type int, inferred from 30
name := "Ada"    // short form: declare name, type string, value "Ada"
```

One strictness rule, in the same spirit as the unused-import error you may have
met last lesson: **a declared-but-unused variable is a compile error.** Go
would rather stop you than let dead names accumulate.

## var or := ?

Both create a variable, so which do you pick?

- `:=` — the default *inside functions* when you have an initial value. Short,
  and the type is obvious from the right-hand side.
- `var` — when you *want* the type spelled out (`var price float64 = 30` —
  otherwise `30` would infer `int`), when you want to start from the zero
  value (next section), or at **package level** (outside any function), where
  `:=` is not allowed.

If you remember one rule: `:=` for most in-function declarations, `var` when
the type or the starting value needs to be deliberate.

## The basic types

Four types cover most early Go programs:

| Type      | Holds                    | Examples          |
|-----------|--------------------------|-------------------|
| `int`     | whole numbers            | `0`, `42`, `-7`   |
| `float64` | numbers with a fraction  | `3.5`, `-0.01`    |
| `string`  | text                     | `"Hello"`, `""`   |
| `bool`    | truth values             | `true`, `false`   |

(There are sized variants like `int64` and `float32`; ignore them until you
need them — `int` and `float64` are the defaults for a reason.)

Numbers support arithmetic: `+ - * /`. Strings support `+` too, but it means
**concatenation** — `"Go" + "pher"` is `"Gopher"`, and `"1" + "2"` is `"12"`,
not `"3"`. The type decides what an operator does.

One trap to meet early: when *both* sides of `/` are integers, the result is
an integer — the fraction is thrown away. `7 / 2` is `3`, not `3.5`. This is
called integer division, and the exercise makes you fix it.

## Zero values

What does a variable hold before you assign anything?

```go
var count int      // 0
var price float64  // 0
var name string    // "" (the empty string)
var active bool    // false
```

In Go the answer is never "garbage" or "undefined": every type has a **zero
value**, and every declared variable starts there. Numbers start at `0`,
strings at `""`, booleans at `false`. Many languages leave uninitialized
memory unpredictable or crash at runtime; Go guarantees a known starting
point, so a forgotten assignment produces a *boringly predictable* bug instead
of a mysterious one. Go code leans on this — you will often see a bare
`var total int` that simply starts counting from its zero value.

## Static typing and conversions

Go is **statically typed**: every variable has one type, fixed at the moment
of declaration, and the compiler checks every use *before the program runs*.
`age = "thirty"` is refused at compile time, not discovered mid-run.

The compiler never converts between types silently, even "safe" directions
like `int` to `float64`. Mixing types is a compile error; when you mean it,
you say so with a **conversion** — the type used like a function:

```go
var cents int = 350
dollars := float64(cents) / 100   // 3.5 — convert first, then divide
truncated := int(3.9)             // 3 — float→int drops the fraction
```

Order matters: `float64(cents / 100)` divides *integers first* (350/100 = 3),
then converts the already-wrong answer. Predicting where a conversion is
required — and on which side — is the skill this lesson's tests check.

## Printf: printing with precision

Last lesson promised you `Printf`. It formats values into a template using
**verbs** — placeholders that say what type to expect:

```go
fmt.Printf("%d apples cost $%.2f each: %s\n", 3, 0.5, "bargain")
// 3 apples cost $0.50 each: bargain
```

The verbs you need now: `%d` (integer), `%f` (float — `%.2f` fixes two
decimals), `%s` (string), `%q` (string *with quotes*: `"Ada"`), `%t` (bool),
`%v` (any value, default format), and `%T` (the value's *type* — handy when
debugging inference). Unlike `Println`, `Printf` adds no newline: end the
template with `\n`.

Its sibling `fmt.Sprintf` formats the same way but **returns the string**
instead of printing it — which makes it testable, exactly like `Greeting` last
lesson. The exercise uses `Sprintf` throughout.

## Constants and iota

Some values must never change: days in a week, cents in a dollar. Declare them
with `const`; assigning to a constant is a compile error, so a typo that
"changes" one is caught for free:

```go
const CentsPerDollar = 100        // untyped: adapts to int or float64 context
const Greeting string = "hello"   // typed: fixed as string, like a variable
```

An *untyped* constant is flexible — `CentsPerDollar` can sit in `int` math or
`float64` math without conversion. A *typed* one behaves like a variable whose
value is locked. Prefer untyped unless you need the lock.

For a family of related constants, Go provides `iota` — a counter that starts
at 0 and increments on each line of a `const` block. A line without an
expression repeats the previous one:

```go
const (
	January = iota + 1   // 1
	February             // 2 (repeats "iota + 1" with the next iota)
	March                // 3
)
```

One expression, twelve months. You'll use exactly this shape in the exercise.

## Exercise

Open [`exercise/`](exercise/) — a Go module with three files:

- `types.go` — a `const` block and three functions with `TODO`s.
- `main.go` — prints everything, plus one small `TODO` of your own.
- `types_test.go` — the specification. **Read it first.**

Acceptance criteria:

1. `ZeroReport()` returns exactly `count=0 price=0 name="" active=false`,
   built with `fmt.Sprintf` from four variables declared with `var` and *no*
   explicit value — the zero values do the work. Hardcoding the string
   defeats the purpose.
2. `Average(7, 2)` returns `3.5` — convert to `float64` *before* dividing.
3. `PriceTag("coffee", 350)` returns `coffee: $3.50` — always two decimals,
   for any item and any price in cents. Use `:=` for the in-between value.
4. The weekday constants `Monday` through `Sunday` equal 1 through 7, produced
   by a single `iota` expression on the first line.
5. `main` also prints a `learning Go` line from a variable you declare with
   `:=` — run `go run .` and check with your own eyes.
6. `go test ./...` passes, and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They FAIL right now. Make them green, one function at a time.

## Further reading

- [A Tour of Go — Variables](https://go.dev/tour/basics/8)
- [A Tour of Go — Basic types](https://go.dev/tour/basics/11)
- [The Go Blog — Constants](https://go.dev/blog/constants)
- [pkg.go.dev — fmt (all the verbs)](https://pkg.go.dev/fmt)
