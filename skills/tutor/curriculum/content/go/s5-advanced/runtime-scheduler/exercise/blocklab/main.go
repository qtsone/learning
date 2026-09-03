// Command blocklab parks goroutines two different ways — runtime-managed
// sleeps vs raw blocking syscalls — so you can watch what each costs in OS
// threads. Observe with GODEBUG=schedtrace=1000 and compare the threads=
// field between the two modes. Unix only (it uses pipe(2)).
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"
)

func main() {
	n := flag.Int("g", 64, "number of goroutines to park")
	mode := flag.String("mode", "sleep", "sleep | syscall")
	d := flag.Duration("for", 10*time.Second, "how long to keep observing")
	flag.Parse()

	for range *n {
		switch *mode {
		case "sleep":
			// Parked by the runtime's timer machinery: the G waits, no
			// thread is held.
			go time.Sleep(*d + time.Minute)
		case "syscall":
			go blockInRead()
		default:
			fmt.Fprintln(os.Stderr, "-mode must be sleep or syscall")
			os.Exit(2)
		}
	}

	for elapsed := time.Duration(0); elapsed < *d; elapsed += time.Second {
		time.Sleep(time.Second)
		fmt.Printf("mode=%s parked=%d goroutines=%d\n",
			*mode, *n, runtime.NumGoroutine())
	}
	fmt.Println("done — compare the threads= field of the schedtrace lines")
}

// blockInRead traps this goroutine's OS thread in a raw read(2) on a pipe
// nobody ever writes to. The syscall package bypasses the runtime's
// netpoller, so the thread itself sits in the kernel — the same thing a
// blocking cgo call or an un-pollable fd does to your program. (Production
// code would use golang.org/x/sys or os.Pipe, which the netpoller handles;
// the raw call here is deliberate.)
func blockInRead() {
	var fds [2]int
	if err := syscall.Pipe(fds[:]); err != nil {
		panic(err)
	}
	buf := make([]byte, 1)
	_, _ = syscall.Read(fds[0], buf) // never returns
}
