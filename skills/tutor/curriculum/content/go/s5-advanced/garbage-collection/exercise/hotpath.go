// Package hotpath renders event logs on the hot path of an ingest service.
// The functions here are called thousands of times per second in production,
// so their allocation behavior matters as much as their correctness.
package hotpath

import (
	"bytes"
	"io"
)

// Event is one ingested log event.
type Event struct {
	ID    int64
	Level string
	Msg   string
}

// FormatEvents renders events as "LEVEL: message" lines, one line per event.
//
// TODO: this version allocates a fresh string on every iteration — each +=
// copies the entire result built so far. Rebuild it around strings.Builder,
// sized up front with Grow (sum the byte lengths first), so one call costs
// O(1) allocations no matter how many events it renders.
func FormatEvents(events []Event) string {
	s := ""
	for _, e := range events {
		s += e.Level + ": " + e.Msg + "\n"
	}
	return s
}

// EventIDs returns the IDs of all events, in order.
//
// TODO: the slice starts with zero capacity, so append re-grows (and
// re-copies) it about a dozen times for a few thousand events. Preallocate
// the exact capacity you already know.
func EventIDs(events []Event) []int64 {
	var ids []int64
	for _, e := range events {
		ids = append(ids, e.ID)
	}
	return ids
}

// WriteEvents writes the same rendering as FormatEvents to w, in a single
// w.Write call. It is called concurrently from many goroutines.
//
// TODO: two problems. First, the loop builds a throwaway line string per
// event — write the pieces straight into the buffer instead. Second, every
// call builds its scratch buffer from nothing and discards it; reuse buffers
// across calls with a package-level sync.Pool of *bytes.Buffer (Get, defer
// Put, Reset before use). Do not share one global buffer — the concurrency
// test and the race detector are watching.
func WriteEvents(w io.Writer, events []Event) error {
	var buf bytes.Buffer
	for _, e := range events {
		line := e.Level + ": " + e.Msg + "\n"
		buf.WriteString(line)
	}
	_, err := w.Write(buf.Bytes())
	return err
}
