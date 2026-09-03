package main

import (
	"context"
	"log/slog"
	"net/http"
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
	// TODO: wrap h so the first middleware listed runs first. Think about
	// which direction to loop over mws.
	return h
}

// RequestID gives every request an id: it reuses the caller's X-Request-ID
// header when one is present and generates one otherwise, puts the id on the
// request context so inner handlers and middleware can read it with
// RequestIDFrom, and echoes it on the response so a caller can quote it in a
// bug report.
func RequestID(next http.Handler) http.Handler {
	// TODO: implement. math/rand/v2 is fine for generating ids here.
	// You never mutate the request you were handed — hand next a copy made
	// with r.WithContext.
	return next
}

// Logging returns a middleware that logs one line per request through
// logger, with method, path, status, and request_id attributes. The
// request_id comes from the context, so it is only populated when RequestID
// sits OUTSIDE Logging in the chain.
func Logging(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		// TODO: wrap next; capture the status code with statusWriter; log
		// after the handler returns.
		return next
	}
}

// Recover converts a handler panic into a 500 response. Without it, a panic
// kills the connection mid-response with nothing useful sent to the client.
func Recover(next http.Handler) http.Handler {
	// TODO: defer a recover() around next.ServeHTTP; on panic, respond with
	// http.Error(..., http.StatusInternalServerError).
	return next
}

// statusWriter records the status code a handler writes so middleware can
// read it after the handler returns. It embeds http.ResponseWriter, so it
// satisfies the interface for free and only overrides what it cares about.
type statusWriter struct {
	http.ResponseWriter
	status int
}

// TODO: override WriteHeader so statusWriter remembers the code before
// delegating to the embedded ResponseWriter. Careful: a handler that never
// calls WriteHeader still responds 200 — make sure that is what gets logged.
