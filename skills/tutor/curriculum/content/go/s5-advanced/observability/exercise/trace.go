package main

import (
	"context"
	"net/http"
	"sync"
	"time"
)

const (
	// traceparentHeader is the W3C Trace Context header every tracing vendor
	// agrees on: "00-<32 hex trace id>-<16 hex span id>-<2 hex flags>".
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
// The flags byte is "01" when sampled and "00" when not.
func (sc SpanContext) Traceparent() string {
	// TODO
	return ""
}

// ParseTraceparent reads a traceparent header value. It reports false for
// anything malformed — a caller's broken header must start a fresh trace,
// never corrupt yours. Reject: the wrong number of fields, a version other
// than "00", ids of the wrong length, anything that is not lowercase hex, and
// all-zero ids.
func ParseTraceparent(v string) (SpanContext, bool) {
	// TODO
	return SpanContext{}, false
}

type spanContextKey struct{}

// ContextWithSpanContext returns a copy of ctx carrying sc.
func ContextWithSpanContext(ctx context.Context, sc SpanContext) context.Context {
	// TODO: the same unexported-key trick as WithRequestID.
	return ctx
}

// SpanContextFrom returns the span context on ctx, if any.
func SpanContextFrom(ctx context.Context) (SpanContext, bool) {
	// TODO
	return SpanContext{}, false
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
// joins that trace as a child — same trace id, same sampling decision, parent
// set to the span that was active. Otherwise it starts a new trace with a
// fresh 32-hex-digit trace id. The returned context carries the new span, so
// anything called with it — a child span, a log record, an outbound request —
// lands in the same trace.
func (t *Tracer) Start(ctx context.Context, name string) (context.Context, *Span) {
	// TODO: ids only need to be unique, not unguessable — math/rand/v2 and
	// fmt.Sprintf("%016x", …) are the right tools.
	return ctx, &Span{tracer: t}
}

// SetName renames the span. Useful when the good name is only known later —
// an HTTP span is best named after the route, which the router only reveals
// after it has matched.
func (s *Span) SetName(name string) {
	// TODO
}

// End records the span's duration and hands it to the tracer.
func (s *Span) End() {
	// TODO
}

// Finished returns the spans that have ended, in the order they ended.
func (t *Tracer) Finished() []SpanData {
	// TODO: hand back a copy, not the live slice.
	return nil
}

// Inject writes the active span context into outbound request headers. This
// one function is why a trace survives a network hop. With no span on the
// context it does nothing.
func Inject(ctx context.Context, h http.Header) {
	// TODO
}
