package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// The envelope pair from the REST-services lesson: success under "data",
// failure under "error", so a client can branch before parsing.

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

// writeError sends a generic message. The client gets a sentence it can act
// on; the log gets the truth. An auth endpoint is the last place to explain
// *why* a credential failed.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"message": msg}})
}
