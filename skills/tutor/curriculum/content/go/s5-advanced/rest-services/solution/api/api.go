// Package api is the HTTP edge of the notes service: routing, decoding,
// encoding, and the single place where domain errors become status codes.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"tutor.local/rest-services/note"
)

// Server holds the handlers' one dependency: the domain service.
type Server struct {
	svc *note.Service
}

func New(svc *note.Service) *Server {
	return &Server{svc: svc}
}

// Routes builds the router. Method+pattern registration (Go 1.22 mux) gives
// you 404 for unknown paths and 405 for wrong methods without writing a line.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notes", s.handleCreate)
	mux.HandleFunc("GET /notes", s.handleList)
	mux.HandleFunc("GET /notes/{id}", s.handleGet)
	mux.HandleFunc("PUT /notes/{id}", s.handleUpdate)
	mux.HandleFunc("DELETE /notes/{id}", s.handleDelete)
	return mux
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	d, err := decodeDraft(r)
	if err != nil {
		respondError(w, err)
		return
	}
	n, err := s.svc.Create(d)
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, http.StatusCreated, n)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	ns, err := s.svc.List()
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, http.StatusOK, ns)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	n, err := s.svc.Get(id)
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, http.StatusOK, n)
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	d, err := decodeDraft(r)
	if err != nil {
		respondError(w, err)
		return
	}
	n, err := s.svc.Update(id, d)
	if err != nil {
		respondError(w, err)
		return
	}
	respond(w, http.StatusOK, n)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		respondError(w, err)
		return
	}
	if err := s.svc.Delete(id); err != nil {
		respondError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeDraft(r *http.Request) (note.Draft, error) {
	var d note.Draft
	if err := json.NewDecoder(r.Body).Decode(&d); err != nil {
		return note.Draft{}, badRequest{msg: "invalid JSON"}
	}
	return d, nil
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0, badRequest{msg: "invalid id"}
	}
	return id, nil
}

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

func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(envelope{Data: data}); err != nil {
		// The status line is already on the wire; all that is left is to
		// tell an operator the body was truncated.
		slog.Error("encoding response", "err", err)
	}
}

// respondError is the only place in the program where an error meets a
// status code. Adding a domain error later is one new case here.
func respondError(w http.ResponseWriter, err error) {
	status, payload := http.StatusInternalServerError, errPayload{Message: "internal error"}

	var ve note.ValidationError
	var br badRequest
	switch {
	case errors.As(err, &ve):
		status, payload = http.StatusBadRequest, errPayload{Message: "validation failed", Fields: ve}
	case errors.Is(err, note.ErrNotFound):
		status, payload = http.StatusNotFound, errPayload{Message: "note not found"}
	case errors.As(err, &br):
		status, payload = http.StatusBadRequest, errPayload{Message: br.msg}
	default:
		// The client gets a generic message; the log gets the truth.
		slog.Error("request failed", "err", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errEnvelope{Error: payload}); err != nil {
		slog.Error("encoding error response", "err", err)
	}
}

// badRequest marks a request the HTTP layer itself rejects (malformed JSON,
// non-integer id) before the domain ever sees it.
type badRequest struct{ msg string }

func (b badRequest) Error() string { return b.msg }
