# Tutor notes — I/O & Files

## Where the learner is

Last lesson before the stage capstone. They can write functions, slices, maps,
structs, methods; they wrap errors with `%w` and test sentinels with
`errors.Is`; they know `strings.Builder`/`Contains`/`Split` and read
table-driven tests fluently. This is the first time their programs touch the
disk and their first encounter with `defer`. The notebook exercise is a
deliberate rehearsal of the capstone's persistence layer — say so if
motivation flags.

## Common misconceptions

- **"defer runs at the end of the block"** — it runs when the *function*
  returns. A `defer` inside a loop stacks up one pending call per iteration
  until the function exits; that's a resource leak, not cleanup.
- **`defer f.Close()` everywhere** — fine for reads; on writes a failing
  `Close` can mean lost data, so its error must be checked. The exercise
  forces this in `Append`.
- **Skipping `scanner.Err()`** — `Scan()` returns `false` for both EOF and a
  read failure; without the check a half-read file looks complete.
- **`err == os.ErrNotExist`** — `os` returns a `*PathError` *wrapping* the
  sentinel; only `errors.Is` sees through it. If they hit this, revisit the
  errors lesson's wrapping model.
- **Thinking `os.WriteFile` appends** — it truncates and replaces. Learners
  who "lose" notes in `TestAppendKeepsSavedNotes` usually implemented Append
  as read-modify-WriteFile or plain WriteFile.
- **`Load` returning one blank note for an empty file** —
  `strings.Split("", "\n")` yields `[""]`; the empty-file case needs explicit
  handling. Have them print intermediate values with `%q`.
- **Relative-path confusion** — "file not found" because the path is resolved
  against the working directory, not the source file. The tests dodge this
  with `t.TempDir()`, but it resurfaces the moment they experiment manually.

## Grilling points

- "Walk me through `Search` when the file opens fine but a read fails halfway.
  Where does your code notice, and what would happen without that line?"
- "Your function defers two `Close` calls on two files. Which runs first, and
  why is that the right order for nested resources?"
- "The notebook file grows to 10 GB. Which of your four functions still
  behave well? Which would you rewrite, and to what?" (Load/Save slurp;
  Search already streams; Append is fine.)
- "Why is a missing file 'empty notebook' in `Load` but an error in `Search`?
  Could you defend the opposite choice?" (The point: semantics are a design
  decision in the function's contract, not a property of the OS error.)
- "Read `os.O_APPEND|os.O_CREATE|os.O_WRONLY` aloud as a sentence. What does
  each flag contribute, and what breaks if you drop each one?"
- "Why does `bufio` exist — what exactly is expensive about unbuffered reads?"

## Grading rubric

- **A** — All tests pass. `Save`/`Load` use whole-file `os` calls; `Search`
  streams with `os.Open` + `defer f.Close()` + `bufio.Scanner` and checks
  `scanner.Err()`; `Append` uses `O_APPEND` and checks `Close`'s error;
  errors are wrapped with file context via `%w`; gofmt-clean; learner can
  justify whole-file vs streaming per function and explain defer's guarantee
  and LIFO order unprompted.
- **B** — Tests pass but with rough edges: `Search` slurps the file instead
  of streaming, `scanner.Err()` or `Close` errors unchecked, bare errors
  returned without context. Explanation mostly solid. Fix the edges in
  conversation before moving on.
- **C** — Tests pass only after heavy hinting, or the learner cannot say what
  `defer` guarantees / why `errors.Is` is needed. Time-boxed remediation on
  the shaky concept; pass only if it lands.
- **Fail** — Tests failing, or the solution shape was dictated and cannot be
  explained back. Remediate, don't advance — the capstone leans on every
  pattern here.

## Remediation ladder

1. "Run `go test ./...` and read the first failure aloud: which function,
   what did it expect, what did it get?"
2. For `Load`/`Save` mismatches: "Print the exact file content with `%q` —
   what does it end with? Now what does your split produce?"
3. For `Search`: "List the streaming steps in order: open, check err, defer
   close, scan loop, check scanner.Err. Point at each line in your code —
   which is missing?"
4. Sketch the missing function's skeleton verbally (open / error check /
   defer / loop / final err check), then let them type every line.

## After passing

Preview: "Next is the stage capstone — a small CLI tracker. Your Save/Load
pair is its persistence layer almost verbatim; you'll bolt a command-line
interface and your structs/maps knowledge on top."
