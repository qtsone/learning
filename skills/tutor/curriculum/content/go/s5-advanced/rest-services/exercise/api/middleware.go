package api

import (
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder remembers the status a handler wrote so the access log can
// report it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Unwrap keeps http.NewResponseController able to reach the real writer, so
// wrapping does not hide Flusher/Hijacker from handlers.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// Logging writes one structured access-log line per request. main wires it
// around Routes; tests hit Routes directly so their output stays quiet.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(rec, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start),
		)
	})
}
