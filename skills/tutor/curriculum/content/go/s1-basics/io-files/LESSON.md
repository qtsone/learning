# I/O & Files

> `go.basics.io-files` · ~3-4h · Stage: Programming Basics (Go)

## Objectives

By the end of this lesson you can:

- Read and write whole files with the `os` package, handling errors at every step.
- Explain what `defer` guarantees and use it to close files reliably, including its LIFO order.
- Use `bufio.Scanner` to process a file line by line and explain why buffering matters.
- Choose between reading a whole file into memory and streaming it, and justify the choice.

## A file is just bytes

Everything your programs have computed so far vanished the moment they exited:
variables, slices, maps — all of it lives in memory, and memory is wiped when
the process ends. Files are how a program leaves something behind. To the
operating system a file is a named sequence of bytes on disk, nothing more —
"text file" just means the bytes happen to be readable text.

Go's doorway to files is the **os** package. You name a file by its *path* —
absolute (`/home/ada/notes.txt`) or relative to the program's working
directory (`notes.txt`), exactly like in your S0 terminal lessons. Relative
paths are resolved against wherever you *run* the program, not where the
source code sits — a classic source of "file not found" confusion.

## Whole files in one call

The simplest file operations swallow or produce the entire file at once:

```go
data, err := os.ReadFile("notes.txt") // data is []byte — a slice of bytes
if err != nil {
	return err
}
fmt.Println(string(data)) // convert bytes to text, like in the strings lesson

err = os.WriteFile("notes.txt", []byte("feed the cat\n"), 0o644)
```

Two things to notice:

- Files hold bytes, so both functions traffic in `[]byte`. You already know
  the conversions from the strings lesson: `string(data)` and `[]byte(s)`.
- `os.WriteFile` needs permission bits for the case where it *creates* the
  file. `0o644` is octal notation for Unix permission bits: you can read and
  write the file, everyone else can only read — the everyday default.

`os.WriteFile` **replaces** the whole file — if it existed, the old content is
gone. That sounds destructive, but it makes a great building block: read the
file, change your data in memory, write it all back.

## Errors at every step

Files fail for reasons your code cannot prevent: the file doesn't exist, the
disk is full, permissions forbid access, the path is a directory. That's why
*every* `os` call returns an `error` — and why you check every one, the way
the errors lesson drilled. Two habits matter here:

```go
data, err := os.ReadFile(path)
if errors.Is(err, os.ErrNotExist) {
	// A brand-new notebook: not a failure, just nothing saved yet.
	return nil, nil
}
if err != nil {
	return nil, fmt.Errorf("load notebook %s: %w", path, err)
}
```

- `os.ErrNotExist` is a **sentinel error** (remember those?). You test for it
  with `errors.Is`, never `==`, because `os` functions actually return a
  `*PathError` that *wraps* the sentinel — `errors.Is` walks the chain, `==`
  compares only the outer value.
- Wrapping with `%w` adds *which file, doing what* while keeping the original
  error inspectable. "no such file or directory" alone is useless three
  functions up the stack.

## Open, use, close

`os.ReadFile` and `os.WriteFile` open and close the file for you. For anything
finer-grained — appending, streaming — you manage the file yourself:

```go
f, err := os.Open(path) // read-only; returns *os.File
```

An `*os.File` is a handle the operating system lends your process (a *file
descriptor*). The OS caps how many a process may hold, so every successful
open must be paired with `f.Close()` — a program that forgets leaks handles
until opens start failing. The general opener is `os.OpenFile`, where flags
say what you intend:

```go
f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
```

Read the flags as a sentence: write-only, create it if missing, and every
write goes to the end. That combination is how logs and journals grow.

## defer: cleanup that cannot be forgotten

Pairing every open with a close sounds easy until a function has three early
`return err` paths — do you repeat `f.Close()` before each one? Go's answer is
**defer**:

```go
f, err := os.Open(path)
if err != nil {
	return err
}
defer f.Close()
// ... every return below this line closes f automatically
```

`defer` schedules the call to run **when the surrounding function returns —
no matter how**: the normal path, any early `return`, even a panic. Write
`defer f.Close()` on the line after the error check and closing is simply
solved; no return path can forget it.

Multiple defers run in **LIFO order** — last deferred, first run:

```go
defer fmt.Println("world") // scheduled first, runs second
defer fmt.Println("hello") // scheduled second, runs FIRST
```

That reversal is deliberate: resources acquired later often depend on ones
acquired earlier, so they must be released first — cleanup unwinds in the
opposite order of setup, like closing nested boxes.

One caveat before you defer everything: when **writing**, `Close` itself can
fail — the OS may only flush data to disk at close time, so an error there
means *your data didn't make it*. On a write path, check `Close`'s error
explicitly (or sidestep the issue with `os.WriteFile`). `defer f.Close()`
with the error ignored is the accepted idiom for *reading* only.

## Line by line with bufio.Scanner

Reading a whole file means holding the whole file in memory. For a config
file, fine; for a multi-gigabyte log, not fine. The alternative is
**streaming**: read a piece, process it, move on. The tool for
line-oriented streaming is `bufio.Scanner`:

```go
f, err := os.Open(path)
if err != nil {
	return err
}
defer f.Close()

scanner := bufio.NewScanner(f)
for scanner.Scan() { // advances to the next line; false when done
	line := scanner.Text() // the current line, newline stripped
	fmt.Println(line)
}
if err := scanner.Err(); err != nil { // did "done" mean EOF or failure?
	return err
}
```

`Scan` returns `false` both at a clean end of file **and** when a read fails
mid-way. The loop can't tell the difference — `scanner.Err()` after the loop
can (it's `nil` after a clean end). Skipping that check silently treats a
half-read file as a complete one. It is the single most-forgotten line in Go
file handling; don't be that program.

Why "bufio"? Every read from a file is a **system call** — a request to the
OS kernel, orders of magnitude slower than touching memory. Reading a file
byte-by-byte would mean millions of them. `bufio` reads a large chunk per
system call into a memory buffer, then slices your lines out of that buffer
for near-free. Buffering is the difference between I/O that crawls and I/O
you never think about.

## Whole file or streaming?

You now have two ways to read. Choose deliberately:

- **Whole file** (`os.ReadFile`) when the file is small, you need all of it,
  and simplicity wins — configs, saved app state, anything you'll write back
  in full. Memory cost: the entire file.
- **Streaming** (`os.Open` + `bufio.Scanner`) when the file may be large or
  unbounded, you process sequentially, or you can stop early once you've
  found what you need. Memory cost: one buffer, regardless of file size.

The axis is *memory versus simplicity*. The upcoming capstone tracker keeps
its data in a small file, loads it whole, updates in memory, and saves it
back — the whole-file pattern is exactly right there. `grep` over a 10 GB
log would be streaming, no debate.

## Exercise

Open [`exercise/`](exercise/) — package `notes`, a plain-text notebook that
stores one note per line. `notes.go` declares four functions with `TODO`s;
`notes_test.go` is the specification (the tests use `t.TempDir()`, which
gives each test a throwaway directory, so nothing touches your real files).

Acceptance criteria:

1. `Save(path, notes)` writes each note followed by `"\n"` in one
   `os.WriteFile` call, replacing the file; an empty notebook writes an
   empty file.
2. `Load(path)` reads the whole file and returns the notes; a missing file
   means an empty notebook (no error, use `errors.Is` with
   `os.ErrNotExist`); an empty file means no notes; `Load` after `Save`
   returns exactly what was saved.
3. `Append(path, note)` adds one line to the end via `os.OpenFile` with
   `O_APPEND|O_CREATE|O_WRONLY`, keeping existing notes and checking the
   `Close` error.
4. `Search(path, keyword)` streams with `os.Open` + `defer f.Close()` +
   `bufio.Scanner`, returns the lines containing `keyword` in file order,
   checks `scanner.Err()`, and reports a missing file as an error that
   wraps `os.ErrNotExist`.
5. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...
```

They fail on the starter — make them green, one function at a time (the test
names tell you which function each one exercises).

## Further reading

- [pkg.go.dev/os — file operations](https://pkg.go.dev/os)
- [pkg.go.dev/bufio — Scanner](https://pkg.go.dev/bufio#Scanner)
- [Effective Go — Defer](https://go.dev/doc/effective_go#defer)
- [Go blog — Defer, Panic, and Recover](https://go.dev/blog/defer-panic-and-recover)
