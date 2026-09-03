# Tutor notes — Building REST Services

## Where the learner is

Second lesson of S5, straight after http-servers. They can route with the 1.22
mux, write middleware, and drive handlers with httptest, but everything so far
lived in one package where handlers did all the work. This lesson is the first
time they are asked to *shape* a program rather than make it work: three
packages, an interface declared on the consumer side, and one error-mapping
function. They already have the ingredients — S3's `errors.Is/As` and
interfaces, S4's JSON and testing habits — so the difficulty is architectural
judgment, not syntax.

Expect the domain half (`note/service.go`) to go quickly and the edge half
(`api/api.go`) to be where they slow down, mostly on envelopes and on deciding
which errors belong to which layer.

## Common misconceptions

- **"Layers are ceremony for big projects."** Push back with the test suite,
  not with theory: `note/service_test.go` checks trimming, rune limits, and
  ordering without a single HTTP call. Ask what that test would have to look
  like if the rules lived in the handler.
- **Pointing the dependency arrow backwards** — wanting `note` to import
  `memstore` (or `api`). They usually notice only when the compiler reports an
  import cycle. Reinforce: the interface is declared by the *consumer*; storage
  implements the domain's vocabulary, never the reverse.
- **Validation as prose** — returning `errors.New("title is required")`. It
  passes nothing structured to the client and forces the handler to string-match.
  `ValidationError` is *data* precisely so the HTTP layer can render fields.
- **The typed-nil trap** — building `ValidationError{}`, finding it empty, and
  returning it anyway. `err != nil` is then true and every valid request 400s.
  If their whole suite fails with "validation failed" on good input, this is it.
- **`len(s)` for character limits** — silently correct for ASCII, wrong for
  every accented title. `TestLimitsCountRunesNotBytes` exists to catch exactly
  this; make sure they understand *why* it passed 120 `é` and not 121 `x`.
- **Validating before trimming** — a title of `"   "` sails through the
  required check and gets stored as empty. Order is a rule, not a preference.
- **Mapping errors per handler** — a `switch` in every handler, or worse, an
  `if err == note.ErrNotFound` chain. Ask them where they would add a 409.
- **Leaking the store error into the body** — `http.Error(w, err.Error(), 500)`
  is muscle memory from small programs. The test asserts the response contains
  neither the host nor "connection refused"; connect it to the S4 security lesson.
- **Writing a body after `WriteHeader(204)`** — or setting Content-Type for it.
  204 promises no content; `rec.Body.Len() != 0` catches it.
- **Reimplementing 405** — hand-rolling method checks inside handlers because
  they forgot the mux does it once patterns carry methods.
- **Nil slice from `List`** — encodes as `null`, and the empty-list test compares
  the body byte for byte. Trace it back to the JSON lesson's nil-vs-empty rule.

## Grilling points

- "Your service moves to PostgreSQL next month. Name every file that changes."
  (One: a new store implementation, plus the wiring line in `main`. If they
  name `api` or `note`, the layering hasn't landed.)
- "`note` never imports `net/http`. What would you lose the day someone adds
  that import?" (Testability, reuse from a CLI or a queue consumer, and the
  ability to reason about rules without a request in hand.)
- "Show me `service_test.go` and `api_test.go` side by side. What does each one
  actually protect, and which would you rather have 50 of?"
- "A new rule: creating a note with a duplicate title is a conflict, 409. Where
  does the sentinel live, where does the status appear, and how many files did
  you touch?" (`note` for the sentinel, one new case in `respondError`.)
- "Why is `invalid id` a 400 from `badRequest` rather than something the domain
  returns?" (The domain speaks `int64`; a non-integer path segment never becomes
  a domain concept — it dies at the edge.)
- "Someone proposes returning `{"data": null, "error": null}` on every response
  so clients always see both keys. Argue for and against."
- "The 500 test asserts the body says nothing useful. Who *does* get the truth,
  and how would you find it at 3am?" (slog, structured, server side — set up the
  observability lesson.)
- "Your `Update` is a full replace. What would change if it were a PATCH?"
  (Partial updates need pointer/optional fields and a merge; validation moves
  to the merged result — good hint that PUT was the simpler choice here.)

## Grading rubric

- **A** — All tests green under `-race`; handlers are decode/delegate/encode and
  none is over ~10 lines; exactly one function maps errors to statuses; trimming
  happens before validation and limits count runes; `validate` returns a literal
  `nil` when clean; `List` is sorted and non-nil; 204 has no body; can explain
  the dependency direction and where a new domain error would go.
- **B** — Tests green with rough edges: validation logic duplicated between
  `Create` and `Update`, status codes decided in two places, envelope written
  ad hoc per handler, or `fmt.Sprintf` of the limit hard-coded away from the
  constants. Explanation of layering is right but shallow.
- **C** — Tests pass only after the remediation ladder, or they cannot say why
  `service_test.go` needs no httptest, or they think `memstore` is where the
  rules live. Time-box remediation and re-grill before advancing.
- **Fail** — Tests failing, the store error visible in a response body, or the
  layer split undone (rules moved into handlers to "make it simpler").
  Remediate from `note/service.go` upward — the domain must be green first.

## Remediation ladder

1. "Run `go test -race ./note/` alone. The domain is the foundation; get that
   green before you open `api.go`. Read the first failure aloud — which rule
   from the acceptance criteria does it name?"
2. Domain stuck: "Forget HTTP entirely. Write the three steps `Create` performs
   in plain English, in order." (Normalize → validate → store.) If validation
   is the sticking point: "What does the test compare `ve` against? Build that
   exact map, then decide when to return it and when to return `nil`."
3. Edge stuck: "Pick one endpoint — `GET /notes/{id}`. Write it end to end:
   which helper gets the id, which call gets the note, which helper writes the
   body? Every other handler is the same shape with different verbs."
4. Mapping stuck: "Make a two-column list: every error the service can hand you,
   and the status it deserves. Now write that list as one `switch` with
   `errors.As` for the typed ones and `errors.Is` for the sentinel." Only if
   they are still stuck, sketch the switch skeleton from LESSON.md — with the
   arms empty — and let them fill in each `status, payload` pair themselves.

## After passing

Preview: "Next comes gRPC — the same layering, a different transport. Your
domain package will not care, which is exactly the point."
