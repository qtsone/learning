# Tutor notes — Authentication

## Where the learner is

First lesson of the web-services focus pack, straight after S5. They can build a
production HTTP service — mux patterns, middleware chains, layered handlers,
slog, graceful shutdown — and they have met fake clocks and table-driven
subtests in advanced testing. What is new here is *security thinking*: code that
is correct on the happy path and still catastrophic, and threat models
(offline cracking, enumeration, fixation, algorithm confusion) as the reason a
line of code exists.

Expect two failure modes. The confident learner writes a working login in twenty
minutes and misses every subtle requirement; the cautious one gets lost in
crypto anxiety and wants to be told "just use library X". Both need the same
correction: the security properties are *testable behaviours*, and the tests in
this exercise name every one of them.

This is also the lesson where "I found this on a blog" is dangerous. Push back
on any copied JWT or password snippet whose provenance they cannot explain.

## Common misconceptions

- **"Hashing is hashing"** — SHA-256 with a salt feels responsible and is still
  wrong: speed is the vulnerability. Make them say why bcrypt's cost knob exists
  and what happens to a stolen table of SHA-256 hashes overnight.
- **"The salt is the secret"** — salts are public, stored with the hash, and
  defeat precomputation and hash-sharing, not brute force. That's the *pepper*
  they may be half-remembering.
- **"I'll hash on the client"** — then the hash *is* the password. Client-side
  hashing can only ever be an addition to server-side hashing.
- **"Constant-time comparison everywhere"** or **"nowhere"** — both wrong. It
  matters for secrets an attacker can retry (MACs, API keys, reset tokens); a
  map lookup of a 256-bit random id is fine.
- **"A different message for unknown users is friendlier"** — it is an account
  enumeration endpoint. Same status, same body, same work.
- **"Logout = clear the cookie"** — the server must delete the session; the
  cookie is the client's copy of a key, not the lock.
- **"JWTs are more secure than sessions"** — they are less revocable. The
  security argument is usually a scaling argument in disguise.
- **"The token said the user is an admin"** — only after the signature verified.
  Watch for a `ParseHS256` that reads claims before (or without) checking the
  MAC, and for `==` where `hmac.Equal` belongs.
- **"exp is optional"** — a token with no expiry is a permanent credential.
- **"OIDC is just OAuth2"** — OAuth2 delegates *authorization*; OIDC adds the
  `id_token` that says who the user is.
- **Rotation confusion** — rotating a session id is not the same as expiring it,
  and rotating on login is not enough; privilege changes rotate too.

## Grilling points

Ask, in the learner's own words (quiz.json has the core set; these go deeper):

- "Your database leaks tonight. Walk me through what an attacker can do with
  your `password` column — and how that story changes with cost 4 versus 12."
- "Two logins, one with a real username and one with garbage. Why must the
  server spend the same effort on both, and how does the test prove it without
  a stopwatch?"
- "I hand you a session id I made up and then log in as myself. Why is your
  service not now handing me an authenticated session under my chosen id?"
- "Which of `HttpOnly`, `Secure` and `SameSite` protects against which attack?
  Which one would you lose most sleep over dropping?"
- "You promote a user to admin. Name two things that must happen to their
  session, and why the second one is not optional."
- "Here is a token with `alg: none` and a valid-looking payload. Trace your
  `ParseHS256` line by line and tell me where it dies."
- "Your CEO's laptop is stolen with a live JWT on it, expiring in 12 hours.
  What can you actually do? Now answer the same question for a session."
- "Where in the OIDC flow does the password go, and what stops someone who
  intercepts the authorization code from redeeming it?"
- "You are handed a mobile app plus a web app plus a partner integration. Which
  mechanism goes where, and what did you trade away?"

## Grading rubric

- **A** — All tests pass. The check order in `ParseHS256` is deliberate and they
  can justify it; `hmac.Equal` is used and explained; the dummy-hash path is
  understood as work-equalisation rather than cargo cult; cookie attributes are
  each defended; login and promote both rotate; logout deletes server-side. They
  can articulate the sessions/JWT/OIDC trade-off with revocation at the centre,
  and their code is clean Go — locks held briefly, no random ids from the
  wrong package, errors wrapped or sentinel-compared with `errors.Is`.
- **B** — Tests pass; one or two rationales are shaky (typically: why the
  unknown-user path still hashes, or why alg is pinned rather than read). Minor
  code smells: a mutex held across a `crypto/rand` read, `fmt.Errorf` where a
  sentinel was needed, claims parsed before the signature check but not *used*.
- **C** — Tests pass only after heavy hinting, or the JWT half is copied and
  they cannot walk it. Pass only if a focused remediation lands; the ordering
  rule in `ParseHS256` and the enumeration rule are non-negotiable.
- **Fail** — Tests red; or a passing solution built on `==` for the signature,
  claims trusted before verification, plaintext or reversible passwords, a
  session id from `math/rand/v2`, or an inability to say what `HttpOnly` does.
  Remediate, do not advance — the next lesson builds enforcement on top of this.

## Remediation ladder

1. "Read the failing test name aloud. Which security property is it asserting,
   and where in your code would that property live?"
2. Narrow to the layer: "Is this a hashing problem, a store problem, or an HTTP
   problem?" Then point at the one file, not the fix.
3. Name the rule, not the code: e.g. "an unknown username must cost the same as
   a known one — what would you have to call, and with what?", or "your parse
   trusts something before it verifies something. Which line runs first?"
4. For the truly stuck: walk the shape verbally — "step one splits, step two
   compares alg against the string HS256, step three recomputes the MAC and
   compares it with `hmac.Equal`, step four parses claims, step five checks exp"
   — and have them type it. Never paste `solution/`; if they have seen it,
   require a from-scratch re-implementation of that function plus an
   explanation of every line.

## After passing

Preview: "You can now prove who is calling. The next lesson answers the second
question — what they are allowed to do — and turns the role you just rotated a
session for into a policy decision that is enforced in one place and tested for
every role, action and owner combination."
