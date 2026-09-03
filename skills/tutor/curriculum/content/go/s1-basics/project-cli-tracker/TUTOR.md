# Tutor notes — Mini-Project: CLI Tracker

## Where the learner is

End of S1. They have every ingredient — structs, slices, maps, pointers,
methods, errors, strings, tests, files — but this is the first time nothing
is pre-chewed: an empty struct, four stub functions, and three test files.
Expect blank-page paralysis more than technical gaps. In `guided` mode,
push them to pass 1 (`go test ./tracker`, core only) and celebrate the
first green test; do not let them start with `main.go`. This is a 5-8h
project — encourage committing after each pass so progress is visible.

## Common misconceptions

- **Value receiver on `Complete`** — compiles, tests fail "task still
  open". They mutated a copy. This is *the* pointer-receiver payoff moment;
  send them back to the methods lesson if the fix is cargo-culted.
- **`List` returning the internal slice** — `TestListReturnsACopy` fails
  and they don't see why. Revisit slices lesson: a slice value shares its
  backing array; `append([]Task(nil), t.tasks...)` or `copy` makes a real
  copy.
- **`%v` instead of `%w`** — `errors.Is` fails even though the message
  looks right. The chain matters, not the text. Have them print the error
  and ask "where is the sentinel in this value?"
- **String-matching errors** — checking `err.Error() == "file does not
  exist"` for a missing file instead of `errors.Is(err, os.ErrNotExist)`.
- **`strings.Split` on `|`** — shreds titles containing the separator; the
  roundtrip test with `"read | write"` catches it. Point at `SplitN`'s
  count argument.
- **Phantom corrupt line at EOF** — splitting file content on `"\n"` gives
  a trailing `""`; if they don't skip blank lines, loading their own saved
  file fails on the "line" after the last task.
- **A `nextID` counter field that resets** — works until
  `TestAddContinuesIDsAfterLoad`. Either restore it in `Load` or derive it
  from the data (max+1); make them articulate which invariant they chose.
- **`dispatch` swallowing tracker errors** — returning
  `errors.New("no such task")` instead of passing the wrapped `ErrNotFound`
  through; `TestDispatchCompleteUnknownID` catches it.

## Grilling points

- "Walk me through `go run . complete 2` end to end: every function, every
  file touched, in order." (Load → dispatch → Atoi → Complete → Save →
  Println. This is the money question — they should own the whole path.)
- "Why do `dispatch` and the `tracker` package exist separately? What would
  the tests look like if `main` did everything?" (hello-world's
  Greeting-vs-main split, grown up.)
- "You store tasks in a slice. Where would a map of id→Task have been
  better, and what would it have cost you?" (O(1) lookup vs lost insertion
  order — connects to the maps lesson's iteration-order warning.)
- "How does `errors.Is` find `ErrCorrupt` inside `\"load tracker: line 2:
  …\"`? What did `%w` actually do?"
- "Why does `List` return a copy? Show me a bug that returning the internal
  slice would allow."
- "Two terminals run the tracker at the same time. What happens to
  `tracker.txt`?" (Last save wins, silently. No fix expected — plant the
  seed for concurrency and databases in later stages.)

## Grading rubric

- **A** — All tests pass plus a meaningful own test (criterion 8);
  layout respected (no printing inside `tracker`, no logic hoarded in
  `main`); errors wrapped with context; pointer receivers used
  deliberately; manual criterion 7 demonstrated; gofmt-clean; can answer
  the end-to-end grilling question without notes.
- **B** — Tests pass but with rough edges: duplicated logic between
  `dispatch` and `tracker`, contextless errors (`return ErrCorrupt` bare),
  an own test that only repeats a provided case; explanation mostly solid.
- **C** — Tests pass only after heavy hinting, or `main` was never actually
  run by hand, or the learner cannot explain their own `Tracker` fields.
  Pass only if time-boxed remediation lands; otherwise another iteration.
- **Fail** — Suite red, or the end-to-end walkthrough reveals the data flow
  is magic to them. The capstone gates the stage: remediate, don't advance.

## Remediation ladder

1. "Run `go test ./tracker` alone. Read the first failure top to bottom —
   what did it expect, what did it get, which function is that?"
2. "Complete says it worked, but List still shows `[ ]`. What does a method
   with a value receiver modify?" (Let them find `*Tracker` themselves.)
3. "Your Load error prints fine but `errors.Is` says no. Which verb did you
   wrap with — `%v` or `%w` — and what is the difference in the error
   *value*, not the text?"
4. Sketch the pipeline together on paper — `main → Load → dispatch → method
   → Save → Println` — then implement one arrow at a time, running the
   matching test file after each. Never type code for them.

## After passing

Preview: "S1 done — you can build and ship a real program. S2 keeps the
same Go but changes the question: not *does it work* but *how does it
behave as the data grows*. Your tracker's `Complete` walks every task to
find one id; next stage is about noticing that, measuring it, and knowing
the structures that fix it."
