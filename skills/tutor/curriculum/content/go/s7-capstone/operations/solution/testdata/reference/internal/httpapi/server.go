// Package httpapi is the transport surface: routes, request logging, and the
// health, readiness and metrics endpoints that operations depends on.
package httpapi

import (
	"errors"
	"expvar"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"sync/atomic"
	"time"
)

// ErrInvalidID is returned for a work id that does not match idPattern.
var ErrInvalidID = errors.New("work id must be 1-64 characters of A-Z a-z 0-9 _ -")

// idPattern is deliberately strict: the id is echoed to the caller and would
// reach a log line or a file name the moment this service grows either.
var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ValidateWorkID checks the {id} path segment. It is the service's only trust
// boundary for request data, so it validates before anything uses the value —
// and its error never quotes the offending bytes, because that text is sent
// back to whoever supplied them.
func ValidateWorkID(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("%w: got %d byte(s)", ErrInvalidID, len(id))
	}
	return nil
}

// Counters published over /metrics. Names carry no per-request values in them:
// a counter per user id is a memory leak with a dashboard.
var (
	requestsTotal = expvar.NewInt("http_requests_total")
	errorsTotal   = expvar.NewInt("http_errors_total")
)

// Server owns the readiness flag. Liveness ("is this process broken?") and
// readiness ("can it serve right now?") are different questions, so they get
// different endpoints.
type Server struct {
	log   *slog.Logger
	ready atomic.Bool
}

// New returns a server that reports itself not-ready until Ready is called.
func New(log *slog.Logger) *Server {
	return &Server{log: log}
}

// Ready marks the service able to serve traffic.
func (s *Server) Ready() { s.ready.Store(true) }

// NotReady marks it unable to. It is the first step of a clean shutdown: the
// load balancer stops sending new work while in-flight requests drain.
func (s *Server) NotReady() { s.ready.Store(false) }

// Routes builds the handler. Health and readiness are deliberately outside the
// request-logging middleware: a probe every second would drown the log.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("GET /readyz", s.readyz)
	mux.Handle("GET /metrics", expvar.Handler())
	mux.Handle("GET /work/{id}", s.logRequests("/work/{id}", http.HandlerFunc(s.work)))
	return mux
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "ok")
}

func (s *Server) readyz(w http.ResponseWriter, _ *http.Request) {
	if !s.ready.Load() {
		http.Error(w, "draining", http.StatusServiceUnavailable)
		return
	}
	fmt.Fprintln(w, "ready")
}

// work is the one piece of real behaviour: it validates an id and echoes it.
func (s *Server) work(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := ValidateWorkID(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "done %s\n", id)
}

type recorder struct {
	http.ResponseWriter
	status int
}

func (rec *recorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

// logRequests emits one structured line per request. Fixed keys, so the line
// can be filtered and counted rather than read, and the route pattern rather
// than the request path, so the field stays low-cardinality.
func (s *Server) logRequests(route string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &recorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		requestsTotal.Add(1)
		if rec.status >= http.StatusInternalServerError {
			errorsTotal.Add(1)
		}
		s.log.InfoContext(r.Context(), "request",
			"method", r.Method,
			"route", route,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
