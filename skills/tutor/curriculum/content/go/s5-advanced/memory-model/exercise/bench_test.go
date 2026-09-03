package memlab

import "testing"

// Run these by hand and record allocs/op in NOTES.md:
//
//	go test -bench=Summarize -benchmem
//
// The ns/op column varies from machine to machine (and balloons under
// -race); the allocs/op column is the stable, comparable number.

var (
	benchXs = func() []float64 {
		xs := make([]float64, 4096)
		for i := range xs {
			xs[i] = float64(i%101) - 50
		}
		return xs
	}()
	benchReportPtr *Report
	benchReport    Report
)

func BenchmarkSummarizeHeap(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchReportPtr = SummarizeHeap(benchXs)
	}
}

func BenchmarkSummarize(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		benchReport = Summarize(benchXs)
	}
}
