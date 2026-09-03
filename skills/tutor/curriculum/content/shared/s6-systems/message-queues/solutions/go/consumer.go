package queue

// Processor performs a message's side effect (charge the card, send the
// email). Under at-least-once delivery it may be handed the same message
// more than once; IdempotentConsumer exists to make that safe.
type Processor func(Message) error

// DedupeKey names the piece of work a message represents. Two messages with
// the same key are the same work and must have their side effect run once
// between them.
type DedupeKey func(Message) string

// ByMessageID keys on the broker's message ID: right when a redelivery is the
// only way a duplicate can appear, wrong as soon as a producer can publish the
// same business event twice (see Relay in outbox.go).
func ByMessageID(msg Message) string { return msg.ID }

// IdempotentConsumer runs each message's side effect at most once per dedupe
// key, no matter how many times the broker delivers it. In a real system the
// dedupe record lives in a database shared by all consumer replicas; here it
// is in memory.
type IdempotentConsumer struct {
	process Processor
	key     DedupeKey
	done    map[string]bool // keys whose side effect has succeeded
}

// NewIdempotentConsumer wraps p with duplicate suppression keyed by key. A nil
// key falls back to ByMessageID.
func NewIdempotentConsumer(p Processor, key DedupeKey) *IdempotentConsumer {
	if key == nil {
		key = ByMessageID
	}
	return &IdempotentConsumer{process: p, key: key, done: make(map[string]bool)}
}

// Handle processes msg exactly once per dedupe key: duplicates of completed
// work are skipped (and safe to ack — the work has genuinely happened), while
// a failed attempt surfaces its error unrecorded so that a later redelivery
// retries it.
func (c *IdempotentConsumer) Handle(msg Message) error {
	key := c.key(msg)
	if c.done[key] {
		return nil
	}
	if err := c.process(msg); err != nil {
		return err
	}
	c.done[key] = true
	return nil
}
