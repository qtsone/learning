// Package api is the HTTP edge: routing, decoding, encoding — and the two
// places where the authorization policy is consulted.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"tutor.local/authorization/authn"
	"tutor.local/authorization/authz"
	"tutor.local/authorization/doc"
)

// Server holds two dependencies: where documents live, and who decides who may
// touch them.
type Server struct {
	store  doc.Store
	policy *authz.Policy
}

func New(store doc.Store, policy *authz.Policy) *Server {
	return &Server{store: store, policy: policy}
}

// route binds a pattern to the action the policy must approve before the
// handler runs. The struct is the trick: there is no way to register a route
// without naming an action, and a forgotten action is the zero Action — which
// has no rule, and therefore is denied rather than open.
type route struct {
	pattern string
	action  authz.Action
	handler http.HandlerFunc
}

func (s *Server) routes() []route {
	return []route{
		{"GET /docs", authz.ActionList, s.handleList},
		{"POST /docs", authz.ActionCreate, s.handleCreate},
		{"GET /docs/{id}", authz.ActionRead, s.handleGet},
		{"PUT /docs/{id}", authz.ActionUpdate, s.handleUpdate},
		{"DELETE /docs/{id}", authz.ActionDelete, s.handleDelete},
		// Someone shipped this route and never added a rule for it. That
		// happens; the question is what your service does about it.
		{"POST /docs/{id}/archive", authz.ActionArchive, s.handleArchive},
	}
}

// Handler builds the chain: authentication annotates the request, the 1.22 mux
// picks a route, and that route's policy gate runs before any handler of ours.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	for _, rt := range s.routes() {
		mux.Handle(rt.pattern, s.policy.Require(rt.action)(rt.handler))
	}
	return authn.Middleware(mux)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	all, err := s.store.List()
	if err != nil {
		respondErr(w, err)
		return
	}
	// The route gate approved "may list at all"; it said nothing about which
	// rows. Filtering is authorization too — one Check per row, same rule as a
	// single fetch, so the listing can never show what a GET would refuse.
	sub, _ := authz.SubjectFrom(r.Context())
	visible := make([]doc.Document, 0, len(all))
	for _, d := range all {
		if s.policy.Allows(sub, authz.ActionRead, authz.Resource{ID: d.ID, OwnerID: d.OwnerID}) {
			visible = append(visible, d)
		}
	}
	respond(w, http.StatusOK, visible)
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var in doc.Draft
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		respondErr(w, badRequest{"invalid JSON"})
		return
	}
	// Ownership is established from the authenticated subject. A client-supplied
	// owner would be a one-line privilege escalation, which is why doc.Draft
	// has no such field.
	sub, _ := authz.SubjectFrom(r.Context())
	created, err := s.store.Create(sub.UserID, in)
	if err != nil {
		respondErr(w, err)
		return
	}
	respond(w, http.StatusCreated, created)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	d, err := s.store.Get(r.PathValue("id"))
	if err != nil {
		respondErr(w, err)
		return
	}
	// The route gate only knew the role; the owner of THIS document is only
	// knowable now, after the load.
	if !s.policy.Enforce(w, r, authz.ActionRead, authz.Resource{ID: d.ID, OwnerID: d.OwnerID}) {
		return
	}
	respond(w, http.StatusOK, d)
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	cur, err := s.store.Get(r.PathValue("id"))
	if err != nil {
		respondErr(w, err)
		return
	}
	// Before decoding the body and long before writing anything.
	if !s.policy.Enforce(w, r, authz.ActionUpdate, authz.Resource{ID: cur.ID, OwnerID: cur.OwnerID}) {
		return
	}
	var in doc.Draft
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		respondErr(w, badRequest{"invalid JSON"})
		return
	}
	updated, err := s.store.Update(cur.ID, in)
	if err != nil {
		respondErr(w, err)
		return
	}
	respond(w, http.StatusOK, updated)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	cur, err := s.store.Get(r.PathValue("id"))
	if err != nil {
		respondErr(w, err)
		return
	}
	if !s.policy.Enforce(w, r, authz.ActionDelete, authz.Resource{ID: cur.ID, OwnerID: cur.OwnerID}) {
		return
	}
	if err := s.store.Delete(cur.ID); err != nil {
		respondErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleArchive is an ordinary, working handler sitting behind a route whose
// action has no rule. If it ever runs, deny-by-default has failed — and the
// test proves it by checking whether the document came back archived.
func (s *Server) handleArchive(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Archive(r.PathValue("id")); err != nil {
		respondErr(w, err)
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "archived"})
}

// badRequest marks a request the HTTP layer itself rejects, before the domain
// sees it.
type badRequest struct{ msg string }

func (b badRequest) Error() string { return b.msg }

func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
}

func respondErr(w http.ResponseWriter, err error) {
	status, msg := http.StatusInternalServerError, "internal error"
	var br badRequest
	switch {
	case errors.Is(err, doc.ErrNotFound):
		status, msg = http.StatusNotFound, "document not found"
	case errors.As(err, &br):
		status, msg = http.StatusBadRequest, br.msg
	default:
		slog.Error("request failed", "err", err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"message": msg},
	})
}
