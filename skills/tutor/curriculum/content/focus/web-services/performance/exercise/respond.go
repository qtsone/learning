package apiperf

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

// Pagination limits. DefaultLimit is what a client that says nothing gets;
// MaxLimit is the ceiling, and it exists for the same reason the hardening
// lesson's body cap does — an unbounded `?limit=` is an unbounded amount of
// work a stranger can ask you to do.
const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// maxBodyBytes caps a JSON request body, as in the hardening lesson.
const maxBodyBytes int64 = 64 << 10

// requestError is a fault the caller can fix, carrying the status to send.
type requestError struct {
	Status  int
	Message string
}

func (e *requestError) Error() string { return e.Message }

func badRequest(msg string) error {
	return &requestError{Status: http.StatusBadRequest, Message: msg}
}

// writeJSON writes v as the whole response body.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError answers a *requestError with the status it carries; anything else
// is our bug and becomes a bare 500, with no internals in the body.
func writeError(w http.ResponseWriter, err error) {
	var re *requestError
	if errors.As(err, &re) {
		writeJSON(w, re.Status, map[string]string{"error": re.Message})
		return
	}
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
}

// decodeJSON reads one JSON value from the request into dst, capped and strict
// — the short version of what you built in the hardening lesson.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return &requestError{Status: http.StatusRequestEntityTooLarge, Message: "request body too large"}
		}
		return badRequest("malformed JSON body")
	}
	if dec.More() {
		return badRequest("body must hold a single JSON object")
	}
	return nil
}

// parseLimit reads the ?limit= parameter: empty means DefaultLimit, anything
// above MaxLimit is clamped down to it, and garbage is the caller's mistake.
//
// Clamping rather than rejecting a too-large limit is a deliberate choice:
// clients that ask for 1000 rows get a fast answer with a cursor instead of an
// error they have to handle. Rejecting is defensible too — what is not
// defensible is honouring it.
func parseLimit(raw string) (int, error) {
	if raw == "" {
		return DefaultLimit, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, badRequest("limit must be a positive integer")
	}
	if n > MaxLimit {
		n = MaxLimit
	}
	return n, nil
}
