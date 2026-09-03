# Tutor notes — Pointers

## Where the learner is

Ninth lesson of S1. They can write functions, use slices, maps, and structs,
and have read table-driven tests since hello-world. This is their first
explicit contact with memory addresses, and "pointers" carries scary folklore
from C. Defuse it early: Go pointers have no arithmetic, no manual free, and
the compiler plus GC remove the classic footguns. The mental model to install
is boxes-and-addresses: a pointer *holds* an address, `*` *follows* it.

## Common misconceptions

- **Conflating `p`, `*p`, and `&x`** — saying "`*p` is the pointer". Have them
  narrate a three-line trace out loud ("p holds the address of x; *p is the
  value in x's box"). Until this is fluent, nothing else lands.
- **"Structs are passed by reference"** — no; Go copies *every* argument.
  Pointers don't suspend pass-by-value: the *address* is what gets copied.
- **"Slices/maps need pointers too"** — they already behave reference-like for
  their contents; `*[]int` is almost always a smell. Connect back to the
  slice-header picture from arrays-slices.
- **C intuition about dangling pointers** — fear that `return &pl` from
  `NewPlayer` is a bug. In Go it's the standard constructor idiom; escape
  analysis moves the value to the heap and the GC owns its lifetime.
- **Thinking they need `(*p).HP`** — they don't; `p.HP` auto-dereferences.
  Conversely, some write `*p.HP`, which doesn't parse the way they hope.
- **Guarding after the fact** — dereferencing first, nil-checking later, or
  believing the compiler will catch nil dereferences (it can't; it's runtime).

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go deeper):

- "`x := 5; p := &x; *p = 10` — what is `x` now, and how many int boxes exist
  in that snippet?" (One box, two names.)
- "Change `Heal` to take `Player` instead of `*Player` and rerun the tests.
  What fails and why?" — make them actually do it and read the failure.
- "In `Swap`, the parameters `a` and `b` are themselves copies. Why does it
  still work?" (Copied addresses point at the same boxes.)
- "What's the difference between a nil `*Player` and `&Player{}`?" (Nothing
  vs. a real, zero-valued Player — absent vs. zero, like maps' comma-ok.)
- "When would you deliberately pass a value even though a pointer would work?"
  (No mutation needed, small type, can't-be-nil safety.)

## Grading rubric

- **A** — All tests pass; `Swap` uses tuple assignment (or a clean temp
  variable); nil guards are first-line early returns, not nested pyramids;
  `NewPlayer` uses `&Player{…}` or `&pl`; gofmt-clean; learner can trace
  holds-vs-points-to and justify pointer-vs-value for each function.
- **B** — Tests pass but with clumsiness (e.g. `(*p).HP` everywhere, redundant
  nil checks, `new(Player)` followed by field-by-field assignment) or one
  shaky explanation; solid on the core model.
- **C** — Tests pass only after heavy hinting, or learner cannot explain why
  `Heal` needs a pointer, or believes Go passes structs by reference. Pass
  only if time-boxed remediation lands; else another iteration.
- **Fail** — Tests failing, a nil guard added by trial-and-error without being
  able to say what panicked, or solution copied without explanation.
  Remediate, don't advance.

## Remediation ladder

1. "Read the failing test aloud. Does it check a *returned* value, or a
   variable the function was supposed to *change*?"
2. "Draw two boxes for `a` and `b`, then draw what `Swap` receives. Where do
   the arrows point?"
3. "Your test panicked with `nil pointer dereference`. Which line follows a
   pointer? What must be true about that pointer for the line to be safe, and
   which Go statement can check it first?"
4. Walk one function's shape verbally — for `Heal`: "guard nil and return
   early; add through `p.HP`; then an `if` to clamp at `p.MaxHP`" — but let
   them type every character.

## After passing

Preview: "Next lesson attaches functions to types — methods. The value-versus-
pointer choice you just practiced becomes the value-versus-pointer *receiver*
choice, and `Heal(p, n)` turns into `p.Heal(n)`."
