# Design review

The tests decide whether the service *works*. This file is where you say why it
is built the way it is — the half of a capstone that a review actually spends
its time on. Answer in your own words, a short paragraph each, naming the
alternative you rejected and what it would have cost.

Fill this in as you go, not at the end: the answers are easier to write while
the decision is still fresh, and a decision you cannot write down is usually one
you have not made.

## 1. Sessions, not tokens

This service authenticates with a server-side session in a cookie. Make the
argument against the alternatives (a stateless JWT; OIDC with a third-party
provider), and say what specifically would change your mind — what would have to
be true of this service's deployment for tokens to be the right answer?

## 2. The chain order

Write down your middleware order and defend each adjacency: why is
`SecurityHeaders` where it is, why does `RateLimit` sit where it does relative to
`Authenticate`, and what does the route gate cost you if it moves outward or
inward? Name one thing your order makes worse, because every order does.

## 3. One policy, four call sites

The route gate, the object check, the listing scope and the stream filter all
consult the same rule table. Explain why that is one enforcement point rather
than four, what would go wrong if the listing scope were written as "auditors
see everything", and which of the four you would expect a new teammate to forget
first.

## 4. SSE, not WebSockets

Justify the transport for this feature. What would have to appear in the product
for you to switch, and what does the switch cost in this codebase (auth,
middleware, proxies, tests)? Say what your stream does when a client falls
behind, and why that is the humane option here.

## 5. A queue in the database

The queue is a table in the same database as the data. Explain what that buys —
be specific about the transaction — and name the point at which you would move
to a dedicated broker. Then explain, in one sentence each: why the job id is
`notify:<task-id>` and not a UUID, and why the pool publishes the event after
the commit rather than the handler publishing it inside.

## 6. What you would do next

Two or three things this service needs before it faces the internet, and the one
you would do first. Being able to name your own gaps is the point.
