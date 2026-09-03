// Package hotpath renders event logs on the hot path of an ingest service.
// The functions here are called thousands of times per second in production,
// so their allocation behavior matters as much as their correctness.
package hotpath

import (
	"bytes"
	"io"
	"strings"
	"sync"
)

// Event is one ingested log event.
type Event struct {
	ID    int64
	Level string
	Msg   string
}

// FormatEvents renders events as "LEVEL: message" lines, one line per event.
// It sizes the builder up front, so the whole render is one allocation.
func FormatEvents(events []Event) string {
	total := 0
	for _, e := range events {
		total += len(e.Level) + len(e.Msg) + 3 // ": " plus "\n"
	}
	var b strings.Builder
	b.Grow(total)
	for _, e := range events {
		b.WriteString(e.Level)
		b.WriteString(": ")
		b.WriteString(e.Msg)
		b.WriteByte('\n')
	}
	return b.String()
}

// EventIDs returns the IDs of all events, in order.
func EventIDs(events []Event) []int64 {
	ids := make([]int64, 0, len(events))
	for _, e := range events {
		ids = append(ids, e.ID)
	}
	return ids
}

// bufPool recycles scratch buffers across WriteEvents calls. After a few
// calls the pooled buffers have grown to working size, and steady-state
// traffic allocates nothing.
var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// WriteEvents writes the same rendering as FormatEvents to w, in a single
// w.Write call. It is safe for concurrent use: each call borrows its own
// buffer from the pool.
func WriteEvents(w io.Writer, events []Event) error {
	buf := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buf)
	buf.Reset()
	for _, e := range events {
		buf.WriteString(e.Level)
		buf.WriteString(": ")
		buf.WriteString(e.Msg)
		buf.WriteByte('\n')
	}
	// Putting buf back after Write is safe: io.Writer implementations must
	// not retain p beyond the call.
	_, err := w.Write(buf.Bytes())
	return err
}
