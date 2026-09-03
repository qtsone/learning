// Command preemptlab pits main against a goroutine spinning in a loop, on a
// single P. Whether main ever runs again depends on which preemption
// mechanisms are available:
//
//	./preemptlab                                  # async preemption saves main
//	GODEBUG=asyncpreemptoff=1 ./preemptlab        # hangs — Ctrl-C to kill
//	GODEBUG=asyncpreemptoff=1 ./preemptlab -call  # cooperative preemption saves main
package main

import (
	"flag"
	"fmt"
	"runtime"
	"time"
)

var sink int64

// tick exists to put a function call — and therefore a stack-check preamble,
// the cooperative preemption point — inside the hot loop.
//
//go:noinline
func tick(x int64) int64 { return x + 1 }

func main() {
	call := flag.Bool("call", false, "make the hot loop call a non-inlined function")
	flag.Parse()

	runtime.GOMAXPROCS(1)
	fmt.Printf("single P; starting spinner (call=%v); main sleeps 100ms…\n", *call)

	go func() {
		var x int64
		if *call {
			for {
				x = tick(x)
				sink = x
			}
		} else {
			for {
				x++
				sink = x
			}
		}
	}()

	time.Sleep(100 * time.Millisecond)
	fmt.Println("main got the only P back — the spinner was preempted")
}
