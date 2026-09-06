# CONTEXT.md — Project Contract & Build State

> Source of truth for the `tutor` project. Any agent (or human) resuming work on this
> repo MUST read this file first. Update the **Build Status** section as phases complete.
> The full grilled contract lives here; `docs/DESIGN.md` carries deeper rationale.

## Vision

`qtsone/learning` is an open-source **curriculum engine + tutoring skill** ("tutor").
Loaded onto a Claude agent, it turns the agent into a hand-holding-to-expert programming
tutor. The curriculum is fully pre-authored and versioned in this repo; a deterministic
script scaffolds it into any learner workspace and manages progress state idempotently.
First language: **Go** (0 → expert); second: **Python** (Phase 6, in progress). The
architecture supports many languages by composing shared, language-agnostic modules
with language tracks and focus packs.

Commercial context: qtsone may additionally offer this work under separate commercial
terms (dual licensing). Licensing is designed so qtsone can do this while all public
copies/derivatives must stay open (see Licensing).

## Decision log (grilled + confirmed 2026-08-16; rows 19-32 confirmed 2026-09-06)

| # | Decision | Choice |
|---|----------|--------|
| 1 | Positioning | New standalone skill `tutor` (not an extension of the `teach` skill) |
| 2 | Skill home | This repo at `skills/tutor/`, symlinked into `~/.claude/skills/tutor`; workspaces are wherever `/tutor` is invoked |
| 3 | Content model | **Fully pre-authored** lessons (real files in repo). Scaffold = deterministic copy; re-scaffold = hash diff |
| 4 | Graph model | Central `registry.json` (lessons, prereq DAG, tracks, packs) + content pools (`shared/`, `go/`, `focus/`, `python/`) |
| 5 | Script tech | Python 3 **stdlib only**, JSON registry, zero dependencies |
| 6 | State model | Script-owned `state.json` + `manifest.json` (all mutations via subcommands) + LLM-owned `journal.md` |
| 7 | Improve loop | Tutor logs curriculum observations to journal; `/tutor contribute` aggregates → branch → PR to this repo. Never automatic |
| 8 | Re-scaffold | Update in place unless (upstream changed AND user modified) → sidecar + conflict. Changed **passed** lessons → `needs_review`, covered before advancing |
| 9 | Tutor-only files | `TUTOR.md`, `quiz.json`, `solution/`, `snippets/` live beside lessons and in overlays but are **never scaffolded** |
| 10 | Mastery | Strict gates: theory → Socratic check → tests pass → rubric review → letter grade recorded. Fail → remediation. Force-skip recorded as `skipped`. Stage capstones + spaced review |
| 11 | Roadmap | 8 stages, ~90 lessons on Go path (see Curriculum), + focus packs |
| 12 | Focus packs | Registry-defined: `containers`, `web-services`, `cli-tooling` (authored), `ml` (stub). Freeform focuses = tutor-generated `custom` lessons, excluded from idempotent diffing |
| 13 | Launch scope | **Author everything now**: all Go-path stages + 3 packs (~112 lessons). Python = registry stub only (superseded by rows 19+: the Python track is authored in Phase 6) |
| 14 | Guidance | `guidance` in state: `guided` / `standard` / `spartan`. Default `guided`; tutor proposes changes, only user changes it |
| 15 | Invocation | One skill `/tutor` + freeform args; LLM maps intent to deterministic script calls |
| 16 | Lesson IDs | Stable slugs (`go.basics.arrays-slices`); ordering computed; sync renames dirs safely (never clobbers content) |
| 17 | License | **AGPL-3.0-or-later on everything** + enforced CLA (sublicense rights enable qtsone's commercial dual-licensing) + §7 learner exception (learners' exercise solutions unencumbered) + explicit qtsone copyright |
| 18 | OSS hygiene | README, CONTRIBUTING → CLA, CLA CI check, SECURITY.md, CODE_OF_CONDUCT.md, issue forms, CODEOWNERS; CI validates registry schema / DAG acyclicity / content presence / every language's solutions pass tests, and rejects machine-specific paths (lesson prose uses the example persona `ada`). Renovate manages GitHub Actions only. Local, uncommitted git hooks keep private names and machine paths out of files and commit messages |
| 19 | Per-language verify | `languages.<code>.verify.type` names the language's runner (`gotest`, `pytest`). Shared and pack test lessons use the neutral type `tests`, resolved per workspace language by `resolve_verify(reg, lid, language)`; `gotest`/`pytest` only on language-pool lessons; `script` and `discussion` anywhere. No registry schema bump |
| 20 | Python runner | `uv run -q --locked pytest -q`. Every pytest exercise ships `pyproject.toml`, a committed `uv.lock` and `.python-version`; uv is the Python path's toolchain prerequisite; the engine preflights the toolchain binary and prints an install hint |
| 21 | Python floor | `requires-python = ">=3.14"`, bar phrase "Python 3.14+ idioms"; CI runs 3.14, adds 3.15 when it ships |
| 22 | Type checker | mypy 2.x (`>=2,<3`) is the graded checker, run as `mypy --strict` from `check.sh`; ty named as the beta editor LSP |
| 23 | Language status | No new status. `ci` and `validate` test whatever is authored for every language; missing content is reported only for `available` (warning, error under `--strict`); `init` keeps refusing `stub`. Python stays `stub` until Phase 6.7 |
| 24 | Release model | Author everything, then flip (the Go precedent) |
| 25 | Language overlays | One folder per language: `content/<lang>/shared/<stage>/<lesson>/` (and `content/<lang>/focus/<pack>/<lesson>/`) holds `snippets/<slot>.md`, `exercise/` (with `README.md`), `solution/`, optional `TUTOR.md` and `quiz.json`. Shared `LESSON.md` stays neutral with `<!-- lang: <slot> -->` anchors; the engine renders `LESSON.md` per language at scaffold time. `exercises/<lang>/` and `solutions/<lang>/` are retired; Go migrated by script, byte-identical for Go learners |
| 26 | Exercise briefs | The `## Exercise` section of a shared lesson is language-neutral; file names, run commands and language-bound criteria live in the overlay's `exercise/README.md` |
| 27 | Shared batch | All 42 shared-lesson edits land in one `content(shared)` PR, so Go learners see exactly one `needs_review` wave; the pack port (Phase 6.8) is a second, announced wave |
| 28 | Polyglot learners | One workspace per language. `init <lang> --carry-over DIR` writes the earlier grade and date into `notes` for every lesson on the new path that is `passed` in `DIR`, status stays `todo`; SKILL.md fast-tracks those (short spaced-review check, the new language's exercise, grade) |
| 29 | CI shape | `validate --strict` + `tests/test_engine.py` in the validate job; `solutions` is a `[go, python]` matrix on pull requests with path filters, each row running `ci --language <code>`; setup-uv pinned by commit SHA; a scoped ruff step over `content/python` |
| 30 | Workspace hygiene | `init` writes a `.gitignore` seed (generic Python entries + `languages.<code>.workspace_ignore`); `JUNK_NAMES` gains `.venv`, `venv`, `.hypothesis`, `.coverage`, `.tox`, `.nox`, `.git`; `init` and `verify` warn when an ancestor directory holds a `pyproject.toml` |
| 31 | Python track shape | 54 language lessons (p1 17, p3 18, p5 13, p7 6), 96 on the path. Verify types are honest: `packaging`, `typing`, `generics`, `operations` are `script`; `tooling`, `runtime-gil`, `planning`, `oss-contribution` are `discussion`; the rest `pytest`. Only prereq: `build-core` → `planning`. Ids freeze at first content ship |
| 32 | Python variant conventions | Empty collections raise `IndexError`/`KeyError`; one injected clock shape `now: Callable[[], float]`; operation-count probes replace wall-clock guards; one stage-wide `sys.setrecursionlimit` note; `hashlib.scrypt` as the exercise KDF; a committed test-only PEM pair with `regen.py`; httpx against a threaded `http.server`, never `MockTransport` |

## Architecture contract

### Repo layout
```
learning/
├── CONTEXT.md                  # this file
├── README.md, CONTRIBUTING.md, SECURITY.md, CODE_OF_CONDUCT.md
├── LICENSE, LICENSE-EXCEPTION.md, NOTICE
├── ruff.toml                   # lint/format settings for Python content (target py314)
├── renovate.json               # GitHub Actions only: pins inside the curriculum are teaching material
├── .github/                    # CLA.md, CODEOWNERS, workflows/ci.yaml, issue forms, PR template
├── docs/                       # DESIGN.md, authoring-guide.md, curriculum-outline.md
├── tools/                      # authoring and review workflow scripts
├── tests/                      # test_engine.py: black-box engine tests (stdlib unittest)
└── skills/tutor/
    ├── SKILL.md                # tutor behavior: teaching protocol, intent → script map
    ├── scripts/tutor.py        # deterministic engine (stdlib only)
    └── curriculum/
        ├── registry.json       # the whole graph
        └── content/
            ├── shared/<stage>/<lesson>/     # language-neutral theory; exercise/ only when agnostic
            ├── focus/<pack>/<lesson>/       # same shape (Go-shaped until Phase 6.8)
            ├── go/<stage>/<lesson>/         # Go language stages
            ├── go/shared/<stage>/<lesson>/  # Go overlay for a shared lesson
            ├── go/focus/<pack>/<lesson>/    # Go overlay for a pack lesson (Phase 6.8)
            └── python/                      # same shape: p1-basics/ …, shared/, focus/
```

### Lesson directory (in repo)
```
<lesson>/              # language-pool lesson, or the neutral half of a shared/pack lesson
├── LESSON.md          # theory, objectives, curated further-reading   [scaffolded, rendered per language]
├── exercise/          # starter code + tests (language pool), or an   [scaffolded]
│                      #   agnostic check.sh / worksheet (shared, focus)
├── TUTOR.md           # teaching notes, misconceptions, grilling pts  [never scaffolded]
├── quiz.json          # question bank + grading rubric                [never scaffolded]
└── solution/          # canonical solution                            [never scaffolded]

content/<lang>/<owner dir>/<slug>/   # overlay; owner dir = the stage's or pack's registry `dir`
├── snippets/<slot>.md # one file per `<!-- lang: <slot> -->` anchor    [rendered into LESSON.md]
├── exercise/          # starter + tests + README.md brief; wins over   [scaffolded as exercise/]
│                      #   the lesson dir's exercise/ on the same path
├── solution/          # replaces the lesson dir's solution/ for ci    [never scaffolded]
├── TUTOR.md           # additive to the shared TUTOR.md               [never scaffolded]
└── quiz.json          # questions merged by id over shared quiz.json  [never scaffolded]
```
Language-pool lessons have no overlay: their lesson dir is the language dir. The effective
exercise for (lesson, language) is the lesson dir's `exercise/` overlaid by the overlay's
`exercise/`; the effective solution is the overlay's `solution/` if present, else the lesson
dir's. No directory under `content/` may be named `exercises` or `solutions` (`validate` errors).

A lesson is "authored" for a language iff `LESSON.md` exists and the effective exercise
carries what the resolved verify type needs: a test file matching the runner's glob for
test lessons, `check.sh` for `script`, nothing more for `discussion` (derived from
filesystem, no status field). Unauthored lessons appear in ROADMAP.md as *content
pending*; scaffold skips them.

### Verify types
Lesson `verify.type` is one of `tests`, `gotest`, `pytest`, `script`, `discussion`.
`tests` means "the workspace language's runner" (`languages.<code>.verify.type`, a key of
the engine's test-file globs: `gotest` → `*_test.go`, `pytest` → `test_*.py`) and is
allowed only on shared and pack lessons. `gotest` and `pytest` are allowed only on
language-pool lessons and must equal that language's runner. `script` (`bash ./check.sh`)
and `discussion` are allowed anywhere. `resolve_verify(reg, lid, language)` is the one
place the mapping happens; `verify`, `ci`, `validate` and `graph` go through it. Runner
commands: `go test -race ./...`, `uv run -q --locked pytest -q`, `bash ./check.sh`; the
engine checks the toolchain binary is on PATH before running and prints an install hint
when it is not.

### Per-language rendering
A shared or pack `LESSON.md` is language-neutral and may carry anchor lines of the form
`<!-- lang: <slot> -->` (kebab-case slot, unique per file, on its own line after the
neutral paragraph it illustrates). At scaffold time the engine replaces each anchor line
with the verbatim bytes of `<overlay>/snippets/<slot>.md` for the workspace language and
deletes an anchor that has no snippet together with the blank line after it, so paragraph
spacing and the rendered bytes of other languages are unchanged; nothing else in the file
changes, and the manifest hash covers the rendered bytes. Only `LESSON.md` is rendered: the
tutor reads `TUTOR.md` and `quiz.json` from both the shared dir and the overlay (overlay
`TUTOR.md` is additive; overlay quiz questions replace shared ones with the same `id` and
add the rest). The Go migration moved every `In Go:` block into a snippet and left an
anchor in place, so a Go workspace scaffolds byte-identically before and after. A language
that has nothing to say for a slot ships no snippet. `validate` checks anchors are unique,
every snippet has an anchor, and snippets end with a newline.

### Workspace layout (learner side)
```
workspace/
├── ROADMAP.md         # generated: ordered roadmap, checkboxes, grades
├── .tutor/
│   ├── state.json     # script-owned progress state
│   ├── manifest.json  # script-owned scaffold hashes (lesson-id keyed) + composition_hash
│   ├── journal.md     # LLM-owned session notes + curriculum observations
│   └── attic/         # lessons removed upstream (never deleted)
├── lessons/<NN>-<group>/<NN>-<lesson-slug>/
└── projects/          # capstones
```

### tutor.py subcommands (all state mutations go through these)
```
init <language> [--focus a,b] [--carry-over DIR]
                                 # create workspace / add focuses; idempotent (implies sync);
                                 #   refuses stub languages; one workspace per language; writes
                                 #   a .gitignore seed; --carry-over notes the grades of lessons
                                 #   passed in DIR's workspace on the new path (status stays todo)
sync                             # re-scaffold + JSON diff report (added/updated/conflicts/
                                 #   removed→attic/renamed/needs_review)
status [--json]                  # session briefing: progress, next lesson, needs_review,
                                 #   pending sync preview (read-only diff against upstream)
mark <lesson-id> <status|resolved> [--grade A..F] [--note ...]
                                 #   resolved: needs_review → the status held before the change
verify <lesson-id>               # run the lesson's runner for the workspace language; counts an
                                 #   attempt; refuses when the exercise dir is not scaffolded
guidance <guided|standard|spartan>
custom add <slug> --title T      # register a tutor-generated custom lesson (excluded from sync)
graph [--language X] [--focus ..] [--format tree|mermaid|json]   # no workspace needed;
                                 #   authored counts and verify types per language
validate [--strict]              # repo-level: registry schema, DAG order, verify-type and
                                 #   layout invariants, anchors/snippets, content presence for
                                 #   available languages; --strict promotes completeness
                                 #   warnings to errors
ci [--filter substr] [--language CODE]
                                 # strict validate, then every authored lesson's solution
                                 #   against its exercise, once per language that can take it
```

### Sync semantics (per file)
- upstream unchanged → untouched.
- upstream changed, workspace file pristine (== manifest hash) → update in place.
- upstream changed AND workspace modified → write `<file>.upstream` sidecar + conflict entry.
- lesson `passed`/`skipped` with any upstream change → state → `needs_review`; tutor must
  cover the delta before the learner advances past it.
- new lesson → scaffold, report `added`; removed lesson → move dir to `.tutor/attic/`.
- composition order changed → rename dirs (manifest keys are lesson ids, renames are safe).

### state.json (schema 1)
`{schema, language, focuses[], guidance, created, updated, lessons: {<id>: {status, grade?, attempts, notes?, timestamps}}, custom_lessons[]}`
Statuses: `todo | in_progress | passed | skipped | needs_review`. Locked/available is computed.
`manifest.json` carries `composition_hash` (canonical JSON of the composed groups, each
lesson's registry entry, the language entry and the registry version); `status` reports
`sync_needed` when it differs, so registry edits to another language do not nag.

### Teaching protocol (enforced by SKILL.md)
1. Every session: run `status` first, brief the learner, honor `guidance` mode.
2. Per lesson: assign reading → Socratic check-in (from `quiz.json` + freeform grilling)
   → learner writes exercise code in workspace → `verify` passes → code review against
   rubric (grade + what/why + improvements) → `mark passed --grade`.
3. Never reveal solutions before pass (unless learner insists; note it in journal).
4. Remediate on fail; re-quiz later per spaced review. Stage boundary: capstone + review quiz.
5. Log curriculum observations to journal as structured entries:
   `- [<lesson-id>] <issue|gap|errata|difficulty> — <observation> — suggested: <fix>`.

## Curriculum (Go path)

| Stage | Pool | ~Lessons | Content |
|-------|------|---------|---------|
| S0 Foundations | shared | 8 | what programs are, terminal, git, editors, reading docs |
| S1 Programming basics | go | 15 | syntax, types, control flow, funcs, pointers, errors, testing |
| S2 CS fundamentals | shared | 12 | data structures, algorithms, Big-O, recursion |
| S3 Intermediate | go | 15 | interfaces, generics, stdlib, concurrency, tooling |
| S4 Engineering practice | shared | 10 | clean code, TDD, debugging, SQL, security, CI |
| S5 Advanced | go | 12 | runtime/GC/scheduler, memory model, profiling, servers, gRPC |
| S6 Systems & design | shared | 12 | system design, distributed systems, networking, observability |
| S7 Expert capstone | go | 6 | production project, perf engineering, OSS contribution |

Packs (insertion points where prerequisites are met, typically after S3/S5):
`containers` (~8: Docker, multi-stage builds, k8s, deploying Go), `web-services` (~8),
`cli-tooling` (~6), `ml` (stub). Packs are Go-only today and port per language in Phase 6.8
with the overlay shape. Python track: `stub` in the registry until Phase 6.7 flips it.

## Curriculum (Python path)

| Stage | Pool | Lessons | Content |
|-------|------|---------|---------|
| S0 Foundations | shared | 8 | as above; Python snippets and a `dev-environment` check |
| P1 Programming basics | python | 17 | uv and the REPL, names and types, control flow, functions, modules, strings and bytes, lists/dicts/sets, iteration, references and mutability, classes, dunders, dataclasses, exceptions, pytest, files; CLI tracker |
| S2 CS fundamentals | shared | 12 | as above; Python overlays, five exercises redesigned (big-o, recursion, sorting, heaps, problem-patterns) |
| P3 Intermediate | python | 18 | iterators and generators, decorators, context managers, Protocols, inheritance, packaging, pattern matching, typing, generics, JSON, time, threads and the GIL, asyncio, synchronization, cancellation, concurrency patterns, tooling; concurrent tool |
| S4 Engineering practice | shared | 10 | as above; Python overlays (sqlite3, httpx, a uv-aware ci-cd referee) |
| P5 Advanced | python | 13 | ASGI servers, FastAPI, gRPC, SQLAlchemy, the CPython runtime and GIL, memory, free-threading, profiling, the extension boundary, advanced testing, metaprogramming, observability; production service |
| S6 Systems & design | shared | 12 | as above; four test overlays (networking, caching, message-queues, scalability), eight snippet-only |
| P7 Expert capstone | python | 6 | planning, core build, hardening, operations, OSS contribution, performance engineering |

96 lessons on the path (Go: 90). One workspace per language: a learner who finished the Go
path starts Python with `init python --carry-over <go-workspace>`; every shared lesson
passed there gets a note with the earlier grade and date, status `todo`, and the tutor
fast-tracks it (short spaced-review check, the Python exercise, grade). The gate stays
honest because the exercise is per language.

## Build status

- [x] Phase 0 — CONTEXT.md (this file) committed
- [x] Phase 1 — Framework: `tutor.py`, `SKILL.md`, registry schema, OSS files (LICENSE+exception, CLA, CONTRIBUTING, README), CI workflow, docs/DESIGN.md
- [x] Phase 2 — Full registry graph: 136 lessons declared (112 Go path incl. packs + 24 Python stubs); `validate` passes with 0 errors. Golden template lesson `go.basics.hello-world` fully authored — USE IT AS THE EXEMPLAR for all Phase 3 authoring (file anatomy, tone, exercise/test/solution/TUTOR.md/quiz.json conventions)
- [x] Phase 3 — Content authoring (multi-agent workflows; author → adversarial review → fix; solutions actually run against tests). All 112 Go-path lessons authored; every solution passes `ci`:
  - [x] S0 foundations (8)
  - [x] S1 go basics (15)
  - [x] S2 cs fundamentals (12)
  - [x] S3 go intermediate (15)
  - [x] S4 engineering practice (10)
  - [x] S5 go advanced (12)
  - [x] S6 systems & design (12)
  - [x] S7 capstone (6)
  - [x] pack: containers (8)
  - [x] pack: web-services (8)
  - [x] pack: cli-tooling (6)
- [x] Phase 4 — End-to-end validation: scratch workspace driven only through `tutor.py`. Covered init/status/graph/verify/mark/guidance, byte-identical no-op sync and re-init, and every sync quadrant against a mutated curriculum copy (pristine update in place, sidecar + conflict, `needs_review` for passed and skipped, added lesson, removal to attic, stage/pack reorder with renames), then resume from saved state and a green `ci`. Engine deviations found were fixed in separate commits (no-op sync wrote state, `attempts`/`notes` missing from state, files added upstream not detected, attic clobbered on re-removal)
- [x] Phase 5 — Release: symlink install verified from a clean shell; docs reconciled with the engine (README engine reference + workspace layout, SKILL.md sync report fields, CONTRIBUTING and the authoring guide matched to the shipped lesson anatomy and length bands); one lesson per stage and pack spot-checked against the golden lesson, drift fixed per commit; engine fix (tool caches next to lessons were scaffolded); `validate`, `ci`, and the CI workflow green; CI badge; tagged v0.1.0 with a GitHub release
- [ ] Phase 6 — Python track: 54 language lessons + 42 shared overlays; Python stays `stub` until 6.7. Each sub-phase ends with `validate --strict`, `ci` and `tests/test_engine.py` green:
  - [x] 6.0 Contract, engine, docs, tools: rows 19-32 and this checklist; per-language verify, overlays and rendering, `ci --language`, `validate --strict`, carry-over, hygiene; Go migrated to overlays by script; registry rewritten to the 54 ids; outline, authoring guide, workflow scripts (`LANG` table, `variant` mode), CI matrix, `ruff.toml`, templates. Exit: `graph --language python` shows the real path; `ci --language python --filter <id>` can run a lesson the day it exists; Go workspaces byte-identical before and after (rendering test)
  - [ ] 6.1 Exemplars: `python.basics.hello-python` and the Python overlay of `shared.cs.arrays-linked-lists`, hand-authored to the golden bar and referenced from the guide and the scripts
  - [ ] 6.2 p1: the other 16 lessons in slot order. Exit: `ci --language python --filter python.basics` green
  - [ ] 6.3 The shared batch, one `content(shared)` PR: neutral Exercise sections, stage-role wording, neutral registry objectives, reading-docs split, ci-cd solutions, `--self-test` mode for the dev-environment graders, TUTOR/quiz language sections, the unmarked Go prose and ```go fences the migration left in 8 shared lessons (grep for them; `validate` warns on toolchain commands in `## Exercise`); then the S0 and S2 Python overlays with the five S2 redesigns. Exit: exactly one `needs_review` wave for Go learners; `ci --language python --filter shared.cs` green
  - [ ] 6.4 p3 (18; the concurrency arc authored together) then the S4 overlays (10). Exit: path s0…s4 complete for Python
  - [ ] 6.5 p5 (13; each dependency's 3.14 wheel checked first) then the S6 overlays (4 test + 8 snippet-only). Exit: path s0…s6 complete
  - [ ] 6.6 p7 (6) with harnesses and reference projects; every reference clears every earlier harness. Exit: 96 lessons authored
  - [ ] 6.7 Validate, flip, release: Phase 4-style end-to-end on scratch workspaces (`init python`, both overlays of a shared lesson, no-op sync, carry-over, every sync quadrant); `languages.python.status` → `available`; README, SKILL.md, CONTRIBUTING, this file, DESIGN.md; registry version bump. Exit: `feat(python): open the Python track` → v0.2.0, matrix row matches Go
  - [ ] 6.8 Packs: insertion-level `languages` filter and per-language duplicate check with sync tests; cli-tooling, containers, web-services ported in that order; ml and data-engineering stubs. Exit: packs open per language; second, announced `needs_review` wave for pack learners

Release and versioning policy: the registry `version` is CalVer and bumps on any change to a
composed path; opening a language is a `feat` (semantic-release cuts a minor release);
lessons land as `content(<stage>)` / `content(<pack>)` commits and ship with the next
release; the shared batch is one `content(shared)` PR.

Conventions for resuming a build session: read this file, `git log --oneline -10`,
then continue the first unchecked phase. Commit at phase boundaries to `main` with
conventional-commit messages. Content authoring commits per stage. `tests/test_engine.py`
(`python3 -m unittest discover -s tests -v`) must stay green alongside `validate --strict`
and `ci`; content phases open a PR per phase so the `solutions` CI job, which runs on pull
requests only, exercises every language's row before merge.
