# System Design Intro

> `shared.systems.design-intro` · ~2h · Stage: Systems & Design

## Objectives

By the end of this lesson you can:

- Translate a vague product prompt into functional and non-functional
  requirements, explicitly separating must-haves from nice-to-haves.
- Produce back-of-the-envelope estimates for QPS, storage, and bandwidth from
  stated user counts and justify each assumption.
- Explain why every design choice is a trade-off and frame at least two
  alternatives with costs for a given requirement.
- Structure a design discussion: requirements, estimation, high-level
  architecture, deep dives, bottlenecks.

## From one service to a system

Through S5 you built and operated a production-grade service: HTTP handlers,
a database, tests, profiles, dashboards. That is one box. System design is
what happens *before* anyone writes a handler: deciding which boxes exist,
what each is responsible for, how they talk, and — above all — whether the
whole thing survives contact with its expected load.

The medium changes too. Until now your output was code the compiler checked.
A design's output is prose, diagrams, and arithmetic, and nothing checks it
except scrutiny — yours, or a reviewer's. This stage teaches you to apply
that scrutiny deliberately, and the whole stage feeds its own design
capstone, and then S7, where you design and then build.

One warning up front: there is no correct answer in this stage the way a test
suite defines correct. There are defensible designs and indefensible ones,
and the difference is *process* — stated assumptions, numbers instead of
vibes, and named alternatives. That process is what this lesson drills.

## Vague prompts are the job

Real design work starts from something like: "We need photo sharing in the
app by Q3." Not a spec — a wish. Your first move is never to draw boxes; it
is to interrogate the wish until it becomes requirements.

**Functional requirements** are what the system does: "a user can upload a
photo", "a follower sees new photos". Write them as short capability
sentences. Then split them ruthlessly:

- **Must-have** — the product is pointless without it.
- **Nice-to-have** — real value, but version two. Naming something
  nice-to-have is not dismissing it; it is scoping. "Out of scope for now"
  is a design decision you write down, so nobody discovers it by surprise.

**Non-functional requirements** are how well it must do those things: how
many users, how fast, how available, how durable, how consistent, at what
cost. These rarely appear in the prompt. You get them by asking:

- How many users now? In a year? Daily vs registered?
- Read-heavy or write-heavy? (Almost everything consumer-facing is
  read-heavy — one upload, many views.)
- What is the cost of being down for five minutes? Of losing one record?
- How stale may data be? Must my own write be instantly visible to me?

When you cannot ask — or the answer is "we don't know" — you *assume, out
loud*: "I'll assume 15% of registered users are active daily; if it's 50%
everything below scales ×3." A stated assumption is checkable and
correctable. A silent one is a landmine. This habit — every number carries
its assumption — is the single strongest signal that you design like an
engineer rather than a brainstormer.

Non-functional requirements are what actually shape the architecture. The
same functional spec — "share photos with followers" — is a weekend project
at 100 users and a distributed-storage problem at 100 million. When someone
asks why two systems with identical features look nothing alike inside, the
answer is almost always a non-functional requirement.

## Back-of-the-envelope estimation

Before choosing any architecture, run the numbers. The point is not
precision — it is finding *which axis dominates*. Is this a storage problem?
A read-QPS problem? A bandwidth problem? The dominant axis tells you where
the design needs to spend its complexity, and which deep dive matters.

The rules of the game:

1. **Round aggressively.** Work in powers of ten and friendly factors.
   A day is 86,400 seconds — call it 10⁵. 3.7 million is 4 million.
   You are estimating orders of magnitude, not invoicing.
2. **State every input.** "2M uploads/day, assuming 1 photo per daily user"
   — never a bare number.
3. **Separate reads from writes.** Compute each per second, then note the
   ratio. A 100:1 read/write ratio and a 1:1 ratio produce different systems.
4. **Average, then peak.** Traffic is not uniform; multiply the average by a
   peak factor (2-5× is a sane default for consumer traffic; say which you
   picked).
5. **Sanity-check the result.** Compare against something you know. Does
   1.8 PB/year sound absurd? Large photo platforms really do add petabytes
   yearly, so no. Does 50 requests/second need twenty servers? It does not.

A worked example — a messaging feature, 2M daily users, each sending 20
messages of ~100 bytes:

```text
writes: 2M × 20        = 40M msgs/day  ≈ 40M / 10⁵ s  ≈ 400 writes/s avg
peak:   400 × 3        ≈ 1,200 writes/s
reads:  each msg read ~2× (recipient + sender's device sync) ≈ 800 reads/s avg
storage: 40M × 100 B   = 4 GB/day      ≈ 1.5 TB/year (before replication)
```

Four lines, one minute, and you already know: storage is trivial, write QPS
is modest, and if this system has a hard problem it is *delivery* (fan-out,
online/offline devices), not capacity. That conclusion — cheap to reach,
load-bearing for everything after it — is why estimation comes before
architecture.

Anchor numbers worth memorizing (rounded, deliberately):

| Quantity | Ballpark |
|---|---|
| Seconds per day | ~10⁵ (86,400) |
| Memory read | ~100 ns |
| SSD random read | ~100 µs |
| Round trip within a datacenter | ~0.5 ms |
| Round trip across an ocean | ~100-150 ms |
| One char / one 64-bit ID | 1 B / 8 B |
| Phone photo (original / resized for feed) | ~3 MB / ~300 KB |
| One commodity server, simple cached reads | ~10⁴-10⁵ req/s |
| One relational DB node, mixed queries | ~10³-10⁴ queries/s |

The last two are honest ranges, not constants — real capacity depends on the
workload, which is why you profiled in S5 instead of guessing.

In Go: you already own personal versions of these anchors. Your S5
benchmarks told you what one `net/http` endpoint does on your hardware, and
your pprof profiles showed what a database round trip costs relative to
handler logic. When you estimate "one server handles X", you are allowed —
encouraged — to cite your own measured numbers instead of the table above.

## Everything is a trade-off

There are no best designs; there are designs that fit their requirements.
Every choice buys something by paying something: latency for durability,
simplicity for flexibility, consistency for availability, money for almost
anything. (How deep the consistency/availability tension runs is the
distributed-systems lesson later this stage — for now it is enough that the
tension exists.)

So the unit of design reasoning is not "the right answer"; it is the
**framed decision**:

1. Name the decision. *Where do uploaded photos live?*
2. Name at least two real alternatives. *In the database as blobs; in object
   storage with a DB row holding metadata and a key.*
3. Attach the cost of each. *Blobs: one system, simple transactions — but
   backups balloon, the DB does work it is bad at, and serving bytes ties up
   DB connections. Object storage: cheap, built for large binaries — but now
   two systems can disagree, and you must handle "row exists, object
   missing".*
4. Pick, and tie the pick to a requirement. *Photos dominate storage by three
   orders of magnitude over metadata, so object storage wins.*
5. Say what would flip it. *If "photos" were 10 KB avatars at low volume,
   blobs-in-DB would be perfectly fine.*

Step 5 is the tell of a strong designer. If nothing could flip your
decision, you have a dogma, not a design. Interviewers and senior reviewers
probe exactly here: "when would the other option win?"

## The shape of a design discussion

Design conversations — interviews, RFC reviews, whiteboards with your team —
reward structure. The canonical shape, which the rest of this stage assumes
and the S6 design capstone will grade you on:

1. **Requirements & scope** (~10 min of an hour). Functional must/nice,
   non-functional targets, explicit out-of-scope list. Get agreement before
   moving on — designing the wrong thing quickly impresses nobody.
2. **Estimation** (~5 min). QPS read/write, storage, bandwidth. End by
   naming the dominant axis out loud.
3. **High-level architecture** (~15 min). Boxes and arrows: clients, entry
   point, services, storage. Walk the one or two most important flows
   end-to-end ("a user uploads a photo: request hits…"). Every box must
   earn its place by serving a requirement from step 1.
4. **Deep dives** (~20 min). Pick the one or two components your numbers
   flagged as hard, and go deep there. Depth where it matters beats shallow
   coverage of everything.
5. **Bottlenecks & evolution** (~10 min). What breaks first at 10× the
   load? What did you deliberately leave out, and when does that decision
   expire? Single points of failure?

Two habits make this structure work. **Drive with the numbers**: let step 2
choose your step 4 — "reads dominate 50:1, so I'll spend my time on the read
path". **Loop back**: when a deep dive uncovers a new requirement question,
return to step 1 and amend it; the phases are a checklist, not a one-way
street. And remember what evaluators actually grade: not whether your boxes
match their reference answer, but whether your process would survive a
different prompt tomorrow.

## Exercise

Open [`exercise/`](exercise/). You will run the full structure on a vague
brief — a photo-sharing product — by filling four worksheets:
requirements, estimates, framed trade-offs, and a design-discussion plan.
Start with `README.md`; the brief is in `brief.md`.

Acceptance criteria:

1. `01-requirements.md`: at least 5 functional requirements split must/nice,
   at least 4 non-functional requirements with concrete targets, an explicit
   out-of-scope list, and at least 3 questions you would ask the sponsor.
2. `02-estimates.md`: every input in the assumptions table has a one-line
   justification; read QPS, write QPS, storage/year, and bandwidth are each
   computed with visible arithmetic, average and peak; the dominant axis is
   named with one sentence of evidence.
3. `03-tradeoffs.md`: two decisions, each framed with two-plus alternatives,
   costs for every alternative, a pick tied to a requirement, and a "what
   would flip it" condition.
4. `04-discussion-plan.md`: a five-phase agenda with time budget, a text
   sketch of the high-level architecture, your chosen deep dives justified
   by your numbers, and at least two bottlenecks.

There is no automated check. When the worksheets are done, tell your tutor —
they will run a design review against them: expect to defend your numbers
and to be asked "when would the other option win?".

## Further reading

- [The System Design Primer](https://github.com/donnemartin/system-design-primer)
  — the canonical open-source map of this stage's territory; skim the intro,
  return per-lesson.
- [Interactive latency numbers](https://colin-scott.github.io/personal_website/research/interactive_latency.html)
  — Jeff Dean's "numbers everyone should know", visual and updated by year.
- [Designing Data-Intensive Applications](https://dataintensive.net/) — the
  book behind much of S6; chapter 1 ("Reliable, Scalable, Maintainable")
  pairs directly with this lesson.
