# Building REST Services

> `go.advanced.rest-services` · ~3-5h · Stage: Advanced Go

## Objectives

By the end of this lesson you can:

- Structure a REST service into handler, service, and storage layers, and
  justify the dependency direction between them.
- Implement request decoding and validation that returns field-level 400
  errors instead of panicking or leaking internals.
- Design an error-mapping layer that translates domain errors to appropriate
  HTTP status codes in exactly one place.
- Implement consistent JSON response envelopes for success and error cases,
  including correct status codes for create/update/delete.
- Explain why handlers should stay thin and demonstrate it by unit-testing
  business logic without httptest.

## The handler that ate the program

In the last lesson your handlers were the whole program: they parsed the
request, made the decision, and wrote the response. That is fine for three
endpoints. It stops being fine the day you have fifteen endpoints, each one
re-implementing "decode JSON, check the input, hit the store, pick a status
code" with slightly different bugs. Every rule lives in a function you can
only reach through HTTP, so every test is an httptest ceremony, and changing
the storage means editing every handler.

The fix is the oldest one in software: layers. A REST service worth shipping
separates into three:

```
        api          the HTTP edge: routing, decode, encode, status codes
         │  calls
         ▼
       note          the domain: types, rules, error vocabulary, Service
         ▲  implements
         │
      memstore       storage: keeps notes; today a map, later a database
```

- **api** (the handler layer) speaks HTTP and nothing else. It turns requests
  into plain Go values, calls the service, and turns results — including
  errors — back into responses.
- **note** (the domain/service layer) holds the business rules: what a valid
  note is, how lists are ordered, what "not found" means. It has no idea HTTP
  exists — no `net/http` import anywhere in it.
- **memstore** (the storage layer) persists notes. Today that is a
  mutex-guarded map; a database-backed store slots in behind the same
  interface without anything above it noticing.

## Dependencies point at the domain

Look at the arrows again: `api` imports `note`, and `memstore` imports `note`.
Nothing imports `api` or `memstore` (except `main`, which wires the pieces
together). The domain sits at the center and depends on nobody.

The trick that makes this work is one you already know from the interfaces
lesson — *define interfaces where they are consumed*. The `note` package
declares what it needs from storage:

```go
// In package note — the CONSUMER of storage.
type Store interface {
	Create(d Draft) (Note, error)
	Get(id int64) (Note, error)
	List() ([]Note, error)
	Update(id int64, d Draft) (Note, error)
	Delete(id int64) error
}
```

and `memstore.Store` satisfies it implicitly, without ever being mentioned by
name in `note`. The payoff is concrete:

- **Swapping storage is additive.** A SQLite implementation of this same
  interface is one new package plus one line in `main` — `note` and `api` do
  not change by a line. The databases lesson next builds exactly that
  machinery (migrations, pools, transactions) against a different domain, and
  the capstone puts a SQL store behind a consumer-declared interface just like
  this one.
- **The domain stays testable in isolation.** A test can hand `NewService`
  any `Store` — the real map store, or a five-line stub that fails on demand.
- **The error vocabulary belongs to the domain.** `note.ErrNotFound` is
  defined next to the interface, and every implementation contracts to return
  it. Storage speaks the domain's language, not the other way round.

If you ever find yourself wanting `note` to import `memstore`, stop: you are
about to point the arrow backwards, and the compiler will eventually agree
with you via an import cycle.

## Thin handlers

With the layers in place, a handler's whole job is three verbs: **decode,
delegate, encode**.

```go
func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	d, err := decodeDraft(r)
	if err != nil {
		respondError(w, err)
		return
	}
	n, err := s.svc.Create(d)
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, http.StatusCreated, n)
}
```

No rules, no storage, no status-code decisions beyond the happy path. The
working heuristic: a handler that grows past about ten lines is doing a lower
layer's job. The test suite enforces the same split — everything in
`note/service_test.go` runs without importing `net/http/httptest`, because
the rules it checks (trimming, validation, ordering) never needed HTTP in the
first place. Unit tests of plain functions are faster to write, faster to
run, and point at the broken rule instead of at a status code.

## Decode, then validate — and answer with data

Decoding is the easy half; you did it in the JSON lesson:

```go
var d note.Draft
if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
	// Malformed JSON: the client's fault, and fixable by the client → 400.
}
```

Validation is where services usually go wrong, in one of two directions:
panicking on bad input (a client can now crash your goroutine at will — the
security lesson should be ringing in your ears), or answering `"bad
request"` with no hint of *what* was bad. The idiomatic middle is to treat
validation failures as **data**:

```go
// In package note.
type ValidationError map[string]string // field name → what is wrong

func (e ValidationError) Error() string { … }
```

The service builds one — `{"title": "required"}` — and returns it as an
ordinary `error`. The HTTP layer recognizes the type and renders a
field-level 400 the client can actually act on. Two habits to bake in:

- **Normalize before you validate.** Trim whitespace first, so `"   "` fails
  the "required" check instead of sneaking through as a non-empty title.
- **Count characters, not bytes.** A limit of 120 means 120 *runes*
  (`utf8.RuneCountInString`); `len()` on a string counts bytes and would
  reject 61 accented characters.

One trap to know about the typed-nil kind (remember the interfaces lesson):
if you build an empty `ValidationError{}` and return it, the caller's
`err != nil` is *true* — a non-nil interface holding an empty map. Only
return the map when it has entries; otherwise return a literal `nil`.

## Map errors to statuses in exactly one place

By the time an error reaches the handler it is a domain fact: "the input was
invalid", "there is no note 7", "storage failed". Turning those facts into
HTTP is a *translation*, and translations drift when every handler does its
own. The cure is a single function — the only place in the program where an
error meets a status code:

```go
func respondError(w http.ResponseWriter, err error) {
	var ve note.ValidationError
	var br badRequest
	switch {
	case errors.As(err, &ve):
		// 400 + the field map
	case errors.Is(err, note.ErrNotFound):
		// 404
	case errors.As(err, &br):
		// 400 + br's message (malformed JSON, non-integer id)
	default:
		slog.Error("request failed", "err", err)
		// 500 — and the body says "internal error", nothing more
	}
}
```

This is your S3 error toolkit doing production work: sentinels compared with
`errors.Is`, typed errors extracted with `errors.As`. Adding a domain error
later (say, `ErrConflict` → 409) is one new case here — not a hunt through
every handler.

The `default` arm carries a rule you must never bend: **the client gets a
generic message; the log gets the truth.** A raw error string like
`dial tcp 10.0.0.7:5432: connection refused` in a response body hands an
attacker your topology and your users a stack of confusion. `log/slog` exists
so operators can see the real error, structured, on the server side.

## Envelopes and status codes

Clients live easier when every response has the same shape. This service uses
a minimal envelope pair:

```json
{"data": {"id": 1, "title": "first", "content": "hello"}}

{"error": {"message": "validation failed", "fields": {"title": "required"}}}
```

Success is always under `"data"`, failure always under `"error"` — a client
can branch on which key exists before caring what is inside. Set
`Content-Type: application/json` before writing, and remember the JSON
lesson's nil-slice gotcha: an empty listing must encode as `[]`, not `null`,
so return an empty slice, never a nil one.

Status codes for CRUD are convention, and clients depend on the convention:

| Operation           | Success            | Notes                              |
|---------------------|--------------------|------------------------------------|
| `POST /notes`       | **201 Created**    | body: the created note, with id    |
| `GET /notes`        | **200 OK**         | body: array, `[]` when empty       |
| `GET /notes/{id}`   | **200 OK**         | unknown id → 404                   |
| `PUT /notes/{id}`   | **200 OK**         | full replace; validated like POST  |
| `DELETE /notes/{id}`| **204 No Content** | *no body at all* — 204 promises it |

Client-fault failures are 400 (malformed JSON, invalid fields, non-integer
id), missing resources are 404, and anything the client cannot fix is 500.

Routing is the 1.22 mux you met last lesson — method plus pattern, `{id}`
read back with `r.PathValue("id")`:

```go
mux.HandleFunc("POST /notes", s.handleCreate)
mux.HandleFunc("GET /notes/{id}", s.handleGet)
```

The mux also answers 404 for unknown paths and **405 Method Not Allowed**
for known paths with the wrong method — behavior you get for free and should
not reimplement.

## Exercise

Open [`exercise/`](exercise/) — a notes service laid out in the three layers:

```
cmd/notesd/   main: wiring, timeouts, shutdown     (provided)
note/         domain: types, errors, Store, Service
memstore/     map-backed note.Store                (provided)
api/          HTTP edge: routes, handlers, envelopes
```

`note/note.go`, `memstore/`, `api/middleware.go`, and `cmd/notesd/` are
complete — read them, they are part of the lesson. `cmd/notesd/main.go` is
last lesson's server shape reused without ceremony: every timeout set, a
listener you could point at port 0 in a test, `signal.NotifyContext`, and
`Shutdown` on the way out. Your work sites are the
`TODO`s in **`note/service.go`** and **`api/api.go`**. The two test files are
the specification; note which one never imports httptest.

Acceptance criteria:

1. `POST /notes` with a valid draft returns **201** and a
   `{"data": {…}}` envelope carrying the stored note (server-assigned id),
   with `Content-Type: application/json`.
2. A body that is not valid JSON returns **400** with
   `{"error": {"message": "invalid JSON"}}`.
3. An invalid draft returns **400** with message `validation failed` and a
   `fields` map. Rules (applied after trimming whitespace from title and
   content): empty title → `required`; title over 120 characters →
   `must be at most 120 characters`; content over 8000 characters →
   `must be at most 8000 characters`. Limits count runes, not bytes.
4. `GET /notes` returns **200** with notes sorted by id ascending, and
   exactly `{"data":[]}` when the service is empty.
5. `GET /notes/{id}` and `PUT /notes/{id}` return **200** with the note;
   `DELETE /notes/{id}` returns **204** with an empty body. An unknown id
   maps to **404** `note not found`; a non-integer id to **400**
   `invalid id`.
6. A failing store maps to **500** with exactly
   `{"error": {"message": "internal error"}}` — the underlying error appears
   in the slog output, never in the response body.
7. A wrong method on a known path answers **405** without any code of yours.
8. The service-layer rules — trimming, validation, id-sorted non-nil `List`,
   store errors passed through — pass their unit tests in
   `note/service_test.go`, which uses no httptest.
9. `go test -race ./...` is green and the code is `gofmt`-formatted.

Run the tests from the module root:

```sh
cd exercise
go test -race ./...
```

Both packages fail on the starter. Start with `note/service.go` — get the
domain green without touching HTTP — then build the edge in `api/api.go`.
When everything passes, run the server (`go run ./cmd/notesd`) and poke it
with `curl` for the satisfaction of it.

## Further reading

- [go.dev blog — Routing Enhancements for Go 1.22](https://go.dev/blog/routing-enhancements)
- [pkg.go.dev — net/http ServeMux (pattern syntax)](https://pkg.go.dev/net/http#ServeMux)
- [pkg.go.dev — log/slog](https://pkg.go.dev/log/slog)
- [Grafana blog — How I write HTTP services in Go after 13 years (Mat Ryer)](https://grafana.com/blog/2024/02/09/how-i-write-http-services-in-go-after-13-years/)
