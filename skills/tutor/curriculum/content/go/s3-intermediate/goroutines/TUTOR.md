# Tutor notes — Goroutines

## Where the learner is

Ninth lesson of S3 and their first concurrency anywhere in the roadmap. They
are fluent with closures (goroutine bodies), generics (Map's signature), and
S1's defer/panic semantics; the Time lesson previewed "`<-ch` blocks until a
value arrives" in one sentence. **Channels are the next lesson** — when they
ask "how do I get a value *out* of a goroutine?", the honest answer is
"today: write into memory you own and synchronize with WaitGroup; next
lesson: channels." Park select, mutexes/atomics (the Sync lesson), and
context. A solution that reaches for channels is premature rather than wrong
— steer back: this lesson is ownership plus WaitGroup.

Insist on `go test -race ./...` for every run. The habit for the whole
concurrency arc starts here, and the exercise choreography (deliberately
writing a racy Total, reading the report) is a required rep, not optional
color.

## Common misconceptions

- **"`go f()` gives me the result back somehow"** — a go statement returns
  nothing and there is no join handle. Completion and results are the
  programmer's problem: WaitGroup plus owned memory today, channels next
  lesson.
- **"Go waits for goroutines before exiting"** — `main` returning kills the
  process instantly; other goroutines stop mid-flight, deferred calls
  unrun. If their hello-goroutine demo prints nothing, this is why.
- **Sleep as synchronization** — `time.Sleep` after spawning "fixes" the
  demo, and they may try it in the exercise. A sleep is a guess about
  timing, not a guarantee of completion; the barrier tests and `-race` are
  built to make it lose. Zero tolerance from here to the capstone.
- **Add inside the goroutine** — "Add is about the goroutine, so it goes in
  the goroutine." Wait can run before the new goroutine ever executes Add
  and sail through at counter zero. Symptoms: flaky early returns, or a
  `-race`/"WaitGroup misuse" report. Rule: Add where Wait can see it,
  before the `go`.
- **"`total++` is one operation"** — it is read/modify/write, and
  interleavings lose updates. Deeper point: any data race makes behavior
  undefined; the memory model only promises meaning to race-free programs,
  so "it printed the right number" proves nothing.
- **"`-race` passed, therefore no races"** — the detector is dynamic; it
  flags conflicting accesses that actually happened in that run. A clean
  run is evidence, not proof — but every report is a real bug.
- **"Goroutines make it faster"** — concurrency is structure; parallelism
  needs free cores and enough work per goroutine. Goroutine-per-element on
  trivial work can be slower than a plain loop; correctness, not speed, is
  this lesson's win condition (worker pools come in Patterns).

## Grilling points

- "Narrate RunAll on three tasks: the WaitGroup counter value after every
  Add, Done, and at the moment Wait unblocks."
- "In Map, goroutines write into `out` and the caller reads it after Wait
  with no locks. Why is that read not a data race?" (Distinct indices for
  the writers; Done happens-before Wait returning for the reader.)
- "Show me the interleaving where Add-inside-the-goroutine breaks. Who
  observes counter zero, and what happens next?"
- "In your racy Total, what did the two stack traces in the race report
  point at? What was the shared variable, and where was each goroutine
  created?"
- "A worker returns early on error before its Done line. Who exactly leaks
  — the worker or someone else? What one-line habit makes the bug
  impossible?"
- "Why can this laptop run a million goroutines but not a million OS
  threads? Name two concrete costs that differ."
- "Your Map launches one goroutine per element of a ten-million-element
  slice. Legal? Wise? What would you want instead?" (Seed for worker pools
  and semaphores in Concurrency Patterns.)

## Grading rubric

- **A** — `go test -race ./...` clean; every Add precedes its `go`, every
  Done is deferred; Map and Total are race-free by ownership (per-index,
  per-cell) with results read only after Wait; no sleeps, no channels; can
  walk through the racy-Total report they produced and explain the
  happens-before reasoning and why `main` returning kills work.
- **B** — Race-clean and passing with rough edges: Done not deferred, chunk
  math awkward or duplicated, or the happens-before explanation needs one
  prompt. Solid on Add-before-go and on reading the race report.
- **C** — Passes only after heavy hinting, or the code dodges the point:
  goroutine-then-immediate-Wait per item (sequential in disguise), sleeps
  smuggled in, or Total "fixed" by shrinking the input rather than by
  ownership. Works, but the model is missing — remediate before Channels.
- **Fail** — Races under `-race`, dropped tasks, or they cannot explain
  what `go` does, why Wait is needed, or what the race report says. Do not
  advance: every remaining lesson in the stage builds on today's model.

## Remediation ladder

1. "Run `go test -race ./...` and read one failure aloud — which function,
   what did it expect, what happened? If it's the watchdog panic, what does
   'barrier never opened' say about how your work is running?"
2. "Where does your function return, relative to the goroutines finishing?
   What in the code *guarantees* that order — not usually produces it,
   guarantees it?"
3. "The WaitGroup is a counter. Point at the line where it goes up, the
   line where it goes down, the line that blocks. Walk those lines for two
   tasks: can Wait ever pass while work remains?"
4. Talk the RunAll shape through — loop, `Add(1)`, `go` a closure that
   defers `Done` and calls the task, `Wait` after the loop — let them type
   it, then have them transfer the same shape to Map and Total unaided.

## After passing

Preview: "Next, goroutines finally get to talk: channels — typed pipes with
blocking send and receive — and the `<-t.C` you met in the Time lesson
becomes obvious in hindsight."
