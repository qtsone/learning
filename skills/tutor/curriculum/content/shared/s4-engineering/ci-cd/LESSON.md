# CI/CD

> `shared.eng.ci-cd` · ~2-3h · Stage: Engineering Practice

## Objectives

By the end of this lesson you can:

- Explain what continuous integration catches and why merge-time automation
  beats manual pre-release testing.
- Write a GitHub Actions workflow that runs tests and lint on every push and
  pull request.
- Explain the difference between continuous integration, delivery, and
  deployment.
- Add a quality gate (formatting or vet/lint) that fails the pipeline and
  demonstrate it catching a violation.
- Diagnose a failing pipeline run from its logs and fix the underlying cause.

## The integration problem

You already have the habits that make code trustworthy: tests from the TDD
lesson, a disciplined diagnosis loop from the debugging lesson. But habits
live in one head. The moment two people — or you-today and you-in-three-weeks
— share a repository, a new failure mode appears: each change looks fine on
its author's machine, and the *combination* is broken. Teams used to discover
this the worst possible way, in a manual "integration phase" before a release:
weeks of merging long-lived branches and fixing collisions between changes
that had silently diverged for months.

**Continuous integration (CI)** is the counter-move: merge small changes into
the shared main branch frequently — daily or better — and have a machine run
the full check suite on *every* push and *every* proposed merge. What it
catches is precisely the class of bugs manual testing misses:

- "Works on my machine" — code that depends on a file never committed, a tool
  installed only locally, an environment variable only you have set.
- Combination breakage — your change and a teammate's are each fine alone and
  wrong together; running the whole suite on the merged result finds it.
- Discipline decay — humans skip the checks when tired or rushed. A machine
  runs every check, every time, and posts the result where everyone sees it.

The economics are about feedback distance. A bug caught minutes after the
push that introduced it is trivial to place — the diff is small and fresh in
your mind. The same bug caught during a pre-release test pass weeks later
hides among hundreds of commits. Merge-time automation beats manual
pre-release testing not because machines test better, but because they test
*immediately, on everything, without being asked*.

## A pipeline is just your commands, run by a robot

Strip away the vendor branding and a CI **pipeline** is three ideas:

1. A **trigger** — an event in the repository (a push, a pull request) starts
   a run.
2. A **fresh machine** — the service boots a clean environment, checks out
   your repository, and installs the declared tools. Nothing from your laptop
   leaks in; if the repo doesn't say it, it isn't there.
3. **Steps** — a list of shell commands executed in order. Remember the
   terminal lesson from S0: every command exits with a status code, `0` for
   success. A step that exits non-zero fails the run, and the run's
   red-or-green verdict is attached to your commit or pull request.

That third point is the key mental model: *there is no CI magic*. A pipeline
runs the same `test`, `vet`, and format commands you run locally — the value
is the fresh machine and the "every push, no exceptions" trigger. It also
gives you the universal debugging move: anything a pipeline does, you can
reproduce at your own prompt by running the same commands from a clean
checkout.

In Go: a typical pipeline runs exactly what you already type by hand —
`go vet ./...` and `go test ./...` — plus a formatting check.

## Integration, delivery, deployment

"CI/CD" bundles three practices that are worth keeping distinct:

- **Continuous integration** — every push and merge request is built and
  tested automatically. Output: a red/green verdict. This lesson's focus.
- **Continuous delivery** — on top of CI, every green build of the main
  branch is packaged into a deployable artifact (a binary, a container image)
  and *could* be released at any moment. A human still decides when to press
  the button.
- **Continuous deployment** — remove the human: every green main-branch build
  deploys to production automatically. Same pipeline, one more automated step,
  and much higher demands on your test suite — the tests *are* the release
  decision now.

The progression is a trust ladder. You earn delivery by having CI you believe;
you earn deployment by having delivery so boring that the button press adds
nothing. Most teams live at delivery, and that is a fine place to live.

## GitHub Actions: the anatomy of a workflow

GitHub Actions is GitHub's built-in CI service and today's most common first
encounter with pipelines. You configure it with **workflow** files — YAML in
the repository at `.github/workflows/*.yml`. Committing the file *is* the
setup; there is no dashboard to click through.

A minimal, real workflow:

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
      - name: Test
        run: go test ./...
```

Read it top to bottom:

- `on:` — the triggers. This one fires on every push to any branch and on
  every pull request. These two cover the CI contract: nothing lands on main
  without a green run.
- `jobs:` — a workflow holds one or more jobs; each job gets its own fresh
  virtual machine, named by `runs-on:` (`ubuntu-latest` is the usual choice).
- `steps:` — the ordered commands. Two kinds:
  - `uses:` runs a published, reusable **action** — a packaged step.
    `actions/checkout` clones your repository into the fresh machine (without
    it the machine is empty!); `actions/setup-go` installs a Go toolchain,
    here reading the version from your `go.mod`. The `@v4` suffix pins the
    action's version.
  - `run:` executes a shell command — your commands, exactly as you'd type
    them.

One warning about YAML: indentation is structure. If the file won't parse,
GitHub shows a red workflow with a syntax error at a line number — check the
spaces on and around that line first, and never indent with tabs.

## Quality gates

A **gate** is any step whose failure blocks the run — and since a step fails
on any non-zero exit code, a gate is simply a command that exits non-zero when
it is unhappy. Tests are the obvious gate. Two cheaper ones belong in front of
them:

- **Formatting** — mechanical style enforcement, so review never wastes a
  minute on it (the clean-code lesson's argument, now automated).
- **Static analysis / lint** — tools that read the code without running it and
  flag likely bugs. This catches a class of mistake that tests miss: code that
  is wrong in a way none of your tests happens to execute.

Order gates cheapest-first: a formatting complaint in three seconds beats
waiting out the whole test suite first.

In Go, the two gates and one trap:

- `go vet ./...` — the standard analyzer; exits non-zero on findings. Its
  classic catch is a `Printf`-family format verb that disagrees with its
  argument — code that compiles and prints garbage.
- `gofmt -l .` — lists files that are not formatted. The trap: it *always
  exits 0*, even when it finds offenders, so as a bare step it can never fail.
  The idiom is to fail when its output is non-empty:

  ```sh
  test -z "$(gofmt -l .)"
  ```

  `test -z` exits non-zero when the string is non-empty — an offending file
  name in the output turns the step red.

## Reading a red run

A failing pipeline is a log-reading exercise, and you trained for it in the
debugging lesson. The moves:

1. **Open the failing step's log, not the summary.** The verdict says "red";
   only the step output says why.
2. **Find the first error.** Later failures are often fallout from the first
   one — same rule as compiler output.
3. **Reproduce locally.** Run the exact command from the failing step at your
   own prompt. Same failure: it's your code — diagnose as usual. Can't
   reproduce: suspect the environment difference — a file you forgot to
   commit, a tool version mismatch, something on your machine the workflow
   never installs.
4. **Fix the underlying cause.** Deleting the failing test, skipping the step,
   or hard-coding the expected value makes the light green and the code no
   better — the gate exists to protect main, not to be appeased. Then push and
   let the pipeline confirm.

## Exercise

Open [`exercise/`](exercise/) — a small Go repository that has never had CI.
Its previous owner left in a hurry. You will give it a pipeline, then act as
the pipeline's runner, and discover the pipeline was needed all along.

There is no GitHub server in this exercise, and nothing here touches the
network: `check.sh` plays referee. It inspects your workflow file and runs
your gates locally, exactly as a runner would.

Steps:

1. Read `stats.go` and `stats_test.go`. **Fix nothing yet** — walking in, you
   don't know whether this code is healthy. That's what the pipeline is for.
2. Write `.github/workflows/ci.yml` (inside the exercise folder): trigger on
   `push` and `pull_request`; one job on `ubuntu-latest`; steps for checkout,
   Go setup, the `gofmt` gate, `go vet ./...`, and `go test ./...` — gates
   cheapest-first.
3. Be the runner: from the exercise folder, run your three gate commands in
   the workflow's order. Two of them fail — and not independently: `go test`
   runs a subset of vet's checks before any test executes, so at first it
   refuses to run the tests at all and repeats the very diagnostic `go vet`
   just gave you. Read each log: which gate, which file, what exactly is
   wrong?
4. Diagnose and fix the underlying causes — in the code, cheapest gate
   first. Only once vet is green does `go test` deliver its real verdict: a
   genuine test failure, with its own log to read. The tests and the
   workflow gates are correct as specified; deleting or weakening them is
   appeasing the gate, not fixing the code.
5. Run the referee until green:

   ```sh
   bash ./check.sh
   ```

Acceptance criteria:

1. A workflow file exists under `.github/workflows/` (`.yml` or `.yaml`).
2. It triggers on both `push` and `pull_request`.
3. It has a job with `runs-on:` and steps using `actions/checkout` and
   `actions/setup-go`.
4. Its steps run a `gofmt` formatting gate, `go vet ./...`, and
   `go test ./...`.
5. `gofmt -l .` reports nothing.
6. `go vet ./...` passes — you found and fixed what it caught.
7. `go test ./...` passes — you diagnosed the failure from its log and fixed
   the code, not the test. `TestMean` and `Describe` still exist.

Afterwards, optionally: push this folder to any GitHub repository of yours and
watch the same workflow run for real, unchanged. It needs no modification —
that is the point.

## Further reading

- [Martin Fowler — Continuous Integration](https://martinfowler.com/articles/continuousIntegration.html)
  — the canonical essay on why merge-time automation wins.
- [GitHub Actions — Quickstart](https://docs.github.com/en/actions/writing-workflows/quickstart)
  — your first workflow, end to end.
- [GitHub Actions — Workflow syntax reference](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions)
  — every key you can put in a workflow file.
- [Martin Fowler — Continuous Delivery](https://martinfowler.com/bliki/ContinuousDelivery.html)
  — the delivery/deployment distinction in its author's words.
