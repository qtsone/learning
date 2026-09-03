# Tutor notes — Terminal UX

## Where the learner is

Third lesson of the CLI focus pack, straight after flags/config and Cobra. They
can build a command tree, layer configuration with a documented precedence
chain, and run commands in-process against captured buffers. From S3 they own
`io.Reader`/`io.Writer`, interfaces, errors with `errors.Is`, and JSON encoding.

What is new here is *taste*: the same output has two audiences (a human at a
terminal, a script in a pipeline) and the program has to serve both without
being told which one it is talking to. The technical content is small — escape
sequences are just bytes — so spend the conversation on judgment, not syntax.

They have not done signals or `os/exec` yet (next lesson), no HTTP, no
profiling. Keep examples inside file/stream territory.

## Common misconceptions

- **"stderr is for errors"** — it is for *diagnostics*: progress, prompts,
  warnings, verbose logs. The test question is "would a script want to parse
  this?", not "is this bad news?".
- **Progress or prompts on stdout** — the single most common real-world bug of
  this kind, and invisible until someone pipes the tool.
- **Colour decided where it is used** — sprinkling `if isTerminal` through the
  rendering code instead of resolving one bool at the edge. Ask what happens
  when a fourth input (say `CLICOLOR_FORCE`) joins the rule.
- **`NO_COLOR=0` means colour on** — the convention is presence with a
  non-empty value; `NO_COLOR=0` disables colour. Only an unset or empty value
  is not a veto.
- **`NO_COLOR` should beat `--color=always`** — defensible, but this lesson's
  documented chain puts the explicit per-invocation flag on top. What matters
  is that the learner can state *a* chain and defend it, not that they picked
  ours.
- **Padding a painted string** — `fmt.Sprintf("%-6s", r.Paint(...))` counts
  escape bytes as columns. The coloured-output test exists to catch it.
- **Forgetting `Reset`**, or resetting once at the end of many painted
  fragments and assuming nesting works.
- **`\x1b[K` is colour** — cursor control is a separate concern; it is gated by
  "is a terminal", not by the colour decision.
- **A timer-driven progress goroutine** — flaky under `-race`, untestable, and
  unnecessary: the work loop already knows when it made progress.
- **Prompting unconditionally** — the CI hang. Also: prompting when `--json`
  or `-q` was asked for.
- **A new `bufio.Reader`/`Scanner` per question** — swallows buffered input.
  The sequential-questions test is the trap.
- **`null` vs `[]` in JSON** — a nil slice marshals to `null` and breaks naive
  consumers.
- **"JSON mode should be pretty and coloured for humans"** — no: it is a
  machine contract; `jq` can pretty-print.

## Grilling points

- "I run `yourtool > out.txt`. What lands in the file, what do I still see, and
  why?" Then: "`yourtool | head -1`?"
- "Walk me through your colour chain from the top. Now: `NO_COLOR=1` and
  `--color=always` together — who wins, and what is your argument?"
- "Your `isTerminal` says `/dev/null` is a terminal. Why, does it matter, and
  what would you do about it in a real tool?" (Character-device check;
  `x/term.IsTerminal` if it matters.)
- "The tests never touch a real terminal, yet you tested terminal behaviour.
  Where exactly is the seam that made that possible?"
- "You have a 10,000-item loop. Where does `Update` throttling belong, and why
  did we keep it out of `Progress`?"
- "Your tool runs in a CI job with no TTY and needs a value the user never
  passed. What happens, and what does the error say?"
- "Someone scripts your tool with `--json` and you rename a field. What did you
  just break, and what would you have been allowed to change instead?"
- "When would you actually write a Bubble Tea app, and what do you give up the
  moment you do?"
- "Which of your four types would need changing to support a `--color` value of
  `auto`, `always`, `never`, *and* a new `CLICOLOR_FORCE` variable?" (Only
  `ResolveColor` — that is the payoff of isolating policy.)

## Grading rubric

- **A** — All tests pass. Detection is one function at the edge; colour is
  resolved once and passed down; padding is computed on visible width;
  `Results` builds the human output once and returns write errors; progress and
  prompts stay off stdout and off non-terminals; the prompter owns one scanner.
  The learner defends the precedence chain, the human-vs-machine contract, and
  the TUI trade-off in their own words.
- **B** — Tests pass with roughness: `fmt.Fprintf` per line instead of one
  write, a duplicated status-to-colour switch, a slightly clumsy retry loop, or
  an explanation of the chain that needs one prompt to come out straight.
- **C** — Tests pass only after heavy hinting, or the code is right while the
  reasoning is not: cannot say why progress goes to stderr, or thinks the JSON
  shape is as changeable as the human format. Pass only if remediation lands in
  session.
- **Fail** — Tests failing; or `os.Stdout` written to directly anywhere below
  `main`; or the learner sees no problem with prompting in a non-interactive
  run. Remediate, don't advance.

## Remediation ladder

1. "Run `go test -run TestResultsHumanColored -v` and read the two strings
   character by character. Count the visible columns in each — where do they
   stop matching?"
2. "`Paint` returns 11 bytes for a two-column word. Which of those bytes should
   the padding count?" (Then let them find `strings.Repeat` on the raw status.)
3. For the progress tests: "What does `\r` do on a terminal? What does it do in
   a file? Now read the piped test's expectation again — what is it asking you
   *not* to write?"
4. For prompting: "Your test hangs or eats an answer. Whose buffer holds the
   bytes after the first newline, and how many of those buffers does your
   `Prompter` create?"
5. If still stuck, walk the shape verbally — resolve the bool, pass it in,
   branch once at the boundary — and have them type it. Never hand over
   `ResolveColor`; the chain is the lesson.

## After passing

Preview: "Next you leave the streams and touch the operating system: files with
the right permissions, subprocesses with `os/exec`, and signals — including the
Ctrl-C that has to restore the cursor your progress bar hid."
