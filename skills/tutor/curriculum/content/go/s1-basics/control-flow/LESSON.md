# Control Flow

> `go.basics.control-flow` · ~2-3h · Stage: Programming Basics (Go)

## Objectives

By the end of this lesson you can:

- Write `if`/`else` chains, including the `if` statement with a short initializer.
- Choose between `switch` and `if`/`else` chains and explain Go's implicit break behavior.
- Implement all forms of `for` (classic, condition-only, infinite, range) for a given problem.
- Use `break` and `continue` correctly, including with labels, and explain what each skips.

## Programs that choose

Your programs so far ran every line, top to bottom, every time. Real programs
*choose*. The keyword is `if`; its condition must be a `bool` (the true/false
type from the variables lesson):

```go
if temperature > 30 {
	fmt.Println("hot")
} else if temperature > 15 {
	fmt.Println("mild")
} else {
	fmt.Println("cold")
}
```

Two rules that surprise people arriving from other languages:

- **No parentheses** around the condition — write `if x > 0`, not `if (x > 0)`.
- **Braces are mandatory**, even for one line, and `{` sits on the `if`'s
  line. That's syntax, not style.

Exactly one branch of a chain runs — Go checks conditions top to bottom and
takes the first that's true.

## The `if` with a short statement

You can run a small statement *before* the condition, separated by `;`:

```go
if r := n % 2; r != 0 {
	fmt.Println("odd, remainder", r)
}
```

`%` is the **remainder** operator (`n % 2` is `0` exactly when `n` is even).
The variable `r` exists only inside the `if` and its `else` — after the
closing brace it's gone. That scoping is the point: a helper used for one
decision doesn't leak into the rest of the function. Real Go code uses this
shape constantly.

## `switch`: the tidy chain

When you compare one value against several candidates, an `if`/`else` chain
gets repetitive. `switch` says it once:

```go
switch day {
case "Saturday", "Sunday":
	fmt.Println("weekend")
case "Friday":
	fmt.Println("almost")
default:
	fmt.Println("weekday")
}
```

What to notice:

- A case can list **several values**, separated by commas.
- `default` runs when nothing matched — the final `else`.
- **Go breaks implicitly.** A finished case ends the whole `switch`. C-family
  languages fall *through* to the next case unless you write `break`; Go
  inverts that — the rare explicit `fallthrough` keyword restores it.

A `switch` with no value tidies a long chain over *conditions*:

```go
switch {
case score >= 90:
	grade = "A"
case score >= 75:
	grade = "B"
default:
	grade = "C"
}
```

Rule of thumb: one value against fixed candidates → `switch`; unrelated
conditions → `if`/`else`; three or more `else if`s in a row → the
condition-less `switch` usually reads better.

## `for`: the only loop

Other languages have `while`, `do`, `foreach`… Go has exactly one loop
keyword, `for`, in four shapes. First the classic three-part form:

```go
for i := 1; i <= 5; i++ {
	fmt.Println(i)
}
```

The parts, separated by `;`: **init** runs once, before anything; the
**condition** is checked *before* every iteration — false ends the loop; the
**post** runs *after* every iteration (`i++` means `i = i + 1`). This prints
1 through 5, then moves on.

Drop the init and post and you have the second shape — a `while` loop by any
other name — repeat as long as the condition holds:

```go
for n > 1 {
	n = n / 2
	steps++
}
```

Drop the condition too and you get the third shape, the **infinite loop**:
`for { … }`. That's not a bug waiting to happen — it's the honest way to
write "loop until something inside decides we're done": servers waiting for
requests, games waiting for input. `break` exits the loop immediately:

```go
power := 1
for {
	if power > 1000 {
		break
	}
	power *= 2
}
```

## The fourth shape: `range`

To do something *n times*, Go (since 1.22) lets you range over an integer:

```go
for i := range 3 {
	fmt.Println(i)   // prints 0, then 1, then 2
}
```

Careful: `range n` yields `0` through `n-1` — n values starting at zero, not
1 through n. If you don't need the counter, `for range 3 { … }` works too.
`range` is really Go's "walk over each element" loop — it also walks
collections, which you'll meet in the arrays & slices lesson later this
stage. Counting is its simplest customer.

## Skipping and stopping: `continue`, `break`, labels

Inside any loop:

- `continue` abandons the *current iteration* and jumps to the next (in a
  classic `for`, the post statement still runs — `i` still advances).
- `break` abandons the *whole loop*.

Both act on the **innermost** loop. With nested loops you sometimes want to
move the *outer* loop along from deep inside the inner one — label the outer
loop and name it:

```go
candidates:
for n := 2; n <= limit; n++ {
	for d := 2; d < n; d++ {
		if n%d == 0 {
			continue candidates // next n, not next d
		}
	}
	fmt.Println(n, "is prime")
}
```

A plain `continue` there would only skip to the next divisor `d`; `continue
candidates` abandons the inner loop and moves to the next `n`. `break` takes
a label the same way. Labels are rare — reach for them only when nesting
forces you to.

## Exercise

Open [`exercise/`](exercise/) — a ready Go module:

- `flow.go` — seven small functions, one per control-flow form. Each has a
  `TODO` telling you which form to practice.
- `flow_test.go` — the specification. **Read it first.**
- `main.go` — a playground; `go run .` prints your functions' answers so you
  can watch them come alive as you work.

Do them in order — each is a few lines, and they get gradually meatier.

Acceptance criteria:

1. `Sign` returns `"negative"`, `"zero"`, or `"positive"` — written as an
   `if`/`else` chain.
2. `Award` maps 1/2/3 to `"gold"`/`"silver"`/`"bronze"` and everything else to
   `"none"` — written as a `switch` with a `default`.
3. `SumEvens` sums the even numbers from 1 through `limit` — a classic
   three-part `for` that skips odd numbers with `continue`.
4. `Repeat` concatenates `word` `times` times (`""` when `times <= 0`) — a
   `range`-over-integer loop.
5. `CollatzSteps` counts steps to reach 1 (even → halve it, odd → `3n+1`) — a
   condition-only `for`.
6. `FirstPowerAbove` returns the smallest power of two strictly greater than
   `limit` — an infinite `for` that exits with `break`.
7. `CountPrimes` counts the primes from 2 through `limit` — nested loops where
   a **labeled** `continue` rejects non-primes.
8. Somewhere in the file, use at least one `if` with a short statement (the
   remainder check in `SumEvens` is a natural spot).
9. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...
```

They all fail right now — that's your worklist. Green them one function at a
time. If a loop never ends, `Ctrl-C` kills the program; then look at what your
condition (or your `break`) actually checks.

## Further reading

- [A Tour of Go — Flow control](https://go.dev/tour/flowcontrol/1)
- [Effective Go — Control structures](https://go.dev/doc/effective_go#control-structures)
- [Go spec — For statements](https://go.dev/ref/spec#For_statements)
- [Go 1.22 release notes — range over int](https://go.dev/doc/go1.22#language)
