# GraphQL in Go

> `focus.web.graphql` · ~4-5h · Stage: Focus: Web Services

## Objectives

By the end of this lesson you can:

- Explain the schema-first workflow and define a GraphQL schema with object
  types, queries, and mutations.
- Implement resolvers in Go that map schema fields to data-fetching code.
- Explain how nested resolvers cause the N+1 query problem, and implement a
  dataloader-style batching fix.
- Choose between GraphQL and REST for a given API scenario and justify the
  decision.
- Explain why GraphQL needs its own hardening — depth and complexity limits —
  beyond the HTTP protections you built last lesson.

## The trade you are making

Every REST endpoint you have written answers a question you chose in advance.
`GET /posts` returns the fields you decided a post has; a client that wants the
author's name too makes a second request, and a client that only wanted titles
downloads everything anyway. Two failure modes with names: **over-fetching**
(the response carries fields nobody used) and **under-fetching**, whose usual
symptom is a mobile screen that costs six round trips on a bad connection.

GraphQL moves that decision to the client. The server publishes a **schema** —
a typed graph of what exists — and the client sends a query describing exactly
the shape it wants back:

```graphql
{ posts(first: 10) { title author { name } } }
```

One request, one response shaped like the query, nothing extra. That is the
entire pitch, and it is a real one. Everything else in this lesson is the bill:
the client now controls how much work your server does, which changes how you
fetch data, how you fail, how you cache, and what an attacker can ask for.

Hold on to one sentence: **GraphQL does not remove work, it moves who decides
how much of it happens.**

## Schema first

The schema is the contract, written in the **Schema Definition Language**, and
it is the artifact both sides agree on before anyone writes code:

```graphql
type Post {
  id: ID!
  title: String!
  author: Author!
  comments(first: Int): [Comment!]!
}
```

Reading the punctuation, because it is load-bearing:

- `String`, `Int`, `Boolean`, `Float`, `ID` are the built-in scalars. `ID` is a
  string with intent: an opaque identifier, not something to do arithmetic on.
- `!` means non-null. `String` may be null, `String!` may not.
- `[Comment!]!` is a non-null list of non-null comments: the list is always
  there, and it never contains a null — though it may be empty.

Those `!`s are an operational decision, not decoration. If a non-null field's
resolver fails, the error cannot be represented there, so it propagates up to
the nearest nullable parent — and if there is none, the *whole response* becomes
`"data": null`. One flaky field marked `!` can null out a page. The rule of
thumb: mark a field non-null when its absence really is a bug, and leave it
nullable when it depends on something that can fail on its own.

Two type definitions are special. `Query` is the root of every read, `Mutation`
the root of every write:

```graphql
type Query {
  posts(first: Int): [Post!]!
  post(id: ID!): Post
}

type Mutation {
  createPost(authorId: ID!, title: String!): Post!
}
```

`Query.post` returns a nullable `Post`: "no such id" is an answer, not a
failure — which is how GraphQL says 404 without a status code.

**Schema-first** is a workflow, not just a file format. You write the SDL,
generate the Go types and interfaces from it, and implement what the generator
left as holes. The compiler then enforces the contract: add a field to the SDL
and your build breaks until a resolver exists. The alternative,
*code-first*, derives the schema from Go structs and tags — less ceremony, but
the schema becomes a build artifact nobody reviews, and reviewing the schema is
the point.

There is a third root type, `Subscription`, for a stream of events pushed to
the client. The transport under it is exactly what you built in the realtime
lesson: SSE or a WebSocket carrying `graphql-transport-ws` frames, with the
same per-client goroutine, the same bounded buffer, the same slow-consumer
question. GraphQL gives it a schema; it does not give it different physics.

## Resolvers, one field at a time

Here is the idea everything else follows from. A GraphQL server does not
execute a query. It executes a *tree of fields*, and each field has a function:

```go
type Resolver func(ctx context.Context, parent any, args Args) (any, error)
```

That is the whole execution model. `Query.posts` runs and returns five posts.
Then `Post.title` runs — five times, once per post. Then `Post.author` runs,
five times. Each call knows its parent and its arguments and nothing else: not
its siblings, not how many times it is about to be called, not what the client
asked for elsewhere in the tree.

That ignorance is the source of GraphQL's flexibility *and* of its defining
performance hazard. Read `resolve.go` in the exercise and notice that the naive
`Post.author` resolver is completely correct:

```go
func(ctx context.Context, parent any, _ Args) (any, error) {
	return store.AuthorByID(ctx, parent.(Post).AuthorID)
}
```

One post, one author, one row. It would pass any review. It is also the bug.

## N+1, measured

Run `{ posts(first: 10) { title author { name } } }` against that resolver and
count the trips to the database: one for the list of posts, then one per post
for its author. Five posts, six queries. Five hundred posts, five hundred and
one — from a query the client thinks is *one* request, over a schema where
nothing looks wrong.

This is the **N+1 problem**. You have met it before in S4's SQL work, where the
cure was a join you write once. GraphQL makes it structural rather than
occasional, for two reasons:

1. **You cannot see it in the schema.** `author: Author!` says nothing about
   how many queries it costs.
2. **You cannot fix it with a join.** The resolver for `Post.author` does not
   know that 499 sibling posts are also about to ask for an author. There is no
   place to put the join, because there is no place that sees the whole list.

The exercise's tests count store calls, so "this is slow" becomes a number:
`AuthorByID` called 5 times, versus `AuthorsByIDs` called once with 3 keys.
Make performance claims you can assert.

## Dataloaders: batch and cache

The fix, invented for exactly this problem, is a **dataloader**. It is two
ideas in one small object:

- **Batching.** `Load(key)` does not fetch. It records that the key is wanted
  and hands back a placeholder. Later, all the collected keys go to the
  database in *one* call — `WHERE id IN (…)` — and the placeholders fill in.
- **Caching, per request.** The same key asked for twice returns the same
  placeholder. Four posts by two authors are two rows, and a comment written by
  an author the post level already loaded is free.

"Later" is the interesting word. Something has to decide when the batch is
complete, and the standard answer in production servers is a *time window*:
resolve siblings concurrently, let the loader collect keys for a tick, then
fire. It works — and it makes every test about batching a race against a timer,
which is exactly the kind of test you learned to refuse in S5.

The executor in this exercise takes the deterministic route. It resolves the
query one **level** at a time: run every resolver at this depth, then dispatch
the loaders, then unpack the results into the next level. A resolver that wants
a batched value returns a `Deferred` — a function the executor calls *after*
dispatch:

```go
res := l.Authors.Load(parent.(Post).AuthorID)   // queue the key, fetch nothing
return Deferred(func() (any, error) { return res.Get() }), nil
```

Same batching, no goroutines, no timer, and a test can assert "exactly one call
to the store" without flaking. Real servers reach the same place with more
machinery; the *shape* — queue a key, return a promise, dispatch at a known
point — is identical.

Three details that bite in production:

- **The cache is per request, and must stay that way.** A loader that outlives
  a request is a cache with no invalidation, and in an API where authorization
  decides what a user may see (the authorization lesson), a stale entry hands
  one user another user's row.
- **The batch function must answer every key it was given.** Returning a map
  keyed by id — as the exercise's store does — makes that unmissable. The
  classic dataloader API returns a slice that must be *in key order*, and the
  classic dataloader bug is one misaligned index silently giving every user the
  wrong name.
- **A key with no row is a per-key error, not a batch failure.** One missing
  author must null one field, not fail the query.

Batching does not make the fetch free. `IN (…)` with 5,000 ids is its own
outage, which is why loaders take a maximum batch size, and why the complexity
limiting below is not optional.

## Errors: partial data is normal

REST answers one question, so one status code covers it. GraphQL answers a
tree, and parts of a tree can fail independently. So the response carries two
top-level keys:

```json
{
  "data": {"posts": [{"title": "…", "author": {"name": "Ada Lovelace"}},
                     {"title": "The orphaned draft", "author": null}]},
  "errors": [{"message": "author \"ghost\": not found",
              "path": ["posts", 1, "author"]}]
}
```

The client got everything that resolved. One field is null, and `path` says
precisely which one. **The HTTP status is 200**, because the request was
understood and executed — the failure is in the data, not in the transport.
A client that only checks `resp.StatusCode` will happily render a half-empty
page and report no problem, which is the single most common GraphQL client bug.

There is one status distinction worth keeping, and the exercise makes it: a
request that never executed — unparseable query, unknown field, over the
complexity limit — is a **request error**, and answers **400 with no `data`
key at all**. Older servers answer 200 for those too; the GraphQL-over-HTTP
specification now distinguishes them, and it is worth distinguishing, because
"your query is wrong" and "your data is partly missing" are different bugs for
different people (remember 400-vs-422 from last lesson).

Two habits worth forming:

- **Resolver errors go to the client.** This executor puts `err.Error()`
  straight in the response, which is fine for a lesson and wrong for a service:
  the same rule as last lesson applies, and production servers map internal
  errors to a safe message plus a machine-readable code in `extensions`, and
  log the detail.
- **Errors are structured, not prose.** `extensions: {"code": "FORBIDDEN"}` is
  what a client switches on. Message text is for humans and will change.

## Why GraphQL needs its own hardening

Last lesson bounded what a request could cost: body size, rate, time,
per-endpoint. Those bounds all assumed something that is no longer true —
**that you decide what an endpoint does.** Here the client sends the work
itself, in a 200-byte body, over one URL. Your rate limiter counts one request.

The two cheap-to-write, expensive-to-serve shapes:

**Deep.** The schema in this exercise has a cycle: a post has an author, an
author has posts, and so on forever. Nothing in the type system stops it.

```graphql
{ posts { author { posts { author { posts { title } } } } } }
```

**Wide.** No cycle needed, just multiplication:

```graphql
{ posts(first: 100) { comments(first: 100) { author { name } } } }
```

That is 10,000 author resolutions. Batched, it is a handful of queries with
enormous result sets; unbatched, it is 10,101 of them. Either way the client
spent five lines.

So a GraphQL endpoint needs two guards that REST does not:

- **Depth limiting** counts nesting and refuses past a maximum. Cheap, blunt,
  and the only thing that bounds a cyclic schema.
- **Complexity limiting** estimates cost *before* executing: each field costs
  its weight times the number of parents it will run for, and a list field
  multiplies the parent count for everything inside it — by the `first` the
  client passed, or by the field's default page size when the client passed
  nothing. That default is the important half: a list field with no bound and
  no assumed page size looks free to the analyser and returns your whole table
  at runtime.

Neither subsumes the other, which is the point of the exercise's two rejection
tests: the deep query above is complexity-cheap with `first: 1` everywhere, and
`{ posts(first: 5000) { title } }` has a depth of 2. Run both checks, before
the first resolver — after the first resolver, the work has already begun.

Four more, named so you know they exist: **timeouts** on the request context
(`http.TimeoutHandler` still applies, and every resolver takes a `ctx`);
**disabling introspection** in production, which hides the schema from a
casual browser and from nobody serious; **persisted queries**, where clients
register their queries at build time and send an id at runtime, which turns
arbitrary-query risk back into a fixed endpoint list and restores caching; and
**paginating every list field**, because a list without a bound is the
complexity analyser's blind spot.

Authentication and authorization do not change: the identity middleware from
this pack's authentication lesson still runs at the edge, and the policy check
still happens where the object is loaded — which in GraphQL is inside a
resolver, per field, and is easy to forget on the sixth nested type. One
endpoint does not mean one authorization decision.

## When not to choose GraphQL

The honest comparison, once the demo is over:

| | REST | GraphQL |
|---|---|---|
| **Who shapes the response** | server | client |
| **HTTP caching** | free: URL + `ETag`, works in browsers, CDNs, proxies | none: one URL, one POST body — you build application-level caching |
| **Cost of a request** | known per endpoint | depends on the query; must be estimated and bounded |
| **Data fetching** | one handler sees the whole request | per-field resolvers; batching is a permanent obligation |
| **Errors** | status codes clients already handle | 200 plus a partial tree; clients must read two keys |
| **Versioning** | `/v2`, or additive fields | additive fields plus `@deprecated`; field usage tracking tells you when a field is safe to remove |
| **Tooling floor** | `curl` | a client library, codegen, a schema registry |

Reach for GraphQL when: many different clients need many different shapes of
the same graph (a web app, an iOS app, and a partner integration over one
schema); the data really is a graph and clients keep asking for it three hops
deep; and your organisation can afford the schema as a governed, reviewed
artifact.

Stay with REST when: there is one client and you control it (you are inventing
a query language to talk to yourself); caching at the edge is doing real work
for you — a public read-heavy API is exactly the case GraphQL is worst at;
the API is a handful of endpoints; or the payload is not a graph at all (file
upload, streaming, bulk export). And note the practical middle: adding one
`/posts?include=author,comments` endpoint solves under-fetching for a lot of
services without any of the above.

The most expensive mistake is adopting GraphQL and then not doing the work it
requires — no loaders, no complexity limits, no schema review. That is a REST
API with worse caching and an unbounded cost per request.

## About gqlgen, and what this exercise does instead

In Go, the standard choice is **gqlgen**: schema-first, code-generating, and
what you should reach for at work. You write the SDL, run
`go run github.com/99designs/gqlgen generate`, and it produces the type-safe
Go interfaces you fill in. (`graphql-go/graphql` is the older code-first
alternative; `gqlgen` is what new services use.)

This exercise deliberately does not use it, and the reason is pedagogical:
gqlgen's generated executor is thousands of lines you would not read, and the
things this lesson is about — the resolver-per-field walk, the point where
loaders dispatch, how complexity is counted — would happen inside code you
treat as a black box. So the module is a ~200-line executor over the standard
library, supporting one operation, literal arguments, no fragments and no
variables. It is not a GraphQL server; it is the part of one you need to be
able to draw.

The practical consequence: the module has **no dependencies**, no `go.sum`,
and nothing to regenerate. When you use gqlgen afterwards, everything in this
lesson maps onto it directly — its resolvers have this signature, its
`dataloaden` loaders have this shape, and its complexity extension charges the
arithmetic you are about to implement.

## Exercise

Open [`exercise/`](exercise/) — a Go module, package `gql`. Read
`schema.graphql` first (the contract), then `execute.go` (how a query becomes a
tree of resolver calls, one level at a time). Provided and working: the parser,
the executor, validation, the instrumented store, the HTTP transport, and the
naive resolvers. Yours: the dataloader, three batched resolvers, and the two
limits.

Acceptance criteria:

1. `Loader.Load` queues a key and returns a `*Result` **without fetching**;
   the same key twice returns the same `*Result`, so one key is one row per
   request.
2. `Loader.Dispatch` calls its batch function **exactly once** for all queued
   keys and resolves them; an empty queue makes no call at all; a key already
   resolved is never queued again.
3. A batch function that fails gives its error to every key it was given; a key
   the batch function did not return gets a per-key error wrapping
   `ErrNotFound`, naming the key, while its siblings resolve normally.
4. `batchedPostAuthor`, `batchedPostComments` and `batchedCommentAuthor` load
   through the request's loaders and return a `Deferred`, following
   `batchedAuthorPosts`. `comments` honours `first` through `applyFirst` and
   returns `List(…)`.
5. `{ posts(first: 10) { title author { name } } }` costs **one**
   `AuthorsByIDs` call carrying the three distinct author ids, and zero
   `AuthorByID` calls — against five `AuthorByID` calls on `NaiveSchema`.
6. The two-level query costs **three** store calls in total: the comment
   authors are already in the loader's cache from the post level.
7. The batched and naive schemas return byte-identical `data`. Batching is
   invisible to the client.
8. `Depth` counts nesting: a field in the operation's own selection set is
   depth 1, each nested selection set adds one.
9. `Complexity` charges `Weight × parents` per field (weight defaults to 1),
   and a list field multiplies the parent count for its children by its
   `first` argument, or by `PageSize` when the client omits it.
10. `Limits.Check` refuses a query that is too deep *or* too complex, checking
    depth first, with a message naming the measurement and the limit; a
    non-positive limit disables that check. A refused query reaches **no
    resolver**: the store sees zero calls, the status is 400, and the response
    has no `data` key.

Run the tests from inside `exercise/`:

```sh
cd exercise
go test -race ./...
```

Work `loader.go` first — the loader tests are self-contained, and until they
pass nothing else can. Then the three resolvers, then `limit.go`.

## Further reading

- [GraphQL specification — Execution](https://spec.graphql.org/October2021/#sec-Execution)
- [GraphQL over HTTP specification](https://graphql.github.io/graphql-over-http/draft/)
- [gqlgen — getting started (schema-first codegen for Go)](https://gqlgen.com/getting-started/)
- [graphql/dataloader — batching and caching](https://github.com/graphql/dataloader)
- [GraphQL — Best practices: security and rate limiting](https://graphql.org/learn/security/)
