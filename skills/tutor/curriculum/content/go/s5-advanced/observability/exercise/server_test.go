package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// service is the whole observable stack under test: the mux from
// provided.go behind the middleware chain, with the logs captured in memory.
type service struct {
	handler http.Handler
	reg     *Registry
	tracer  *Tracer
	ready   *Readiness
	logs    *bytes.Buffer
}

func newService(t *testing.T) *service {
	t.Helper()
	logs := &bytes.Buffer{}
	logger := NewLogger(logs, slog.LevelInfo)

	// Recover logs through the default logger, so point it at the same
	// buffer for the duration of the test.
	previous := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(previous) })

	reg := NewRegistry()
	tracer := &Tracer{}
	ready := NewReadiness()
	handler := Chain(NewMux(reg, ready),
		RequestID,
		RouteContext,
		Tracing(tracer),
		Observe(reg, logger),
		Recover,
	)
	return &service{handler: handler, reg: reg, tracer: tracer, ready: ready, logs: logs}
}

func (s *service) get(target string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	return rec
}

// requestRecord returns the single access log line, ignoring any other
// records (a panic report, say).
func requestRecord(t *testing.T, s *service) map[string]any {
	t.Helper()
	var found []map[string]any
	for _, rec := range records(t, s.logs) {
		if rec["msg"] == "request" {
			found = append(found, rec)
		}
	}
	if len(found) != 1 {
		t.Fatalf("got %d log records with msg %q, want exactly one per request. Logs:\n%s",
			len(found), "request", s.logs.String())
	}
	return found[0]
}

func TestHealthzIgnoresDependencies(t *testing.T) {
	s := newService(t)
	s.ready.Register("db", func(ctx context.Context) error {
		return errors.New("connection refused")
	})

	rec := s.get("/healthz", nil)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /healthz = %d with a failing dependency, want 200 — liveness must not consult dependencies", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "ok" {
		t.Errorf("GET /healthz body = %q, want %q", got, "ok")
	}
}

func TestReadyzReflectsChecks(t *testing.T) {
	s := newService(t)

	decode := func(t *testing.T, rec *httptest.ResponseRecorder) struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	} {
		t.Helper()
		var body struct {
			Status string            `json:"status"`
			Checks map[string]string `json:"checks"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("GET /readyz body %q is not the documented JSON: %v", rec.Body.String(), err)
		}
		return body
	}

	t.Run("no checks means ready", func(t *testing.T) {
		rec := s.get("/readyz", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /readyz = %d with no checks, want 200", rec.Code)
		}
		if got := rec.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}
		if body := decode(t, rec); body.Status != "ok" {
			t.Errorf(`"status" = %q, want "ok"`, body.Status)
		}
	})

	s.ready.Register("cache", func(ctx context.Context) error { return nil })
	s.ready.Register("db", func(ctx context.Context) error { return errors.New("connection refused") })

	t.Run("one failing check makes the instance unready", func(t *testing.T) {
		rec := s.get("/readyz", nil)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("GET /readyz = %d with a failing check, want 503", rec.Code)
		}
		body := decode(t, rec)
		if body.Status != "unready" {
			t.Errorf(`"status" = %q, want "unready"`, body.Status)
		}
		if body.Checks["db"] != "connection refused" {
			t.Errorf(`checks["db"] = %q, want the error text %q — name the culprit`, body.Checks["db"], "connection refused")
		}
		if body.Checks["cache"] != "ok" {
			t.Errorf(`checks["cache"] = %q, want "ok"`, body.Checks["cache"])
		}
	})
}

func TestMetricsAreLabelledByRouteNotPath(t *testing.T) {
	s := newService(t)
	s.get("/items/1", nil)
	s.get("/items/2", nil)

	rec := s.get("/metrics", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /metrics = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/plain; version=0.0.4; charset=utf-8" {
		t.Errorf("Content-Type = %q, want %q", got, "text/plain; version=0.0.4; charset=utf-8")
	}

	body := rec.Body.String()
	for _, want := range []string{
		`http_requests_total{method="GET",route="/items/{id}",status="200"} 2`,
		`http_request_duration_seconds_bucket{le="+Inf",route="/items/{id}"} 2`,
		`http_request_duration_seconds_count{route="/items/{id}"} 2`,
		`# TYPE http_requests_in_flight gauge`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics output is missing %q. Got:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{`route="/items/1"`, `route="/items/2"`} {
		if strings.Contains(body, unwanted) {
			t.Errorf("metrics output contains %s — labelling by raw path gives one series per id, which is how monitoring systems fall over. Use the route.", unwanted)
		}
	}
	if got := s.reg.Gauge("http_requests_in_flight", nil).Value(); got != 0 {
		t.Errorf("http_requests_in_flight = %v after every request finished, want 0 — decrement it even when the handler returns early", got)
	}
}

func TestUnmatchedRequestsAreLabelledUnmatched(t *testing.T) {
	s := newService(t)
	rec := s.get("/nope", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /nope = %d, want 404", rec.Code)
	}

	if logged := requestRecord(t, s); logged["route"] != "unmatched" || logged["status"] != float64(404) {
		t.Errorf(`access log route/status = %v/%v, want "unmatched"/404`, logged["route"], logged["status"])
	}

	body := s.get("/metrics", nil).Body.String()
	want := `http_requests_total{method="GET",route="unmatched",status="404"} 1`
	if !strings.Contains(body, want) {
		t.Errorf("metrics output is missing %q — a request that matched no route still needs a bounded label. Got:\n%s", want, body)
	}
}

func TestPanicIsCountedAndLogged(t *testing.T) {
	s := newService(t)
	rec := s.get("/boom", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("GET /boom = %d, want 500 — Recover sits inside Observe", rec.Code)
	}

	logged := requestRecord(t, s)
	if logged["status"] != float64(500) {
		t.Errorf(`access log "status" = %v, want 500`, logged["status"])
	}
	if logged["route"] != "/boom" {
		t.Errorf(`access log "route" = %v, want "/boom"`, logged["route"])
	}

	body := s.get("/metrics", nil).Body.String()
	want := `http_requests_total{method="GET",route="/boom",status="500"} 1`
	if !strings.Contains(body, want) {
		t.Errorf("metrics output is missing %q — the request the users noticed is the one that must be counted. Got:\n%s", want, body)
	}
	if got := s.reg.Gauge("http_requests_in_flight", nil).Value(); got != 0 {
		t.Errorf("http_requests_in_flight = %v after a panicking request, want 0", got)
	}
}

func TestSignalsCorrelate(t *testing.T) {
	s := newService(t)
	incoming := "00-" + sampleTraceID + "-" + sampleSpanID + "-01"
	rec := s.get("/items/7", map[string]string{
		requestIDHeader:   "req-42",
		traceparentHeader: incoming,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /items/7 = %d, want 200", rec.Code)
	}

	if got := rec.Header().Get(requestIDHeader); got != "req-42" {
		t.Errorf("%s response header = %q, want %q", requestIDHeader, got, "req-42")
	}
	if got := rec.Header().Get(traceIDHeader); got != sampleTraceID {
		t.Errorf("%s response header = %q, want the caller's trace id %q", traceIDHeader, got, sampleTraceID)
	}

	finished := s.tracer.Finished()
	if len(finished) != 1 {
		t.Fatalf("the tracer recorded %d spans, want exactly one server span per request", len(finished))
	}
	span := finished[0]
	if span.TraceID != sampleTraceID {
		t.Errorf("span TraceID = %q, want the caller's %q — a valid traceparent means you join their trace", span.TraceID, sampleTraceID)
	}
	if span.ParentSpanID != sampleSpanID {
		t.Errorf("span ParentSpanID = %q, want the caller's span id %q", span.ParentSpanID, sampleSpanID)
	}
	if span.Name != "GET /items/{id}" {
		t.Errorf("span Name = %q, want %q — rename to the route once the mux has matched, or every id becomes its own operation", span.Name, "GET /items/{id}")
	}

	logged := requestRecord(t, s)
	for _, c := range []struct{ key, want string }{
		{"request_id", "req-42"},
		{"trace_id", sampleTraceID},
		{"span_id", span.SpanID},
		{"route", "/items/{id}"},
		{"method", "GET"},
	} {
		if logged[c.key] != c.want {
			t.Errorf("access log %q = %v, want %q — this is the join between the three signals", c.key, logged[c.key], c.want)
		}
	}
	if logged["status"] != float64(200) {
		t.Errorf(`access log "status" = %v, want 200`, logged["status"])
	}
	if _, ok := logged["duration"]; !ok {
		t.Error(`access log has no "duration" attr`)
	}
}

func TestTracingStartsAFreshTraceOnABrokenHeader(t *testing.T) {
	s := newService(t)
	s.get("/items/7", map[string]string{traceparentHeader: "not-a-traceparent"})

	finished := s.tracer.Finished()
	if len(finished) != 1 {
		t.Fatalf("the tracer recorded %d spans, want 1", len(finished))
	}
	if finished[0].ParentSpanID != "" {
		t.Errorf("ParentSpanID = %q, want empty — a malformed header is not a parent", finished[0].ParentSpanID)
	}
	if !isHexOfLen(finished[0].TraceID, 32) {
		t.Errorf("TraceID = %q, want a freshly generated 32-hex-digit id", finished[0].TraceID)
	}
}
