# The io Philosophy

> `go.intermediate.stdlib-io` · ~3-4h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Explain how `io.Reader`/`io.Writer` act as universal composition points
  across `os`, `net`, `bytes`, `strings`, and `compress` packages.
- Implement a custom `io.Reader` or `io.Writer` adapter that transforms data
  as it flows through (e.g. uppercase, counting).
- Choose between `bytes.Buffer`, `strings.Builder`, and `bufio` wrappers for a
  given task and justify the choice.
- Implement stream composition by chaining readers/writers instead of
  buffering whole payloads in memory, and explain the memory benefit.
- Explain the `io.Reader` contract subtleties: partial reads, returning
  `n > 0` with `io.EOF`, and why callers must handle both.

## Two tiny interfaces

In the interfaces lesson you learned that small interfaces are Go's
composition points. The two most successful interfaces in the language are
each a single method:

```go
type Reader interface {
	Read(p []byte) (n int, err error)
}

type Writer interface {
	Write(p []byte) (n int, err error)
}
```

That's the whole `io` philosophy: *a stream of bytes you can pull from* and
*a stream of bytes you can push into*. Because the interface is so small,
nearly everything satisfies it:

- `os.File` — files (you used these in the S1 file I/O lesson).
- `strings.Reader`, `bytes.Reader` — read from data already in memory.
- `bytes.Buffer` — an in-memory stream that is both a Reader and a Writer.
- `net.Conn` — network connections (you'll meet them in the advanced stage).
- `gzip.Reader`/`gzip.Writer` from `compress/gzip` — de/compression that
  wraps *any other* Reader or Writer.

The payoff is combinatorial. A function like `io.Copy(dst Writer, src Reader)`
doesn't know or care whether it's copying file→network, memory→file, or
gzip→hash. Write your code against `io.Reader` and it works with every source
that exists now or ever will — this is "accept interfaces" doing real work.

## The Reader contract

`Read` looks trivial and is the most commonly misread contract in Go. Read
its rules carefully; the exercise tests them.

1. **`Read` fills `p` with *at most* `len(p)` bytes** and reports how many it
   actually delivered as `n`. It is allowed to return fewer — even `n == 1`
   when `len(p)` is 4096. This is a *partial read* and it is normal, not an
   error. A network connection hands you whatever has arrived; a file reader
   hands you whatever one syscall produced.
2. **The end of the stream is the error `io.EOF`** — a sentinel value, not a
   failure. You compare with `err == io.EOF` (or `errors.Is`).
3. **A single call may return both data and an error**: `n > 0` together with
   `err == io.EOF` means "here are the final bytes, and that's the end."
   Readers are *allowed* to do this, so callers *must* handle it.

Rule 3 dictates the shape of every correct read loop: **process `p[:n]` first,
look at `err` second.**

```go
buf := make([]byte, 4096)
for {
	n, err := r.Read(buf)
	if n > 0 {
		process(buf[:n])        // consume the data BEFORE judging err
	}
	if err == io.EOF {
		break                   // clean end of stream
	}
	if err != nil {
		return err              // a real failure
	}
}
```

The classic bug is checking `err != nil` first and returning early — that
silently drops the last chunk of every stream whose reader delivers data with
`io.EOF`. Your tests in this lesson simulate exactly such a reader (the
standard library ships `testing/iotest` for this purpose), so the bug can't
hide.

In practice you rarely write this loop by hand: `io.ReadAll`, `io.Copy`, and
`bufio.Scanner` contain it, correctly. But when you *implement* a Reader —
next section — the contract is yours to uphold.

## Adapters: transforming the stream in flight

Because Reader is an interface, you can wrap one Reader in another that
changes the bytes on the way through. The wrapper is called an **adapter**
(you saw the pattern abstractly in the composition lesson; `gzip.Reader` is
the stdlib's flagship example). Here's an adapter that redacts digits:

```go
type redactReader struct {
	r io.Reader
}

func NewRedactReader(r io.Reader) *redactReader {
	return &redactReader{r: r}
}

func (rr *redactReader) Read(p []byte) (int, error) {
	n, err := rr.r.Read(p)      // delegate the actual reading
	for i := range n {
		if p[i] >= '0' && p[i] <= '9' {
			p[i] = 'x'
		}
	}
	return n, err               // pass n AND err through untouched
}
```

Three things make this correct, and all three are contract rules from the
previous section:

- It transforms only `p[:n]` — whatever the inner reader actually delivered.
- It transforms **before** looking at `err`, so bytes that arrive together
  with `io.EOF` are not dropped.
- It returns the inner reader's `n` and `err` unchanged: the adapter adds
  behavior, it doesn't reinterpret the stream.

Writer adapters mirror the shape: delegate to the inner `Write`, do your
extra work, return what the inner writer reported. One subtlety on that side:
a writer may perform a **short write** — accept fewer than `len(p)` bytes and
return an error. If your adapter counts or records bytes, count the returned
`n`, never `len(p)`.

Note the constructor: it accepts an `io.Reader` (any source) but returns the
concrete `*redactReader` — "accept interfaces, return structs", exactly as
`bufio.NewReader` and `gzip.NewWriter` do.

## Composition: chain, don't slurp

The beginner move for "read a file, compress it, send it" is to load the file
into a `[]byte`, compress that into another `[]byte`, then write it out. That
costs memory proportional to the payload — fine for 1KB, fatal for 10GB.

The io way is to build a pipeline and let `io.Copy` pump it:

```go
gz := gzip.NewWriter(dst)       // dst is any io.Writer
defer gz.Close()
n, err := io.Copy(gz, src)      // src is any io.Reader
```

`io.Copy` allocates one fixed 32KB buffer and loops: read a chunk, write a
chunk. Memory use is **constant** no matter how large the stream — that's the
memory benefit of chaining, and it's why Go servers can proxy gigabytes
without breaking a sweat. Each element of the chain (a counting adapter, a
compressor, a hasher) touches chunks as they flow; nobody holds the whole
payload.

`io.ReadAll` still has its place — when you genuinely need all the bytes in
memory at once (parsing a small config file, for instance). The judgment call
you're building: *does the next step need the whole payload, or just a
stream?* Prefer the stream.

## Choosing your buffer

Three tools overlap enough to confuse; pick by task:

| Tool | What it is | Reach for it when |
|------|-----------|-------------------|
| `bytes.Buffer` | Growable in-memory byte stream; both Reader **and** Writer | You need an in-memory destination or source — e.g. capturing a Writer's output in tests |
| `strings.Builder` | Write-only accumulator whose `String()` is copy-free | You're assembling a `string` result piece by piece (you used `+` concatenation in S1 — Builder avoids the quadratic copying) |
| `bufio.Reader` / `bufio.Writer` | Buffering wrappers around a *real* stream | The underlying Reader/Writer is expensive per call (files, network): batch many small reads/writes into few syscalls. `bufio.Writer` needs `Flush()` |
| `bufio.Scanner` | Line/token splitter over any Reader | Reading input line by line — it owns the read loop and hides the chunking |

Gotchas worth knowing now: `strings.Builder` cannot be read from or reset
cheaply — it's for building one result. `bufio.Scanner` has a default maximum
token size of 64KB; a longer line makes it stop with `bufio.ErrTooLong`
(raise the limit with `Scanner.Buffer` when that's a real risk). And a
forgotten `Flush()` on `bufio.Writer` is the classic "why is my file empty"
bug.

## Bytes vs runes, one more time

Adapters work on `[]byte`, and a chunk boundary can split a multi-byte UTF-8
rune in half — so byte-level transforms are only safe when they can't corrupt
multi-byte sequences. ASCII transforms qualify: recall from the S1 strings
lesson that every byte of a multi-byte rune has its high bit set (≥ 0x80), so
a transform that only touches bytes in ASCII ranges like `'a'..'z'` can never
damage a é or a 世. Anything rune-aware (proper Unicode case-mapping, say)
needs buffering logic beyond this lesson's scope — know the boundary exists.

## Exercise

Open [`exercise/`](exercise/) — a module with `streams.go` (your work sites,
marked `TODO`) and `streams_test.go`. Read the tests first: they use
`testing/iotest` to simulate hostile-but-legal readers (one byte per call;
data delivered together with `io.EOF`), which is exactly how the contract
gets enforced.

You will build four pieces:

1. **`NewUpperReader(r io.Reader) *UpperReader`** — a Reader adapter that
   uppercases ASCII letters `'a'..'z'` as data flows through; every other
   byte passes untouched.
2. **`NewCountingWriter(w io.Writer) *CountingWriter`** — a Writer adapter
   that forwards writes and tracks the total bytes *actually written*,
   reported by `Count()`.
3. **`LineCount(r io.Reader) (int, error)`** — count lines in a stream
   without loading it whole (`bufio.Scanner` is the right tool).
4. **`Shout(dst io.Writer, src io.Reader) (int64, error)`** — stream `src`
   into `dst` uppercased, by *composing* your adapter with `io.Copy`.

Acceptance criteria:

1. `UpperReader` uppercases `"go rocks"` to `"GO ROCKS"` and leaves digits,
   punctuation, and non-ASCII bytes (`"héllo"` → `"HéLLO"`) untouched.
2. `UpperReader` works with a reader that returns one byte per `Read` call,
   and never drops bytes delivered together with `io.EOF`. It propagates the
   inner reader's errors unchanged.
3. `CountingWriter` forwards bytes to the wrapped writer and `Count()`
   reports the running total. When the wrapped writer accepts only part of
   `p` and errors, `Count()` reflects the accepted bytes, and `Write` returns
   the wrapped writer's `n` and error.
4. `LineCount` handles empty input (0), a final line with no trailing
   newline, and blank lines; it reports the reader's error if reading fails.
5. `Shout` writes the uppercased stream to `dst`, returns the number of bytes
   copied, and propagates read errors. Its implementation chains
   `NewUpperReader` with `io.Copy` — no `io.ReadAll`, no whole-payload
   buffer.
6. `go test ./...` passes and the code is `gofmt`-clean.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...
```

They fail on the starter — make them green.

## Further reading

- [pkg.go.dev — io](https://pkg.go.dev/io) — read the `Reader` and `Writer`
  doc comments in full; they are the contract, stated precisely.
- [pkg.go.dev — bufio](https://pkg.go.dev/bufio)
- [pkg.go.dev — testing/iotest](https://pkg.go.dev/testing/iotest) — the
  hostile readers your tests use.
- [Go Blog — Error handling and Go](https://go.dev/blog/error-handling-and-go)
  — refresher on sentinel errors like `io.EOF`.
