# DESIGN — Architecture & Rationale

`CONTEXT.md` is the project contract: the decisions, the layout, the build state.
This document explains **why** those decisions were made. If the two ever disagree
on a fact, `CONTEXT.md` wins; if you want to understand a decision before changing
it, read this first.

## What this is

`tutor` is a curriculum engine split into two halves with a hard boundary between them:

- **A deterministic engine** — `skills/tutor/scripts/tutor.py` (Python 3, stdlib only)
  that scaffolds pre-authored content into a learner workspace and owns all progress
  state.
- **An LLM behavior layer** — `skills/tutor/SKILL.md`, which tells a Claude agent how
  to teach: run `status` first, gate mastery, grill Socratically, and translate freeform
  learner intent into engine subcommands.

Everything below is a consequence of keeping that boundary crisp: the script does
everything that must be reproducible; the LLM does everything that must be adaptive.

## Why fully pre-authored content

Lessons are real files in this repo (`skills/tutor/curriculum/content/`), not prompts
that ask an LLM to generate a lesson at scaffold time. This was the foundational choice,
for two reasons:

1. **Idempotency.** Scaffolding is a pure function of repo content and the workspace
   language: `init` and `sync` copy files (rendering `LESSON.md` on the way, see
   Per-language rendering below), so running them twice produces the same workspace.
   An LLM generating content at scaffold time would produce a different curriculum for
   every learner and every run — impossible to test, review, version, or reason about.
   With pre-authored content, quality is fixed in the repo by an author/review loop (and
   CI), not left to inference-time luck.
2. **Hash-diffable change detection.** Because every scaffolded file has exactly one
   upstream source, `sync` can record its hash in `manifest.json` and later distinguish
   with certainty: *upstream changed*, *learner changed it*, *both changed*. That single
   property is what makes safe re-scaffolding (see Sync below) possible at all. Generated
   content has no stable upstream to diff against.

The one deliberate exception: freeform focus areas the registry doesn't cover become
tutor-generated `custom` lessons, tracked in `state.json` under `custom_lessons` and
**excluded** from idempotent diffing — they have no upstream, so they must not
participate in the hash contract.

## Why a central registry + content pools

The graph lives in one file — `skills/tutor/curriculum/registry.json` — declaring every
lesson id, its prerequisites (a DAG), stage membership, tracks, and focus packs. Content
lives in pools: `content/shared/`, `content/go/`, `content/focus/`, `content/python/`.

The driver is **multi-language composition without duplication**. Roughly half the
curriculum (S0 foundations, S2 CS fundamentals, S4 engineering practice, S6 systems &
design — see `docs/curriculum-outline.md`) is language-agnostic and written once in
`shared/`. Adding a language track is a bounded list, and none of it touches `shared/`,
`go/` or `focus/`:

- a `languages.<code>` registry entry with its runner (`verify.type`) and its path (which
  shared stages it takes, in which order, around its own stages);
- its stage lessons under `content/<code>/<stage>/<lesson>/`;
- one overlay folder under `content/<code>/shared/<stage>/<lesson>/` (and, once packs are
  ported, `content/<code>/focus/<pack>/<lesson>/`) for every shared or pack lesson it takes
  that needs an exercise, a solution or language-specific tutor notes;
- snippets at the anchors where the shared theory leaves room for a language to speak;
- a row in the CI `solutions` matrix.

No theory is forked, so an errata fix in a shared lesson reaches every track at once.

Splitting graph from content also gives each side the right change-review granularity:
reordering the Go track or moving a pack's insertion point is a registry-only diff;
fixing a lesson is a content-only diff. Lesson **ids are stable slugs**
(`go.basics.arrays-slices`) and ordering is computed from registry position — so
reordering never renames an id, and `sync` can rename workspace directories safely
because `manifest.json` is keyed by lesson id, not by path.

Focus packs (`containers`, `web-services`, `cli-tooling`) are the same mechanism at a
smaller scale: registry-declared lesson sets with insertion points where prerequisites
are met, composable onto any track instead of being baked into one.

### Per-language rendering

A shared lesson's `LESSON.md` is one neutral file with `<!-- lang: <slot> -->` anchors
where a language may add a paragraph or a code block; each language keeps its blocks as
`snippets/<slot>.md` in its own overlay folder under `content/<lang>/`, beside that
language's exercise and solution for the lesson. Overlays live under the language, not
under the shared lesson, because that is what keeps adding a language a pure addition: a
Python contributor creates files under `content/python/` and never edits a file Go learners
are scaffolded from, so Go's rendered bytes stay identical and no Go learner gets a
`needs_review` for a change that was never about Go. The alternatives both fail this test:
a `LESSON.<lang>.md` per language forks the theory this section exists to keep single, and
fenced per-language blocks inside the shared file make every language edit the same 42
files.

`LESSON.md` is rendered at scaffold time rather than stored per language for the same
reason content is pre-authored: one upstream source per scaffolded file. The engine splices
the workspace language's snippets in at the anchors (an anchor with no snippet vanishes
together with the blank line after it, so paragraph spacing and every other language's
rendered bytes are unchanged) and hashes the rendered bytes into `manifest.json`, so `sync`
still knows exactly what it wrote. A stored per-language copy would be a second source of
truth that drifts from the neutral text; a rendering is a pure function of the two.

## Why script-owned state

`state.json` and `manifest.json` (in the workspace's `.tutor/`) are mutated **only** by
`tutor.py` subcommands — `mark`, `sync`, `guidance`, etc. The LLM never edits them by
hand. Two reasons:

1. **Schema safety.** LLMs are excellent at intent and unreliable at byte-exact JSON
   surgery. A single malformed edit to progress state corrupts the learner's history
   silently. Funneling every mutation through subcommands means one code path validates,
   one code path writes, and `state.json` carries a `schema` version so future engine
   versions can migrate deterministically.
2. **Deterministic resume.** A tutoring relationship spans weeks of sessions across
   different agent instances with no shared memory. The only durable truth is state on
   disk. Because only the script writes it, `status` at session start reconstructs
   exactly where the learner is — progress, next lesson, pending `needs_review` — with
   no dependence on what a previous LLM session remembered or hallucinated.

The LLM still needs somewhere to write, so the state is split by owner:
`.tutor/journal.md` is **LLM-owned** free-form prose (session notes, learner struggles,
curriculum observations). The script never parses it; the LLM never bypasses it. Each
side has one file it fully owns, and neither can corrupt the other's.

## Sync: the ownership rule and `needs_review`

`sync` re-scaffolds against a workspace the learner has been living in. The per-file
rule (full table in `CONTEXT.md` §Sync semantics) reduces to one principle:
**the engine may only overwrite what it can prove it still owns.**

- Workspace file pristine (hash matches `manifest.json`) → the engine still owns it →
  update in place.
- Learner modified it and upstream also changed → the learner owns it now → never
  clobber; write a `<file>.upstream` sidecar and report a conflict for the tutor and
  learner to resolve together.
- Lesson removed upstream → move to `.tutor/attic/`, never delete. Learner work is
  irreplaceable; repo content is not.

**`needs_review` propagation** closes the pedagogical gap that pure file-sync would
leave: if a lesson the learner already `passed` (or `skipped`) changes upstream, the
files may update cleanly but the learner's *knowledge* is now stale — they were graded
against the old material. So `sync` flips the lesson's status to `needs_review`, and
`SKILL.md` obliges the tutor to cover the delta before the learner advances past it.
A mastery gate that could be invalidated silently by a content update wouldn't be a
gate.

## Guidance modes

`state.json` carries a `guidance` field: `guided` / `standard` / `spartan`. It exists
because the distance from hand-holding to expert is precisely the axis this curriculum
travels — the scaffolding that helps a beginner (step-by-step nudges, early hints)
actively harms an advanced learner who needs to struggle productively.

Modes are explicit state rather than LLM vibes so behavior is consistent across
sessions, and the **tutor may propose a change but only the user changes it** — how
much help to receive is a learner-autonomy decision, and an LLM silently dialing
difficulty up or down would erode trust in the gates' meaning.

## Licensing: the deterrence model

Three instruments (see `LICENSE`, `LICENSE-EXCEPTION.md`, `.github/CLA.md`, `NOTICE`)
form one mechanism:

- **AGPL-3.0-or-later on everything** — code *and* content. A curriculum's natural
  competitor is a closed SaaS wrapper; plain GPL wouldn't reach network use, and
  permissive licensing would invite proprietary forks. AGPL makes the closed-competitor
  path structurally impossible: anyone serving this curriculum, even over a network,
  must publish their derivative in full.
- **CLA with sublicense rights** (enforced by CI on every PR) — every contribution
  arrives with the right for qtsone to relicense it. This is what lets qtsone offer
  the work under separate commercial terms: it dual-licenses the aggregate to itself.
  Without the CLA, the first outside contribution would freeze the content as
  AGPL-only and cut off the commercial channel that funds authoring.
- **Learner Exception (AGPL §7 additional permission)** — learners' exercise solutions
  start from scaffolded starter code, which makes them arguably derivative works. Left
  raw, AGPL would encumber every learner's own practice code — absurd and hostile.
  The §7 grant carves learners' solutions out entirely: use them anywhere, any license,
  no strings.

Net effect: closed competitors are impossible, learners are unencumbered, contributors'
work stays public forever, and qtsone retains exactly one privileged commercial path.
The asymmetry is the point, and the CLA makes it explicit at contribution time rather
than discovered later.

## The improvement loop

The tutor is the best-placed observer of curriculum defects — it watches real learners
hit real walls. That signal is captured, but never auto-published:

1. During sessions, the tutor logs structured observations to `.tutor/journal.md`:
   `- [<lesson-id>] <issue|gap|errata|difficulty> — <observation> — suggested: <fix>`.
2. When the learner runs `/tutor contribute`, the skill aggregates journal entries into
   concrete content changes, creates a branch, and opens a PR against this repo.

**Never automatic** — the journal may contain the learner's mistakes and context, PRs
are contributions requiring human intent (and a CLA signature), and upstream quality is
protected by the same review + CI gate (`.github/workflows/ci.yaml`: registry/DAG/content
validation, then every authored solution run against its exercise tests) as any other
change. The loop makes the curriculum self-correcting at the speed of its learners
while keeping a human hand on every merge.

## File map

| Path | Role |
|------|------|
| `CONTEXT.md` | Contract, decision log, build state |
| `docs/curriculum-outline.md` | Canonical lesson list (wins over registry/content) |
| `skills/tutor/SKILL.md` | LLM behavior: teaching protocol, intent → script map |
| `skills/tutor/scripts/tutor.py` | Deterministic engine, sole writer of state |
| `skills/tutor/curriculum/registry.json` | The graph: lessons, prereq DAG, tracks, packs |
| `skills/tutor/curriculum/content/` | Pre-authored pools: `shared/` and `focus/` (neutral), `go/` and `python/` (language stages plus that language's overlays for shared and pack lessons) |
| `tests/` | `test_engine.py`: black-box engine tests against a synthetic curriculum (stdlib `unittest`) |
| `.github/workflows/ci.yaml` | validate (+ engine tests) → solutions matrix per language → CLA gate |
| `LICENSE`, `LICENSE-EXCEPTION.md`, `.github/CLA.md`, `NOTICE` | The licensing mechanism |
