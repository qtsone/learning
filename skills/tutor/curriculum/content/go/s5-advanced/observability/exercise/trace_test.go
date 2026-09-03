package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

const (
	sampleTraceID = "0af7651916cd43dd8448eb211c80319c"
	sampleSpanID  = "b7ad6b7169203331"
)

func TestParseTraceparent(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantOK  bool
		sampled bool
	}{
		{"sampled", "00-" + sampleTraceID + "-" + sampleSpanID + "-01", true, true},
		{"not sampled", "00-" + sampleTraceID + "-" + sampleSpanID + "-00", true, false},
		{"other flag bits still parse", "00-" + sampleTraceID + "-" + sampleSpanID + "-03", true, true},
		{"empty", "", false, false},
		{"too few fields", "00-" + sampleTraceID + "-" + sampleSpanID, false, false},
		{"too many fields", "00-" + sampleTraceID + "-" + sampleSpanID + "-01-extra", false, false},
		{"unknown version", "01-" + sampleTraceID + "-" + sampleSpanID + "-01", false, false},
		{"uppercase hex", "00-0AF7651916CD43DD8448EB211C80319C-" + sampleSpanID + "-01", false, false},
		{"short trace id", "00-0af7651916cd43dd8448eb211c8031-" + sampleSpanID + "-01", false, false},
		{"short span id", "00-" + sampleTraceID + "-b7ad6b71692033-01", false, false},
		{"non-hex flags", "00-" + sampleTraceID + "-" + sampleSpanID + "-zz", false, false},
		{"zero trace id", "00-00000000000000000000000000000000-" + sampleSpanID + "-01", false, false},
		{"zero span id", "00-" + sampleTraceID + "-0000000000000000-01", false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sc, ok := ParseTraceparent(c.in)
			if ok != c.wantOK {
				t.Fatalf("ParseTraceparent(%q) ok = %v, want %v", c.in, ok, c.wantOK)
			}
			if !c.wantOK {
				return
			}
			if sc.TraceID != sampleTraceID || sc.SpanID != sampleSpanID {
				t.Errorf("parsed ids = %q/%q, want %q/%q", sc.TraceID, sc.SpanID, sampleTraceID, sampleSpanID)
			}
			if sc.Sampled != c.sampled {
				t.Errorf("Sampled = %v, want %v (bit 0 of the flags byte)", sc.Sampled, c.sampled)
			}
		})
	}
}

func TestTraceparentRoundTrip(t *testing.T) {
	for _, sampled := range []bool{true, false} {
		sc := SpanContext{TraceID: sampleTraceID, SpanID: sampleSpanID, Sampled: sampled}
		header := sc.Traceparent()
		back, ok := ParseTraceparent(header)
		if !ok {
			t.Fatalf("ParseTraceparent(%q) rejected a header this package produced", header)
		}
		if back != sc {
			t.Errorf("round trip through %q gave %+v, want %+v", header, back, sc)
		}
	}
}

func isHexOfLen(s string, n int) bool {
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

func TestTracerStartsRootAndChildSpans(t *testing.T) {
	var tracer Tracer

	rootCtx, root := tracer.Start(context.Background(), "root")
	rootSC, ok := SpanContextFrom(rootCtx)
	if !ok {
		t.Fatal("the context returned by Start carries no span context — children and log records read it from there")
	}
	if !isHexOfLen(rootSC.TraceID, 32) || !isHexOfLen(rootSC.SpanID, 16) {
		t.Fatalf("generated ids %q/%q are not 32 and 16 lowercase hex digits", rootSC.TraceID, rootSC.SpanID)
	}

	childCtx, child := tracer.Start(rootCtx, "child")
	childSC, _ := SpanContextFrom(childCtx)
	if childSC.TraceID != rootSC.TraceID {
		t.Errorf("child trace id = %q, want the parent's %q — a child stays in the same trace", childSC.TraceID, rootSC.TraceID)
	}
	if childSC.SpanID == rootSC.SpanID {
		t.Error("child reused the parent's span id — every span needs its own")
	}

	child.End()
	root.End()

	finished := tracer.Finished()
	if len(finished) != 2 {
		t.Fatalf("Finished() has %d spans, want 2", len(finished))
	}
	if finished[0].Name != "child" || finished[1].Name != "root" {
		t.Errorf("Finished() = %q, %q, want them in End order: child, root", finished[0].Name, finished[1].Name)
	}
	if finished[0].ParentSpanID != rootSC.SpanID {
		t.Errorf("child ParentSpanID = %q, want the root span id %q", finished[0].ParentSpanID, rootSC.SpanID)
	}
	if finished[1].ParentSpanID != "" {
		t.Errorf("root ParentSpanID = %q, want empty — nothing was above it", finished[1].ParentSpanID)
	}
	if finished[0].Duration < 0 {
		t.Errorf("child Duration = %v, want a non-negative measurement", finished[0].Duration)
	}
}

func TestTracerContinuesARemoteTrace(t *testing.T) {
	var tracer Tracer
	remote, ok := ParseTraceparent("00-" + sampleTraceID + "-" + sampleSpanID + "-00")
	if !ok {
		t.Fatal("fix ParseTraceparent first")
	}

	ctx, span := tracer.Start(ContextWithSpanContext(context.Background(), remote), "GET /items")
	span.SetName("GET /items/{id}")
	span.End()

	sc, _ := SpanContextFrom(ctx)
	if sc.TraceID != sampleTraceID {
		t.Errorf("trace id = %q, want the caller's %q — joining a trace means adopting its id", sc.TraceID, sampleTraceID)
	}
	if sc.Sampled {
		t.Error("Sampled = true, want the caller's decision (false) to be inherited")
	}

	finished := tracer.Finished()
	if len(finished) != 1 {
		t.Fatalf("Finished() has %d spans, want 1", len(finished))
	}
	if finished[0].ParentSpanID != sampleSpanID {
		t.Errorf("ParentSpanID = %q, want the caller's span id %q", finished[0].ParentSpanID, sampleSpanID)
	}
	if finished[0].Name != "GET /items/{id}" {
		t.Errorf("Name = %q, want the name set by SetName", finished[0].Name)
	}
}

func TestInjectPropagatesTraceparent(t *testing.T) {
	seen := make(chan string, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.Header.Get(traceparentHeader)
	}))
	defer upstream.Close()

	var tracer Tracer
	ctx, span := tracer.Start(context.Background(), "outbound")
	defer span.End()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstream.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	Inject(ctx, req.Header)
	resp, err := upstream.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	header := <-seen
	got, ok := ParseTraceparent(header)
	if !ok {
		t.Fatalf("the upstream server received traceparent %q, which does not parse", header)
	}
	want, _ := SpanContextFrom(ctx)
	if got != want {
		t.Errorf("upstream saw %+v, want the active span context %+v", got, want)
	}
}

func TestInjectWithoutASpanDoesNothing(t *testing.T) {
	h := http.Header{}
	Inject(context.Background(), h)
	if got := h.Get(traceparentHeader); got != "" {
		t.Errorf("Inject set %s = %q with no span on the context, want no header at all", traceparentHeader, got)
	}
}

func TestTracerIsSafeUnderConcurrency(t *testing.T) {
	var tracer Tracer
	const goroutines = 16

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, span := tracer.Start(context.Background(), "work")
			span.End()
		}()
	}
	wg.Wait()

	if got := len(tracer.Finished()); got != goroutines {
		t.Errorf("Finished() has %d spans, want %d", got, goroutines)
	}
}
