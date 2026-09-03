package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"tutor.local/project-service/sqlitestore"
	"tutor.local/project-service/task"
)

// testToken is the only credential these tests know. It is a fixture, not a
// default: nothing in the service ships with a token baked in.
const testToken = "test-token-2f9c"

// clockStart is the instant the harness clock starts at; each call to the
// clock advances it by a minute, so "created" and "updated" stamps differ
// without any test ever waiting for real time.
var clockStart = time.Date(2024, 5, 1, 9, 0, 0, 0, time.UTC)

type harness struct {
	handler http.Handler
	metrics *Metrics
	store   *sqlitestore.Store
	logs    *bytes.Buffer
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	calls := 0
	clock := func() time.Time {
		t := clockStart.Add(time.Duration(calls) * time.Minute)
		calls++
		return t
	}

	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	metrics := NewMetrics()
	srv := New(task.NewService(store, clock), metrics, logger)

	return &harness{
		handler: srv.Routes(Auth(map[string]string{"tests": testToken})),
		metrics: metrics,
		store:   store,
		logs:    logs,
	}
}

// do sends an authenticated request through the whole router.
func (h *harness) do(t *testing.T, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+testToken)
	return h.serve(t, req)
}

// serve sends a request exactly as given — no credentials added.
func (h *harness) serve(t *testing.T, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

type dataEnvelope struct {
	Data json.RawMessage `json:"data"`
}

type errorEnvelope struct {
	Error struct {
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields"`
	} `json:"error"`
}

func decodeTask(t *testing.T, rec *httptest.ResponseRecorder) task.Task {
	t.Helper()
	var env dataEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding %q: %v", rec.Body.String(), err)
	}
	var tk task.Task
	if err := json.Unmarshal(env.Data, &tk); err != nil {
		t.Fatalf("decoding data of %q: %v", rec.Body.String(), err)
	}
	return tk
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) errorEnvelope {
	t.Helper()
	var env errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding %q: %v", rec.Body.String(), err)
	}
	return env
}

func checkStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, want, rec.Body.String())
	}
}

func (h *harness) createTask(t *testing.T, title string) task.Task {
	t.Helper()
	rec := h.do(t, http.MethodPost, "/tasks", `{"title":`+strconv.Quote(title)+`}`)
	checkStatus(t, rec, http.StatusCreated)
	return decodeTask(t, rec)
}

func TestCreateReturns201AndTheStoredTask(t *testing.T) {
	h := newHarness(t)

	rec := h.do(t, http.MethodPost, "/tasks", `{"title":"  ship the capstone  "}`)

	checkStatus(t, rec, http.StatusCreated)
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	got := decodeTask(t, rec)
	if got.ID == 0 {
		t.Error("id = 0, want the id storage assigned")
	}
	if got.Title != "ship the capstone" {
		t.Errorf("title = %q, want %q", got.Title, "ship the capstone")
	}
	if got.Status != task.StatusOpen {
		t.Errorf("status = %q, want %q", got.Status, task.StatusOpen)
	}
	if !got.CreatedAt.Equal(clockStart) {
		t.Errorf("created_at = %v, want %v", got.CreatedAt, clockStart)
	}
}

func TestCreateRejectsMalformedJSON(t *testing.T) {
	h := newHarness(t)

	rec := h.do(t, http.MethodPost, "/tasks", `{"title":`)

	checkStatus(t, rec, http.StatusBadRequest)
	if got := decodeError(t, rec).Error.Message; got != "invalid JSON" {
		t.Errorf("message = %q, want %q", got, "invalid JSON")
	}
}

func TestCreateReportsFieldErrors(t *testing.T) {
	h := newHarness(t)

	rec := h.do(t, http.MethodPost, "/tasks", `{"title":"   "}`)

	checkStatus(t, rec, http.StatusBadRequest)
	env := decodeError(t, rec)
	if env.Error.Message != "validation failed" {
		t.Errorf("message = %q, want %q", env.Error.Message, "validation failed")
	}
	if env.Error.Fields["title"] != "required" {
		t.Errorf("fields = %v, want title: %q", env.Error.Fields, "required")
	}
}

func TestListIsAnArrayEvenWhenEmpty(t *testing.T) {
	h := newHarness(t)

	rec := h.do(t, http.MethodGet, "/tasks", "")

	checkStatus(t, rec, http.StatusOK)
	if got := strings.TrimSpace(rec.Body.String()); got != `{"data":[]}` {
		t.Errorf("body = %s, want %s (a nil slice encodes as null and breaks clients)", got, `{"data":[]}`)
	}
}

func TestListSortsAndFilters(t *testing.T) {
	h := newHarness(t)
	first := h.createTask(t, "first")
	second := h.createTask(t, "second")
	third := h.createTask(t, "third")

	rec := h.do(t, http.MethodPatch, "/tasks/"+strconv.FormatInt(second.ID, 10), `{"status":"done"}`)
	checkStatus(t, rec, http.StatusOK)

	all := decodeTasks(t, h.do(t, http.MethodGet, "/tasks", ""))
	if len(all) != 3 {
		t.Fatalf("GET /tasks returned %d tasks, want 3", len(all))
	}
	if all[0].ID != first.ID || all[1].ID != second.ID || all[2].ID != third.ID {
		t.Errorf("ids = %d, %d, %d, want ascending %d, %d, %d",
			all[0].ID, all[1].ID, all[2].ID, first.ID, second.ID, third.ID)
	}

	open := decodeTasks(t, h.do(t, http.MethodGet, "/tasks?status=open", ""))
	if len(open) != 2 {
		t.Fatalf("GET /tasks?status=open returned %d tasks, want 2", len(open))
	}
	done := decodeTasks(t, h.do(t, http.MethodGet, "/tasks?status=done", ""))
	if len(done) != 1 || done[0].ID != second.ID {
		t.Errorf("GET /tasks?status=done returned %v, want just task %d", done, second.ID)
	}
}

func TestListRejectsAnUnknownFilter(t *testing.T) {
	h := newHarness(t)

	rec := h.do(t, http.MethodGet, "/tasks?status=archived", "")

	checkStatus(t, rec, http.StatusBadRequest)
	env := decodeError(t, rec)
	if env.Error.Fields["status"] == "" {
		t.Errorf("fields = %v, want an entry for status", env.Error.Fields)
	}
}

func TestGetReturnsTheTask(t *testing.T) {
	h := newHarness(t)
	created := h.createTask(t, "read me back")

	rec := h.do(t, http.MethodGet, "/tasks/"+strconv.FormatInt(created.ID, 10), "")

	checkStatus(t, rec, http.StatusOK)
	if got := decodeTask(t, rec); got.ID != created.ID || got.Title != created.Title {
		t.Errorf("got %+v, want %+v", got, created)
	}
}

func TestUnknownIDIs404(t *testing.T) {
	h := newHarness(t)

	rec := h.do(t, http.MethodGet, "/tasks/999", "")

	checkStatus(t, rec, http.StatusNotFound)
	if got := decodeError(t, rec).Error.Message; got != "task not found" {
		t.Errorf("message = %q, want %q", got, "task not found")
	}
}

func TestNonIntegerIDIs400(t *testing.T) {
	h := newHarness(t)

	rec := h.do(t, http.MethodGet, "/tasks/abc", "")

	checkStatus(t, rec, http.StatusBadRequest)
	if got := decodeError(t, rec).Error.Message; got != "invalid id" {
		t.Errorf("message = %q, want %q", got, "invalid id")
	}
}

func TestPatchCompletesATask(t *testing.T) {
	h := newHarness(t)
	created := h.createTask(t, "finish me")

	rec := h.do(t, http.MethodPatch, "/tasks/"+strconv.FormatInt(created.ID, 10), `{"status":"done"}`)

	checkStatus(t, rec, http.StatusOK)
	got := decodeTask(t, rec)
	if got.Status != task.StatusDone {
		t.Errorf("status = %q, want %q", got.Status, task.StatusDone)
	}
	if !got.UpdatedAt.After(created.UpdatedAt) {
		t.Errorf("updated_at = %v, want later than the create stamp %v", got.UpdatedAt, created.UpdatedAt)
	}
}

func TestPatchReopeningIs409(t *testing.T) {
	h := newHarness(t)
	created := h.createTask(t, "finish me")
	id := strconv.FormatInt(created.ID, 10)
	checkStatus(t, h.do(t, http.MethodPatch, "/tasks/"+id, `{"status":"done"}`), http.StatusOK)

	rec := h.do(t, http.MethodPatch, "/tasks/"+id, `{"status":"open"}`)

	checkStatus(t, rec, http.StatusConflict)
	if got := decodeError(t, rec).Error.Message; got != "task is already done" {
		t.Errorf("message = %q, want %q", got, "task is already done")
	}
}

func TestPatchRejectsAnUnknownStatus(t *testing.T) {
	h := newHarness(t)
	created := h.createTask(t, "finish me")

	rec := h.do(t, http.MethodPatch, "/tasks/"+strconv.FormatInt(created.ID, 10), `{"status":"archived"}`)

	checkStatus(t, rec, http.StatusBadRequest)
	if got := decodeError(t, rec).Error.Fields["status"]; got == "" {
		t.Errorf("fields = %v, want an entry for status", decodeError(t, rec).Error.Fields)
	}
}

func TestDeleteReturns204WithNoBody(t *testing.T) {
	h := newHarness(t)
	created := h.createTask(t, "temporary")
	id := strconv.FormatInt(created.ID, 10)

	rec := h.do(t, http.MethodDelete, "/tasks/"+id, "")

	checkStatus(t, rec, http.StatusNoContent)
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty: 204 promises no body", rec.Body.String())
	}
	checkStatus(t, h.do(t, http.MethodGet, "/tasks/"+id, ""), http.StatusNotFound)
	checkStatus(t, h.do(t, http.MethodDelete, "/tasks/"+id, ""), http.StatusNotFound)
}

// 404 for unknown paths and 405 for a wrong method are the mux's job. Both
// bodies come from net/http, so only the status is asserted.
func TestRoutingFailuresComeFromTheMux(t *testing.T) {
	h := newHarness(t)

	checkStatus(t, h.do(t, http.MethodGet, "/nope", ""), http.StatusNotFound)
	checkStatus(t, h.do(t, http.MethodPut, "/tasks/1", ""), http.StatusMethodNotAllowed)
}

// failingStore is a task.Store whose every method fails, so the 500 path can
// be exercised without breaking a real database.
type failingStore struct{ err error }

func (f failingStore) Create(context.Context, task.Task) (task.Task, error) {
	return task.Task{}, f.err
}
func (f failingStore) Get(context.Context, int64) (task.Task, error) { return task.Task{}, f.err }
func (f failingStore) List(context.Context, task.Status) ([]task.Task, error) {
	return nil, f.err
}
func (f failingStore) SetStatus(context.Context, int64, task.Status, time.Time) (task.Task, error) {
	return task.Task{}, f.err
}
func (f failingStore) Delete(context.Context, int64) error { return f.err }
func (f failingStore) Ping(context.Context) error          { return f.err }

func TestStorageFailureIs500AndLeaksNothing(t *testing.T) {
	secret := errors.New("dial tcp 10.0.0.7:5432: connection refused")
	logs := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	srv := New(task.NewService(failingStore{err: secret}, nil), NewMetrics(), logger)
	handler := srv.Routes(Auth(map[string]string{"tests": testToken}))

	req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
	req.Header.Set("Authorization", "Bearer "+testToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	checkStatus(t, rec, http.StatusInternalServerError)
	if got := strings.TrimSpace(rec.Body.String()); got != `{"error":{"message":"internal error"}}` {
		t.Errorf("body = %s, want %s", got, `{"error":{"message":"internal error"}}`)
	}
	if strings.Contains(rec.Body.String(), "10.0.0.7") {
		t.Error("the response body leaks the underlying error; the client gets a generic message")
	}
	if !strings.Contains(logs.String(), "10.0.0.7") {
		t.Errorf("the log does not contain the real error: %s\nthe client gets a generic message, the log gets the truth", logs.String())
	}
}

func TestHealthzNeedsNoCredentials(t *testing.T) {
	h := newHarness(t)

	rec := h.serve(t, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	checkStatus(t, rec, http.StatusOK)
}

// Liveness answers "the process is up"; readiness answers "it can serve
// traffic". They differ exactly when a dependency is broken, and a
// restarting service depends on the difference.
func TestReadyzFollowsTheDatabase(t *testing.T) {
	h := newHarness(t)

	checkStatus(t, h.serve(t, httptest.NewRequest(http.MethodGet, "/readyz", nil)), http.StatusOK)

	if err := h.store.Close(); err != nil {
		t.Fatalf("closing the store: %v", err)
	}

	rec := h.serve(t, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	checkStatus(t, rec, http.StatusServiceUnavailable)
	checkStatus(t, h.serve(t, httptest.NewRequest(http.MethodGet, "/healthz", nil)), http.StatusOK)
}

func TestTaskRoutesRequireAuthentication(t *testing.T) {
	h := newHarness(t)

	cases := []struct {
		name   string
		header string
	}{
		{"no header", ""},
		{"wrong scheme", "Basic " + testToken},
		{"unknown token", "Bearer not-the-token"},
		{"empty token", "Bearer "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := h.serve(t, req)
			checkStatus(t, rec, http.StatusUnauthorized)
			if got := rec.Header().Get("WWW-Authenticate"); !strings.Contains(got, "Bearer") {
				t.Errorf("WWW-Authenticate = %q, want it to name the Bearer scheme", got)
			}
			if got := decodeError(t, rec).Error.Message; got != "unauthorized" {
				t.Errorf("message = %q, want %q", got, "unauthorized")
			}
		})
	}
}

func decodeTasks(t *testing.T, rec *httptest.ResponseRecorder) []task.Task {
	t.Helper()
	var env dataEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decoding %q: %v", rec.Body.String(), err)
	}
	var ts []task.Task
	if err := json.Unmarshal(env.Data, &ts); err != nil {
		t.Fatalf("decoding data of %q: %v", rec.Body.String(), err)
	}
	return ts
}
