package closures

import "testing"

func TestCounterIncrements(t *testing.T) {
	inc, _ := Counter()
	for want := 1; want <= 3; want++ {
		if got := inc(); got != want {
			t.Fatalf("inc() call %d = %d, want %d", want, got, want)
		}
	}
}

func TestCounterResetSharesState(t *testing.T) {
	inc, reset := Counter()
	inc()
	inc()
	reset()
	if got := inc(); got != 1 {
		t.Errorf("inc() after reset() = %d, want 1 (reset must act on the same captured count as inc)", got)
	}
}

func TestCountersAreIndependent(t *testing.T) {
	incA, _ := Counter()
	incB, _ := Counter()
	incA()
	incA()
	if got := incB(); got != 1 {
		t.Errorf("second counter's inc() = %d, want 1 (each Counter() call must own fresh state)", got)
	}
}
