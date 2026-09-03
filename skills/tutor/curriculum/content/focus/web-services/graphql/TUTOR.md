# Tutor notes — GraphQL in Go

## Where the learner is

Fifth lesson of the web-services focus pack. They have authentication,
authorization with a single deny-by-default enforcement point, realtime
transports, and edge hardening (body limits, strict decoding, token buckets on
an injected clock, CORS, timeouts) behind them, on top of S5's production
`net/http` server and S4's SQL. They have **not** done S6 systems design — no
CAP, no consistent hashing, no SLO vocabulary; if a distributed idea is needed,
say it in plain terms.

The intellectual move here is a reversal of the one they just made. Hardening
said: *you* decide what an endpoint costs, so bound it per endpoint. GraphQL
says: the client decides the shape of the work, over one URL, in a 200-byte
body. Every consequence in this lesson — resolver-per-field fetching, loaders,
depth and complexity limits, the loss of HTTP caching — falls out of that one
sentence. If they leave able to say "GraphQL does not remove work, it moves who
decides how much of it happens", the lesson landed.

Two failure shapes to watch. The first is the convert: they have seen the demo,
the query is elegant, and they now want GraphQL everywhere. Push them onto the
REST column of the comparison table until they can name a case where GraphQL is
strictly worse (public read-heavy API behind a CDN — GraphQL's worst case, and
it is a very common shape). The second is the dismisser: "it's just POST with
extra steps". Make them price the six-round-trip mobile screen, then ask what
they would build instead — usually they reinvent sparse fieldsets and `include`
parameters, badly.

A note on the exercise's scope. The module is a hand-written ~200-line executor
because gqlgen's generated code hides exactly the three things the lesson is
about: where resolvers are called, where loaders dispatch, and where complexity
is counted. If the learner asks "why aren't we using gqlgen" — good question,
and the LESSON answers it. Be clear that gqlgen is the right choice at work and
that everything here maps onto it one-to-one.

## Common misconceptions

- **"GraphQL is faster than REST."** It is fewer round trips for a client that
  wanted a graph. On the server it is *more* work per request unless you batch,
  and it throws away HTTP caching, which for a read-heavy public API is the
  single biggest performance lever there is. Fewer requests is not less work.
- **"One endpoint means one authorization decision."** The identity middleware
  still runs at the edge, but the policy check now lives inside resolvers, per
  field, and is easy to forget on the sixth nested type. `Post.author` returning
  an author the caller may not see is an authorization bug that no edge
  middleware can catch.
- **"N+1 is a modelling mistake."** The exercise's whole design refutes this:
  the naive and batched schemas are *identical* except for four resolvers. The
  schema is fine; the fetching is not.
- **"Just use a join."** Ask them where they would put it. `Post.author` does
  not know 499 siblings are about to ask the same question — there is no place
  in the execution model that sees the whole list. That is what a loader is
  for. (Contrast S4's SQL, where you controlled the whole query.)
- **"The dataloader is a cache, so I'll share it between requests."** It is a
  per-request identity map with no invalidation. Shared, it hands one user
  another user's row — go straight back to the authorization lesson.
- **"Errors mean a non-200."** A GraphQL response that executed is 200 with
  `data` *and* `errors`. A client that only checks `resp.StatusCode` renders a
  half-empty page and reports success. The distinction the exercise draws:
  never-executed (parse, validation, over the limit) → 400, no `data` key;
  executed → 200 with whatever resolved.
- **"Depth limiting is enough."** `{ posts(first: 5000) { title } }` has depth
  2. **"Complexity limiting is enough."** The six-level cyclic query with
  `first: 1` everywhere is complexity-cheap. Neither subsumes the other, which
  is why the exercise has two rejection tests.
- **"Disabling introspection secures the endpoint."** It hides the schema from
  a browser tab and from nobody serious — field names are guessable and clients
  ship queries. It is hygiene, not a control.
- **"`!` is just documentation."** A failing non-null field propagates its null
  up to the nearest nullable parent, and with none it nulls the entire
  response. One flaky field marked `!` can blank a page.
- **"The complexity number is the real cost."** It is an estimate, and it is
  only honest if resolvers actually honour the argument it charged for. A
  resolver that ignores `first` turns the analyser into decoration — which is
  why `applyFirst` is in the acceptance criteria.

## Grilling points

- "Draw the resolver calls for `{ posts(first: 5) { title author { name } } }`
  on the naive schema. How many functions ran, how many store calls, and which
  resolver could have known it was about to be called five times?" (None — that
  is the point.)
- "Your `Load` returns a `*Result` that isn't filled in yet. What stops a
  resolver from reading it too early, and what would happen if the executor
  dispatched *after* unpacking the level instead of before?"
- "Two posts share an author, and one of that author's comments is in the same
  query. How many rows does the store fetch for that author, and what in your
  code makes that true?" (One — the `results` map is dedupe and cache at once.)
- "The batch function returns 4 rows for 5 keys. Where does the fifth key's
  error come from, and why must it not fail the other four?"
- "Why does the store's batch method return a `map[string]Author` and not a
  `[]Author`? What bug does that design remove?" (Ordering — the classic
  dataloader off-by-one that gives every user the wrong name.)
- "Give me the query that gets past your depth limit but should not, and the
  one that gets past your complexity limit but should not."
- "A client sends `{ posts { title } }` with no `first`. What does your
  analyser charge, and what would happen at runtime if `PageSize` were 0?"
- "Your service is a public, read-heavy product catalogue behind a CDN. Sell me
  GraphQL." (They should refuse, and name the cache as the reason.)
- "Where does your rate limiter from last lesson still help here, and where has
  it stopped meaning anything?" (Still bounds requests per second and body
  bytes; no longer bounds work per request.)
- "You need to remove a field from the schema. How do you know it is safe?"
  (`@deprecated` plus per-field usage tracking — versioning in GraphQL is an
  observability problem, not a URL problem.)
- "Where would you put an authorization check for `Post.author`, and what
  happens if `Author.posts` forgets one?"

## Grading rubric

- **A** — All tests pass under `-race`. `Load` never fetches and returns the
  same `*Result` for a repeated key; `Dispatch` clears the queue *before*
  calling the batch function, calls it once, and handles the three cases (empty
  queue, batch error to every key, per-key `ErrNotFound`) distinctly. The three
  resolvers follow `batchedAuthorPosts` without duplicating loader wiring, and
  `comments` honours `first`. `Complexity` multiplies parents through list
  fields and defaults weight and page size sensibly. The learner explains N+1
  in terms of the execution model rather than "GraphQL is slow", and can name a
  scenario where they would choose REST.
- **B** — Tests pass; the design is sound but the seams are rough: `Dispatch`
  leaves the queue populated on the error path, per-key errors are built
  ad hoc instead of through `missingKeyError`, complexity is computed with the
  list multiplier applied to the list field itself rather than to its children,
  or a resolver drops `first`. Explanations mostly right with one misconception
  above still live.
- **C** — Tests pass only after substantial hinting, or the loader is treated
  as a magic object they cannot draw. Pass only if a time-boxed remediation
  lands; otherwise iterate.
- **Fail** — Tests failing; or the learner cannot say why `Post.author` cannot
  be fixed with a join; or they would ship a GraphQL endpoint with no depth or
  complexity limit because "we have a rate limiter". Remediate — the second one
  is a security position, not a style preference.

## Remediation ladder

1. "Run one failing test with `-run` and read the failure aloud. What did it
   expect, what did it get?" (The loader tests are written to be read in order;
   start at `TestLoaderBatchesQueuedKeysIntoOneCall`.)
2. Stay on the concept, not the code: "Draw `Load("a")`, `Load("b")`,
   `Load("a")`, then `Dispatch`. What must exist after the three Loads, and how
   many things go to the database?" For complexity: "`{ posts(first: 2) {
   comments(first: 3) { body } } }` — how many times does `body` run? Now write
   the arithmetic that produces that number without running anything."
3. Name the tool without the shape: "the loader needs a map from key to result
   *and* a slice of keys not yet fetched — what is each one for?"; "a
   `Deferred` is a function the executor calls after `Dispatch`, so a resolver
   returns one instead of a value"; "the multiplier applies to a list field's
   *children*, not to the list field itself."
4. Walk the algorithm verbally — for `Dispatch`: bail on empty, take the keys,
   clear the queue, one call, then loop the keys setting batch error / missing
   error / value — and let them type it. Show `batchedAuthorPosts` side by side
   with their attempt before writing anything yourself.

## After passing

Preview: "Next: background jobs. GraphQL let the client decide how much work a
request does; the next question is what to do with work that should not happen
inside a request at all."
