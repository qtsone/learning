// Claude Code Workflow script: adversarially reviews an already-authored stage
// and applies the actionable findings. Split out of author-stage.workflow.js so
// review can run after lessons are authored in batches.
//
// Invoke with args: { repo, stageTitle, dir, idPrefix, lessons, context, language, pool, variant }
//   language language code under review (default "go"); selects the toolchain and bar and,
//            for a shared or focus stage, the overlay under content/<language>/<dir>/
//   pool     "shared" | "focus" | a language code; defaults to the first segment of dir
//   variant  true (shared or focus pool only) for an overlay-only pass: fix agents write ONLY under
//            the <language> overlay and findings against shared files are report-only
export const meta = {
  name: 'tutor-review-stage',
  description: 'Adversarially review an authored stage and fix high/medium findings',
  phases: [
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

const { repo: REPO, stageTitle, dir, idPrefix, lessons, context, language = 'go', pool = dir.split('/')[0], variant = false } = args
const L = LANG[language]
if (!L) throw new Error(`unknown language "${language}"; known: ${Object.keys(LANG).join(', ')}`)
const OVERLAY_POOL = pool === 'shared' || pool === 'focus'
if (!OVERLAY_POOL && pool !== language) throw new Error(`pool "${pool}" is a language stage; review it with language "${pool}"`)
if (variant && !OVERLAY_POOL) throw new Error(`variant needs a shared or focus pool, got "${pool}"`)
const CONTENT = `${REPO}/skills/tutor/curriculum/content`
const EXEMPLAR = `${CONTENT}/${L.exemplar}`
const OVERLAY_EXEMPLAR = `${CONTENT}/${language}/shared/s2-cs/arrays-linked-lists`
const TUTOR = `python3 ${REPO}/skills/tutor/scripts/tutor.py`
const slugOf = (lid) => lid.split('.').pop()
const lessonDir = (lid) => `${CONTENT}/${dir}/${slugOf(lid)}`
const overlayDir = (lid) => (OVERLAY_POOL ? `${CONTENT}/${language}/${dir}/${slugOf(lid)}` : lessonDir(lid))
const STAGE_CODE_DIR = OVERLAY_POOL ? `${CONTENT}/${language}/${dir}` : `${CONTENT}/${dir}`
const siblings = Object.keys(LANG).filter((k) => k !== language)
const ciCmd = (lid) => `${TUTOR} ci --language ${language} --filter ${lid}`
const inDir = (cmd, d) => cmd.replace('<dir>', d)

log(`language ${language}: ${L.bar}` + (language === 'python' ? ' — uv and ruff on this machine must match those versions (agents run "uv --version" and "ruff --version" in their self-checks)' : ''))

const GUIDE =
  `Read ${REPO}/docs/authoring-guide.md fully, then study the exemplar lesson at ` +
  `${EXEMPLAR} (LESSON.md, exercise/, TUTOR.md, quiz.json, solution/)` +
  (OVERLAY_POOL ? ` and the overlay exemplar at ${OVERLAY_EXEMPLAR} (snippets/, exercise/ with README.md, solution/)` : '') +
  ` BEFORE writing anything. Never run git commands. `


// The prompt asks each agent to paste the ci summary line; a reply without one,
// or with the wrong count, is a failed gate even when the agent sounded done.
const GATE_RE = /(\d+) solution\(s\) tested, (\d+) failed/
const gatePassed = (reply, expected) => {
  const m = GATE_RE.exec(reply || '')
  return Boolean(m) && m[2] === '0' && (expected === null || Number(m[1]) === expected)
}

const ciGate = (lid) =>
  `The ci gate is "${ciCmd(lid)}". Paste its summary line ("ci: N solution(s) tested, F failed, S skipped (...)") in your reply; ` +
  `anything other than "1 solution(s) tested, 0 failed" means the lesson is NOT done — except discussion lessons${variant ? ' and lessons whose exercise is the shared language-agnostic check.sh (the overlay ships no exercise or solution for those)' : ''}, where "0 solution(s) tested" is expected.`

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
  `medium = quality gap a learner would feel; low = nice-to-have. Be specific (file + concrete fix). ` +
  `Note these lessons were authored in separate batches, so also check CROSS-LESSON coherence: ` +
  `duplicated material, contradictory claims, broken back-references, dependency versions that ` +
  `disagree between lessons.` +
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
        `\nChange only what the findings require; keep the lesson's voice. Re-run the guide's self-checks until green. ${ciGate(lid)} Reply: what changed + the ci summary line, max 4 lines.`,
      { label: `fix:${slugOf(lid)}`, phase: 'Fix' }
    )
  )
)

return {
  actionableFindings: actionable.length,
  fixedLessons: Object.keys(byLesson),
  fixFails: Object.keys(byLesson).filter((_, i) => !fixResults[i]),
  fixGateFails: Object.keys(byLesson).filter((_, i) => fixResults[i] && !gatePassed(fixResults[i], null)),
  lowFindings: low.map((f) => `[${f.lesson}] ${f.file}: ${f.issue}`),
}
