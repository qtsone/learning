# Functions

> `go.basics.functions` · ~2-3h · Stage: Programming Basics (Go)

## Objectives

By the end of this lesson you can:

- Implement functions with multiple return values and use the (result, error) convention.
- Explain what named return values do and when they help or hurt readability.
- Implement a variadic function and call it by spreading a slice with `...`.
- Decompose a small program into functions with clear single responsibilities.

## Anatomy of a function

You have been calling functions since your first program (`fmt.Println`), and
you completed one in hello-world (`Greeting`). Now the full picture. A
**function** names a piece of work: values go in, a result comes back.

```go
func Half(total int) int {
	return total / 2
}
```

Read the first line left to right:

- `func` — declares a function.
- `Half` — its name; call it as `Half(90)`.
- `(total int)` — the **parameters**: names the function gives to incoming
  values. The values the caller actually sends (`90`) are **arguments**.
- The `int` after the parentheses — the **result type**: what comes back.
- `return` hands a value to the caller and ends the function.

Consecutive parameters of one type can share it: `func Split(total, people int)`
makes both `int`. Every parameter and result is typed — as in Variables &
Types, the compiler holds you to the declared types at every call.

Why not pile everything into `main`? Because a named, self-contained unit can
be reused, reasoned about, and — as hello-world's tests showed — tested in
isolation. This lesson's exercise is built on that idea.

## Returning more than one value

Go functions can return as many values as you declare. The classic need:
integer division. `1000 / 3` is `333`, and the leftover cent vanishes —
unless you also return `%`, the division remainder:

```go
func Split(total, people int) (int, int) {
	return total / people, total % people
}
```

The result types now sit in parentheses, and `return` supplies one value per
result, in order. The caller assigns them the same way:

```go
share, remainder := Split(1000, 3) // share = 333, remainder = 1
```

You must do something with every result. To discard one deliberately, use the
**blank identifier** `_` you've seen in starter code:

```go
share, _ := Split(1000, 3)
```

## When work can fail: the (result, error) pair

What should `Split(1000, 0)` do? Dividing by zero would crash the program.
Go's answer is a convention you will meet in every Go codebase: a function
that can fail returns its normal results *plus* a final value of type
`error`.

`error` is an ordinary type whose values describe what went wrong. The
standard library's `errors` package builds one from a message, and the
special value `nil` means "no error — it worked":

```go
import "errors"

func Split(total, people int) (int, int, error) {
	if people < 1 {
		return 0, 0, errors.New("split: need at least one person")
	}
	return total / people, total % people, nil
}
```

Note the shape: on failure, zero values for the normal results and a non-nil
error; on success, real results and `nil`. The caller checks the error
*first*, before touching the results:

```go
share, remainder, err := Split(1000, people)
if err != nil {
	fmt.Println("cannot split:", err)
	return
}
```

`if err != nil` is the most common statement in Go, and its repetitiveness is
deliberate: failures are handled where they happen, with control flow you
already know, not by an invisible mechanism. There is much more to errors —
wrapping, sentinel values — and this stage has a whole lesson on them;
`errors.New` plus the check above is all you need for now.

## Named return values

Result types can carry names, just like parameters:

```go
func Split(total, people int) (share, remainder int, err error) {
```

Two things happen. First — the real benefit — the signature documents itself:
a caller reading only this line knows which `int` is which. Second, the names
become variables inside the function, starting at their zero values, and a
bare `return` (no values) returns whatever they currently hold — a **naked
return**.

Name your results whenever a function returns two values of the same type;
`(int, int, error)` makes callers guess. But treat naked returns with
suspicion: in any function longer than a few lines, a bare `return` forces
the reader to scan backwards to learn what is actually returned. Named
results with *explicit* return values give you the documentation without the
guesswork — that's the style this lesson's solution uses.

## Variadic functions

`fmt.Println` accepts one argument, or three, or none. You can write that
too: mark the final parameter **variadic** with `...`:

```go
func Sum(prices ...int) int {
	total := 0
	for _, price := range prices {
		total += price // shorthand for total = total + price
	}
	return total
}
```

Callers pass as many `int`s as they like: `Sum()`, `Sum(100)`,
`Sum(1250, 899, 1451)`. Inside the function, `prices` is a **slice** — Go's
list type, written `[]int`. Slices get a full lesson later this stage; today
you need two moves. A slice **literal** creates one:

```go
prices := []int{1250, 899, 1451}
```

and the `for … range` loop above visits each element in order, setting
`price` to the next value on every pass. Notice it names *two* variables
where control-flow's `range n` had one: ranging over a collection yields a
pair per pass — the element's position, then the element itself. Here only
the price matters, so the position goes to `_` — the blank identifier from
earlier in this lesson, doing its discard job again. The full two-value story
arrives with arrays-slices later this stage.

How do you hand that slice to `Sum`? Not `Sum(prices)` — the compiler rejects
it, because `prices` is a `[]int` where an `int` is expected. **Spread** it
with `...` at the call site:

```go
Sum(prices...) // each element becomes one argument
```

The same spread forwards one variadic function's parameters to another —
you'll need exactly that in the exercise. Only the last parameter of a
function may be variadic.

## Small functions with one job

The exercise asks for `SplitBill(people, prices...)`: total the prices, then
split among the people. You could write one big function that loops, guards,
and divides — but you'll already have `Sum` and `Split`. Connect them
instead:

```go
func SplitBill(people int, prices ...int) (share, remainder int, err error) {
	return Split(Sum(prices...), people)
}
```

Each function keeps one responsibility: `Sum` adds, `Split` divides and
guards, `SplitBill` only connects them. When a test fails, its name tells you
which job is broken; when a rule changes ("minimum one person" becomes
"minimum two"), exactly one function changes. Rule of thumb: you should be
able to say what a function does in one short sentence with no "and" in it.

Notice too that `return` can hand back another call's results directly when
the shapes match — `Split`'s error flows through `SplitBill` to the caller
untouched.

## Exercise

Open [`exercise/`](exercise/) — a Go module that splits a restaurant bill.
Money is in whole **cents** (`1250` is $12.50): floating-point math rounds
oddly, so real money code sticks to integers — take that habit early.

- `bill.go` — three functions for you to complete (`Sum`, `Split`,
  `SplitBill`) and one already finished (`FormatCents`) as an example of a
  small helper with one job.
- `main.go` — two `TODO`s wiring the functions into a program.
- `bill_test.go` — the specification. Read it first.

Acceptance criteria:

1. `Sum()` returns `0`; `Sum(100, 200, 300)` returns `600`; and it works when
   called by spreading a slice: `Sum(prices...)`.
2. `Split(total, people)` returns the per-person share and the remainder
   using `/` and `%`; when `people < 1` it returns zero results and a
   non-nil error.
3. `SplitBill(people, prices...)` is built from `Sum` and `Split` — it does
   not re-add the prices or re-check `people` itself — and forwards its
   variadic prices with `prices...`.
4. `main` splits `[]int{1250, 899, 1451}` among 3 people and prints each
   share using `FormatCents`; if the split fails it prints the error and
   returns. Check with `go run .`.
5. `go test ./...` passes, and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They fail on the starter — read the first failure, fix that one function,
re-run.

## Further reading

- [A Tour of Go — Functions](https://go.dev/tour/basics/4)
- [A Tour of Go — Multiple results](https://go.dev/tour/basics/6)
- [Effective Go — Multiple return values and named results](https://go.dev/doc/effective_go#multiple-returns)
- [Go spec — Passing arguments to `...` parameters](https://go.dev/ref/spec#Passing_arguments_to_..._parameters)
