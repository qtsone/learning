package main

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestRegistryReturnsTheSameMetricForTheSameLabels(t *testing.T) {
	reg := NewRegistry()
	a := reg.Counter("app_requests_total", Labels{"route": "/items"})
	b := reg.Counter("app_requests_total", Labels{"route": "/items"})
	if a != b {
		t.Fatal("two lookups with the same name and labels returned different counters — a handler that looks up per request would count into a fresh counter every time")
	}
	c := reg.Counter("app_requests_total", Labels{"route": "/other"})
	if a == c {
		t.Fatal("different label sets returned the same counter — each label combination is its own series")
	}
	a.Inc()
	a.Add(2)
	if got := a.Value(); got != 3 {
		t.Errorf("counter value = %v, want 3", got)
	}
	if got := c.Value(); got != 0 {
		t.Errorf("the other series moved to %v, want 0", got)
	}
}

func TestCounterRejectsNegativeDelta(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Counter.Add(-1) returned normally — a counter that can go down silently corrupts every rate() built on it")
		}
	}()
	NewRegistry().Counter("app_requests_total", nil).Add(-1)
}

func TestGaugeGoesUpAndDown(t *testing.T) {
	g := NewRegistry().Gauge("app_in_flight", nil)
	g.Add(1)
	g.Add(1)
	g.Add(-1)
	if got := g.Value(); got != 1 {
		t.Errorf("gauge value = %v, want 1", got)
	}
	g.Set(7)
	if got := g.Value(); got != 7 {
		t.Errorf("after Set(7) gauge value = %v, want 7", got)
	}
}

func TestRegistryExposition(t *testing.T) {
	reg := NewRegistry()
	reg.Counter("app_requests_total", Labels{"method": "GET", "status": "200"}).Inc()
	reg.Counter("app_requests_total", Labels{"method": "GET", "status": "200"}).Add(2)
	reg.Counter("app_requests_total", Labels{"method": "POST", "status": "500"}).Inc()
	reg.Gauge("app_in_flight", nil).Set(3)
	reg.Gauge("app_in_flight", nil).Add(-1)

	h := reg.Histogram("app_latency_seconds", []float64{0.5, 1}, Labels{"route": "/items"})
	h.Observe(0.25)
	h.Observe(0.75)
	h.Observe(2)

	want := `# TYPE app_in_flight gauge
app_in_flight 2
# TYPE app_latency_seconds histogram
app_latency_seconds_bucket{le="0.5",route="/items"} 1
app_latency_seconds_bucket{le="1",route="/items"} 2
app_latency_seconds_bucket{le="+Inf",route="/items"} 3
app_latency_seconds_sum{route="/items"} 3
app_latency_seconds_count{route="/items"} 3
# TYPE app_requests_total counter
app_requests_total{method="GET",status="200"} 3
app_requests_total{method="POST",status="500"} 1
`

	var buf bytes.Buffer
	if err := reg.Write(&buf); err != nil {
		t.Fatalf("Write returned %v", err)
	}
	if got := buf.String(); got != want {
		t.Errorf("exposition output\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// Label values are attacker- and accident-supplied often enough that the
// exposition format defines escapes for them. Getting this wrong produces a
// file no scraper can parse — and, because the rendered labels double as the
// registry's key, silently merges two unrelated series into one.
func TestExpositionEscapesLabelValues(t *testing.T) {
	reg := NewRegistry()
	reg.Counter("app_errors_total", Labels{"msg": `he said "hi"`}).Inc()
	reg.Counter("app_errors_total", Labels{"msg": "line\nbreak"}).Inc()
	reg.Counter("app_errors_total", Labels{"msg": `back\slash`}).Inc()
	reg.Counter("app_errors_total", Labels{"msg": "héllo"}).Inc()

	want := `# TYPE app_errors_total counter
app_errors_total{msg="back\\slash"} 1
app_errors_total{msg="he said \"hi\""} 1
app_errors_total{msg="héllo"} 1
app_errors_total{msg="line\nbreak"} 1
`

	var buf bytes.Buffer
	if err := reg.Write(&buf); err != nil {
		t.Fatalf("Write returned %v", err)
	}
	if got := buf.String(); got != want {
		t.Errorf("escaped exposition output\n--- got ---\n%s\n--- want ---\n%s\n"+
			"(escape backslash, quote and newline; leave UTF-8 alone — this is not %%q)",
			got, want)
	}
}

func TestHistogramBucketsAreCumulative(t *testing.T) {
	reg := NewRegistry()
	h := reg.Histogram("app_latency_seconds", []float64{1, 0.25, 0.5}, nil)
	h.Observe(0.25)

	var buf bytes.Buffer
	if err := reg.Write(&buf); err != nil {
		t.Fatalf("Write returned %v", err)
	}
	want := []string{
		`app_latency_seconds_bucket{le="0.25"} 1`,
		`app_latency_seconds_bucket{le="0.5"} 1`,
		`app_latency_seconds_bucket{le="1"} 1`,
		`app_latency_seconds_bucket{le="+Inf"} 1`,
	}
	got := buf.String()
	for _, line := range want {
		if !strings.Contains(got, line) {
			t.Errorf("missing line %q — buckets are cumulative (an observation counts in every bucket it fits under) and bounds are sorted ascending. Got:\n%s", line, got)
		}
	}
}

func TestRegistryIsSafeUnderConcurrency(t *testing.T) {
	reg := NewRegistry()
	const goroutines, each = 8, 200

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < each; j++ {
				reg.Counter("app_requests_total", Labels{"route": "/items"}).Inc()
				reg.Gauge("app_in_flight", nil).Add(1)
				reg.Histogram("app_latency_seconds", DefaultBuckets, nil).Observe(0.01)
				reg.Gauge("app_in_flight", nil).Add(-1)
			}
		}()
	}
	// Scraping runs concurrently with the handlers in a real service, so the
	// test scrapes while the writers are hot.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < each; j++ {
			if err := reg.Write(io.Discard); err != nil {
				t.Errorf("Write returned %v", err)
				return
			}
		}
	}()
	wg.Wait()

	if got := reg.Counter("app_requests_total", Labels{"route": "/items"}).Value(); got != goroutines*each {
		t.Errorf("counter value = %v, want %d — lost increments mean the counter is not goroutine-safe", got, goroutines*each)
	}
	if got := reg.Gauge("app_in_flight", nil).Value(); got != 0 {
		t.Errorf("gauge value = %v, want 0", got)
	}
}
