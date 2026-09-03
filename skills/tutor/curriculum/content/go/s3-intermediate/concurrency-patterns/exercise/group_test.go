package patterns

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestGroupAllSucceed(t *testing.T) {
	g, ctx := WithContext(context.Background())
	var calls atomic.Int32
	for range 5 {
		g.Go(func() error {
			calls.Add(1)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		t.Fatalf("Wait() = %v, want nil when every function succeeds", err)
	}
	if n := calls.Load(); n != 5 {
		t.Errorf("%d of 5 functions ran — Go must run every function", n)
	}
	select {
	case <-ctx.Done():
	default:
		t.Error("group context is still live after Wait — Wait must cancel it so the success path doesn't leak")
	}
}

func TestGroupFirstErrorWinsAndWaitsForAll(t *testing.T) {
	boom := errors.New("boom")
	g, ctx := WithContext(context.Background())
	var lateFinished atomic.Bool

	g.Go(func() error { return boom })
	g.Go(func() error {
		// Runs until the first error cancels ctx, then errors too — late
		// errors must be dropped, and Wait must still wait for this one.
		select {
		case <-ctx.Done():
		case <-time.After(2 * time.Second):
		}
		lateFinished.Store(true)
		return errors.New("late error, must not win")
	})

	err := g.Wait()
	if !errors.Is(err, boom) {
		t.Errorf("Wait() = %v, want the FIRST error %v", err, boom)
	}
	if !lateFinished.Load() {
		t.Error("Wait returned before every function finished — it must wait for all of them, errors included")
	}
}

func TestGroupCancelsSiblingsOnFirstError(t *testing.T) {
	g, ctx := WithContext(context.Background())
	var sawCancel atomic.Bool

	g.Go(func() error { return errors.New("early failure") })
	g.Go(func() error {
		select {
		case <-ctx.Done():
			sawCancel.Store(true)
		case <-time.After(2 * time.Second):
		}
		return nil
	})

	if err := g.Wait(); err == nil {
		t.Error("Wait() = nil, want the error from the failing function")
	}
	if !sawCancel.Load() {
		t.Error("sibling never saw ctx.Done() — the first error must cancel the group context")
	}
}
