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

// Signal delivery is a Unix concept, so this file is built only there — the
// //go:build unix line at the top is how you keep a platform-specific test out
// of a Windows build instead of skipping it at run time.
func TestWithInterruptCancelsOnSignal(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			// The test sends the signal to itself. Registering our own channel
			// first is a safety net: if WithInterrupt is not (yet) listening,
			// the default action for both of these signals is to terminate the
			// process, and this test binary would die rather than fail.
			// Registrations do not compete — every registered channel gets a
			// copy of the signal.
			guard := make(chan os.Signal, 1)
			signal.Notify(guard, sig)
			defer signal.Stop(guard)

			ctx, stop := WithInterrupt(context.Background())
			defer stop()

			if err := syscall.Kill(syscall.Getpid(), sig); err != nil {
				t.Fatalf("sending %v to self: %v", sig, err)
			}

			select {
			case <-ctx.Done():
			case <-time.After(failsafe):
				t.Fatalf("%v did not cancel the context — is WithInterrupt registered for it?", sig)
			}
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Errorf("ctx.Err() = %v, want context.Canceled", ctx.Err())
			}
		})
	}
}

// A signal must not abandon work in progress: the context becomes done, the
// work loop notices, and the deferred cleanup still runs. Nothing slow belongs
// in the handler itself.
func TestInterruptedWorkStillCleansUp(t *testing.T) {
	guard := make(chan os.Signal, 1)
	signal.Notify(guard, syscall.SIGINT)
	defer signal.Stop(guard)

	ctx, stop := WithInterrupt(context.Background())
	defer stop()

	cleaned := false
	err := func() error {
		defer func() { cleaned = true }()
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(failsafe):
			return errors.New("SIGINT did not cancel the context")
		}
	}()

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("work returned err = %v, want context.Canceled", err)
	}
	if !cleaned {
		t.Error("cleanup did not run")
	}
	if got := ExitCodeFor(err); got != ExitInterrupted {
		t.Errorf("ExitCodeFor(%v) = %d, want %d", err, got, ExitInterrupted)
	}
}
