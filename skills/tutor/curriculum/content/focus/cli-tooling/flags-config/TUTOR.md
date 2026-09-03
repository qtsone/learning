# Tutor notes — Flags & Configuration

## Where the learner is

First lesson of the CLI focus pack, inserted right after S3. They know the
whole of intermediate Go: interfaces, generics, errors with `Is`/`As`, JSON
(including pointer fields for absent-vs-zero), `time`, `io.Reader`/`io.Writer`,
table-driven tests, and the full concurrency arc. They have shipped two CLIs —
the S1 todo tracker and the S3 concurrent link checker, whose
`run(ctx, args, in, out) error` shape and own `flag.FlagSet` are the
direct ancestors of this lesson's `Loader`.

What is genuinely new: `flag.Value`/`flag.Func`, `FlagSet.Visit`, controlling
usage output, and the whole idea that configuration *layering* is a designed,
documented, testable contract rather than a pile of `if x != "" `.

They have **not** done S4 or later: no TDD lesson, no security lesson, no HTTP
servers, no databases. Keep secrets talk to the one practical point (do not put
secrets in flags — `ps` and shell history see them) and do not wander into
threat modelling. Cobra arrives next lesson; if they ask "why not just use
cobra/viper", the honest answer is "you are about to, and you will understand
what it is doing for you because of this hour".

Nothing here is concurrent, so the race detector is quiet — but note the
loader takes `LookupEnv` as a function precisely so tests never mutate global
process state. That discipline is worth naming aloud.

## Common misconceptions

- **"Zero means unset."** The big one. They read `raw.verbose` directly and
  clobber the config file on every run. Symptom: `TestLoadFlagCountsOnlyWhenPresent`
  fails, or the "defaults when nothing else is present" case reports
  `SourceFlag` everywhere. Cure: `fset.Visit` visits *only* flags that appeared
  on the command line; `VisitAll` visits all of them.
- **`os.Getenv` instead of `os.LookupEnv`.** Throws away the "is it set?" bit,
  so `TOOLKIT_ENDPOINT=""` becomes indistinguishable from an absent variable
  and the "endpoint from the environment is empty" case fails.
- **Non-pointer JSON fields.** `Retries int` in `fileConfig` makes a file
  without a `retries` key silently set retries to 0. Point back at the JSON
  lesson: absent and zero are different documents.
- **Validating each layer as it arrives.** Feels defensive, is wrong: a file
  may set `retries: 20` that a flag then corrects. Validate the resolved
  result, attributing failures through `Origins`.
- **`ExitOnError` (or plain `flag.String`).** Any `os.Exit` inside `Load` makes
  the package untestable and unusable as a library. Same trap as the global
  `flag` set they already avoided in S3.
- **Value receiver on `Set`.** `func (t TagList) Set(...)` compiles but appends
  to a copy; the flag package's `Var` also then rejects the type. Point at the
  methods lesson: pointer receiver for anything that mutates.
- **Forgetting `Origins` on some path.** Origins written only in the flag layer,
  or only when the value changes. Every write of a field writes its origin, in
  the same breath.
- **"Precedence is obvious, why write it down?"** It is obvious *to them today*.
  Ask what happens when someone else adds a fifth layer next quarter.

## Grilling points

- "A user swears `-timeout 10s` is being ignored. Walk me through how your
  program answers that in under a minute." (The `Origins` map — and if they say
  "I'd add a print statement", ask what they ship to the user.)
- "Why does the environment beat the config file and not the other way round?
  Give me a concrete scenario that breaks under the inverted order." (Same repo
  against staging without editing a tracked file.)
- "Why must flags beat the environment?" (A stale export in a shell profile
  silently overruling a human is an afternoon of debugging.)
- "You register `-retries` with a default of 0 but the real default is 3. Where
  does 3 live, and why isn't it in the `IntVar` call?"
- "Rewrite `-tag` with `flag.Func`. What do you lose?" (Control of `String()`,
  so nothing sensible shows in usage or a `--show-config` dump.)
- "Should a second `-tag` append to the file's tags or replace them? Defend
  your answer, then tell me how a user gets the *other* behaviour."
- "Which of these five settings would you refuse to accept as a flag, and
  why?" (A token: `ps`, shell history, CI logs.)
- "Where should a `-config` file that does not exist be an error, and where
  should it be silence? Why the asymmetry?"
- "Your `ValueError` wraps the `strconv` error. What would break for callers if
  you formatted it into the message with `%v` instead of `%w`?"

## Grading rubric

- **A** — All tests pass. The per-layer split the starter hands them is
  respected rather than smeared back into `Load`, `fset.Visit` (not
  zero-checking) decides flag presence, `LookupEnv`'s
  second value decides env presence, `fileConfig` uses pointer fields, origins
  are written next to every value assignment, and validation runs once on the
  resolved result. Usage text is theirs, not `flag`'s default. They can justify
  the ordering with a scenario, not a slogan.
- **B** — Tests pass but the layering is ad hoc inside the given functions
  (origins set in a second pass, validation duplicated per layer), or
  the usage output is padded but not thought through. Explanation is sound.
- **C** — Tests pass only after heavy hinting, or they cannot say why flags
  outrank the environment, or `Origins` is treated as decoration they wrote to
  satisfy a test. Pass only if a short remediation lands; otherwise iterate.
- **Fail** — Zero-checking instead of `Visit` still in place, `os.Exit` inside
  `Load`, tests failing, or precedence they cannot restate. Remediate; the next
  lesson builds command trees on exactly this foundation.

## Remediation ladder

1. "Run only the failing test with `go test -run TestLoadPrecedence/...`. Which
   layer's origin is wrong, and which layer wrote it last?"
2. "You have `-verbose=false` on the command line and `verbose: true` in the
   file. Print the value of your flag variable when the flag is *absent*. Now:
   what does that value tell you about whether the user typed it?"
3. "`FlagSet` has two visitor methods. Read their doc comments and tell me
   which one answers 'did the user type this flag?'" (Then the same question
   for `os.Getenv` versus `os.LookupEnv`, and for a JSON field versus a pointer
   to one.)
4. Sketch the shape aloud — start from `Defaults()`, then each of the three
   given layer functions in order, writing value and origin together, `Validate`
   last — and let them type it. Only show the file layer's `if fc.X != nil`
   block if they are still stuck after that.

## After passing

Preview: "Next lesson turns this single command into a tree of subcommands with
cobra — and you will recognise its flag binding as the mechanism you just built
by hand."
