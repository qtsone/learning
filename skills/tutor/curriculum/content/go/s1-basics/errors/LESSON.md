# Errors

> `go.basics.errors` · ~3-4h · Stage: Programming Basics (Go)

## Objectives

By the end of this lesson you can:

- Explain why Go treats errors as ordinary values rather than exceptions.
- Implement the check-and-return error handling pattern with `if err != nil`.
- Wrap errors with `fmt.Errorf` and `%w` so callers can inspect the chain with
  `errors.Is`.
- Define and use a sentinel error, and explain when a sentinel beats an ad-hoc
  error string.
- Add context to errors as they propagate without losing the underlying cause.

## Things go wrong — then what?

The compiler catches typos and type mistakes before your program runs. But some
failures only exist at runtime: a file is missing, a map has no such key, the
user typed `banana` where a number belonged. Every language needs an answer to
the question: how does a function tell its caller "that didn't work"?

Many languages answer with **exceptions**: a failure jumps out of the normal
control flow and flies up the call stack until something catches it. Go's
answer is more boring, on purpose: failure is **just another return value**.
You already know functions can return more than one value (the functions
lesson); a function that can fail returns its result *and* an error:

```go
func Balance(account string) (int, error)
```

`error` is a built-in type whose value describes what went wrong. Two
conventions make it work everywhere:

- The error is the **last** return value.
- `nil` means "nothing went wrong". A non-nil error carries a message you can
  print with `fmt.Println(err)`.

(Under the hood, `error` is an *interface* — the intermediate stage unpacks
that mechanism. For now: a value that carries a failure description.)

Why values? Because values obey the rules you have spent this whole stage
learning: you can return them, store them, compare them, pass them along.
There is no invisible second control flow. Reading a function shows you every
place it can fail, because every failure is right there in the return values.

## The check-and-return pattern

The moment you call something fallible, you check:

```go
balance, err := b.Balance("alice")
if err != nil {
	return 0, err // can't handle it here — pass it up to our caller
}
// from here on, balance is trustworthy
```

The rules of the pattern:

1. Check `err` immediately after the call, before doing anything else.
2. If `err != nil`, do **not** use the other return value — by convention it
   is a meaningless zero value (`0`, `""`, `nil`).
3. If this function can't fix the problem, return the error so a caller who
   *can* decides. Handle an error once, at the level that knows what to do —
   don't log it at every hop.
4. Failure branches indent and return early; the happy path flows straight
   down the left margin.

Yes, you will type `if err != nil` hundreds of times. It is Go's most mocked
and most defended idiom. The repetition *is* the feature: every possible
failure is visible at the exact line where it can happen.

## Making your own errors

Two standard-library tools create errors:

```go
errors.New("account not found")                            // fixed message
fmt.Errorf("deposit %d: amount must be positive", amount)  // formatted message
```

`errors.New` (from the `errors` package) is for constant messages;
`fmt.Errorf` is `Sprintf` for errors. Message style matters: **lowercase, no
trailing punctuation, no "error:" prefix**. Messages get chained together (you
will see how in a moment), and `Error: Deposit Failed!` reads terribly in the
middle of a chain.

## Sentinel errors

Some failures are one-off — the message is only ever read by a human, and an
ad-hoc `fmt.Errorf` at the failure site is exactly right. Other failures are
*expected, specific outcomes* that calling code wants to detect and branch on:
"not found — so create it", "insufficient funds — so show the top-up screen".

Detecting by message text is brittle:

```go
if err.Error() == "account not found" { … }   // breaks when the wording changes
```

Instead, declare the error once, at package level, and export it. This is a
**sentinel error**:

```go
var ErrAccountNotFound = errors.New("account not found")
```

The name starts with `Err` by convention. Now there is exactly one value for
this failure, and callers compare by *identity*, not by text. Reach for a
sentinel when callers need to react to that specific failure; stick with an
ad-hoc error when the message is only for humans.

## Wrapping: add context without losing the cause

An error born deep in a program is truthful but vague: `account not found` —
which account? During what operation? Each function along the way knows
something the layers below don't, so each one should add its context as the
error travels up:

```go
if err := b.Withdraw(from, amount); err != nil {
	return fmt.Errorf("transfer from %q to %q: %w", from, to, err)
}
```

The verb `%w` — *w* for wrap — formats the error into the message like `%v`
would, but also keeps the original error *inside* the new one. Stack a few
layers and the message reads like a story, outermost context first:

```
transfer from "alice" to "bob": withdraw 100 from "alice": have 50: insufficient funds
```

Each layer contributed one fragment — that is why fragments are lowercase and
unpunctuated. And because every layer used `%w`, the original sentinel is
still in there. `errors.Is` walks the whole chain:

```go
if errors.Is(err, ErrInsufficientFunds) {
	// true even though the sentinel is wrapped two layers deep
}
```

Two traps to avoid:

- `err == ErrInsufficientFunds` compares only the outermost error, so it
  breaks the moment anyone wraps. With sentinels, always use `errors.Is`.
- `%v` produces the *same message* as `%w` but throws the original error away
  — the chain is severed and `errors.Is` returns false. If the thing you are
  formatting is an error you received, wrap it with `%w`.

## Exercise

Open [`exercise/`](exercise/) — a tiny in-memory bank in `bank.go`. The
sentinels and the `Bank` type are written for you; four methods carry `TODO`s.
`bank_test.go` is the specification — read it before you code.

Acceptance criteria:

1. `Deposit(account, amount)` adds `amount` to the account, creating the
   account on its first deposit. A non-positive amount returns an error and
   must not create or change any account.
2. `Balance(account)` returns the balance. For an unknown account it returns
   an error that wraps `ErrAccountNotFound` and names the account in its
   message.
3. `Withdraw(account, amount)` subtracts on success. Failures, each leaving
   balances untouched: non-positive amount → an error; unknown account → an
   error wrapping `ErrAccountNotFound`; `amount` greater than the balance → an
   error wrapping `ErrInsufficientFunds` whose message includes the account,
   the amount, and the current balance.
4. `Transfer(from, to, amount)` withdraws from `from`, deposits to `to`. On
   failure it adds its own context (the word `transfer` and both account
   names) — and `errors.Is` must still find the underlying sentinel through
   every layer. A failed transfer changes no balances.
5. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They fail on the starter — read each failure message and make it green.

## Further reading

- [Go blog — Error handling and Go](https://go.dev/blog/error-handling-and-go)
- [Go blog — Working with Errors in Go 1.13](https://go.dev/blog/go1.13-errors)
- [pkg.go.dev — errors package](https://pkg.go.dev/errors)
- [Effective Go — Errors](https://go.dev/doc/effective_go#errors)
