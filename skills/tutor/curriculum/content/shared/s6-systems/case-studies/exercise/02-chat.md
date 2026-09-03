# Case B — Huddle

Brief B. Budget ~70 minutes. Same rules: phases in order, arithmetic
visible, every pick with a loser and a flip condition.

The lesson's chat system was a consumer messenger where connection count
dominated everything. Huddle has the same functional shape and completely
different non-functional requirements. Let your estimates — not your memory
of the walkthrough — decide where the hard part is.

## 1. Requirements and scope

### Functional

| # | Capability | Must / Nice | Why |
|---|---|---|---|
| F1 | … | … | … |
| F2 | … | … | … |
| F3 | … | … | … |
| F4 | … | … | … |
| F5 | … | … | … |
| F6 | … | … | … |

### Non-functional

| # | Property | Target | Source |
|---|---|---|---|
| N1 | Delivery latency | … | … |
| N2 | Durability / ordering guarantee | … | … |
| N3 | Routing SLA | … | … |
| N4 | Retention and searchability | … | … |
| N5 | Residency | … | … |
| N6 | … | … | … |

### Out of scope for v1

- …

### Questions for the sponsor

| # | Question | What changes if the answer is X |
|---|---|---|
| Q1 | … | … |
| Q2 | … | … |
| Q3 | … | … |

## 2. Estimation

| # | Assumption | Value | Justification |
|---|---|---|---|
| A1 | Sessions per agent (devices/tabs) | … | … |
| A2 | Sessions per customer | … | … |
| A3 | Concurrent connections per gateway box | … | … |
| A4 | Recipients per message (all sessions that receive it) | … | … |
| A5 | Duration a conversation stays "open" | … | … |
| A6 | … | … | … |

```text
messages:     12M/day / 10⁵ s              ≈ …/s avg, ×3 ≈ …/s peak
deliveries:   peak sends × A4              ≈ …/s
connections:  250k customers × A2 + 25k agents × A1 ≈ …
gateway boxes: connections / A3            ≈ …  (+ headroom for …)
text storage: 12M × 200 B × 365 × 7        ≈ …
attachments:  12M × 8% × 400 KB × 365 × 7  ≈ …
search index: 90 days of text              ≈ …
```

**Dominant axis** (and the line that proves it):

> …

**A number in the brief that does not add up.** Compare the concurrent
conversation count against total agent capacity. State the contradiction,
state the assumption that resolves it, and say which sponsor question you
would ask:

> …

## 3. High-level design

```text
…
```

Walk three flows with a latency figure per step:

**Flow 1 — a customer opens the widget and sends the first message; no agent
is free for 40 seconds.**

1. …

**Flow 2 — an agent transfers the conversation to a colleague in another
team while the customer is typing.**

1. …

**Flow 3 — a supervisor joins a 200-message-old conversation.**

1. …

## 4. Deep dives

### D1 — Connections and routing

- Transport choice and why (say what you rejected): …
- Gateway sizing from A3, and what physically limits a box: …
- Session registry: what it stores, who writes it, how a crashed gateway's
  entries disappear: …
- Heartbeat interval and the failure it detects: …
- Load-balancing policy for long-lived connections, and why the obvious one
  is wrong: …
- **Deploy math.** All gateways restart during a release. How many clients
  reconnect, over what window, at what request rate?

  ```text
  …
  ```

- **Routing.** How a waiting conversation is matched to an agent
  longest-wait-first, with skill tags and a 6-conversation cap. Name the
  mechanism that prevents the same conversation being assigned to two
  agents, and the mechanism that prevents a conversation being lost when the
  assigning process crashes mid-assignment: …
- How you would measure the 60-second first-reply SLA (which SLI exactly): …
- **Chose … over …; it costs us …** · **Flips when**: …

### D2 — Conversation model and ordering

- Your partitioning key, and what it makes cheap: …
- Who assigns order, and what a client sorts by: …
- Why not timestamps (name both failure sources): …
- How a client detects and repairs a gap: …
- Transfers and supervisor joins: does a transfer start a new conversation
  or continue one? Justify with the brief's transcript requirement: …
- Two agents and a customer send at the same instant — what determines the
  final order, and can any participant disagree with it? …
- **Chose … over …; it costs us …** · **Flips when**: …

### D3 — Offline, reconnect, and acknowledgements

- What a customer's device stores so a mid-chat page refresh loses nothing: …
- The reconnect protocol, in four steps or fewer: …
- Message ids and deduplication: who generates the id, where dedupe happens
  on send, where it happens on render: …
- The acknowledgement ladder — list each distinct fact the product exposes
  and who owns it:

  | Acknowledgement | Meaning | Written by | Shown as |
  |---|---|---|---|
  | … | … | … | … |

- A customer closes the tab for two days and returns: what do they receive,
  and how is it bounded? …
- Attachment upload on a flaky network — what makes it resumable and
  idempotent: …
- **Chose … over …; it costs us …** · **Flips when**: …

### D4 — Retention, search, and residency

- Storage tiers with the arithmetic for each (hot, warm, cold), including
  what "retrievable within 24 hours" permits you to do:

  ```text
  …
  ```

- What the 90-day search index contains and what it costs: …
- Residency: draw the boundary. Which components are per-region, which are
  global, and exactly what data crosses:

  ```text
  …
  ```

- An EU customer's conversation is transferred to a US-based agent. What
  happens, and which requirement decides it? …
- Deletion: a company leaves Huddle and asks for erasure while legal
  retention still applies. Name the conflict and how your design records
  the decision: …
- **Chose … over …; it costs us …** · **Flips when**: …

## 5. Bottlenecks and 10×

Ten times the load: 120M messages/day, 250k agents, 2.5M open conversations.

| Drill check | Where it lives in your design | Breaks at what point |
|---|---|---|
| The singleton | … | … |
| The hot key or partition | … | … |
| The unbounded thing | … | … |
| The synchronous fan-out | … | … |
| The coordination point | … | … |

**What breaks first**: …

**First symptom on a dashboard**: …

**Next evolution step, and its cost**: …

**Which of your v1 simplifications expires first, and at what number**: …

## 6. Design statement

At most 200 words: what you are building, the dominant axis, your two
hardest-defended decisions, and the first thing that will break.

> …
