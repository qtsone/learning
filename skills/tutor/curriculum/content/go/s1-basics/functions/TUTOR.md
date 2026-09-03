# Tutor notes — Functions

## Where the learner is

Fourth lesson of S1. They can declare variables, use the basic types and
constants, and steer with `if`/`switch`/`for`. They completed one function
(`Greeting`) in hello-world, but this is their first time *designing*
functions: first multiple returns, first `error` value, first sighting of a
slice. The slice preview here is deliberately minimal (literal + range +
spread) — resist expanding it; the arrays-slices lesson owns that ground.
Likewise keep errors at `errors.New` and `err != nil`; wrapping and sentinels
belong to the errors lesson.

## Common misconceptions

- **`error` feels like special syntax** — it's an ordinary value; `nil` means
  "no error". If they've heard of exceptions, contrast: Go failures travel
  through return values and plain `if`, nothing invisible happens.
- **Using results before checking the error** — reading `share` without
  `if err != nil` first. Ask what `share` holds when the split failed.
- **`Sum(prices)` without the spread** — compile error `cannot use prices
  (variable of type []int) as int value`. Have them read the message aloud;
  it says exactly what's wrong.
- **Re-adding prices (or re-checking people) inside `SplitBill`** — the tests
  still pass, but it misses the decomposition objective. Catch it in review;
  see the rubric.
- **Naked returns everywhere** after meeting named results — steer to "names
  for documentation, explicit values on `return`".
- **Multiple returns imagined as one tuple value** — they can't be stored or
  indexed as a unit; each result must be assigned or discarded with `_`.
- **Magic failure values** — proposing `-1` instead of an error. Ask how a
  caller would ever notice, and what happens when `-1` is a legal answer.

## Grilling points

- "Walk me through `SplitBill(0, prices...)` value by value." (`Sum` still
  runs and returns a total; `Split` refuses; the direct `return` forwards the
  error untouched to the caller.)
- "Why zero values *plus* an error on failure, instead of just `-1`?"
- "Delete the `...` in `Sum(prices...)` and compile. What does the error say,
  and why is the compiler right to refuse?"
- "`prices ...int` in the signature and `prices...` at a call site — same
  three dots, two meanings. Explain both." (Gather into a slice / spread a
  slice.)
- "When would you name your return values but still write explicit returns?"
- "`FormatCents` was given to you. What is its one job, and why doesn't that
  code live inside `main`?"

## Grading rubric

- **A** — All tests pass; `SplitBill` composes `Sum` and `Split` with no
  duplicated loop or guard; the error message says what's wrong; `main`
  checks the error before using results; gofmt-clean; learner explains both
  meanings of `...` and the (result, error) shape unprompted.
- **B** — Tests pass, but `SplitBill` re-adds prices or re-checks
  `people < 1`, or `main` uses the results before (or without) the error
  check; explanation mostly solid.
- **C** — Tests pass only after heavy hinting, or the learner cannot say what
  each name in `share, remainder, err := …` holds. Time-boxed remediation
  before advancing.
- **Fail** — Tests failing, or a copied solution the learner cannot walk
  through line by line. Remediate, don't advance.

## Remediation ladder

1. "Run `go test ./...` and read only the first failure. Which function is it
   about, and what did it expect versus get?"
2. "For `Sum`: what must come back when there are no prices at all? What
   value does your total start at, and what happens to it each time round the
   loop?"
3. "For `Split`: which check must happen *before* any division? When that
   check fires, what does the convention say to return in the other two
   spots?"
4. "For `SplitBill`, say the plan out loud in one sentence: total the prices,
   then split the total. You already own a function for each half — connect
   them, and remember a slice needs `...` when you pass it on."

## After passing

Preview: "Next lesson zooms out from one file: packages and modules — where
functions live, why some names are Capitalized, and what `go.mod` has been
doing for you all along."
