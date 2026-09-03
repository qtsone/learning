// Package api is the HTTP edge of the notes service: routing, decoding,
// encoding, and the single place where domain errors become status codes.
package api

import (
	"net/http"

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
	// TODO: register the five endpoints with method+wildcard patterns:
	//   POST   /notes        -> create
	//   GET    /notes        -> list
	//   GET    /notes/{id}   -> get one
	//   PUT    /notes/{id}   -> full replace
	//   DELETE /notes/{id}   -> delete
	return mux
}

// TODO: write the five handlers. Keep each thin — decode, delegate to
// s.svc, encode. A handler past ~10 lines is doing a lower layer's job.
//
// You will want these helpers:
//
//   respond(w, status, data)  write the {"data": ...} success envelope with
//                             Content-Type: application/json
//   respondError(w, err)      THE one place errors become HTTP:
//                               note.ValidationError -> 400, message
//                                 "validation failed", plus a "fields" map
//                               note.ErrNotFound     -> 404 "note not found"
//                               badRequest           -> 400, its message
//                               anything else        -> 500 "internal error"
//                                 — slog the real error; never send it
//   decodeDraft(r)            JSON-decode the body into a note.Draft;
//                             failure is badRequest{"invalid JSON"}
//   pathID(r)                 parse {id} via r.PathValue("id"); a
//                             non-integer is badRequest{"invalid id"}

// badRequest marks a request the HTTP layer itself rejects (malformed JSON,
// non-integer id) before the domain ever sees it.
type badRequest struct{ msg string }

func (b badRequest) Error() string { return b.msg }
