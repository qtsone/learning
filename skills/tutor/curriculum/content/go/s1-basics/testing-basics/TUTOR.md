# Tutor notes — Testing Basics

## Where the learner is

Twelve lessons in: comfortable with slices, maps, structs, functions, errors,
and strings/runes. They have *consumed* `go test` output in every exercise but
have never authored a test. The whole lesson is a perspective flip — lean on
that framing ("you've read a dozen test files; now write one"). The planted
bug is in `Average`: `float64(sum / len(nums))` divides ints before
converting, so any case with a remainder (e.g. `{1, 2}` → wants 1.5, gets 1)
exposes it, while evenly-dividing cases pass. Do not reveal the location or
the mechanism unprompted — the hunt is the lesson.

## Common misconceptions

- **"I need an assert library / test framework"** — Go tests are plain Go:
  call, `if`, `t.Errorf`. If they go hunting for `assertEquals`, redirect to
  the got/want idiom.
- **Name convention traps** — `Testlongest` (lowercase after `Test`) or a
  missing `t *testing.T` parameter means the function is silently skipped,
  and `go test` happily reports `ok`. Teach them to confirm with
  `go test -v` that every test they wrote actually ran.
- **`t.Fatal` vs `t.Error` confusion** — `Fatalf` in a bare loop over cases
  abandons the remaining rows; inside `t.Run` it only kills that subtest.
  Also the inverse: using `Errorf` after a failed error check, then
  dereferencing a meaningless value.
- **"Green means correct"** — a vacuous test (stub deleted, no assertions)
  passes. So does a test whose cases can't distinguish right from wrong
  (`Average({2, 4})`). Green only means "no test complained".
- **Fixing the test to match the code** — when their uneven-division case
  fails, some learners "correct" `want` to the truncated value. The doc
  comment is the contract; the code is wrong, not the arithmetic.
- **`-run` quoting** — subtest names with spaces become underscores in
  patterns (`TestAverage/empty_slice`), and the pattern should be quoted so
  the shell doesn't mangle it.
- **Coverage as a goal** — chasing 100% by adding cases that execute lines
  without checking anything. Coverage locates unreached code; it never
  certifies reached code.

## Grilling points

- "Your `Average` table has `{2, 4, 6}` and `{1, 2}`. One of those caught the
  bug and one couldn't have. Which, and why?" (Integer division only
  truncates when there's a remainder — case *selection* is the skill.)
- "Rename `TestLongest` to `Testlongest` and run `go test`. What happens, and
  how would you notice in a bigger suite?"
- "In your `TestAverage` subtest, why `t.Fatalf` for the unexpected error but
  `t.Errorf` for a wrong value?"
- "You're at 100% coverage of `stats.go`. Name a bug those same tests would
  still miss." (Any wrong-output bug on inputs outside the table — coverage
  measures execution, not verification.)
- "Show me the exact command to run only the empty-slice case of
  `TestAverage`." (`go test -run 'TestAverage/empty_slice' -v`.)
- "Why does the table pattern beat three copies of call-and-if?" (Cases as
  data; one checking site; one-line cost per new case.)

## Grading rubric

- **A** — All three tests written to spec (plain / table-driven / table with
  error cases); subtests well named; the bug found via their own failing test
  and fixed in `stats.go` with the pre-division conversion; `-run` and
  `-cover` used and explained, including why 100% coverage didn't imply
  correctness.
- **B** — Tests pass and the bug is fixed, but with rough edges: weak failure
  messages (missing got or want), a required case missing, `Fatalf`/`Errorf`
  used interchangeably without reasoning, or `-run`/coverage fumbled when
  asked.
- **C** — Bug found only after being pointed toward `Average`, or tables
  written after heavy structural hinting; explanations shaky. Pass only if
  remediation lands; otherwise iterate.
- **Fail** — Vacuous tests (stubs deleted, nothing asserted), the test edited
  to accept the truncated average, or inability to explain what makes a
  function a test. Green CI with hollow tests is an automatic fail — read
  their test file, not just the verdict.

Note: exact `==` on the expected floats here (1.5, 4, 7) is fine — all are
exactly representable. If a learner raises float-comparison worries, credit
the instinct, note epsilon comparison is a later topic, and move on.

## Remediation ladder

1. "Run `go test -v` and read the output top to bottom. Which tests ran,
   which subtests failed, and what did the failure message say was got and
   want?"
2. "For the failing average: compute `Average([]int{1, 2})` by hand. Now read
   the function's last line. What is the type of `sum`, of `len(nums)`, and
   of `sum / len(nums)`?"
3. "Remember integer division from the variables lesson: `3 / 2` is `1`. The
   `float64(...)` wraps the division — does the conversion happen before or
   after the remainder is thrown away?"
4. Walk the fix verbally — convert *both* operands to `float64`, *then*
   divide — and let them type it and rerun the suite themselves.

## After passing

Preview: "Next lesson your programs finally touch the disk — reading and
writing files with `os` and `bufio`, and `defer` for cleaning up after
yourself."
