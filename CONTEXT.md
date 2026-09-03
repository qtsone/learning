# CONTEXT.md — Project Contract & Build State

> Source of truth for the `tutor` project. Any agent (or human) resuming work on this
> repo MUST read this file first. Update the **Build Status** section as phases complete.
> The full grilled contract lives here; `docs/DESIGN.md` carries deeper rationale.

## Vision

`qtsone/learning` is an open-source **curriculum engine + tutoring skill** ("tutor").
Loaded onto a Claude agent, it turns the agent into a hand-holding-to-expert programming
tutor. The curriculum is fully pre-authored and versioned in this repo; a deterministic
script scaffolds it into any learner workspace and manages progress state idempotently.
First language: **Go** (0 → expert). The architecture supports many languages by
composing shared, language-agnostic modules with language tracks and focus packs.

Commercial context: qtsone may additionally offer this work under separate commercial
terms (dual licensing). Licensing is designed so qtsone can do this while all public
copies/derivatives must stay open (see Licensing).

## Decision log (grilled + confirmed 2026-08-16)

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
| 9 | Tutor-only files | `TUTOR.md`, `solution*/`, `quiz.json` live beside lessons in repo but are **never scaffolded** |
| 10 | Mastery | Strict gates: theory → Socratic check → tests pass → rubric review → letter grade recorded. Fail → remediation. Force-skip recorded as `skipped`. Stage capstones + spaced review |
| 11 | Roadmap | 8 stages, ~90 lessons on Go path (see Curriculum), + focus packs |
| 12 | Focus packs | Registry-defined: `containers`, `web-services`, `cli-tooling` (authored), `ml` (stub). Freeform focuses = tutor-generated `custom` lessons, excluded from idempotent diffing |
| 13 | Launch scope | **Author everything now**: all Go-path stages + 3 packs (~112 lessons). Python = registry stub only |
| 14 | Guidance | `guidance` in state: `guided` / `standard` / `spartan`. Default `guided`; tutor proposes changes, only user changes it |
| 15 | Invocation | One skill `/tutor` + freeform args; LLM maps intent to deterministic script calls |
| 16 | Lesson IDs | Stable slugs (`go.basics.slices`); ordering computed; sync renames dirs safely (never clobbers content) |
| 17 | License | **AGPL-3.0-or-later on everything** + enforced CLA (sublicense rights enable qtsone's commercial dual-licensing) + §7 learner exception (learners' exercise solutions unencumbered) + explicit qtsone copyright |
| 18 | OSS hygiene | README, CONTRIBUTING → CLA, CLA CI check, SECURITY.md, CODE_OF_CONDUCT.md, issue forms, CODEOWNERS; CI validates registry schema / DAG acyclicity / content presence / Go solutions pass tests, and rejects machine-specific paths (lesson prose uses the example persona `ada`). Renovate manages GitHub Actions only. Local, uncommitted git hooks keep private names and machine paths out of files and commit messages |

## Architecture contract

### Repo layout
```
learning/
├── CONTEXT.md                  # this file
├── README.md, CONTRIBUTING.md, SECURITY.md, CODE_OF_CONDUCT.md
├── LICENSE, LICENSE-EXCEPTION.md, NOTICE
├── renovate.json               # GitHub Actions only: pins inside the curriculum are teaching material
├── .github/                    # CLA.md, CODEOWNERS, workflows/ci.yaml, issue forms, PR template
├── docs/                       # DESIGN.md, authoring-guide.md, curriculum-outline.md
├── tools/                      # authoring and review workflow scripts
└── skills/tutor/
    ├── SKILL.md                # tutor behavior: teaching protocol, intent → script map
    ├── scripts/tutor.py        # deterministic engine (stdlib only)
    └── curriculum/
        ├── registry.json       # the whole graph
        └── content/
            ├── shared/<stage>/<lesson>/     # theory shared; exercises per language
            ├── go/<stage>/<lesson>/
            ├── focus/<pack>/<lesson>/
            └── python/                      # stub (registry only)
```

### Lesson directory (in repo)
```
<lesson>/
├── LESSON.md          # theory, objectives, curated further-reading   [scaffolded]
├── exercise/          # starter code + _test.go verification          [scaffolded]
│                      # shared lessons use exercises/<lang>/ instead
├── TUTOR.md           # teaching notes, misconceptions, grilling pts  [never scaffolded]
├── quiz.json          # question bank + grading rubric                [never scaffolded]
└── solution/          # canonical solution (per-lang for shared)      [never scaffolded]
```
A lesson is "authored" iff `LESSON.md` exists (derived from filesystem, no status field).
Unauthored lessons appear in ROADMAP.md as *content pending*; scaffold skips them.

### Workspace layout (learner side)
```
workspace/
├── ROADMAP.md         # generated: ordered roadmap, checkboxes, grades
├── .tutor/
│   ├── state.json     # script-owned progress state
│   ├── manifest.json  # script-owned scaffold hashes (lesson-id keyed)
│   ├── journal.md     # LLM-owned session notes + curriculum observations
│   └── attic/         # lessons removed upstream (never deleted)
├── lessons/<NN>-<group>/<NN>-<lesson-slug>/
└── projects/          # capstones
```

### tutor.py subcommands (all state mutations go through these)
```
init <language> [--focus a,b]    # create workspace / add focuses; idempotent (implies sync)
sync                             # re-scaffold + JSON diff report (added/updated/conflicts/
                                 #   removed→attic/renamed/needs_review)
status [--json]                  # session briefing: progress, next lesson, needs_review
mark <lesson-id> <status|resolved> [--grade A..F] [--note ...]
                                 #   resolved: needs_review → the status held before the change
verify <lesson-id>               # run the lesson's tests (e.g. go test in exercise dir); counts an attempt
guidance <guided|standard|spartan>
custom add <slug> --title T      # register a tutor-generated custom lesson (excluded from sync)
graph [--language X] [--focus ..] [--format tree|mermaid|json]   # no workspace needed
validate [--strict]              # repo-level: registry schema, DAG order, content presence
ci [--filter substr]             # validate + run every solution against its tests
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
`cli-tooling` (~6), `ml` (stub). Python track: registry stub proving composition.

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
- [ ] Phase 4 — End-to-end validation: scaffold scratch workspace; exercise init/sync/resume/conflict/needs_review paths; run all solution tests
- [ ] Phase 5 — Symlink install (`~/.claude/skills/tutor`), final review, tag v0.1.0

Conventions for resuming a build session: read this file, `git log --oneline -10`,
then continue the first unchecked phase. Commit at phase boundaries to `main` with
conventional-commit messages. Content authoring commits per stage.
