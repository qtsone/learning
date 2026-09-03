package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func serve(t *testing.T, mux *http.ServeMux, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(method, target, nil))
	return rec
}

func TestRoutes(t *testing.T) {
	mux := NewMux("1.2.3")
	cases := []struct {
		name       string
		method     string
		target     string
		wantStatus int
		wantBody   string
	}{
		{"home", "GET", "/", http.StatusOK, "greeter service"},
		{"health", "GET", "/health", http.StatusOK, "ok"},
		{"version", "GET", "/version", http.StatusOK, "1.2.3"},
		{"greet", "GET", "/greet/Ada", http.StatusOK, "Hello, Ada!"},
		{"greet another name", "GET", "/greet/Gopher", http.StatusOK, "Hello, Gopher!"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := serve(t, mux, c.method, c.target)
			if rec.Code != c.wantStatus {
				t.Fatalf("%s %s: status = %d, want %d", c.method, c.target, rec.Code, c.wantStatus)
			}
			if got := rec.Body.String(); got != c.wantBody {
				t.Errorf("%s %s: body = %q, want %q", c.method, c.target, got, c.wantBody)
			}
		})
	}
}

func TestRoutingEdges(t *testing.T) {
	mux := NewMux("1.2.3")

	t.Run("unknown path is 404, not the home page", func(t *testing.T) {
		if rec := serve(t, mux, "GET", "/nope"); rec.Code != http.StatusNotFound {
			t.Errorf("GET /nope: status = %d, want 404 (register the home page as {$}, not /)", rec.Code)
		}
	})

	t.Run("wrong method is 405", func(t *testing.T) {
		if rec := serve(t, mux, "POST", "/health"); rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST /health: status = %d, want 405 (method prefixes give you this for free)", rec.Code)
		}
	})
}
