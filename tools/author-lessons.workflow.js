// Claude Code Workflow script: authors or completes a batch of curriculum lessons.
// Unlike author-stage.workflow.js this runs the Author phase only, in small
// batches, so an interrupted run loses at most one batch. Review/fix comes
// afterwards from review-stage.workflow.js once every lesson exists.
//
// Invoke with args: { repo, stageTitle, dir, pool, order, lessons, context, language }
//   pool     "shared" | "focus" | a language code ("go", "python") for a language stage
//   order    ordered lesson ids of the whole stage (for "assume only earlier material")
//   lessons  [{ id, mode, verify }] where mode is "author" (new), "finish" (partial on disk) or
//            "variant" (a <language> overlay for an existing shared or focus lesson), and verify is
//            the lesson's registry verify type ("tests" | "script" | "discussion") — required for
//            "variant", where it decides the deliverables; optional otherwise (sharpens the ci gate)
//   language language code the agents write for (default "go"); selects the exemplar and
//            toolchain and, for a shared or focus lesson, the overlay under content/<language>/<dir>/
export const meta = {
  name: 'tutor-author-lessons',
  description: 'Author or complete a batch of curriculum lessons (author phase only)',
  phases: [{ title: 'Author', detail: 'one agent per lesson' }],
}

const LANG = {
  go:     { exemplar: 'go/s1-basics/hello-world', bar: 'Go 1.22+ idioms (log/slog, generics where natural, errors.Is/As, no ioutil, no deprecated APIs)',
            fmt: 'gofmt -l <dir> (no output)', lint: 'go vet ./...', test: 'go test -race ./...',
            collect: 'go build ./... && go vet ./...', marker: '// TODO:', snippet: 'In Go:', runner: 'gotest' },
  python: { exemplar: 'python/p1-basics/hello-python', bar: 'Python 3.14+ idioms (checked against uv 0.12, ruff 0.16, pytest 9, mypy 2)',
            fmt: 'ruff format --check <dir>', lint: 'ruff check <dir>', test: 'uv run -q --locked pytest -q',
            collect: 'uv run -q --locked pytest --collect-only -q', marker: '# TODO:', snippet: 'In Python:', runner: 'pytest' },
}

const { repo: REPO, stageTitle, dir, pool, order, lessons, context, language = 'go' } = args
const L = LANG[language]
if (!L) throw new Error(`unknown language "${language}"; known: ${Object.keys(LANG).join(', ')}`)
const OVERLAY_POOL = pool === 'shared' || pool === 'focus'
if (!OVERLAY_POOL && pool !== language) throw new Error(`pool "${pool}" is a language stage; author it with language "${pool}"`)
const VERIFY_TYPES = ['tests', 'script', 'discussion']
for (const l of lessons) {
  if (!['author', 'finish', 'variant'].includes(l.mode)) throw new Error(`${l.id}: unknown mode "${l.mode}"`)
  if (l.mode === 'variant' && !OVERLAY_POOL) throw new Error(`${l.id}: mode "variant" needs a shared or focus pool, got "${pool}"`)
  if (l.mode === 'variant' && !VERIFY_TYPES.includes(l.verify)) throw new Error(`${l.id}: mode "variant" needs verify (${VERIFY_TYPES.join(' | ')}, the registry's verify.type), got "${l.verify}"`)
}
const CONTENT = `${REPO}/skills/tutor/curriculum/content`
const EXEMPLAR = `${CONTENT}/${L.exemplar}`
const OVERLAY_EXEMPLAR = `${CONTENT}/${language}/shared/s2-cs/arrays-linked-lists`
const TUTOR = `python3 ${REPO}/skills/tutor/scripts/tutor.py`
const slugOf = (lid) => lid.split('.').pop()
const lessonDir = (lid) => `${CONTENT}/${dir}/${slugOf(lid)}`
const overlayDir = (lid) => (OVERLAY_POOL ? `${CONTENT}/${language}/${dir}/${slugOf(lid)}` : lessonDir(lid))
const siblings = Object.keys(LANG).filter((k) => k !== language)
const siblingDirs = (lid) => siblings.map((k) => `${CONTENT}/${k}/${dir}/${slugOf(lid)}`).join(', ')
const ciCmd = (lid) => `${TUTOR} ci --language ${language} --filter ${lid}`
const inDir = (cmd, d) => cmd.replace('<dir>', d)
const TOOLCHECK = language === 'python' ? '"uv --version" and "ruff --version" match the bar, ' : ''
const EXERCISE_FILES = language === 'python' ? 'README.md, pyproject.toml, uv.lock, .python-version, starter, tests' : 'README.md, go.mod, starter, tests'
// A discussion lesson has no solution, and a script variant rides the shared language-agnostic
// check.sh (no overlay exercise, no solution): for both, ci reporting 0 solutions is the pass.
const shipsSolution = (l) => !(l.verify === 'discussion' || (l.mode === 'variant' && l.verify === 'script'))
const expectedTested = (l) => (l.verify ? (shipsSolution(l) ? 1 : 0) : null)

// The prompt asks each agent to paste the ci summary line; a reply without one,
// or with the wrong count, is a failed gate even when the agent sounded done.
const GATE_RE = /(\d+) solution\(s\) tested, (\d+) failed/
const gatePassed = (reply, expected) => {
  const m = GATE_RE.exec(reply || '')
  return Boolean(m) && m[2] === '0' && (expected === null || Number(m[1]) === expected)
}

log(`language ${language}: ${L.bar}` + (language === 'python' ? ' — uv and ruff on this machine must match those versions (agents run "uv --version" and "ruff --version" in their self-checks)' : ''))

const GUIDE =
  `Read ${REPO}/docs/authoring-guide.md fully, then study the exemplar lesson at ` +
  `${EXEMPLAR} (LESSON.md, exercise/, TUTOR.md, quiz.json, solution/)` +
  (OVERLAY_POOL ? ` and the overlay exemplar at ${OVERLAY_EXEMPLAR} (snippets/, exercise/ with README.md, solution/)` : '') +
  ` BEFORE writing anything. Never run git commands. `

const poolRule = (lid) =>
  OVERLAY_POOL
    ? `This is a ${pool.toUpperCase()}-pool lesson. Theory stays language-portable in the shared LESSON.md: no language-specific code there; where a language may illustrate a paragraph, put a <!-- lang: <slot> --> anchor on its own line after that paragraph (slot names kebab-case, unique per file). Language material goes in the ${language} overlay ${overlayDir(lid)}/: snippets/<slot>.md (one per anchor, starting with the "${L.snippet}" marker paragraph, ending with one newline), exercise/ with a README.md brief (files, run commands, language-bound criteria), solution/, and optional TUTOR.md and quiz.json (additive to the shared ones). The shared "## Exercise" section is language-neutral (what to build and the criteria that hold in any language) and defers to exercise/README.md for the checks; a language-agnostic exercise (terminal/git/etc.) stays in the shared lesson's plain exercise/.`
    : 'Exercise goes in exercise/, solution overlay in solution/.'

const finishRule = `
IMPORTANT — this lesson is PARTIALLY authored: a previous agent was interrupted mid-write.
FIRST inventory what already exists on disk and read every existing file end to end.
Keep and build on work that meets the guide's bar; rewrite only what is missing, truncated,
or wrong. The usual casualty is TUTOR.md / quiz.json (written last) — but verify the
existing LESSON.md, exercise and solution really pass the self-checks rather than assuming it.`

const VARIANT_DELIVERABLES = {
  tests: `exercise/ (${EXERCISE_FILES} — the README.md is the brief: files, run commands, language-bound criteria) and solution/. The exercise must satisfy the shared "## Exercise" section's language-neutral acceptance criteria exactly: the tests encode those criteria, no more and no fewer.`,
  script: `NO exercise files and NO solution/: the shared exercise/check.sh is language-agnostic and is already the effective exercise for ${language}. Add exercise/README.md only if ${language} needs its own brief (tool names, commands), nothing else.`,
  discussion: `NO exercise/ and NO solution/ (a discussion lesson has neither).`,
}

const variantRule = ({ id: lid, verify }) => `\
This is a VARIANT: the shared lesson at ${lessonDir(lid)}/ already exists and is the source of truth.
Read its LESSON.md (note every <!-- lang: <slot> --> anchor), TUTOR.md and quiz.json first, then the
sibling overlay at ${siblingDirs(lid)} where it exists.
Write ONLY under ${overlayDir(lid)}/ — never rewrite the shared LESSON.md, TUTOR.md or quiz.json (a wrong
shared claim goes in your reply, not in the file). Deliver: snippets/<slot>.md for every anchor where
${language} has something to say (each starts with the "${L.snippet}" marker paragraph and ends with one
newline; ship no file for a slot that does not apply), mirroring the sibling's snippets slot for slot
wherever the claim transfers so both languages make the same point at the same anchor; TUTOR.md and
quiz.json only where the shared ones do not transfer (both are additive). The lesson's verify type is
"${verify}", which settles the rest: ${VARIANT_DELIVERABLES[verify]}`

const selfChecks = (l) =>
  shipsSolution(l)
    ? `${TOOLCHECK}the starter marks each work site with a "${L.marker}" comment and collects/compiles ("${L.collect}" in the exercise dir) but its tests FAIL ("${L.test}"), ` +
      `"${inDir(L.fmt, overlayDir(l.id))}" and "${inDir(L.lint, overlayDir(l.id))}" clean, quiz.json parses, ` +
      `and the solution makes "${ciCmd(l.id)}" pass.`
    : `${TOOLCHECK}${OVERLAY_POOL ? `every snippets/<slot>.md matches an anchor in the shared LESSON.md, starts with the "${L.snippet}" marker paragraph and ends with one newline, ` : ''}` +
      `quiz.json parses${l.mode === 'variant' ? ' (if you ship one)' : ''}, and "${ciCmd(l.id)}" is clean with "0 solution(s) tested" (its validate pass covers anchors and snippets).`

const ciGate = (l) =>
  `The ci gate is "${ciCmd(l.id)}". Paste its summary line ("ci: N solution(s) tested, F failed, S skipped (...)") in your reply; ` +
  (!l.verify
    ? `anything other than "1 solution(s) tested, 0 failed" means the lesson is NOT done — except discussion lessons, where "0 solution(s) tested" is expected.`
    : shipsSolution(l)
      ? `anything other than "1 solution(s) tested, 0 failed" means the lesson is NOT done.`
      : `for this lesson "0 solution(s) tested, 0 failed" is the expected line (${l.verify === 'discussion' ? 'a discussion lesson has no solution to run' : 'the shared check.sh is language-agnostic and carries no solution'}); a solution counted here means you shipped exercise or solution files this lesson must not have.`)

const verbOf = { author: 'Author', finish: 'Complete', variant: `Author the ${language} variant of` }
const whereOf = (id, mode) =>
  mode === 'variant'
    ? `(overlay at ${overlayDir(id)}/)`
    : `at ${lessonDir(id)}/${OVERLAY_POOL ? ` (shared files) and its ${language} overlay at ${overlayDir(id)}/` : ''}`

const finishFirst = (l) =>
  l.mode !== 'variant'
    ? 'TUTOR.md and quiz.json before polishing anything else'
    : shipsSolution(l)
      ? 'exercise/README.md and solution/ before polishing snippets'
      : 'the snippets before any optional TUTOR.md or quiz.json'

const prompt = (l) => {
  const { id, mode } = l
  return (
    GUIDE +
    `${verbOf[mode]} the curriculum lesson "${id}" ${whereOf(id, mode)} .
Get your registry entry (title, duration, objectives, verify type): run
python3 -c "import json;print(json.dumps(json.load(open('${REPO}/skills/tutor/curriculum/registry.json'))['lessons']['${id}'],indent=2))"
Also read your row (content hints) and neighbors in the "${stageTitle}" table of ${REPO}/docs/curriculum-outline.md.
Learner context entering this stage: ${context}
This stage's lesson order: ${order.join(' -> ')} — assume ONLY knowledge from earlier lessons and earlier stages; do not leak later material.
${mode === 'variant' ? variantRule(l) : poolRule(id)}${mode === 'finish' ? finishRule : ''}
${mode === 'variant' ? '' : 'Write ALL required files per the guide (LESSON.md, exercise per verify type, TUTOR.md, quiz.json, solution overlay when test-verified). '}Then run the guide's self-checks and iterate until all pass — including: ${selfChecks(l)}
${ciGate(l)}
Do not leave the lesson half-written: if you are running long, finish ${finishFirst(l)}.
Final reply, max 6 lines: files written + self-check results, including the ci summary line.`
  )
}

phase('Author')
const results = await parallel(
  lessons.map((l) => () => agent(prompt(l), { label: `${l.mode}:${slugOf(l.id)}`, phase: 'Author' }))
)
const fails = lessons.filter((_, i) => !results[i]).map((l) => l.id)
const gateFails = lessons.filter((l, i) => results[i] && !gatePassed(results[i], expectedTested(l))).map((l) => l.id)
log(`completed ${lessons.length - fails.length}/${lessons.length}` + (fails.length ? ` FAILED: ${fails.join(',')}` : '') + (gateFails.length ? ` CI GATE NOT MET: ${gateFails.join(',')}` : ''))

return { done: lessons.length - fails.length - gateFails.length, fails, gateFails }
