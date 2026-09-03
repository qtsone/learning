package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type release struct {
	Name  string `json:"name"`
	Major int    `json:"major"`
}

func TestNewSetsTimeout(t *testing.T) {
	c := New("http://example.invalid")
	if c.HTTP == nil {
		t.Fatal("New: c.HTTP is nil, want a configured *http.Client")
	}
	if c.HTTP.Timeout != 10*time.Second {
		t.Errorf("New: c.HTTP.Timeout = %v, want %v (a client without a timeout can hang forever)",
			c.HTTP.Timeout, 10*time.Second)
	}
}

func TestGetJSONDecodesSuccess(t *testing.T) {
	var accept atomic.Value
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept.Store(r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"name":"go","major":1}`)
	}))
	defer ts.Close()

	c := New(ts.URL)
	var rel release
	if err := c.GetJSON(context.Background(), "/releases/latest", &rel); err != nil {
		t.Fatalf("GetJSON on a 200 JSON response returned %v, want nil", err)
	}
	if rel.Name != "go" || rel.Major != 1 {
		t.Errorf("decoded %+v, want {Name:go Major:1}", rel)
	}
	if got := accept.Load(); got != "application/json" {
		t.Errorf("Accept header = %q, want %q", got, "application/json")
	}
}

func TestGetJSONNon2xxReturnsAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "release not found", http.StatusNotFound)
	}))
	defer ts.Close()

	c := New(ts.URL)
	var rel release
	err := c.GetJSON(context.Background(), "/nope", &rel)
	if err == nil {
		t.Fatal("GetJSON on a 404 returned nil, want an *APIError")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("GetJSON error = %T (%v), want errors.As to find an *APIError", err, err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusNotFound)
	}
	if !strings.Contains(apiErr.Body, "release not found") {
		t.Errorf("Body = %q, want it to contain the server's message %q", apiErr.Body, "release not found")
	}
	if !strings.Contains(apiErr.Error(), "404") {
		t.Errorf("Error() = %q, want it to mention status 404", apiErr.Error())
	}
}

func TestGetJSONTruncatesHugeErrorBodies(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(bytes.Repeat([]byte("x"), 16*maxErrorBody))
	}))
	defer ts.Close()

	c := New(ts.URL)
	var rel release
	err := c.GetJSON(context.Background(), "/", &rel)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("GetJSON error = %v, want an *APIError for a 500", err)
	}
	if len(apiErr.Body) == 0 || len(apiErr.Body) > maxErrorBody {
		t.Errorf("Body length = %d, want 1..%d (read error bodies through a size limit)",
			len(apiErr.Body), maxErrorBody)
	}
}

func TestGetJSONReportsDecodeErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<html>definitely not json</html>`)
	}))
	defer ts.Close()

	c := New(ts.URL)
	var rel release
	err := c.GetJSON(context.Background(), "/", &rel)
	if err == nil {
		t.Fatal("GetJSON on a 200 with a non-JSON body returned nil, want a decode error")
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Errorf("a broken 200 body is a decode failure, not an *APIError; got %v", err)
	}
}

type closeRecordingBody struct {
	io.ReadCloser
	closed *atomic.Bool
}

func (b *closeRecordingBody) Close() error {
	b.closed.Store(true)
	return b.ReadCloser.Close()
}

type closeRecordingTransport struct {
	closed atomic.Bool
}

func (tr *closeRecordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err == nil {
		resp.Body = &closeRecordingBody{ReadCloser: resp.Body, closed: &tr.closed}
	}
	return resp, err
}

func TestGetJSONClosesBody(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"on success", func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, `{"name":"go","major":1}`)
		}},
		{"on api error", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "boom", http.StatusBadGateway)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(tc.handler)
			defer ts.Close()
			tr := &closeRecordingTransport{}
			c := New(ts.URL)
			c.HTTP.Transport = tr
			var rel release
			_ = c.GetJSON(context.Background(), "/", &rel)
			if !tr.closed.Load() {
				t.Error("response body was never closed — close it on every path, and send the request through c.HTTP")
			}
		})
	}
}

func TestGetJSONHonorsContext(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"name":"go","major":1}`)
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := New(ts.URL)
	var rel release
	err := c.GetJSON(ctx, "/", &rel)
	if err == nil {
		t.Fatal("GetJSON with a cancelled context returned nil, want an error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want errors.Is(err, context.Canceled) — build the request with http.NewRequestWithContext", err)
	}
}
