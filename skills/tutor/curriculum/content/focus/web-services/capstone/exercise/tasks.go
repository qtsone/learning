package board

import (
	"errors"
	"net/http"
	"strconv"
)

// TaskPage is one page of the feed. Items is never nil: a client that has to
// distinguish `null` from `[]` will eventually get it wrong.
type TaskPage struct {
	Items      []Task `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// handleListTasks serves GET /tasks?limit=&cursor=.
//
// It is written here the way it usually is first: correct, and blind. It
// returns every task in the database to anybody the route gate let through,
// pages nothing, and lets a caller ask for as many rows as they like.
//
// TODO, in this order:
//  1. Restrict the query to what the caller may read, with s.policy.ScopeFor.
//     The route gate said "you may list", which is not the same as "you may see
//     these rows" — and the restriction belongs in the query, not in a filter
//     over rows you already loaded.
//  2. Read ?limit= (default DefaultPageLimit, clamped to MaxPageLimit, 400 on
//     something that is not a number) and ?cursor= (400 on ErrBadCursor).
//  3. Set NextCursor only when a further page really exists — ask for one more
//     row than you return, and you know.
//  4. Make the response cacheable *safely*: an ETag, a Cache-Control that no
//     shared cache may act on, and a Vary that names what the body depends on.
//     Two users must never be served each other's page out of a cache.
//     writeCachedJSON does the ETag and the 304; the headers are your call.
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.store.ListTasks(r.Context(), Scope{All: true}, nil, MaxPageLimit)
	if err != nil {
		s.log.Error("list tasks", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeData(w, http.StatusOK, TaskPage{Items: tasks})
}

// handleGetTask serves GET /tasks/{id}.
//
// TODO: the route gate could only answer "may a member ever read a task?".
// This is where the object exists, so this is where the rest of the decision
// happens — before the task reaches the response. Load it (ErrNotFound → 404),
// then s.Enforce with the owner in hand.
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.TaskByID(r.Context(), r.PathValue("id"))
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		s.log.Error("load task", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeData(w, http.StatusOK, task)
}

// handleCreateTask serves POST /tasks.
//
// TODO, the two halves this lesson is about:
//
//   - The task and the work it triggers must land in **one** transaction.
//     Enqueue a NewJob of kind JobKindNotify, with an id that names the work
//     ("notify:"+task.ID) rather than the delivery, on the same tx as the
//     insert. Commit once. If either write fails, neither row may exist — an
//     order with no job and a job with no order are both bugs you cannot see
//     from inside the handler.
//   - Announce it *after* the commit, never before: publish
//     taskEvent(EventTaskCreated, task) to s.hub. An event about a row that
//     then rolled back cannot be recalled from a subscriber's screen.
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	sub := SubjectFrom(r.Context())

	var req CreateTaskRequest
	if rerr := DecodeJSON(w, r, DefaultMaxBodyBytes, &req); rerr != nil {
		writeRequestError(w, rerr)
		return
	}

	now := s.clock.Now()
	task := Task{
		ID:      s.newID(),
		OwnerID: sub.UserID, // from the session, never from the body
		Title:   req.Title,
		State:   StateOpen, CreatedAt: now, UpdatedAt: now,
	}

	tx, err := s.store.DB.BeginTx(r.Context(), nil)
	if err != nil {
		s.log.Error("begin", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer tx.Rollback()

	if err := s.store.InsertTask(r.Context(), tx, task); err != nil {
		s.log.Error("insert task", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := tx.Commit(); err != nil {
		s.log.Error("commit", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeData(w, http.StatusCreated, task)
}

// handlePatchTask serves PATCH /tasks/{id}, moving a task between states.
//
// TODO: enforce ownership on the loaded object the way handleGetTask must, and
// publish taskEvent(EventTaskUpdated, updated) after the commit — a denied
// request must leave storage byte-for-byte unchanged and put nothing on the
// wire. Note the order the body is decoded in below, and what a refusal
// currently costs: the refusal is supposed to be a row lookup and a map read,
// not that plus a 64 KiB read and a strict decode.
func (s *Server) handlePatchTask(w http.ResponseWriter, r *http.Request) {
	var req UpdateTaskRequest
	if rerr := DecodeJSON(w, r, DefaultMaxBodyBytes, &req); rerr != nil {
		writeRequestError(w, rerr)
		return
	}

	ctx := r.Context()
	task, err := s.store.TaskByID(ctx, r.PathValue("id"))
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	if err != nil {
		s.log.Error("load task", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	tx, err := s.store.DB.BeginTx(ctx, nil)
	if err != nil {
		s.log.Error("begin", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	defer tx.Rollback()

	now := s.clock.Now()
	if err := s.store.UpdateTaskState(ctx, tx, task.ID, req.State, now); err != nil {
		s.log.Error("update task", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := tx.Commit(); err != nil {
		s.log.Error("commit", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	task.State, task.UpdatedAt = req.State, now
	writeData(w, http.StatusOK, task)
}

// handleDeleteTask serves DELETE /tasks/{id} and really does delete the row.
// Nothing else stands between a caller and this code: the only thing that stops
// it is the missing rule, which is what deny-by-default is worth.
func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteTask(r.Context(), r.PathValue("id")); err != nil && !errors.Is(err, ErrNotFound) {
		s.log.Error("delete task", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleSetRole serves POST /users/{id}/role.
//
// TODO: the route gate has already established that the caller may change
// roles. Apply the change with s.store.SetRole (ErrNotFound → 404), then deal
// with the consequence: sessions minted under the old role are still out there,
// and they are now wrong. The auth lesson's rule was that a privilege change
// never leaves a pre-change session usable. Answer 200 with the updated user.
func (s *Server) handleSetRole(w http.ResponseWriter, r *http.Request) {
	var req SetRoleRequest
	if rerr := DecodeJSON(w, r, DefaultMaxBodyBytes, &req); rerr != nil {
		writeRequestError(w, rerr)
		return
	}
	user, err := s.store.SetRole(r.Context(), r.PathValue("id"), req.Role)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		s.log.Error("set role", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeData(w, http.StatusOK, user)
}

// pageLimit reads ?limit=, clamped to MaxPageLimit. A missing limit is the
// default; a limit that is not a number is a client error.
func pageLimit(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return DefaultPageLimit, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, errors.New("limit must be a positive integer")
	}
	if n > MaxPageLimit {
		n = MaxPageLimit
	}
	return n, nil
}
