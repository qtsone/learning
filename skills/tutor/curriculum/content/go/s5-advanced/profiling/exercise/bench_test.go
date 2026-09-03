package report

import (
	"strconv"
	"testing"
)

// Sinks defeat dead-code elimination: the compiler may not delete a call
// whose result lands in a package-level variable.
var (
	sinkStats  []UserStat
	sinkString string
)

// Run these by hand, WITHOUT -race (instrumentation skews every number):
//
//	go test -bench=. -run='^$' -count=10 | tee before.txt
//	... fix the code ...
//	go test -bench=. -run='^$' -count=10 | tee after.txt
//	benchstat before.txt after.txt
//
// Capture profiles from the same benchmarks:
//
//	go test -bench=BenchmarkTopUsers/10000 -run='^$' -cpuprofile=cpu.out -memprofile=mem.out .
//	go tool pprof -top cpu.out

func BenchmarkTopUsers(b *testing.B) {
	for _, size := range []int{1_000, 10_000} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			entries := genEntries(size)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				sinkStats = TopUsers(entries, 10)
			}
		})
	}
}

func BenchmarkRender(b *testing.B) {
	for _, size := range []int{100, 2_000} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			stats := genStats(size)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				sinkString = Render(stats)
			}
		})
	}
}
