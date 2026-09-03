package main

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
)

// Tracing starts a server span for every request, continuing the caller's
// trace when the request carries a valid traceparent header. It names the
// span "<METHOD> <path>", renames it to "<METHOD> <route>" once the router
// has revealed the route, echoes the trace id on the traceIDHeader, and hands
// the span-carrying context down the chain.
func Tracing(tracer *Tracer) Middleware {
	return func(next http.Handler) http.Handler {
		// TODO: parse the incoming traceparent, start the span, defer the
		// rename-and-End, set the response header, serve with the new
		// context.
		return next
	}
}

// Observe records the RED metrics for every request — rate, errors,
// duration — plus the in-flight gauge, and emits exactly one access log line
// through logger with the message "request".
//
// Metrics: http_requests_total{method,route,status} counted,
// http_request_duration_seconds{route} observed with DefaultBuckets, and
// http_requests_in_flight (no labels) held up for the duration of the
// request. Log attrs: method, route, status, duration.
//
// The route label comes from RouteFrom — never from r.URL.Path. Requests that
// matched no route are labelled "unmatched".
func Observe(reg *Registry, logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		// TODO: time the request, wrap the ResponseWriter in a statusWriter
		// to learn the status, then record metrics and log once. Log through
		// the *request* context so your ContextHandler can attach the ids.
		return next
	}
}

// HealthHandler answers the liveness question: is this process running at
// all? It replies 200 with the body "ok" and must never consult a dependency
// — a health check that fails because the database is down gets the process
// killed and restarted, which helps nobody.
func HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO
	})
}

// Check reports whether one dependency is usable right now.
type Check func(ctx context.Context) error

// Readiness answers the traffic question: should this instance receive
// requests? Unlike liveness it does consult dependencies.
type Readiness struct {
	mu sync.Mutex
	// TODO: the checks, and enough state to run them in a stable order.
}

// NewReadiness returns a Readiness with no checks registered, which reports
// ready.
func NewReadiness() *Readiness {
	// TODO
	return &Readiness{}
}

// Register adds or replaces a named check. It is safe to call while requests
// are in flight.
func (rd *Readiness) Register(name string, c Check) {
	// TODO
}

// Handler runs every registered check against the request context and answers
// 200 when all pass, 503 when any fails, with a JSON body of
// {"status": "ok"|"unready", "checks": {"<name>": "ok"|"<error text>"}} and
// Content-Type application/json. A probe that says only "unready" turns every
// rollout into a guessing game, so name the failing check.
func (rd *Readiness) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: snapshot the checks under the lock, run them outside it,
		// then answer.
	})
}

// MetricsHandler exposes the registry for a scraper to pull, with
// Content-Type "text/plain; version=0.0.4; charset=utf-8".
func MetricsHandler(reg *Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO
	})
}
