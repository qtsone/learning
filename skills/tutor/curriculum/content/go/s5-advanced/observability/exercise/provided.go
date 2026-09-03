// This file is already yours. It is the middleware plumbing you built in the
// HTTP servers lesson plus this service's routing table. Read it — every
// piece of observability you are about to write leans on it — but you do not
// need to change it.
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
)

// Middleware wraps a handler with extra behavior that runs around it.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware so the FIRST one listed is the outermost layer.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

const requestIDHeader = "X-Request-ID"

type requestIDKey struct{}

// WithRequestID returns a copy of ctx carrying id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDFrom returns the id on ctx, or "" if there is none.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// RequestID reuses the caller's X-Request-ID or generates one, puts it on the
// context, and echoes it on the response.
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

// routeHolder is a mutable box the router fills in. The matched route is only
// known *after* the mux has matched, which is deeper in the chain than the
// middleware that needs it — so outer middleware plants an empty box on the
// context and reads it once the handler has returned.
type routeHolder struct{ pattern string }

type routeKey struct{}

// RouteContext plants the box. Put it outside every middleware that labels
// anything with the route.
func RouteContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), routeKey{}, &routeHolder{})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Route records pattern as the route this request matched. Every route in
// NewMux is registered through it.
func Route(pattern string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if holder, ok := r.Context().Value(routeKey{}).(*routeHolder); ok {
			holder.pattern = pattern
		}
		h.ServeHTTP(w, r)
	})
}

// RouteFrom returns the matched route pattern, or "" when nothing matched
// (or when RouteContext is missing from the chain).
func RouteFrom(ctx context.Context) string {
	holder, ok := ctx.Value(routeKey{}).(*routeHolder)
	if !ok {
		return ""
	}
	return holder.pattern
}

// Recover turns a handler panic into a 500. It logs through the default
// logger, which is why main calls slog.SetDefault: even code that never saw
// your logger then produces correlated records.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				slog.ErrorContext(r.Context(), "handler panic", "panic", p, "path", r.URL.Path)
				http.Error(w, http.StatusText(http.StatusInternalServerError),
					http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// statusWriter records the status a handler wrote so middleware can read it
// afterwards.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.NewResponseController reach the real ResponseWriter.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// NewMux wires the service's routes. Every handler is registered through
// Route, so metrics and spans can be labelled with the pattern instead of the
// raw path.
func NewMux(reg *Registry, ready *Readiness) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", Route("/healthz", HealthHandler()))
	mux.Handle("GET /readyz", Route("/readyz", ready.Handler()))
	mux.Handle("GET /metrics", Route("/metrics", MetricsHandler(reg)))
	mux.Handle("GET /items/{id}", Route("/items/{id}", itemHandler()))
	mux.Handle("GET /boom", Route("/boom", panicHandler()))
	return mux
}

// itemHandler stands in for the service's real work.
func itemHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"id": r.PathValue("id")})
	})
}

// panicHandler exists so you can watch a panic become a counted, logged 500.
func panicHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
}
