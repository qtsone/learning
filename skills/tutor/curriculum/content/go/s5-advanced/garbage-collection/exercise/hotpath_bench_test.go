package hotpath

import (
	"io"
	"testing"
)

// Run these by hand, before and after your fixes, WITHOUT the race detector:
//
//	go test -bench=. -benchmem
//
// Watch allocs/op and B/op collapse. Nothing in the automated verification
// depends on these numbers — they are your evidence, not your gate.

// Each benchmark calls b.ResetTimer after building its fixture: ResetTimer
// zeroes the allocation counters too, so the setup's allocations are not
// billed to allocs/op.

func BenchmarkFormatEvents(b *testing.B) {
	events := makeEvents(allocProbeSize)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatEvents(events)
	}
}

func BenchmarkEventIDs(b *testing.B) {
	events := makeEvents(allocProbeSize)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EventIDs(events)
	}
}

func BenchmarkWriteEvents(b *testing.B) {
	events := makeEvents(allocProbeSize)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := WriteEvents(io.Discard, events); err != nil {
			b.Fatal(err)
		}
	}
}
