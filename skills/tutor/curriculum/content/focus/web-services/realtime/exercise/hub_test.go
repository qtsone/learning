package realtime

import (
	"errors"
	"sync"
	"testing"
)

func TestHubFansOutToEverySubscriber(t *testing.T) {
	h := NewHub(4, 8, nil)

	a, err := h.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	b, err := h.Subscribe()
	if err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	if got := h.Subscribers(); got != 2 {
		t.Fatalf("Subscribers() = %d, want 2", got)
	}

	delivered, evicted := h.Publish(Event{ID: "1", Data: "hello"})
	if delivered != 2 || evicted != 0 {
		t.Fatalf("Publish() = (%d, %d), want (2, 0)", delivered, evicted)
	}
	for name, sub := range map[string]*Subscriber{"a": a, "b": b} {
		if got := recvEvent(t, sub.Events()); got.Data != "hello" {
			t.Errorf("subscriber %s received %+v, want the published event", name, got)
		}
	}
}

func TestHubUnsubscribeIsIdempotent(t *testing.T) {
	h := NewHub(1, 0, nil)
	sub, _ := h.Subscribe()

	h.Unsubscribe(sub)
	if got := h.Subscribers(); got != 0 {
		t.Fatalf("Subscribers() = %d after Unsubscribe, want 0", got)
	}
	expectClosed(t, sub.Events())

	// Handlers defer Unsubscribe and cannot know whether the hub already
	// dropped them, so a second call must not panic on a double close.
	h.Unsubscribe(sub)
	h.Unsubscribe(nil)

	if delivered, _ := h.Publish(Event{Data: "x"}); delivered != 0 {
		t.Errorf("Publish() delivered %d events after Unsubscribe, want 0", delivered)
	}
}

// The whole point of a bounded buffer: a subscriber that stops reading is
// dropped, so one stalled client cannot make the publisher wait.
func TestHubEvictsASlowSubscriber(t *testing.T) {
	h := NewHub(2, 8, quietLogger())
	slow, _ := h.Subscribe()
	fast, _ := h.Subscribe()

	// Fill the buffers: two events fit, nobody has read anything.
	for i, id := range []string{"1", "2"} {
		if delivered, evicted := h.Publish(Event{ID: id, Data: id}); delivered != 2 || evicted != 0 {
			t.Fatalf("Publish() %d = (%d, %d), want (2, 0): a buffer of 2 holds two events", i+1, delivered, evicted)
		}
	}
	// Drain the fast one only. The third event has nowhere to go for slow.
	recvEvent(t, fast.Events())
	recvEvent(t, fast.Events())

	delivered, evicted := h.Publish(Event{ID: "3", Data: "3"})
	if delivered != 1 || evicted != 1 {
		t.Fatalf("Publish() = (%d, %d), want (1, 1): the full subscriber is evicted, the drained one is served", delivered, evicted)
	}
	if got := h.Subscribers(); got != 1 {
		t.Errorf("Subscribers() = %d, want 1: an evicted subscriber is removed from the hub", got)
	}
	expectClosed(t, slow.Events())
	if got := recvEvent(t, fast.Events()); got.ID != "3" {
		t.Errorf("the fast subscriber received %q, want event 3", got.ID)
	}
}

func TestHubReplaysFromTheBacklog(t *testing.T) {
	h := NewHub(4, 3, nil)
	for _, id := range []string{"1", "2", "3", "4"} {
		h.Publish(Event{ID: id, Data: id})
	}

	missed, ok := h.Since("2")
	if !ok {
		t.Fatal("Since(\"2\") not found: the backlog keeps the last 3 identified events, which are 2, 3 and 4")
	}
	if len(missed) != 2 || missed[0].ID != "3" || missed[1].ID != "4" {
		t.Fatalf("Since(\"2\") = %+v, want events 3 and 4, in order", missed)
	}

	if _, ok := h.Since("1"); ok {
		t.Error("Since(\"1\") reported a hit: event 1 has fallen out of a backlog of 3, and pretending otherwise hides a gap from the client")
	}
	if _, ok := h.Since(""); ok {
		t.Error("Since(\"\") reported a hit: a client that never saw an id has nothing to replay")
	}

	// Unidentified events are not replayable, so they do not enter the backlog.
	h.Publish(Event{Data: "anonymous"})
	if missed, _ := h.Since("3"); len(missed) != 1 || missed[0].ID != "4" {
		t.Errorf("Since(\"3\") = %+v, want only event 4", missed)
	}
}

func TestHubShutdownEndsEverySubscription(t *testing.T) {
	h := NewHub(4, 4, nil)
	a, _ := h.Subscribe()
	b, _ := h.Subscribe()

	h.Shutdown()

	expectClosed(t, a.Events())
	expectClosed(t, b.Events())
	if got := h.Subscribers(); got != 0 {
		t.Errorf("Subscribers() = %d after Shutdown, want 0", got)
	}
	if _, err := h.Subscribe(); !errors.Is(err, ErrHubClosed) {
		t.Errorf("Subscribe() after Shutdown returned %v, want ErrHubClosed", err)
	}
	if delivered, evicted := h.Publish(Event{Data: "x"}); delivered != 0 || evicted != 0 {
		t.Errorf("Publish() = (%d, %d) after Shutdown, want (0, 0)", delivered, evicted)
	}
	h.Shutdown() // must not panic on a second close
}

// Every method is reachable from a different request goroutine at the same
// time. The interesting failure is not a wrong count, it is a send on a closed
// channel: closing and sending must happen under one lock.
func TestHubIsSafeForConcurrentUse(t *testing.T) {
	h := NewHub(1, 16, quietLogger())

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				sub, err := h.Subscribe()
				if err != nil {
					return
				}
				select {
				case <-sub.Events():
				default:
				}
				h.Unsubscribe(sub)
			}
		}()
	}
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				h.Publish(Event{ID: "x", Data: "x"})
				h.Since("x")
				h.Subscribers()
			}
		}()
	}
	wg.Wait()

	if got := h.Subscribers(); got != 0 {
		t.Errorf("Subscribers() = %d after every goroutine unsubscribed, want 0", got)
	}
}
