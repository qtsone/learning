package queue

import "strings"

// Event is a domain event recorded in the outbox alongside the state change
// that caused it. ID is the business-level event ID — it is what consumers
// dedupe on when the relay publishes the same event twice.
type Event struct {
	ID      string
	Payload string
}

// Wire renders the event as a message body: "<id>|<payload>". The event ID
// travels inside the message because the broker's ID cannot carry it — a
// republished event is a brand-new broker message with a brand-new ID.
func (e Event) Wire() string { return e.ID + "|" + e.Payload }

// EventID is the DedupeKey for messages a relay published: it recovers the
// event ID from the body, falling back to the broker's message ID for bodies
// that carry none.
func EventID(msg Message) string {
	id, _, ok := strings.Cut(msg.Body, "|")
	if !ok {
		return msg.ID
	}
	return id
}

// Store simulates a database that commits atomically across two tables: a
// key-value state table and an outbox of events awaiting publication. It is
// the producer half of the transactional outbox pattern.
type Store struct {
	state     map[string]string
	outbox    []Event
	published int // outbox[:published] have already been relayed
}

// NewStore returns an empty store.
func NewStore() *Store {
	return &Store{state: make(map[string]string)}
}

// Tx stages writes and events; nothing is visible until ExecTx commits.
type Tx struct {
	writes map[string]string
	events []Event
}

// Set stages a write of value under key.
func (tx *Tx) Set(key, value string) {
	tx.writes[key] = value
}

// Emit stages a domain event to be published if the transaction commits.
func (tx *Tx) Emit(e Event) {
	tx.events = append(tx.events, e)
}

// ExecTx runs fn against a fresh transaction. If fn returns nil, the staged
// writes and events commit together — this is the atomicity that closes the
// dual-write problem. If fn returns an error, neither commits and ExecTx
// returns that error.
func (s *Store) ExecTx(fn func(tx *Tx) error) error {
	tx := &Tx{writes: make(map[string]string)}
	if err := fn(tx); err != nil {
		return err
	}
	for k, v := range tx.writes {
		s.state[k] = v
	}
	s.outbox = append(s.outbox, tx.events...)
	return nil
}

// Get reads committed state.
func (s *Store) Get(key string) (string, bool) {
	v, ok := s.state[key]
	return v, ok
}

// Relay publishes every not-yet-published outbox event to b in commit order —
// each as its Wire form, so the event ID reaches the consumer — marks it
// published, and returns how many it published. Calling Relay again publishes
// nothing until new events are committed.
func (s *Store) Relay(b *Broker) int {
	n := 0
	for ; s.published < len(s.outbox); s.published++ {
		b.Publish(s.outbox[s.published].Wire())
		n++
	}
	return n
}
