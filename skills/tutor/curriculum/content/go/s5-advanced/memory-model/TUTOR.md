# Tutor notes — Memory Model & Escape Analysis

## Where the learner is

Seventh lesson of S5, straight after Garbage Collection. They have written
concurrent Go since S3 (goroutines, channels, `sync.Mutex`, `WaitGroup`,
`sync.Once`, `context`) and have run everything under `-race` ever since — but
always as *rules they follow*, never as a contract they can quote. Last lesson
gave them the collector's bill for heap objects and left the obvious question
open: which values end up on the heap at all. This lesson answers both.

Two things to withhold. **pprof is the next lesson** — when they ask "how do I
find the allocating call site in a real service?", confirm it's the right
question and park it; here the tools are `-gcflags='-m'`, `AllocsPerRun` and
`-benchmem`. And **`unsafe`/`reflect` are late in this stage** — if they start
speculating about pointer arithmetic or "just cast it", redirect: nothing in
this exercise needs it.

The theory here is unusually load-bearing. A learner who fixes all four TODOs
but still says "the race was a timing bug" has not passed. Grade the
explanation at least as hard as the diff.

## Common misconceptions

- **"A benign race just gives me a stale value."** The memory model says a
  racing program's behavior is *undefined*, not merely stale. Multi-word
  values (strings, slices, interfaces) can tear; the compiler may hoist a read
  out of a loop and never re-read it at all. Make them say "undefined" out
  loud, with the tearing example.
- **"Pointer assignment is atomic on my CPU, so `Store` was fine."** Hardware
  atomicity is not a happens-before edge. Even if the pointer word arrives
  intact, the reader has no guarantee it sees the `Config`'s *fields*. This is
  the whole point of criterion 3.
- **"Two atomics, one per field, makes `Meter` safe."** Each field would be
  individually intact and the pair would still be inconsistent: `Snapshot`
  could pair a new `hits` with an old `last`, or a positive count with `""`.
  Atomicity composes only over one word. The invariant spans two fields, so
  the lock spans two fields.
- **"Atomics are the fast version of mutexes, so prefer them."** Mutex is the
  default; atomics are a special case for single-word publication with no
  invariant across words. `Store` is that case; `Meter` is not.
- **"`time.Sleep` / `runtime.Gosched()` / a channel used only as a delay fixes
  it."** No edge, no fix. Any "it stopped reporting" achieved by slowing
  things down is a red flag — check whether they shrank the test.
- **"The race detector caught it because the timing happened to be bad."**
  It tracks the happens-before graph, so it flags unsynchronized access pairs
  it *observes execute*, regardless of interleaving. The flip side matters
  too: a clean `-race` run proves nothing about code paths that didn't run.
- **"Taking a pointer forces heap allocation"** and its cousin **"`new` is
  heap, `var` is stack."** Neither. `&x` on a local that provably dies with
  the frame stays on the stack. Placement is a compiler proof, not a syntax.
- **"Escaping is a bug."** Escaping is frequently the correct answer — the API
  demanded it. `SummarizeHeap` is not *wrong*, it is *more expensive*, and
  only a benchmark tells you whether that matters.
- **"`-m` prints nothing, so nothing escapes."** Build cache. They need `-a`
  or a real code change; if the compiler didn't run, there is no output.
- **"`AllocsPerRun` measures speed."** It counts allocations, it is
  deterministic, and it temporarily forces `GOMAXPROCS(1)` — so it must not be
  used inside a `t.Parallel()` test. Good thing to point out for their own
  future test suites.
- **Publishing an immutable value, then mutating it.** If anyone does
  `s.Current().Version++`, the atomic bought them nothing. "Copy-on-write, not
  mutate-in-place" is the contract that makes pointer publication safe.

## Grilling points

- "Point at the exact happens-before edge that makes your `Meter` correct.
  Which `Unlock` pairs with which `Lock`, and what does that edge make
  visible?"
- "You used `atomic.Pointer` for `Store` but a mutex for `Meter`. Defend the
  asymmetry. Now defend the opposite choice — when would a mutex be the better
  call for `Store`?" (Multi-field updates, or work that must happen under the
  same critical section.)
- "The race test failed on your very first `-race` run and would have failed
  on mine too. Why is that not luck?"
- "Here's the `ready`/`conn` snippet from the lesson. I add
  `time.Sleep(10*time.Millisecond)` before the read. Fixed? Why not — name the
  missing edge."
- "In your `-m` output, find the line that was `AppendSample`'s allocation.
  Why could `strconv.AppendInt` remove it when `fmt.Sprintf` could not?"
  (`...any` boxing + a fresh result string vs. writing digits into the
  caller's buffer.)
- "`Summarize` returns a 32-byte struct by value; `SummarizeHeap` returns an
  8-byte pointer. The pointer moves fewer bytes. Why is the value version
  cheaper anyway?" (Copy is a stack write; the pointer version costs an
  allocation plus GC work later. Bytes copied is the wrong metric.)
- "When *would* returning `*Report` be the right API?" (Large structs, shared
  identity, in-place mutation by the caller, `nil` as a meaningful result.)
- "Your `AppendSample` gate says zero allocations *when `dst` has capacity*.
  What happens on `AppendSample(nil, …)` and why doesn't that break the
  contract?"
- "Last lesson's `sync.Pool` and this lesson's caller-owned buffer solve the
  same problem. How do you choose?" (Pool for scratch space with unknown
  callers and concurrency; caller-owned buffer when the caller has a natural
  loop to hang the buffer off — simpler, no contract to get wrong.)
- Stretch: "`once.Do(f)` — state its edge precisely." (`f` *returning*
  happens-before *any* `Do` returns, including calls that never ran `f`.)

## Grading rubric

- **A** — All tests pass under `-race`. `Meter` is guarded by one mutex
  covering both fields (an `RWMutex` is fine if they can justify it);
  `Store` uses `atomic.Pointer[Config]`, including in `NewStore`;
  `AppendSample` builds the line with `append` + `strconv.AppendInt` and no
  intermediate string; `Summarize` is a genuine value-returning
  implementation, not a dereference of `SummarizeHeap`. `NOTES.md` has real
  before/after `-m` lines with a correct one-sentence cause for each escape,
  and both benchmark lines with the `allocs/op` difference explained. They can
  name the happens-before edges without prompting and say "undefined behavior"
  rather than "stale value".
- **B** — Tests pass and the code is right, but the reasoning is thin: `Store`
  fixed with a mutex and no engagement with why atomics fit single-word
  publication, or `NOTES.md` numbers pasted without explanation, or the escape
  causes described as "it allocates" rather than "it outlives the frame
  because…". Nudge once per gap; a learner who repairs the explanation on the
  spot is an A-minus, not a B.
- **C** — Green only after heavy hinting, or the `Meter` fix is cargo-culted
  (mutex added everywhere including places it isn't needed, with no account of
  the invariant), or `NOTES.md` section 1 was back-filled from the refactored
  code. Pass only if remediation lands inside the session.
- **Fail** — Tests failing; or the race "fixed" by shrinking the test, adding
  sleeps, or deleting the concurrent `Snapshot` goroutine; or `NOTES.md`
  empty; or they still describe a data race as a timing bug that yields a
  stale value. Remediate, don't advance.

## Remediation ladder

1. "Read the race report out loud. It names two accesses, two goroutines, and
   two source lines. Which field, and which two lines? The detector already
   did the diagnosis — you're just reading it." (If the first failure is an
   allocation gate instead, it names `AppendSample` or `Summarize`. Either
   way: one function at a time.)
2. Now make them name the concept under *their* failure. `Meter`: "Say the
   invariant in one sentence. Not 'the fields are shared' — what must be true
   of the *pair* every time `Snapshot` returns?" `Store`: "How much state is
   there, really? One pointer, and the `Config` behind it is never mutated
   after publication. Watch `NewStore` too — a plain field assignment there is
   still a write to the same location." `AppendSample`: "Run the `-m` command
   and find the `format.go` lines. Everything handed to a `...any` parameter
   is boxed, and `Sprintf` builds a brand-new string on top of that."
   `Summarize`: "Put the two signatures side by side. Which one forces the
   result to outlive the frame?"
3. Now the tool. `Meter`: "Which tool protects a *pair* of fields, and which
   one only protects a word? Then point at the `Unlock`/`Lock` edge your
   choice creates." `Store`: "Go to the `sync/atomic` docs and find the type
   that publishes exactly one typed pointer." `AppendSample`: "What in
   `strconv` writes digits *into* a byte slice you already own? Build the line
   with `append` and never materialize a string." `Summarize`:
   "`SummarizeHeap` already has the loop you need. Change the shape, not the
   arithmetic."
4. Only if still stuck on the atomics: show the three-line shape
   (`cfg atomic.Pointer[Config]`, `s.cfg.Store(c)`, `s.cfg.Load()`) and then
   make them explain which edge `Store`/`Load` creates and what it makes
   visible about the `Config`'s fields.

## After passing

Preview: "You can now reason about where memory lives and when a write becomes
visible. Next lesson stops reasoning and starts measuring: pprof, CPU and heap
profiles, and the discipline of an optimization loop that only changes what a
profile blamed."
