// Command costlab parks a crowd of goroutines and measures what each one
// costs in stack memory. Use -depth to make every goroutine recurse before
// parking and watch the per-goroutine cost grow with its stack.
package main

import (
	"flag"
	"fmt"
	"runtime"
)

var sink byte

// dive burns -depth stack frames before parking, forcing the runtime to grow
// the goroutine's stack. The pad array fattens each frame so growth shows up
// at modest depths.
//
//go:noinline
func dive(d int, ready, block chan struct{}) {
	var pad [256]byte
	pad[d%len(pad)] = byte(d)
	sink = pad[d%len(pad)]
	if d <= 0 {
		ready <- struct{}{}
		<-block
		return
	}
	dive(d-1, ready, block)
}

func main() {
	n := flag.Int("g", 100000, "number of goroutines to park")
	depth := flag.Int("depth", 0, "recursion depth before parking")
	flag.Parse()

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	ready := make(chan struct{})
	block := make(chan struct{})
	for range *n {
		go dive(*depth, ready, block)
	}
	for range *n {
		<-ready
	}

	runtime.GC()
	runtime.ReadMemStats(&after)

	stack := after.StackSys - before.StackSys
	fmt.Printf("parked goroutines: %d (runtime sees %d)\n",
		*n, runtime.NumGoroutine())
	fmt.Printf("stack memory: %.1f MiB total ≈ %.0f bytes per goroutine (depth=%d)\n",
		float64(stack)/(1<<20), float64(stack)/float64(*n), *depth)
	fmt.Printf("for scale: %d OS threads at a 1 MiB stack each would reserve %d MiB\n",
		*n, *n)
	close(block)
}
