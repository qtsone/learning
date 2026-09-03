# Tutor notes — Interfaces

## Where the learner is

First lesson of S3, first genuinely new *language* concept since S1. They are
fluent in structs, methods (including the value-vs-pointer receiver choice and
an introductory look at method sets), errors with `%w`/`errors.Is`, table
tests, and basic file I/O — but they have never written an interface, and the
S1 methods lesson deliberately ended on the method-set cliffhanger this lesson
resolves. Learners coming from OOP languages will try to map interfaces onto
classes and inheritance; learners without that background often find implicit
satisfaction *easier*. Type assertions, embedding, and generics are all later
lessons — steer discussion away from inspecting an interface's dynamic type
("that's next-next lesson") and never introduce `any` beyond a passing
mention.

## Common misconceptions

- **"Where do I declare that CountingWriter implements io.Writer?"** —
  Nowhere; that's the point. If they keep looking for an `implements`
  keyword, revisit structural vs nominal satisfaction.
- **Value vs pointer confusion at the call site** — `io.Copy(w, …)` with a
  value `CountingWriter` fails to compile. This is the method-set rule in
  action, not a compiler quirk. Have them read the error aloud
  ("Write method has pointer receiver").
- **`Write` returning `(0, nil)`** — the starter does this deliberately;
  `io.Copy` fails with `io.ErrShortWrite`. Learners often think the return
  value is decorative. It's the contract.
- **Returning the downstream `n` from `UpperWriter.Write`** — passes these
  tests (ASCII preserves length) but is subtly wrong: `n` must describe bytes
  consumed from `p`. The solution returns `len(p)` explicitly; probe for
  this during review even though the tests can't catch it.
- **Uppercasing `p` in place** — mutating the caller's slice violates the
  Writer contract; `TestUpperWriterLeavesInputAlone` catches it. Connect back
  to S1 slices sharing backing arrays.
- **"Interfaces are for future flexibility, so export lots of them"** — the
  opposite of Go judgment. Interfaces are earned at the point of use;
  provider-side speculative interfaces are a smell.
- **Nil interface vs interface holding a nil pointer** — mentioned once in
  the lesson; don't deep-dive, but correct anyone who says "an interface
  holding nil is nil".

## Grilling points

- "Delete the `var _ io.Writer = (*CountingWriter)(nil)` line. What did you
  just lose, and when would you have found out?"
- "Why does `io.Copy(&w, …)` need the `&`? What exactly is in
  `*CountingWriter`'s method set that isn't in `CountingWriter`'s — and why
  did the language designers make that rule?"
- "`os.File` was written years before your `Saver` interface. Could an
  `*os.File`-backed store satisfy `Saver` without editing package `os`?
  What property of Go makes the answer yes?"
- "Why does `Saver` live in the `report` package instead of next to a
  storage implementation? Who imports whom, and why does the direction
  matter?"
- "Your `memStore` fake is ~10 lines. How big would it be if `Saver` had ten
  methods? What does that say about interface size?"
- "`UpperWriter.Write` sends transformed bytes downstream. If the transform
  changed the length, what should `n` be — `len(p)` or what the destination
  reported? Why?"
- "Why does `NewUpperWriter` return `*UpperWriter` and not `io.Writer`?"
  (Accept interfaces, return structs — callers keep the concrete type's full
  method set and can abstract it themselves.)

## Grading rubric

- **A** — All tests pass; `UpperWriter.Write` returns `len(p)` on success
  (not the downstream `n`); `Archive` wraps with `%w` and context; learner
  can state the method-set rule, explain implicit satisfaction and its
  package-boundary payoff, and argue for small consumer-side interfaces
  unprompted; gofmt-clean.
- **B** — Tests pass but with a contract rough edge (downstream `n`
  returned, or uppercase via manual byte loop duplicating `bytes.ToUpper`),
  or the consumer-side rationale only surfaces with prompting.
- **C** — Tests pass after heavy hinting, or the learner treats interface
  satisfaction as something the runtime figures out, or cannot explain why
  `&w` was needed. Time-boxed remediation before advancing.
- **Fail** — Tests failing, or working code the learner cannot connect to
  the method-set rule or the Writer contract. Remediate, don't advance.

## Remediation ladder

1. "Run the tests and read the first failure aloud. Which method does the
   message point at, and what did the test expect it to return?"
2. "`io.ErrShortWrite` means a Write claimed to consume fewer bytes than it
   was given. Look at your `Write`'s return values — what are you telling
   `io.Copy`?"
3. "For `UpperWriter`: you need an uppercased *copy* of `p` sent to the
   writer you saved in `NewUpperWriter`. Which `bytes` function makes that
   copy for you?"
4. "For `Archive`: loop over events; build the body with `fmt.Sprintf`; if
   `Save` fails, return `fmt.Errorf(\"…%s…: %w\", …)` — the same wrapping
   shape as the S1 errors lesson's `transfer` example."

## After passing

Preview: "Interfaces give you Go's 'has the right methods' half of design;
next lesson adds the other half — composing behavior by embedding structs and
interfaces instead of inheriting it."
