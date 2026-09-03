# Methods

> `go.basics.methods` · ~2-3h · Stage: Programming Basics (Go)

## Objectives

By the end of this lesson you can:

- Declare methods with receivers and explain how they differ from plain
  functions.
- Choose between value and pointer receivers and justify the choice for a
  given type.
- Explain why a method with a value receiver cannot mutate the caller's
  value — it edits a copy — and why mutation demands a pointer receiver.
- Explain, at an introductory level, how the method set of `T` differs from
  that of `*T`.

## From function to method

In the pointers lesson you wrote functions like this:

```go
func Heal(p *Player, amount int) { … }

Heal(&hero, 20)
```

The `*Player` parameter is the star of that show: the function exists to
operate on one player. Go lets you say so in the syntax. A **method** is a
function with one parameter promoted to a special position — the
**receiver** — written between `func` and the name:

```go
type Account struct {
	Owner   string
	Balance int // cents
}

// A plain function…
func CanAfford(a Account, amount int) bool {
	return amount <= a.Balance
}

// …and the same logic as a method.
func (a Account) CanAfford(amount int) bool {
	return amount <= a.Balance
}
```

Nothing else changed — same body, same rules. But the call site now uses the
dot you know from struct fields:

```go
acc := Account{Owner: "Ada", Balance: 100}
acc.CanAfford(50) // true — reads like a sentence about an account
```

Hold on to this: **a method is an ordinary function whose first parameter
moved in front of the name.** No hidden behavior, no magic. So why bother?

- Calls read like the domain: `acc.Withdraw(500)` versus
  `Withdraw(acc, 500)`. Small win, compounds over a whole codebase.
- Behavior travels with the data: type `acc.` in your editor and gopls lists
  everything an `Account` can do.
- Each type is its own namespace: `Account` and some future `Savings` type
  can *both* have a `Deposit` method in one package, while two plain
  functions named `Deposit` could not coexist.

## Any named type can carry methods

Receivers are not just for structs. Any type you declare with `type` — in
the same package as the method — can have them:

```go
type Celsius float64

func (c Celsius) Freezing() bool {
	return c <= 0
}

t := Celsius(-4)
t.Freezing() // true
```

The same-package rule is strict: you cannot attach methods to `int`, to
`string`, or to a type another package owns. That is exactly why thin named
types like `Celsius` (or the exercise's `Cents`) exist — declaring your own
name for `float64` or `int` buys you a place to hang behavior. Remember from
the variables lesson that such types convert explicitly: `Celsius(-4)`, not
a bare `-4` where a `Celsius` is expected.

## The copy trap

Now the central question of this lesson. Watch this method fail:

```go
func (a Account) Deposit(amount int) {
	a.Balance += amount // compiles, runs, and changes… a copy
}

acc := Account{Owner: "Ada", Balance: 100}
acc.Deposit(50)
fmt.Println(acc.Balance) // still 100
```

You already know why, because you have met this rule three times: Go passes
**every** argument by value, and the receiver is an argument. Calling
`acc.Deposit(50)` copies `acc` into `a`, field by field, exactly like the
struct assignments in the structs lesson and `bump(n int)` in the pointers
lesson. The method faithfully adds 50 to the copy, the copy dies when the
method returns, and the caller's `acc` never hears about any of it.

The fix is the same as it was for functions — pass an address:

```go
func (a *Account) Deposit(amount int) {
	a.Balance += amount // through the pointer: the caller's account
}
```

A **pointer receiver**. Inside the body nothing needed to change —
`a.Balance` auto-dereferences, as you saw with `*Player` in the pointers
lesson.

## The call site stays the same

Here is the part that surprises people. With the pointer receiver, this
still works, unchanged:

```go
acc.Deposit(50) // acc is a value, receiver is *Account — yet it compiles
```

Strictly you would have to write `(&acc).Deposit(50)`. Go finds that as
tedious as `(*p).Name` was, so the compiler inserts the `&` for you whenever
the value has an address. The sugar runs the other way too: if `p` is a
`*Account`, calling the value-receiver method `p.CanAfford(50)` quietly
becomes `(*p).CanAfford(50)`.

Consequence: you cannot tell from a call site which kind of receiver a
method has. The behavior is decided once, at the *declaration* — so that is
where you look, and that is where you choose carefully.

The sugar has one requirement: an address must exist. Some values are
homeless copies — a struct literal used directly, or a map element (the map
may shuffle its entries as it grows, so Go refuses to hand out their
addresses):

```go
Account{Balance: 100}.Deposit(50) // compile error: cannot take the address
m := map[string]Account{"ada": {Balance: 100}}
m["ada"].Deposit(50)              // compile error: map elements have no address
```

Both errors are the compiler protecting you from mutating a copy that
nobody could ever observe. (For the map case, the idiom is to store
`*Account` values in the map, or copy the element out, change it, and put
it back — the maps lesson's read-modify-write.)

## Choosing the receiver

For every method, the pointers lesson's three questions apply, plus one new
rule of thumb:

1. **Mutation** — must the method change the receiver? Pointer, no debate.
2. **Size** — is the struct large and the method called constantly? A
   pointer copies one address instead of every field.
3. **Neither?** — prefer the value. A value receiver promises "calling this
   cannot change your data", which is worth a lot when reading code.
4. **Consistency** — in real codebases, if *any* method of a type needs a
   pointer receiver, the convention is to give *all* its methods pointer
   receivers, so the type behaves one way everywhere. Today's exercise
   deliberately breaks this rule and mixes receivers on one type — so that
   you make the choice freshly for each method. Expect a tutor question
   about it.

## Method sets — first contact

Every type has a **method set**: the collection of methods that belong to
it. The rule, precisely:

- The method set of `T` contains the methods with **value** receivers.
- The method set of `*T` contains the methods with value receivers **and**
  those with pointer receivers — everything.

The asymmetry has a mechanical reason. Given a `*Account`, Go can always
reach the `Account` value — just follow the pointer — so both receiver
kinds can be served. Given a bare `Account` value, there is not always an
address to offer a pointer method: as you just saw, literals and map
elements have none. The auto-`&` sugar papers over the difference for
ordinary variables, which is why you can go weeks without noticing method
sets exist.

Why plant the flag now? Today, method sets explain this lesson's compile
errors. Later, in the Intermediate Go stage, they become load-bearing:
*interfaces* decide whether a type fits by inspecting its method set, and
value-versus-pointer receiver turns into the difference between "fits" and
"does not compile". File the rule away; it will be waiting for you.

## Exercise

Open [`exercise/`](exercise/) — a Go module with the package `bank`:

- `account.go` — an `Account` struct, a `Cents` type, and four methods.
  Three carry `TODO`s; one is fully written and *still wrong*.
- `account_test.go` — the specification. **Read it first.**

Acceptance criteria:

1. `CanAfford(amount)` reports whether the balance covers `amount` — a
   value receiver; asking a question must not change the account.
2. `Deposit(amount)` adds to the balance *and the caller sees it*. The
   starter's body is already correct and it compiles — yet the test fails.
   Fix the receiver, not the body. Negative amounts are ignored.
3. `Withdraw(amount)` subtracts and returns `true` when the account can
   afford it; for a negative or unaffordable amount it returns `false` and
   leaves the balance untouched. Let `CanAfford` do the deciding — don't
   re-derive it.
4. `Cents.Dollars()` renders `Cents(1234)` as `"$12.34"`, `Cents(5)` as
   `"$0.05"`, and `Cents(300)` as `"$3.00"` — a method on a named
   non-struct type.
5. `go test ./...` passes, and the code is `gofmt`-formatted.

Run the tests from inside the `exercise/` folder:

```sh
cd exercise
go test ./...
```

They FAIL on the starter — that's your worklist. The `Deposit` failure is
the most instructive one: the money vanished, and the test can't find it.
Figure out which copy it went into before you touch the code.

## Further reading

- [A Tour of Go — Methods](https://go.dev/tour/methods/1)
- [Effective Go — Pointers vs. Values](https://go.dev/doc/effective_go#pointers_vs_values)
- [Go spec — Method sets](https://go.dev/ref/spec#Method_sets)
- [Go wiki — CodeReviewComments: Receiver Type](https://go.dev/wiki/CodeReviewComments#receiver-type)
