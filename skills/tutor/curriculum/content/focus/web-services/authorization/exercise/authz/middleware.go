package authz

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// Require returns middleware that gates a route on action. It is the coarse
// half of enforcement: it runs before the handler, so a denied request never
// reaches handler code at all.
//
// It must: read the subject from the request context, ask CheckRoute, audit
// the decision, and on a denial write the standard denial response and stop.
// On an allow it calls next.
func (p *Policy) Require(action Action) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// TODO: replace this pass-through with the gate described above.
		// As shipped, every route is open to everyone — including anonymous
		// callers.
		return next
	}
}

// Enforce is the object-level half: call it from a handler once the resource
// is loaded, and let the return value decide whether to continue.
//
//	d, err := s.store.Get(id)
//	...
//	if !s.policy.Enforce(w, r, ActionRead, Resource{ID: d.ID, OwnerID: d.OwnerID}) {
//		return
//	}
//
// It must ask Check with the resource in hand, audit the decision, write the
// denial response when it denies, and report whether the caller may proceed.
func (p *Policy) Enforce(w http.ResponseWriter, r *http.Request, action Action, res Resource) bool {
	// TODO: implement. The starter waves everything through, which is the
	// IDOR bug the API tests are hunting.
	return true
}

// audit writes one structured line per decision — allow and deny alike.
// Provided: the point is that the decision function stays pure and every
// enforcement point logs the same shape, so "who was denied what, and why" is
// a query rather than an archaeology project.
func (p *Policy) audit(r *http.Request, sub Subject, action Action, res *Resource, d Decision) {
	level := slog.LevelInfo
	if !d.Allow {
		level = slog.LevelWarn
	}
	attrs := []any{
		"user", sub.UserID,
		"role", string(sub.Role),
		"action", string(action),
		"allow", d.Allow,
		"reason", d.Reason,
		"method", r.Method,
		"path", r.URL.Path,
	}
	if res != nil {
		attrs = append(attrs, "resource", res.ID, "owner", res.OwnerID)
	}
	p.log.Log(r.Context(), level, "authorization", attrs...)
}

// writeDenial turns a denial into a response. Provided, and worth reading
// twice:
//
//   - 401 when nobody is authenticated ("I do not know who you are"), 403 when
//     someone is but may not do this ("I know who you are; no").
//   - The body never carries d.Reason. The reason is for your log; telling a
//     caller "not owner" confirms the object exists and belongs to someone
//     else, which is free reconnaissance.
func writeDenial(w http.ResponseWriter, d Decision) {
	status, msg := http.StatusForbidden, "forbidden"
	if d.Reason == ReasonAnonymous {
		status, msg = http.StatusUnauthorized, "unauthenticated"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"message": msg},
	})
}
