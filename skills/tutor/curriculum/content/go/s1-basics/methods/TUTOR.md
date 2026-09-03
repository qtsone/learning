# Tutor notes — Methods

## Where the learner is

Tenth lesson of the stage, straight after pointers — the receiver decision
is pass-by-value plus pointers wearing new syntax, so lean on that: every
"why" here should route back to `bump(n int)` and struct copying. They have
NOT met errors or interfaces; keep method sets at the "explains today's
compile errors, becomes load-bearing with interfaces in S3" depth and go no
further. The broken `Deposit` in the starter is deliberate — let them sit
with the failing test before hinting.

## Common misconceptions

- **"Methods are a special kind of function with hidden powers"** — the
  receiver is an ordinary parameter in a special position. Demystify by
  having them rewrite one method as a plain function.
- **"The Deposit bug is in the body"** — they rewrite `a.Balance += amount`
  five different ways instead of looking at the receiver. The body was
  always right; that's the whole point of the exercise.
- **"The call site decides"** — believing `(&acc).Deposit(50)` and
  `acc.Deposit(50)` behave differently. Behavior is fixed at the
  declaration; the compiler's auto-`&` just makes calls look alike.
- **"Reading a field needs a pointer receiver"** — value receivers read
  fine; pointers are for mutation and size.
- **"Method set = whatever compiles on my variable"** — the auto-`&` sugar
  on addressable variables blurs the line; the T vs *T distinction only
  shows on literals and map elements.
- **Passing an `int` where `Cents` is expected** — named types don't
  convert implicitly; they need `Cents(1234)` (variables lesson).

## Grilling points

- "Rewrite `Withdraw` as a plain function taking `*Account`. What changed?
  So what exactly does the method syntax buy you?"
- "In the broken `Deposit`, where precisely do the 50 cents go? Trace the
  copy from call to return."
- "`acc.Withdraw(10)` with `acc` an `Account` value — what does the
  compiler secretly write, and what must be true of `acc` for it to work?"
- "Why does `Account{Balance: 100}.Deposit(50)` refuse to compile while
  `CanAfford` on the same literal is fine?"
- "List `Account`'s method set. Now `*Account`'s. Which direction comes for
  free, and why?"
- "This exercise mixes value and pointer receivers on one type. Would you
  ship `Account` that way? What does the convention say?"

## Grading rubric

- **A** — All tests pass; `Deposit` fixed by changing only the receiver;
  `Withdraw` delegates to `CanAfford`; gofmt-clean; learner justifies the
  receiver choice for every method and states the T vs *T method-set rule.
- **B** — Tests pass, but `Withdraw` re-derives the affordability check, or
  receiver justifications are shaky ("pointer because it worked").
- **C** — Tests pass only after being told outright to change the receiver,
  or method sets remain a mystery. Time-boxed remediation, else another
  iteration.
- **Fail** — Tests failing, or the learner cannot explain why the pointer
  receiver fixed `Deposit`. Remediate, don't advance.

## Remediation ladder

1. "Read the failing `Deposit` test message aloud. The body adds 50 —
   where did it add it?"
2. "Inside the method, what is `a` exactly — the caller's `acc`, or
   something else? What did `bump(n int)` do in the pointers lesson?"
3. "You already know a parameter kind that lets a function change the
   caller's value. A receiver can be that kind too — what would the
   signature look like?"
4. Spell it out verbally — receiver becomes `(a *Account)`, body and call
   site stay untouched — and let them type it and re-run the tests.

## After passing

Preview: "Next: errors. `Withdraw`'s bare `false` says something failed but
not what — the errors lesson upgrades that answer to a value that explains
itself."
