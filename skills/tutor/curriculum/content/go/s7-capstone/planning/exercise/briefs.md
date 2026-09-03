# Four briefs — inspiration, not a menu

Each of these clears the selection rubric: real packages, concurrency that
earns its place, state that survives a restart, at least one interface with two
implementations, and a demo you can run in under a minute. Each is also *one*
project, not three — which is what makes them useful calibration even if you
build none of them.

Read them for size and shape. A project of your own that clears the rubric is
better, because you will still care about it at hour 35.

---

## Courier — a backup daemon you would actually trust

> I keep meaning to back up my laptop. I have tried three tools; two needed an
> account, one silently stopped a year ago and I found out when I needed it.
> I want something I can read the source of, that copies a few directories
> somewhere else, tells me plainly when it last succeeded, and can put the
> files back.

**Hard axis: crash safety.** The interesting question is not "how do I copy a
file" — it is what is on disk after `kill -9` halfway through, and what happens
on the next run. Content addressing (a file is stored under the hash of its
bytes) gives you deduplication and makes repeat runs idempotent, which is
exactly the property that makes an interrupted backup safe to resume.

**Shape.** A CLI plus a long-running mode. Concurrency: a bounded pool hashing
and writing files in parallel, with a walker feeding it. Storage: a
content-addressed tree on disk plus a small index of snapshots. Interface: a
`Store` with a local-filesystem implementation and an in-memory one for tests
(a second real backend is a stretch goal, not a milestone).

**How it over-scopes.** Encryption, a remote server, a web UI, compression,
scheduling, and a restore-to-any-point-in-time UI. Pick at most one and make
the rest non-goals. Deduplicating chunks *within* a file (rolling hashes) is a
separate project wearing this one's clothes.

---

## Digest — a feed reader that respects other people's servers

> I follow 60 blogs. Feed readers either want a subscription or push me a
> firehose. I want one process that fetches them, keeps what it has seen, and
> gives me one page each morning with the new things on it.

**Hard axis: politeness under concurrency.** Fetching 60 feeds in parallel is
three lines. Fetching them without hammering a host that serves eight of them,
honouring `ETag`/`If-Modified-Since` so most fetches transfer nothing, backing
off when a host returns 429 or 500s, and finishing even when one server hangs
forever — that is the project.

**Shape.** A CLI to add and list feeds, a fetcher with per-host rate limiting
and a global worker pool, an embedded store of feeds and seen items, and a
small HTTP handler rendering today's digest. Interface: a `Fetcher` your tests
substitute with a fake serving fixture bytes, so the suite never touches the
network.

**How it over-scopes.** Full-text search, recommendations, a mobile app,
OPML round-tripping, and rendering every broken feed on the internet correctly.
Parsing exactly two formats well beats parsing five badly.

---

## Ledger — structured logs you can ask questions of

> Our little service writes JSON lines to a file. When something breaks I
> `grep`, and `grep` cannot answer "how many 500s per hour, by endpoint, over
> the last two days". I want to point something at the log directory and ask
> that.

**Hard axis: data that grows.** Ingest is easy; staying fast as the data grows
is the lesson. Time-bucketed segment files, an index over the fields you query,
a retention policy that deletes old segments, and compaction that merges small
ones — plus the honesty to measure whether your index actually helps.

**Shape.** An ingester (tail files or read stdin), a segment store on disk, a
query layer with a deliberately small query language (filter by field, group
by field, count over a time range), and an HTTP endpoint serving it. Interface:
a `Source` implemented by a file tailer and by a test source. Concurrency:
ingest, compaction, and queries running at once over shared state — the reason
you will care about the race detector.

**How it over-scopes.** A real query language with joins, a UI with charts,
distributed ingestion, and alerting. One aggregation over one time range,
done properly, is a capstone; SQL is not.

---

## Relay — a job queue whose failures you can explain

> Our web app does slow things inline: resizing images, sending mail. I want to
> hand a job to something and know it either happens or lands somewhere I can
> see, even if the worker dies mid-job.

**Hard axis: delivery semantics.** A lease with a visibility timeout, retries
with backoff, an attempt counter, a dead-letter path, and a durable record of
all of it. The design question that decides everything: what happens when a
worker takes a job and disappears — and what does that mean for jobs that are
not safe to run twice?

**Shape.** An HTTP API (enqueue, lease, ack, nack), a durable store, a
sweeper goroutine expiring leases, and a small worker package a client program
imports. Interface: a `Queue` with a durable implementation and an in-memory
one, so the worker package can be tested without a server.

**How it over-scopes.** Multiple queues with priorities, cron scheduling,
exactly-once semantics, a dashboard, and clustering. Exactly-once in
particular is a trap: the mature answer is at-least-once delivery plus
idempotent jobs, and writing that down as a non-goal is worth an ADR.

---

## Bringing your own

Clear it with your tutor before you write the PRD. It needs to survive four
questions:

1. What is the one flow that makes it worth having?
2. Where is the hard part, and is it hard for a reason you can name?
3. What state survives a restart, and what breaks if it is lost?
4. What do you already have — data, credentials, hardware — that it needs?

A project that answers all four in two sentences each is a good capstone. A
project that needs a paragraph for question 1 is probably three projects.
