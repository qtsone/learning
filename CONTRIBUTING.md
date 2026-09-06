# Contributing

## CLA first

All contributions require a signed Contributor License Agreement:
[.github/CLA.md](.github/CLA.md). It grants qtsone sublicensing rights (this is what
makes commercial dual-licensing possible while the repo stays AGPL). Signing happens
via a PR check — open your PR and follow the bot's instructions; nothing merges
without a signature on record.

## Code of conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). The audience
includes people writing their first program; keep that in mind in reviews and issues.

## Reporting problems

- A lesson, exercise, test, quiz, or solution that is wrong or unclear: open a
  **Curriculum issue** and give the lesson id (`go.basics.arrays-slices`).
- `tutor.py` or the `/tutor` skill misbehaving: open an **Engine or skill bug**.
- Security problems: see [SECURITY.md](SECURITY.md); do not open a public issue.

## Repo structure

```
skills/tutor/
├── SKILL.md                     # tutor behavior: teaching protocol, intent → script map
├── scripts/tutor.py             # deterministic engine (Python 3 stdlib only, zero deps)
└── curriculum/
    ├── registry.json            # the whole graph: lessons, prereq DAG, tracks, packs
    └── content/
        ├── shared/<stage>/<lesson>/        # language-neutral theory (with anchors), TUTOR.md, quiz.json
        ├── focus/<pack>/<lesson>/          # same shape, one pack per directory
        ├── go/
        │   ├── <stage>/<lesson>/           # Go stages (s1-basics … s7-capstone)
        │   ├── shared/<stage>/<lesson>/    # Go overlay for a shared lesson
        │   └── focus/<pack>/<lesson>/      # Go overlay for a pack lesson
        └── python/
            ├── <stage>/<lesson>/           # Python stages (p1-basics … p7-capstone); registry only until the track opens
            ├── shared/<stage>/<lesson>/    # Python overlay for a shared lesson
            └── focus/<pack>/<lesson>/      # Python overlay for a pack lesson
```

`registry.json` declares every lesson id, its prerequisites, and stage/pack
composition. Content pools hold the actual lesson directories; a language's
overlays for shared and pack lessons live under `content/<lang>/shared/` and
`content/<lang>/focus/`, so adding a language touches nothing under `shared/`,
`focus/`, or another language's folder. A lesson is "authored" for a language
iff its `LESSON.md` exists and the exercise its verify type needs is on disk for
that language — there is no status field to maintain.

## Authoring a lesson

Read [`docs/authoring-guide.md`](docs/authoring-guide.md) before writing content. It
holds the file-by-file anatomy, the tone and length bands, and the self-checks. The
exemplars to copy are `skills/tutor/curriculum/content/go/s1-basics/hello-world/`
for Go and `skills/tutor/curriculum/content/python/p1-basics/hello-python/` for
Python; the exemplar overlay for a shared lesson is
`skills/tutor/curriculum/content/python/shared/s2-cs/arrays-linked-lists/`. The
Python ones land with the Python track.

### Lesson directory anatomy

```
<lesson>/                  # a language-stage lesson, or the neutral part of a shared/pack lesson
├── LESSON.md              # theory, objectives, exercise brief, further reading   [scaffolded, rendered per language]
├── exercise/              # starter code + tests, or check.sh                     [scaffolded]
│                          # in shared/pack lessons only when it is language-agnostic
├── TUTOR.md               # teaching notes, misconceptions, rubric, hint ladder   [never scaffolded]
├── quiz.json              # question bank + pass rule                             [never scaffolded]
└── solution/              # reference solution, overlaid onto exercise/ by ci     [never scaffolded]

content/<lang>/shared/<stage>/<lesson>/   # language overlay for a shared lesson (focus/<pack>/ for packs)
├── snippets/<slot>.md     # one per `<!-- lang: <slot> -->` anchor in LESSON.md   [rendered into LESSON.md]
├── exercise/              # starter + tests + README.md brief; wins file by file   [scaffolded as exercise/]
├── solution/              # overlay for ci                                        [never scaffolded]
├── TUTOR.md               # language-specific notes, additive to the shared one   [never scaffolded]
└── quiz.json              # questions merged by id over the shared quiz.json      [never scaffolded]
```

Only `LESSON.md` (rendered for the workspace language) and the effective exercise
reach the learner's workspace. `TUTOR.md`, `quiz.json`, `solution/`, and
`snippets/` are tutor-only: the skill reads them from this repo to teach, quiz,
and review — they must never be scaffolded as files. Solutions must actually pass
the exercise tests under the lesson's runner, for each language that has a
variant (`ci` runs them).

### IDs and ordering

[`docs/curriculum-outline.md`](docs/curriculum-outline.md) is canonical for lesson
ids and ordering. The registry and content directories are derived from it; if they
disagree, the outline wins and the others get fixed. IDs are stable slugs
(`go.basics.arrays-slices`) — never rename one. Ordering is positional in the registry's
stage lists, so inserting lessons is safe.

## Validation

Before opening a PR:

```sh
python3 skills/tutor/scripts/tutor.py validate --strict   # schema, DAG, verify-type invariants, anchors; completeness warnings become errors
python3 skills/tutor/scripts/tutor.py ci --language go    # strict validate + every Go solution against its tests (python, or omit for all)
python3 -m unittest discover -s tests -v                  # engine changes: the end-to-end engine tests
```

CI runs `validate --strict` and the engine tests on every PR, and `ci --language`
for each language the PR touches; a PR that fails them will not merge.

CI also rejects machine-specific home-directory paths. Lesson prose uses the example
persona `ada` (`/home/ada/...`) wherever a path is needed, never a real username.

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/). Types in use:
`feat`, `fix(engine)`, `docs`, `ci`, `chore(deps)`, and `content(<stage or pack>)`
for curriculum changes — `content(s2)`, `content(p1)`, `content(shared)` for the
shared-pool batch, `content(containers)`; `content(deps)` for a manual sweep of
the version pins inside exercises.

Releases are cut from `main` automatically by semantic-release: `feat` bumps the
minor version, `fix` the patch, and a `BREAKING CHANGE:` footer the major. Other
types do not trigger a release on their own and ship with the next one.

The registry `version` is CalVer (`YYYY.MM.N`) and bumps on any change to a
composed path — a lesson added, removed, renamed, or reordered on a language's
path or in a pack. Opening a language is a `feat` (minor bump). Lessons land as
`content(...)` commits and ship with the next release; on their own they bump
neither the registry version nor the release.

## Curriculum feedback from learners

You don't have to author content to improve it. While teaching, the tutor logs
structured curriculum observations (gaps, errata, difficulty spikes) to the
learner's journal; `/tutor contribute` aggregates them into a branch and opens a PR
against this repo. Those PRs are triaged like any other contribution — the journal
entries are the evidence.
