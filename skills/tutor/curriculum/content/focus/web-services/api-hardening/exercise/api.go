package harden

import (
	"encoding/json"
	"net/http"
)

// CreateTaskHandler is the one business handler in this exercise. It exists to
// give the middleware stack something real to protect: notice how little it
// does, and that every rule about size, shape and content was enforced before
// its body ran.
func CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateTaskRequest
	if err := DecodeJSON(w, r, DefaultMaxBodyBytes, &req); err != nil {
		WriteRequestError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"title":    req.Title,
		"priority": req.Priority,
	})
}
