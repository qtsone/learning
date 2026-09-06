package scalability

import "testing"

func TestBoundedFIFO(t *testing.T) {
	q := NewBounded[string](3)
	for _, v := range []string{"a", "b", "c"} {
		if !q.Offer(v) {
			t.Fatalf("Offer(%q) = false with room to spare", v)
		}
	}
	for _, want := range []string{"a", "b", "c"} {
		got, ok := q.Take()
		if !ok || got != want {
			t.Fatalf("Take() = %q, %v; want %q, true (FIFO order)", got, ok, want)
		}
	}
	if _, ok := q.Take(); ok {
		t.Fatal("Take() on an empty queue reported ok=true")
	}
}

func TestBoundedRefusesWhenFull(t *testing.T) {
	q := NewBounded[int](2)
	q.Offer(1)
	q.Offer(2)
	if q.Offer(3) {
		t.Fatal("Offer succeeded on a full queue; a bounded queue must refuse, not grow")
	}
	if q.Len() != 2 {
		t.Fatalf("Len() = %d after a refused Offer; want 2", q.Len())
	}
	q.Take()
	if !q.Offer(3) {
		t.Fatal("Offer failed after Take freed a slot")
	}
}

func TestSimulateHeadroom(t *testing.T) {
	got := SimulateLoad(10, 5, 8, 20)
	want := LoadResult{Served: 100, Shed: 0, MaxQueue: 5, Backlog: 0}
	if got != want {
		t.Fatalf("SimulateLoad(10, 5, 8, 20) = %+v; want %+v — with headroom every arrival is served", got, want)
	}
}

func TestSimulateOverloadBounded(t *testing.T) {
	got := SimulateLoad(50, 10, 8, 100)
	want := LoadResult{Served: 800, Shed: 158, MaxQueue: 50, Backlog: 42}
	if got != want {
		t.Fatalf("SimulateLoad(50, 10, 8, 100) = %+v; want %+v — "+
			"the queue caps at 50 and the overflow is shed", got, want)
	}
}

func TestSimulateOverloadUnbounded(t *testing.T) {
	got := SimulateLoad(1_000_000, 10, 8, 100)
	want := LoadResult{Served: 800, Shed: 0, MaxQueue: 208, Backlog: 200}
	if got != want {
		t.Fatalf("SimulateLoad(1_000_000, 10, 8, 100) = %+v; want %+v — "+
			"nothing is shed, so the backlog absorbs the whole overload", got, want)
	}
}

func TestBoundedBacklogStaysFlatUnboundedGrows(t *testing.T) {
	bounded100 := SimulateLoad(50, 10, 8, 100)
	bounded1000 := SimulateLoad(50, 10, 8, 1000)
	if bounded100.Backlog != bounded1000.Backlog {
		t.Fatalf("bounded backlog changed with run length: %d after 100 ticks, %d after 1000 — "+
			"a bounded queue reaches a steady state", bounded100.Backlog, bounded1000.Backlog)
	}

	unbounded100 := SimulateLoad(1_000_000, 10, 8, 100)
	unbounded1000 := SimulateLoad(1_000_000, 10, 8, 1000)
	if unbounded1000.Backlog != 10*unbounded100.Backlog {
		t.Fatalf("unbounded backlog after 1000 ticks = %d; want 10× the 100-tick backlog (%d) — "+
			"without a bound, backlog (and so latency) grows linearly with time under overload",
			unbounded1000.Backlog, unbounded100.Backlog)
	}
}
