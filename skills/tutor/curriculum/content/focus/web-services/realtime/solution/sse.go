package realtime

import (
	"errors"
	"io"
	"net/http"
	"time"
)

// ResyncEvent is what a reconnecting client gets when its Last-Event-ID has
// already fallen out of the backlog. A comment would be invisible to browser
// JavaScript, so the gap has to be announced as a real event the application
// can act on: refetch the current state, then keep streaming.
var ResyncEvent = Event{Name: "resync", Data: "history unavailable: refetch current state"}

// Handler streams a hub to one client as text/event-stream.
//
// One request is one goroutine holding one connection and one buffer, for as
// long as the client stays. Everything here exists to bound that: the request
// context ends the loop when the client goes away, the hub ends it when the
// process is shutting down or the client falls behind, and the heartbeat keeps
// an idle connection from being reaped by a proxy in between.
type Handler struct {
	// Hub is the fan-out this stream subscribes to.
	Hub *Hub

	// Clock provides the heartbeat ticker. Nil means RealClock.
	Clock Clock

	// Heartbeat is how often a comment frame is sent to keep the connection
	// alive through idle timeouts. Zero disables heartbeats.
	Heartbeat time.Duration

	// Retry is the reconnection delay advertised to the client at the start of
	// the stream. Zero sends no retry field, leaving the client's default
	// (about three seconds in browsers).
	Retry time.Duration
}

func (h *Handler) clock() Clock {
	if h.Clock == nil {
		return RealClock{}
	}
	return h.Clock
}

// ServeHTTP streams events until the client leaves or the subscription ends.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Subscribing first is what keeps the error paths available: after the
	// first flush the status is 200 and there is no way back.
	sub, err := h.Hub.Subscribe()
	if err != nil {
		http.Error(w, "server is shutting down", http.StatusServiceUnavailable)
		return
	}
	defer h.Hub.Unsubscribe(sub)

	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	// Reverse proxies buffer responses by default, which turns a stream into
	// a very slow download. nginx and its relatives honour this header.
	header.Set("X-Accel-Buffering", "no")

	rc := http.NewResponseController(w)
	if err := rc.Flush(); err != nil {
		// Nothing has been written yet — a ResponseWriter with no flusher in
		// its chain reports this without touching the response — so a real
		// status is still possible.
		if errors.Is(err, http.ErrNotSupported) {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		}
		return
	}

	write := func(frame string) bool {
		if frame == "" {
			return true
		}
		if _, err := io.WriteString(w, frame); err != nil {
			return false
		}
		return rc.Flush() == nil
	}

	if h.Retry > 0 {
		frame, err := Event{Retry: h.Retry}.Frame()
		if err != nil || !write(frame) {
			return
		}
	}

	// Subscribe-then-replay can send an event twice if it is published in
	// between; replay-then-subscribe would lose it. A duplicate the client can
	// spot by id beats a silent gap.
	if last := r.Header.Get("Last-Event-ID"); last != "" {
		missed, ok := h.Hub.Since(last)
		if !ok {
			missed = []Event{ResyncEvent}
		}
		for _, e := range missed {
			frame, err := e.Frame()
			if err != nil {
				continue
			}
			if !write(frame) {
				return
			}
		}
	}

	var ticks <-chan time.Time
	if h.Heartbeat > 0 {
		c, stop := h.clock().NewTicker(h.Heartbeat)
		defer stop()
		ticks = c
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			// The client hung up, or a shutdown cancelled the base context.
			return
		case e, ok := <-sub.Events():
			if !ok {
				// Evicted for falling behind, or the hub shut down. Either
				// way this stream is over; the client reconnects with
				// Last-Event-ID.
				return
			}
			frame, err := e.Frame()
			if err != nil {
				// One unrepresentable event must not kill the stream.
				continue
			}
			if !write(frame) {
				return
			}
		case <-ticks:
			if !write(Comment("heartbeat " + h.clock().Now().UTC().Format(time.RFC3339))) {
				return
			}
		}
	}
}
