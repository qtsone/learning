// Your work in this file is the Registry: what it stores, how it hands back
// the same metric for the same name and labels, and Write. The three metric
// types below it are already written — read Histogram.Observe and the
// appendTo methods, which are the exposition format made concrete.
package main

import (
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Labels are the dimensions of a metric sample. Keep the value sets small
// and bounded: every distinct combination is a separate time series.
type Labels map[string]string

// DefaultBuckets are latency buckets in seconds, from "fast enough that
// nobody noticed" to "the client gave up".
var DefaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// metric is what the registry stores: something that can render its own
// samples under a family name.
type metric interface {
	appendTo(b []byte, name string) []byte
}

// Registry holds every metric in the process and renders them in the
// Prometheus text exposition format.
type Registry struct {
	mu sync.Mutex
	// TODO: what does a registry store? A metric is identified by its name
	// AND its label set, and two lookups with the same name and labels must
	// return the same object. The exposition format also needs each family's
	// type ("counter", "gauge", "histogram"). renderLabels below turns a
	// label set into a string, which makes a usable key.
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	// TODO: initialise whatever you added above.
	return &Registry{}
}

// Counter returns the counter registered under name and labels, creating it
// on first use. Repeated calls with the same name and labels return the same
// counter, which is what lets a hot handler do reg.Counter(...).Inc() on
// every request.
func (r *Registry) Counter(name string, labels Labels) *Counter {
	// TODO: look up or create. Registering the same name with a different
	// metric type is a programming error — panic rather than emit nonsense.
	// Store a copy of labels: the caller may reuse the map.
	return &Counter{}
}

// Gauge returns the gauge registered under name and labels.
func (r *Registry) Gauge(name string, labels Labels) *Gauge {
	// TODO
	return &Gauge{}
}

// Histogram returns the histogram registered under name and labels. buckets
// are upper bounds in the metric's own unit; they only matter the first time,
// when the histogram is created. A new Histogram needs its bounds sorted
// ascending and a counts slice of the same length.
func (r *Registry) Histogram(name string, buckets []float64, labels Labels) *Histogram {
	// TODO
	return &Histogram{}
}

// Write renders every metric in the Prometheus text exposition format:
//
//	# TYPE app_requests_total counter
//	app_requests_total{method="GET",status="200"} 3
//	# TYPE app_latency_seconds histogram
//	app_latency_seconds_bucket{le="0.5"} 1
//	app_latency_seconds_bucket{le="+Inf"} 3
//	app_latency_seconds_sum 3
//	app_latency_seconds_count 3
//
// Families are sorted by name and samples by their rendered label set, so the
// output is stable enough to diff and to test. Each metric renders its own
// samples: m.appendTo(b, name).
func (r *Registry) Write(w io.Writer) error {
	// TODO: snapshot what you need under the lock, then render. Rendering
	// while holding the registry lock would block every handler in the
	// process; iterating the maps without it is a data race.
	return nil
}

// Counter counts events that only ever accumulate: requests served, errors
// returned, bytes written.
type Counter struct {
	labels Labels

	mu    sync.Mutex
	value float64
}

// Inc adds one.
func (c *Counter) Inc() { c.Add(1) }

// Add increases the counter by delta. A negative delta is a bug in the
// caller, not a value to store: a decreasing counter breaks every rate()
// query downstream, so this panics instead of lying quietly.
func (c *Counter) Add(delta float64) {
	if delta < 0 {
		panic("metrics: Counter.Add called with a negative delta")
	}
	c.mu.Lock()
	c.value += delta
	c.mu.Unlock()
}

// Value returns the current total.
func (c *Counter) Value() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

func (c *Counter) appendTo(b []byte, name string) []byte {
	return appendSample(b, name, c.labels, c.Value())
}

// Gauge measures something that goes up and down: requests in flight, queue
// depth, connections open.
type Gauge struct {
	labels Labels

	mu    sync.Mutex
	value float64
}

// Set replaces the current value.
func (g *Gauge) Set(v float64) {
	g.mu.Lock()
	g.value = v
	g.mu.Unlock()
}

// Add moves the value by delta, which may be negative.
func (g *Gauge) Add(delta float64) {
	g.mu.Lock()
	g.value += delta
	g.mu.Unlock()
}

// Value returns the current value.
func (g *Gauge) Value() float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.value
}

func (g *Gauge) appendTo(b []byte, name string) []byte {
	return appendSample(b, name, g.labels, g.Value())
}

// Histogram counts observations into cumulative buckets, plus a running sum
// and count. It stores no individual observations, which is exactly why it is
// cheap enough to update on every request.
type Histogram struct {
	labels Labels
	bounds []float64

	mu     sync.Mutex
	counts []uint64
	sum    float64
	count  uint64
}

// Observe records one measurement. Buckets are cumulative: an observation
// counts in every bucket whose upper bound it fits under.
func (h *Histogram) Observe(v float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sum += v
	h.count++
	for i, upper := range h.bounds {
		if v <= upper {
			h.counts[i]++
		}
	}
}

func (h *Histogram) appendTo(b []byte, name string) []byte {
	h.mu.Lock()
	counts := append([]uint64(nil), h.counts...)
	sum, total := h.sum, h.count
	h.mu.Unlock()

	for i, upper := range h.bounds {
		b = appendSample(b, name+"_bucket",
			withLabel(h.labels, "le", formatValue(upper)), float64(counts[i]))
	}
	b = appendSample(b, name+"_bucket", withLabel(h.labels, "le", "+Inf"), float64(total))
	b = appendSample(b, name+"_sum", h.labels, sum)
	return appendSample(b, name+"_count", h.labels, float64(total))
}

func appendSample(b []byte, name string, labels Labels, v float64) []byte {
	b = append(b, name...)
	b = append(b, renderLabels(labels)...)
	b = append(b, ' ')
	b = append(b, formatValue(v)...)
	return append(b, '\n')
}

// renderLabels formats a label set as {k="v",k2="v2"} with keys sorted, so
// the same label set always renders to the same string — it doubles as the
// registry's lookup key. Escaping is what makes that second job safe: without
// it a value containing a quote would render to the same string as a
// different label set, and two unrelated series would silently merge.
func renderLabels(labels Labels) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	sb.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(k)
		sb.WriteString(`="`)
		sb.WriteString(escapeLabelValue(labels[k]))
		sb.WriteString(`"`)
	}
	sb.WriteByte('}')
	return sb.String()
}

// escapeLabelValue applies the three escapes the Prometheus text format
// defines for label values: backslash, double quote, and line feed. Note it
// is not Go's %q — the exposition format is UTF-8, so non-ASCII characters
// pass through unchanged rather than becoming \u escapes.
func escapeLabelValue(v string) string {
	if !strings.ContainsAny(v, "\\\"\n") {
		return v
	}
	var sb strings.Builder
	sb.Grow(len(v) + 4)
	for _, r := range v {
		switch r {
		case '\\':
			sb.WriteString(`\\`)
		case '"':
			sb.WriteString(`\"`)
		case '\n':
			sb.WriteString(`\n`)
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func formatValue(v float64) string {
	switch {
	case math.IsInf(v, 1):
		return "+Inf"
	case math.IsInf(v, -1):
		return "-Inf"
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}

func cloneLabels(labels Labels) Labels {
	if len(labels) == 0 {
		return nil
	}
	out := make(Labels, len(labels))
	for k, v := range labels {
		out[k] = v
	}
	return out
}

func withLabel(labels Labels, k, v string) Labels {
	out := make(Labels, len(labels)+1)
	for lk, lv := range labels {
		out[lk] = lv
	}
	out[k] = v
	return out
}
