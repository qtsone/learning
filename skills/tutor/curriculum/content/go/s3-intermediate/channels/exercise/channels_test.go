package channels

import (
	"slices"
	"testing"
	"time"
)

// patience bounds every wait in this file. The tests are deterministic —
// nothing here sleeps to synchronize; the timeout only converts a deadlock
// or a missing close into a readable failure instead of a hung test run.
const patience = 2 * time.Second

// returnsPromptly calls fn on another goroutine and fails the test if it
// does not return within patience — the symptom of a constructor doing its
// sends synchronously instead of inside a goroutine.
func returnsPromptly(t *testing.T, name string, fn func() <-chan int) <-chan int {
	t.Helper()
	result := make(chan (<-chan int), 1)
	go func() { result <- fn() }()
	select {
	case ch := <-result:
		return ch
	case <-time.After(patience):
		t.Fatalf("%s blocked instead of returning — start the sends in a goroutine inside it", name)
		return nil
	}
}

// collect drains ch, failing the test if it does not close within patience —
// the classic symptom of a sender that forgot to close its channel.
func collect(t *testing.T, ch <-chan int) []int {
	t.Helper()
	var got []int
	timeout := time.After(patience)
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				return got
			}
			got = append(got, v)
		case <-timeout:
			t.Fatalf("channel did not close within %v (received %v so far) — does the sender close it when done?", patience, got)
		}
	}
}

func TestGenerate(t *testing.T) {
	ch := returnsPromptly(t, "Generate", func() <-chan int { return Generate(1, 2, 3) })
	if got, want := collect(t, ch), []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("Generate(1, 2, 3) emitted %v, want %v", got, want)
	}
}

func TestGenerateEmpty(t *testing.T) {
	ch := returnsPromptly(t, "Generate", func() <-chan int { return Generate() })
	if got := collect(t, ch); len(got) != 0 {
		t.Errorf("Generate() emitted %v, want no values before closing", got)
	}
}

func TestSquare(t *testing.T) {
	in := make(chan int, 4)
	for _, v := range []int{1, 2, 3, 4} {
		in <- v
	}
	close(in)
	out := returnsPromptly(t, "Square", func() <-chan int { return Square(in) })
	if got, want := collect(t, out), []int{1, 4, 9, 16}; !slices.Equal(got, want) {
		t.Errorf("Square over 1..4 emitted %v, want %v", got, want)
	}
}

func TestSum(t *testing.T) {
	in := make(chan int, 3)
	in <- 2
	in <- 4
	in <- 6
	close(in)
	got := make(chan int, 1)
	go func() { got <- Sum(in) }()
	select {
	case total := <-got:
		if total != 12 {
			t.Errorf("Sum over 2, 4, 6 = %d, want 12", total)
		}
	case <-time.After(patience):
		t.Fatal("Sum never returned — range over the channel; the loop ends when the sender closes it")
	}
}

func TestPipeline(t *testing.T) {
	got := make(chan int, 1)
	go func() { got <- Sum(Square(Generate(1, 2, 3))) }()
	select {
	case total := <-got:
		if total != 14 {
			t.Errorf("Sum(Square(Generate(1, 2, 3))) = %d, want 14", total)
		}
	case <-time.After(patience):
		t.Fatal("pipeline never finished — check that every stage closes its output channel")
	}
}

func TestTryRecv(t *testing.T) {
	// tryRecv guards each call so a blocking implementation fails with a
	// message instead of hanging the test run.
	tryRecv := func(t *testing.T, ch <-chan int, situation string) (int, bool) {
		t.Helper()
		type result struct {
			v  int
			ok bool
		}
		res := make(chan result, 1)
		go func() {
			v, ok := TryRecv(ch)
			res <- result{v, ok}
		}()
		select {
		case r := <-res:
			return r.v, r.ok
		case <-time.After(patience):
			t.Fatalf("TryRecv blocked on %s — that is what the default case prevents", situation)
			return 0, false
		}
	}

	t.Run("value ready", func(t *testing.T) {
		ch := make(chan int, 1)
		ch <- 42
		if v, ok := tryRecv(t, ch, "a channel with a value ready"); !ok || v != 42 {
			t.Errorf("TryRecv = (%d, %t), want (42, true)", v, ok)
		}
	})
	t.Run("empty channel", func(t *testing.T) {
		if v, ok := tryRecv(t, make(chan int), "an empty channel"); ok || v != 0 {
			t.Errorf("TryRecv on an empty channel = (%d, %t), want (0, false)", v, ok)
		}
	})
	t.Run("nil channel", func(t *testing.T) {
		if v, ok := tryRecv(t, nil, "a nil channel"); ok || v != 0 {
			t.Errorf("TryRecv on a nil channel = (%d, %t), want (0, false)", v, ok)
		}
	})
	t.Run("closed and drained", func(t *testing.T) {
		ch := make(chan int)
		close(ch)
		if v, ok := tryRecv(t, ch, "a closed channel"); ok || v != 0 {
			t.Errorf("TryRecv on a closed, drained channel = (%d, %t), want (0, false): its zero values are not real data", v, ok)
		}
	})
	t.Run("closed with a buffered value", func(t *testing.T) {
		ch := make(chan int, 1)
		ch <- 7
		close(ch)
		if v, ok := tryRecv(t, ch, "a closed channel holding a value"); !ok || v != 7 {
			t.Errorf("TryRecv = (%d, %t), want (7, true): closed channels still deliver buffered leftovers", v, ok)
		}
	})
}

func TestCounter(t *testing.T) {
	done := make(chan struct{})
	ch := returnsPromptly(t, "Counter", func() <-chan int { return Counter(done) })

	timeout := time.After(patience)
	for want := 1; want <= 5; want++ {
		select {
		case v, ok := <-ch:
			if !ok {
				t.Fatalf("Counter closed after %d value(s); it must keep counting until done is closed", want-1)
			}
			if v != want {
				t.Fatalf("value #%d = %d, want %d", want, v, want)
			}
		case <-timeout:
			t.Fatalf("Counter stopped emitting after %d value(s) with done still open", want-1)
		}
	}

	close(done)

	// Counter's select may already be committed to one more send when done
	// closes; stragglers are fine, but the channel must close.
	deadline := time.After(patience)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatal("Counter's channel never closed after done was closed — select on done and close the output")
		}
	}
}

func TestMergeTwo(t *testing.T) {
	a := make(chan int, 3)
	for _, v := range []int{1, 2, 3} {
		a <- v
	}
	close(a)
	b := make(chan int, 2)
	b <- 10
	b <- 20
	close(b)

	out := returnsPromptly(t, "MergeTwo", func() <-chan int { return MergeTwo(a, b) })
	got := collect(t, out)
	slices.Sort(got)
	if want := []int{1, 2, 3, 10, 20}; !slices.Equal(got, want) {
		t.Errorf("MergeTwo delivered %v (sorted), want %v", got, want)
	}
}

func TestMergeTwoOneSideEmpty(t *testing.T) {
	a := make(chan int)
	close(a)
	b := make(chan int, 1)
	b <- 7
	close(b)

	out := returnsPromptly(t, "MergeTwo", func() <-chan int { return MergeTwo(a, b) })
	if got, want := collect(t, out), []int{7}; !slices.Equal(got, want) {
		t.Errorf("MergeTwo with one closed-empty input delivered %v, want %v", got, want)
	}
}

// A single producer alternates between the two inputs over unbuffered
// channels. An implementation that drains a completely before touching b
// deadlocks here (the producer is stuck sending on b); a select loop copes.
func TestMergeTwoInterleavedProducer(t *testing.T) {
	a := make(chan int)
	b := make(chan int)
	go func() {
		a <- 1
		b <- 2
		a <- 3
		close(a)
		close(b)
	}()

	out := returnsPromptly(t, "MergeTwo", func() <-chan int { return MergeTwo(a, b) })
	got := collect(t, out)
	slices.Sort(got)
	if want := []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("MergeTwo delivered %v (sorted), want %v", got, want)
	}
}
