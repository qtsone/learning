package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func (h *harness) metricsBody(t *testing.T) string {
	t.Helper()
	rec := h.serve(t, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /metrics = %d, want 200 (and no credentials required)", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want a text/plain exposition format", ct)
	}
	return rec.Body.String()
}

func mustContain(t *testing.T, body, line string) {
	t.Helper()
	if !strings.Contains(body, line) {
		t.Errorf("missing sample:\n\t%s\ngot:\n%s", line, body)
	}
}

func TestMetricsCountRequestsByMethodRouteAndStatus(t *testing.T) {
	h := newHarness(t)
	h.do(t, http.MethodGet, "/tasks", "")
	h.do(t, http.MethodGet, "/tasks", "")
	h.do(t, http.MethodGet, "/tasks/999", "")
	h.serve(t, httptest.NewRequest(http.MethodGet, "/tasks", nil)) // unauthenticated

	body := h.metricsBody(t)

	mustContain(t, body, "# TYPE http_requests_total counter")
	mustContain(t, body, `http_requests_total{method="GET",route="/tasks",status="200"} 2`)
	mustContain(t, body, `http_requests_total{method="GET",route="/tasks/{id}",status="404"} 1`)
	mustContain(t, body, `http_requests_total{method="GET",route="/tasks",status="401"} 1`)
	if strings.Contains(body, `route="/tasks/999"`) {
		t.Error("the route label is the raw path: one time series per id is the cardinality explosion the lesson warns about")
	}
}

// Health probes and the metrics scrape itself are not application traffic;
// counting them buries the signal you actually want.
func TestMetricsIgnoreOperationalEndpoints(t *testing.T) {
	h := newHarness(t)
	h.serve(t, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	h.serve(t, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	body := h.metricsBody(t)

	for _, route := range []string{"/healthz", "/readyz", "/metrics"} {
		if strings.Contains(body, `route="`+route+`"`) {
			t.Errorf("%s is instrumented; only the task routes are", route)
		}
	}
}

func TestMetricsRecordDurationsAndInFlight(t *testing.T) {
	h := newHarness(t)
	h.do(t, http.MethodGet, "/tasks", "")
	h.do(t, http.MethodGet, "/tasks", "")

	body := h.metricsBody(t)

	mustContain(t, body, "# TYPE http_request_duration_seconds histogram")
	mustContain(t, body, `http_request_duration_seconds_count{method="GET",route="/tasks"} 2`)
	// The +Inf bucket of a histogram always holds every observation.
	mustContain(t, body, `http_request_duration_seconds_bucket{method="GET",route="/tasks",le="+Inf"} 2`)
	mustContain(t, body, "# TYPE http_requests_in_flight gauge")
	mustContain(t, body, "http_requests_in_flight 0")

	// Buckets are cumulative: each bound holds everything at or below it.
	var previous int64
	for _, bound := range durationBuckets {
		prefix := `http_request_duration_seconds_bucket{method="GET",route="/tasks",le="` +
			strconv.FormatFloat(bound, 'g', -1, 64) + `"} `
		value, ok := sampleValue(body, prefix)
		if !ok {
			t.Fatalf("no bucket sample for le=%v:\n%s", bound, body)
		}
		if value < previous {
			t.Errorf("bucket le=%v holds %d, less than the bucket below it (%d): buckets are cumulative",
				bound, value, previous)
		}
		previous = value
	}
}

// The exposition must be byte-identical between scrapes that saw no traffic
// in between. Go randomises map iteration, so this fails unless you sort.
func TestMetricsRenderIsStable(t *testing.T) {
	h := newHarness(t)
	h.do(t, http.MethodGet, "/tasks", "")
	h.do(t, http.MethodPost, "/tasks", `{"title":"one"}`)
	h.do(t, http.MethodGet, "/tasks/999", "")

	first := h.metricsBody(t)
	second := h.metricsBody(t)

	if first != second {
		t.Errorf("two scrapes disagree:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

func TestMetricsAreSafeUnderConcurrentRequests(t *testing.T) {
	h := newHarness(t)
	const requests = 32

	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
			req.Header.Set("Authorization", "Bearer "+testToken)
			rec := httptest.NewRecorder()
			h.handler.ServeHTTP(rec, req)
		}()
	}
	wg.Wait()

	body := h.metricsBody(t)
	mustContain(t, body, `http_requests_total{method="GET",route="/tasks",status="200"} `+strconv.Itoa(requests))
	mustContain(t, body, "http_requests_in_flight 0")
}

// sampleValue finds the integer value of the sample line starting with prefix.
func sampleValue(body, prefix string) (int64, bool) {
	for _, line := range strings.Split(body, "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		v, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, prefix)), 10, 64)
		if err != nil {
			return 0, false
		}
		return v, true
	}
	return 0, false
}
