package queue

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// fakeClock lets tests move time forward deterministically; nothing sleeps.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

const testVisibility = 30 * time.Second

func newTestBroker(maxAttempts int) (*Broker, *fakeClock) {
	clk := newFakeClock()
	return New(clk, Config{Visibility: testVisibility, MaxAttempts: maxAttempts}), clk
}

func mustReceive(t *testing.T, b *Broker) Message {
	t.Helper()
	msg, ok := b.Receive()
	if !ok {
		t.Fatal("Receive returned ok=false, want a message")
	}
	return msg
}

func TestPublishAssignsSequentialIDs(t *testing.T) {
	b, _ := newTestBroker(3)
	if id := b.Publish("x"); id != "m1" {
		t.Fatalf("first Publish: got id %q, want %q", id, "m1")
	}
	if id := b.Publish("y"); id != "m2" {
		t.Fatalf("second Publish: got id %q, want %q", id, "m2")
	}
}

func TestPublishReceiveFIFO(t *testing.T) {
	b, _ := newTestBroker(3)
	for _, body := range []string{"a", "b", "c"} {
		b.Publish(body)
	}
	for i, want := range []string{"a", "b", "c"} {
		msg := mustReceive(t, b)
		if msg.Body != want {
			t.Fatalf("delivery %d: got body %q, want %q (FIFO order)", i+1, msg.Body, want)
		}
		if msg.Attempts != 1 {
			t.Fatalf("first delivery of %q: got Attempts=%d, want 1", msg.Body, msg.Attempts)
		}
	}
	if msg, ok := b.Receive(); ok {
		t.Fatalf("queue drained, but Receive returned %+v; in-flight messages must be invisible", msg)
	}
}

func TestAckStopsRedelivery(t *testing.T) {
	b, clk := newTestBroker(3)
	b.Publish("job")
	msg := mustReceive(t, b)
	if err := b.Ack(msg.ID); err != nil {
		t.Fatalf("Ack(%q): unexpected error %v", msg.ID, err)
	}
	clk.Advance(10 * testVisibility)
	if again, ok := b.Receive(); ok {
		t.Fatalf("acked message came back as %+v; Ack must remove it permanently", again)
	}
}

func TestUnknownIDErrors(t *testing.T) {
	b, _ := newTestBroker(3)
	if err := b.Ack("m99"); !errors.Is(err, ErrUnknownID) {
		t.Fatalf("Ack of unknown id: got %v, want an error wrapping ErrUnknownID", err)
	}
	if err := b.Nack("m99"); !errors.Is(err, ErrUnknownID) {
		t.Fatalf("Nack of unknown id: got %v, want an error wrapping ErrUnknownID", err)
	}
	if err := b.DeadLetter("m99"); !errors.Is(err, ErrUnknownID) {
		t.Fatalf("DeadLetter of unknown id: got %v, want an error wrapping ErrUnknownID", err)
	}
}

func TestRedeliveryAfterVisibilityTimeout(t *testing.T) {
	b, clk := newTestBroker(3)
	b.Publish("job")
	first := mustReceive(t, b)

	if _, ok := b.Receive(); ok {
		t.Fatal("message is in flight; it must be invisible until its deadline passes")
	}
	clk.Advance(testVisibility - time.Second)
	if _, ok := b.Receive(); ok {
		t.Fatal("visibility has not expired yet; the message must still be invisible")
	}
	clk.Advance(time.Second)
	second := mustReceive(t, b)
	if second.ID != first.ID || second.Body != "job" {
		t.Fatalf("redelivery: got %+v, want the same message (ID %q)", second, first.ID)
	}
	if second.Attempts != 2 {
		t.Fatalf("redelivery: got Attempts=%d, want 2", second.Attempts)
	}
}

func TestExpiredMessageRequeuesAtBack(t *testing.T) {
	b, clk := newTestBroker(3)
	b.Publish("first")
	b.Publish("second")
	if msg := mustReceive(t, b); msg.Body != "first" {
		t.Fatalf("got %q, want %q first", msg.Body, "first")
	}
	clk.Advance(testVisibility)
	if msg := mustReceive(t, b); msg.Body != "second" {
		t.Fatalf("after expiry: got %q, want %q — expired messages rejoin the BACK of the queue", msg.Body, "second")
	}
	msg := mustReceive(t, b)
	if msg.Body != "first" || msg.Attempts != 2 {
		t.Fatalf("after expiry: got %+v, want body %q with Attempts=2", msg, "first")
	}
}

func TestNackRequeuesImmediately(t *testing.T) {
	b, _ := newTestBroker(3)
	b.Publish("job")
	first := mustReceive(t, b)
	if err := b.Nack(first.ID); err != nil {
		t.Fatalf("Nack(%q): unexpected error %v", first.ID, err)
	}
	second := mustReceive(t, b) // note: the clock has not moved
	if second.ID != first.ID || second.Attempts != 2 {
		t.Fatalf("after Nack: got %+v, want the same message redelivered without waiting for the deadline", second)
	}
}

func TestPoisonMessageDeadLettersAfterMaxAttempts(t *testing.T) {
	const maxAttempts = 3
	b, _ := newTestBroker(maxAttempts)
	b.Publish("poison")
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		msg := mustReceive(t, b)
		if msg.Attempts != attempt {
			t.Fatalf("delivery %d: got Attempts=%d, want %d", attempt, msg.Attempts, attempt)
		}
		if err := b.Nack(msg.ID); err != nil {
			t.Fatalf("Nack on attempt %d: %v", attempt, err)
		}
	}
	if msg, ok := b.Receive(); ok {
		t.Fatalf("got %+v after %d failed attempts; the message must stop being delivered", msg, maxAttempts)
	}
	dead := b.DeadLetters()
	if len(dead) != 1 {
		t.Fatalf("DeadLetters: got %d messages, want 1 — exhausted messages are parked, not deleted", len(dead))
	}
	if dead[0].Body != "poison" || dead[0].Attempts != maxAttempts {
		t.Fatalf("DeadLetters[0] = %+v, want body %q with Attempts=%d", dead[0], "poison", maxAttempts)
	}
}

func TestDeadLetterParksOnTheFirstAttempt(t *testing.T) {
	b, clk := newTestBroker(3)
	b.Publish("not json")
	msg := mustReceive(t, b)
	// The consumer knows this payload will never parse: no retry can help.
	if err := b.DeadLetter(msg.ID); err != nil {
		t.Fatalf("DeadLetter(%q): unexpected error %v", msg.ID, err)
	}
	clk.Advance(10 * testVisibility)
	if again, ok := b.Receive(); ok {
		t.Fatalf("got %+v; a dead-lettered message must never be delivered again, however many attempts it had left", again)
	}
	dead := b.DeadLetters()
	if len(dead) != 1 {
		t.Fatalf("DeadLetters: got %d messages, want 1", len(dead))
	}
	if dead[0].Body != "not json" || dead[0].Attempts != 1 {
		t.Fatalf("DeadLetters[0] = %+v, want body %q with Attempts=1 — parked after a single delivery", dead[0], "not json")
	}
}

func TestExpiryAlsoCountsTowardDeadLettering(t *testing.T) {
	const maxAttempts = 2
	b, clk := newTestBroker(maxAttempts)
	b.Publish("crashy")
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		msg := mustReceive(t, b)
		if msg.Attempts != attempt {
			t.Fatalf("delivery %d: got Attempts=%d, want %d", attempt, msg.Attempts, attempt)
		}
		clk.Advance(testVisibility) // the consumer "crashed": no ack, no nack
	}
	if _, ok := b.Receive(); ok {
		t.Fatal("message exhausted its attempts via expiry; want no further deliveries")
	}
	dead := b.DeadLetters()
	if len(dead) != 1 || dead[0].Body != "crashy" {
		t.Fatalf("DeadLetters: got %v, want just the crashy message", dead)
	}
}

func TestConcurrentPublishIsSafe(t *testing.T) {
	b, _ := newTestBroker(3)
	const n = 50
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Publish(fmt.Sprintf("job-%d", i))
		}()
	}
	wg.Wait()

	bodies := make(map[string]bool)
	ids := make(map[string]bool)
	for range n {
		msg := mustReceive(t, b)
		if bodies[msg.Body] {
			t.Fatalf("body %q delivered twice", msg.Body)
		}
		bodies[msg.Body] = true
		if ids[msg.ID] {
			t.Fatalf("id %q assigned to two messages", msg.ID)
		}
		ids[msg.ID] = true
	}
	if _, ok := b.Receive(); ok {
		t.Fatal("all published messages drained; the queue must be empty")
	}
}
