package board

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// streamRecorder is a ResponseWriter that reports every flush separately, so a
// test can observe a response that is still being written. httptest's recorder
// cannot: reading its buffer while the handler writes is a data race, and it
// would not say whether anything was actually flushed.
type streamRecorder struct {
	mu       sync.Mutex
	header   http.Header
	snapshot http.Header
	code     int
	pending  bytes.Buffer
	flushes  chan string
}

func newStreamRecorder() *streamRecorder {
	return &streamRecorder{header: make(http.Header), flushes: make(chan string, 64)}
}

func (s *streamRecorder) Header() http.Header { return s.header }

func (s *streamRecorder) WriteHeader(code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.code != 0 {
		return
	}
	s.code = code
	s.snapshot = s.header.Clone()
}

func (s *streamRecorder) Write(p []byte) (int, error) {
	s.WriteHeader(http.StatusOK)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pending.Write(p)
}

// Flush publishes everything written since the last flush. Unflushed bytes are
// invisible to the test, exactly as they are invisible to a real client.
func (s *streamRecorder) Flush() {
	s.WriteHeader(http.StatusOK)
	s.mu.Lock()
	chunk := s.pending.String()
	s.pending.Reset()
	s.mu.Unlock()
	if chunk == "" {
		return
	}
	select {
	case s.flushes <- chunk:
	case <-time.After(hangGuard):
	}
}

func (s *streamRecorder) status() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.code
}

func (s *streamRecorder) headerValue(key string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.snapshot == nil {
		return s.header.Get(key)
	}
	return s.snapshot.Get(key)
}

// stream runs the service on its own goroutine, the way a real server does, and
// collects what it flushes.
type stream struct {
	rec    *streamRecorder
	cancel context.CancelFunc
	done   chan struct{}
	seen   strings.Builder
}

// openStream connects to GET /events as one user.
func (e *env) openStream(t *testing.T, cookie *http.Cookie, lastEventID string) *stream {
	t.Helper()
	r := e.request(http.MethodGet, "/events", nil)
	if cookie != nil {
		r.AddCookie(cookie)
	}
	if lastEventID != "" {
		r.Header.Set("Last-Event-ID", lastEventID)
	}
	ctx, cancel := context.WithCancel(r.Context())
	s := &stream{rec: newStreamRecorder(), cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(s.done)
		e.handler.ServeHTTP(s.rec, r.WithContext(ctx))
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-s.done:
		case <-time.After(hangGuard):
			t.Error("a streaming handler was still running after its request context was cancelled")
		}
	})
	return s
}

func (s *stream) drain() {
	for {
		select {
		case chunk := <-s.rec.flushes:
			s.seen.WriteString(chunk)
		default:
			return
		}
	}
}

// await waits until the flushed bytes so far contain want.
func (s *stream) await(t *testing.T, want string) {
	t.Helper()
	deadline := time.After(hangGuard)
	for {
		if strings.Contains(s.seen.String(), want) {
			return
		}
		select {
		case chunk := <-s.rec.flushes:
			s.seen.WriteString(chunk)
		case <-s.done:
			s.drain()
			if strings.Contains(s.seen.String(), want) {
				return
			}
			t.Fatalf("the handler returned without sending %q\nstream so far:\n%s", want, s.seen.String())
		case <-deadline:
			t.Fatalf("timed out waiting for %q (did you flush after writing it?)\nstream so far:\n%s", want, s.seen.String())
		}
	}
}

// sent reports whether text has appeared so far. Call it only after awaiting
// something that must come later, so the answer is not a race.
func (s *stream) sent(text string) bool {
	s.drain()
	return strings.Contains(s.seen.String(), text)
}

// serveGuarded serves a request that must return on its own, and fails instead
// of hanging the suite when it does not.
func (e *env) serveGuarded(t *testing.T, r *http.Request, why string) *httptest.ResponseRecorder {
	t.Helper()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		e.handler.ServeHTTP(rec, r.WithContext(ctx))
	}()
	select {
	case <-done:
	case <-time.After(hangGuard):
		t.Fatalf("the handler did not return: %s", why)
	}
	return rec
}

func TestStreamIsBehindTheSameGate(t *testing.T) {
	e := newEnv(t)

	rec := e.serveGuarded(t, e.request(http.MethodGet, "/events", nil),
		"an unauthenticated request must be refused before any streaming starts")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("GET /events with no session = %d, want 401", rec.Code)
	}
	if n := e.hub.Subscribers(); n != 0 {
		t.Errorf("hub has %d subscriber(s) after a refused request, want 0", n)
	}
}

func TestStreamSetsItsHeadersAndAdvertisesRetry(t *testing.T) {
	e := newEnv(t)
	s := e.openStream(t, e.login("alice"), "")
	s.await(t, "retry: 3000")

	for key, want := range map[string]string{
		"Content-Type":      "text/event-stream",
		"Cache-Control":     "no-cache",
		"X-Accel-Buffering": "no",
	} {
		if got := s.rec.headerValue(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if got := s.rec.status(); got != http.StatusOK {
		t.Errorf("status = %d, want 200", got)
	}
}

// The hub fans every event to every subscriber; it has no idea who may see
// what. Filtering is the stream's job, and it is the same policy the listing
// uses — an unfiltered stream is the unscoped listing arriving over a different
// pipe, where nobody thinks to look for it.
func TestStreamSendsOnlyEventsTheSubscriberMayRead(t *testing.T) {
	e := newEnv(t)
	bobTask := e.seedTask("bob", "bob's secret plan")
	aliceTask := e.seedTask("alice", "alice's own task")

	s := e.openStream(t, e.login("alice"), "")
	s.await(t, "retry:")

	e.hub.Publish(taskEvent(EventTaskCreated, bobTask))
	e.hub.Publish(taskEvent(EventTaskCreated, aliceTask))

	s.await(t, "alice's own task")
	if s.sent("bob's secret plan") {
		t.Errorf("a member's stream carried another member's task:\n%s", s.seen.String())
	}
}

func TestStreamSendsEverythingToAnAuditor(t *testing.T) {
	e := newEnv(t)
	bobTask := e.seedTask("bob", "bob's secret plan")

	s := e.openStream(t, e.login("iris"), "")
	s.await(t, "retry:")

	e.hub.Publish(taskEvent(EventTaskCreated, bobTask))
	s.await(t, "bob's secret plan")
}

func TestStreamReplaysWhatTheClientMissed(t *testing.T) {
	e := newEnv(t)
	aliceTask := e.seedTask("alice", "alice one")
	bobTask := e.seedTask("bob", "bob one")

	first := e.hub.Publish(taskEvent(EventTaskCreated, aliceTask))
	e.hub.Publish(taskEvent(EventTaskCreated, bobTask))
	missedByAlice := e.hub.Publish(taskEvent(EventTaskUpdated, aliceTask))

	s := e.openStream(t, e.login("alice"), first.ID)
	s.await(t, "id: "+missedByAlice.ID)
	if s.sent("bob one") {
		t.Errorf("the replay was not filtered — a reconnecting member got another member's events:\n%s", s.seen.String())
	}
}

func TestStreamAsksForAResyncWhenTheBacklogCannotReachBack(t *testing.T) {
	e := newEnv(t)
	s := e.openStream(t, e.login("alice"), "an-id-from-an-hour-ago")

	s.await(t, "event: resync")
}

func TestStreamHeartbeatsFromTheInjectedClock(t *testing.T) {
	e := newEnv(t)
	s := e.openStream(t, e.login("alice"), "")
	s.await(t, "retry:")

	e.clock.tick(t)
	s.await(t, ": heartbeat")
}

func TestStreamUnsubscribesWhenTheClientGoesAway(t *testing.T) {
	e := newEnv(t)
	s := e.openStream(t, e.login("alice"), "")
	s.await(t, "retry:")

	if n := e.hub.Subscribers(); n != 1 {
		t.Fatalf("hub has %d subscriber(s), want 1", n)
	}
	s.cancel()
	select {
	case <-s.done:
	case <-time.After(hangGuard):
		t.Fatal("the handler is still running: the streaming loop must return when the request context is cancelled")
	}
	if n := e.hub.Subscribers(); n != 0 {
		t.Errorf("hub has %d subscriber(s) after the client left, want 0: every exit path unsubscribes", n)
	}
}

// Destroying a user's sessions is the service's revocation primitive, and a
// revocation that a live connection ignores is not one. The identity behind an
// open stream has to be re-resolved, not captured when the connection opened:
// otherwise a demoted auditor keeps reading everybody's tasks over a pipe that
// was authorized hours ago.
func TestARoleChangeEndsTheTargetsStream(t *testing.T) {
	e := newEnv(t)
	iris := e.login("iris") // an auditor: entitled to every event
	s := e.openStream(t, iris, "")
	s.await(t, "retry:")

	rec := e.do(http.MethodPost, "/users/"+e.users["iris"].ID+"/role", SetRoleRequest{Role: RoleMember}, e.login("root"))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /users/{id}/role = %d, want 200: body = %s", rec.Code, rec.Body.String())
	}

	bobTask := e.seedTask("bob", "bob's secret plan")
	e.hub.Publish(taskEvent(EventTaskCreated, bobTask))

	select {
	case <-s.done:
	case <-time.After(hangGuard):
		t.Fatal("the stream is still running after its session was destroyed: the subject must be re-resolved per event, " +
			"not captured when the connection opened")
	}
	if s.sent("bob's secret plan") {
		t.Errorf("a revoked stream carried an event its subscriber may no longer read:\n%s", s.seen.String())
	}
}

// The same rule with nothing to publish: an idle stream notices on its next
// heartbeat, which is what bounds how long a revoked tab keeps its connection.
func TestAnIdleStreamEndsOnTheHeartbeatAfterALogout(t *testing.T) {
	e := newEnv(t)
	alice := e.login("alice")
	s := e.openStream(t, alice, "")
	s.await(t, "retry:")

	if rec := e.do(http.MethodPost, "/logout", nil, alice); rec.Code != http.StatusNoContent {
		t.Fatalf("POST /logout = %d, want 204: body = %s", rec.Code, rec.Body.String())
	}

	e.clock.tick(t)
	select {
	case <-s.done:
	case <-time.After(hangGuard):
		t.Fatal("the stream survived a logout: the heartbeat is also when an idle stream re-checks its session")
	}
}

func TestStreamRefusesWhenTheHubIsShuttingDown(t *testing.T) {
	e := newEnv(t)
	cookie := e.login("alice")
	e.hub.Shutdown()

	r := e.request(http.MethodGet, "/events", nil)
	r.AddCookie(cookie)
	rec := e.serveGuarded(t, r, "a subscribe that fails must answer, not stream")
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 while the hub is closed", rec.Code)
	}
}

// The whole point of publishing after the commit: what a subscriber sees is
// what actually happened.
func TestCreateAnnouncesTheTaskAfterItIsStored(t *testing.T) {
	e := newEnv(t)
	watcher := e.watch()

	rec := e.do(http.MethodPost, "/tasks", CreateTaskRequest{Title: "ship it"}, e.login("alice"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: body = %s", rec.Code, rec.Body.String())
	}
	var task Task
	decodeData(t, rec, &task)

	ev := nextEvent(t, watcher)
	if ev.Name != EventTaskCreated {
		t.Errorf("event name = %q, want %q", ev.Name, EventTaskCreated)
	}
	if ev.OwnerID != task.OwnerID {
		t.Errorf("event owner = %q, want %q: the owner is what every stream filters on", ev.OwnerID, task.OwnerID)
	}
	if ev.ID == "" || !strings.Contains(ev.Data, task.ID) {
		t.Errorf("event = %+v, want an identified event carrying the task", ev)
	}
	if _, err := e.store.TaskByID(context.Background(), task.ID); err != nil {
		t.Errorf("the task was announced but is not stored: %v", err)
	}
}

func TestUpdateAnnouncesTheNewState(t *testing.T) {
	e := newEnv(t)
	task := e.seedTask("alice", "alice's task")
	watcher := e.watch()

	rec := e.do(http.MethodPatch, "/tasks/"+task.ID, UpdateTaskRequest{State: StateDone}, e.login("alice"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: body = %s", rec.Code, rec.Body.String())
	}
	ev := nextEvent(t, watcher)
	if ev.Name != EventTaskUpdated || !strings.Contains(ev.Data, string(StateDone)) {
		t.Errorf("event = %+v, want a %s carrying the new state", ev, EventTaskUpdated)
	}
}

func TestADeniedUpdateAnnouncesNothing(t *testing.T) {
	e := newEnv(t)
	task := e.seedTask("alice", "alice's task")
	watcher := e.watch()

	if rec := e.do(http.MethodPatch, "/tasks/"+task.ID, UpdateTaskRequest{State: StateDone}, e.login("bob")); rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	noEvent(t, watcher)
}
