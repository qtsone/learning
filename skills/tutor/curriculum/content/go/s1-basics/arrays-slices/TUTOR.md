# Tutor notes — Arrays & Slices

## Where the learner is

Sixth lesson of the stage: they can write functions with parameters and
returns, use `if`/`for`, and organize code in a module — but this is their
first data structure and the first genuinely tricky mental model in Go
(headers, aliasing, capacity). Expect the code to come easier than the
explanations; budget quiz time for len/cap prediction drills and don't let a
green test suite end the conversation.

## Common misconceptions

- **"A slice contains its elements"** — it's a three-field header (pointer,
  len, cap) pointing into a backing array. Until this clicks, aliasing looks
  like magic. Have them draw the header next to the array.
- **"Slicing copies data"** — `s[1:4]` allocates nothing; it's a new window
  onto the same array. The flip side: **"assigning copies data"** — `dst :=
  src` copies only the header.
- **`cap` confusion** — guessing `cap(s[lo:hi])` is `hi - lo`. It's
  `cap(s) - lo`: capacity runs from the pointer to the end of the array.
- **"append always reallocates" (or never does)** — it reallocates exactly
  when `len == cap`. The visible symptom of half-understanding this is
  `t := append(s, v)` followed by using both `t` and `s`.
- **The remove one-liner** — `append(s[:i], s[i+1:]...)` shifts elements
  inside the input's backing array. The spare-capacity test cases exist to
  catch it (and the equivalent Insert trick); expect "why did my input
  change?" questions — that moment is the whole lesson.
- **`copy` into an unsized destination** — `var dst []int; copy(dst, src)`
  copies zero elements with no error. Destination length must be set first.
- **Off-by-one at the boundaries** — `Insert` at `i == len(s)` is legal
  (append); `Remove` at `i == len(s)` is not. Half-open ranges are the cure,
  not the cause.

## Grilling points

Quiz.json has the core set; these go deeper:

- "`s := make([]int, 3, 5); t := s[1:3]` — give me len and cap of both.
  Now `t[0] = 9`: what does `s` look like? Now `t = append(t, 9)` twice —
  which append touched memory `s` can see, and which allocated?"
- "Why must `append`'s result be reassigned? What exactly goes stale if you
  keep the old header around?"
- "Your `Remove` builds a fresh slice. What's the famous one-line idiom, and
  in what situation is it actually the right choice?" (When the caller owns
  the slice and wants in-place efficiency — contracts matter.)
- "In `Clone`, why `make` with `len(src)` and not `make([]int, 0)`? What
  would `copy` do in the second case?"
- "Where does the memory for `append` on a nil slice come from?"

## Grading rubric

- **A** — All tests pass; `Clone` uses make-then-`copy`; `Insert`/`Remove`
  build fresh slices without touching the input; learner predicts len/cap
  correctly in a live drill and explains the reassignment rule and aliasing
  unprompted; gofmt-clean.
- **B** — Tests pass but code is clumsy (element-by-element loops where
  `copy` would do, growing with repeated `append` from zero cap when the
  final size is known); len/cap predictions shaky but self-corrected.
- **C** — Tests pass only after heavy hinting, or questioning reveals a
  "slice = array with the data inside" model. Time-boxed remediation on the
  header diagram; pass only if the drill lands afterwards.
- **Fail** — Tests failing, inputs mutated, or the learner cannot explain why
  `Clone`'s `copy` makes the two slices independent. Remediate, don't advance.

## Remediation ladder

1. "Read the failing test name and message aloud — which function, what did
   it get, what did it want?"
2. "For a 'modified its input' failure: which line of your function writes
   into memory the input can see? Where do the elements of `s[:i]` live?"
3. "Draw the three-field header for the input and for every slice your
   function creates. Do any two pointers point into the same array?"
4. Talk through the `Insert` shape — `make` with cap `len(s)+1`, then append
   the three parts in order — and let them type it and adapt it for `Remove`.

## After passing

Preview: "Slices are values in a row; next lesson adds the other everyday
collection — maps, values you look up by key."
