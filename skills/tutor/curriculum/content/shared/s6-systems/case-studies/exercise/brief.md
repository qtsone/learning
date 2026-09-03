# Briefs

Briefs A and B. Confirmed facts are facts — use them. Anything not stated is
yours to assume out loud, and your tutor will answer sponsor questions in
character during the review.

The third brief lives in [`brief-c.md`](brief-c.md) and stays sealed: it
belongs to the speed round, and reading it early spends the thing that case
measures.

---

## Brief A — Trackly

Trackly sells short links to marketing teams. A customer creates links for a
campaign, puts them in emails, ads, and printed material, and watches clicks
arrive on a live dashboard while the campaign runs.

### Confirmed facts

- 120,000 business accounts. **2,000,000 links created per day**,
  **400,000,000 redirects per day**. Audience is worldwide.
- Traffic is campaign-shaped, not smooth: a single customer's email send can
  produce **40,000 redirects/s for 3-5 minutes**. Ordinary hours run near
  the daily average.
- **~30% of links use a customer-chosen alias** (`trck.ly/spring-sale`) in a
  single global namespace, first come first served. About 2,000 paths are
  reserved (`/api`, `/login`, `/pricing`, …) and must never be issued.
- **Links are editable.** Marketers repoint a live link when a landing page
  moves; roughly 2% of links are edited at least once, sometimes while the
  campaign is running.
- **Links may expire** at a customer-set time; an expired link serves a
  branded expiry page, not a redirect.
- Average destination URL is ~200 bytes; campaign URLs with tracking
  parameters reach 500 bytes.

### Stated tolerances (non-negotiable)

1. Redirect p99 ≤ 100 ms, measured worldwide.
2. Redirect availability 99.99%. Dashboard and link creation: 99.9%.
3. An edit or an expiry takes effect **globally within 60 seconds**.
4. **Abuse takedown**: legal or the abuse team disables a link and it must
   stop redirecting **everywhere within 60 seconds**. This one is audited.
5. Dashboard click counts may be approximate (±1%) and up to 60 seconds
   stale — **except** for the ~5% of links on usage-billed plans, whose
   clicks are invoiced and must be **exact and auditable** for 3 years.
6. Raw click events are retained 90 days; rollups 3 years.
7. A destination that was ever served must remain recoverable for audit: the
   edit history of a link is itself data.

### Out of scope

Vendor and cloud choices, the dashboard frontend, authentication, billing
UI, QR codes. Assume a CDN with programmable edge locations, a
production-grade broker, and both a relational and a key-value store are
available. Bot and crawler classification exists as a library — you decide
where in the pipeline it runs and say why.

---

## Brief B — Huddle

Huddle is embedded support chat: a widget on a company's site or mobile app
where a customer opens a conversation and an agent answers.

### Confirmed facts

- 4,000 customer companies, **30,000 support agents**. At peak, **25,000
  agents online** and **250,000 open customer conversations**.
- An agent handles up to **6 concurrent conversations**; assignment is
  longest-wait-first among agents with the right skill tag.
- **12,000,000 messages/day**, peaking at ~3× the average, shaped by
  business hours per region. Median message 200 bytes; ~8% carry a file
  attachment averaging 400 KB.
- Customers are on flaky mobile networks and refresh pages mid-chat. Agents
  work with two or three sessions open (desktop app plus browser tabs).
- **Supervisors join conversations mid-stream** and must see the full
  history immediately. **Transfers** move a conversation to another agent or
  team; the transcript stays one ordered conversation across transfers.
- 35% of traffic is EU, 45% North America, 20% rest of world.

### Stated tolerances (non-negotiable)

1. A message accepted by the server is never lost and never shown out of
   order within its conversation.
2. Delivery p95 ≤ 500 ms to a connected participant in the same region.
3. Send/deliver availability 99.95%. History browsing: 99.9%.
4. First agent reply within 60 s for 90% of conversations — the routing
   path is on that SLA.
5. **Full transcripts are retained 7 years** (legal). Full-text search
   covers the last 90 days; older transcripts must be retrievable within 24
   hours.
6. **EU data residency**: messages, attachments, and transcripts for EU
   customer companies are stored and processed in the EU. Operational
   metadata may leave; message content may not.
7. A deploy must not drop conversations: reconnect is allowed, message loss
   is not.

### Out of scope

Voice and video, automated/AI replies, CRM integrations, the widget
frontend, and identity for the agent console. Assume a broker, a relational
store, an object store, and a search engine are available in every region.
