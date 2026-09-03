package board

import (
	"errors"
	"io"
	"net/http"
	"time"
)

// ResyncEvent is what a reconnecting client gets when its Last-Event-ID has
// already fallen out of the backlog. A comment would be invisible to the page,
// so the gap has to be announced as a real event the application can act on:
// refetch, then keep streaming.
var ResyncEvent = Event{Name: "resync", Data: "history unavailable: refetch current state"}

// handleEvents serves GET /events as text/event-stream.
//
// The preamble below is the shape you built in the realtime lesson: subscribe
// before writing anything (after the first flush the status is 200 and there is
// no way back), then the headers, then a flush that proves this response can
// stream at all.
//
// What is new is that this stream now runs inside a service with an
// access-control model, and the hub has none: it fans every event to every
// subscriber. As written, this handler hands a member every other member's
// task titles — the same leak as an unscoped listing, arriving over a different
// pipe and harder to notice.
//
// TODO, three things:
//  1. Write only what this subscriber may read. Every Event carries an OwnerID;
//     the policy already answers the question — do not invent a second answer
//     here. The filter applies to replayed events exactly as it does to live
//     ones. Resolve the identity with s.subjectFor and the session id on the
//     request's Subject rather than capturing SubjectFrom(r.Context()): this
//     connection lasts hours, and a role change or a logout inside those hours
//     has to reach it. A session that is gone ends the stream.
//  2. Honour Last-Event-ID: ask s.hub.Since for what the client missed and
//     write it (filtered) before the live stream, or send ResyncEvent when the
//     backlog no longer reaches that far. Subscribe first, then replay: the
//     other order loses an event published in between, and a duplicate the
//     client can spot by id beats a silent gap.
//  3. Heartbeat. A connection with no bytes on it is closed by proxies and
//     mobile NAT without telling either end. Send Comment(...) on every tick of
//     a ticker from s.clock.NewTicker(s.heartbeat) when s.heartbeat > 0, and
//     release the ticker when you return.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	sub, err := s.hub.Subscribe()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "server is shutting down")
		return
	}
	// Every exit path unsubscribes. A handler that returns without this leaks an
	// entry the publisher walks on every event.
	defer s.hub.Unsubscribe(sub)

	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	// Reverse proxies buffer responses by default, which turns a stream into a
	// very slow download.
	header.Set("X-Accel-Buffering", "no")

	rc := http.NewResponseController(w)
	if err := rc.Flush(); err != nil {
		// Nothing has been written yet — a ResponseWriter with no flusher in
		// its chain reports this without touching the response — so a real
		// status is still possible.
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	write := func(frame string) bool {
		if frame == "" {
			return true
		}
		// NewHTTPServer sets a WriteTimeout, which bounds a whole response —
		// and this response lasts hours. So the deadline moves forward one
		// frame at a time instead of the route being exempt from deadlines.
		// The instant comes from time.Now and not the injected Clock: this one
		// is handed to the network poller, which compares it against the real
		// clock. A ResponseWriter with no deadline support says so, and then
		// there is nothing to bound.
		if err := rc.SetWriteDeadline(time.Now().Add(StreamFrameTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
			return false
		}
		if _, err := io.WriteString(w, frame); err != nil {
			return false
		}
		return rc.Flush() == nil
	}

	// Every frame goes through Frame's validation. Nothing published here
	// carries user text today, but a bare "\r" ends a line in
	// text/event-stream, so the day somebody publishes an event built from a
	// title is the day an unvalidated field forges an `event:` on the wire.
	// One unsendable event is dropped; the stream survives it.
	writeEvent := func(e Event) bool {
		frame, err := e.Frame()
		if err != nil {
			s.log.Error("dropping an unsendable event", "err", err, "event_id", e.ID)
			return true
		}
		return write(frame)
	}

	if s.streamRetry > 0 && !writeEvent(Event{Retry: s.streamRetry}) {
		return
	}

	var ticks <-chan time.Time
	_ = ticks // TODO: a heartbeat ticker from the injected clock

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			// The client hung up, or a shutdown cancelled the base context.
			return
		case e, ok := <-sub.Events():
			if !ok {
				// Evicted for falling behind, or the hub shut down. Either way
				// this stream is over; the client reconnects with Last-Event-ID.
				return
			}
			if !writeEvent(e) {
				return
			}
		}
	}
}
