# Tutor notes — Time

## Where the learner is

Eighth lesson of S3. They are fluent with interfaces, generics, closures, the
io philosophy, and JSON tags/marshalers, and they wrapped errors with `%w`
back in S1. **No concurrency yet** — goroutines are the next lesson and
channels the one after. The timers section deliberately previews `<-ch` in
one sentence; if they pull toward select loops, goroutine timers, or
"how does the ticker run in the background?", park it: "two lessons from
now, and it will land better with channels under you." The exercise is
purely sequential on purpose.

## Common misconceptions

- **Layout treated as a verb table** — writing `"YYYY-MM-DD"` or `"%Y-%m-%d"`
  as the layout, or inventing a reference date (`"2007-01-02"` parses —
  wrongly). The layout is one specific example date; any other date is a bug.
- **`time.Parse` assumes local time** — it assumes **UTC** when the layout
  carries no zone. Learners who "fix" ParseLocal with plain `Parse` plus a
  manual offset are fighting this; point them at `ParseInLocation`.
- **`==` for time comparison** — compiles, then fails across zones or after
  a monotonic strip. Same trap wearing a different hat: `time.Time` map keys.
- **"The wall clock measures elapsed time"** — NTP steps and DST make wall
  arithmetic lie; only two `time.Now()` readings in one process carry the
  monotonic clock. Subtracting parsed timestamps measures the *data*, not
  the *run*.
- **`.In()`/`.UTC()` "changes the time"** — they change the lens, never the
  instant. The FormatUTC test with an EET input exists to force this insight.
- **`t.Add(24*time.Hour)` is "tomorrow"** — across a DST switch it lands on
  a different wall-clock hour; calendar-tomorrow is `AddDate(0, 0, 1)`.
- **`time.After` is harmless anywhere** — pre-1.23 it leaked until firing
  when abandoned in loops; 1.23 made abandoned timers collectable, but
  Ticker/Reset is still the loop idiom. Verify they can state both halves.

## Grilling points

- "Why does Go make you write a layout as an example date? What bug class
  dies, and what new one is born?" (Verb typos die; wrong-reference-date
  layouts are born — hence the named constants.)
- "Two `time.Time` values print the same instant but `==` is false. Name
  everything `==` compares that `Equal` ignores."
- "Your S2 timing probes did `time.Since(start)`. NTP steps the clock back
  two seconds mid-benchmark — what does your probe report, and why?"
- "Cleanup must run every 30 seconds until shutdown: Timer, Ticker, or
  After? What must you not forget, and what happens if you forget it?"
- "You marshal a `time.Now()` to JSON and unmarshal it back. What did the
  round-trip silently drop, and when would you notice?"
- "In ParseLocal, why parse *in* the location instead of parsing then
  calling `.In(loc)`?" (The instant is already wrong by then — `.In` can't
  repair a UTC misparse.)

## Grading rubric

- **A** — All tests pass; `ParseLocal` wraps both failure paths with `%w`
  and names the bad input; layouts live in constants, reference date exact;
  `Equal`/`Before` throughout (no `==`, no Unix-seconds detours);
  gofmt-clean; can explain wall-vs-monotonic and the After-in-loops story
  unprompted.
- **B** — Tests pass with rough edges: errors returned bare or with vague
  messages, layout string repeated inline, or the monotonic explanation
  needs one prompt to surface. Solid on zones and Equal.
- **C** — Tests pass only after heavy hinting, or the code dodges the point
  (comparing `Unix()` everywhere instead of `Equal`, `Parse` + manual offset
  arithmetic instead of `ParseInLocation`). Works, but the model is missing
  — remediate before advancing.
- **Fail** — Tests failing, or they cannot explain why the layout is an
  example date or what `Equal` compares that `==` doesn't. Reteach; the
  concurrency arc ahead leans on deadlines and tickers.

## Remediation ladder

1. "Run `go test ./...` and read one failure aloud: which function, what
   did it get, what did it want?"
2. For parse failures: "Print the error itself — does it complain about the
   zone or the value? Which of your two steps produced it?"
3. "ParseLocal is two steps: `LoadLocation` gives you a `*Location` — now
   scan pkg.go.dev/time for the Parse variant that *accepts* one."
4. Talk the shape through — load, check, `ParseInLocation` with the exact
   `2006-01-02 15:04` layout, wrap with `%w` — and let them type every line.

## After passing

Preview: "Next is Goroutines — the `go` keyword, WaitGroups, and your first
race-detector run; two lessons on, channels will make today's `t.C` fully
click."
