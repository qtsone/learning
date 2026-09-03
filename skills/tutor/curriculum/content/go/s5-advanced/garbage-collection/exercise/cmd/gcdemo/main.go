// Command gcdemo allocates steadily while keeping a fixed-size live set,
// so you can watch the collector's pacing decisions from the outside.
//
// Run it with gctrace on and vary the knobs (see LESSON.md):
//
//	GODEBUG=gctrace=1 go run ./cmd/gcdemo
//	GODEBUG=gctrace=1 GOGC=50 go run ./cmd/gcdemo
//	GODEBUG=gctrace=1 GOGC=400 go run ./cmd/gcdemo
//	GODEBUG=gctrace=1 GOGC=off GOMEMLIMIT=100MiB go run ./cmd/gcdemo
package main

import (
	"fmt"
	"runtime"
)

func main() {
	// ~64 MiB stays live (16384 slices of 4 KiB); everything else the loop
	// allocates — about 1 GiB in total — becomes garbage almost immediately.
	const (
		liveObjects = 1 << 14
		totalAllocs = 1 << 18
		objectSize  = 4 << 10
	)
	live := make([][]byte, liveObjects)
	for i := 0; i < totalAllocs; i++ {
		buf := make([]byte, objectSize)
		buf[0] = byte(i)
		live[i%liveObjects] = buf
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	fmt.Printf("allocated %d MiB in %d KiB chunks, kept %d MiB live\n",
		ms.TotalAlloc>>20, objectSize>>10, int64(liveObjects*objectSize)>>20)
	fmt.Printf("heap in use at exit: %d MiB, GC cycles: %d\n",
		ms.HeapAlloc>>20, ms.NumGC)
}
