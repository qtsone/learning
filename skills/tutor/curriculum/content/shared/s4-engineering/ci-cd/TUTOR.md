# Tutor notes — CI/CD

## Where the learner is

Deep into S4: they write tests by habit (tdd), debug from logs methodically
(debugging), and know exit codes and git from S0. They have almost certainly
*seen* a green checkmark on GitHub; this lesson replaces "CI is a thing GitHub
does" with "CI is my own commands on a trigger". Next up are code-review and
documentation, both of which lean on the pipeline as the mechanical reviewer —
land that framing here.

## Common misconceptions

- **"CI is magic infrastructure"** — it is a fresh machine running the same
  shell commands they run locally. If a run is mysterious, have them run the
  step's exact command at their own prompt.
- **"CI tests better than my machine"** — same tests, same results. The value
  is *fresh environment* + *every push, no exceptions*, not better testing.
- **"CD means deploys happen automatically"** — that's continuous
  *deployment*. Continuous *delivery* keeps a human on the release button;
  every green main build is deployable, not deployed.
- **"A green step means the tool was happy"** — a step passes on exit code 0,
  nothing else. `gofmt -l` exits 0 even when it lists offenders; a bare
  `gofmt -l .` step is a gate that can never fail. This trap is in the
  exercise on purpose.
- **"Fix the red by removing the check"** — deleting the failing test or the
  vet-flagged function makes the light green and the code worse. check.sh
  guards against it, but probe whether they understand *why* it's wrong.
- **"go test caught the vet bug, so the vet gate is redundant"** — sharp
  learners notice `go test` also flags the format-verb bug: `go test` runs a
  small high-confidence subset of vet checks. The full `go vet ./...` gate
  covers analyzers the test subset skips — separate gates, separate jobs.

## Grilling points

- "Your workflow is green. What exactly do you now know about the repo — and
  what do you *not* know?" (Checks pass on a fresh machine; says nothing about
  uncovered behavior.)
- "Why does the runner need `actions/checkout` at all?" (Fresh machine is
  empty — nothing from any laptop leaks in; that's the feature.)
- "Walk me through what happens, step by step, from `git push` to the red X."
- "Why run gofmt and vet *before* the tests?" (Cheapest feedback first.)
- "Your pipeline is red but everything passes on your machine. Name three
  causes." (Uncommitted file, tool/version mismatch, local env var or state
  the workflow never sets.)
- "When would you *not* want continuous deployment?" (Test suite not trusted
  as a release decision, regulated environments, coordinated releases.)

## Grading rubric

- **A** — check.sh fully green; workflow is clean YAML with pinned action
  versions and gates ordered cheapest-first; learner explains the `test -z
  "$(gofmt -l .)"` idiom, both seeded defects, and the integration/delivery/
  deployment ladder unprompted.
- **B** — check.sh green; workflow works but shows copy-paste artifacts
  (unexplained keys, tests before lint with no rationale) or the vet fix was
  found by trial and error; CI-vs-CD explanation needs one prompt.
- **C** — green only after heavy hinting, or can't explain why the gofmt gate
  needs the `test -z` wrapper, or treats the runner as magic. Time-boxed
  remediation on the "robot runs my commands" model; else another iteration.
- **Fail** — check.sh red; or tests/`Describe` were deleted or weakened to
  pass; or the learner cannot connect a red step to its log. Remediate.

## Remediation ladder

1. "Run `bash ./check.sh` and read only the first FAIL line. What does it say
   to fix?"
2. Workflow shape trouble: "Open the anatomy section of LESSON.md. Which of
   the four parts — trigger, job, runner, steps — is your file missing?"
   YAML errors are almost always indentation.
3. Red gates: "Be the runner. Run `go vet ./...` yourself — which function,
   which verb, what type is the argument? Now `go test ./...` — the log names
   the inputs and both numbers. `Mean([2 4 6])` returned 6: what divisor
   produces 6 from a sum of 12?"
4. Walk the fixes verbally — `%s` should be `%d` for the int in `Describe`;
   the divisor is `len(values)`, not `len(values)-1` — but let them edit and
   re-run the gates themselves.

## Reference solution

`.github/workflows/ci.yml`:

```yaml
name: ci

on:
  push:
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: "go.mod"
      - name: Formatting
        run: test -z "$(gofmt -l .)"
      - name: Vet
        run: go vet ./...
      - name: Test
        run: go test ./...
```

Code fixes in `stats.go`:

- `Mean`: divisor `float64(len(values)-1)` → `float64(len(values))`.
- `Describe`: `fmt.Sprintf("n=%s mean=%g", …)` → `"n=%d mean=%g"` (the vet
  finding: `%s` with an `int` argument).

## After passing

Preview: "Your pipeline is now the first, tireless reviewer of every change —
next lesson adds the human one: code review, and how to give and take it
well."
