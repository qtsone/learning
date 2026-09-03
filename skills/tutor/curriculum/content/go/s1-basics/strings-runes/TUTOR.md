# Tutor notes — Strings, Bytes & Runes

## Where the learner is

Twelfth lesson of S1, straight after errors. They have used strings since
hello-world without ever looking inside one; this is their first contact with
encodings, and `len("héllo") == 6` should genuinely surprise them — let it,
then explain. They are fluent with slices, `range`, maps, and functions, so
the mechanics (`[]rune` conversion, swap loops, `append` in `CleanFields`)
reuse muscles they already have; the new material is the byte/rune model and
two stdlib tools (`strings`, `strings.Builder`).

## Common misconceptions

- **"`len(s)` counts characters"** — it counts bytes. Non-ASCII text is where
  the two diverge; every test table here includes such a case on purpose.
- **"`s[i]` is the i-th character"** — it is the i-th *byte*, and for a
  multi-byte rune that's a meaningless fragment (`"héllo"[1]` is 195). The
  `Initials` accented case exposes exactly this via `word[0]`.
- **"`range` steps the index by 1"** — the index is a byte offset and jumps
  past multi-byte runes (0, 1, 3, 4, 5 for `"héllo"`). Learners who compute
  positions from it get confused; runes-per-step, bytes-per-index.
- **Reversing bytes instead of runes** — passes the ASCII cases, garbles
  `"héllo"` and `"日本語"`. The test failure message points at this.
- **`string(n)` for numbers** — expecting `string(65)` to be `"65"`; it is
  `"A"` (and vet flags the un-parenthesized form). `fmt.Sprint` for digits.
- **"`Fields` and `Split(s, " ")` are the same"** — `Split` keeps empties and
  cuts on single spaces only; `Fields` collapses any whitespace run. `Slug`'s
  "extra spaces" case fails with `Split`.
- **"`+=` in a loop is fine / Builder is just style"** — each `+=` copies the
  whole accumulated string; the cost is quadratic. Builder appends into a
  growing buffer. Tests can't catch this — grill it and check the `Initials`
  code.
- **Expecting `strings` functions to mutate** — strings are immutable; calling
  `strings.ToLower(s)` without using the return value does nothing.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go deeper):

- "`len("héllo")` is 6 but `CountRunes` says 5 — where is the extra byte, and
  what is it half of?"
- "Walk me through `for i, r := range "héllo"` — what values does `i` take,
  and why does it skip 2?"
- "Your `Reverse` converts to `[]rune` first. What exactly breaks if you swap
  the bytes instead — and which test case proves it?"
- "In `Initials`, why is `word[0]` wrong even though it compiles and passes
  the ASCII cases?"
- "Suppose `Initials` used `result += …` instead of a Builder. What happens on
  each `+=`, and why does the total work grow faster than the input?"
- "When would you pick `strings.Join` over a Builder?" (Already have a slice —
  Join is the whole loop, correctly separated.)

## Grading rubric

- **A** — All tests pass; `CountRunes` ranges over the string (or uses
  `utf8.RuneCountInString`), never `len`; `Reverse` swaps a `[]rune` in place;
  `CleanFields` is a `Split`/`TrimSpace`/`append` loop; `Slug` composes
  `ToLower`/`Fields`/`Join`; `Initials` uses a `strings.Builder` and takes the
  first *rune* of each word; gofmt-clean; learner explains bytes vs runes and
  the quadratic `+=` cost unprompted.
- **B** — Tests pass but with clunky spots (`+=` where a Builder belongs, a
  fresh `[]rune` conversion inside every loop iteration, `Split(s, " ")`
  patched up instead of `Fields`) or formatting misses; explanations mostly
  solid.
- **C** — Tests pass only after heavy hinting, or the learner cannot say what
  a rune is or why indexing gave them 195. Pass only if time-boxed remediation
  lands; else another iteration.
- **Fail** — Tests failing, or a solution the learner cannot walk through.
  Remediate, don't advance.

## Remediation ladder

1. "Run `go test ./...` and read the first failure aloud — which case failed,
   what did it get, what did it want? Notice it's a non-ASCII case."
2. "Print `len("héllo")` and count the letters on your fingers. What unit is
   `len` counting? What does each step of `range` hand you instead?"
3. "For `Reverse`: you need a slice where each element is one whole character,
   whatever its byte count. Which conversion from the lesson gives you that?"
4. Sketch the shapes verbally — `range` decodes runes; `[]rune`, swap `i`/`j`
   ends inward; `Split` then `TrimSpace` then keep non-empties; Builder plus
   `WriteString` in the loop — but let them type every line.

## After passing

Preview: "Next: testing basics — you've been *reading* table-driven tests for
eleven lessons; next you finally write your own, `t.Run` and all."
