package hotpath

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
)

// allocProbeSize is deliberately large: the naive implementations allocate
// proportionally to it (hundreds to thousands of allocs), while the fixed
// ones stay O(1). The gates below use generous bounds so they are robust
// under the race detector — they measure allocation counts, never time.
const allocProbeSize = 2048

// makeEvents builds realistic events. The messages are deliberately longer
// than 32 bytes: below that size the compiler can build a temporary string
// concatenation in a stack buffer, which would hide the per-event allocation
// the WriteEvents gate is looking for.
func makeEvents(n int) []Event {
	levels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	events := make([]Event, n)
	for i := range events {
		events[i] = Event{
			ID:    int64(i + 1),
			Level: levels[i%len(levels)],
			Msg:   fmt.Sprintf("event %d accepted by the ingest pipeline", i+1),
		}
	}
	return events
}

// wantRendering is the reference rendering, built the slow-but-obvious way.
func wantRendering(events []Event) string {
	var b strings.Builder
	for _, e := range events {
		fmt.Fprintf(&b, "%s: %s\n", e.Level, e.Msg)
	}
	return b.String()
}

func TestFormatEvents(t *testing.T) {
	cases := []struct {
		name string
		n    int
	}{
		{"empty", 0},
		{"one event", 1},
		{"many events", 100},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			events := makeEvents(c.n)
			if got, want := FormatEvents(events), wantRendering(events); got != want {
				t.Errorf("FormatEvents(%d events) = %q, want %q", c.n, got, want)
			}
		})
	}
}

func TestEventIDs(t *testing.T) {
	events := makeEvents(5)
	got := EventIDs(events)
	want := []int64{1, 2, 3, 4, 5}
	if len(got) != len(want) {
		t.Fatalf("EventIDs returned %d ids, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("EventIDs()[%d] = %d, want %d", i, got[i], want[i])
		}
	}
	if got := EventIDs(nil); len(got) != 0 {
		t.Errorf("EventIDs(nil) returned %d ids, want 0", len(got))
	}
}

func TestWriteEvents(t *testing.T) {
	events := makeEvents(100)
	var buf bytes.Buffer
	if err := WriteEvents(&buf, events); err != nil {
		t.Fatalf("WriteEvents returned unexpected error: %v", err)
	}
	if got, want := buf.String(), wantRendering(events); got != want {
		t.Errorf("WriteEvents wrote %q, want %q", got, want)
	}
}

// TestWriteEventsSingleWrite pins the contract the optimization must not
// break: the whole rendering reaches w in one Write call, however you build
// it. (Writers are often sockets; one syscall beats 2048.)
func TestWriteEventsSingleWrite(t *testing.T) {
	sink := &lengthWriter{}
	if err := WriteEvents(sink, makeEvents(64)); err != nil {
		t.Fatalf("WriteEvents returned unexpected error: %v", err)
	}
	if sink.calls != 1 {
		t.Errorf("WriteEvents made %d Write calls, want 1", sink.calls)
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write(p []byte) (int, error) { return 0, w.err }

func TestWriteEventsPropagatesError(t *testing.T) {
	sentinel := errors.New("disk full")
	err := WriteEvents(failingWriter{err: sentinel}, makeEvents(3))
	if !errors.Is(err, sentinel) {
		t.Errorf("WriteEvents with failing writer returned %v, want %v", err, sentinel)
	}
}

// TestWriteEventsConcurrent hammers WriteEvents from several goroutines.
// A correct implementation (fresh buffer per call, or a sync.Pool) passes;
// a single shared package-level buffer corrupts output and trips the race
// detector.
func TestWriteEventsConcurrent(t *testing.T) {
	events := makeEvents(512)
	want := wantRendering(events)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			for i := 0; i < 25; i++ {
				buf.Reset()
				if err := WriteEvents(&buf, events); err != nil {
					t.Errorf("WriteEvents returned unexpected error: %v", err)
					return
				}
				if buf.String() != want {
					t.Error("WriteEvents produced corrupted output under concurrency — is scratch state shared between goroutines?")
					return
				}
			}
		}()
	}
	wg.Wait()
}

// --- Allocation gates -------------------------------------------------------
//
// These are the tests that fail on the starter code. They count allocations
// with testing.AllocsPerRun, which is deterministic; the bounds leave several
// times more headroom than the reference solution needs, so a correct fix
// passes comfortably — but the naive versions (hundreds to thousands of
// allocs) cannot.

func TestFormatEventsAllocs(t *testing.T) {
	events := makeEvents(allocProbeSize)
	want := wantRendering(events)
	avg := testing.AllocsPerRun(10, func() {
		if got := FormatEvents(events); len(got) != len(want) {
			t.Fatalf("FormatEvents returned %d bytes, want %d", len(got), len(want))
		}
	})
	const maxAllocs = 8
	if avg > maxAllocs {
		t.Errorf("FormatEvents(%d events) averaged %.1f allocs/op, want <= %d\n"+
			"build the result once: sum the lengths, Grow a strings.Builder, write into it",
			allocProbeSize, avg, maxAllocs)
	}
}

func TestEventIDsAllocs(t *testing.T) {
	events := makeEvents(allocProbeSize)
	avg := testing.AllocsPerRun(10, func() {
		if got := EventIDs(events); len(got) != allocProbeSize {
			t.Fatalf("EventIDs returned %d ids, want %d", len(got), allocProbeSize)
		}
	})
	const maxAllocs = 4
	if avg > maxAllocs {
		t.Errorf("EventIDs(%d events) averaged %.1f allocs/op, want <= %d\n"+
			"you know the final length before the loop — make([]int64, 0, len(events))",
			allocProbeSize, avg, maxAllocs)
	}
}

func TestWriteEventsAllocs(t *testing.T) {
	events := makeEvents(allocProbeSize)
	wantLen := len(wantRendering(events))
	sink := &lengthWriter{}
	avg := testing.AllocsPerRun(30, func() {
		sink.n = 0
		if err := WriteEvents(sink, events); err != nil {
			t.Fatalf("WriteEvents returned unexpected error: %v", err)
		}
		if sink.n != wantLen {
			t.Fatalf("WriteEvents wrote %d bytes, want %d", sink.n, wantLen)
		}
	})
	// Generous on purpose: a pooled buffer occasionally re-grows under the
	// race detector (it randomly drops pool puts), so steady state is a few
	// allocs/op, not zero. Per-event allocations are ~2000/op and cannot
	// hide under this bound.
	const maxAllocs = 64
	if avg > maxAllocs {
		t.Errorf("WriteEvents(%d events) averaged %.1f allocs/op in steady state, want <= %d\n"+
			"stop building a line string per event: write the pieces into a scratch buffer reused via sync.Pool",
			allocProbeSize, avg, maxAllocs)
	}
}

// lengthWriter counts bytes and calls without allocating or retaining them.
type lengthWriter struct {
	n     int
	calls int
}

func (w *lengthWriter) Write(p []byte) (int, error) {
	w.n += len(p)
	w.calls++
	return len(p), nil
}
