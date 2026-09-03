package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestExitCodeFor(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"success", nil, ExitOK},
		{"plain failure", errors.New("boom"), ExitFailure},
		{"cancelled", context.Canceled, ExitInterrupted},
		{"deadline", context.DeadlineExceeded, ExitTimeout},
		{"wrapped cancellation", fmt.Errorf("running step: %w", context.Canceled), ExitInterrupted},
		{"wrapped deadline", fmt.Errorf("running step: %w", context.DeadlineExceeded), ExitTimeout},
		{"step error", &StepError{Index: 2, Name: "build", Err: context.Canceled}, ExitInterrupted},
		{"failed step", &StepError{Index: 2, Name: "build", Result: Result{ExitCode: 1}}, ExitFailure},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ExitCodeFor(c.err); got != c.want {
				t.Errorf("ExitCodeFor(%v) = %d, want %d", c.err, got, c.want)
			}
		})
	}
}

func TestWithInterruptInheritsParent(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	ctx, stop := WithInterrupt(parent)
	defer stop()

	cancelParent()
	select {
	case <-ctx.Done():
	case <-time.After(failsafe):
		t.Fatal("cancelling the parent did not cancel the returned context — derive from parent")
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Errorf("ctx.Err() = %v, want context.Canceled", ctx.Err())
	}
}

// failsafe bounds the tests that wait for a context to become done. It is not
// synchronisation — every wait below is expected to finish immediately — it
// just turns "hangs until the test binary times out" into a readable failure.
const failsafe = 3 * time.Second
