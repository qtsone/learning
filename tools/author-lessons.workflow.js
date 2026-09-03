// Claude Code Workflow script: authors or completes a batch of curriculum lessons.
// Unlike author-stage.workflow.js this runs the Author phase only, in small
// batches, so an interrupted run loses at most one batch. Review/fix comes
// afterwards from review-stage.workflow.js once every lesson exists.
//
// Invoke with args: { repo, stageTitle, dir, pool, order, lessons, context }
//   order    ordered lesson ids of the whole stage (for "assume only earlier material")
//   lessons  [{ id, mode }] where mode is "author" (new) or "finish" (partial on disk)
export const meta = {
  name: 'tutor-author-lessons',
  description: 'Author or complete a batch of curriculum lessons (author phase only)',
  phases: [{ title: 'Author', detail: 'one agent per lesson' }],
}

const { repo: REPO, stageTitle, dir, pool, order, lessons, context } = args
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

const finishRule = `
IMPORTANT — this lesson is PARTIALLY authored: a previous agent was interrupted mid-write.
FIRST inventory what already exists on disk and read every existing file end to end.
Keep and build on work that meets the guide's bar; rewrite only what is missing, truncated,
or wrong. The usual casualty is TUTOR.md / quiz.json (written last) — but verify the
existing LESSON.md, exercise and solution really pass the self-checks rather than assuming it.`

const prompt = ({ id, mode }) =>
  GUIDE +
  `${mode === 'finish' ? 'Complete' : 'Author'} the curriculum lesson "${id}" at ${CONTENT}/${dir}/${slugOf(id)}/ .
Get your registry entry (title, duration, objectives, verify type): run
python3 -c "import json;print(json.dumps(json.load(open('${REPO}/skills/tutor/curriculum/registry.json'))['lessons']['${id}'],indent=2))"
Also read your row (content hints) and neighbors in the "${stageTitle}" table of ${REPO}/docs/curriculum-outline.md.
Learner context entering this stage: ${context}
This stage's lesson order: ${order.join(' -> ')} — assume ONLY knowledge from earlier lessons and earlier stages; do not leak later material.
${poolRule}${mode === 'finish' ? finishRule : ''}
Write ALL required files per the guide (LESSON.md, exercise per verify type, TUTOR.md, quiz.json, solution overlay when test-verified). Then run the guide's self-checks and iterate until all pass — including: starter compiles but tests FAIL, solution makes "python3 ${REPO}/skills/tutor/scripts/tutor.py ci --filter ${id}" pass, gofmt clean, quiz.json parses.
Do not leave the lesson half-written: if you are running long, finish TUTOR.md and quiz.json before polishing anything else.
Final reply, max 6 lines: files written + self-check results.`

phase('Author')
const results = await parallel(
  lessons.map((l) => () => agent(prompt(l), { label: `${l.mode}:${slugOf(l.id)}`, phase: 'Author' }))
)
const fails = lessons.filter((_, i) => !results[i]).map((l) => l.id)
log(`completed ${lessons.length - fails.length}/${lessons.length}` + (fails.length ? ` FAILED: ${fails.join(',')}` : ''))

return { done: lessons.length - fails.length, fails }
