# Tutor notes — Errors

## Where the learner is

Ten lessons into Go: they can write functions with multiple returns, use maps
with comma-ok, define structs, and attach methods with pointer receivers — the
exercise deliberately exercises all of that. This is their first systematic
look at *runtime* failure; until now "it failed" meant a compile error or a red
test. Expect the `if err != nil` repetition to feel wrong to them at first —
address the "why so verbose" reaction head-on, don't dismiss it.

## Common misconceptions

- **String matching instead of identity** — `err.Error() == "account not
  found"` or `strings.Contains`. Ask what happens when the message gains
  context; steer to `errors.Is`.
- **`==` on a wrapped sentinel** — works until someone wraps, then silently
  false. `errors.Is` walks the chain; `==` checks only the outermost error.
- **`%v` where `%w` belongs** — the message looks identical, so only the
  `errors.Is` tests expose it. This is the most instructive failure in the
  exercise; let them hit it before explaining.
- **Believing a non-nil error crashes the program** — conflating errors with
  exceptions (or with `panic`, which they may have seen in output). An error
  is inert data until code acts on it.
- **Using the value when `err != nil`** — e.g. trusting `Balance`'s `0` after
  an error. The zero value is a placeholder, not information.
- **Wrapping everywhere, context nowhere** — mechanical `fmt.Errorf("error:
  %w", err)` adds noise, not context. Each layer should add what only it
  knows: operation and inputs.
- **Mutating before validating** — subtracting the balance and then noticing
  the error. The "failed ops change nothing" tests catch this.

## Grilling points

- "Change one `%w` to `%v` in `Withdraw`. Which tests fail, and why does the
  printed message not change at all?"
- "`Transfer` never mentions `ErrInsufficientFunds`. How can callers of
  `Transfer` still detect it?"
- "Why is `amount must be positive` an ad-hoc error but `account not found` a
  sentinel? What would make you promote the former?"
- "Read your failing transfer message aloud, fragment by fragment: which
  function contributed each piece?"
- "Why does `Balance` return `(0, err)` instead of just `err`? What does the
  caller promise not to do with that `0`?"

## Grading rubric

- **A** — All tests pass; `%w` used exactly where an existing error gains
  context; sentinel cases built once, not re-created with `errors.New` at each
  site; messages lowercase, unpunctuated, and informative (account, amounts);
  failed operations provably mutate nothing; learner can explain `%w` vs `%v`
  and `errors.Is` vs `==` unprompted.
- **B** — Tests pass but with noisy or duplicated context (e.g. account named
  twice in one chain), inconsistent message style, or an explanation that
  wobbles on why `errors.Is` beats `==`.
- **C** — Tests pass only after heavy hinting, or the learner treats `%w` as
  magic punctuation rather than "keep the cause inside". Pass only if a
  time-boxed re-explanation lands.
- **Fail** — Tests failing, sentinel detection done by string matching, or the
  learner cannot trace a wrapped error chain. Remediate, don't advance.

## Remediation ladder

1. "Run `go test ./...` and read the first failure aloud: what did the test
   expect, what did it get, and which acceptance criterion is that?"
2. "Which function *creates* each error, and which functions only *add
   context*? Mark the one place each sentinel should appear in your code."
3. "In `Withdraw`: the comma-ok lookup from the maps lesson gives you the
   not-found branch. What are the other two guards, and in what order must
   they run so nothing is mutated on failure?"
4. Talk through `Withdraw`'s shape line by line — guard the amount, comma-ok
   lookup wrapping `ErrAccountNotFound`, compare and wrap
   `ErrInsufficientFunds`, then subtract — but let them type every character.

## After passing

Preview: "Next up: what Go text really is — strings, bytes, and runes, and why
`len("café")` isn't 4."
