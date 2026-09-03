package board

import (
	"context"
	"net/http"
)

// subjectFor turns a session id into an identity, or reports that there is no
// longer one behind it. The session store enforces expiry on read, so "gone"
// here covers expired, logged out and destroyed by a role change alike.
//
// Two callers need it, and the second is why it is a function rather than four
// lines inside Authenticate. Authenticate asks once per request. The event
// stream asks again on every event, because a connection that lasts hours
// outlives the answer it was given when it opened.
func (s *Server) subjectFor(ctx context.Context, sessionID string) (Subject, bool) {
	sess, ok := s.sessions.Get(sessionID)
	if !ok {
		return Subject{}, false
	}
	user, err := s.store.UserByID(ctx, sess.UserID)
	if err != nil {
		return Subject{}, false
	}
	return Subject{UserID: user.ID, Role: user.Role, SessionID: sess.ID}, true
}

// Authenticate turns a cookie into an identity and puts it on the request
// context. It **rejects nobody**: an absent, unknown or expired session simply
// leaves the anonymous zero Subject behind, and every refusal — 401 for the
// anonymous, 403 for the merely unwelcome — comes out of the authorization
// layer instead. One gate is one thing to audit.
func (s *Server) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(SessionCookieName)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		sub, ok := s.subjectFor(r.Context(), c.Value)
		if !ok {
			// The browser is holding a key to a lock that no longer exists:
			// expired, logged out, or revoked by a privilege change.
			s.clearSessionCookie(w)
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithSubject(r.Context(), sub)))
	})
}

// Require is the route gate: it asks the policy what it can answer before any
// object has been loaded, audits the decision, and on a denial answers without
// calling the handler at all — so no handler can half-do something before
// noticing it was not allowed to.
func (s *Server) Require(action Action) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sub := SubjectFrom(r.Context())
			req := Request{Subject: sub, Action: action}
			d := s.policy.CheckRoute(sub, action)
			s.policy.Audit(r, req, d)
			if !d.Allow {
				writeDenial(w, sub)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Enforce completes a decision the route gate could not: it runs inside a
// handler, once the object is in hand. It writes the denial itself and reports
// whether the caller may proceed.
//
// This is still one enforcement point in the sense that matters — one function
// interprets the rules, and every call site funnels through it. What is spread
// out is the *invocation*, because only the handler has the data.
func (s *Server) Enforce(w http.ResponseWriter, r *http.Request, action Action, res Resource) bool {
	sub := SubjectFrom(r.Context())
	req := Request{Subject: sub, Action: action, Resource: &res}
	d := s.policy.Check(req)
	s.policy.Audit(r, req, d)
	if !d.Allow {
		writeDenial(w, sub)
		return false
	}
	return true
}

// rateLimitKey decides what "a client" means for the limiter.
//
// An account id is the strongest key available: it survives a change of
// address, it cannot be shared by everyone behind one NAT, and it is the thing
// a quota is actually about. Only a caller with no identity yet — a login
// attempt, mostly — falls back to the address. The prefixes keep the two
// namespaces from ever colliding.
func (s *Server) rateLimitKey(r *http.Request) string {
	if sub := SubjectFrom(r.Context()); !sub.Anonymous() {
		return "user:" + sub.UserID
	}
	return "addr:" + ClientIP(r)
}

// RateLimit refuses a caller who has run out of tokens with 429 and a
// Retry-After header, and never calls next for them.
func (s *Server) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok, retryAfter := s.limiter.Allow(s.rateLimitKey(r)); !ok {
			w.Header().Set("Retry-After", RetryAfterSeconds(retryAfter))
			writeError(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		next.ServeHTTP(w, r)
	})
}
