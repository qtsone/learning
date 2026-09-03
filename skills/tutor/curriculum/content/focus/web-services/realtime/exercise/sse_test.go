package realtime

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestHandler(hub *Hub, clk Clock) *Handler {
	return &Handler{
		Hub:       hub,
		Clock:     clk,
		Heartbeat: 15 * time.Second,
		Retry:     3 * time.Second,
	}
}

func streamRequest(lastEventID string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/events", nil)
	if lastEventID != "" {
		r.Header.Set("Last-Event-ID", lastEventID)
	}
	return r
}

func TestHandlerSetsStreamingHeadersAndOpensTheStream(t *testing.T) {
	hub := NewHub(4, 8, nil)
	s := startStream(t, newTestHandler(hub, newFakeClock()), streamRequest(""))

	// The retry frame is the first thing on the wire, so awaiting it proves
	// the response was flushed rather than buffered.
	s.await(t, "retry: 3000\n\n")
	s.stop(t)

	if got := s.rec.status(); got != http.StatusOK {
		t.Errorf("status = %d, want 200", got)
	}
	for header, want := range map[string]string{
		"Content-Type":      "text/event-stream",
		"Cache-Control":     "no-cache",
		"X-Accel-Buffering": "no",
	} {
		if got := s.rec.headerValue(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
	if got := s.rec.headerValue("Connection"); got != "" {
		t.Errorf("Connection = %q, want it unset: net/http owns that header and it is meaningless over HTTP/2", got)
	}
}

func TestHandlerStreamsPublishedEvents(t *testing.T) {
	hub := NewHub(4, 8, nil)
	s := startStream(t, newTestHandler(hub, newFakeClock()), streamRequest(""))
	waitSubscribers(t, hub, 1)

	hub.Publish(Event{ID: "1", Name: "task.created", Data: `{"id":1}`})
	s.await(t, "id: 1\nevent: task.created\ndata: {\"id\":1}\n\n")

	hub.Publish(Event{ID: "2", Data: "second"})
	s.await(t, "id: 2\ndata: second\n\n")

	s.stop(t)
	if got := hub.Subscribers(); got != 0 {
		t.Errorf("Subscribers() = %d after the handler returned, want 0: every exit path must unsubscribe", got)
	}
}

func TestHandlerHeartbeatComesFromTheInjectedClock(t *testing.T) {
	hub := NewHub(4, 8, nil)
	clk := newFakeClock()
	s := startStream(t, newTestHandler(hub, clk), streamRequest(""))
	waitSubscribers(t, hub, 1)

	clk.tick(t)
	s.await(t, ": heartbeat 2024-05-01T12:00:00Z\n\n")

	clk.Advance(30 * time.Second)
	clk.tick(t)
	s.await(t, ": heartbeat 2024-05-01T12:00:30Z\n\n")

	s.stop(t)
}

func TestHandlerReturnsWhenTheClientDisconnects(t *testing.T) {
	hub := NewHub(4, 8, nil)
	s := startStream(t, newTestHandler(hub, newFakeClock()), streamRequest(""))
	waitSubscribers(t, hub, 1)

	s.stop(t) // cancelling the request context is what a disconnect looks like

	if got := hub.Subscribers(); got != 0 {
		t.Errorf("Subscribers() = %d, want 0: a disconnected client must not leave a subscription (or a goroutine) behind", got)
	}
}

func TestHandlerReplaysWhatAReconnectingClientMissed(t *testing.T) {
	hub := NewHub(4, 8, nil)
	for _, id := range []string{"1", "2", "3"} {
		hub.Publish(Event{ID: id, Data: "event " + id})
	}

	s := startStream(t, newTestHandler(hub, newFakeClock()), streamRequest("1"))
	s.await(t, "id: 3\ndata: event 3\n\n")

	if !s.sent("id: 2\ndata: event 2\n\n") {
		t.Error("event 2 was not replayed: Last-Event-ID: 1 means the client has seen everything up to and including 1")
	}
	if s.sent("data: event 1\n") {
		t.Error("event 1 was replayed: the client already has it")
	}

	// Replay must hand over to the live stream without a gap.
	waitSubscribers(t, hub, 1)
	hub.Publish(Event{ID: "4", Data: "event 4"})
	s.await(t, "id: 4\ndata: event 4\n\n")
	s.stop(t)
}

func TestHandlerAnnouncesResyncWhenTheBacklogCannotReachBack(t *testing.T) {
	hub := NewHub(4, 2, nil)
	for _, id := range []string{"1", "2", "3"} {
		hub.Publish(Event{ID: id, Data: "event " + id})
	}

	s := startStream(t, newTestHandler(hub, newFakeClock()), streamRequest("1"))

	want, err := ResyncEvent.Frame()
	if err != nil {
		t.Fatalf("framing ResyncEvent: %v", err)
	}
	s.await(t, want)
	if s.sent("data: event 2\n") {
		t.Error("events were replayed for an id the backlog no longer holds: a partial replay is a silent gap")
	}
	s.stop(t)
}

func TestHandlerEndsWhenTheHubShutsDown(t *testing.T) {
	hub := NewHub(4, 8, nil)
	s := startStream(t, newTestHandler(hub, newFakeClock()), streamRequest(""))
	waitSubscribers(t, hub, 1)

	hub.Shutdown()

	s.awaitReturn(t, "a closed subscriber channel ends the stream — otherwise http.Server.Shutdown waits for this handler until its deadline")
}

func TestHandlerRefusesToOpenAStreamAfterShutdown(t *testing.T) {
	hub := NewHub(4, 8, nil)
	hub.Shutdown()

	rec := httptest.NewRecorder()
	serveGuarded(t, newTestHandler(hub, newFakeClock()), rec, streamRequest(""))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503: a request that arrives during shutdown gets an answer, not a stream", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got == "text/event-stream" {
		t.Error("the response was declared as an event stream: subscribe before you commit to 200, or you cannot report the failure")
	}
}

func TestHandlerRefusesAResponseItCannotFlush(t *testing.T) {
	hub := NewHub(4, 8, nil)
	rec := newPlainRecorder()

	serveGuarded(t, newTestHandler(hub, newFakeClock()), rec, streamRequest(""))

	if rec.code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: a response that cannot be flushed cannot stream", rec.code)
	}
	if got := hub.Subscribers(); got != 0 {
		t.Errorf("Subscribers() = %d, want 0: the failed stream must give its subscription back", got)
	}
}
