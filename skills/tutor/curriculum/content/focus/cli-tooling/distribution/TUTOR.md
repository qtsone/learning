# Tutor notes — Distribution

## Where the learner is

Fifth lesson of the CLI focus pack. They can layer configuration, build a cobra
command tree, keep stdout and stderr honest, and drive child processes and
signals. Everything so far ran from a working directory; this lesson is the
first that treats the tool as an artifact other people download.

The content is only half Go. The other half is release engineering judgement —
what a version number promises, what a channel costs — and it is the half worth
spending conversation on. The Go part (three linker-patched vars) is twenty
lines; the reason it is fiddly is that failure is *silent*.

They have not done S4 (no CI lesson yet in this ordering if the pack came
straight after S3), no HTTP, no containers. Keep the release pipeline concrete —
"a workflow that runs on tag push" — without teaching CI. The exercise runs
entirely offline: goreleaser is never executed by the checker, and no learner is
expected to install it, though encourage it if they can.

## Common misconceptions

- **"`-X` failed loudly if I got it wrong"** — it does not. A wrong symbol path
  builds fine and ships `dev` to every user. The habit to install: assert on
  `--version` output in the pipeline.
- **`main.version` from a subpackage** — the symbol is the full import path,
  `tutor.local/digest/internal/buildinfo.version`. Watch for them copying a
  blog post that puts the vars in `main`.
- **Trying to patch a `const`, or a var initialised by a function call** — must
  be a package-level string var with a constant initialiser.
- **"The version is in the source"** — a hard-coded constant is the thing they
  will forget to bump; the tag is the single source of truth, and both the
  binary and the release notes derive from it.
- **"Only the code is the API"** — flag names, exit codes, and JSON field names
  are the contract. Adding a JSON field is minor; renaming one is major.
- **"A new subcommand is a breaking change"** — no: additive, minor. Conversely
  "it's just a default value" for a default whose change alters behaviour — that
  *is* major.
- **`0.x` confusion** — pre-1.0 makes no promises; the fix for "I'm not ready to
  commit" is staying on `0.x`, not inventing private rules for `1.x`.
- **Stamping `$(date)`** — destroys reproducibility. The commit date is a
  property of the source; the build clock is not.
- **"`-trimpath` is an optimisation"** — it is about what is *baked in*: your
  machine's absolute paths, in the binary and in panic traces.
- **"Checksums prove I built it"** — they prove the download matches the
  published file. Provenance needs signatures.
- **"goreleaser builds it"** — goreleaser calls `go build`; what it really
  automates is the matrix, archives, checksums, notes, upload and downstream
  packages. It will not rescue a config whose ldflags are wrong.
- **Tap naming** — the repo must be `homebrew-tap` (or `homebrew-<anything>`);
  `you/tap` in `brew install` is the short form of that repository.
- **"`go install` users get my version"** — they do not; that is what the
  `ReadBuildInfo` fallback is for, and why `(devel)` must be rejected.

## Grilling points

- "I run `yourtool --version` and it says `dev`. Give me three different causes,
  and how you'd tell them apart."
- "You add `--json` to a command that had only human output. Major, minor, or
  patch? Now you rename one of its fields. Now you fix a typo in a help string."
- "Someone is parsing your human-readable output with `awk` and your patch
  release realigned the columns. Did you break semver? What should you have
  given them instead?"
- "Why does your `--version` line carry the commit *and* the date, when the
  version already identifies the release?" (Unreleased builds; rebuilt
  artifacts; dirty trees.)
- "Explain to a C programmer why `GOOS=windows go build` needs nothing
  installed. Where does the Windows standard library come from?"
- "What actually changes when you set `CGO_ENABLED=0`, and what do you give up?"
- "I build your tool twice and get different bytes. Walk me through your
  suspects, most likely first."
- "Push tag `v1.4.2`, CI runs goreleaser. Trace where the string `1.4.2` ends up
  — every place."
- "Your tap repo is called `you/brew-tools` and `brew tap you/brew-tools`
  fails. Why?"
- "You have 200 users on Homebrew and want to remove a flag. What is your
  sequence over the next two releases?"
- "Pick a channel you would *not* offer for this tool, and defend it."

## Grading rubric

- **A** — `bash check.sh` is green. `buildinfo` is three vars plus one formatter
  with the `ReadBuildInfo` fallback isolated in its own function; the
  `.goreleaser.yaml` ldflags mirror the documented build exactly, using template
  values rather than hard-coded ones; `.CommitDate` chosen over `.Date` with the
  reason given. The learner can classify version bumps on the spot and explain
  what a tap is without hand-waving.
- **B** — Green checks, rougher edges: the fallback inlined into `String()` in a
  hard-to-read way, `.Date` used with no argument for it, a config that satisfies
  the checks but was assembled by pattern-matching the lesson. Reasoning mostly
  right, one prompt needed.
- **C** — Green only after heavy hinting, or right code with wrong model: cannot
  say why `-X` silently did nothing, or thinks renaming a JSON field is a patch.
  Pass only if remediation lands in session.
- **Fail** — Checks failing; or a hard-coded version constant with the ldflags
  ignored; or the learner cannot name a single thing in their tool that a major
  bump would be for. Remediate, don't advance.

## Remediation ladder

1. "Run the checker and read the first FAIL only. Which of the two files does it
   name — the Go one or the YAML one?"
2. For a stubborn `dev`: "Run `go build -ldflags \"-X main.version=1.4.2\" .` and
   then the documented one. Both succeed. What does that tell you about how the
   linker treats a symbol it cannot find?" Then: "Print the import path of the
   package your var lives in — line for line, is it the string in your `-X`?"
3. For the `(devel)` trap: "Comment out your fallback and re-run. Which check
   flips? Now: what does `bi.Main.Version` return for a build that never went
   through a module proxy?"
4. For reproducibility: "Build twice into two files and run
   `cmp -l a b | head`. Then think about what is in your ldflags that a clock
   could have produced."
5. For the config: "Take the annotated YAML in the lesson and, key by key, ask
   what the checker would look for. `-trimpath` is in `flags`, not `ldflags` —
   why are those two different lists?"
6. If still stuck, walk the *shape* verbally (three vars, one `Sprintf`, a
   fallback guarded on `dev`) and let them type it. Never hand over the
   `.goreleaser.yaml` wholesale — assembling it is the exercise.

## After passing

Preview: "Last lesson of the pack: the capstone. You ship one tool that uses all
of it — a command tree, layered config, output that behaves in a pipe, a
subcommand that reaches outside the process, cancellation on Ctrl-C, and the
`--version` you just learned to stamp."
