# Tutor notes — Packages & Modules

## Where the learner is

Four lessons in: they can write variables, control flow, and functions, all
inside a single `package main` folder. Modules have been invisible plumbing —
every exercise shipped a ready `go.mod` they never questioned. This lesson
makes the plumbing visible. It's also the first time they import their own
code, so expect friction around import paths; the concepts are simple, the
paths are where typos live.

## Common misconceptions

- **Import path vs package name** — they'll write
  `import "temperature"` (bare folder name) instead of the module-rooted
  `tutor.local/packages-modules/temperature`. The error mentions the standard
  library or a missing module; walk them back to "module path + folder path".
- **"The import gives me the path, so I call `tutor.local/….CToF`"** — after
  importing, you use the *package name* (last segment): `temperature.CToF`.
- **`package main` in the sub-packages** — copying the only package line
  they've ever written. One folder = one package, and only `main` is
  runnable; libraries take their folder's name.
- **Capitalization is style, not semantics** — they may think `cToF` vs
  `CToF` is a naming preference like camelCase debates elsewhere. It's access
  control. The LESSON's rename experiment makes this concrete — check they
  actually did it.
- **Editing go.mod by hand to add a dependency** — the file looks editable
  (it is text) but the habit is `go mod tidy`/`go get`; hand-edits skip
  go.sum and drift from reality.
- **go.sum confusion** — deleting it, or thinking it's a second dependency
  list. It's checksums; `go mod tidy` regenerates it.

## Grilling points

- "Your `go.mod` says `module tutor.local/packages-modules`. What breaks if
  you change that line to `module foo`, and where?" (Every internal import
  path is rooted at the module path — they'd all need to change.)
- "Why can `report` call `temperature.CToF` but not a lowercase helper you
  might add to `temperature`? Who enforces that, and when?" (The compiler, at
  build time — `undefined` error.)
- "The test files say `package temperature_test`. What can those tests see,
  and why is that a feature?" (Only exports — the tests exercise the same
  surface importers get.)
- "You ran `go mod tidy` in the scratch module. What did it change on disk,
  and what would it do if you deleted the `rsc.io/quote` import and ran it
  again?" (Adds/removes `require` lines and go.sum entries to match source.)
- "Real modules are named like `github.com/you/project`. Why a URL-ish name
  instead of just `project`?" (Global uniqueness; the path doubles as the
  place to fetch it from.)

## Grading rubric

- **A** — All tests pass; `report.Line` calls into `temperature` (no
  duplicated conversion or classification logic); `main` imports `report`
  with the module-rooted path and prints at least two cities; gofmt-clean;
  learner did the rename experiment and can explain the resulting error, and
  can describe every line of go.mod plus what `go mod tidy` did in the
  scratch module.
- **B** — Tests pass but with logic duplicated in `report` (rebuilding the
  conversion inline) or `main` bypassing `report`; or the external-dependency
  walkthrough was skipped; explanation of exports and go.mod mostly solid.
- **C** — Tests pass only after heavy path-debugging by the tutor, or the
  learner treats import paths as magic strings they copy without being able
  to derive them from module path + folder. Time-boxed remediation before
  passing.
- **Fail** — Tests failing, or the learner cannot explain the difference
  between a package and a module, or why `CToF` is capitalized. Remediate,
  don't advance.

## Remediation ladder

1. "Run `go test ./...` from the exercise root and read the first error only.
   Is it a compile error or a failing assertion? What file and line?"
2. "Open `go.mod` and read the module line aloud. Now build the import path
   for the `temperature` folder from it — module path, slash, folder name."
3. "In `report.go`: which two facts in the output string can `report` not
   compute by itself? Which exported temperature functions provide them?"
4. Sketch `Line` verbally — convert, classify, then one `fmt.Sprintf` with
   `%s`, `%.1f`, `%.1f`, `%s` — and let them type it and fix the format-verb
   mismatches themselves.

## After passing

Preview: "Next lesson is arrays and slices — your first data structure:
one name that holds many values, and the workhorse of all Go code."
