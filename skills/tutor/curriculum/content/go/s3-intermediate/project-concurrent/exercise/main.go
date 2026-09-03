package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
)

// main is provided and complete: it wires the operating system to run.
// signal.NotifyContext returns a context that is canceled on the first
// Ctrl-C (SIGINT); that one cancellation propagates through Check into
// every in-flight request. Everything testable lives in run.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "linkcheck:", err)
		os.Exit(1)
	}
}
