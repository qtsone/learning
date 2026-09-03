# JSON & Encoding

> `go.intermediate.json-encoding` · ~2-3h · Stage: Intermediate Go

## Objectives

By the end of this lesson you can:

- Implement struct tags controlling field names, `omitempty`, and field
  skipping with `-`.
- Explain why only exported fields are marshaled and how unmarshal handles
  unknown or missing JSON fields.
- Implement a custom marshaler/unmarshaler (`MarshalJSON`/`UnmarshalJSON`) for
  a type the default encoding cannot represent.
- Choose between a pointer field and a zero value to distinguish *absent* from
  *empty* in JSON, and justify the choice.
- Implement streaming encode/decode with `json.Encoder`/`json.Decoder` over an
  `io.Reader`/`io.Writer`.

## Marshal: struct in, bytes out

JSON is the lingua franca of APIs, config files, and log pipelines, and
`encoding/json` is Go's translator. `json.Marshal` takes any Go value and
returns its JSON representation as a `[]byte`:

```go
type Task struct {
	ID    int
	Title string
}

b, err := json.Marshal(Task{ID: 7, Title: "Fix login bug"})
// b == []byte(`{"ID":7,"Title":"Fix login bug"}`)
```

Two things to notice. First, `Marshal` can fail (a channel or function value
has no JSON form), so it returns an error — handle it like any other. Second,
the JSON keys are `ID` and `Title`, capitalized exactly like the Go fields.
That is rarely what the other side of the wire wants, which is what struct
tags are for.

**Only exported fields appear.** `encoding/json` lives in a different package
from your struct, so it can only see what you export — the same visibility
rule you have relied on since the packages lesson. An unexported field is not
"skipped by choice"; it is *invisible* to the encoder, silently. A field that
mysteriously never shows up in your JSON is almost always a lowercase field
name.

## Struct tags steer the encoding

A struct tag is a string literal attached to a field. The `json:` key inside
it is a mini-language the encoder reads:

```go
type Task struct {
	ID         int      `json:"id"`
	Title      string   `json:"title"`
	Notes      string   `json:"notes,omitempty"`
	AuditToken string   `json:"-"`
}
```

- `json:"id"` — use `id` as the JSON key instead of the field name.
- `json:"notes,omitempty"` — drop the field entirely when it holds its type's
  zero value (`""`, `0`, `false`, `nil`, empty slice/map).
- `json:"-"` — never encode this field, even though it is exported. Use it for
  data other Go code needs but the wire must never see.

Tags are part of your API contract: change one and every consumer of your
JSON sees a different document. Treat renaming a tag with the same care as
renaming an exported function.

One trap worth knowing now: `omitempty` keys off the *zero value*, so it
cannot tell "the user set this to zero" from "the user never set it" — and
putting it on a field like `Done bool` would silently delete every
`"done":false` from your output, changing the document's meaning. More on
that distinction below.

## Unmarshal: bytes in, struct out

`json.Unmarshal` goes the other way. It needs a pointer — otherwise it would
be filling in a copy, which Go's pass-by-value semantics (S1 pointers lesson)
would throw away:

```go
var task Task
err := json.Unmarshal(b, &task)
```

Unmarshal is deliberately forgiving, and you should be able to predict its
behavior in three situations:

- **Unknown JSON fields are ignored.** A document with `"reporter":"zoe"` in
  it decodes fine into a struct with no such field. This is a feature: the
  other side can add fields without breaking you.
- **Missing JSON fields are left untouched.** Unmarshal only writes fields it
  finds in the document. Decode into a fresh struct and missing fields keep
  their zero values; decode into a struct that already has data and missing
  fields keep the data. Nothing tells you the field was absent.
- **Key matching is case-insensitive.** `"title"` fills `Title` even without
  a tag. Prefer exact-match tags anyway — explicit beats lucky.

If you decode into `map[string]any` instead of a struct, every JSON number
becomes a `float64` — JSON has no integer type, so the decoder cannot know
you wanted an `int`. This is why a typed struct is almost always the better
target: the types you declare are the schema you enforce.

## Absent or empty? Pointers know the difference

"Missing fields are left untouched" creates a real design problem. Suppose
you apply partial updates — a PATCH — to a task:

```json
{"title": ""}
```

Did the client ask to *clear* the title, or did they just not send it? With a
`Title string` field you cannot tell: both cases leave you staring at `""`.

The idiomatic fix is a pointer field:

```go
type TaskPatch struct {
	Title *string `json:"title"`
	Done  *bool   `json:"done"`
}
```

Now the field has three states, not two: `nil` means *absent from the
document*, a pointer to `""` means *present and empty*, a pointer to `"fix"`
means *present with a value*. After unmarshaling you check `if patch.Title !=
nil` and apply `*patch.Title`.

Do not reach for pointers everywhere. A plain value with `omitempty` is
simpler when zero and absent genuinely mean the same thing (an empty `Notes`
field). Reach for a pointer when your program must *act differently* on
absent versus zero — patches, tri-state flags, "was this configured at all?"
Justify the choice per field; that judgment is the objective.

## When the default encoding is wrong: custom marshalers

Sometimes the default representation is legal JSON but the wrong contract.
An enum stored as `type Priority int` marshals as `0`, `1`, `2` — opaque
numbers that break the moment you reorder the constants. The wire wants
`"low"`, `"medium"`, `"high"`.

`encoding/json` asks each value, via an interface, whether it would rather
encode itself:

```go
type Marshaler interface {
	MarshalJSON() ([]byte, error)
}

type Unmarshaler interface {
	UnmarshalJSON([]byte) error
}
```

This is the interfaces lesson paying rent: you never register anything.
Implement the method and the encoder discovers it through implicit
satisfaction, exactly like `io.Reader`.

```go
func (p Priority) MarshalJSON() ([]byte, error) {
	switch p {
	case Low:
		return json.Marshal("low")
	// …
	}
}

func (p *Priority) UnmarshalJSON(data []byte) error {
	var name string
	if err := json.Unmarshal(data, &name); err != nil {
		return err
	}
	// map name back to a Priority; reject unknown names
}
```

Details that matter:

- `MarshalJSON` must return *valid JSON bytes*. Returning `[]byte("low")` is
  a bug — that is not JSON. Delegating to `json.Marshal("low")` gets the
  quoting right for free.
- `UnmarshalJSON` must be on the **pointer receiver**: it mutates the
  receiver, and (methods lesson) a value receiver would mutate a copy. The
  method set rule also means only `*Priority` satisfies `Unmarshaler` — the
  decoder always has a pointer, so this works out.
- Reject inputs you do not recognize with a useful error. A silent default
  turns typos in someone's config file into mystery behavior weeks later.

## Streams: Encoder and Decoder

`json.Marshal` wants the whole value in memory as one `[]byte`. That is fine
for a config file, wrong for a million log records. The stdlib-io lesson's
philosophy applies: work against `io.Reader` and `io.Writer`, and size stops
mattering.

```go
enc := json.NewEncoder(w)   // w is any io.Writer
err := enc.Encode(task)     // writes the JSON plus a trailing '\n'
```

Because `Encode` appends a newline, calling it in a loop naturally produces
**NDJSON** (newline-delimited JSON): one object per line, the standard shape
for log shipping and data pipelines — appendable, greppable, streamable.

Decoding mirrors it. A `json.Decoder` reads *one JSON value per `Decode`
call* from an `io.Reader`, so a stream is a loop:

```go
dec := json.NewDecoder(r)
for {
	var t Task
	if err := dec.Decode(&t); err != nil {
		if errors.Is(err, io.EOF) {
			break // clean end of stream
		}
		return nil, err // truncated or malformed input
	}
	tasks = append(tasks, t)
}
```

`io.EOF` at a value boundary is the *success* case — the stream simply ended.
Any other error (including `io.ErrUnexpectedEOF`, a value cut off mid-way) is
a real failure. Distinguishing the two with `errors.Is` is the same sentinel
discipline you learned in the errors and type-assertions lessons.

`Decoder` also earns its keep on single values: decoding an HTTP body with
`json.NewDecoder(resp.Body).Decode(&v)` skips buffering the whole body into
memory first.

## Exercise

Open [`exercise/`](exercise/) — a small task-tracker core in `task.go`, with
`task_test.go` as the specification. Read the tests first.

You will finish four things:

- The struct tags on `Task`.
- `Priority`'s custom `MarshalJSON`/`UnmarshalJSON`.
- `ApplyPatch`, a partial update built on pointer fields.
- `WriteTasks`/`ReadTasks`, NDJSON streaming over `io.Writer`/`io.Reader`.

Acceptance criteria:

1. A full `Task` marshals to exactly
   `{"id":…,"title":…,"notes":…,"done":…,"priority":…,"assignee":…}` — lower-
   case keys, in that order.
2. `Notes` and `Assignee` disappear from the JSON when empty/`nil`;
   `AuditToken` never appears at all.
3. `Priority` travels as `"low"`/`"medium"`/`"high"` both directions; any
   other string is an unmarshal error.
4. Unmarshaling ignores unknown JSON fields and leaves fields missing from
   the document untouched.
5. `ApplyPatch(&task, patch)` applies exactly the fields present in the patch
   JSON — including setting a present-but-empty `"title":""` — leaves absent
   fields alone, and returns an error for malformed JSON.
6. `WriteTasks` emits one task per line via `json.Encoder`; `ReadTasks`
   consumes tasks via `json.Decoder` until a clean `io.EOF`, returning an
   error for truncated input and no error for empty input.
7. `go test ./...` passes and the code is `gofmt`-formatted.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test ./...
```

They fail on the starter — read each failure message, make it green, move to
the next.

## Further reading

- [go.dev blog — JSON and Go](https://go.dev/blog/json)
- [pkg.go.dev — encoding/json (Marshal docs cover the full tag syntax)](https://pkg.go.dev/encoding/json)
- [pkg.go.dev — json.Decoder](https://pkg.go.dev/encoding/json#Decoder)
