package board

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// The envelope pair from the REST lesson: success under "data", failure under
// "error", so a client can branch before parsing.

// FieldError names one field a client got wrong.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// RequestError is a failure with a status already decided. Handlers return it
// upward instead of guessing a status at the top.
type RequestError struct {
	Status  int
	Message string
	Fields  []FieldError
}

// Error implements error.
func (e *RequestError) Error() string { return e.Message }

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("write response", "err", err)
	}
}

func writeData(w http.ResponseWriter, status int, v any) {
	writeJSON(w, status, map[string]any{"data": v})
}

// writeError sends a message the client can act on. What it never sends is why
// the *service* decided: a denial reason confirms that an object exists and
// belongs to somebody else, for free.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"message": msg}})
}

func writeRequestError(w http.ResponseWriter, err *RequestError) {
	body := map[string]any{"message": err.Message}
	if len(err.Fields) > 0 {
		body["fields"] = err.Fields
	}
	writeJSON(w, err.Status, map[string]any{"error": body})
}

// writeDenial is the one place a refusal becomes a status code. 401 means "we
// do not know who you are"; 403 means "we know exactly who you are, and no".
// A service that cannot tell those apart has already merged the two concerns.
func writeDenial(w http.ResponseWriter, sub Subject) {
	if sub.Anonymous() {
		writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	writeError(w, http.StatusForbidden, "forbidden")
}
