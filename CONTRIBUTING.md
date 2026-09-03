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
  **Curriculum issue** and give the lesson id (`go.basics.slices`).
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
        ├── shared/<stage>/<lesson>/    # language-agnostic theory; exercises per language
        ├── go/<stage>/<lesson>/
        ├── focus/<pack>/<lesson>/
        └── python/                     # stub (registry only)
```

`registry.json` declares every lesson id, its prerequisites, and stage/pack
composition. Content pools hold the actual lesson directories. A lesson is
"authored" iff its `LESSON.md` exists — there is no status field to maintain.

## Authoring a lesson

Read [`docs/authoring-guide.md`](docs/authoring-guide.md) before writing content. It
holds the file-by-file anatomy, the tone and length bands, and the self-checks. The
exemplar to copy is `skills/tutor/curriculum/content/go/s1-basics/hello-world/`.

### Lesson directory anatomy

```
<lesson>/
├── LESSON.md          # theory, objectives, exercise brief, further reading   [scaffolded]
├── exercise/          # starter code + tests, or check.sh                     [scaffolded]
│                      # shared lessons with a language-specific exercise use
│                      # exercises/<lang>/ instead (scaffolded as exercise/)
├── TUTOR.md           # teaching notes, misconceptions, rubric, hint ladder   [never scaffolded]
├── quiz.json          # question bank + pass rule                             [never scaffolded]
└── solution/          # reference solution, overlaid onto exercise/ by ci     [never scaffolded]
                       # shared lessons use solutions/<lang>/ instead
```

Only `LESSON.md` and the exercise reach the learner's workspace. `TUTOR.md`,
`quiz.json`, and `solution/` are tutor-only: the skill reads them from this repo to
teach, quiz, and review — they must never be scaffolded. Solutions must actually
pass the exercise tests under `go test -race` (`ci` runs them).

### IDs and ordering

[`docs/curriculum-outline.md`](docs/curriculum-outline.md) is canonical for lesson
ids and ordering. The registry and content directories are derived from it; if they
disagree, the outline wins and the others get fixed. IDs are stable slugs
(`go.basics.slices`) — never rename one. Ordering is positional in the registry's
stage lists, so inserting lessons is safe.

## Validation

Before opening a PR:

```sh
python3 skills/tutor/scripts/tutor.py validate   # registry schema + DAG acyclicity + content presence
python3 skills/tutor/scripts/tutor.py ci         # validate + run every solution against its tests
```

CI runs `ci`; a PR that fails it will not merge.

CI also rejects machine-specific home-directory paths. Lesson prose uses the example
persona `ada` (`/home/ada/...`) wherever a path is needed, never a real username.

## Commits

Use [Conventional Commits](https://www.conventionalcommits.org/). Types in use:
`feat`, `fix(engine)`, `docs`, `ci`, `chore(deps)`, and `content(<stage or pack>)`
for curriculum changes.

Releases are cut from `main` automatically by semantic-release: `feat` bumps the
minor version, `fix` the patch, and a `BREAKING CHANGE:` footer the major. Other
types do not trigger a release on their own and ship with the next one.

## Curriculum feedback from learners

You don't have to author content to improve it. While teaching, the tutor logs
structured curriculum observations (gaps, errata, difficulty spikes) to the
learner's journal; `/tutor contribute` aggregates them into a branch and opens a PR
against this repo. Those PRs are triaged like any other contribution — the journal
entries are the evidence.
