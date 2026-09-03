package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// durationBuckets are the upper bounds, in seconds, of the request-duration
// histogram. A histogram does not remember individual durations: it counts
// how many observations fell at or below each bound, which is enough to
// estimate a quantile and cheap enough to keep for every route.
//
// These bounds are a choice, not a law, and NOTES.md asks you to defend them:
// buckets you cannot answer a question with are buckets you are paying to
// store.
var durationBuckets = []float64{0.005, 0.025, 0.1, 0.5, 1, 2.5}

// series identifies one method+route pair — one line on a dashboard.
type series struct {
	method string
	route  string
}

// statusSeries adds the response status, the third dimension of the request
// counter (the E of RED lives here: status 5xx over status all).
type statusSeries struct {
	series
	status int
}

// histogram holds cumulative bucket counts plus the sum and count that make
// an average possible.
type histogram struct {
	buckets []int64
	count   int64
	sum     float64
}

// Metrics is a hand-rolled, in-process metrics registry: request counts by
// method/route/status (the R and E of RED), a duration histogram (the D),
// and an in-flight gauge (the U of USE).
//
// This file is PROVIDED, complete. You wrote its twin — registry, cumulative
// histogram, text exposition — in the observability lesson, and re-typing an
// exposition formatter teaches nothing the second time. What is still yours
// here is everything the formatter cannot decide: which metrics exist, which
// labels they carry, where the bucket bounds go, and where Instrument sits in
// the chain. Read it; NOTES.md asks you to defend it.
//
// It is written to by every request goroutine and read by /metrics, so every
// field is guarded.
type Metrics struct {
	mu        sync.Mutex
	requests  map[statusSeries]int64
	durations map[series]*histogram
	inFlight  int64
}

// NewMetrics returns an empty registry.
func NewMetrics() *Metrics {
	return &Metrics{
		requests:  make(map[statusSeries]int64),
		durations: make(map[series]*histogram),
	}
}

// Instrument wraps h so that every request through it is counted under the
// given method and route. The route is the mux *pattern*, not the request
// path: "/tasks/{id}" is one time series, while "/tasks/1", "/tasks/2", …
// would be one series per task — the cardinality explosion that takes
// metrics systems down.
func (m *Metrics) Instrument(method, route string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.addInFlight(1)
		defer m.addInFlight(-1)

		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		start := time.Now()
		h.ServeHTTP(sw, r)
		m.observe(method, route, sw.status, time.Since(start).Seconds())
	})
}

func (m *Metrics) addInFlight(delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inFlight += delta
}

func (m *Metrics) observe(method, route string, status int, seconds float64) {
	key := series{method: method, route: route}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests[statusSeries{series: key, status: status}]++

	h := m.durations[key]
	if h == nil {
		h = &histogram{buckets: make([]int64, len(durationBuckets))}
		m.durations[key] = h
	}
	h.count++
	h.sum += seconds
	// Buckets are cumulative: an observation lands in every bucket whose
	// bound it fits under, which is what makes le="+Inf" the total.
	for i, bound := range durationBuckets {
		if seconds <= bound {
			h.buckets[i]++
		}
	}
}

// ServeHTTP renders the registry in the Prometheus text exposition format.
// The output is built under the lock and written after it: rendering must
// not block request goroutines on a slow scraper's socket.
func (m *Metrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body := m.render()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	io.WriteString(w, body)
}

func (m *Metrics) render() string {
	var b strings.Builder

	m.mu.Lock()
	defer m.mu.Unlock()

	// Map iteration order is randomised, so every listing gets sorted: a
	// diff between two scrapes should show changed numbers, not reshuffled
	// lines.
	counters := make([]statusSeries, 0, len(m.requests))
	for k := range m.requests {
		counters = append(counters, k)
	}
	sort.Slice(counters, func(i, j int) bool {
		a, c := counters[i], counters[j]
		if a.method != c.method {
			return a.method < c.method
		}
		if a.route != c.route {
			return a.route < c.route
		}
		return a.status < c.status
	})

	// %q is Go quoting, not Prometheus quoting — they differ on non-ASCII,
	// which Go escapes as \u and the exposition format wants left alone. It is
	// safe here only because every label below is drawn from a closed ASCII
	// set: HTTP method tokens, routes you registered, and numbers. Render a
	// user-supplied label value and you need the real escaper from the
	// observability lesson instead.
	b.WriteString("# HELP http_requests_total Total HTTP requests by method, route and status.\n")
	b.WriteString("# TYPE http_requests_total counter\n")
	for _, k := range counters {
		fmt.Fprintf(&b, "http_requests_total{method=%q,route=%q,status=%q} %d\n",
			k.method, k.route, strconv.Itoa(k.status), m.requests[k])
	}

	b.WriteString("# HELP http_requests_in_flight Requests currently being served.\n")
	b.WriteString("# TYPE http_requests_in_flight gauge\n")
	fmt.Fprintf(&b, "http_requests_in_flight %d\n", m.inFlight)

	routes := make([]series, 0, len(m.durations))
	for k := range m.durations {
		routes = append(routes, k)
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].method != routes[j].method {
			return routes[i].method < routes[j].method
		}
		return routes[i].route < routes[j].route
	})

	b.WriteString("# HELP http_request_duration_seconds Request duration by method and route.\n")
	b.WriteString("# TYPE http_request_duration_seconds histogram\n")
	for _, k := range routes {
		h := m.durations[k]
		for i, bound := range durationBuckets {
			fmt.Fprintf(&b, "http_request_duration_seconds_bucket{method=%q,route=%q,le=%q} %d\n",
				k.method, k.route, strconv.FormatFloat(bound, 'g', -1, 64), h.buckets[i])
		}
		fmt.Fprintf(&b, "http_request_duration_seconds_bucket{method=%q,route=%q,le=\"+Inf\"} %d\n",
			k.method, k.route, h.count)
		fmt.Fprintf(&b, "http_request_duration_seconds_sum{method=%q,route=%q} %g\n",
			k.method, k.route, h.sum)
		fmt.Fprintf(&b, "http_request_duration_seconds_count{method=%q,route=%q} %d\n",
			k.method, k.route, h.count)
	}

	return b.String()
}
