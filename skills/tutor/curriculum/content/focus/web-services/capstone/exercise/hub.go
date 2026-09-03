package board

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// ErrHubClosed is returned by Subscribe once the hub has been shut down. A
// handler that gets it must not start streaming: the process is going away.
var ErrHubClosed = errors.New("board: hub is closed")

// ErrInvalidField reports a field value that cannot be put on the wire. In
// text/event-stream a line ends at "\n" or "\r", so a value carrying one could
// close the current field and open another: the SSE equivalent of header
// injection.
//
// Every field this service publishes today is server-generated — ids are
// counters and Data is json.Marshal output, which escapes both — so the guard
// never fires. It is carried unchanged from the realtime lesson anyway, because
// it protects the *next* event somebody adds: the one built from a task title,
// a comment or an error message.
var ErrInvalidField = errors.New("board: invalid event field")

// Frame renders the event as one SSE frame, ending with the blank line that
// terminates it. A retry-only event renders just the reconnection delay. It
// returns ErrInvalidField if ID or Name contains a newline, or if any field
// contains a carriage return or a NUL.
func (e Event) Frame() (string, error) {
	for _, f := range []struct {
		name         string
		value        string
		allowNewline bool
	}{
		{"id", e.ID, false},
		{"event", e.Name, false},
		{"data", e.Data, true},
	} {
		if err := checkField(f.value, f.allowNewline); err != nil {
			return "", fmt.Errorf("%w: %s: %v", ErrInvalidField, f.name, err)
		}
	}

	var b strings.Builder
	if e.ID != "" {
		b.WriteString("id: " + e.ID + "\n")
	}
	if e.Name != "" {
		b.WriteString("event: " + e.Name + "\n")
	}
	if ms := e.Retry.Milliseconds(); ms > 0 {
		b.WriteString("retry: " + strconv.FormatInt(ms, 10) + "\n")
	}
	if e.Data != "" {
		for _, line := range strings.Split(e.Data, "\n") {
			b.WriteString("data: " + line + "\n")
		}
	}
	if b.Len() == 0 {
		return "", nil
	}
	b.WriteString("\n")
	return b.String(), nil
}

func checkField(value string, allowNewline bool) error {
	if strings.ContainsAny(value, "\r\x00") {
		return errors.New("contains a carriage return or NUL")
	}
	if !allowNewline && strings.Contains(value, "\n") {
		return errors.New("contains a newline")
	}
	return nil
}

// Comment renders an SSE comment frame. Clients parse and drop it, which makes
// it perfect for heartbeats and useless for telling the application anything.
func Comment(text string) string { return ": " + text + "\n\n" }

// Event names, as constants: a client subscribes to these strings, so they are
// API surface and renaming one is a breaking change.
const (
	EventTaskCreated  = "task.created"
	EventTaskUpdated  = "task.updated"
	EventTaskNotified = "task.notified"
)

// taskEvent builds the event announcing something that happened to a task. The
// OwnerID it carries is what every stream filters on.
func taskEvent(name string, t Task) Event {
	data, err := json.Marshal(t)
	if err != nil {
		// A Task is always marshalable; if that ever stops being true, an
		// event with no payload beats a panic in a publisher.
		return Event{Name: name, OwnerID: t.OwnerID}
	}
	return Event{Name: name, OwnerID: t.OwnerID, Data: string(data)}
}

// Subscriber is one client's end of the hub. The hub sends events into it and
// the hub — nobody else — closes it, so a send can never race a close. A closed
// channel is the signal to stop streaming, whatever the reason.
type Subscriber struct {
	events chan Event
	closed bool // guarded by the hub's mutex
}

// Events is the channel this subscriber reads.
func (s *Subscriber) Events() <-chan Event { return s.events }

// Hub fans one stream of events out to every connected subscriber. It knows
// nothing about who may see what: every event carries an OwnerID and each
// stream applies the policy for its own subscriber.
type Hub struct {
	mu      sync.Mutex
	subs    map[*Subscriber]struct{}
	backlog []Event
	buffer  int
	keep    int
	seq     int64
	closed  bool
	logger  *slog.Logger
}

// NewHub returns a hub whose subscribers each get a buffer of the given size
// and which remembers the last keep events for replay.
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
	return &Hub{subs: make(map[*Subscriber]struct{}), buffer: buffer, keep: keep, logger: logger}
}

// Subscribe registers a new subscriber, or returns ErrHubClosed.
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

// Unsubscribe removes s and closes its channel. It is safe to call twice, with
// a subscriber the hub never had, or with nil.
func (h *Hub) Unsubscribe(s *Subscriber) {
	if s == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.remove(s)
}

// remove drops s and closes it exactly once. The caller holds h.mu, which is
// what makes "closed" and "removed" one atomic fact: sending and closing happen
// under the same lock, so no publisher can send on a closed channel.
func (h *Hub) remove(s *Subscriber) {
	delete(h.subs, s)
	if !s.closed {
		s.closed = true
		close(s.events)
	}
}

// Publish stamps the event with the next id, records it for replay, and sends
// it to every subscriber. It returns the stamped event.
//
// The send is non-blocking: a subscriber whose buffer is full is evicted, its
// stream ends, and the client reconnects with Last-Event-ID and is replayed
// what it missed. Blocking here would let one stalled reader stop the fan-out.
func (h *Hub) Publish(e Event) Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return e
	}
	if e.ID == "" {
		h.seq++
		e.ID = strconv.FormatInt(h.seq, 10)
	}
	if h.keep > 0 {
		h.backlog = append(h.backlog, e)
		if len(h.backlog) > h.keep {
			h.backlog = slices.Clone(h.backlog[len(h.backlog)-h.keep:])
		}
	}
	for s := range h.subs {
		select {
		case s.events <- e:
		default:
			h.remove(s)
			h.logger.Warn("evicted a subscriber that fell behind", "buffer", h.buffer, "event_id", e.ID)
		}
	}
	return e
}

// Since returns the events recorded after lastID, and whether lastID was found
// at all. False means the backlog no longer reaches back that far: the gap
// cannot be filled and the client has to be told to resynchronise.
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

// Subscribers reports how many subscribers are registered. Tests use it to
// prove handlers clean up after themselves.
func (h *Hub) Subscribers() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.subs)
}

// Shutdown closes every subscriber and refuses new subscriptions. Call it
// *before* http.Server.Shutdown: a streaming handler that is never told to stop
// blocks a graceful shutdown until its deadline.
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
