// Package httpapi is the HTTP edge of the task service: routing, decoding,
// encoding, and the single place where a domain error becomes a status code.
// It is the only package that imports net/http.
package httpapi

import (
	"encoding/json"
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

// Routes builds the router. auth is applied to the task endpoints only —
// deciding which routes it wraps is part of the exercise.
func (s *Server) Routes(auth Middleware) http.Handler {
	// TODO: register the routes from LESSON.md's acceptance criteria with
	// 1.22 method+wildcard patterns, instrument the task routes through
	// s.metrics, and wrap those same routes — and only those — with auth.
	// The nesting is a decision, not a detail: instrumentation goes OUTSIDE
	// authentication, or a service being hammered with bad credentials looks
	// idle on the dashboard.
	return http.NewServeMux()
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
// status code.
func (s *Server) respondError(w http.ResponseWriter, r *http.Request, err error) {
	// TODO: map the domain's vocabulary onto HTTP — validation errors,
	// task.ErrNotFound, task.ErrAlreadyDone, badRequest — and default to a
	// 500 whose body says nothing while the log says everything.
	writeError(w, http.StatusInternalServerError, "internal error", nil)
}
