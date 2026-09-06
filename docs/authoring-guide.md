# Lesson Authoring Guide

Exemplars — match their anatomy, tone, and rigor:

- Go, language pool: `skills/tutor/curriculum/content/go/s1-basics/hello-world/`.
- Python, language pool: `skills/tutor/curriculum/content/python/p1-basics/hello-python/`
  (lands with Phase 1 of the Python track).
- Language overlay for a shared lesson:
  `skills/tutor/curriculum/content/python/shared/s2-cs/arrays-linked-lists/`
  (lands with Phase 1 of the Python track).

Once present, the two Python directories are the templates for a Python
lesson and for any language's overlay. `docs/curriculum-outline.md` is
canonical for ids/titles/ordering; `registry.json` for objectives/duration/verify.

## Lesson directory anatomy

Two shapes. A **language-pool** lesson (`go/<stage>/<lesson>/`,
`python/<stage>/<lesson>/`) is self-contained:

```
<lesson>/
├── LESSON.md            # scaffolded to the learner
├── exercise/            # scaffolded — starter + tests, or check.sh
├── TUTOR.md             # tutor-only (never scaffolded)
├── quiz.json            # tutor-only
└── solution/            # tutor-only — test-verified: overlay of only the files
                         #   that change; script-verified: the reference
                         #   artifacts the grader and tutor check against
```

A **shared or pack** lesson (`shared/<stage>/<lesson>/`, `focus/<pack>/<lesson>/`)
is language-neutral, and each language adds an overlay in its own folder:
`content/<language>/<owner dir>/<lesson>/`, where the owner dir is the stage's
or pack's registry `dir` (`shared/s2-cs`, `focus/containers`):

```
shared/s2-cs/<lesson>/           # language-neutral
├── LESSON.md                    # theory with <!-- lang: <slot> --> anchors [rendered, then scaffolded]
├── exercise/                    # only when language-agnostic: check.sh,
│                                #   worksheets                             [scaffolded]
├── TUTOR.md                     # tutor-only
├── quiz.json                    # tutor-only
└── solution/                    # only beside an agnostic exercise/        [tutor-only]

python/shared/s2-cs/<lesson>/    # Python overlay (go/shared/… is the Go one)
├── snippets/<slot>.md           # one per anchor, rendered into LESSON.md  [rendered]
├── exercise/                    # starter + tests + README.md brief        [scaffolded as exercise/]
├── solution/                    # overlay for ci                           [tutor-only]
├── TUTOR.md                     # optional, additive to the shared one     [tutor-only]
└── quiz.json                    # optional, merged by question id          [tutor-only]
```

The effective exercise for a language is the shared `exercise/` overlaid by
the overlay's `exercise/` (overlay wins on the same relative path); the
effective solution is the overlay's `solution/` if it exists, else the shared
one. `exercises/<lang>/` and `solutions/<lang>/` are retired: `validate` errors
on any directory named `exercises` or `solutions` under `content/`.
Language-pool lessons have no overlay. Pack lessons keep a Go-shaped
`exercise/` and `solution/` (treated as agnostic) until the packs are ported.

## LESSON.md

Structure (see exemplar):

1. `# <Title>` then the meta line: `> \`<id>\` · ~<duration> · Stage: <stage title>`
2. `## Objectives` — the registry objectives, phrased "By the end you can …".
3. Theory sections — short, concrete headings; build one concept at a time.
4. `## Exercise` — what to build, explicit **acceptance criteria** (numbered),
   how to run the checks (shared lessons: see "Shared and pack lessons").
5. `## Further reading` — 2-4 curated links (up to 6 when a lesson spans
   several tools or specifications, as pack lessons often do), official
   sources strongly preferred. Go: go.dev, pkg.go.dev, the spec, canonical
   blog posts. Python: docs.python.org, peps.python.org, packaging.python.org,
   docs.pytest.org, docs.astral.sh (uv, ruff), mypy.readthedocs.io.

Tone and craft:

- Second person, plain language, no hype, no emoji. Explain *why*, not just
  *what*. Anticipate the reader's "wait, but…" questions and answer them.
- Assume exactly the knowledge of the preceding roadmap — reference earlier
  lessons ("remember S0's compiler model") and preview later ones by stage,
  never by leaking their content. Shared lessons name stages by role, not
  label (see below), because the stage before them differs per language.
- Length: a lesson is as long as its job, and its job grows through the
  roadmap. Bands, by what the lesson has to carry:

  | Where | Lines | Why |
  |-------|-------|-----|
  | Foundations and basics (the shared foundations stage, each language's basics stage) | 130-250 | one new idea at a time, nothing assumed; the exemplar sits at the bottom of the band |
  | Code-first stages (CS fundamentals, the intermediate language stage, engineering practice) | 230-340 | the exercise carries the teaching |
  | Advanced language stage | 250-360 | systems theory the code cannot show, plus production-scale acceptance criteria |
  | Systems and design | ≤380 | the theory *is* the lesson; the exercise is a worksheet, not a compiler |
  | Expert capstone | 330-400 | theory, a grading contract for a project the author cannot see, and two sets of criteria (mechanical + review) |
  | Focus packs | ≤440 | a pack compresses a specialist domain into 6-8 lessons, so each carries context a stage lesson would borrow from its neighbours |

  **Hard ceiling ~450 lines, anywhere.** Only a lesson that walks complete
  worked designs end to end earns it (`shared.systems.case-studies` is the
  one lesson past it), because half a worked design teaches worse than a
  long one. Past your band: split or cut. Dense beats long everywhere —
  length is never the goal, and a lesson that hits its band by padding has
  failed at both.
- Go accuracy bar: Go 1.22+ idioms — `log/slog`, generics where natural,
  1.22 `net/http` mux patterns, `errors.Is/As`, no `ioutil`, no deprecated
  APIs. When in doubt, check current docs.
- Python accuracy bar: Python 3.14+ idioms (commands checked against uv 0.12,
  ruff 0.16, pytest 9, mypy 2) — PEP 695 generics and `type` aliases,
  `X | None`, dataclasses + Protocol, pathlib, `TaskGroup` /
  `asyncio.timeout`, no `from __future__ import annotations`, no PEP 594
  modules, ruff-clean (`ruff format --check` + `ruff check`), mypy-clean
  where the lesson grades typing. When in doubt, check current docs.
- Shared-pool lessons: theory must stay language-portable. Pseudocode and
  diagrams are fine; concrete snippets live in each language's overlay and
  are rendered in at anchors.

## Shared and pack lessons

### Anchors and snippets

- An anchor is a whole line matching `^<!-- lang: ([a-z0-9][a-z0-9-]*) -->$`
  in `LESSON.md`. Slot names are unique per file, kebab-case, and name what
  the snippet shows (`dynamic-array-growth`, `stdlib-list`) — never the
  language or the position.
- Placement: on its own line between paragraphs, after the neutral paragraph
  it illustrates.
- Rendering happens at scaffold time, per workspace language: each anchor
  line (including its newline) is replaced by the verbatim bytes of
  `<overlay>/snippets/<slot>.md`; an anchor with no snippet for that
  language vanishes together with the blank line after it, so paragraph
  spacing and the rendered bytes of other languages are unchanged. Nothing
  else in `LESSON.md` changes. A language that has nothing to say for a
  slot simply ships no snippet.
- Snippet files: one per anchor, named after the slot, starting with the
  marker paragraph (`In Go:` / `In Python:`), ending with exactly one
  trailing newline. They carry only what the language adds — the neutral
  paragraph above the anchor already made the point.
- Only `LESSON.md` is rendered. `TUTOR.md` and `quiz.json` are never
  rendered: the tutor reads them from the shared dir and the overlay
  (overlay `TUTOR.md` is additive; overlay `quiz.json` merges by question
  `id`). No anchors in `TUTOR.md`, and none in language-pool lessons
  (`validate` warns).
- `validate` checks: anchors unique per file, every `snippets/*.md` has a
  matching anchor, every snippet ends with a newline.

### The Exercise section

- The shared `## Exercise` is language-neutral: what to build, the acceptance
  criteria that hold in any language, and "run the checks described in
  `exercise/README.md`". No file names, no toolchain commands — `validate`
  warns on `go test`, `go build`, `go run`, `go vet`, `gofmt`, `go mod`,
  `pytest` or `uv run` inside a shared `## Exercise`.
- The overlay's `exercise/README.md` carries the language-bound brief: file
  names, run commands, and the criteria that only make sense in that language
  (formatting, race-cleanness, which error an empty collection raises). Every
  overlay `exercise/` ships one (`validate` warns when it is missing).
- Refer to other stages by role, never by label: "your basics-stage
  collections lesson", "your intermediate-stage interfaces lesson", "the
  service you built in your advanced stage", "your language's capstone
  stage" — never "S1" or "p1".

## exercise/

- Go lessons: a self-contained module — `go.mod` with
  `module tutor.local/<name>` (the lesson slug unless a shorter name reads
  better in import paths, as in `tutor.local/board`) and `go 1.22`. Starter
  code **must compile** but tests **must fail** (learners fight the problem,
  not the scaffolding). Mark work sites with `// TODO:` comments. Two
  deliberate exceptions: lessons whose exercise *is* writing tests
  (`go.basics.testing-basics`, `shared.eng.tdd`, `go.advanced.advanced-testing`,
  `python.basics.testing-basics`, `python.advanced.advanced-testing`)
  ship stub tests as the starter and the finished tests in the solution, and
  debugging-style lessons ship complete, seeded-bug code with no `TODO`s.
- Tests (`*_test.go`) are the specification: table-driven where natural,
  failure messages that teach (`got %q, want %q`). Test the acceptance
  criteria exactly — no hidden requirements. `verify` and `ci` run
  `go test -race ./...`, so tests and solutions must be race-clean.
- Python lessons: flat layout — `<module>.py` + `test_<module>.py` directly
  in `exercise/` — through the basics stage and the intermediate stage up to
  `python.intermediate.packaging`; `src/` layout from `packaging` on. No
  `sys.path` hacks or `conftest.py` tricks to make imports work: the layout
  does that. Every pytest exercise ships `pyproject.toml`
  (`requires-python = ">=3.14"`, `dependencies`,
  `[dependency-groups] dev = ["pytest==<pinned>"]`), a committed `uv.lock`,
  and `.python-version` containing `3.14`; `validate` requires all three.
  Template for a basics lesson with no dependencies:

  ```toml
  [project]
  name = "<lesson-slug>"
  version = "0.0.0"
  requires-python = ">=3.14"
  dependencies = []

  [dependency-groups]
  dev = ["pytest==9.1.1"]

  [tool.uv]
  package = false
  ```

  Where the lesson grades typing, add `mypy>=2,<3` to the dev group; the
  graded checker is `mypy --strict`. Starter files must import and collect
  cleanly (`uv run -q --locked pytest --collect-only -q` succeeds) but
  `pytest` must fail. Work sites are `# TODO:` comments; stubs return a
  placeholder (`return ""`, `return None`) or `raise NotImplementedError`.
  Tests (`test_*.py`) follow the same specification rule,
  `pytest.mark.parametrize` where a Go lesson would use a table; `verify` and
  `ci` run `uv run -q --locked pytest -q`.
- Output pasted into prose comes from `pytest -q --no-header` and
  `uv run -q`, scrubbed of machine paths; any path uses the `ada` persona
  (`/home/ada/...`).
- `script`-verify lessons: `exercise/check.sh` — bash, self-contained,
  idempotent, exits non-zero with a clear "what to fix" message per failed
  check. The engine runs it as `bash ./check.sh` from the exercise dir; the
  learner runs it repeatedly until green. Graders that inspect the learner's
  machine rather than files (dev-environment) ship a `--self-test` mode that
  runs the same checks against a fixture directory so CI can exercise them
  (lands with Phase 3 of the Python track); every other script lesson ships a
  `solution/`.
- `discussion`-verify lessons: exercise dir optional; if present it's an
  exploration task the tutor reviews in conversation.
- Shared lessons whose exercise is language-specific put it in the overlay's
  `exercise/` (same content rules, plus the `README.md` brief);
  language-agnostic ones (terminal, git) use the shared lesson's plain
  `exercise/`.

## solution/

Test-verified lessons: contains **only the files that differ** from the
starter (it is overlaid onto the effective exercise at CI time; for a shared
lesson that is the overlay's `solution/`). Must be idiomatic,
formatter-clean — `gofmt` for Go, `ruff format` + `ruff check` for Python —
and the code you'd defend in review — it is also the tutor's reference during
grading. Python overlays only the files that change, as Go does.

Script-verified lessons: the reference artifacts the exercise asks for
(Dockerfile, manifests, a `NOTES.md` with the expected observations). `ci`
overlays them onto the exercise and runs `check.sh`, so the reference must
pass its own grader. A script lesson without a `solution/` is skipped by
`ci`, which is acceptable only when the grader inspects the learner's
machine rather than files (the foundations-stage terminal, git and
dev-environment lessons — the last is covered by its `--self-test` mode
instead). `shared.eng.ci-cd` is skipped today as well; its solutions land
with Phase 3 of the Python track.

Discussion-verified lessons have no `solution/`.

## TUTOR.md

Sections, in order (see exemplar): `## Where the learner is`,
`## Common misconceptions`, `## Grilling points`, `## Grading rubric`
(A/B/C/Fail, exercise-specific), `## Remediation ladder` (3-4 escalating
hints, never jumping to the answer), `## After passing` (one-line preview of
what's next).

Shared lessons: the shared `TUTOR.md` carries what holds in every language —
misconceptions about the concept, grilling points, the rubric's shape, hints
phrased in terms of the idea. The overlay `TUTOR.md` (optional, additive)
carries what does not transfer: where the learner is in that language's track,
language-bound misconceptions, rubric lines about idioms and tooling, hints
that name files or APIs. Same section headings, only the ones you need.

## quiz.json

```json
{
  "pass_rule": "all core questions substantially correct …",
  "questions": [
    {"id": "slug", "difficulty": "core|stretch",
     "prompt": "…", "expected_points": ["…", "…"]}
  ]
}
```

4-7 questions, at least 3 `core` — up to 9 where five registry objectives each
need a core question of their own, as most of the advanced stage does.
Coverage sets the count, not the reverse: together the core questions must
cover every registry objective. Prompts are conversation starters, not exam
prose.

Shared lessons: the shared `quiz.json` holds the questions answerable in any
language and covers every objective on its own. The overlay `quiz.json`
(optional, same shape) replaces a shared question with the same `id` and adds
the rest — use it where the expected points invert per language, keeping the
`id` so coverage stays whole.

## Self-check before you're done (all must pass)

Go:

```sh
cd exercise && go test -race ./... ; cd -   # must FAIL on the starter
gofmt -l <lesson-dir>                        # no output
python3 -m json.tool <lesson-dir>/quiz.json  # parses
python3 skills/tutor/scripts/tutor.py ci --language go --filter <lesson-id>   # solution passes
```

Python:

```sh
cd exercise && uv run -q --locked pytest --collect-only -q && uv run -q --locked pytest -q; cd -   # collects, then FAILS
ruff format --check <lesson-dir> && ruff check <lesson-dir>
python3 -m json.tool <lesson-dir>/quiz.json
python3 skills/tutor/scripts/tutor.py ci --language python --filter <lesson-id>   # prints "1 solution(s) tested, 0 failed"
```

For an overlay, `exercise` and `<lesson-dir>` are the overlay's; `<lesson-id>`
is the shared lesson's id. The `ci` summary line must show
`1 solution(s) tested, 0 failed` (discussion lessons: `0 solution(s) tested`).
`ci` also runs `validate --strict` (anchors, snippets, the pytest trio,
TUTOR.md/quiz.json presence and parse).

### Extra check: capstone reference projects

The S7 capstone harnesses grade whatever project they are pointed at, so each
lesson's `solution/testdata/reference/` is an exemplar of "a capstone at this
point in the stage" — and the tutor reads it during grading. Those exemplars
accumulate: **every reference must pass its own lesson's Go harness *and* every
earlier one.** A lesson-6 reference that fails lesson 3's harness tells the
learner the stage's premise is a lie.

```sh
# from a scratch dir, for each harness H and each reference R at or after it
cp -R <H>/exercise/. work/ && cp <H>/solution/capstone.path work/
mkdir -p work/testdata && cp -R <R>/solution/testdata/reference work/testdata/
(cd work && go test ./...)
```

`operations` is the documented exception in one direction only. Its referee
(`check.sh`) grades a deployable-service shape the `notes` lineage deliberately
does not carry, so it runs against the operations fixture alone — but that
fixture is still a Go project, so it must clear the build-core and hardening
harnesses like everything else. See `operations/solution/README.md`.

The P7 harnesses are pytest suites that shell out to `uv run pytest` (with
coverage through pytest-cov, from the dev group or `uv run --with pytest-cov`;
settled when P7 is authored),
`ruff format --check` / `ruff check` and `mypy --strict` against the project
at `capstone.path`. References accumulate the same way — every reference
passes its own harness and every earlier one — with
`uv run -q --locked pytest -q` in place of `go test ./...` in the loop above.
`operations` mirrors the Go exception: its `check.sh` grades a
deployable-service shape against its own FastAPI fixture alone, and that
fixture is still a Python project that must clear the build-core and
hardening harnesses.
