# API Performance

> `focus.web.performance` · ~4-5h · Stage: Focus: Web Services

## Objectives

By the end of this lesson you can:

- Implement response caching for an API endpoint with explicit invalidation and
  correct HTTP cache headers (`Cache-Control`, `ETag`).
- Detect an N+1 database access pattern in a handler and eliminate it with a
  batched or joined query.
- Implement cursor-based pagination and explain why it outperforms `OFFSET`
  pagination on large tables.
- Run a load test against an endpoint, read p50/p95/p99 latency and throughput,
  and identify the bottleneck.
- Choose which optimisation — caching, query shape, pagination — to apply first
  for a measured slow endpoint, and justify the choice with profiling or
  load-test data.

## Measure, or you are decorating

Everything in this lesson is a trade. Caching trades staleness for speed;
pagination trades random access for depth; a bigger connection pool trades
database contention for queueing in Go. None of those trades is worth making
against a slowness you have not located, and the industry's collective
experience is blunt about this: the part of the program you *feel* is slow is
usually not the part that is.

So the discipline comes first.

**Ask a question with a number in it.** "Is the feed fast?" has no answer.
"Can one replica serve 200 requests a second with p99 under 300 ms on a table
of two million articles?" has one, and it tells you when to stop.

**Read percentiles, never means.** The mean is the one statistic that hides the
users you are losing. A p50 of 20 ms with a p99 of 4 s is not "20 ms with a
rare hiccup" — it is one request in a hundred taking four seconds, and if a
page makes twenty API calls, roughly two out of three page loads contain a p95
request (1 − 0.95²⁰ ≈ 0.64). Percentiles compose against you.

| statistic | what it tells you |
|---|---|
| p50 | what the code does when nothing is in its way |
| p95 / p99 | contention, queueing, cold caches, the big accounts — the *tail* |
| max | one event; useful as a story, useless as a target |
| throughput | how much of it you can do at once |
| error rate | required — a fast p99 next to 30% 500s is not fast |

**Warm up, then measure.** Go has no JIT, but the first requests still pay for
an empty connection pool, an empty in-process cache and an unpopulated
filesystem cache. Discard the first stretch of every run.

**Fix the dataset.** A service over 100 rows and the same service over ten
million are different programs; only one of them is the one you deploy.

**Beware the closed loop.** A load tool that waits for each response before
sending the next request throttles itself exactly when your service is
struggling, so the queue you are trying to measure never forms. Drive at a
*fixed rate* (`vegeta attack -rate`, `k6` arrival-rate executors, `oha -q`) and
watch what happens to latency when the rate passes what the service can do.

**Change one thing.** Two changes and one number is not an experiment. Keep the
baseline; `go test -bench` with `-count=10` piped through `benchstat` exists
because a single benchmark run is noise.

You already have the tools: benchmarks and `pprof` from S5 tell you where the
CPU and the allocations went inside the process; a load test tells you what
the process looks like from outside. This lesson is mostly about what those two
views show once you know how to read them.

## The N+1 problem, one layer down

You met N+1 in the GraphQL lesson: a resolver for `Post.author` that runs once
per post, turning one client query into five hundred and one database queries.
The same bug lives in plain REST handlers, and it hides better, because
nothing about the code looks wrong:

```go
for _, a := range articles {
	author, err := s.Store.AuthorByID(ctx, a.AuthorID)   // one round trip. each.
	...
}
```

Two cures, and the choice between them is not automatic:

- **Join.** `SELECT … FROM articles JOIN authors ON …`. One statement, one
  trip. Best when the relationship is one-to-one and the joined columns are
  small.
- **Batch.** Fetch the page, collect the ids, and fetch those rows with one
  `IN (…)`. Two statements. Best when the child rows are shared (twenty
  articles by five authors join to twenty copies of an author row, but batch
  to five), when the child is a *list* — joining a one-to-many multiplies your
  result set and you de-duplicate in Go — or when the two things live in
  different stores.

The exercise batches, and gives the batch function the same name the GraphQL
dataloader had: `AuthorsByIDs`. That is the point. A dataloader is not a
GraphQL feature; it is this shape, with a queue in front of it.

What makes N+1 tractable is that it has a number. The exercise's `Store`
counts every round trip, and the test asserts *two* queries for a page of 5,
20 or 50. "Feels slow" is an argument; "51 round trips to serve 50 rows" is a
bug report. Put the same instrument in production as a queries-per-request
histogram beside your S5 request metrics, and you will find these before your
users do.

## Pagination that stays fast at depth

`LIMIT 20 OFFSET 0` is fine. `LIMIT 20 OFFSET 200000` is a different query
wearing the same clothes: the database has no way to jump to the 200,000th
row, so it produces two hundred thousand rows in order and throws them away
before giving you twenty. Cost grows with depth, and the deepest pages are the
ones your crawlers and your "export everything" scripts ask for.

It is also *wrong*, which people notice later than the slowness. Between page
one and page two, three new articles arrive. `OFFSET 3` now points three rows
further back than it did, so page two repeats rows page one already showed —
and if rows are deleted instead, page two skips rows nobody ever sees. The
exercise pins this with a test that asserts the duplication happens.

**Keyset (cursor) pagination** asks the other question. Instead of "skip 200,000
rows", it says "give me the rows after *this* one":

```sql
SELECT … FROM articles
 WHERE (created_at, id) < (?, ?)      -- everything strictly after the cursor
 ORDER BY created_at DESC, id DESC
 LIMIT 20;
```

The cursor is the sort key of the last row you handed out, so it is an anchor
in the data rather than a count of rows. Inserts and deletes elsewhere cannot
move it. And with an index on `(created_at DESC, id DESC)`, the database seeks
straight to it and reads twenty rows: the cost is the same on page 1 and page
10,000.

Three details decide whether it works:

1. **The sort key must be unique.** `created_at` alone is not: two rows in the
   same millisecond straddling a page boundary get repeated or skipped. Append
   the primary key and compare the pair.
2. **The index must match the ordering.** Change the `ORDER BY` and the index
   stops helping. An index is a promise about one access pattern, not a speed
   setting.
3. **Ask the planner, do not assume.** These two `WHERE` clauses are logically
   identical:

   ```sql
   WHERE (created_at, id) < (?, ?)
   WHERE created_at < ? OR (created_at = ? AND id < ?)
   ```

   `EXPLAIN QUERY PLAN` on the exercise's schema says otherwise:

   ```
   row values:  SEARCH articles USING INDEX articles_feed (created_at<?)
   OR form:     SCAN articles USING INDEX articles_feed
   OFFSET:      SCAN articles USING INDEX articles_feed
   ```

   A planner facing `OR` usually gives up on the index, so the OR form walks
   the whole index exactly like `OFFSET` does — correct pagination, no speed.
   The benchmark agrees: at depth 5,000 the seek is roughly five times faster
   here, and the gap widens with the table. Postgres spells the tool `EXPLAIN
   (ANALYZE, BUFFERS)`, MySQL `EXPLAIN ANALYZE`; row values work in both, but
   not everywhere, so **check the plan** rather than trusting the syntax.

What keyset costs you is honest: no "jump to page 57", no total count without a
separate query, and re-sorting means re-designing the cursor. Offer page
numbers where a human clicks them and the table is small; use cursors for
feeds, exports, infinite scroll and every API a machine consumes.

Make the cursor **opaque** — base64 of `<timestamp>:<id>` in the exercise — so
clients cannot parse and depend on its shape. Opaque is not secret: anyone can
decode and edit one, so validate it on the way in (a bad cursor is a 400) and
keep passing it as a query parameter, never as string-concatenated SQL. If a
cursor must not be forgeable — one that encodes a filter a user should not
change — sign it with the same HMAC habit from the authentication lesson.

## Conditional requests: ETag and Cache-Control

The cheapest response is one with no body.

An **ETag** is a short identifier for a specific version of a representation.
Send it with the response; a client that comes back sends it in
`If-None-Match`, and if it still matches you answer `304 Not Modified` with no
body at all.

```
GET /articles                 →  200, ETag: "9f2c…", Cache-Control: public, max-age=30
GET /articles                    If-None-Match: "9f2c…"
                              →  304, ETag: "9f2c…"   (no body)
```

Deriving the tag by hashing the bytes you are about to send is the version that
cannot be wrong. Cheaper tags — a row version, a `updated_at` stamp — avoid the
hash but must account for *every* input that changes the output, including the
one somebody adds next quarter. Tags come in two strengths: `"abc"` is strong
("byte-for-byte this"), `W/"abc"` is weak ("semantically this"). Conditional
GETs compare weakly, so `W/"abc"` and `"abc"` match; strong comparison is for
range requests.

`Cache-Control` is the other half, and it is a different statement: the ETag
says *what* this is, `Cache-Control` says *who may keep it and for how long*.

| directive | meaning |
|---|---|
| `max-age=30` | fresh for 30 s; reuse without asking |
| `public` | any cache may store it — CDNs, proxies |
| `private` | only the end client. Anything behind a session cookie |
| `no-cache` | store it, but revalidate before reuse (this is not "do not cache") |
| `no-store` | never write it down. Tokens, one-time codes |
| `stale-while-revalidate=60` | serve the stale copy while refreshing in the background |

Get `public` wrong on an authenticated response and a shared cache will hand
one user's data to the next — a real, repeatedly-shipped vulnerability, and the
reason the authorization lesson's rule (decide per request, in one place)
extends to your cache headers. When a response varies by a request header, say
so with `Vary`.

Be clear-eyed about what a 304 buys: it saves **bandwidth**, not work, unless
you also avoid recomputing the body. Hashing a response you had to build
anyway is a smaller win than not building it — which is what the next section
is for.

## In-process caching, honestly

A `map` in front of your store is the fastest cache you can have and the
easiest to get wrong. Two bounds are mandatory:

- a **TTL**, which bounds staleness — how wrong an answer is allowed to be;
- a **size limit**, which bounds memory. A cache keyed by anything a client
  controls (here: limit and cursor) and bounded only by time is a way for a
  stranger to fill your heap one distinct key at a time. Evict least-recently-
  used when the bound is reached.

Then invalidate. In the exercise, a `POST /articles` purges the cache, and that
line is not an optimisation — without it a client does not see its own write
for thirty seconds and files a bug about your API losing data. Purging
everything is blunt but never wrong; narrowing it (with keyset pagination, an
insert at the head only changes pages that contain the head) is a measurement's
job, not a reflex.

Three things the code will not tell you:

**Per-instance caches lie at scale.** Run three replicas and each sees a third
of the traffic, so each hit rate falls; you hold three copies of the same
bytes; and a purge on the replica that handled the write leaves the other two
serving stale pages until their own TTL expires. That is often fine — say so
out loud, with a number ("at most 30 s stale") — but it is a design decision,
not an accident. The alternative is a cache outside the process, which is
another moving part to run, monitor and invalidate.

**A hot key expiring is a stampede.** Every request that misses at that instant
recomputes the same page. Cures: single-flight (one goroutine computes, the
rest wait on it), jittered TTLs so keys do not expire in lockstep — the same
reasoning as the jitter in the background-jobs lesson — and serving stale
content while one worker refreshes.

**A cache with no hit-rate instrument is a guess with a mutex.** Two counters
are enough to tell you whether the staleness is buying anything. A hit rate
that collapses after a deploy usually means a key grew a new dimension.

## Compression: sometimes a cost

`gzip` on a 40 KB JSON feed typically cuts it by 80% and is an easy win on a
mobile network. On a 200-byte response it can produce *more* bytes than it
saves and costs CPU on both ends. Rules that survive contact with reality:

- Set a floor (≈1 KB) and a content-type allowlist; JSON, HTML, CSS and JS
  compress well, JPEG/PNG/MP4/zip do not compress at all.
- Always send `Vary: Accept-Encoding`, or a shared cache will serve gzip to a
  client that asked for none.
- Compression is CPU. On a service that is already CPU-bound it can *lower*
  throughput; on a service waiting on a database it is nearly free. Measure
  which one you are.
- Do not compress secrets on a channel where an attacker can influence part of
  the body — that is the BREACH class of attack, and it is why sensitive
  responses are often left alone.
- The standard library has `compress/gzip` but no middleware; you write about
  thirty lines or take a dependency. Often the right answer is neither: let
  your reverse proxy or CDN do it.

## Connection pools are queues, not dials

`db.SetMaxOpenConns(n)` looks like a speed setting and is a concurrency limit.
The database can only do so much at once; past that point extra connections
buy contention, context switches and memory (in Postgres, a process each), and
throughput goes *down* while latency goes up.

Think in round numbers. If a query takes 5 ms and you allow 10 connections,
the pool can retire about 10/0.005 = 2,000 queries a second. Asking for more
concurrency than that does not create capacity; it moves the wait from the
database into your process, where `db.Stats().WaitCount` and `WaitDuration`
will show it. That is a feature: a bounded pool is a bulkhead protecting the
database from your service, and a queue you can see beats a database you
cannot.

Two companions. `SetMaxIdleConns` should usually equal `MaxOpenConns` — a lower
idle count means connections are opened and closed under steady load, and a TLS
handshake per query is a strange way to save memory. `SetConnMaxLifetime`
(a few minutes) lets connections rotate so a failover or a load balancer's idle
timeout does not hand you a dead connection.

## Do less per request

Once the plan and the round trips are right, what is left is work you chose to
do:

- **Select what you return.** The exercise's feed omits article bodies. Columns
  you do not render cost bytes on the wire, memory in the driver and parse time
  in the client.
- **Precompute at write time.** A counter maintained on insert beats
  `COUNT(*)` on read, and reads usually outnumber writes by an order of
  magnitude.
- **Do startup work at startup.** Compile regexps and templates once, not per
  request.
- **Reuse buffers** for hot paths (`sync.Pool`, a reused `bytes.Buffer`) —
  after `pprof -alloc_objects` says allocation is your problem, not before.
- **Stream instead of buffering** when responses are large: `json.NewEncoder(w)`
  writes as it goes, where `json.Marshal` builds the whole body in memory
  first. The honest tension: you cannot hash a body you did not buffer, so
  ETags and streaming pull in opposite directions. Buffer small responses and
  tag them; stream large ones and skip the tag.
- **Cache the rendered bytes, not the struct.** A hit that still marshals and
  hashes has given back most of what it saved.

## Which one first

Work in this order, and let a measurement move you down it:

1. **Stop doing unnecessary work.** N+1 queries, a missing index, selecting
   columns you drop, `COUNT(*)` on every page. This is where the order-of-
   magnitude wins live and none of it costs correctness.
2. **Fix the query shape.** Read the plan. `SCAN` where you expected `SEARCH`,
   `OFFSET` at depth, a sort the index could have provided.
3. **Then cache.** Caching earlier hides the two problems above until the cache
   misses — and it misses precisely when you are busiest: a deploy, a purge, a
   stampede. A cache in front of a bad query is a slow query with worse
   failure modes.
4. **Then the payload.** Compression, streaming, buffer reuse, pool sizing.
   These move single-digit percentages once the big ones are gone, and they are
   worth having then.

At every step: what did the load test say, at what rate, at which percentile?

## Exercise

Open [`exercise/`](exercise/) — one `apiperf` package over SQLite serving a
feed: `GET /articles?limit=&cursor=` and `POST /articles`. `db.go`, `clock.go`,
`model.go`, `respond.go` and `store.go` are provided; read `store.go` first,
because its round-trip counter and `ExplainLast` are the instruments the whole
suite is built on. You complete six files:

```
keyset.go   EncodeCursor, DecodeCursor, Store.ListArticles
batch.go    Store.AuthorsByIDs
cache.go    Cache — bounded, TTL'd, LRU, injected clock
etag.go     ETag, MatchETag
feed.go     Service.Feed, Service.FeedJSON
server.go   handleListArticles, handleCreateArticle
```

`Service.Feed` starts out written the way it usually is first: correct, paged
with `OFFSET`, one author query per article. Making it fast without breaking it
is the exercise.

Acceptance criteria:

1. `AuthorsByIDs` fetches many authors in **one** round trip, collapses
   duplicate ids, omits ids with no row, and makes no query at all for an empty
   list.
2. `EncodeCursor`/`DecodeCursor` round-trip a `Cursor`, produce URL-safe base64
   over `<unixnano>:<id>`, and turn every malformed input — empty, non-base64,
   wrong field count, non-integer field — into an error matching
   `ErrBadCursor`.
3. `ListArticles` is one query, no `OFFSET`, ordered newest first, starting
   *strictly* after the cursor, with ties on `created_at` broken by `id`.
4. That cursor query plans as a `SEARCH` into `articles_feed`, not a `SCAN` —
   the test asks the planner through `ExplainLast`.
5. Pages stay stable when rows are inserted at the head between requests: no
   row appears twice, none is skipped.
6. `Cache` is bounded (LRU eviction at the limit), expires entries at exactly
   `stored + ttl` against the injected clock, drops an expired entry when it is
   read, refreshes expiry on re-`Set` without growing, counts hits and misses,
   survives `-race` from many goroutines, and `Purge` empties it while keeping
   the counters.
7. `ETag` is the quoted hex SHA-256 of the body; `MatchETag` implements
   `If-None-Match` with `*`, comma-separated lists and weak comparison.
8. `Feed` makes exactly **two** round trips for a non-empty page at any limit,
   one for an empty feed, never returns a nil `Items`, and sets `NextCursor`
   only when a further page really exists.
9. `FeedJSON` serves a cache hit with **zero** database round trips, returning
   the same bytes and tag as the miss that filled it, and keys pages by both
   limit and cursor.
10. `GET /articles` answers 200 with `ETag` and `Cache-Control: public,
    max-age=30`, answers 304 with no body but the same headers when
    `If-None-Match` matches, clamps `?limit=` to `MaxLimit`, and 400s a bad
    limit or an unreadable cursor.
11. `POST /articles` validates (blank title, unknown author, unknown field,
    oversized body), creates the article, **invalidates the cache**, and
    answers 201 — a client holding the previous ETag must get 200 with the new
    article, not a 304.
12. `go test -race ./...` is green and the code is `gofmt`-formatted.

```sh
cd exercise
go test -race ./...
```

Nothing in the suite sleeps and nothing asserts a duration: every claim about
speed is a query count, a cache bound or a query plan. The benchmarks are
separate, committed, and yours to run and explain:

```sh
go test -run '^$' -bench . -benchmem -count=10 ./... | tee new.txt
```

Have an opinion *before* you look at `ListArticlesOffsetDeep` vs
`ListArticlesKeysetDeep`, `AuthorsOneByOne` vs `AuthorsBatched`, and
`FeedUncached` vs `FeedCached`. Then explain the numbers you actually got —
including any that surprised you.

Suggested order: `keyset.go`, `batch.go`, then `feed.go` (most of the suite
depends on those three), then `cache.go`, `etag.go` and `server.go`.

## Further reading

- [RFC 9110 §8.8, §13 — Conditional requests and validators](https://www.rfc-editor.org/rfc/rfc9110#name-conditional-requests)
- [MDN — Cache-Control](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control)
- [Use the Index, Luke — Paging through results](https://use-the-index-luke.com/sql/partial-results/fetch-next-page)
- [SQLite — EXPLAIN QUERY PLAN](https://www.sqlite.org/eqp.html)
- [pkg.go.dev — database/sql DB.SetMaxOpenConns](https://pkg.go.dev/database/sql#DB.SetMaxOpenConns)
- [go.dev blog — Profiling Go programs](https://go.dev/blog/pprof)
