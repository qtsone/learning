// Package httpapi is the HTTP edge of the task service: routing, decoding,
// encoding, and the single place where a domain error becomes a status code.
// It is the only package that imports net/http.
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"tutor.local/project-service/task"
)

// Server holds the edge's dependencies: the domain service, the metrics
// registry it reports to, and a logger.
type Server struct {
	svc     *task.Service
	metrics *Metrics
	logger  *slog.Logger
}

// New wires the HTTP layer. A nil logger falls back to slog's default.
func New(svc *task.Service, metrics *Metrics, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{svc: svc, metrics: metrics, logger: logger}
}

// Routes builds the router. auth is applied to the task endpoints only.
func (s *Server) Routes(auth Middleware) http.Handler {
	mux := http.NewServeMux()

	// Operational endpoints. No credentials: a kubelet probe has none, and
	// a health check that can fail on authentication is a health check that
	// lies. No instrumentation either — probe and scrape traffic would bury
	// the application signal underneath it.
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("GET /readyz", s.handleReadyz)
	mux.Handle("GET /metrics", s.metrics)

	// Task endpoints. Instrumentation goes OUTSIDE authentication, so a
	// wave of 401s shows up in the metrics instead of vanishing.
	protect := func(method, route string, h http.HandlerFunc) {
		mux.Handle(method+" "+route, s.metrics.Instrument(method, route, auth(h)))
	}
	protect(http.MethodPost, "/tasks", s.handleCreate)
	protect(http.MethodGet, "/tasks", s.handleList)
	protect(http.MethodGet, "/tasks/{id}", s.handleGet)
	protect(http.MethodPatch, "/tasks/{id}", s.handleSetStatus)
	protect(http.MethodDelete, "/tasks/{id}", s.handleDelete)

	return mux
}

// Handlers do three things and stop: decode, delegate, encode.

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	var d task.Draft
	if err := decodeJSON(r, &d); err != nil {
		s.respondError(w, r, err)
		return
	}
	created, err := s.svc.Create(r.Context(), d)
	if err != nil {
		s.respondError(w, r, err)
		return
	}
	respond(w, http.StatusCreated, created)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.svc.List(r.Context(), task.Status(r.URL.Query().Get("status")))
	if err != nil {
		s.respondError(w, r, err)
		return
	}
	respond(w, http.StatusOK, tasks)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.respondError(w, r, err)
		return
	}
	got, err := s.svc.Get(r.Context(), id)
	if err != nil {
		s.respondError(w, r, err)
		return
	}
	respond(w, http.StatusOK, got)
}

func (s *Server) handleSetStatus(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.respondError(w, r, err)
		return
	}
	var patch task.StatusPatch
	if err := decodeJSON(r, &patch); err != nil {
		s.respondError(w, r, err)
		return
	}
	updated, err := s.svc.SetStatus(r.Context(), id, patch.Status)
	if err != nil {
		s.respondError(w, r, err)
		return
	}
	respond(w, http.StatusOK, updated)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.respondError(w, r, err)
		return
	}
	if err := s.svc.Delete(r.Context(), id); err != nil {
		s.respondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent) // 204 promises no body
}

// handleHealthz answers liveness: this process is running and its handlers
// respond. It touches no dependency on purpose — a liveness probe that fails
// when the database blinks gets the process killed for someone else's fault.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReadyz answers readiness: this process can serve traffic right now.
// That is a different question, and it has to ask the database.
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if err := s.svc.Ping(r.Context()); err != nil {
		s.logger.ErrorContext(r.Context(), "readiness check failed", "err", err)
		writeError(w, http.StatusServiceUnavailable, "not ready", nil)
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "ready"})
}

// pathID reads the {id} wildcard as an int64.
func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0, badRequest{msg: "invalid id"}
	}
	return id, nil
}

// decodeJSON reads a JSON request body into v, turning any decoding failure
// into a client error rather than a 500.
func decodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return badRequest{msg: "invalid JSON"}
	}
	return nil
}

// badRequest marks a request the HTTP layer itself rejects — malformed JSON,
// a non-integer id — before the domain ever sees it.
type badRequest struct{ msg string }

func (b badRequest) Error() string { return b.msg }

type envelope struct {
	Data any `json:"data"`
}

type errEnvelope struct {
	Error errPayload `json:"error"`
}

type errPayload struct {
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// respond writes a success envelope: {"data": …}.
func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(envelope{Data: data}); err != nil {
		// The status line is already on the wire; all that is left is to
		// tell an operator the body was truncated.
		slog.Error("encoding response", "err", err)
	}
}

// writeError writes an error envelope: {"error": {"message": …}}. Handlers
// and middleware both go through it, so the service has exactly one error
// shape on the wire.
func writeError(w http.ResponseWriter, status int, message string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errEnvelope{
		Error: errPayload{Message: message, Fields: fields},
	}); err != nil {
		slog.Error("encoding error response", "err", err)
	}
}

// respondError is the only place in the program where an error meets a
// status code. Adding a domain error later is one new case here, not a hunt
// through every handler.
func (s *Server) respondError(w http.ResponseWriter, r *http.Request, err error) {
	var ve task.ValidationError
	var br badRequest
	switch {
	case errors.As(err, &ve):
		writeError(w, http.StatusBadRequest, "validation failed", ve)
	case errors.Is(err, task.ErrNotFound):
		writeError(w, http.StatusNotFound, "task not found", nil)
	case errors.Is(err, task.ErrAlreadyDone):
		writeError(w, http.StatusConflict, "task is already done", nil)
	case errors.As(err, &br):
		writeError(w, http.StatusBadRequest, br.msg, nil)
	default:
		// The client gets a generic message; the log gets the truth.
		s.logger.ErrorContext(r.Context(), "request failed",
			"err", err,
			"method", r.Method,
			"path", r.URL.Path,
			"request_id", RequestIDFrom(r.Context()),
		)
		writeError(w, http.StatusInternalServerError, "internal error", nil)
	}
}
