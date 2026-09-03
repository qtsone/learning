package api_test

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"tutor.local/rest-services/api"
	"tutor.local/rest-services/memstore"
	"tutor.local/rest-services/note"
)

// TestMain silences the default logger: the 500 tests deliberately hit the
// internal-error path, which logs the real error.
func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

func newHandler(t *testing.T) http.Handler {
	t.Helper()
	return api.New(note.NewService(memstore.New())).Routes()
}

func do(t *testing.T, h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, r)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

type noteBody struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type errBody struct {
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields"`
}

func decodeData[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var env struct {
		Data T `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("body is not a {\"data\": ...} envelope: %v\nbody: %s", err, rec.Body.String())
	}
	return env.Data
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) errBody {
	t.Helper()
	var env struct {
		Error *errBody `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || env.Error == nil {
		t.Fatalf("body is not an {\"error\": {...}} envelope\nbody: %s", rec.Body.String())
	}
	return *env.Error
}

func TestCreateNote(t *testing.T) {
	h := newHandler(t)
	rec := do(t, h, http.MethodPost, "/notes", `{"title":"first","content":"hello"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /notes status = %d, want %d\nbody: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	n := decodeData[noteBody](t, rec)
	if n.ID != 1 || n.Title != "first" || n.Content != "hello" {
		t.Errorf("created note = %+v, want {ID:1 Title:first Content:hello}", n)
	}
}

func TestCreateRejectsMalformedJSON(t *testing.T) {
	h := newHandler(t)
	rec := do(t, h, http.MethodPost, "/notes", `{"title":`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed JSON: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if e := decodeError(t, rec); e.Message != "invalid JSON" {
		t.Errorf("error message = %q, want %q", e.Message, "invalid JSON")
	}
}

func TestCreateRejectsInvalidDraft(t *testing.T) {
	h := newHandler(t)
	rec := do(t, h, http.MethodPost, "/notes", `{"title":"","content":"x"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid draft: status = %d, want %d\nbody: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	e := decodeError(t, rec)
	if e.Message != "validation failed" {
		t.Errorf("error message = %q, want %q", e.Message, "validation failed")
	}
	if e.Fields["title"] != "required" {
		t.Errorf("fields[title] = %q, want %q (field-level errors, not prose)", e.Fields["title"], "required")
	}
}

func TestListEmpty(t *testing.T) {
	h := newHandler(t)
	rec := do(t, h, http.MethodGet, "/notes", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /notes status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"data":[]}` {
		t.Errorf("GET /notes body = %s, want {\"data\":[]} — an empty list encodes as [], never null", got)
	}
}

func TestListReturnsNotesInIDOrder(t *testing.T) {
	h := newHandler(t)
	do(t, h, http.MethodPost, "/notes", `{"title":"a"}`)
	do(t, h, http.MethodPost, "/notes", `{"title":"b"}`)
	rec := do(t, h, http.MethodGet, "/notes", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /notes status = %d, want %d", rec.Code, http.StatusOK)
	}
	ns := decodeData[[]noteBody](t, rec)
	if len(ns) != 2 {
		t.Fatalf("GET /notes returned %d notes, want 2", len(ns))
	}
	if ns[0].ID != 1 || ns[0].Title != "a" || ns[1].ID != 2 || ns[1].Title != "b" {
		t.Errorf("notes = %+v, want [{1 a} {2 b}] in id order", ns)
	}
}

func TestGetNote(t *testing.T) {
	h := newHandler(t)
	do(t, h, http.MethodPost, "/notes", `{"title":"keep","content":"me"}`)
	rec := do(t, h, http.MethodGet, "/notes/1", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /notes/1 status = %d, want %d", rec.Code, http.StatusOK)
	}
	if n := decodeData[noteBody](t, rec); n.ID != 1 || n.Title != "keep" {
		t.Errorf("note = %+v, want {ID:1 Title:keep}", n)
	}
}

func TestGetMissingNoteIs404(t *testing.T) {
	h := newHandler(t)
	rec := do(t, h, http.MethodGet, "/notes/999", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /notes/999 status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if e := decodeError(t, rec); e.Message != "note not found" {
		t.Errorf("error message = %q, want %q", e.Message, "note not found")
	}
}

func TestGetNonIntegerIDIs400(t *testing.T) {
	h := newHandler(t)
	rec := do(t, h, http.MethodGet, "/notes/abc", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("GET /notes/abc status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if e := decodeError(t, rec); e.Message != "invalid id" {
		t.Errorf("error message = %q, want %q", e.Message, "invalid id")
	}
}

func TestUpdateNote(t *testing.T) {
	h := newHandler(t)
	do(t, h, http.MethodPost, "/notes", `{"title":"old","content":"old"}`)
	rec := do(t, h, http.MethodPut, "/notes/1", `{"title":"new","content":"newer"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT /notes/1 status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if n := decodeData[noteBody](t, rec); n.ID != 1 || n.Title != "new" || n.Content != "newer" {
		t.Errorf("updated note = %+v, want {ID:1 Title:new Content:newer}", n)
	}

	rec = do(t, h, http.MethodGet, "/notes/1", "")
	if n := decodeData[noteBody](t, rec); n.Title != "new" {
		t.Errorf("after PUT, GET returned title %q, want %q (update must persist)", n.Title, "new")
	}
}

func TestUpdateMissingNoteIs404(t *testing.T) {
	h := newHandler(t)
	rec := do(t, h, http.MethodPut, "/notes/999", `{"title":"ok"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("PUT /notes/999 status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestUpdateRejectsInvalidDraft(t *testing.T) {
	h := newHandler(t)
	do(t, h, http.MethodPost, "/notes", `{"title":"ok"}`)
	rec := do(t, h, http.MethodPut, "/notes/1", `{"title":""}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("PUT with blank title: status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if e := decodeError(t, rec); e.Fields["title"] != "required" {
		t.Errorf("fields[title] = %q, want %q", e.Fields["title"], "required")
	}
}

func TestDeleteNote(t *testing.T) {
	h := newHandler(t)
	do(t, h, http.MethodPost, "/notes", `{"title":"doomed"}`)
	rec := do(t, h, http.MethodDelete, "/notes/1", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE /notes/1 status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("DELETE body = %q, want empty — 204 promises no content", rec.Body.String())
	}

	rec = do(t, h, http.MethodGet, "/notes/1", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("GET after DELETE status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestDeleteMissingNoteIs404(t *testing.T) {
	h := newHandler(t)
	rec := do(t, h, http.MethodDelete, "/notes/999", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("DELETE /notes/999 status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestWrongMethodIs405(t *testing.T) {
	h := newHandler(t)
	rec := do(t, h, http.MethodPatch, "/notes", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PATCH /notes status = %d, want %d — method patterns give this for free", rec.Code, http.StatusMethodNotAllowed)
	}
}

// explodingStore simulates storage that has fallen over with an error whose
// text must never reach a client.
type explodingStore struct{}

var errStoreDown = errors.New("dial tcp 10.0.0.7:5432: connection refused")

func (explodingStore) Create(note.Draft) (note.Note, error) { return note.Note{}, errStoreDown }
func (explodingStore) Get(int64) (note.Note, error)         { return note.Note{}, errStoreDown }
func (explodingStore) List() ([]note.Note, error)           { return nil, errStoreDown }
func (explodingStore) Update(int64, note.Draft) (note.Note, error) {
	return note.Note{}, errStoreDown
}
func (explodingStore) Delete(int64) error { return errStoreDown }

func TestStoreFailureIsOpaque500(t *testing.T) {
	h := api.New(note.NewService(explodingStore{})).Routes()
	rec := do(t, h, http.MethodGet, "/notes", "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("store failure: status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if e := decodeError(t, rec); e.Message != "internal error" {
		t.Errorf("error message = %q, want exactly %q", e.Message, "internal error")
	}
	if body := rec.Body.String(); strings.Contains(body, "connection refused") || strings.Contains(body, "10.0.0.7") {
		t.Errorf("response leaked internal error details: %s", body)
	}
}
