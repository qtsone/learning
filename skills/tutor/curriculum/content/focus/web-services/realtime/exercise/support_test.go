package realtime

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// quietLogger keeps the eviction warnings out of the test output. Passing nil
// to NewHub would mean slog.Default(), which writes to stderr.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// hangGuard bounds how long a test waits for something that should already
// have happened. It is a guard against a hung test, never an assertion about
// how fast your code is: every timing rule in this package is driven by the
// injected clock, and no test sleeps to let time pass.
const hangGuard = 2 * time.Second

// testStart is the fixed instant the fake clock starts at, so the heartbeat
// frame is the same text on every machine on every day.
var testStart = time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)

// fakeClock is the S5 pattern extended with a ticker a test fires by hand.
type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	tickers chan chan time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: testStart, tickers: make(chan chan time.Time, 4)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

// NewTicker hands the handler an unbuffered channel and keeps a reference so
// the test can deliver a tick to it. The channel being unbuffered is what
// makes tick a synchronisation point instead of a race.
func (c *fakeClock) NewTicker(time.Duration) (<-chan time.Time, func()) {
	ch := make(chan time.Time)
	select {
	case c.tickers <- ch:
	default:
	}
	return ch, func() {}
}

// tick delivers one heartbeat tick and waits until the handler takes it.
func (c *fakeClock) tick(t *testing.T) {
	t.Helper()
	var ch chan time.Time
	select {
	case ch = <-c.tickers:
	case <-time.After(hangGuard):
		t.Fatal("no ticker was created from the injected Clock: heartbeats must come from Clock.NewTicker, not time.Tick or time.NewTicker")
	}
	select {
	case ch <- c.Now():
	case <-time.After(hangGuard):
		t.Fatal("the tick was never received: the streaming loop must select on the ticker channel")
	}
	select { // put it back so a later tick reaches the same handler
	case c.tickers <- ch:
	default:
	}
}

// streamRecorder is a ResponseWriter that reports every flush separately, so a
// test can observe a response that is still being written. httptest's recorder
// cannot: reading its buffer while the handler writes is a data race, and it
// would not tell you whether anything was actually flushed.
type streamRecorder struct {
	mu       sync.Mutex
	header   http.Header
	snapshot http.Header
	code     int
	pending  bytes.Buffer
	flushes  chan string
}

func newStreamRecorder() *streamRecorder {
	return &streamRecorder{header: make(http.Header), flushes: make(chan string, 64)}
}

func (s *streamRecorder) Header() http.Header { return s.header }

func (s *streamRecorder) WriteHeader(code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.code != 0 {
		return
	}
	s.code = code
	s.snapshot = s.header.Clone()
}

func (s *streamRecorder) Write(p []byte) (int, error) {
	s.WriteHeader(http.StatusOK)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pending.Write(p)
}

// Flush publishes everything written since the last flush. Unflushed bytes are
// invisible to the test, exactly as they are invisible to a real client.
func (s *streamRecorder) Flush() {
	s.WriteHeader(http.StatusOK)
	s.mu.Lock()
	chunk := s.pending.String()
	s.pending.Reset()
	s.mu.Unlock()
	if chunk == "" {
		return
	}
	select {
	case s.flushes <- chunk:
	case <-time.After(hangGuard):
	}
}

func (s *streamRecorder) status() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.code
}

func (s *streamRecorder) headerValue(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshot.Get(key)
}

// plainRecorder implements ResponseWriter and nothing else — no Flush, no
// Unwrap — which is how the "this response cannot stream" path gets tested.
type plainRecorder struct {
	header http.Header
	code   int
	body   bytes.Buffer
}

func newPlainRecorder() *plainRecorder {
	return &plainRecorder{header: make(http.Header)}
}

func (p *plainRecorder) Header() http.Header { return p.header }

func (p *plainRecorder) WriteHeader(code int) {
	if p.code == 0 {
		p.code = code
	}
}

func (p *plainRecorder) Write(b []byte) (int, error) {
	p.WriteHeader(http.StatusOK)
	return p.body.Write(b)
}

// stream runs a handler on its own goroutine, the way a real server does, and
// collects what it flushes.
type stream struct {
	rec    *streamRecorder
	cancel context.CancelFunc
	done   chan struct{}
	seen   strings.Builder
}

func startStream(t *testing.T, h http.Handler, r *http.Request) *stream {
	t.Helper()
	ctx, cancel := context.WithCancel(r.Context())
	s := &stream{rec: newStreamRecorder(), cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(s.done)
		h.ServeHTTP(s.rec, r.WithContext(ctx))
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-s.done:
		case <-time.After(hangGuard):
			t.Error("a streaming handler was still running after its request context was cancelled")
		}
	})
	return s
}

func (s *stream) drain() {
	for {
		select {
		case chunk := <-s.rec.flushes:
			s.seen.WriteString(chunk)
		default:
			return
		}
	}
}

// await waits until the flushed bytes so far contain want.
func (s *stream) await(t *testing.T, want string) {
	t.Helper()
	deadline := time.After(hangGuard)
	for {
		if strings.Contains(s.seen.String(), want) {
			return
		}
		select {
		case chunk := <-s.rec.flushes:
			s.seen.WriteString(chunk)
		case <-s.done:
			s.drain()
			if strings.Contains(s.seen.String(), want) {
				return
			}
			t.Fatalf("the handler returned without sending %q\nstream so far:\n%s", want, s.seen.String())
		case <-deadline:
			t.Fatalf("timed out waiting for %q (did you flush after writing it?)\nstream so far:\n%s", want, s.seen.String())
		}
	}
}

// sent reports whether text has appeared in the stream so far. Call it only
// after awaiting something that must come later, so the answer is not a race.
func (s *stream) sent(text string) bool {
	s.drain()
	return strings.Contains(s.seen.String(), text)
}

// stop cancels the request, as a client disconnecting does, and waits for the
// handler to notice.
func (s *stream) stop(t *testing.T) {
	t.Helper()
	s.cancel()
	s.awaitReturn(t, "the streaming loop must select on r.Context().Done() and return when the client goes away")
}

func (s *stream) awaitReturn(t *testing.T, why string) {
	t.Helper()
	select {
	case <-s.done:
	case <-time.After(hangGuard):
		t.Fatalf("the handler is still running: %s", why)
	}
}

// serveGuarded calls a handler that is expected to return on its own.
func serveGuarded(t *testing.T, h http.Handler, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.ServeHTTP(w, r)
	}()
	select {
	case <-done:
	case <-time.After(hangGuard):
		t.Fatal("the handler did not return: it must not start streaming when it cannot")
	}
}

// waitSubscribers blocks until the hub has n subscribers, so a test can
// publish knowing the handler is registered. It spins instead of sleeping:
// there is no wall-clock claim here, only an ordering one.
func waitSubscribers(t *testing.T, h *Hub, n int) {
	t.Helper()
	deadline := time.Now().Add(hangGuard)
	for h.Subscribers() != n {
		if time.Now().After(deadline) {
			t.Fatalf("hub has %d subscriber(s), want %d", h.Subscribers(), n)
		}
		runtime.Gosched()
	}
}

func recvEvent(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	select {
	case e, ok := <-ch:
		if !ok {
			t.Fatal("the subscriber's channel was closed, want an event")
		}
		return e
	case <-time.After(hangGuard):
		t.Fatal("no event arrived on the subscriber's channel")
		return Event{}
	}
}

func expectClosed(t *testing.T, ch <-chan Event) {
	t.Helper()
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
			// Buffered events are delivered before the close is visible.
		case <-time.After(hangGuard):
			t.Fatal("the subscriber's channel is still open, want it closed")
			return
		}
	}
}
