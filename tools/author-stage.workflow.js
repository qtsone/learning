// Claude Code Workflow script: authors one curriculum stage.
// Invoke with args: { repo, stage, stageTitle, dir, pool, idPrefix, lessons: [ids], context, language }
//   repo       absolute path to this repository checkout
//   stage      registry stage key (e.g. "s0") or pack name
//   dir        content dir relative to curriculum/content (e.g. "shared/s0-foundations")
//   pool       "shared" | "focus" | a language code ("go", "python") for a language stage
//   idPrefix   lesson-id prefix for `tutor.py ci --filter` (e.g. "shared.foundations.")
//   lessons    ordered lesson ids for this stage
//   context    prose: what the learner knows entering this stage
//   language   language code the agents write for (default "go"); selects the exemplar and
//              toolchain and, for a shared or focus stage, the overlay under content/<language>/<dir>/
//   variant    true (shared or focus pool only) to author the <language> overlays of an EXISTING
//              stage: every agent writes ONLY under the overlay, and review findings against the
//              shared files are report-only
//   verify     { <lesson id>: "tests" | "script" | "discussion" } — each lesson's registry verify
//              type; required with variant (it decides the overlay's deliverables), optional otherwise
export const meta = {
  name: 'tutor-author-stage',
  description: 'Author one curriculum stage: parallel authors, dual adversarial review, targeted fixes',
  phases: [
    { title: 'Author', detail: 'one agent per lesson' },
    { title: 'Review', detail: 'technical + pedagogy adversarial reviewers' },
    { title: 'Fix', detail: 'apply high/medium findings per lesson' },
  ],
}

const LANG = {
  go:     { exemplar: 'go/s1-basics/hello-world', bar: 'Go 1.22+ idioms (log/slog, generics where natural, errors.Is/As, no ioutil, no deprecated APIs)',
            fmt: 'gofmt -l <dir> (no output)', lint: 'go vet ./...', test: 'go test -race ./...',
            collect: 'go build ./... && go vet ./...', marker: '// TODO:', snippet: 'In Go:', runner: 'gotest' },
  python: { exemplar: 'python/p1-basics/hello-python', bar: 'Python 3.14+ idioms (checked against uv 0.12, ruff 0.16, pytest 9, mypy 2)',
            fmt: 'ruff format --check <dir>', lint: 'ruff check <dir>', test: 'uv run -q --locked pytest -q',
            collect: 'uv run -q --locked pytest --collect-only -q', marker: '# TODO:', snippet: 'In Python:', runner: 'pytest' },
}

const { repo: REPO, stage, stageTitle, dir, pool, idPrefix, lessons, context, language = 'go', variant = false, verify = {} } = args
const L = LANG[language]
if (!L) throw new Error(`unknown language "${language}"; known: ${Object.keys(LANG).join(', ')}`)
const OVERLAY_POOL = pool === 'shared' || pool === 'focus'
if (!OVERLAY_POOL && pool !== language) throw new Error(`pool "${pool}" is a language stage; author it with language "${pool}"`)
if (variant && !OVERLAY_POOL) throw new Error(`variant needs a shared or focus pool, got "${pool}"`)
const VERIFY_TYPES = ['tests', 'script', 'discussion']
if (variant) {
  for (const lid of lessons) {
    if (!VERIFY_TYPES.includes(verify[lid])) throw new Error(`${lid}: variant needs verify[id] (${VERIFY_TYPES.join(' | ')}, the registry's verify.type), got "${verify[lid]}"`)
  }
}
const CONTENT = `${REPO}/skills/tutor/curriculum/content`
const EXEMPLAR = `${CONTENT}/${L.exemplar}`
const OVERLAY_EXEMPLAR = `${CONTENT}/${language}/shared/s2-cs/arrays-linked-lists`
const TUTOR = `python3 ${REPO}/skills/tutor/scripts/tutor.py`
const slugOf = (lid) => lid.split('.').pop()
const lessonDir = (lid) => `${CONTENT}/${dir}/${slugOf(lid)}`
const overlayDir = (lid) => (OVERLAY_POOL ? `${CONTENT}/${language}/${dir}/${slugOf(lid)}` : lessonDir(lid))
const STAGE_CODE_DIR = OVERLAY_POOL ? `${CONTENT}/${language}/${dir}` : `${CONTENT}/${dir}`
const siblings = Object.keys(LANG).filter((k) => k !== language)
const siblingDirs = (lid) => siblings.map((k) => `${CONTENT}/${k}/${dir}/${slugOf(lid)}`).join(', ')
const ciCmd = (lid) => `${TUTOR} ci --language ${language} --filter ${lid}`
const inDir = (cmd, d) => cmd.replace('<dir>', d)
const TOOLCHECK = language === 'python' ? '"uv --version" and "ruff --version" match the bar, ' : ''
const EXERCISE_FILES = language === 'python' ? 'README.md, pyproject.toml, uv.lock, .python-version, starter, tests' : 'README.md, go.mod, starter, tests'
const lessonOf = (lid) => ({ id: lid, mode: variant ? 'variant' : 'author', verify: verify[lid] })
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
    ? `anything other than "1 solution(s) tested, 0 failed" means the lesson is NOT done — except discussion lessons${variant ? ' and lessons whose exercise is the shared language-agnostic check.sh (the overlay ships no exercise or solution for those)' : ''}, where "0 solution(s) tested" is expected.`
    : shipsSolution(l)
      ? `anything other than "1 solution(s) tested, 0 failed" means the lesson is NOT done.`
      : `for this lesson "0 solution(s) tested, 0 failed" is the expected line (${l.verify === 'discussion' ? 'a discussion lesson has no solution to run' : 'the shared check.sh is language-agnostic and carries no solution'}); a solution counted here means you shipped exercise or solution files this lesson must not have.`)

const authorPrompt = (lid) => {
  const l = lessonOf(lid)
  return (
    GUIDE +
    `${variant ? `Author the ${language} variant of` : 'Author'} the curriculum lesson "${lid}" ${variant ? `(overlay at ${overlayDir(lid)}/)` : `at ${lessonDir(lid)}/${OVERLAY_POOL ? ` (shared files) and its ${language} overlay at ${overlayDir(lid)}/` : ''}`} .
Get your registry entry (title, duration, objectives, verify type): run
python3 -c "import json;print(json.dumps(json.load(open('${REPO}/skills/tutor/curriculum/registry.json'))['lessons']['${lid}'],indent=2))"
Also read your row (content hints) and neighbors in the "${stageTitle}" table of ${REPO}/docs/curriculum-outline.md.
Learner context entering this stage: ${context}
This stage's lesson order: ${lessons.join(' -> ')} — assume ONLY knowledge from earlier lessons and earlier stages; do not leak later material.
${variant ? variantRule(l) : poolRule(lid)}
${variant ? '' : 'Write ALL required files per the guide (LESSON.md, exercise per verify type, TUTOR.md, quiz.json, solution overlay when test-verified). '}Then run the guide's self-checks and iterate until all pass — including: ${selfChecks(l)}
${ciGate(l)}
Final reply, max 6 lines: files written + self-check results, including the ci summary line.`
  )
}

const VARIANT_RULE = variant
  ? ` This is an OVERLAY-ONLY pass (the ${language} variant of an existing stage): the shared LESSON.md, TUTOR.md, quiz.json and any language-agnostic exercise/ are frozen. Still report findings against them — they are report-only for this pass and go no further than the reply — but every fix you expect applied must land under the ${language} overlay.`
  : ''
const fixWhere = (lid) =>
  variant
    ? `${overlayDir(lid)}/ (the ${language} overlay of the shared lesson at ${lessonDir(lid)}/). Write ONLY under ${overlayDir(lid)}/ — never rewrite the shared LESSON.md, TUTOR.md or quiz.json; a finding against a shared file goes in your reply, not in the file`
    : `${lessonDir(lid)}/${OVERLAY_POOL ? ` (shared files; the ${language} overlay is ${overlayDir(lid)}/ — language-specific fixes go there, and the shared LESSON.md/TUTOR.md/quiz.json change only for language-neutral findings)` : ''}`

const FINDINGS = {
  type: 'object',
  properties: {
    findings: {
      type: 'array',
      items: {
        type: 'object',
        properties: {
          lesson: { type: 'string' },
          file: { type: 'string' },
          severity: { type: 'string', enum: ['high', 'medium', 'low'] },
          issue: { type: 'string' },
          fix: { type: 'string' },
        },
        required: ['lesson', 'file', 'severity', 'issue', 'fix'],
      },
    },
  },
  required: ['findings'],
}

const REVIEW_LENS = {
  go: '',
  python:
    ' PYTHON lens, on top: idiom (comprehensions where clearer, EAFP where taught, context managers, f-strings, pathlib, enumerate/zip, no global); ' +
    'typing (annotations on public functions in solutions from the typing lesson on, PEP 604/585/695 syntax, no Any leaks); no mutable default arguments; ' +
    'exceptions (no bare except, raise ... from, pytest.raises with match=); packaging (no sys.path hacks, no import *, __main__ guards, no conftest tricks); ' +
    'tests (parametrize, no order dependence, no hidden requirements); formatting (ruff format --check and ruff check clean; no caches committed; pyproject.toml + uv.lock + .python-version present beside every pytest exercise).',
}

const COHERENCE =
  ` CROSS-LANGUAGE coherence, for every overlay: each snippets/<slot>.md makes the same claim as the sibling slot under ${siblings.map((k) => `${CONTENT}/${k}/${dir}/<lesson>/snippets/`).join(' and ')} (flag a slot that contradicts or silently drops the sibling's point); the two exercise test suites encode the same acceptance criteria as the shared "## Exercise" section.`

const reviewerBase =
  `You are an adversarial reviewer of freshly authored curriculum stage "${stageTitle}" ` +
  `(lessons: ${lessons.join(', ')}) under ${CONTENT}/${dir}/ . ` +
  (OVERLAY_POOL ? `The ${language} overlays (snippets/, exercise/ with README.md, solution/, TUTOR.md, quiz.json) live under ${CONTENT}/${language}/${dir}/<lesson>/ . ` : '') +
  `Read ${REPO}/docs/authoring-guide.md and skim the exemplar ${EXEMPLAR} for the bar. ` +
  `Do NOT edit any file — report findings only. Severity: high = wrong/broken/misleading; ` +
  `medium = quality gap a learner would feel; low = nice-to-have. Be specific (file + concrete fix).` +
  VARIANT_RULE +
  ' '

const techPrompt =
  reviewerBase +
  `TECHNICAL lens. For every lesson: run "${TUTOR} ci --language ${language} --filter ${idPrefix}" once for the stage; for each test-verified lesson verify the STARTER collects/compiles ("${L.collect}") but its tests FAIL (run "${L.test}" in the ${OVERLAY_POOL ? "overlay's " : ''}exercise dir as-is); check factual accuracy of LESSON.md claims (${L.bar}), tests actually encode the stated acceptance criteria (no hidden or missing requirements), solutions are idiomatic code that is clean under "${inDir(L.fmt, STAGE_CODE_DIR)}" and "${inDir(L.lint, STAGE_CODE_DIR)}", check.sh scripts are safe/idempotent/clear, quiz expected_points technically correct.` +
  REVIEW_LENS[language] +
  (OVERLAY_POOL ? COHERENCE : '')

const pedaPrompt =
  reviewerBase +
  `PEDAGOGY lens. For every lesson read LESSON.md, TUTOR.md, quiz.json${OVERLAY_POOL ? ` and the overlay's snippets/, exercise/README.md, TUTOR.md, quiz.json` : ''}: tone/anatomy matches the exemplar; NO forward references to material not yet taught (check against stage order and the roadmap in ${REPO}/docs/curriculum-outline.md); registry objectives all covered by theory AND by core quiz questions (pull objectives from ${REPO}/skills/tutor/curriculum/registry.json); exercise difficulty fits the learner context ("${context}"); acceptance criteria unambiguous; remediation ladders escalate gradually; grading rubrics exercise-specific, not generic.`

phase('Author')
const authored = await parallel(
  lessons.map((lid) => () => agent(authorPrompt(lid), { label: `author:${slugOf(lid)}`, phase: 'Author' }))
)
const authorFails = lessons.filter((_, i) => !authored[i])
const authorGateFails = lessons.filter((lid, i) => authored[i] && !gatePassed(authored[i], expectedTested(lessonOf(lid))))
log(`authored ${lessons.length - authorFails.length}/${lessons.length}` + (authorFails.length ? ` FAILED: ${authorFails.join(',')}` : '') + (authorGateFails.length ? ` CI GATE NOT MET: ${authorGateFails.join(',')}` : ''))

phase('Review')
const reviews = await parallel([
  () => agent(techPrompt, { label: 'review:technical', phase: 'Review', schema: FINDINGS }),
  () => agent(pedaPrompt, { label: 'review:pedagogy', phase: 'Review', schema: FINDINGS }),
])
const allFindings = reviews.filter(Boolean).flatMap((r) => r.findings)
const actionable = allFindings.filter((f) => f.severity !== 'low')
const low = allFindings.filter((f) => f.severity === 'low')
log(`findings: ${actionable.length} actionable, ${low.length} low`)

phase('Fix')
const byLesson = {}
for (const f of actionable) (byLesson[f.lesson] ??= []).push(f)
const fixResults = await parallel(
  Object.entries(byLesson).map(([lid, fs]) => () =>
    agent(
      GUIDE +
        `Apply these review findings to lesson "${lid}" at ${fixWhere(lid)} :\n` +
        fs.map((f) => `- [${f.severity}] ${f.file}: ${f.issue} — fix: ${f.fix}`).join('\n') +
        `\nChange only what the findings require; keep the lesson's voice. Re-run the guide's self-checks until green. ${ciGate(lessonOf(lid))} Reply: what changed + the ci summary line, max 4 lines.`,
      { label: `fix:${slugOf(lid)}`, phase: 'Fix' }
    )
  )
)

return {
  stage,
  authored: lessons.length - authorFails.length,
  authorFails,
  actionableFindings: actionable.length,
  fixedLessons: Object.keys(byLesson),
  fixFails: Object.keys(byLesson).filter((_, i) => !fixResults[i]),
  authorGateFails,
  fixGateFails: Object.keys(byLesson).filter((lid, i) => fixResults[i] && !gatePassed(fixResults[i], expectedTested(lessonOf(lid)))),
  lowFindings: low.map((f) => `[${f.lesson}] ${f.file}: ${f.issue}`),
}
