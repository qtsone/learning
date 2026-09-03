package main

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
)

// Middleware wraps a handler with extra behavior that runs around it.
type Middleware func(http.Handler) http.Handler

const requestIDHeader = "X-Request-ID"

// requestIDKey is the unexported context key for the per-request id — the
// same private-type trick you used in the S3 context lesson, so no other
// package can collide with (or read) this key by accident.
type requestIDKey struct{}

// WithRequestID returns a copy of ctx carrying id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDFrom returns the id RequestID put on ctx, or "" if there is none.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// Chain applies middleware so the FIRST one listed is the outermost layer:
// Chain(h, A, B) serves requests as A(B(h)).
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	// Backwards: the last middleware listed is applied first, so it ends up
	// innermost and the first one listed ends up outermost.
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// RequestID gives every request an id: it reuses the caller's X-Request-ID
// header when one is present and generates one otherwise, puts the id on the
// request context so inner handlers and middleware can read it with
// RequestIDFrom, and echoes it on the response so a caller can quote it in a
// bug report.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(WithRequestID(r.Context(), id)))
	})
}

// newRequestID returns a random correlation id. These only need to be
// distinct within a log window, not unguessable, so the (goroutine-safe)
// top-level math/rand/v2 source is the right tool.
func newRequestID() string {
	return strconv.FormatUint(rand.Uint64(), 16)
}

// Logging returns a middleware that logs one line per request through
// logger, with method, path, status, and request_id attributes. The
// request_id comes from the context, so it is only populated when RequestID
// sits OUTSIDE Logging in the chain.
func Logging(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			logger.LogAttrs(r.Context(), slog.LevelInfo, "request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sw.status),
				slog.String("request_id", RequestIDFrom(r.Context())),
			)
		})
	}
}

// Recover converts a handler panic into a 500 response. Without it, a panic
// kills the connection mid-response with nothing useful sent to the client.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				slog.ErrorContext(r.Context(), "handler panic",
					"panic", p,
					"path", r.URL.Path,
					"request_id", RequestIDFrom(r.Context()),
				)
				http.Error(w, http.StatusText(http.StatusInternalServerError),
					http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// statusWriter records the status code a handler writes so middleware can
// read it after the handler returns. It embeds http.ResponseWriter, so it
// satisfies the interface for free and only overrides what it cares about.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.NewResponseController reach the real ResponseWriter
// through this wrapper, so handlers keep access to flushing and hijacking.
func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
