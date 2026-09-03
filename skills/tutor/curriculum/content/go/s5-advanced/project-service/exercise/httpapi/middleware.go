package httpapi

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// Middleware wraps a handler with behavior that runs around it.
type Middleware func(http.Handler) http.Handler

const requestIDHeader = "X-Request-ID"

// timeoutBody is what a client gets when Timeout fires. It is the same
// envelope shape as every other error, so a client only has to know one.
const timeoutBody = `{"error":{"message":"request timeout"}}`

type (
	requestIDKey struct{}
	clientKey    struct{}
)

// WithRequestID returns a copy of ctx carrying id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDFrom returns the id RequestID put on ctx, or "".
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// WithClient returns a copy of ctx carrying the authenticated client's name.
func WithClient(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, clientKey{}, name)
}

// ClientFrom returns the client name Auth put on ctx, or "" when the request
// was never authenticated.
func ClientFrom(ctx context.Context) string {
	name, _ := ctx.Value(clientKey{}).(string)
	return name
}

// Chain applies middleware so the FIRST one listed is outermost:
// Chain(h, A, B) serves requests as A(B(h)).
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// RequestID reuses an incoming X-Request-ID header or generates one, puts it
// on the request context, and echoes it on the response so a caller can
// quote it in a bug report.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(requestIDHeader)
		if id == "" {
			id = strconv.FormatUint(rand.Uint64(), 16)
		}
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(WithRequestID(r.Context(), id)))
	})
}

// statusWriter records the status a handler wrote so middleware can read it
// afterwards. The zero value of status is set by callers to 200, because a
// handler that never calls WriteHeader still produced a 200.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.NewResponseController reach the real ResponseWriter, so
// handlers keep access to flushing and hijacking through this wrapper.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// AccessLog returns a middleware that emits exactly one structured line per
// request through logger.
func AccessLog(logger *slog.Logger) Middleware {
	// TODO: wrap the writer so you can report the status, time the call,
	// and log method, path, status, duration_ms, and request_id. The
	// acceptance criteria fix the attribute names — an operator's queries
	// break when they drift.
	return func(next http.Handler) http.Handler { return next }
}

// Recover returns a middleware that turns a handler panic into a 500
// response instead of a killed connection.
func Recover(logger *slog.Logger) Middleware {
	// TODO: recover, log the panic with the request id, and answer with the
	// standard error envelope (writeError is in api.go). The panic must not
	// escape.
	return func(next http.Handler) http.Handler { return next }
}

// Timeout returns a middleware that bounds how long a handler may take.
func Timeout(d time.Duration) Middleware {
	// TODO: net/http already ships this one. Use it, with timeoutBody as
	// the message, and be ready to explain in review what it does to the
	// handler's goroutine and its context.
	return func(next http.Handler) http.Handler { return next }
}

// Auth returns a middleware that accepts only requests carrying a bearer
// token from tokens, a map of client name to secret token. The authenticated
// client's name goes on the request context.
func Auth(tokens map[string]string) Middleware {
	// TODO: precompute a digest per configured token at construction time,
	// compare digests in constant time (crypto/subtle), and reject anything
	// else with 401, a WWW-Authenticate header, and the standard envelope.
	// Never log or echo the presented token.
	return func(next http.Handler) http.Handler { return next }
}
