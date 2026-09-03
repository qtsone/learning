# tutor

An open-source curriculum engine plus a Claude skill that turns an agent into a
0-to-expert programming tutor. The full curriculum is pre-authored and versioned in
this repo; a deterministic script scaffolds it into any workspace and tracks your
progress. First language: **Go**. Multi-language by design — shared stages
(foundations, CS, engineering practice, systems) compose with per-language tracks
and focus packs.

## How it works

1. The curriculum lives here as real files: lessons, exercises with tests, quizzes, solutions.
2. `tutor.py` deterministically scaffolds the lesson graph into any folder you choose.
3. The tutor skill guides you through it with strict mastery gates: theory, Socratic
   check-in, your code passing the tests, a graded code review — no gate, no advance.
4. Progress lives in script-owned state; re-running is idempotent, so you can resume
   anytime and `sync` pulls upstream curriculum updates without clobbering your work.
5. Solutions and teaching notes stay tutor-only — you never see them before you pass.

## Quickstart

You need Python 3.8 or newer, git, and the Go toolchain (1.22 or newer) for the Go
path. The engine has no other dependencies.

```sh
git clone https://github.com/qtsone/learning.git
cd learning
mkdir -p ~/.claude/skills
ln -s "$PWD/skills/tutor" ~/.claude/skills/tutor
```

Then, in any empty directory, start a Claude session and run:

```
/tutor go                       # the full Go path
/tutor go --focus containers    # Go path + the containers focus pack
```

Resume anytime from the same directory with plain `/tutor` — it reads your state and
picks up exactly where you left off. Because the skill is a symlink into your clone,
`git pull` here is how you receive curriculum updates; the next `/tutor` session
syncs them into your workspace.

## The engine

`skills/tutor/scripts/tutor.py` is a single Python 3 script, standard library only.
The skill drives it; you can call it directly. Every subcommand acts on the current
directory (or `--workspace DIR`) and prints JSON wherever a program might read the
result.

| Subcommand | What it does |
|------------|--------------|
| `init <language> [--focus a,b]` | Create the workspace, or add focus packs to an existing one, then sync. Refuses to run inside this repo. |
| `sync` | Re-scaffold from the curriculum and print a diff report: `added`, `updated`, `conflicts`, `renamed`, `removed`, `removed_files`, `needs_review`, `pending_content`. |
| `status [--json]` | Session briefing: progress counts, lessons needing review, open conflicts, the next lesson and its directory. Flags `sync_needed` when the registry changed since the last sync. |
| `mark <lesson-id> <status\|resolved> [--grade A..F] [--note ...]` | Set a lesson to `todo`, `in_progress`, `passed`, `skipped`, or `needs_review`. `resolved` returns a reviewed lesson to the status it held before the change. |
| `verify <lesson-id>` | Run the lesson's checks in its exercise directory (`go test -race ./...` or `bash ./check.sh`), record an attempt, and exit with the check's status. Discussion lessons have no automated check. |
| `guidance <guided\|standard\|spartan>` | Set how much hand-holding the tutor gives. |
| `custom add <slug> --title T` | Register a tutor-authored lesson under `lessons/90-custom/`; custom lessons are outside sync. |
| `graph [--language X] [--focus a,b] [--format tree\|mermaid\|json]` | Show the language matrix, or the composed roadmap for one language. Needs no workspace. |
| `validate [--strict]` | Repo check: registry schema, prerequisite order, content presence, no stray build artifacts. |
| `ci [--filter substr]` | `validate`, then run every solution against its exercise tests. |

### Workspace layout

```
workspace/
├── ROADMAP.md                     # generated: ordered roadmap, checkboxes, grades
├── lessons/<NN>-<group>/<NN>-<slug>/
│   ├── LESSON.md                  # theory, objectives, exercise brief, further reading
│   └── exercise/                  # starter code and the tests that define "done"
├── lessons/90-custom/<slug>/      # tutor-authored custom lessons
├── projects/                      # capstones
└── .tutor/
    ├── state.json                 # progress; written only by tutor.py
    ├── manifest.json              # scaffold hashes, keyed by lesson id
    ├── journal.md                 # the tutor's session notes and curriculum observations
    └── attic/                     # lessons removed upstream, parked here, never deleted
```

Sync never overwrites a file you changed. If upstream changed it too, the new
version lands beside yours as `<file>.upstream` and the report lists a conflict for
the tutor to walk you through. A lesson you already passed that changes upstream
flips to `needs_review`, and the tutor covers the delta before you move on.
Tutor-only files (`TUTOR.md`, `quiz.json`, `solution/`) are never scaffolded.

## Roadmap

Eight stages, 90 lessons on the Go path, plus 22 across three focus packs. Shared
stages are language-agnostic; exercises are done in your target language.

| Stage | Pool | Lessons | Covers |
|-------|------|---------|--------|
| S0 Foundations | shared | 8 | what programs are, terminal, git, editors, reading docs |
| S1 Programming basics | go | 15 | syntax, types, control flow, functions, pointers, errors, testing |
| S2 CS fundamentals | shared | 12 | data structures, algorithms, Big-O, recursion |
| S3 Intermediate | go | 15 | interfaces, generics, stdlib, concurrency, tooling |
| S4 Engineering practice | shared | 10 | clean code, TDD, debugging, SQL, security, CI |
| S5 Advanced | go | 12 | runtime/GC/scheduler, memory model, profiling, servers, gRPC |
| S6 Systems & design | shared | 12 | system design, distributed systems, networking, observability |
| S7 Expert capstone | go | 6 | production project, performance engineering, OSS contribution |

Focus packs slot into the path where their prerequisites are met:

- `containers` (8) — Docker, multi-stage builds, Kubernetes, deploying Go; three lessons after S4, five after S5
- `web-services` (8) — auth, realtime, API hardening, background jobs; after S5
- `cli-tooling` (6) — flags/config, Cobra, terminal UX, distribution; after S3
- `ml` — stub, Python-only, no content yet

The canonical lesson list is [`docs/curriculum-outline.md`](docs/curriculum-outline.md).

## Supported languages

| Language | Status |
|----------|--------|
| Go | available (full path + 3 focus packs) |
| Python | registry stub — graph only, content pending |

To see the full lesson matrix for any track:

```sh
python3 skills/tutor/scripts/tutor.py graph
python3 skills/tutor/scripts/tutor.py graph --language go --focus containers
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the contribution flow and
[`docs/authoring-guide.md`](docs/authoring-guide.md) for lesson anatomy and the
authoring bar. The tutor itself logs curriculum observations while teaching, and
`/tutor contribute` turns them into a pull request against this repo.

## License

Licensed under **AGPL-3.0-or-later** (see [LICENSE](LICENSE)), with a Learner
Exception ([LICENSE-EXCEPTION.md](LICENSE-EXCEPTION.md)): your solutions to the
exercises are yours, unencumbered by the AGPL.

Contributions require signing the [CLA](.github/CLA.md). The CLA grants qtsone
sublicensing rights, which is how qtsone dual-licenses the curriculum commercially
while every public copy and derivative stays open.
