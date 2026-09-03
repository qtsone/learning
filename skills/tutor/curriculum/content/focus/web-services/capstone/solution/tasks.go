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
// The route gate said "you may list", which is not the same as "you may see
// these rows", so the policy narrows the query before it runs. Filtering rows
// after loading them would be correct here and wrong at ten thousand rows —
// and a filter can forget a row, where a WHERE clause cannot.
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	limit, err := pageLimit(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var cursor *Cursor
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		c, err := DecodeCursor(raw)
		if err != nil {
			// A cursor is opaque to the client, so there is nothing useful to
			// say beyond "this is not one of mine".
			writeError(w, http.StatusBadRequest, "malformed cursor")
			return
		}
		cursor = &c
	}

	scope := s.policy.ScopeFor(SubjectFrom(r.Context()))
	// One row more than we return: the extra row is how we learn there is a
	// next page without a second query or a COUNT.
	rows, err := s.store.ListTasks(r.Context(), scope, cursor, limit+1)
	if err != nil {
		s.log.Error("list tasks", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	page := TaskPage{Items: rows}
	if len(rows) > limit {
		page.Items = rows[:limit]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = EncodeCursor(Cursor{CreatedAt: last.CreatedAt, ID: last.ID})
	}
	if page.Items == nil {
		page.Items = []Task{}
	}

	header := w.Header()
	// This body depends on who asked, so no shared cache may keep it — and
	// Vary names the input it depends on, or a cache that stores it anyway
	// hands one user another user's tasks.
	header.Set("Cache-Control", "private, no-cache")
	header.Set("Vary", "Cookie")
	writeCachedJSON(w, r, http.StatusOK, page)
}

// handleGetTask serves GET /tasks/{id}.
//
// The route gate could only answer "may a member ever read a task?". The rest
// of the decision needs the object, so it happens here — before the task
// reaches the response, which is the difference between a service and an IDOR.
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
	if !s.Enforce(w, r, ActionTaskRead, Resource{ID: task.ID, OwnerID: task.OwnerID}) {
		return
	}
	writeData(w, http.StatusOK, task)
}

// handleCreateTask serves POST /tasks.
//
// Two rules carry this handler. The task and the work it triggers commit
// together, because there is no ordering of "write the row" and "publish the
// job" that survives a crash when they are two systems — so they are not two
// systems. And the announcement happens after the commit: a transaction can be
// rolled back, a line on somebody's screen cannot.
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
	// The job id names the work, not the delivery, so enqueueing the same work
	// twice is a primary-key conflict instead of a second notification.
	if err := s.store.Enqueue(r.Context(), tx, NewJob{
		ID:      "notify:" + task.ID,
		Kind:    JobKindNotify,
		Payload: task.ID,
	}); err != nil {
		s.log.Error("enqueue notify", "err", err, "task", task.ID)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := tx.Commit(); err != nil {
		s.log.Error("commit", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	s.hub.Publish(taskEvent(EventTaskCreated, task))
	writeData(w, http.StatusCreated, task)
}

// handlePatchTask serves PATCH /tasks/{id}, moving a task between states.
//
// Load, enforce, *then* decode. The other order works and reads more naturally,
// which is why it is the common mistake: it makes a request that is about to be
// refused pay for a 64 KiB read and a strict decode first. The refusal is
// supposed to cost a row lookup and a map read, and a caller who cannot touch
// this object should not be able to make the server parse anything.
func (s *Server) handlePatchTask(w http.ResponseWriter, r *http.Request) {
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
	if !s.Enforce(w, r, ActionTaskUpdate, Resource{ID: task.ID, OwnerID: task.OwnerID}) {
		return
	}

	var req UpdateTaskRequest
	if rerr := DecodeJSON(w, r, DefaultMaxBodyBytes, &req); rerr != nil {
		writeRequestError(w, rerr)
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
	s.hub.Publish(taskEvent(EventTaskUpdated, task))
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
// Changing a role changes the answer to every question the target's live
// sessions are about to ask, so those sessions do not survive it. Destroying
// them is the strongest version of the auth lesson's rotation rule: an id
// captured under one privilege is never an id under another. The cost is one
// re-login for a user whose permissions just changed, which is a good trade.
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

	revoked := s.sessions.DeleteUser(user.ID)
	s.log.Info("role changed",
		"actor", SubjectFrom(r.Context()).UserID, "user", user.ID, "role", string(user.Role), "sessions_revoked", revoked)

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
