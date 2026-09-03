//go:build unix

package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// Ctrl-C on a terminal sends SIGINT to the process. This test sends SIGINT to
// itself and asserts that the wiring turns it into a cancelled context, which
// is what every other cancellation test in this package assumes.
//
// Note what it does not do: sleep. The assertion is a channel receive, and the
// watchdog exists only so a broken implementation fails in seconds instead of
// hanging forever. A sleep-based version of this test would be flaky on a busy
// machine and slow on an idle one.
func TestInterruptCancelsTheContext(t *testing.T) {
	// Belt and braces: catching SIGINT anywhere in the process disables the
	// default "die immediately" action, so a half-wired implementation fails
	// this assertion instead of killing the test binary.
	guard := make(chan os.Signal, 1)
	signal.Notify(guard, os.Interrupt)
	defer signal.Stop(guard)

	ctx, stop := notifyContext(context.Background())
	defer stop()

	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("kill: %v", err)
	}

	select {
	case <-ctx.Done():
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Errorf("ctx.Err() = %v, want context.Canceled", ctx.Err())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("SIGINT did not cancel the context: is signal.NotifyContext wired up?")
	}
}
