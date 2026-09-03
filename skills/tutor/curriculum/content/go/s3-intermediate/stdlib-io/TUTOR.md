# Tutor notes — The io Philosophy

## Where the learner is

Mid-S3: they own interfaces, embedding, type assertions, generics, and
closures, and they did concrete file I/O (`os.ReadFile`, `os.WriteFile`) back
in S1. This is their first time *implementing* a stdlib interface contract
rather than just satisfying a signature — the leap from "my type has a Read
method" to "my type upholds the Read contract" is the whole lesson. No
concurrency exists yet; don't reach for pipelines-with-goroutines analogies.

## Common misconceptions

- **"Read fills the buffer"** — expecting `n == len(p)`. Partial reads are
  normal; `iotest.OneByteReader` in the tests exists to break this belief.
- **"err != nil means no data"** — checking the error before consuming
  `p[:n]`, which drops final bytes when a reader returns `n > 0` with
  `io.EOF`. The `DataErrReader` test catches it; this is the lesson's #1
  target bug.
- **"io.EOF is a failure"** — treating it like a real error and wrapping or
  logging it. It's the normal end-of-stream sentinel.
- **Counting `len(p)` instead of `n` in `CountingWriter`** — ignores short
  writes; the `chokedWriter` test catches it.
- **Adapter swallows or rewrites the inner error** — e.g. converting `io.EOF`
  to `nil` or wrapping it, which breaks every caller's termination check.
- **`io.ReadAll` reflex** — implementing `Shout` or `LineCount` by slurping
  the whole input, then operating in memory. Tests may still pass; the
  *grading conversation* must catch it (ask about a 10GB input).
- **`bytes.Buffer` vs `strings.Builder` confusion** — using a Buffer to build
  a string result, or trying to read from a Builder.

## Grilling points

- "Your `Read` gets `n = 5, err = io.EOF` from the inner reader. Walk me
  through exactly what your code does, line by line, and why the order
  matters."
- "Why does `NewUpperReader` accept an `io.Reader` but return a
  `*UpperReader`? Where have you seen that shape in the stdlib?"
- "`Shout` gets a 10GB file on a machine with 1GB of RAM. Does your
  implementation survive? Why — where does the memory ceiling come from?"
  (io.Copy's fixed 32KB buffer.)
- "Why is byte-wise uppercasing safe for UTF-8 input when a chunk boundary
  can split a rune in half?" (ASCII range bytes never occur inside a
  multi-byte sequence — high bit set.)
- "You need to read a 200MB log file line by line. Which tool, and what's the
  64KB catch?" (`bufio.Scanner`, `ErrTooLong`, `Scanner.Buffer`.)
- "When is `io.ReadAll` the *right* call?" (Small payloads you genuinely need
  whole — judgment, not dogma.)

## Grading rubric

- **A** — All tests pass; `Read` transforms `p[:n]` before checking `err` and
  returns both unchanged; `Write` counts `n`, not `len(p)`; `Shout` is a
  one-liner composing the adapter with `io.Copy`; `LineCount` uses
  `bufio.Scanner` and returns `sc.Err()`. Learner explains the n>0+EOF case
  and the constant-memory argument unprompted and crisply.
- **B** — Tests pass but with roughness: an extra intermediate buffer in
  `Read`, a hand-rolled read loop in `Shout` that is nonetheless streaming
  and correct, or a shaky-but-recoverable explanation of the Reader contract.
- **C** — Tests pass only after heavy hinting, or the code passes while the
  learner can't explain *why* the DataErrReader test exists. Pass only if
  remediation lands within the session; otherwise iterate.
- **Fail** — Tests failing; or `Shout`/`LineCount` buffer the whole payload
  and the learner sees nothing wrong with that after the 10GB question; or
  the contract explanation is guesswork. Remediate, don't advance.

## Remediation ladder

1. "Run `go test -run TestUpperReaderDataWithEOF -v`. Read the failure aloud:
   what came back, what was expected? Now read what `iotest.DataErrReader`
   does in its doc comment."
2. "In your `Read`, what do you do first — look at the bytes, or look at the
   error? The contract says a reader may hand you both at once. Which order
   survives that?"
3. "Sketch the happy path: `n, err := u.r.Read(p)`. You now own `p[:n]`.
   What two things are left to do, and what must you *not* change?"
4. Walk the shape verbally — delegate, loop over `p[:n]` uppercasing
   `'a'..'z'`, `return n, err` — and have them type it; then point the same
   before-err discipline at `CountingWriter`'s short-write case.

## After passing

Preview: "Next is JSON — and now that you speak io, you'll see why
`json.NewDecoder` takes an `io.Reader` instead of a `[]byte`."
