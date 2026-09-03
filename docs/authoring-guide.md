# Lesson Authoring Guide

The exemplar is `skills/tutor/curriculum/content/go/s1-basics/hello-world/` —
match its anatomy, tone, and rigor. `docs/curriculum-outline.md` is canonical
for ids/titles/ordering; `registry.json` for objectives/duration/verify.

## Lesson directory anatomy

```
<lesson>/
├── LESSON.md            # scaffolded to the learner
├── exercise/            # scaffolded — language-agnostic form
│   └── …                #   (or exercises/<lang>/ in shared lessons whose
│                        #    exercise is language-specific)
├── TUTOR.md             # tutor-only (never scaffolded)
├── quiz.json            # tutor-only
└── solution/            # tutor-only — overlay of only the files that change
                         #   (solutions/<lang>/ for shared language-specific)
```

## LESSON.md

Structure (see exemplar):

1. `# <Title>` then the meta line: `> \`<id>\` · ~<duration> · Stage: <stage title>`
2. `## Objectives` — the registry objectives, phrased "By the end you can …".
3. Theory sections — short, concrete headings; build one concept at a time.
4. `## Exercise` — what to build, explicit **acceptance criteria** (numbered),
   how to run the checks.
5. `## Further reading` — 2-4 curated links (up to 6 when a lesson spans
   several tools or specifications, as pack lessons often do), official
   sources strongly preferred (go.dev, pkg.go.dev, spec, canonical blog
   posts).

Tone and craft:

- Second person, plain language, no hype, no emoji. Explain *why*, not just
  *what*. Anticipate the reader's "wait, but…" questions and answer them.
- Assume exactly the knowledge of the preceding roadmap — reference earlier
  lessons ("remember S0's compiler model") and preview later ones by stage,
  never by leaking their content.
- Length: a lesson is as long as its job, and its job grows through the
  roadmap. Bands, by what the lesson has to carry:

  | Where | Lines | Why |
  |-------|-------|-----|
  | S0-S1 foundations & basics | 90-150 | one new idea at a time, nothing assumed |
  | Code-first stages (S2-S4) | 120-250 | the exercise carries the teaching |
  | S5 advanced Go | 250-350 | systems theory the code cannot show, plus production-scale acceptance criteria |
  | S6 systems & design | ≤350 | the theory *is* the lesson; the exercise is a worksheet, not a compiler |
  | S7 expert capstone | 320-380 | theory, a grading contract for a project the author cannot see, and two sets of criteria (mechanical + review) |
  | Focus packs | ≤400 | a pack compresses a specialist domain into 6-8 lessons, so each carries context a stage lesson would borrow from its neighbours |

  **Hard ceiling ~450 lines, anywhere.** Only a lesson that walks complete
  worked designs end to end earns it (`shared.systems.case-studies`), because
  half a worked design teaches worse than a long one. Past your band: split
  or cut. Dense beats long everywhere — length is never the goal, and a
  lesson that hits its band by padding has failed at both.
- Go accuracy bar: Go 1.22+ idioms — `log/slog`, generics where natural,
  1.22 `net/http` mux patterns, `errors.Is/As`, no `ioutil`, no deprecated
  APIs. When in doubt, check current docs.
- Shared-pool lessons (S0/S2/S4/S6): theory must stay language-portable.
  Pseudocode and diagrams are fine; concrete snippets go in clearly marked
  `In Go:` blocks (a future Python track adds `In Python:` blocks additively).

## exercise/

- Go lessons: a self-contained module — `go.mod` with
  `module tutor.local/<slug>` and `go 1.22`. Starter code **must compile** but
  tests **must fail** (learners fight the problem, not the scaffolding).
  Mark work sites with `// TODO:` comments.
- Tests (`*_test.go`) are the specification: table-driven where natural,
  failure messages that teach (`got %q, want %q`). Test the acceptance
  criteria exactly — no hidden requirements.
- `script`-verify lessons: `exercise/check.sh` — bash, self-contained,
  idempotent, exits non-zero with a clear "what to fix" message per failed
  check. The learner runs it repeatedly until green.
- `discussion`-verify lessons: exercise dir optional; if present it's an
  exploration task the tutor reviews in conversation.
- Shared lessons whose exercise is language-specific use `exercises/go/…`
  (same content rules); language-agnostic ones (terminal, git) use plain
  `exercise/`.

## solution/

Only for test-verified lessons. Contains **only the files that differ** from
the starter (it is overlaid onto `exercise/` at CI time). Must be idiomatic,
gofmt-clean, and the code you'd defend in review — it is also the tutor's
reference during grading.

## TUTOR.md

Sections, in order (see exemplar): `## Where the learner is`,
`## Common misconceptions`, `## Grilling points`, `## Grading rubric`
(A/B/C/Fail, exercise-specific), `## Remediation ladder` (3-4 escalating
hints, never jumping to the answer), `## After passing` (one-line preview of
what's next).

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
need a core question of their own, as most of S5 does. Coverage sets the count,
not the reverse: together the core questions must cover every registry
objective. Prompts are conversation starters, not exam prose.

## Self-check before you're done (all must pass)

```sh
cd exercise[|s/go] && go test ./... ; cd -   # must FAIL on the starter
gofmt -l <lesson-dir>                        # no output
python3 -m json.tool <lesson>/quiz.json      # parses
python3 skills/tutor/scripts/tutor.py ci --filter <lesson-id>   # solution passes
```

(`ci` also runs `validate`, which checks TUTOR.md/quiz.json presence.)

### Extra check: S7 capstone reference projects

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
