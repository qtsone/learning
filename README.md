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

```sh
git clone https://github.com/qtsone/learning.git
cd learning
ln -s "$PWD/skills/tutor" ~/.claude/skills/tutor
```

Then, in any empty directory, start a Claude session and run:

```
/tutor go                       # the full Go path
/tutor go --focus containers    # Go path + the containers focus pack
```

Resume anytime from the same directory with plain `/tutor` — it reads your state and
picks up exactly where you left off.

## Roadmap

Eight stages, ~90 lessons on the Go path. Shared stages are language-agnostic;
exercises are done in your target language.

| Stage | Pool | ~Lessons | Covers |
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

- `containers` (~8) — Docker, multi-stage builds, Kubernetes, deploying Go
- `web-services` (~8) — auth, realtime, API hardening, background jobs
- `cli-tooling` (~6) — flags/config, Cobra, terminal UX, distribution
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
```

## License

Licensed under **AGPL-3.0-or-later** (see [LICENSE](LICENSE)), with a Learner
Exception ([LICENSE-EXCEPTION.md](LICENSE-EXCEPTION.md)): your solutions to the
exercises are yours, unencumbered by the AGPL.

Contributions require signing the [CLA](.github/CLA.md). The CLA grants qtsone
sublicensing rights, which is how qtsone dual-licenses the curriculum commercially
while every public copy and derivative stays open.
