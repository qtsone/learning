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
// The order of the chain is the argument:
//
//   - SecurityHeaders outermost, so every response carries them — including
//     the 401s, 403s and 429s that no handler ever produced.
//   - Authenticate next, and it refuses nobody. It exists to answer "who is
//     this?", which the two layers below both need.
//   - RateLimit inside authentication, because a limiter keyed on an account
//     is worth far more than one keyed on an address, and it cannot key on an
//     account before somebody has established which account this is. The price
//     is a session lookup for a request that ends in a 429 — a map read, next
//     to a body read and a JSON decode it still refuses to do.
//   - The route gate innermost, per route, so a denial costs no handler.
//
// Login and logout mint and destroy identity, so they cannot sit behind a gate
// that requires one — but they still get the headers and the limiter, and an
// unauthenticated caller hammering a bcrypt verification is exactly who that
// limiter is for.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /login", s.HandleLogin)
	mux.HandleFunc("POST /logout", s.HandleLogout)
	for _, rt := range s.routes() {
		mux.Handle(rt.pattern, s.Require(rt.action)(rt.handler))
	}
	return Chain(mux, SecurityHeaders, s.Authenticate, s.RateLimit)
}
