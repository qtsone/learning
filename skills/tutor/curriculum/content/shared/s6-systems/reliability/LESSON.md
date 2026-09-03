# Reliability & Operations

> `shared.systems.reliability` · ~2-3h · Stage: Systems & Design

## Objectives

By the end of this lesson you can:

- Define SLI, SLO, and error budget, and derive a concrete SLO with alerting
  thresholds for a given service.
- Design symptom-based alerts that page on user impact rather than causes,
  and explain why cause-based paging burns teams out.
- Walk through an incident response: detection, mitigation, communication,
  and a blameless postmortem with action items.
- Choose graceful-degradation strategies (fallbacks, load shedding, circuit
  breaking) for a dependency outage scenario and justify them.

## Reliability is a feature with a price tag

Everything in this stage so far — replication, partitioning, caching, queues,
shedding — exists to keep a system useful when parts of it misbehave. This
lesson is about the discipline that decides *how reliable is reliable enough*
and what you do at 3 a.m. when the answer turns out to be "not this reliable".

Start by killing the reflex target: 100%. It is the wrong goal for almost
every system, for two reasons. First, your users cannot perceive it — their
phone network, their laptop Wi-Fi, and their browser already fail more often
than a 99.99% service does, so the last nines vanish into noise they blame on
the Wi-Fi anyway. Second, each additional nine costs roughly ten times the
effort — more redundancy, slower releases, more operational machinery — and a
system that must never break is a system you may never change. Reliability is
a feature, it competes with other features for engineering time, and like any
feature it needs a *target*, not a superlative.

The vocabulary that makes the target concrete comes from Site Reliability
Engineering, and it is three terms deep.

## SLIs: measure what the user experiences

A **Service Level Indicator (SLI)** is a metric chosen to stand for the
user's experience. The canonical shape is a ratio:

```text
SLI = good events / valid events
```

- *Availability of a feed:* successful feed requests / all feed requests.
- *Latency as a ratio:* feed requests answered in under 1 s / all feed
  requests. Phrasing latency this way — rather than "p99 < 1 s" — matters,
  because a ratio plugs into the same budget arithmetic as availability:
  every slow request is simply a bad event.
- *Durability:* photos still retrievable / photos acknowledged as stored.

Two rules keep an SLI honest. **Measure per user journey, not per
component** — "feed loads" and "upload succeeds" are journeys; "database CPU"
is not, because the database can be on fire while users are fine (cache is
absorbing it) and idle while users suffer. And **state the measurement
point** — a success ratio measured inside your service misses every request
the load balancer dropped before reaching you; measured at the load balancer
it misses DNS failures. Closer to the user is truer and harder; say where
you measure and what that blinds you to.

## SLOs and the error budget

A **Service Level Objective (SLO)** is a target for an SLI over a window:
"99.9% of feed requests succeed, over a rolling 30 days." Choose the target
from two inputs, neither of which is ambition: what users actually need
(would they notice 99.5%? would they leave?), and what you currently measure
(promising 99.99% when last quarter delivered 99.95% is fiction — you set a
target you already fail). If you have an SLA — a *contract* with penalties —
your SLO sits stricter than it, as the tripwire you hit first.

The payoff of a numeric target is its complement. The **error budget** is
`1 − SLO`: the amount of failure you are *allowed*. Over a 30-day window
(43,200 minutes):

| SLO | Budget (fraction) | As full-outage time / 30 days |
|-----|-------------------|-------------------------------|
| 99% | 1% | ~7.2 h |
| 99.9% | 0.1% | ~43 min |
| 99.99% | 0.01% | ~4.3 min |
| 99.999% | 0.001% | ~26 s |

Read the 99.99% row again: four nines means every human step — noticing,
opening a laptop, deciding — already blows the budget for the month. Targets
above three nines imply automated recovery, not faster humans.

The budget converts to requests too: at 1,000 requests/s average, a 30-day
window holds ~2.6 billion requests, so a 99.9% SLO grants ~2.6 million
failures. Partial outages spend it fractionally — a 12% error rate for 50
minutes burns the same budget as a 100% outage for 6.

The budget is not an accounting curiosity; it is a *policy instrument*. While
budget remains, you spend it: risky deploys, chaos experiments, planned
migrations. When it is exhausted, feature releases stop and the team works on
reliability until the window recovers. That rule turns the eternal fight
between "ship faster" and "stop breaking things" into arithmetic both sides
already agreed to — which is the entire point.

## Alert on symptoms, not causes

An alert that pages a human is the most expensive signal your system emits:
it buys interrupted sleep and spent trust, and the account is small. So a
page must mean, simultaneously: **users are affected (or will be within
minutes), and a human must act now**. Anything less urgent is a ticket for
working hours; anything less user-relevant is a dashboard line.

That standard sorts alerts into two families:

| Symptom (page-worthy) | Cause (dashboard/ticket) |
|---|---|
| Feed error rate burning the budget | Replica CPU at 95% |
| Uploads failing for users | Cache hit ratio dropped |
| Checkout latency SLI degrading | A pod restarted |
| — | Disk 80% full (ticket: fix this week) |

Cause-based paging fails for a structural reason: causes are many,
loosely correlated with impact, and often self-healing. A CPU alert fires
during every nightly batch job; a restart alert fires during every deploy.
Each false page teaches the on-call a little more thoroughly that pages are
noise — until the night a real one is muted from habit. This is **alert
fatigue**, and it is not a personal failing of the engineer who ignored the
page; it is the predictable output of a noisy pager. Teams in this state burn
out, and their real incidents get *detected by customers* — the most
expensive monitoring system there is. Causes still matter: they belong on
the dashboards you open *after* a symptom pages you, and in tickets when
they predict trouble (disk filling). They just do not wake people.

The precise way to page on a symptom is the **burn rate**: how fast you are
spending error budget, relative to the rate that would exactly exhaust it at
the window's end. Burn rate 1 means "on pace to spend the whole budget in 30
days" — fine. Burn rate = error rate ÷ budget: at a 99.9% SLO (0.1% budget),
a 1.44% error rate is a 14.4× burn. Practical alerting combines a fast and a
slow condition:

- **Page:** burn rate ≥ 14.4 sustained over 1 h — spends 2% of the monthly
  budget in an hour. A full outage hits this within minutes.
- **Page:** burn rate ≥ 6 over 6 h — catches the slow bleed a fast window
  misses.
- **Ticket:** burn rate ≥ 1 over 3 days — degradation worth fixing, not
  worth waking anyone.

Pairing each with a short secondary window (e.g. "and still burning over the
last 5 min") makes the alert stop firing promptly once you fix the issue.
The numbers are tunable; the structure — page only on budget-relevant burn,
at more than one time scale — is the lesson.

## When it breaks anyway: incident response

The budget exists because incidents will happen. What distinguishes mature
operations is not fewer incidents so much as *shorter, calmer* ones. The
shape of a well-run incident:

1. **Detection.** Ideally your burn-rate page; failingly, support tickets.
   Time-to-detect is a number you compute afterwards — via humans it is
   tens of minutes, via alerts it is two.
2. **Mitigation first, diagnosis later.** Stop the bleeding, then find the
   wound. If impact started shortly after a change shipped, *roll back
   before you understand why* — the correlation is enough, and rollback is
   the mitigation with the best odds. Understanding can wait until users
   are fine; heroic live debugging while the error rate climbs is a smell,
   not a virtue.
3. **Roles and communication.** Past trivial size, one person coordinates
   (incident commander) and does not touch keyboards; others execute; and
   someone owns communication — status page, support, stakeholders — on a
   stated cadence ("update every 20 min, even if the update is 'still
   investigating'"). Silence during an outage does not read as diligence;
   it reads as absence, and it sends support into battle unarmed.
4. **The blameless postmortem.** Written after recovery, while memory is
   fresh: timeline, user impact and budget burned, contributing factors,
   action items. *Blameless* means the analysis assumes competent people
   acting reasonably on the information they had, and asks why the *system*
   — tooling, alerts, process, defaults — allowed the failure. Not because
   it is polite: because it is accurate (the next engineer will act on the
   same information) and because blame teaches people to hide exactly the
   information the analysis needs. Blameless is not consequence-free —
   accountability attaches to fixing the system, as **action items with an
   owner and a deadline**. Note the plural in "contributing factors": real
   incidents have several — a defect *and* the alert that missed it *and*
   the rollback that was slow. "Root cause: human error" is a postmortem
   that has not started yet.

## Degrade gracefully

Your dependencies — a database, a cache, a third-party API — will fail, and
"we are down because they are down" is a design choice, not fate. Graceful
degradation is deciding *in advance* which parts of the experience you
sacrifice to protect the core. The toolbox, most of which you have met:

- **Deadlines everywhere.** Every dependency call gets a timeout derived
  from the user request's overall budget. A dependency that answers in 30 s
  is *down* for interactive purposes; without deadlines its slowness
  propagates upstream as your slowness, eating threads and queue slots.
- **Bounded retries with backoff and jitter.** Retries are load
  amplification pointed at a struggling dependency — remember the retry
  storms and backpressure from the scalability lesson. Retry a small number
  of times, spaced out, randomized, with an overall retry budget.
- **Fallbacks.** Serve stale cache instead of erroring (the caching
  lesson's TTLs become a resilience feature), serve a default instead of a
  personalized answer, serve a placeholder instead of an image. Degraded
  and present beats correct and absent — for non-critical content.
- **Load shedding.** When the core is at capacity, reject the least
  valuable work early and cheaply (scalability lesson) rather than failing
  all work slowly.
- **Circuit breaking** — the new tool. Track failures per dependency; past
  a threshold, *open the circuit*: fail calls immediately without attempting
  them. Periodically let a probe request through (*half-open*); if it
  succeeds, close the circuit and resume. This spares your latency budget
  (fail in 1 ms, not after a 2 s timeout), and spares the dependency a
  hammering while it recovers.
- **Fail open vs fail closed.** Per dependency, decide what its failure
  means: an unreachable auth service must fail *closed* (deny), an
  unreachable recommendation service must fail *open* (show defaults).
  Criticality decides, and deciding at 3 a.m. is too late.

In Go: the deadline tool is one you already use — `context`. A request-scoped
`ctx` with `context.WithTimeout` flows through every downstream call
(`db.QueryContext(ctx, …)`, `req = req.WithContext(ctx)`), so one budget
governs the whole call tree, exactly as S3/S5 drilled. A circuit breaker is a
small state machine (closed → open → half-open) guarded by a mutex; write
one, or reach for a vetted library like `sony/gobreaker` — the states, not
the package, are the concept.

## Exercise

Open [`exercise/`](exercise/). Framepost — the photo-sharing product this
stage opened with — has been live for six months, and you are taking over
its operations. Read `brief.md` (production facts plus a raw incident log),
then fill four worksheets: SLOs, alerts, an incident postmortem, and a
degradation plan. Start with `README.md`.

Acceptance criteria:

1. `01-slos.md`: both journeys get an SLI written as a good/valid ratio with
   an explicit measurement point; each SLO has a target and window justified
   against the brief's baselines (including why not one nine higher); error
   budgets are computed in minutes *and* requests with visible arithmetic.
2. `02-alerts.md`: all eight candidate alerts classified page/ticket/delete
   with one-line reasons and no cause-based page among them; two page alerts
   defined from your own SLOs with burn-rate thresholds and arithmetic
   showing how long a given outage takes to page.
3. `03-incident.md`: a classified timeline; time-to-detect and
   time-to-mitigate computed; budget burn computed with visible arithmetic;
   at least 3 contributing factors, none of which is a person; at least 4
   action items, each with an owner role, a deadline, and the gap it closes;
   the manager's message rewritten blamelessly.
4. `04-degradation.md`: all four outage scenarios handled — threatened SLO
   cited by ID, strategy chosen from the toolbox and justified, user-visible
   behavior stated, the sacrifice named, and a flip condition given.

There is no automated check. When the worksheets are done, tell your tutor —
verification is an operations review: expect to re-derive budget arithmetic
aloud and to defend every page-worthy alert.

## Further reading

- [Google SRE book — Service Level Objectives](https://sre.google/sre-book/service-level-objectives/)
  — the chapter this vocabulary comes from.
- [The Site Reliability Workbook — Alerting on SLOs](https://sre.google/workbook/alerting-on-slos/)
  — burn rates and multiwindow alerts, with the arithmetic worked out.
- [Google SRE book — Postmortem Culture](https://sre.google/sre-book/postmortem-culture/)
  — why blameless works, with example postmortems.
- [Google SRE book — Addressing Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/)
  — load shedding, retries, and degradation under real outage dynamics.
