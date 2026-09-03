# Tutor notes — System Design Intro

## Where the learner is

First lesson of S6. They have shipped real Go services (S3/S5): HTTP, gRPC,
databases, profiling, observability — so they know what one production box
costs and does. What is new is *designing above the box*: requirements
discipline, estimation, trade-off framing. Expect strong implementation
instincts and weak scoping instincts — they will want to draw architecture
immediately. The whole stage funnels into `shared.systems.design-capstone`
(and then S7); the habits graded here (stated assumptions, visible
arithmetic, flip conditions) are the ones that capstone's rubric reuses.

## Running the design review

Verification is a conversation against the four worksheets. Run it as a
friendly interviewer, not a lecturer. Protocol:

1. **Read all four worksheets first**, silently noting: bare numbers without
   assumptions, adjectives posing as targets ("fast", "reliable"),
   architecture leaking into worksheet 1, and any trade-off missing a flip
   condition. These are your agenda.
2. **Play the sponsor** for their worksheet-1 questions. Canonical answers
   (improvise consistently for anything else):
   - Growth: 10M registered end of year one, hoped 3× in year two.
   - "Instant": sponsor accepts "feed visible under ~1 s on a phone".
   - Downtime: launch campaign is 2 weeks; an hour of downtime there is a
     crisis, off-campaign brief blips are tolerable.
   - Feed staleness: a new photo may take "a minute or two" to reach
     followers; own uploads must appear to the uploader immediately.
   - Losing photos: never acceptable after the app said "uploaded".
   - Follower graph: median follower count small (~50), but plan for a few
     accounts with millions.
   After answering one question, ask: *"Which of your numbers or decisions
   does that answer move?"* — worksheet answers that move nothing were not
   real questions.
3. **Audit two calculations live.** Pick one write-path and one
   storage/bandwidth line; have them re-derive it aloud. Reference envelope
   (theirs may differ — grade the method, and probe any input off by more
   than ~10× from these):
   - DAU 10-20% of 10M → 1-2M
   - uploads ~1/DAU/day → 1-2M/day ≈ 10-20/s avg, ~50/s peak
   - views 20-50/DAU/day → 20-100M/day ≈ 200-1,000/s avg, low thousands peak
   - read:write ≈ 20-50:1
   - storage ≈ 1.5-2M × ~3.3 MB ≈ 5-7 TB/day ≈ ~2 PB/year
   - egress ≈ views × 300 KB ≈ 6-30 TB/day ≈ 0.1-0.4 GB/s avg
   The intended dominant-axis answer: storage/bandwidth (photo bytes), with
   read QPS a defensible second. Accept either if argued from their numbers.
4. **Attack one trade-off.** Take their Decision 1 pick and argue the other
   side ("blobs in the DB give you transactional delete — your legal
   requirement — for free; why not?"). Pass signal: they defend with costs
   and requirement IDs, concede where the alternative genuinely wins, and
   their flip condition survives contact.
5. **Walk the plan.** Have them narrate the upload flow from worksheet 4;
   pick one box and ask "which requirement pays for this box?" and one 10×
   bottleneck and ask "what's the first symptom you'd see on a dashboard?"
   (ties to their S5 observability).
6. Close with the quiz (quiz.json), skipping anything already demonstrably
   covered in conversation.

Timebox: 25-40 minutes. If a worksheet is materially incomplete, stop —
send them back with the remediation ladder rather than reviewing a stub.

## Common misconceptions

- **Jumping to boxes** — architecture in worksheet 1, or a design defended
  by fashion ("microservices because scale"). Pull them back: which
  requirement pays for it?
- **Precision theater** — "23.148 uploads/second". Estimation is powers of
  ten; false precision signals they missed the point of the exercise.
- **Silent assumptions** — a DAU number with no source. The number may be
  fine; the silence is the defect being graded.
- **"Best practice" as an argument** — any pick justified without a named
  alternative and its cost. There are no best designs, only fits.
- **Average-only thinking** — sizing everything to mean traffic, forgetting
  the launch campaign the brief explicitly warns about.
- **Treating nice-to-have as rejection** — scoping something out is a
  recorded decision with an expiry, not a dismissal.

## Grilling points

- "Your A1 says 15% DAU. The sponsor now says 50% — walk me through exactly
  which later numbers change, and does your dominant axis flip?"
- "You made likes a must-have. Sell me the version-one that ships two weeks
  sooner without them." (Can they renegotiate scope, or is the sheet frozen
  in their head?)
- "Storage came out near petabytes/year. What real-world anchor makes you
  believe that, rather than suspect a slipped decimal?"
- "Give me the strongest honest argument *for* the alternative you rejected
  in Decision 1 — then tell me why you still win."
- "You have one hour and a hostile audience: which phase of your agenda do
  you refuse to cut, and why that one?"
- "A few accounts have millions of followers. Which worksheet line does that
  break first?" (Seeds the feed fan-out problem — later lessons solve it;
  here they only need to *spot* it.)

## Grading rubric

Grade estimation *habits* and framing, never trivia recall or agreement
with the reference numbers.

- **A** — All four worksheets complete per the acceptance criteria; every
  number traces to a justified assumption with visible arithmetic; sanity
  checks are real comparisons; both trade-offs carry honest costs on the
  picked side and a concrete flip condition; live re-derivation is fluent;
  when the sponsor's answer changed an assumption, they updated downstream
  numbers unprompted.
- **B** — Worksheets complete and method sound, but one or two lapses: a
  bare number, a soft target ("fast"), a flip condition that is vague, or
  wobbly re-derivation that recovers with one nudge. Dominant axis named
  and defended.
- **C** — The mechanics are there but the reasoning is thin: assumptions
  justified by "seems right", trade-off costs one-sided, cannot say what a
  sponsor answer would change. Pass only after a focused redo of the weak
  worksheet lands in this session; otherwise iterate.
- **Fail** — Worksheets incomplete or architecture-first with numbers
  back-filled to justify it; cannot re-derive their own arithmetic; any
  "nothing would flip it". Remediate, don't advance — every later S6 lesson
  builds on these habits.

## Remediation ladder

1. "Pick any number on worksheet 2 and tell me where it came from. Now say
   the sentence 'I assume X because Y' for it — that sentence is the whole
   skill."
2. "Cover worksheet 2 and re-derive write QPS from just A1 and A2, out
   loud, rounding as you go." (Rebuilds the arithmetic habit without the
   sheet as a crutch.)
3. "For your Decision 1 pick, complete: 'the rejected option wins when …'.
   If you can't finish that sentence, list what the rejected option is
   genuinely better at — the flip condition is hiding in that list."
4. Walk one estimate together — you supply the structure (users → actions →
   per-second → peak), they supply every number and justification. Then
   they redo the remaining estimates alone.

## After passing

Preview: "You now have the skeleton every S6 lesson hangs from. Next:
networking deep dive — what TCP, TLS, and load balancing actually cost,
with code — so your boxes-and-arrows stop being magic."
