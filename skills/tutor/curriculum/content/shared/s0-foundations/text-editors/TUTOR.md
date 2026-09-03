# Tutor notes — Editors & IDEs

## Where the learner is

Third lesson ever. They have the source-is-plain-text model from
what-is-a-program and terminal navigation plus the PATH concept from
terminal-basics. No git, no programming language installed — the Go toolchain
arrives only in dev-environment at the end of this stage. That's why the
exercise demonstrates language servers with VS Code's built-in Markdown and
JSON support: do NOT let them rathole on installing Go or the Go extension
now; if they've installed the Go extension early, fine, but full `gopls`
verification is dev-environment's job. Discussion-verified: you gate via the
tour results and conversation, not a script.

## Common misconceptions

- **"The editor understands my code"** — the editor renders text; the
  language server understands the language. If this is fuzzy, the whole LSP
  objective is missed — remediate before moving on.
- **"Autocomplete guesses / is AI"** — completions here are analysis, not
  prediction: the server knows what can legally come next.
- **"VS Code is an IDE" / category confusion** — the interesting answer is
  the trade-off (preconfigured depth vs. assembled flexibility), not the
  label.
- **"I must memorize shortcuts"** — the command palette is the searchable
  index; three habits suffice. Watch for shortcut overwhelm and shrink the
  set, don't grow it.
- **"A word processor would work too"** — recheck the plain-text model from
  lesson one if this surfaces.
- **`code .` fails on macOS** — they skipped "Shell Command: Install 'code'
  command in PATH". Turn it into a PATH refresher, it's a gift: where did the
  command get installed, and why does the shell find it now?
- **Fixing the JSON by intuition** — some learners repair the file without
  reading a single diagnostic. Send them back to the Problems panel: the
  skill being built is *reading* diagnostics, not spotting typos.

## Grilling points

- "Walk me through what happens between your keystroke and a red squiggle
  appearing. Who knows JSON — VS Code or something else?"
- "Why do Markdown and JSON work with no extension, but Go will need one?"
- "N editors, M languages — why did the world settle on a protocol? What
  would the plugin count be without it?" (N×M vs N+M.)
- "You changed eight `colour`s with multi-cursor. When would find-and-replace
  be the better tool, and when would it be dangerous?" (Replace-all can hit
  unintended matches; multi-cursor is visual confirmation per site.)
- "One-line fix on a remote server you can only reach via terminal — which
  editor and why?" (nano/vim; no window can open — ties to terminal-basics.)
- "You're starting a huge single-language codebase at a company that pays for
  tools. Editor or IDE? Defend it." (Either answer is fine; grade the
  reasoning.)

## Grading rubric

- **A** — All six criteria done; demonstrates palette/quick-open/multi-cursor
  live without prompting; explains diagnostics/completions/go-to-definition
  as language-server features in their own words; editor-vs-IDE answer weighs
  concrete trade-offs for a scenario.
- **B** — Tour complete but one keyboard habit still mouse-backed, or the
  LSP explanation is right in outline but wobbly on who-does-what; IDE
  answer correct but shallow ("IDE is bigger").
- **C** — Tour completed only with step-by-step prompting, or fixed the JSON
  without using diagnostics, or can't name what a language server provides.
  Pass only if a time-boxed re-demonstration lands; else another iteration.
- **Fail** — VS Code not installed/working, hunt done by clicking the file
  tree, or "the editor just knows" persists after remediation. Remediate,
  don't advance.

## Remediation ladder

1. "Reopen `inventory.json` and hover the first squiggle. Read the message
   aloud — what is it telling you, and what do you think produced it?"
2. "VS Code was born knowing nothing about JSON. Something else is running
   and being asked questions. What does the editor send it, and what comes
   back?"
3. Redraw the N×M picture together: 5 editors, 10 languages — count the
   integrations with and without a shared protocol.
4. Give the shape directly — editor shows text and forwards questions; a
   separate language-server program answers with diagnostics, completions,
   and definitions — then have them re-explain it over the Markdown link
   jump (`Cmd+Click` / `Ctrl+Click`) they performed.

## After passing

Preview: "Next is version control — git starts keeping a history of every
file you now know how to edit, so no change is ever lost again."
