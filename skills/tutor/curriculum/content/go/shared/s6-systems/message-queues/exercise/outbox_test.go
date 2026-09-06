package queue

import (
	"errors"
	"testing"
)

func TestExecTxCommitsStateAndEventsTogether(t *testing.T) {
	s := NewStore()
	evt := Event{ID: "evt-1", Payload: "order 42 placed"}
	err := s.ExecTx(func(tx *Tx) error {
		tx.Set("order:42", "placed")
		tx.Emit(evt)
		return nil
	})
	if err != nil {
		t.Fatalf("ExecTx: unexpected error %v", err)
	}

	got, ok := s.Get("order:42")
	if !ok || got != "placed" {
		t.Fatalf("Get(order:42) = %q, %v; want %q, true — committed writes must be visible", got, ok, "placed")
	}
	b, _ := newTestBroker(3)
	if n := s.Relay(b); n != 1 {
		t.Fatalf("Relay published %d events, want 1 — the event committed with the write", n)
	}
	msg := mustReceive(t, b)
	if msg.Body != evt.Wire() {
		t.Fatalf("relayed message body = %q, want %q — the event ID rides along with the payload", msg.Body, evt.Wire())
	}
}

func TestExecTxRollsBackBothOnError(t *testing.T) {
	s := NewStore()
	errFunds := errors.New("insufficient funds")
	err := s.ExecTx(func(tx *Tx) error {
		tx.Set("order:43", "placed")
		tx.Emit(Event{ID: "evt-2", Payload: "order 43 placed"})
		return errFunds
	})
	if !errors.Is(err, errFunds) {
		t.Fatalf("ExecTx: got %v, want the transaction's own error returned", err)
	}

	if v, ok := s.Get("order:43"); ok {
		t.Fatalf("Get(order:43) = %q after rollback; the write must not be visible", v)
	}
	b, _ := newTestBroker(3)
	if n := s.Relay(b); n != 0 {
		t.Fatalf("Relay published %d events after rollback, want 0 — no state change, no event", n)
	}
}

func TestRelayPublishesEachEventOnceInOrder(t *testing.T) {
	s := NewStore()
	b, _ := newTestBroker(3)
	for i, payload := range []string{"e1", "e2"} {
		if err := s.ExecTx(func(tx *Tx) error {
			tx.Emit(Event{ID: payload, Payload: payload})
			return nil
		}); err != nil {
			t.Fatalf("ExecTx %d: %v", i+1, err)
		}
	}

	if n := s.Relay(b); n != 2 {
		t.Fatalf("first Relay published %d events, want 2", n)
	}
	if n := s.Relay(b); n != 0 {
		t.Fatalf("second Relay published %d events, want 0 — already-published events must not repeat", n)
	}
	for _, want := range []string{"e1", "e2"} {
		msg := mustReceive(t, b)
		if id := EventID(msg); id != want {
			t.Fatalf("relayed event id = %q, want %q (commit order)", id, want)
		}
	}

	if err := s.ExecTx(func(tx *Tx) error {
		tx.Emit(Event{ID: "e3", Payload: "e3"})
		return nil
	}); err != nil {
		t.Fatalf("ExecTx: %v", err)
	}
	if n := s.Relay(b); n != 1 {
		t.Fatalf("Relay after a new commit published %d events, want 1", n)
	}
}

func TestOutboxPipelineEndToEnd(t *testing.T) {
	s := NewStore()
	b, clk := newTestBroker(3)
	notified := 0
	c := NewIdempotentConsumer(func(Message) error {
		notified++
		return nil
	}, EventID)

	if err := s.ExecTx(func(tx *Tx) error {
		tx.Set("order:7", "paid")
		tx.Emit(Event{ID: "evt-7", Payload: "order 7 paid"})
		return nil
	}); err != nil {
		t.Fatalf("ExecTx: %v", err)
	}
	if n := s.Relay(b); n != 1 {
		t.Fatalf("Relay published %d events, want 1", n)
	}

	// First delivery is processed, but the worker dies before acking.
	first := mustReceive(t, b)
	if err := c.Handle(first); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	clk.Advance(testVisibility)

	// Redelivery: the idempotent consumer absorbs the duplicate; ack closes it.
	second := mustReceive(t, b)
	if err := c.Handle(second); err != nil {
		t.Fatalf("Handle of redelivery: %v", err)
	}
	if err := b.Ack(second.ID); err != nil {
		t.Fatalf("Ack: %v", err)
	}

	if notified != 1 {
		t.Fatalf("notification sent %d times, want exactly 1", notified)
	}
	if v, _ := s.Get("order:7"); v != "paid" {
		t.Fatalf("Get(order:7) = %q, want %q", v, "paid")
	}
	if dead := b.DeadLetters(); len(dead) != 0 {
		t.Fatalf("DeadLetters = %v, want empty — this pipeline succeeded", dead)
	}
}

func TestRelayCrashBeforeMarkDoesNotDuplicateSideEffects(t *testing.T) {
	s := NewStore()
	b, _ := newTestBroker(3)
	notified := 0
	c := NewIdempotentConsumer(func(Message) error {
		notified++
		return nil
	}, EventID)

	if err := s.ExecTx(func(tx *Tx) error {
		tx.Set("order:9", "paid")
		tx.Emit(Event{ID: "evt-9", Payload: "order 9 paid"})
		return nil
	}); err != nil {
		t.Fatalf("ExecTx: %v", err)
	}
	if n := s.Relay(b); n != 1 {
		t.Fatalf("Relay published %d events, want 1", n)
	}

	// The relay published the row and then died before marking it published.
	// The outbox still says "unpublished", so the restarted relay sends it
	// again: this is the duplicate the outbox pattern buys atomicity with.
	s.published = 0
	if n := s.Relay(b); n != 1 {
		t.Fatalf("relay restart published %d events, want 1 — the row was never marked", n)
	}

	first, second := mustReceive(t, b), mustReceive(t, b)
	if first.ID == second.ID {
		t.Fatalf("got message %q twice; a republished event is a NEW broker message with a new ID", first.ID)
	}
	for _, msg := range []Message{first, second} {
		if err := c.Handle(msg); err != nil {
			t.Fatalf("Handle(%q): %v", msg.ID, err)
		}
		if err := b.Ack(msg.ID); err != nil {
			t.Fatalf("Ack(%q): %v", msg.ID, err)
		}
	}
	if notified != 1 {
		t.Fatalf("notification sent %d times, want 1 — two broker messages carry one business event, so dedupe on the event ID, not msg.ID", notified)
	}
}
