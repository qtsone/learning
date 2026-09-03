# Tutor notes — Authorization Patterns

## Where the learner is

Second lesson of the web-services focus pack, straight after authentication.
They can now prove who a caller is; this lesson is the first time they are
asked to decide what a caller may do, and it is mostly a *design* lesson
wearing a coding exercise. Every ingredient is familiar — S5 layering, 1.22
mux, middleware, `log/slog`, table tests — so friction comes from judgment:
where the check belongs, why one function and not fifteen, and why the answer
must carry a reason.

Expect `authz/policy.go` to go quickly once they see the ladder is ordered, and
expect the two-phase split (route gate vs object check) to be the concept that
needs a second pass. The API half is plumbing after that.

The security content here is defensive: real constructions, named attack
shapes (IDOR, privilege escalation via a client-supplied owner, the
403-vs-404 existence oracle). Keep it that way — no probing tooling, no
"try it against a site".

## Common misconceptions

- **"Authentication is the security part; authorization is business logic."**
  It is the reason A01 tops the OWASP list. Ask what the service does today if
  the login is perfect and no check exists: every authenticated user is an
  admin.
- **401 vs 403 confusion** — sending 403 to anonymous callers or 401 to
  authenticated ones. Anchor it: 401 means "I do not know you", 403 means "I
  know you, and no".
- **Ownership belongs in the middleware.** The most common wrong answer, and it
  sounds tidy. Ask: which object? The middleware runs before the load. Any
  version that works reimplements the handler's data access, twice per request.
- **"Two call sites means the enforcement is scattered."** Distinguish the
  *rule* (one function, `Check`) from the *invocation* (wherever the data is).
  The thing you must not duplicate is the rule.
- **Owning something grants everything.** A viewer who owns a document still
  may not update it — the role gate runs first. The matrix row exists precisely
  because people implement ownership as an early `return true`.
- **Default-allow on the unknown.** `rule, ok := p.rules[action]` with the `ok`
  ignored, or a `switch` with a permissive `default`. This is the whole lesson;
  do not let a green suite hide it — ask them to point at the line that denies.
- **Treating a nil resource as "nothing to check".** It reads as harmless and
  it is a bypass: a handler that forgets to pass the object would sail through.
  Fail closed.
- **Filtering the listing "later, in the frontend".** Or not at all. The list
  test exists because a collection endpoint leaks exactly what the item
  endpoint refuses.
- **Trusting `owner_id` from the request body.** Watch for a learner "fixing"
  create by decoding an owner from JSON. That is a one-line privilege
  escalation.
- **Returning the reason to the client** — "helpful" 403 bodies saying "not
  owner". It confirms the object exists and belongs to someone else.
- **Logging only denials.** Then nobody can prove who *did* reach a record
  during an incident.
- **Testing statuses without reasons.** A 403 for the wrong reason is a latent
  bug: the same code will deny the wrong request tomorrow.

## Grilling points

- "Your service adds a `reviewer` role that may read everything but write
  nothing. Name every file that changes." (One: the rules table.)
- "Now product wants: reviewers may edit documents in their own department.
  Which model does that need, and what does it cost you?" (Attribute rule —
  make them describe the `Condition` extension and the testability price.)
- "Why can the ownership check not live in `Require`? Give me the mechanical
  reason, not the aesthetic one." (The object is not loaded; a middleware that
  loaded it duplicates data access and doubles the reads.)
- "Walk me through `CheckRoute`. Which single denial does it soften, and what
  would break if it softened one more?" (Only `ReasonMissingResource`; softening
  a role denial opens every ownership-free action.)
- "A teammate adds `POST /docs/{id}/publish` and forgets the rule. What happens
  in production, and how would you find out?" (403 for everyone; the audit log
  shows `no rule for action` — a bug report, not a breach.)
- "Bob has a valid session and the editor role. Explain, in the request's own
  order of operations, why he cannot delete alice's document."
- "You answer 403 for another user's document and 404 for a missing one. What
  can an attacker learn, and when would you change it?"
- "Where would this policy live if the same rules had to be enforced by a
  second service?" (Shared package, or a policy service — get them to notice
  that rules-as-data is what makes either possible.)
- "The listing filters row by row. At what point is that the wrong shape?"
  (When the page is large: the rule becomes a query predicate; same rule,
  different execution site.)
- "Which is worse: a policy that denies something it should allow, or one that
  allows something it should deny? Justify in terms of who finds out."

## Grading rubric

- **A** — All tests green under `-race`. `Check` is a single ordered ladder
  with an explicit deny for the unknown action and the nil resource; every
  return carries the right reason constant; `CheckRoute` delegates to `Check`
  rather than restating the rules; middleware stops denied requests before the
  handler; object checks sit immediately after the load and before any decode
  or write; the listing filters through the same decision. They can explain the
  two-phase split and where a new role or a new attribute rule would go.
- **B** — Green, with structural rough edges: `CheckRoute` copy-pasting the
  ladder, the object check placed after decoding the body, the filter
  hand-rolled with `d.OwnerID == sub.UserID` instead of the policy, or reasons
  as inline literals. Explanation of *why* is right but thin.
- **C** — Passing only after the remediation ladder, or they cannot say why the
  ownership check is not in the middleware, or they think the route gate
  already covers IDOR. Time-box remediation and re-grill before advancing.
- **Fail** — Tests failing; or an ownership check written as "if the role is
  admin, skip everything else" early-return that also skips the role gate; or
  the unknown action allowed. Remediate from `Check` outward — the pure
  function must be green before HTTP is worth discussing.

## Remediation ladder

1. "Run `go test -race ./authz/` alone. Read the first failing row out loud:
   which subject, which action, which ownership state, and what did the policy
   answer? The matrix names the case for you."
2. Policy stuck: "Write the ladder in English, one line per rung, top to
   bottom, before you write Go. Which rungs can only ever say *no*? Which
   single rung is allowed to say *yes* without looking at the resource?"
3. Route gate stuck: "`CheckRoute` has one job beyond `Check`. Ask `Check`
   without a resource: what does it answer for `doc:update` as an editor? Now,
   which of those answers is a real denial and which one just means 'not yet'?"
4. HTTP half stuck: "Take one endpoint, `GET /docs/{id}`, end to end. Where
   does the subject come from? What must be true before `store.Get` is called,
   and what must be true after it returns? Every other endpoint is that shape."
   Only if they are still stuck, sketch the middleware skeleton — subject,
   decision, audit, branch — with the bodies empty and let them fill it in.

## After passing

Preview: "Next you leave request/response behind for a while: WebSockets and
server-sent events, where a connection outlives the handler — and where the
question 'is this subject still allowed?' gets a lot more interesting."
