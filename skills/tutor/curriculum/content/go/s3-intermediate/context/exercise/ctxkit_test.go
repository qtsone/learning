package ctxkit

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestAwait(t *testing.T) {
	t.Run("value arrives", func(t *testing.T) {
		ch := make(chan string, 1)
		ch <- "payload"
		got, err := Await(context.Background(), ch)
		if err != nil {
			t.Fatalf("Await with value ready returned error %v, want nil", err)
		}
		if got != "payload" {
			t.Fatalf("Await = %q, want %q", got, "payload")
		}
	})

	t.Run("context canceled first", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		got, err := Await(ctx, make(chan string))
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Await on canceled context returned err %v, want context.Canceled", err)
		}
		if got != "" {
			t.Fatalf("Await on canceled context returned %q, want zero value", got)
		}
	})
}

func TestSquareDrainsClosedJobs(t *testing.T) {
	jobs := make(chan int, 3)
	for _, j := range []int{1, 2, 3} {
		jobs <- j
	}
	close(jobs)
	results := make(chan int, 3)

	if err := Square(context.Background(), jobs, results); err != nil {
		t.Fatalf("Square with closed jobs returned %v, want nil", err)
	}
	for i, want := range []int{1, 4, 9} {
		select {
		case got := <-results:
			if got != want {
				t.Fatalf("result %d = %d, want %d", i, got, want)
			}
		default:
			t.Fatalf("results has %d value(s), want 3 — did Square process every job?", i)
		}
	}
	select {
	case extra := <-results:
		t.Fatalf("results has extra value %d after the 3 expected", extra)
	default:
	}
}

func TestSquareStopsOnCancelWhileWaitingForJobs(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	jobs := make(chan int)
	results := make(chan int)
	done := make(chan error, 1)
	go func() { done <- Square(ctx, jobs, results) }()

	for _, j := range []int{2, 3} {
		select {
		case jobs <- j:
		case err := <-done:
			t.Fatalf("worker exited with %v before accepting job %d", err, j)
		}
		select {
		case got := <-results:
			if got != j*j {
				t.Fatalf("result for job %d = %d, want %d", j, got, j*j)
			}
		case err := <-done:
			t.Fatalf("worker exited with %v instead of sending result %d", err, j*j)
		}
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("after cancel, worker returned %v, want context.Canceled", err)
		}
	case <-watchdog(t):
		t.Fatal("worker did not return after cancel — does the receive select include ctx.Done()?")
	}
}

func TestSquareStopsOnCancelWhileSendingResult(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	jobs := make(chan int, 1)
	jobs <- 7
	results := make(chan int) // never drained: the send must block
	done := make(chan error, 1)
	go func() { done <- Square(ctx, jobs, results) }()

	// The worker takes job 7 and blocks sending 49 into an undrained
	// channel. Cancellation is its only way out.
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("after cancel during send, worker returned %v, want context.Canceled", err)
		}
	case <-watchdog(t):
		t.Fatal("worker did not return after cancel — is the results send guarded by ctx.Done()?")
	}
}

// watchdog returns a channel that fires long after any correct solution
// has finished. It is a failure detector for goroutines that ignore
// cancellation — never a synchronization mechanism.
func watchdog(t *testing.T) <-chan time.Time {
	t.Helper()
	timer := time.NewTimer(2 * time.Second)
	t.Cleanup(func() { timer.Stop() })
	return timer.C
}

func TestRetryable(t *testing.T) {
	expiredCtx, cancel := context.WithDeadline(context.Background(),
		time.Now().Add(-time.Minute))
	defer cancel()
	<-expiredCtx.Done()

	canceledCtx, cancelNow := context.WithCancel(context.Background())
	cancelNow()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"deadline exceeded", expiredCtx.Err(), true},
		{"canceled", canceledCtx.Err(), false},
		{"wrapped deadline exceeded",
			fmt.Errorf("fetch user 42: %w", context.DeadlineExceeded), true},
		{"wrapped canceled",
			fmt.Errorf("fetch user 42: %w", context.Canceled), false},
		{"unrelated error", errors.New("connection refused"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Retryable(c.err); got != c.want {
				t.Errorf("Retryable(%v) = %t, want %t", c.err, got, c.want)
			}
		})
	}
}

func TestRequestID(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		ctx := WithRequestID(context.Background(), "req-42")
		id, ok := RequestID(ctx)
		if !ok || id != "req-42" {
			t.Fatalf("RequestID = %q, %t, want %q, true", id, ok, "req-42")
		}
	})

	t.Run("absent", func(t *testing.T) {
		id, ok := RequestID(context.Background())
		if ok || id != "" {
			t.Fatalf("RequestID on bare context = %q, %t, want %q, false", id, ok, "")
		}
	})

	t.Run("survives derived contexts", func(t *testing.T) {
		ctx, cancel := context.WithCancel(WithRequestID(context.Background(), "req-7"))
		defer cancel()
		id, ok := RequestID(ctx)
		if !ok || id != "req-7" {
			t.Fatalf("RequestID through derived context = %q, %t, want %q, true",
				id, ok, "req-7")
		}
	})

	t.Run("no collision with foreign keys", func(t *testing.T) {
		// Simulates another package storing a value under a look-alike
		// key. With an unexported key type, this cannot reach RequestID.
		type foreignKey string
		ctx := context.WithValue(context.Background(),
			foreignKey("request-id"), "spoofed")
		if id, ok := RequestID(ctx); ok {
			t.Fatalf("RequestID sees a foreign package's value %q — use an unexported key type", id)
		}
	})
}
