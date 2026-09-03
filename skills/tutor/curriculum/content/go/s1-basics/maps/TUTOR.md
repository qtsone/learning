# Tutor notes — Maps

## Where the learner is

Seventh lesson of S1. They can write functions with multiple returns, use all
control-flow forms, build and range over slices, and organize code into
packages. Maps are their first key-value structure — and `Take` is the first
time they *see* a function mutate its argument (pointers come three lessons
later). Expect "why did the caller's map change?" — answer with "a map value
is a handle to shared storage" and defer the machinery to the pointers lesson.

## Common misconceptions

- **"Reading a missing key errors"** — it silently returns the value type's
  zero value. Conversely, some think `counts[w]++` needs an existence check
  first; it doesn't, and the check-free version is the idiom.
- **"A `var`-declared map works like `make`"** — a nil map reads fine but
  panics on write. If they hit `assignment to entry in nil map`, this is it.
- **Comma-ok misread** — believing `ok` means "the value is non-zero" rather
  than "the key exists". `Describe`'s `"out of stock"` case exposes this.
- **Expecting stable iteration order** — insertion order or sorted order.
  Order is deliberately randomized; stable output requires sorting keys.
- **`delete` vs setting zero** — `pantry["eggs"] = 0` keeps the entry;
  `delete` removes it. The `Take`-to-zero test distinguishes them.
- **Comparing maps with `==`** — compile error (except against `nil`); tests
  use `maps.Equal`.
- **Mutating before checking** in `Take` — subtracting first, then noticing
  there wasn't enough, leaving the pantry corrupted. The "untouched" tests
  catch it.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go deeper):

- "`Count` never checks whether the word is already in the map. Why does
  `counts[w]++` work the very first time a word appears?"
- "You write `var pantry map[string]int` and then `pantry["flour"] = 1`.
  What happens? What if you only *read* `pantry["flour"]`?"
- "`Take` doesn't return the map — how do its changes reach the caller?"
- "Why does Go *randomize* iteration order instead of merely leaving it
  unspecified?" (Hidden order-dependencies should break loudly and early.)
- "If `NewSet` returned `map[string]bool`, would `Has` still work? Then what
  is the argument for `struct{}`?"

## Grading rubric

- **A** — All tests pass; `Count` uses `counts[w]++` with no existence check;
  `Describe` does a single comma-ok lookup; `Take` checks before mutating and
  deletes the entry at zero; code is gofmt-clean; learner can explain
  randomized order, nil-map behavior, and the empty-struct choice unprompted.
- **B** — Tests pass but with clunky spots (double lookups, rebuilding a map
  where mutation would do, `if`/`else` ladders where comma-ok plus `switch`
  reads better) or formatting misses; explanations mostly solid.
- **C** — Tests pass only after heavy hinting, or the learner cannot explain
  comma-ok or why the caller sees `Take`'s changes. Pass only if time-boxed
  remediation lands; else another iteration.
- **Fail** — Tests failing, or a solution the learner cannot walk through.
  Remediate, don't advance.

## Remediation ladder

1. "Run `go test ./...` and read the first failure aloud — which function,
   what did it get, what did it want?"
2. "For `Describe`: how many distinct situations must you tell apart, and
   which single map operation distinguishes the first two?"
3. "For `SortedItems`: you can't sort a map — what *can* you sort, and how do
   you get the map's keys into one?"
4. Sketch the shapes verbally — comma-ok gives value *and* presence; key-only
   `range` collects keys; `slices.Sort` orders them — but let them type every
   line.

## After passing

Preview: "Next: structs — you've used `struct{}` as an empty placeholder; now
you'll define real ones with fields, and a pantry entry can become a proper
type instead of a bare int."
