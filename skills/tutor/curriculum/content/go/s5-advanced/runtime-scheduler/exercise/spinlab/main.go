// Command spinlab spawns CPU-bound goroutines so you can watch the scheduler
// juggle them. Run it under GODEBUG=schedtrace and with varying GOMAXPROCS.
package main

import (
	"flag"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	n := flag.Int("g", 8, "number of CPU-bound goroutines")
	d := flag.Duration("for", 5*time.Second, "how long to run")
	flag.Parse()

	fmt.Printf("NumCPU=%d GOMAXPROCS=%d goroutines=%d duration=%v\n",
		runtime.NumCPU(), runtime.GOMAXPROCS(0), *n, *d)

	counters := make([]atomic.Int64, *n)
	var stop atomic.Bool
	var wg sync.WaitGroup
	for i := range *n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Deliberately call-free: the atomic poll and the atomic add
			// compile to instructions, not calls, so nothing here offers a
			// cooperative preemption point and only async preemption can take
			// the P back. Part 2 of the README asks exactly that; a select or
			// any function call would smuggle in a second, different answer.
			for !stop.Load() {
				counters[i].Add(1)
			}
		}()
	}

	time.Sleep(*d)
	stop.Store(true)
	wg.Wait()

	var total, minC, maxC int64
	minC = counters[0].Load()
	for i := range counters {
		c := counters[i].Load()
		total += c
		minC = min(minC, c)
		maxC = max(maxC, c)
	}
	fmt.Printf("total iterations: %d\n", total)
	fmt.Printf("per-goroutine spread: min=%d max=%d (every goroutine ran)\n",
		minC, maxC)
}
