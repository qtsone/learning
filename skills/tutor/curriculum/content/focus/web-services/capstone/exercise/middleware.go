package board

import (
	"context"
	"net/http"
)

// subjectFor turns a session id into an identity, or reports that there is no
// longer one behind it.
//
// Two callers need it, and the second is the interesting one. Authenticate asks
// once per request. The event stream asks again on every event, because a
// connection that lasts hours outlives the answer it was given when it opened —
// and "destroying the target's sessions" only revokes anything if something
// re-asks.
//
// TODO: look the session up in s.sessions (which enforces expiry against the
// injected clock), load the account with s.store.UserByID, and build the
// Subject. Report false when either is gone.
func (s *Server) subjectFor(ctx context.Context, sessionID string) (Subject, bool) {
	return Subject{}, false
}

// Authenticate turns a cookie into an identity and puts it on the request
// context. It **rejects nobody**: an absent, unknown or expired session simply
// leaves the anonymous zero Subject behind, and every refusal — 401 for the
// anonymous, 403 for the merely unwelcome — comes out of the authorization
// layer instead. One gate is one thing to audit.
//
// TODO: read the session cookie, resolve it with subjectFor, and call next with
// a context carrying the Subject (WithSubject). A cookie naming a session that
// is gone should be cleared on the way past — the browser is holding a key to a
// lock that no longer exists.
func (s *Server) Authenticate(next http.Handler) http.Handler {
	return next
}

// Require is the route gate: it asks the policy what it can answer before any
// object has been loaded, audits the decision, and on a denial answers without
// calling the handler at all — so no handler can half-do something before
// noticing it was not allowed to.
//
// TODO: read the subject from the request context, ask s.policy.CheckRoute,
// audit it, and either write the denial (writeDenial) or call next.
func (s *Server) Require(action Action) Middleware {
	return func(next http.Handler) http.Handler {
		return next
	}
}

// Enforce completes a decision the route gate could not: it runs inside a
// handler, once the object is in hand. It writes the denial itself and reports
// whether the caller may proceed.
//
// The middleware genuinely cannot do this. It runs before the handler, which
// means before the object is loaded, which means it does not know who owns it —
// and a middleware that loaded the object itself would be a second, divergent
// copy of your data access.
//
// TODO: ask s.policy.Check with the resource in hand, audit the decision, write
// the denial when it denies, and return whether to continue.
func (s *Server) Enforce(w http.ResponseWriter, r *http.Request, action Action, res Resource) bool {
	return true
}

// rateLimitKey decides what "a client" means for the limiter.
//
// TODO: return a key that is per-account for an authenticated caller and per
// address otherwise (ClientIP). Two different logged-in users must never share
// a bucket, and one user's key must not change between their requests. Prefix
// the two kinds of key differently: a user id and an address must not be able
// to collide.
func (s *Server) rateLimitKey(r *http.Request) string {
	return ""
}

// RateLimit refuses a caller who has run out of tokens with 429 and a
// Retry-After header, and never calls next for them.
//
// TODO: spend a token for rateLimitKey(r) against s.limiter; on a refusal set
// Retry-After (RetryAfterSeconds) and write the error envelope; otherwise call
// next.
func (s *Server) RateLimit(next http.Handler) http.Handler {
	return next
}
