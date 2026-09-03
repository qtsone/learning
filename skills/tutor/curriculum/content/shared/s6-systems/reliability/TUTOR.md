# Tutor notes — Reliability & Operations

## Where the learner is

Ninth lesson of S6. They have operated one real service (S5: metrics,
dashboards, profiling) and now carry the stage's design habits: stated
assumptions, visible arithmetic, flip conditions. They have the mechanics
for degradation from earlier lessons — stale caches (caching), queues and
redelivery (message-queues), failover and failure modes
(distributed-systems), shedding and backpressure (scalability). What is new
is the *policy layer*: budgets as arithmetic, paging discipline, incident
process, and pre-deciding degradation. Expect strong instincts about causes
(they have debugged real services) and weak instincts about symptoms —
their pull will be toward paging on the machinery they know.

## Running the operations review

Verification is a conversation against the four worksheets. Run it as the
incoming engineering manager reviewing the new operations owner's plan —
friendly, numerate, allergic to vibes. Protocol:

1. **Read all four worksheets first**, silently noting: bare numbers
   without arithmetic, adjectives posing as targets, any cause-based page
   in 02, any person-shaped contributing factor in 03, any degradation pick
   without a named alternative. These are your agenda.
2. **Audit the budget arithmetic live.** Have them re-derive S1's budget in
   minutes and requests aloud, then Tuesday's bill. Reference envelope
   (grade the method; probe anything off by ~10×):
   - S1 ≈ 99.9%/30 d → 0.1% → ~43 min; 1,000/s × 86,400 × 30 ≈ 2.6B
     requests → ~2.6M failures allowed. (99.95% is defensible if they argue
     headroom honestly; 99.99% is indefensible — the measured baseline
     99.95% already fails it. S3 at 99.9% or 99.5% both fine if argued.)
   - Tuesday: impact 14:04→14:53 ≈ 50 min at ~12% errors. Burn = 12/0.1 =
     120×; budget consumed ≈ 120 × 50/43,200 ≈ 14%. About 7 such incidents
     exhaust a month.
   - Their Part B check: a 100% outage is 1,000× burn → crosses a 14.4×/1 h
     threshold in ~1 min; Tuesday's 120× crosses it in ~7 min — versus the
     actual 19-minute human detection. If they didn't notice their own
     alert beats Tuesday's detection by 12 minutes, hand them that gift and
     ask what it implies about A1.
3. **Attack one page.** Pick their weakest Part A classification and argue
   the other side ("replica CPU predicted the outage two minutes in — why
   won't you page on it?"). Pass signal: they answer with the fatigue
   mechanism (noisy page → muted page → missed page; Tuesday *proves* it —
   the CPU alert fired at 14:06 and bought nothing but a harmful restart).
   A7 is the subtle one: today the push provider *is* on the critical path
   because the call is synchronous — accept page or ticket only if the
   answer engages with that coupling, ideally by pointing at their
   scenario-3 fix.
4. **Replay the incident.** "It is 14:23 Tuesday, you are Dana, dashboards
   show 12% errors and p99 9 s. What is your first move?" Wanted:
   check-what-changed → correlate the 14:02 deploy → roll back before
   understanding why; plus "someone posts to the status page while I do".
   Restart-the-replica answers get the follow-up: what information would
   have stopped you? Then check their rewrite of the manager's message —
   it must keep the urgency and retarget it at the system.
5. **Argue one degradation pick.** Take scenario 2 or 3 and press the
   rejected alternative ("the queue is right there — why not queue uploads
   during failover?"). Pass signal: costs on both sides, the pick tied to
   an SLO ID, a flip condition that survives contact, and — for scenario
   2 — the stampede/thundering-herd risk named without prompting.
6. Close with the quiz (quiz.json), skipping anything already demonstrably
   covered in conversation.

Timebox: 30-45 minutes. If a worksheet is materially incomplete, stop —
send them back with the remediation ladder rather than reviewing a stub.

## Common misconceptions

- **"Aim for 100% / more nines are better"** — reliability is a feature
  with a cost curve (~10× per nine) and a perception floor (the user's own
  network). The target is an engineering decision, not a virtue score.
- **Uptime = availability** — thinking in binary up/down instead of request
  ratios. A 12% error rate for 50 minutes never shows on an up/down probe,
  yet it ate 14% of the month's budget.
- **Page on causes "to catch it early"** — feels proactive, produces
  fatigue. Tuesday's CPU alert is the counterexample: it fired early, was
  ignored (rationally — it cries wolf nightly), and then *misdirected* the
  mitigation. Causes go on dashboards you read after a symptom pages.
- **Diagnosis before mitigation** — the hero-debugging instinct. If impact
  correlates with a change, roll back first; understanding is for the
  postmortem.
- **Blameless = consequence-free** — no: accountability moves from naming
  people to shipping action items with owners and deadlines. The rewrite
  exercise checks exactly this.
- **"Root cause", singular** — accepting "bad deploy" as the whole story.
  Tuesday has at least four contributing factors; a postmortem that finds
  one has stopped early.
- **Retry harder as resilience** — retries amplify load on a struggling
  dependency (scalability lesson). Bounded, jittered, budgeted — or a
  circuit breaker instead.
- **SLO = SLA** — the contract with penalties is the SLA; the SLO is the
  stricter internal tripwire. Only relevant if they raise it, but keep the
  terms straight.

## Grilling points

- "Your S1 is 99.9%. Marketing wants to advertise 99.99%. Give me the
  two-sentence answer you'd send, with numbers." (Baseline 99.95% already
  fails it; four nines is 4.3 min/month — beyond human-speed response.)
- "Re-derive: how long does a total feed outage run before your Page 1
  fires? Walk the burn-rate arithmetic aloud."
- "Your budget is 60% consumed on day 12 of the window with no incident —
  a slow bleed. What do you actually *do* tomorrow morning?" (Budget as
  policy: investigate the bleed, slow releases; not "nothing, no page".)
- "Which of your action items would have shortened Tuesday the most?
  Defend it with the timeline numbers." (Detection: symptom page saves ~12
  min; mitigation: 'check deploys first' runbook saves ~18. Either is
  defensible — with arithmetic.)
- "Scenario 3: why is a circuit breaker better than lowering the timeout
  to 1 s? When is it worse?" (Faster local failure and recovery probing vs
  still burning a thread per call; worse when failures are rare blips and
  the breaker flaps — tie to half-open.)
- "Name a fail-*closed* dependency Framepost doesn't have yet but could —
  and why closed." (Auth/session validation is the classic; degrading it
  open means serving someone else's private data to save availability.)

## Grading rubric

Grade budget arithmetic, symptom-vs-cause discipline, blameless framing,
and SLO-tied justifications — never trivia recall or agreement with the
reference numbers.

- **A** — All four worksheets complete per the acceptance criteria; every
  budget number re-derivable aloud; no cause-based page and the fatigue
  mechanism articulated with Tuesday as evidence; postmortem factors are
  all system-shaped, action items map to timeline gaps with numbers;
  degradation picks cite SLO IDs, cost both sides, and carry real flip
  conditions; the 14:23 replay answer is rollback-first with comms.
- **B** — Complete and sound method, with one or two lapses: a wobbly
  re-derivation that recovers with a nudge, one alert misclassified but
  well-argued, a flip condition that is vague, or a replay answer that
  reaches rollback second instead of first.
- **C** — Mechanics present but reasoning thin: arithmetic copied but not
  re-derivable, symptom/cause sorted by intuition rather than the
  page-test, contributing factors that are thinly disguised people, or
  degradation picks with no costed alternative. Pass only after a focused
  redo of the weak worksheet lands in this session; otherwise iterate.
- **Fail** — Any worksheet materially incomplete; a cause-based page
  defended to the end; a postmortem that blames a person; or budget
  arithmetic they cannot reproduce. Remediate, don't advance — the
  capstone defense assumes these habits.

## Remediation ladder

1. "Cover worksheet 1 and tell me: 99.9% of a 30-day month — how many
   minutes of full outage? Say the multiplication out loud." (Rebuilds the
   budget-as-arithmetic reflex; everything else hangs from it.)
2. "For alert A1, answer only this: the last five times it fired, did a
   user notice anything? So what does the on-call learn each time?" (The
   fatigue mechanism, derived rather than recited.)
3. "Rewrite one contributing factor so no person could appear in it: start
   the sentence with 'The system allowed…' and finish it." (Then they redo
   the rest of the list alone.)
4. Walk scenario 3 together — you supply the question sequence (critical
   path? criticality of the dependency? what does failing open cost?),
   they supply every answer; then they redo one other scenario alone.

## After passing

Preview: "You can now say how reliable a system must be and what happens
when it isn't. Next: architecture patterns — how the boxes themselves get
organized as systems and teams grow — with these SLOs as one of the forces
that shape the choice."
