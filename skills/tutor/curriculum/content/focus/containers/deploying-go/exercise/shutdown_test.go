package main

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testApp() *app {
	return newApp(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError})))
}

func status(t *testing.T, a *app, path string) int {
	t.Helper()
	rec := httptest.NewRecorder()
	a.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code
}

func TestReadyzReportsDraining(t *testing.T) {
	a := testApp()

	a.ready.Store(true)
	if got := status(t, a, "/readyz"); got != http.StatusOK {
		t.Fatalf("GET /readyz while serving: got %d, want %d", got, http.StatusOK)
	}

	a.ready.Store(false)
	if got := status(t, a, "/readyz"); got != http.StatusServiceUnavailable {
		t.Errorf("GET /readyz while draining: got %d, want %d — readiness is how a pod asks to be removed from the Service's endpoints; if it keeps answering 200, new requests keep arriving at a process that is on its way out",
			got, http.StatusServiceUnavailable)
	}
}

func TestHealthzIgnoresReadiness(t *testing.T) {
	a := testApp()
	a.ready.Store(false)

	if got := status(t, a, "/healthz"); got != http.StatusOK {
		t.Errorf("GET /healthz while draining: got %d, want %d — liveness answers \"is this process still working\", and failing it during a drain makes the kubelet restart the container in the middle of finishing real requests",
			got, http.StatusOK)
	}
}

func TestRunDrainsInFlightRequests(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("this environment does not allow listening on localhost: %v", err)
	}

	a := testApp()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runErr := make(chan error, 1)
	go func() { runErr <- a.run(ctx, ln, 5*time.Second) }()

	base := "http://" + ln.Addr().String()
	waitReady(t, base+"/readyz")

	type result struct {
		code int
		err  error
	}
	res := make(chan result, 1)
	go func() {
		resp, err := http.Get(base + "/now?delay=700ms")
		if err != nil {
			res <- result{err: err}
			return
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		res <- result{code: resp.StatusCode}
	}()

	time.Sleep(200 * time.Millisecond) // the slow request is now in flight
	cancel()                           // this is what SIGTERM will do in production

	select {
	case r := <-res:
		if r.err != nil {
			t.Fatalf("the in-flight request failed during shutdown: %v — Close() severs open connections, Shutdown() waits for them to finish", r.err)
		}
		if r.code != http.StatusOK {
			t.Fatalf("the in-flight request got %d, want %d", r.code, http.StatusOK)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the in-flight request never completed")
	}

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("run returned %v, want nil — http.ErrServerClosed is the expected end of a graceful stop, not a failure worth exiting non-zero for", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("run did not return after its context was cancelled")
	}
}

func waitReady(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("the server never became ready on %s", url)
}
