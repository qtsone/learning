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
// The hub fans every event to every subscriber and knows nothing about who may
// see what — deliberately, because a hub that consulted a policy would be a
// second copy of one. Filtering is this handler's job, and it asks the same
// policy the listing asks. An unfiltered stream is the unscoped listing
// arriving over a pipe nobody thinks to audit.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	sub, err := s.hub.Subscribe()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "server is shutting down")
		return
	}
	// Every exit path unsubscribes. A handler that returns without this leaks
	// an entry the publisher walks on every event.
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
		// The server's WriteTimeout bounds a whole response, and this response
		// lasts hours, so the deadline moves forward one frame at a time
		// instead of the route being exempt from deadlines altogether.
		//
		// The instant comes from time.Now and not the injected Clock: this
		// deadline is handed to the network poller, which compares it against
		// the real clock. It is a socket setting, not one of this service's
		// rules about time. A ResponseWriter with no deadline support — a test
		// recorder, say — says so, and then there is nothing to bound.
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

	ctx := r.Context()

	// The identity is re-resolved, never captured. SubjectFrom is the answer
	// the middleware gave when this connection opened, and this connection
	// outlives that instant by hours: a demotion or a logout in between has to
	// reach it, or "destroy the target's sessions" is a promise the stream does
	// not keep. The price is a map read and one row per event — which is what
	// an admin's 03:00 "no" costs on a live connection.
	sessionID := SubjectFrom(ctx).SessionID

	// One decision function, a third call site. The event carries the only
	// attribute the policy needs, so the filter is one line and it moves when
	// the rule table moves.
	visible := func(who Subject, e Event) bool {
		return s.policy.Allows(who, ActionTaskRead, Resource{ID: e.ID, OwnerID: e.OwnerID})
	}

	if s.streamRetry > 0 && !writeEvent(Event{Retry: s.streamRetry}) {
		return
	}

	// Subscribe-then-replay can send an event twice if one is published in
	// between; replay-then-subscribe would lose it. A duplicate the client can
	// spot by id beats a gap it cannot.
	if last := r.Header.Get("Last-Event-ID"); last != "" {
		who, live := s.subjectFor(ctx, sessionID)
		if !live {
			return
		}
		missed, ok := s.hub.Since(last)
		if !ok {
			// The gap cannot be filled, and it is not ours to hide. Resync is
			// about the connection, not about anyone's data, so it is the one
			// frame the filter does not apply to.
			if !writeEvent(ResyncEvent) {
				return
			}
		}
		for _, e := range missed {
			if visible(who, e) && !writeEvent(e) {
				return
			}
		}
	}

	var ticks <-chan time.Time
	if s.heartbeat > 0 {
		c, stop := s.clock.NewTicker(s.heartbeat)
		defer stop()
		ticks = c
	}

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
			who, live := s.subjectFor(ctx, sessionID)
			if !live {
				// The session behind this stream is gone. Ending the response
				// is the only refusal left: the status line went out with the
				// first flush, hours ago.
				return
			}
			if !visible(who, e) {
				continue
			}
			if !writeEvent(e) {
				return
			}
		case <-ticks:
			// A connection with no bytes on it is closed by proxies and mobile
			// NAT without telling either end. The write is also the liveness
			// check: it fails when the client is really gone.
			//
			// The tick is also when an idle stream notices a revocation, so a
			// demoted user's open tab dies within one heartbeat even if nothing
			// is published in the meantime.
			if _, live := s.subjectFor(ctx, sessionID); !live {
				return
			}
			if !write(Comment("heartbeat " + s.clock.Now().UTC().Format(time.RFC3339))) {
				return
			}
		}
	}
}
