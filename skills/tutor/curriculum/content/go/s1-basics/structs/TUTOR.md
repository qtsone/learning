# Tutor notes — Structs

## Where the learner is

Seven lessons into Go. They can write functions with multiple returns, loop
with `for`/`range`, and use slices and maps (including comma-ok). Structs
are their first *user-defined* type built from parts — the jump from "using
Go's types" to "modeling my own". No pointers yet: everything is value
semantics, and that's a feature here — don't reach ahead. If they ask "but
how do I change the original?", say the next lesson (pointers) answers
exactly that, and returning a changed copy is the tool for today.

## Common misconceptions

- **Promotion works in literals** — writing `Contact{Name: "Ada"}` and
  being confused by `unknown field Name`. Promotion is for *access* only;
  literals must nest: `Contact{Person: Person{Name: "Ada"}}`.
- **Embedding is inheritance** — expecting a `Contact` to be usable as a
  `Person`. It isn't; there's a real field named `Person` inside.
- **`==` checks identity** — believing two separately built contacts with
  the same data are "different objects", so not equal. Struct equality is
  field-by-field content comparison.
- **`==` failing on `Group` is a bug in their code** — the compile error
  about uncomparable types is the *language* refusing slices in `==`; the
  fix is a manual comparison, not a different literal.
- **Mutating a parameter mutates the caller's value** — writing
  `c.Name = newName` in `Rename` and worrying it "changes the original".
  It can't: the parameter is already a copy. (The reverse worry appears in
  the pointers lesson.)
- **Positional literals "work, so they're fine"** — they compile today;
  have the learner articulate the two failure modes (silent swap of
  same-typed fields, codebase-wide breakage on a new field).

## Grilling points

- "In `NewContact`, why did you have to write `Person: Person{…}` when you
  can read the field as `c.Name`?"
- "I reorder `Email` and `Phone` in the `Contact` type. Which literals in
  the codebase break, and *how loudly*, for named vs positional?"
- "Two contacts built by two different calls compare equal with `==`.
  What does that tell you about what `==` compares?"
- "Why does `SameContact` compile with `==` but `SameGroup` doesn't?
  What's the one field that ruins it? When do you find out?"
- "Could `Person` be a map key? Could `Group`? Why?" (Ties comparison to
  the maps lesson.)
- "What would happen if `Rename` didn't return anything and just assigned
  to `c.Name`?" (Nothing observable — copies. Plants the pointers seed.)

## Grading rubric

- **A** — All tests pass; `NewContact` uses a field-named literal with a
  nested `Person`; `SameContact` is a single `==`; `SameGroup` checks name,
  length, then members in order; learner can explain promotion vs literal
  nesting and predict which structs support `==`.
- **B** — Tests pass but with clumsy spots (e.g. `SameContact` compares
  field by field instead of `==`, or `SameGroup` misses the length check
  and got saved by test data ordering); explanation mostly solid.
- **C** — Tests pass only after heavy hinting, or the learner cannot say
  why `==` on `Group` fails to compile. Time-boxed remediation before
  advancing.
- **Fail** — Tests failing, or embedding/comparison are magic ("I copied
  the nested literal shape until it compiled"). Remediate, don't advance.

Note for grading: a field-by-field `SameContact` is *correct* but misses
the lesson — ask them to shrink it to one line and explain why that works.

## Remediation ladder

1. "Read the compiler/test message aloud. Which field, which function is
   it pointing at?"
2. For literal errors: "What are the exact fields of `Contact` — not the
   promoted view, the real ones? Now name each field in the literal."
3. For `SameGroup`: "You can't compare the slices in one shot. What did
   the slices lesson say about comparing element by element? Write the
   length check first — why must it come before the loop?"
4. Sketch the shape verbally — "name check, length check, then a `for
   range` comparing `a.Members[i]` to `b.Members[i]`" — and let them type
   it.

## After passing

Preview: "You noticed every change meant returning a copy. Next lesson —
pointers — is Go's answer to *sharing* a value instead of copying it."
