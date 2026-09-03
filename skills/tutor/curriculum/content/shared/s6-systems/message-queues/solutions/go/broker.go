package queue

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Message is one unit of work traveling through the broker.
type Message struct {
	ID       string
	Body     string
	Attempts int // deliveries so far, including the current one
}

// ErrUnknownID reports an Ack or Nack for a message that is not in flight.
var ErrUnknownID = errors.New("queue: message not in flight")

// Config tunes redelivery behavior.
type Config struct {
	// Visibility is how long a received message stays invisible before it
	// may be redelivered.
	Visibility time.Duration
	// MaxAttempts is the total number of deliveries a message gets before
	// it is dead-lettered.
	MaxAttempts int
}

type inFlightMessage struct {
	msg      Message
	deadline time.Time // when visibility expires and redelivery is allowed
}

// Broker is a tiny in-process message broker with at-least-once delivery.
// It is passive: it starts no goroutines and holds no timers — all
// redelivery bookkeeping happens inside Receive, using the injected Clock.
type Broker struct {
	mu     sync.Mutex
	clock  Clock
	cfg    Config
	nextID int

	ready    []Message         // awaiting delivery, FIFO
	inFlight []inFlightMessage // delivered, awaiting ack; oldest delivery first
	dead     []Message         // dead-lettered, in arrival order
}

// New returns an empty broker using clock for all time decisions.
func New(clock Clock, cfg Config) *Broker {
	return &Broker{clock: clock, cfg: cfg}
}

// Publish enqueues body for delivery and returns the new message's ID:
// "m1", "m2", … in publish order.
func (b *Broker) Publish(body string) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	id := fmt.Sprintf("m%d", b.nextID)
	b.ready = append(b.ready, Message{ID: id, Body: body})
	return id
}

// Receive returns the next deliverable message, or ok == false when nothing
// is ready. Before selecting, it sweeps in-flight messages whose visibility
// deadline has passed (deadline <= now), oldest delivery first: those with
// attempts remaining rejoin the back of the ready queue; those that have
// used MaxAttempts deliveries move to the dead-letter queue. The returned
// message has Attempts incremented and stays in flight until Ack, Nack, or
// its new deadline (now + Visibility).
func (b *Broker) Receive() (Message, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sweepExpired()
	if len(b.ready) == 0 {
		return Message{}, false
	}
	msg := b.ready[0]
	b.ready = b.ready[1:]
	msg.Attempts++
	b.inFlight = append(b.inFlight, inFlightMessage{
		msg:      msg,
		deadline: b.clock.Now().Add(b.cfg.Visibility),
	})
	return msg, true
}

// sweepExpired retires every in-flight message whose visibility deadline has
// passed. Callers must hold mu.
func (b *Broker) sweepExpired() {
	now := b.clock.Now()
	kept := b.inFlight[:0]
	for _, f := range b.inFlight {
		if f.deadline.After(now) {
			kept = append(kept, f)
			continue
		}
		b.retire(f.msg)
	}
	b.inFlight = kept
}

// retire routes a failed delivery: back of the ready queue while attempts
// remain, dead-letter queue once they are spent. Callers must hold mu.
func (b *Broker) retire(msg Message) {
	if msg.Attempts >= b.cfg.MaxAttempts {
		b.dead = append(b.dead, msg)
		return
	}
	b.ready = append(b.ready, msg)
}

// removeInFlight unlinks id from the in-flight list, reporting whether it was
// there. Callers must hold mu.
func (b *Broker) removeInFlight(id string) (Message, bool) {
	for i, f := range b.inFlight {
		if f.msg.ID == id {
			b.inFlight = append(b.inFlight[:i], b.inFlight[i+1:]...)
			return f.msg, true
		}
	}
	return Message{}, false
}

// Ack marks the in-flight message done; it will never be delivered again.
func (b *Broker) Ack(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.removeInFlight(id); !ok {
		return fmt.Errorf("ack %q: %w", id, ErrUnknownID)
	}
	return nil
}

// Nack retires the in-flight message immediately, without waiting for its
// visibility deadline: back to the ready queue if attempts remain, to the
// dead-letter queue otherwise. It says "this attempt failed".
func (b *Broker) Nack(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	msg, ok := b.removeInFlight(id)
	if !ok {
		return fmt.Errorf("nack %q: %w", id, ErrUnknownID)
	}
	b.retire(msg)
	return nil
}

// DeadLetter parks the in-flight message in the dead-letter queue right now,
// whatever its attempt count. It says "no attempt will ever succeed" — the
// verb for a payload that will never parse.
func (b *Broker) DeadLetter(id string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	msg, ok := b.removeInFlight(id)
	if !ok {
		return fmt.Errorf("dead-letter %q: %w", id, ErrUnknownID)
	}
	b.dead = append(b.dead, msg)
	return nil
}

// DeadLetters returns a copy of the dead-lettered messages in arrival order.
func (b *Broker) DeadLetters() []Message {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Message, len(b.dead))
	copy(out, b.dead)
	return out
}
