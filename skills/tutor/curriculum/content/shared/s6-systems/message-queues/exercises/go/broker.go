package queue

import (
	"errors"
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
	// TODO: under the lock, assign the next sequential ID, append the
	// message to the ready queue, and return the ID.
	return ""
}

// Receive returns the next deliverable message, or ok == false when nothing
// is ready. Before selecting, it sweeps in-flight messages whose visibility
// deadline has passed (deadline <= now), oldest delivery first: those with
// attempts remaining rejoin the back of the ready queue; those that have
// used MaxAttempts deliveries move to the dead-letter queue. The returned
// message has Attempts incremented and stays in flight until Ack, Nack, or
// its new deadline (now + Visibility).
func (b *Broker) Receive() (Message, bool) {
	// TODO: sweep expired in-flight messages, then deliver the head of the
	// ready queue (if any) and record it as in flight.
	return Message{}, false
}

// Ack marks the in-flight message done; it will never be delivered again.
func (b *Broker) Ack(id string) error {
	// TODO: remove the message from the in-flight list. Unknown or
	// already-acked IDs return an error wrapping ErrUnknownID.
	return ErrUnknownID
}

// Nack retires the in-flight message immediately, without waiting for its
// visibility deadline: back to the ready queue if attempts remain, to the
// dead-letter queue otherwise. It says "this attempt failed".
func (b *Broker) Nack(id string) error {
	// TODO: same lookup as Ack, then route by the attempts rule.
	return ErrUnknownID
}

// DeadLetter parks the in-flight message in the dead-letter queue right now,
// whatever its attempt count. It says "no attempt will ever succeed" — the
// verb for a payload that will never parse.
func (b *Broker) DeadLetter(id string) error {
	// TODO: same lookup as Ack, then park the message without consulting
	// MaxAttempts.
	return ErrUnknownID
}

// DeadLetters returns a copy of the dead-lettered messages in arrival order.
func (b *Broker) DeadLetters() []Message {
	// TODO: return a copy, not the internal slice.
	return nil
}
