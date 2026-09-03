package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// traceparentHeader is the W3C Trace Context header every tracing
	// vendor agrees on: "00-<32 hex trace id>-<16 hex span id>-<2 hex flags>".
	traceparentHeader = "traceparent"
	// traceIDHeader hands the trace id back to the caller so a human can
	// quote it in a bug report.
	traceIDHeader = "X-Trace-Id"
)

// SpanContext is the part of a span that travels: the trace it belongs to,
// the span that is currently active, and whether this trace is being
// recorded. It is immutable and cheap to copy.
type SpanContext struct {
	TraceID string
	SpanID  string
	Sampled bool
}

// Traceparent renders the span context as a W3C traceparent header value.
func (sc SpanContext) Traceparent() string {
	flags := "00"
	if sc.Sampled {
		flags = "01"
	}
	return "00-" + sc.TraceID + "-" + sc.SpanID + "-" + flags
}

// ParseTraceparent reads a traceparent header value. It reports false for
// anything malformed — a caller's broken header must start a fresh trace,
// never corrupt yours.
func ParseTraceparent(v string) (SpanContext, bool) {
	parts := strings.Split(v, "-")
	if len(parts) != 4 || parts[0] != "00" {
		return SpanContext{}, false
	}
	traceID, spanID, flags := parts[1], parts[2], parts[3]
	if !isLowerHex(traceID, 32) || !isLowerHex(spanID, 16) || !isLowerHex(flags, 2) {
		return SpanContext{}, false
	}
	if isAllZero(traceID) || isAllZero(spanID) {
		return SpanContext{}, false
	}
	bits, err := strconv.ParseUint(flags, 16, 8)
	if err != nil {
		return SpanContext{}, false
	}
	return SpanContext{TraceID: traceID, SpanID: spanID, Sampled: bits&1 == 1}, true
}

func isLowerHex(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func isAllZero(s string) bool { return strings.Trim(s, "0") == "" }

type spanContextKey struct{}

// ContextWithSpanContext returns a copy of ctx carrying sc.
func ContextWithSpanContext(ctx context.Context, sc SpanContext) context.Context {
	return context.WithValue(ctx, spanContextKey{}, sc)
}

// SpanContextFrom returns the span context on ctx, if any.
func SpanContextFrom(ctx context.Context) (SpanContext, bool) {
	sc, ok := ctx.Value(spanContextKey{}).(SpanContext)
	return sc, ok
}

// SpanData is a finished span: the record an exporter would ship.
type SpanData struct {
	Name         string
	TraceID      string
	SpanID       string
	ParentSpanID string
	Duration     time.Duration
}

// Tracer starts spans and keeps the finished ones in memory. A real tracer
// hands them to an exporter instead; the shape of what it exports is this.
// The zero Tracer is ready to use.
type Tracer struct {
	mu       sync.Mutex
	finished []SpanData
}

// Span is an in-flight unit of work. Call End exactly once, normally with
// defer.
type Span struct {
	tracer *Tracer
	start  time.Time
	data   SpanData
}

// Start begins a span. If ctx already carries a span context the new span
// joins that trace as a child; otherwise it starts a new trace. The returned
// context carries the new span, so anything called with it — a child span, a
// log record, an outbound request — lands in the same trace.
func (t *Tracer) Start(ctx context.Context, name string) (context.Context, *Span) {
	sc := SpanContext{SpanID: newSpanID(), Sampled: true}
	parent := ""
	if p, ok := SpanContextFrom(ctx); ok {
		sc.TraceID = p.TraceID
		sc.Sampled = p.Sampled
		parent = p.SpanID
	} else {
		sc.TraceID = newTraceID()
	}
	span := &Span{
		tracer: t,
		start:  time.Now(),
		data: SpanData{
			Name:         name,
			TraceID:      sc.TraceID,
			SpanID:       sc.SpanID,
			ParentSpanID: parent,
		},
	}
	return ContextWithSpanContext(ctx, sc), span
}

// SetName renames the span. Useful when the good name is only known later —
// an HTTP span is best named after the route, which the router only reveals
// after it has matched.
func (s *Span) SetName(name string) { s.data.Name = name }

// End records the span's duration and hands it to the tracer.
func (s *Span) End() {
	s.data.Duration = time.Since(s.start)
	s.tracer.mu.Lock()
	s.tracer.finished = append(s.tracer.finished, s.data)
	s.tracer.mu.Unlock()
}

// Finished returns the spans that have ended, in the order they ended.
func (t *Tracer) Finished() []SpanData {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]SpanData(nil), t.finished...)
}

// Inject writes the active span context into outbound request headers. This
// one function is why a trace survives a network hop.
func Inject(ctx context.Context, h http.Header) {
	sc, ok := SpanContextFrom(ctx)
	if !ok {
		return
	}
	h.Set(traceparentHeader, sc.Traceparent())
}

// Ids only need to be unique, not unguessable, so the goroutine-safe
// top-level math/rand/v2 source is the right tool.
func newTraceID() string { return fmt.Sprintf("%016x%016x", rand.Uint64(), rand.Uint64()) }

func newSpanID() string { return fmt.Sprintf("%016x", rand.Uint64()) }
