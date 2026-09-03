# Authorization Patterns

> `focus.web.authorization` · ~3-4h · Stage: Focus: Web Services

## Objectives

By the end of this lesson you can:

- Explain why authentication and authorization must be separate concerns in a
  service.
- Implement an RBAC model in Go with roles, permissions, and a checked policy
  lookup.
- Contrast RBAC and ABAC and choose between them for a given access-control
  requirement, justifying the choice.
- Implement authorization enforcement as `net/http` middleware so handlers
  cannot be reached without a passing policy check.
- Explain why authorization checks belong at the resource level (object
  ownership) and demonstrate preventing an IDOR-style bypass.

## Two questions, and only one of them is about identity

Last lesson answered **who are you**: a password verified against a slow, salted
KDF, a session id rotated on login, an identity carried on every request. This
lesson answers the other question: **what may you do**.

They are not the same question, and they fail differently.

- Authentication fails *loudly*. A broken login locks people out; you hear
  about it within minutes.
- Authorization fails *silently*. A missing check does not break anything. The
  service works, the tests are green, and every user can read every other
  user's data until someone notices — often someone outside your company.

That asymmetry is why broken access control sits at the top of the OWASP Top
Ten, and why this lesson is mostly about making the invisible failure visible.

The two concerns also want different code. Authentication is a lookup that
happens once per request, at the edge. Authorization is a *decision* that
depends on the subject, the operation, and frequently the specific object being
touched — the last of which nobody at the edge has seen yet. Keeping them
separate has a concrete payoff in the status codes:

| Code | Meaning | Sent when |
|------|---------|-----------|
| **401 Unauthorized** | badly named: *unauthenticated* | we do not know who you are |
| **403 Forbidden** | authorization | we know exactly who you are, and no |

If your service cannot tell those two apart, the two concerns have already
merged.

Notice how the authentication middleware in this exercise behaves: it looks up
the caller, attaches a `Subject` to the request context, and **rejects nobody**.
An unknown token simply leaves no subject behind. Every rejection — 401 for the
anonymous, 403 for the merely unwelcome — comes out of one place, the
authorization layer. One gate is one thing to audit.

## RBAC: a table you can review

The naive model grants permissions to users directly: `alice → docs.read`,
`alice → docs.write`, `bob → docs.read`, … It works for ten users and collapses
at a thousand, because every new user is a fresh chance to type the wrong list.
**Role-based access control** puts one indirection in the middle — users get
roles, roles carry permissions — so onboarding becomes "give her the editor
role", and changing what editors may do is one edit rather than a migration
over the user table.

In Go, the permission side is a table of **rules**, keyed by action:

```go
type Rule struct {
	Roles        map[Role]bool // who may perform the action at all
	RequireOwner bool          // must the subject own the object?
	OwnerExempt  map[Role]bool // roles that satisfy ownership without owning
}

var DefaultRules = map[Action]Rule{
	ActionList:   {Roles: Set(RoleViewer, RoleEditor, RoleAdmin)},
	ActionCreate: {Roles: Set(RoleEditor, RoleAdmin)},
	ActionUpdate: {
		Roles:        Set(RoleEditor, RoleAdmin),
		RequireOwner: true,
		OwnerExempt:  Set(RoleAdmin),
	},
}
```

Two properties of that snippet matter more than its contents:

- **The access-control model is data, on one screen.** Reviewing security means
  reading a table, not grepping thirty handlers for `role ==`.
- **Handlers name actions, never roles.** `ActionUpdate` is a fact about the
  endpoint; "editors may update" is a policy decision that can change without
  anyone opening `api.go`. The day product invents a `reviewer` role, you edit
  the table and ship.

RBAC's honest weakness is **role explosion**. Every "well, editors in the EU
team who are also on-call" becomes a new role, and after two years you have 60
roles, half of them held by one person. When you feel that pressure, the rule
you want is usually not about *who* the subject is but about *what the data is*
— which is the next section.

## ABAC: when the rule is about the data

**Attribute-based access control** decides from attributes of the subject, the
resource, and sometimes the environment: `subject.department == resource.
department`, `resource.classification <= subject.clearance`, `now() within
business_hours`.

You have already met the ABAC rule that every application needs, probably
without naming it: **ownership**. "You may edit your own document" cannot be
expressed in roles at all — it compares a subject attribute (`UserID`) with a
resource attribute (`OwnerID`). That is why the `Rule` above has a
`RequireOwner` field: the model in this lesson, like almost every real one, is
RBAC with a named, deliberately small dose of ABAC.

Full ABAC generalizes that field into a predicate:

```go
type Rule struct {
	Roles     map[Role]bool
	Condition func(Request) bool // arbitrary attribute logic
}
```

That one line buys enormous expressiveness, and charges for it:

| | RBAC | ABAC |
|---|---|---|
| **Reviewability** | a table a non-programmer can read | predicates you must execute to understand |
| **Expressiveness** | who you are | who you are, what the object is, when, from where |
| **Testability** | finite: roles × actions | combinatorial; you test the predicates, and hope |
| **Listings** | trivially becomes a query filter | may need every row loaded before it can be judged |
| **Audit** | "she is an editor" | "her department matched the document's, at 14:03" |

The choice, stated as a rule of thumb:

- Start with **RBAC as the spine**, plus a *closed* set of named relationship
  rules (owner, member-of-team). This covers the overwhelming majority of
  services and stays reviewable.
- Reach for **ABAC** when the rules genuinely vary with the data — multi-tenant
  isolation, classification levels, per-record sharing — and when you can pay
  for the tooling to test and explain those decisions.
- Do not build a general policy engine because you might need one. A
  `Condition` field you never populate is free; a policy language nobody can
  audit is not.

This lesson's exercise stops at ownership, deliberately. Adding a condition
function once you have the shape is a ten-minute change — which is the point.

## Deny by default

The single most valuable rule in this lesson:

> If no rule matches, the answer is no.

Not "no rule found, so nothing to enforce". Deny-by-default is what makes the
*forgotten* case safe, and the forgotten case is the one that ships. Consider a
route added on a Friday:

```go
{"POST /docs/{id}/archive", authz.ActionArchive, s.handleArchive},
```

The handler works. Nobody added a rule for `doc:archive`. Under
deny-by-default, every caller gets a 403 and someone files a bug on Monday.
Under default-allow, every caller on the internet can archive every document
and nobody files anything.

The same rule applies to *missing information*, not just missing rules. If a
rule requires ownership and the caller hands the policy no object, the honest
answer is not "well, probably fine" — it is a denial. Failing closed turns a
programming mistake into a 403 in your own test suite instead of a bypass in
production.

Watch it work in the exercise's decision ladder: the unknown action denies, the
zero-valued action denies, the nil resource for an ownership rule denies. The
only way out of `Check` with `Allow: true` is an explicit, reasoned grant at
the bottom.

## One decision function, two enforcement points

Authorization scattered through handlers is authorization you cannot audit.
The cure is the same one S5 used for error-to-status mapping: put the decision
in exactly one function.

```go
func (p *Policy) Check(req Request) Decision
```

`Check` is **pure**: subject, action, and (maybe) resource in; `{Allow, Reason}`
out. No HTTP, no database, no clock. That purity is what lets you enumerate its
entire behavior in a table test — 36 rows for three actions, four subjects and
three ownership states, all in milliseconds.

Enforcement is where that decision meets the request, and it happens at two
points, for a reason worth understanding.

**Route level, in middleware.** `Require(action)` wraps a handler with four
steps — subject out of the request context, `CheckRoute`, audit the decision,
then either the denial response or `next.ServeHTTP`. A denied request never
enters handler code, so no handler can half-do something before noticing.

Wiring it per route is where services get sloppy, so make forgetting impossible
with a route table:

```go
type route struct {
	pattern string
	action  authz.Action
	handler http.HandlerFunc
}

for _, rt := range s.routes() {
	mux.Handle(rt.pattern, s.policy.Require(rt.action)(rt.handler))
}
```

Now a route cannot be registered without naming an action, and a forgotten one
is the zero `Action` — which has no rule, and therefore is denied.

**Object level, in the handler.** Here is the part people get wrong: the
middleware *cannot* check ownership. It runs before the handler, which means
before the document is loaded, which means it does not know who owns it. A
middleware that loaded the object itself would be a second, divergent copy of
your data access — and would load it twice on every request.

So the route gate answers the coarse half ("may an editor ever update
something?") and hands the fine half forward with a reason that says so:
`role grants action, owner check pending`. The handler completes the decision
the moment it has the object:

```go
cur, err := s.store.Get(r.PathValue("id"))
// … err handling: ErrNotFound → 404 …
if !s.policy.Enforce(w, r, authz.ActionUpdate,
	authz.Resource{ID: cur.ID, OwnerID: cur.OwnerID}) {
	return // Enforce already wrote the 403 and logged the reason
}
```

That is still *one* enforcement point in the sense that matters: one function
interprets the rules, and every call site funnels through it. What is
distributed is the *invocation*, because only the handler has the data. What is
never distributed is the rule.

## IDOR: checking the token, not the object

Insecure Direct Object Reference is the bug that comes from stopping at the
route gate. It reads like this:

1. Bob logs in — a real user, real session, `editor` role — and requests
   `GET /docs/bob-doc`. Everything works.
2. He changes the URL to `GET /docs/alice-doc`.
3. The session is valid, the role permits reading, the gate passes, and the
   handler loads the document by id and returns it.

No token was forged, nothing was injected — the service was simply asked a
question it never thought to refuse. In the exercise's starter code, `Enforce`
returns `true` unconditionally, and this is exactly what happens.

The fix is structural, not clever: **every object you load on behalf of a
caller gets an ownership decision before it is used.** Read, update, delete —
and any other verb you add later.

Two consequences worth internalizing:

- **Listings leak the same way.** `GET /docs` passed a route gate that said
  "you may list", which is not the same as "you may see these rows" — so every
  row goes through the same read decision before it reaches the response.
  Filtering row by row is right at this size and wrong at ten thousand rows:
  there the same rule becomes a predicate in the query (`WHERE owner_id = ?`),
  which is faster and, more importantly, cannot forget a row. The rule is
  identical either way; only where it executes changes.

- **Ownership comes from the session, never the body.** The create handler sets
  `OwnerID` from the authenticated subject, and `doc.Draft` has no owner field
  at all, so a client that posts `{"owner_id": "carol"}` changes nothing. A
  service that trusts a client-supplied owner has handed out its own admin
  rights.

One honest trade-off: this service answers **403** when you ask for someone
else's document, and **404** when the document does not exist. That is
transparent and easy to debug — and it is an existence oracle: an attacker
learns which ids are real by the difference. The alternative, answering 404 for
both, hides existence at the cost of confusing your own users and support team.
Pick deliberately, per resource sensitivity; the change is one line in
`writeDenial`, because the decision is in one place.

What you must never do is send the *reason* to the client. `"not owner"` in a
response body confirms the object exists and belongs to someone else, for free.
The reason belongs in the log.

## Decisions you can test and audit

A boolean is a bad answer to a security question. `Decision` carries the answer
and the *why*:

```go
type Decision struct {
	Allow  bool
	Reason string // ReasonNotOwner, ReasonNoRule, ReasonRoleOverride, …
}
```

Reasons are exported constants, not string literals, because two audiences
depend on them. **Your tests** assert not just that a request was denied but
that it was denied *for the expected reason* — "the role lacks the action" and
"the resource was missing" are the same 403 and very different bugs, and a test
that checks only the status code passes happily while your policy denies for
the wrong one. **Your operators**, six months from now, answer "why was this
user refused?" with a log query instead of a debugger.

So both enforcement points log every decision — allows at info, denials at
warn — with subject, role, action, resource, owner, and reason. Note that the
logging lives at the enforcement points, not inside `Check`: the decision
function stays pure and trivially testable, and nothing gets logged twice.

Then test the whole thing as a matrix. Roles × actions × ownership states,
written out, with the deny rows spelled out as carefully as the allow rows. A
few rows in that matrix are worth staring at:

- editor, `doc:update`, someone else's document → **deny** (the IDOR row)
- viewer, `doc:update`, *their own* document → **deny** (owning something does
  not grant an action your role lacks)
- admin, `doc:update`, no resource supplied → **deny** (fail closed on missing
  information, privilege notwithstanding)
- anyone, `doc:archive` → **deny** (there is no rule)

If a security property is not a row in a table, it is a hope.

## Exercise

Open [`exercise/`](exercise/) — a documents service in the S5 layering, with
authorization missing:

```
authz/       the policy: types, rules, decision, enforcement   ← your work
authn/       stub authentication: token -> Subject             (provided)
doc/         domain: Document, Draft, ErrNotFound, Store       (provided)
memstore/    in-memory doc.Store                               (provided)
api/         routes and handlers                               ← four TODOs
cmd/docsd/   wiring: store, policy, server                     (provided)
```

Read `authn/authn.go` first: it is a fixture, not a lesson — hardcoded tokens,
no hashing, no expiry. Last lesson built the real thing. The starter is the
*insecure* version of this service: `Check` allows everything, `Require` is a
pass-through, and `Enforce` waves every object through. The test suites are the
security specification.

Acceptance criteria:

1. `Check` implements the decision ladder, in order: anonymous subject → deny;
   no rule for the action → deny; role not listed in the rule → deny; rule does
   not require ownership → allow; **no resource supplied for an ownership rule
   → deny**; role in `OwnerExempt` → allow; owner mismatch → deny; otherwise
   allow. Every return carries the matching `Reason…` constant.
2. `CheckRoute` gives the same answer as `Check` for everything it can decide
   without an object, and converts only the `ReasonMissingResource` denial into
   an allow with `ReasonOwnerCheckPending`. It never converts a role denial.
3. `Allows` reports `Check`'s verdict for a subject/action/resource triple,
   with no response and no logging.
4. `Require(action)` is middleware that reads the subject from the request
   context, asks `CheckRoute`, audits the decision, and on a denial writes the
   denial response *without calling the next handler*.
5. `Enforce` asks `Check` with the resource in hand, audits, writes the denial
   response when it denies, and returns whether the caller may proceed.
6. Anonymous and unknown-token requests get **401** with message
   `unauthenticated`; authenticated-but-refused requests get **403** with
   message `forbidden`. No response body ever contains a decision reason.
7. `GET`, `PUT` and `DELETE` on `/docs/{id}` enforce ownership after loading
   the document: a valid session with a permitted role may not touch another
   user's document, and a denied request leaves storage byte-for-byte
   unchanged. Admins pass by exemption.
8. `GET /docs` returns only the documents the caller may read — every row, for
   every role.
9. `POST /docs/{id}/archive` is denied for every role, including admin, and the
   document is *not* archived: the route has no rule.
10. Every decision, allow and deny, is logged as `authorization` with `user`,
    `role`, `action`, `allow` and `reason` attributes (plus `resource` and
    `owner` at the object level); denials log at warn level.
11. `go test -race ./...` is green and the code is `gofmt`-formatted.

Run the tests from the module root:

```sh
cd exercise
go test -race ./...
```

Work `authz/policy.go` first and get `authz` green — the matrix test is the
whole model, and once it passes, the HTTP half is plumbing. Then `Require` and
`Enforce`, then the four `TODO`s in `api/api.go`.

## Further reading

- [OWASP Top 10 — A01:2021 Broken Access Control](https://owasp.org/Top10/A01_2021-Broken_Access_Control/)
- [OWASP Authorization Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)
- [NIST — Role Based Access Control (RBAC) and RBAC standards](https://csrc.nist.gov/projects/role-based-access-control)
- [NIST SP 800-162 — Guide to Attribute Based Access Control](https://csrc.nist.gov/pubs/sp/800/162/upd2/final)
