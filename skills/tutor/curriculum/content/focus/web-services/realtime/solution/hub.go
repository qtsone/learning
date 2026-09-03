package realtime

import (
	"errors"
	"log/slog"
	"slices"
	"sync"
)

// ErrHubClosed is returned by Subscribe once the hub has been shut down. A
// handler that gets it must not start streaming: the process is going away.
var ErrHubClosed = errors.New("realtime: hub is closed")

// Subscriber is one client's end of the hub. The hub sends events into it and
// the hub — nobody else — closes it, so that a send can never race a close.
//
// A closed channel is the signal to stop streaming, whatever the reason: the
// subscriber was evicted for falling behind, or the hub shut down.
type Subscriber struct {
	events chan Event

	// closed is guarded by the hub's mutex, so Unsubscribe is idempotent and
	// close() runs exactly once.
	closed bool
}

// Events is the channel this subscriber reads. Receiving a zero Event with
// ok == false means the subscription has ended.
func (s *Subscriber) Events() <-chan Event { return s.events }

// Hub fans one stream of events out to every connected subscriber.
//
// The hub owns the subscriber set and every subscriber's channel. Publish
// never blocks: each subscriber has a bounded buffer, and a subscriber that
// cannot keep up is evicted rather than allowed to hold the publisher (and
// therefore every other subscriber) hostage.
type Hub struct {
	mu      sync.Mutex
	subs    map[*Subscriber]struct{}
	backlog []Event
	buffer  int
	keep    int
	closed  bool
	logger  *slog.Logger
}

// NewHub returns a hub whose subscribers each get a buffer of the given size
// and which remembers the last keep identified events for replay. A nil logger
// means slog.Default().
func NewHub(buffer, keep int, logger *slog.Logger) *Hub {
	if buffer < 0 {
		buffer = 0
	}
	if keep < 0 {
		keep = 0
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		subs:   make(map[*Subscriber]struct{}),
		buffer: buffer,
		keep:   keep,
		logger: logger,
	}
}

// Subscribe registers a new subscriber, or returns ErrHubClosed if the hub is
// shut down.
func (h *Hub) Subscribe() (*Subscriber, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, ErrHubClosed
	}
	s := &Subscriber{events: make(chan Event, h.buffer)}
	h.subs[s] = struct{}{}
	return s, nil
}

// Unsubscribe removes s and closes its channel. It is safe to call more than
// once, with a subscriber the hub never had, or with nil — handlers defer it
// and cannot know which of those cases they are in.
func (h *Hub) Unsubscribe(s *Subscriber) {
	if s == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.remove(s)
}

// remove drops s from the set and closes it exactly once. The caller holds
// h.mu, which is what guarantees no goroutine is mid-send into a channel this
// closes: sending and closing happen under the same lock.
func (h *Hub) remove(s *Subscriber) {
	delete(h.subs, s)
	if !s.closed {
		s.closed = true
		close(s.events)
	}
}

// Publish sends e to every subscriber and reports how many received it and how
// many were evicted for being too far behind.
//
// The send is non-blocking. A subscriber whose buffer is full is dropped and
// its channel closed: its stream ends, and the client reconnects with
// Last-Event-ID and is replayed what it missed. Blocking here instead would
// let one stalled reader stop the whole fan-out.
//
// Identified events (ID != "") are also appended to the replay backlog, which
// keeps at most h.keep of them.
func (h *Hub) Publish(e Event) (delivered, evicted int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return 0, 0
	}
	if e.ID != "" && h.keep > 0 {
		h.backlog = append(h.backlog, e)
		if len(h.backlog) > h.keep {
			h.backlog = slices.Clone(h.backlog[len(h.backlog)-h.keep:])
		}
	}
	for s := range h.subs {
		select {
		case s.events <- e:
			delivered++
		default:
			h.remove(s)
			evicted++
			h.logger.Warn("realtime: evicted a subscriber that fell behind",
				"buffer", h.buffer, "event_id", e.ID)
		}
	}
	return delivered, evicted
}

// Since returns the events recorded after lastID, and whether lastID was found
// at all. A false result means the backlog no longer reaches back that far:
// the gap cannot be filled and the client has to be told to resynchronise.
func (h *Hub) Since(lastID string) ([]Event, bool) {
	if lastID == "" {
		return nil, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, e := range h.backlog {
		if e.ID == lastID {
			return slices.Clone(h.backlog[i+1:]), true
		}
	}
	return nil, false
}

// Subscribers reports how many subscribers are currently registered. Tests use
// it to prove that handlers clean up after themselves.
func (h *Hub) Subscribers() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}

// Shutdown closes every subscriber (ending every stream) and refuses new
// subscriptions. Call it before http.Server.Shutdown: a streaming handler that
// is never told to stop will block a graceful shutdown until its deadline.
func (h *Hub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for s := range h.subs {
		h.remove(s)
	}
}
