package realtime

import (
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
	// TODO: in this order.
	//
	//  1. Subscribe to the hub. ErrHubClosed means the process is going away:
	//     answer 503 and return. This has to happen before any header is
	//     written — once you have committed to 200 you cannot report an error
	//     status any more. Defer Unsubscribe immediately: every exit path
	//     below must give the subscription back.
	//
	//  2. Set the streaming headers:
	//       Content-Type: text/event-stream
	//       Cache-Control: no-cache
	//       X-Accel-Buffering: no
	//     Do not set Connection: net/http manages it, and it means nothing
	//     over HTTP/2.
	//
	//  3. Flush, through http.NewResponseController(w). That sends the header
	//     block so the client's EventSource opens, and it is also your
	//     "can this response stream at all?" probe: a ResponseWriter with no
	//     flusher anywhere in its chain reports http.ErrNotSupported without
	//     writing anything, so you can still answer 500. Any other flush error
	//     means the connection is gone — just return.
	//
	//  4. If Retry > 0, send Event{Retry: h.Retry} and flush.
	//
	//  5. If the request carries a Last-Event-ID header, ask the hub for what
	//     the client missed. Found: send those events. Not found: send
	//     ResyncEvent. Flush.
	//
	//  6. If Heartbeat > 0, create a ticker from the clock and defer its stop
	//     function. A nil channel blocks forever in a select, which is exactly
	//     what you want when heartbeats are disabled.
	//
	//  7. Loop over a select of three cases:
	//       r.Context().Done()  — the client went away; return,
	//       the subscriber's channel — an event to frame, write and flush;
	//                              a closed channel ends the stream,
	//       the ticker — write Comment("heartbeat " + now in RFC3339 UTC),
	//                    using the injected clock, and flush.
	//     A failed write or flush means the client is gone: return.
}
