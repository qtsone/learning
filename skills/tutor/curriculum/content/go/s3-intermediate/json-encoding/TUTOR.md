# Tutor notes — JSON & Encoding

## Where the learner is

Seven lessons into S3. They have interfaces (implicit satisfaction,
io.Reader/Writer), composition, type assertions and `errors.Is/As`, generics,
closures, and the io philosophy from the previous lesson. This is the first
lesson where several of those threads pay off at once: `json.Marshaler` is
implicit satisfaction in the wild, `Encoder`/`Decoder` is the io philosophy
applied, and the EOF loop reuses sentinel-error discipline. Name those
connections out loud — this lesson should feel like the toolkit clicking
together, not a new subject. No concurrency, no `time` yet.

## Common misconceptions

- **Lowercase fields silently vanish** — the single most common JSON bug in
  Go. No error, no warning, the field just isn't there. If their marshal
  output is missing fields, check capitalization before anything else.
- **`omitempty` means "omit null"** — it keys off the Go zero value, not JSON
  null. Corollary they should discover: `omitempty` on a `bool` or `int`
  field silently deletes meaningful `false`/`0` values.
- **Expecting unmarshal to error on unknown or missing fields** — it does
  neither. Learners from stricter ecosystems expect schema validation; Go
  gives forgiveness by default (`Decoder.DisallowUnknownFields` exists but is
  not required here).
- **`MarshalJSON` returning raw bytes** — `[]byte("low")` instead of
  `[]byte(`+"`"+`"low"`+"`"+`)`. The resulting error (`invalid character 'l'…`)
  confuses them because it surfaces at the *call site* of `json.Marshal`, not
  in their method. Delegating to `json.Marshal("low")` fixes it.
- **`UnmarshalJSON` on a value receiver** — compiles, and the decoder still
  finds the method (value-receiver methods are in `*Priority`'s method set —
  the interfaces lesson's rule), but it mutates a copy, so the decoded value
  never lands. If priorities decode as zero, check the receiver first.
- **Treating `io.EOF` from `Decode` as a failure** — it is the clean-end
  signal. Watch for learners returning the EOF error (empty-input test fails)
  or, opposite, swallowing *all* errors (truncated-input test fails). The two
  tests exist to force the distinction.
- **Recursive `MarshalJSON`** — defining it on `Task` itself and calling
  `json.Marshal(t)` inside causes infinite recursion. Not in this exercise's
  path (the method is on `Priority`), but if they experiment, that's the
  explanation.

## Grilling points

- "Marshal a struct with a lowercase field. What happens, and why can't
  `encoding/json` do anything else?" (Visibility across packages — reflection
  sees only exported fields.)
- "How does `json.Marshal` know your `Priority` has a `MarshalJSON`? Where is
  the registration?" (There is none — implicit interface satisfaction; tie to
  `io.Reader`.)
- "Why is `UnmarshalJSON` on `*Priority` but `MarshalJSON` on `Priority`?
  What breaks if you swap each?" (Mutation needs the pointer; method sets.)
- "In `ApplyPatch`, why `*string` and not `string` plus `omitempty`? What
  exact input becomes indistinguishable with plain `string`?" (`{"title":""}`
  vs `{}` — they should produce this example unprompted.)
- "Your `ReadTasks` loop gets an error from `Decode`. Walk me through how you
  decide whether to return it." (`errors.Is(err, io.EOF)` = clean end;
  anything else, including `io.ErrUnexpectedEOF`, is real.)
- "Why does `WriteTasks` take an `io.Writer` and not return a `[]byte` or
  take an `*os.File`?" (Previous lesson's philosophy — callers compose;
  tests use `bytes.Buffer`.)
- Stretch: "Decode `{"n":1}` into `map[string]any`. What is the dynamic type
  of the value, and why?" (`float64`; JSON has no int type.)

## Grading rubric

- **A** — All tests pass; tags exactly right; `UnmarshalJSON` rejects unknown
  names with an informative error; `ApplyPatch` is a straight nil-check
  cascade; `ReadTasks` distinguishes `io.EOF` via `errors.Is` (or `==`,
  if they can say why that also works here); errors wrapped with context
  (`%w`); can justify pointer-vs-zero-value per field.
- **B** — Tests pass but with rough edges: string-comparison on the EOF
  error, unwrapped bare error returns, an unnecessary intermediate struct, or
  the pointer-field rationale only half-articulated.
- **C** — Tests pass after heavy hinting, or working code with a wrong mental
  model (e.g. believes tags are required for unmarshal to work at all, or
  can't explain why `AuditToken` needs `json:"-"` while a lowercase field
  wouldn't). Time-boxed remediation before advancing.
- **Fail** — Tests failing, or a solution they cannot walk through — e.g.
  cannot explain what makes `Decode` stop or why the patch fields are
  pointers. Remediate from the misconception list.

## Remediation ladder

1. "Run `go test` and read the first failure only. Is the problem in the
   bytes you produced or the struct you filled?" (Point them at the exact
   got/want diff.)
2. Tags: "Marshal a two-field struct with no tags in the playground. What
   keys do you get? Now add one tag and re-run." Marshalers: "What does your
   `MarshalJSON` return for `High`, byte for byte? Is that valid JSON on its
   own?"
3. Patch: "Print `patch.Title == nil` for inputs `{}` and `{"title":""}`.
   Two different answers — which test does that unlock?" Streaming: "What
   error value does `Decode` return when the input simply runs out? Which
   package defines it?"
4. Sketch the shape verbally — switch in each marshaler, nil-check cascade in
   `ApplyPatch`, `for { Decode; if EOF break }` in `ReadTasks` — and let them
   type every line.

## After passing

Preview: "Next is `time` — and now that you own custom marshalers, you'll
recognize why `time.Time` ships its own: RFC 3339 strings on the wire is
exactly the trick you just built for Priority."
