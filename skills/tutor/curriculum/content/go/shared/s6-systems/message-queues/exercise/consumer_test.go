package queue

import (
	"errors"
	"testing"
)

func TestHandleRunsSideEffectOncePerMessageID(t *testing.T) {
	calls := 0
	c := NewIdempotentConsumer(func(Message) error {
		calls++
		return nil
	}, nil) // nil key: dedupe on the broker's message ID
	msg := Message{ID: "m1", Body: "charge card", Attempts: 1}

	if err := c.Handle(msg); err != nil {
		t.Fatalf("first Handle: unexpected error %v", err)
	}
	if calls != 1 {
		t.Fatalf("after first Handle: side effect ran %d times, want 1", calls)
	}

	redelivered := msg
	redelivered.Attempts = 2
	if err := c.Handle(redelivered); err != nil {
		t.Fatalf("duplicate Handle: got %v, want nil — a duplicate is safe to ack", err)
	}
	if calls != 1 {
		t.Fatalf("after duplicate Handle: side effect ran %d times, want 1 — redelivery must not repeat it", calls)
	}
}

func TestHandleProcessesDistinctMessages(t *testing.T) {
	var bodies []string
	c := NewIdempotentConsumer(func(m Message) error {
		bodies = append(bodies, m.Body)
		return nil
	}, ByMessageID)
	for _, m := range []Message{
		{ID: "m1", Body: "a", Attempts: 1},
		{ID: "m2", Body: "b", Attempts: 1},
	} {
		if err := c.Handle(m); err != nil {
			t.Fatalf("Handle(%q): unexpected error %v", m.ID, err)
		}
	}
	if len(bodies) != 2 || bodies[0] != "a" || bodies[1] != "b" {
		t.Fatalf("processed %v, want [a b] — dedupe is per message ID, not global", bodies)
	}
}

func TestHandleDedupesOnTheKeyNotTheMessageID(t *testing.T) {
	calls := 0
	c := NewIdempotentConsumer(func(Message) error {
		calls++
		return nil
	}, func(m Message) string { return m.Body })

	for _, id := range []string{"m1", "m2"} {
		if err := c.Handle(Message{ID: id, Body: "charge order 42", Attempts: 1}); err != nil {
			t.Fatalf("Handle(%q): unexpected error %v", id, err)
		}
	}
	if calls != 1 {
		t.Fatalf("side effect ran %d times, want 1 — two broker messages, one piece of work: the key decides, not msg.ID", calls)
	}
}

func TestFailedProcessingIsRetriedOnRedelivery(t *testing.T) {
	calls := 0
	errDownstream := errors.New("downstream unavailable")
	c := NewIdempotentConsumer(func(Message) error {
		calls++
		if calls == 1 {
			return errDownstream
		}
		return nil
	}, ByMessageID)
	msg := Message{ID: "m1", Body: "send email", Attempts: 1}

	if err := c.Handle(msg); !errors.Is(err, errDownstream) {
		t.Fatalf("first Handle: got %v, want the processor's error surfaced", err)
	}
	if err := c.Handle(msg); err != nil {
		t.Fatalf("redelivered Handle: got %v, want nil — a FAILED attempt must not be recorded as done", err)
	}
	if calls != 2 {
		t.Fatalf("processor ran %d times, want 2: record the ID only on success", calls)
	}
	if err := c.Handle(msg); err != nil || calls != 2 {
		t.Fatalf("third Handle: err=%v, calls=%d; want nil and 2 — now it is done", err, calls)
	}
}

func TestRedeliveryEndToEndDoesNotDuplicateSideEffects(t *testing.T) {
	b, clk := newTestBroker(5)
	shipped := 0
	// One consumer value stands in for the shared, durable dedupe store that
	// every consumer replica would read in a real system.
	c := NewIdempotentConsumer(func(Message) error {
		shipped++
		return nil
	}, ByMessageID)

	b.Publish("ship order 42")

	// The first worker processes the message but dies before acking.
	first := mustReceive(t, b)
	if err := c.Handle(first); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	// no Ack — the worker crashed here

	clk.Advance(testVisibility)

	// The broker redelivers; the replacement worker must not ship twice.
	second := mustReceive(t, b)
	if second.ID != first.ID {
		t.Fatalf("redelivery: got ID %q, want %q", second.ID, first.ID)
	}
	if err := c.Handle(second); err != nil {
		t.Fatalf("Handle of redelivery: %v", err)
	}
	if err := b.Ack(second.ID); err != nil {
		t.Fatalf("Ack: %v", err)
	}

	if shipped != 1 {
		t.Fatalf("order shipped %d times, want exactly 1 — at-least-once delivery plus an idempotent consumer", shipped)
	}
	clk.Advance(10 * testVisibility)
	if _, ok := b.Receive(); ok {
		t.Fatal("acked message must never come back")
	}
}
