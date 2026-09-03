package board

import "net/http"

// route ties a URL pattern to the action it performs. Naming the action at the
// route is what makes forgetting a gate impossible: a route cannot be
// registered without one, and an action nobody wrote a rule for is denied.
type route struct {
	pattern string
	action  Action
	handler http.HandlerFunc
}

// routes is the service's authenticated surface. Note the last one: the delete
// route exists, names an action, and no rule grants it.
func (s *Server) routes() []route {
	return []route{
		{"GET /tasks", ActionTaskList, s.handleListTasks},
		{"POST /tasks", ActionTaskCreate, s.handleCreateTask},
		{"GET /tasks/{id}", ActionTaskRead, s.handleGetTask},
		{"PATCH /tasks/{id}", ActionTaskUpdate, s.handlePatchTask},
		{"DELETE /tasks/{id}", ActionTaskDelete, s.handleDeleteTask},
		{"POST /users/{id}/role", ActionRoleSet, s.handleSetRole},
		// Streaming is reading: whoever may list tasks may watch them change.
		{"GET /events", ActionTaskList, s.handleEvents},
	}
}

// Handler is the whole service as one http.Handler.
//
// As written it is the insecure version: routes are registered bare, nothing
// establishes who is calling, and nothing bounds what a caller may cost you.
//
// TODO:
//   - Gate every route in the table with s.Require(rt.action), so a denied
//     request never reaches handler code.
//   - Wrap the mux in the chain. Chain applies the first middleware listed as
//     the outermost, and the order is an argument you will have to make out
//     loud: which of SecurityHeaders, Authenticate and RateLimit sits where,
//     and why. Two constraints to reason from — every response, including the
//     ones no handler produced, must carry the security headers; and the
//     limiter can only key on an account if somebody has already established
//     which account this is.
//   - Login and logout mint and destroy identity, so they cannot sit behind a
//     gate that requires one. They still get everything else the chain does:
//     an unauthenticated caller is exactly who a rate limiter is for.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", s.HandleLogin)
	mux.HandleFunc("POST /logout", s.HandleLogout)
	for _, rt := range s.routes() {
		mux.Handle(rt.pattern, rt.handler)
	}
	return mux
}
