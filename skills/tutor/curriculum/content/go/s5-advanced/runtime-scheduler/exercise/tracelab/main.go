// Command tracelab runs a small three-phase workload — CPU burn, a blocking
// syscall alongside sleepers, and a channel pipeline — and writes trace.out
// for go tool trace. Each phase is labeled with a trace region so you can
// find it in the timeline UI.
package main

import (
	"context"
	"fmt"
	"os"
	"runtime/trace"
	"sync"
	"syscall"
	"time"
)

func main() {
	f, err := os.Create("trace.out")
	if err != nil {
		fmt.Fprintln(os.Stderr, "create trace.out:", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := trace.Start(f); err != nil {
		fmt.Fprintln(os.Stderr, "start trace:", err)
		os.Exit(1)
	}
	defer trace.Stop()

	ctx, task := trace.NewTask(context.Background(), "workload")
	defer task.End()

	trace.WithRegion(ctx, "phase-1-cpu-burn", cpuBurn)
	trace.WithRegion(ctx, "phase-2-block", blockAndSleep)
	trace.WithRegion(ctx, "phase-3-pipeline", pipeline)
	fmt.Println("wrote trace.out — open it with: go tool trace trace.out")
}

// cpuBurn runs twice as many spinners as Ps for ~200ms: in the trace, every
// PROC row is saturated and the Gs visibly time-slice.
func cpuBurn() {
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			deadline := time.Now().Add(200 * time.Millisecond)
			for time.Now().Before(deadline) {
			}
		}()
	}
	wg.Wait()
}

// blockAndSleep parks one goroutine in a raw blocking read(2) for ~150ms
// while others sleep: in the trace, the reader shows as a syscall-blocked G,
// the sleepers cost nothing.
func blockAndSleep() {
	var fds [2]int
	if err := syscall.Pipe(fds[:]); err != nil {
		panic(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 1)
		_, _ = syscall.Read(fds[0], buf)
	}()
	for range 4 {
		go time.Sleep(150 * time.Millisecond)
	}
	time.Sleep(150 * time.Millisecond)
	if _, err := syscall.Write(fds[1], []byte{1}); err != nil {
		panic(err)
	}
	<-done
}

// pipeline pushes 50k ints through three channel stages: in the trace, tiny
// alternating execution slices — scheduling overhead made visible.
func pipeline() {
	const items = 50000
	src := make(chan int)
	sq := make(chan int)
	out := make(chan int)
	go func() {
		defer close(src)
		for i := range items {
			src <- i
		}
	}()
	go func() {
		defer close(sq)
		for v := range src {
			sq <- v * v
		}
	}()
	go func() {
		defer close(out)
		for v := range sq {
			out <- v
		}
	}()
	var sum int
	for v := range out {
		sum += v
	}
	fmt.Println("pipeline checksum:", sum)
}
