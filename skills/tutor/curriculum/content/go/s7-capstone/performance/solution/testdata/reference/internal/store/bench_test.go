package store_test

import (
	"fmt"
	"testing"
	"time"

	"tutor.local/capstone-reference/internal/note"
	"tutor.local/capstone-reference/internal/store"
)

// sinkNotes keeps the compiler from deleting the work being measured.
var sinkNotes []note.Note

func benchStore(b *testing.B, count int) *store.Memory {
	b.Helper()
	s := store.NewMemory()
	when := time.Unix(0, 0).UTC()
	for i := 0; i < count; i++ {
		tags := []string{"work"}
		if i%10 == 0 {
			tags = append(tags, "urgent")
		}
		n, err := note.New(fmt.Sprintf("n%06d", i), fmt.Sprintf("note %d", i), tags, when)
		if err != nil {
			b.Fatalf("note.New: %v", err)
		}
		if err := s.Add(n); err != nil {
			b.Fatalf("Add: %v", err)
		}
	}
	return s
}

// BenchmarkListByTag is the baseline for the change described in PERF.md.
// It sweeps the store size, because watching ns/op as the input grows 100x is
// how you tell an algorithmic win from a constant-factor one. "urgent" matches
// one note in ten, which is the selectivity the command line actually asks for.
func BenchmarkListByTag(b *testing.B) {
	for _, size := range []int{100, 10_000} {
		b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
			s := benchStore(b, size)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				sinkNotes = s.List("urgent")
			}
		})
	}
}

// BenchmarkListAll is the control. The tag index cannot help an unfiltered
// listing, so this is where a change that moved cost sideways would show up.
func BenchmarkListAll(b *testing.B) {
	s := benchStore(b, 10_000)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		sinkNotes = s.List("")
	}
}
