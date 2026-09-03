// Claude Code Workflow script: authors one curriculum stage.
// Invoke with args: { repo, stage, stageTitle, dir, pool, idPrefix, lessons: [ids], context }
//   repo       absolute path to this repository checkout
//   stage      registry stage key (e.g. "s0") or pack name
//   dir        content dir relative to curriculum/content (e.g. "shared/s0-foundations")
//   pool       "shared" | "go" | "focus"
//   idPrefix   lesson-id prefix for `tutor.py ci --filter` (e.g. "shared.foundations.")
//   lessons    ordered lesson ids for this stage
//   context    prose: what the learner knows entering this stage
export const meta = {
  name: 'tutor-author-stage',
  description: 'Author one curriculum stage: parallel authors, dual adversarial review, targeted fixes',
  phases: [
    { title: 'Author', detail: 'one agent per lesson' },
    { title: 'Review', detail: 'technical + pedagogy adversarial reviewers' },
    { title: 'Fix', detail: 'apply high/medium findings per lesson' },
  ],
}

const { repo: REPO, stage, stageTitle, dir, pool, idPrefix, lessons, context } = args
const CONTENT = `${REPO}/skills/tutor/curriculum/content`
const EXEMPLAR = `${CONTENT}/go/s1-basics/hello-world`
const slugOf = (lid) => lid.split('.').pop()

const GUIDE =
  `Read ${REPO}/docs/authoring-guide.md fully, then study the exemplar lesson at ` +
  `${EXEMPLAR} (LESSON.md, exercise/, TUTOR.md, quiz.json, solution/) BEFORE writing anything. ` +
  `Never run git commands. `

const poolRule =
  pool === 'shared'
    ? 'This is a SHARED-pool lesson: theory must stay language-portable (concrete snippets in marked "In Go:" blocks); a Go-specific exercise goes in exercises/go/ (+ solutions/go/), a language-agnostic one (terminal/git/etc.) in plain exercise/.'
    : 'Exercise goes in exercise/, solution overlay in solution/.'

const authorPrompt = (lid) =>
  GUIDE +
  `Author the curriculum lesson "${lid}" at ${CONTENT}/${dir}/${slugOf(lid)}/ .
Get your registry entry (title, duration, objectives, verify type): run
python3 -c "import json;print(json.dumps(json.load(open('${REPO}/skills/tutor/curriculum/registry.json'))['lessons']['${lid}'],indent=2))"
Also read your row (content hints) and neighbors in the "${stageTitle}" table of ${REPO}/docs/curriculum-outline.md.
Learner context entering this stage: ${context}
This stage's lesson order: ${lessons.join(' -> ')} — assume ONLY knowledge from earlier lessons and earlier stages; do not leak later material.
${poolRule}
Write ALL required files per the guide (LESSON.md, exercise per verify type, TUTOR.md, quiz.json, solution overlay when test-verified). Then run the guide's self-checks and iterate until all pass — including: starter compiles but tests FAIL, solution makes "python3 ${REPO}/skills/tutor/scripts/tutor.py ci --filter ${lid}" pass, gofmt clean, quiz.json parses.
Final reply, max 6 lines: files written + self-check results.`

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

const reviewerBase =
  `You are an adversarial reviewer of freshly authored curriculum stage "${stageTitle}" ` +
  `(lessons: ${lessons.join(', ')}) under ${CONTENT}/${dir}/ . ` +
  `Read ${REPO}/docs/authoring-guide.md and skim the exemplar ${EXEMPLAR} for the bar. ` +
  `Do NOT edit any file — report findings only. Severity: high = wrong/broken/misleading; ` +
  `medium = quality gap a learner would feel; low = nice-to-have. Be specific (file + concrete fix). `

const techPrompt =
  reviewerBase +
  `TECHNICAL lens. For every lesson: run "python3 ${REPO}/skills/tutor/scripts/tutor.py ci --filter ${idPrefix}" once for the stage; for each test-verified lesson verify the STARTER compiles but its tests FAIL (run go test in the exercise dir as-is); check factual accuracy of LESSON.md claims (Go 1.22+ idioms, no deprecated APIs), tests actually encode the stated acceptance criteria (no hidden or missing requirements), solutions are idiomatic gofmt-clean code, check.sh scripts are safe/idempotent/clear, quiz expected_points technically correct.`

const pedaPrompt =
  reviewerBase +
  `PEDAGOGY lens. For every lesson read LESSON.md, TUTOR.md, quiz.json: tone/anatomy matches the exemplar; NO forward references to material not yet taught (check against stage order and the roadmap in ${REPO}/docs/curriculum-outline.md); registry objectives all covered by theory AND by core quiz questions (pull objectives from ${REPO}/skills/tutor/curriculum/registry.json); exercise difficulty fits the learner context ("${context}"); acceptance criteria unambiguous; remediation ladders escalate gradually; grading rubrics exercise-specific, not generic.`

phase('Author')
const authored = await parallel(
  lessons.map((lid) => () => agent(authorPrompt(lid), { label: `author:${slugOf(lid)}`, phase: 'Author' }))
)
const authorFails = lessons.filter((_, i) => !authored[i])
log(`authored ${lessons.length - authorFails.length}/${lessons.length}` + (authorFails.length ? ` FAILED: ${authorFails.join(',')}` : ''))

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
        `Apply these review findings to lesson "${lid}" at ${CONTENT}/${dir}/${slugOf(lid)}/ :\n` +
        fs.map((f) => `- [${f.severity}] ${f.file}: ${f.issue} — fix: ${f.fix}`).join('\n') +
        `\nChange only what the findings require; keep the lesson's voice. Re-run the guide's self-checks (including "python3 ${REPO}/skills/tutor/scripts/tutor.py ci --filter ${lid}") until green. Reply: what changed, max 4 lines.`,
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
  lowFindings: low.map((f) => `[${f.lesson}] ${f.file}: ${f.issue}`),
}
