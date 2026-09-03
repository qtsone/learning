# Tutor notes — API Performance

## Where the learner is

Seventh lesson of the web-services focus pack, after authentication,
authorization, realtime, hardening, GraphQL and background jobs. They bring
S5's `database/sql` over `modernc.org/sqlite`, S5's benchmarks and `pprof`,
S5's fake clocks and 1.22 mux, S4's SQL and TDD, and — one lesson back — the
GraphQL resolver N+1 and its dataloader. They have **not** done S6 systems
design: no CAP, no consistent hashing, no SLO vocabulary. "Three replicas" and
"a shared cache in front of the service" appear here and are explained in
place; keep them concrete.

The intellectual move is from *make it correct* to *make it measured*. Every
earlier lesson had a right answer the tests could name. This one has trades,
and the skill being graded is whether the learner can say what a change costs
and what number told them to make it. If they leave able to say "I do not
optimise what I have not measured, and here is the instrument I would put in
production", the lesson landed.

Two shapes to watch. The first is the learner who optimises by instinct — they
will reach for the cache first because caching is the word they know, and never
find the 51 round trips underneath it. The second is the learner who takes the
exercise's query counters as a testing trick rather than the point; ask them
what the production version of that counter is (a queries-per-request histogram
beside their S5 request metrics) until they say it themselves.

## Common misconceptions

- **"Cache first, it is the biggest win."** A cache in front of a bad query is
  a slow query with worse failure modes: it misses exactly when you are busiest
  — a deploy, a purge, a stampede. Walk them down the lesson's order: stop
  doing unnecessary work, fix the query shape, *then* cache, then the payload.
- **"OFFSET is just slower."** It is also *wrong*. `TestOffsetPagesRepeatRowsUnderInserts`
  asserts the broken behaviour on purpose: three inserts at the head and page 2
  repeats page 1. Deletes make it skip rows instead, which nobody ever notices.
- **"Keyset means `WHERE created_at < ?`."** Only if `created_at` is unique. It
  is not — `TestListArticlesBreaksTiesOnID` seeds five rows sharing one
  timestamp, and a single-column cursor either repeats or drops rows at the page
  boundary. The sort key must be unique, so it must include the primary key.
- **"`(a, b) < (?, ?)` and `a < ? OR (a = ? AND b < ?)` are the same query."**
  Logically yes; to the planner, no. `TestListArticlesSeeksInsteadOfScanning`
  asks SQLite, and the OR form comes back a SCAN — correct pagination, no speed.
  The transferable habit is *asking the planner*, not memorising the answer.
- **"Base64 makes the cursor safe."** Opaque is a contract, not secrecy. Anyone
  can decode and edit one, which is why `DecodeCursor` validates, why every
  failure is `ErrBadCursor` → 400, and why the value is still bound as a
  parameter rather than concatenated into SQL.
- **"NextCursor is the last row of the page."** Then the client always makes one
  extra empty request to discover the end — on a polled feed that is real
  traffic. Read `limit+1`, drop the extra, and a cursor means "there is
  definitely more" (`TestFeedLastFullPageHasNoCursor`).
- **"One extra author query per article is nothing — it is a local database."**
  It is nothing per row and everything per page: 51 round trips to serve 50
  rows. `TestFeedRoundTripsDoNotGrowWithThePage` runs at limits 5, 20 and 50
  and demands the same number each time. The point is that N+1 has a *number*.
- **"Batching is a GraphQL thing."** Same shape, one layer down, deliberately
  the same name (`AuthorsByIDs`). A dataloader is this function with a queue in
  front of it. If they answer "just join it", ask when a join loses: shared
  child rows, one-to-many that multiplies the result set, two stores.
- **"A 304 saves the server work."** It saves *bandwidth*. It saves work only
  because the cache hit above it meant the body was never rebuilt — which is
  why `TestRepeatedRequestIsServedFromTheCache` asserts zero round trips and
  the 304 test asserts an empty body.
- **"If-None-Match is string equality."** `*`, comma-separated lists, and weak
  comparison (`W/"abc"` matches `"abc"`). Too strict re-sends every body; too
  loose hands a client somebody else's page forever.
- **"A TTL bounds the cache."** It bounds *staleness*. Memory needs the entry
  count, because the key here is built from client-controlled `limit` and
  `cursor` — a stranger can fill the heap one distinct key at a time.
- **"An expired entry can wait for eviction pressure."** Then a cold key holds
  its bytes for as long as there is room. `Get` on an expired entry is a miss
  *and* a removal.
- **"Invalidation is a performance detail."** It is correctness: without
  `Purge`, a client does not see its own write for 30 seconds and files a bug
  saying your API loses data. `TestCreateArticleInvalidatesTheCache` is that
  bug report.
- **"`public` is the good one."** `public` on anything behind a session cookie
  lets a shared cache serve one user's data to the next. Connect it to the
  authorization lesson's rule: decide per request, in one place — including in
  your headers.
- **"Bigger pool, faster service."** `MaxOpenConns` is a concurrency limit, not
  a dial. Past what the database can do at once, throughput falls and latency
  rises; the bounded pool is a bulkhead, and `db.Stats().WaitCount` is the
  queue you can see. Same instinct behind "gzip everything", which below ~1 KB
  adds bytes and always costs CPU on both ends.
- **"The mean latency is fine."** The mean is the statistic that hides the users
  you are losing. Make them do the composition arithmetic: twenty calls per page
  and a p95 means roughly two page loads in three contain a p95 request.
- **"The benchmark said so."** One run is noise. `-count=10` and `benchstat`,
  same machine, same dataset, one change at a time. And nothing here sleeps:
  the clock is injected, so a test moves time instead of waiting for it.

## Grilling points

- "Your feed endpoint is slow. What is the first command you run, and what
  question are you asking with it?" (Refuse "add a cache" until a number
  appears.)
- "Give me the p50, p95 and p99 story for a service whose mean is 40 ms. Which
  one do you put on the dashboard, and which one wakes you up?"
- "Your load generator waits for each response before sending the next. What
  does it fail to show you, and when does that matter most?" (Closed loop
  throttles itself precisely when the service is queueing.)
- "Page 10,000 of `OFFSET`: what is the database physically doing, and how much
  of that work reaches the client? Now tell me the *correctness* bug."
- "Two rows share a `created_at` and land on a page boundary. Walk me through
  both failure modes. Then: you changed `ORDER BY` to `title` — what happens to
  `articles_feed` and to the cursor?"
- "Show me the plan for your cursor query, then rewrite the `WHERE` as an `OR`
  and show me that plan. Explain the difference to me as if I owned the
  database."
- "Your `Feed` makes two round trips. Argue for a single `JOIN` instead — then
  argue against it."
- "The cache TTL is 30 s and you run three replicas. Somebody POSTs an article.
  Describe, second by second, what the next three readers see." (Purge is local;
  the other two serve stale until their own TTL expires. The right answer names
  a number out loud.)
- "A hot key expires at the same instant on every replica. What happens, and
  name two cures." (Stampede; single-flight, jittered TTL, serve-stale — the
  jitter is the same reasoning as background-jobs' retry jitter.)
- "Your ETag is the SHA-256 of the body, so you must buffer the body. When is
  that the wrong trade?" (Large responses: stream and skip the tag.)
- "`MaxOpenConns` is 10 and a query takes 5 ms. What throughput does that
  allow, and what happens at 3,000 requests a second?"
- "Caching, fixing the N+1, and switching to keyset. You may do one this week.
  Which, and what data decided it?"

## Grading rubric

- **A** — All tests pass under `-race`. `ListArticles` is one parameterised
  statement per branch with a row-value comparison that plans as a SEARCH;
  `AuthorsByIDs` dedupes and short-circuits the empty batch; `Feed` reads
  `limit+1` and makes exactly two round trips at any limit; the cache is
  LRU-bounded, drops expired entries on read and keeps counters across `Purge`;
  `FeedJSON` caches the rendered bytes with their tag; the handler sets
  validators before the 304 branch and `POST` purges. The learner has run the
  benchmarks, states an expectation *and* what surprised them, and can order the
  four optimisations from the lesson with a reason for each.
- **B** — Tests pass; design sound but seams rough: cursor built without the id
  tiebreaker anywhere outside `ListArticles`, `Page` cached instead of the
  rendered bytes, `Cache-Control` set only on the 200, `Purge` present but
  described as an optimisation, or benchmarks run without an opinion formed
  first. Explanations mostly right with one misconception still live.
- **C** — Tests pass only after substantial hinting, or the learner treats the
  query counter and the plan assertion as test scaffolding rather than the two
  instruments the lesson is about. Pass only if a time-boxed remediation lands;
  otherwise iterate.
- **Fail** — Tests failing, or the learner would still reach for a cache before
  measuring, or cannot say why `OFFSET` pagination is a correctness bug as well
  as a speed one. Both are load-bearing for the capstone; remediate rather than
  advance.

## Remediation ladder

1. "Run the one failing test with `-run` and read its message aloud — what
   number did it want, and what number did it get?" Every failure here names a
   count, a bound or a plan.
2. Move to the data, not the code: "Sketch the six rows, then the page-1 query
   and the page-2 query. Which row is the anchor, and what does an insert at
   the head do to each version?" For `Feed`: "Write down every round trip a
   page of 50 makes today. How many are the same question with a different
   parameter?"
3. Name the tool without the shape: "`base64.RawURLEncoding` over
   `<unixnano>:<id>`"; "one `WHERE (created_at, id) < (?, ?)`, and ask
   `ExplainLast` what it thinks"; "`strings.Repeat(\", ?\", n-1)` builds the
   `IN` list, and the ids go in as args"; "ask the store for `limit+1`";
   "`container/list` plus a map is the standard LRU, and eviction takes the
   back."
4. Walk one path verbally end to end — request in, limit parsed, cache probed,
   two queries, marshal, tag, validators, 304-or-body — and let them type it.
   Only write code beside them if step 3 stalls twice.

If they are stuck everywhere at once, order matters: `keyset.go` and `batch.go`
first, then `feed.go`. Most of the suite depends on those three, and `cache.go`,
`etag.go` and `server.go` are each self-contained once `Feed` is honest.

For a learner who is green but shallow, skip the code entirely and run
`go test -run '^$' -bench . -benchmem -count=10 ./...` together. Make them
predict each pair before reading it, then explain the gap they got wrong.

## After passing

Preview: "Last one is the capstone — you extend your service with the pieces
from this pack, and every claim you make about it has to be one you can
demonstrate."
