package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Tracing starts a server span for every request, continuing the caller's
// trace when the request carries a valid traceparent header.
func Tracing(tracer *Tracer) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if remote, ok := ParseTraceparent(r.Header.Get(traceparentHeader)); ok {
				ctx = ContextWithSpanContext(ctx, remote)
			}
			ctx, span := tracer.Start(ctx, r.Method+" "+r.URL.Path)
			defer func() {
				// The route is only known once the mux has matched, which
				// happens inside next — so rename on the way out.
				if route := RouteFrom(ctx); route != "" {
					span.SetName(r.Method + " " + route)
				}
				span.End()
			}()
			if sc, ok := SpanContextFrom(ctx); ok {
				w.Header().Set(traceIDHeader, sc.TraceID)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Observe records the RED metrics for every request — rate, errors,
// duration — plus the in-flight gauge, and emits exactly one access log line
// through logger. It sits outside Recover so a panicking handler is still
// counted and logged as a 500.
func Observe(reg *Registry, logger *slog.Logger) Middleware {
	inFlight := reg.Gauge("http_requests_in_flight", nil)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			inFlight.Add(1)
			defer inFlight.Add(-1)

			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)

			elapsed := time.Since(start)
			route := RouteFrom(r.Context())
			if route == "" {
				route = "unmatched"
			}
			reg.Counter("http_requests_total", Labels{
				"method": r.Method,
				"route":  route,
				"status": strconv.Itoa(sw.status),
			}).Inc()
			reg.Histogram("http_request_duration_seconds", DefaultBuckets, Labels{
				"route": route,
			}).Observe(elapsed.Seconds())

			logger.LogAttrs(r.Context(), slog.LevelInfo, "request",
				slog.String("method", r.Method),
				slog.String("route", route),
				slog.Int("status", sw.status),
				slog.Duration("duration", elapsed),
			)
		})
	}
}

// HealthHandler answers the liveness question: is this process running at
// all? It must never consult a dependency — a health check that fails
// because the database is down gets the process killed and restarted, which
// helps nobody.
func HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		io.WriteString(w, "ok")
	})
}

// Check reports whether one dependency is usable right now.
type Check func(ctx context.Context) error

// Readiness answers the traffic question: should this instance receive
// requests? Unlike liveness it does consult dependencies.
type Readiness struct {
	mu     sync.Mutex
	names  []string
	checks map[string]Check
}

// NewReadiness returns a Readiness with no checks registered, which reports
// ready.
func NewReadiness() *Readiness {
	return &Readiness{checks: make(map[string]Check)}
}

// Register adds or replaces a named check.
func (rd *Readiness) Register(name string, c Check) {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	if _, ok := rd.checks[name]; !ok {
		rd.names = append(rd.names, name)
	}
	rd.checks[name] = c
}

// Handler runs every registered check against the request context and
// answers 200 when all pass, 503 when any fails. The body names the failing
// check: a probe that says only "unready" turns every rollout into a
// guessing game.
func (rd *Readiness) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rd.mu.Lock()
		names := append([]string(nil), rd.names...)
		checks := make([]Check, len(names))
		for i, name := range names {
			checks[i] = rd.checks[name]
		}
		rd.mu.Unlock()

		body := struct {
			Status string            `json:"status"`
			Checks map[string]string `json:"checks"`
		}{Status: "ok", Checks: make(map[string]string, len(names))}

		status := http.StatusOK
		for i, name := range names {
			if err := checks[i](r.Context()); err != nil {
				body.Checks[name] = err.Error()
				body.Status = "unready"
				status = http.StatusServiceUnavailable
				continue
			}
			body.Checks[name] = "ok"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	})
}

// MetricsHandler exposes the registry for a scraper to pull.
func MetricsHandler(reg *Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_ = reg.Write(w)
	})
}
