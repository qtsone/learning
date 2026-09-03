package httpapi

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// NewServer and Run ship complete. These tests are green from the start on
// purpose: run them, read what they assert, and use what you see to write the
// shutdown section of NOTES.md.

// runWait is a deliberately generous ceiling on "Run should have returned by
// now". It is not a performance assertion: these tests never depend on how
// long anything takes, only on it finishing at all, so the race detector's
// slowdown cannot make them flaky.
const runWait = 15 * time.Second

func TestNewServerSetsEveryTimeout(t *testing.T) {
	mux := http.NewServeMux()
	srv := NewServer("127.0.0.1:9999", mux)

	if srv.Addr != "127.0.0.1:9999" || srv.Handler != http.Handler(mux) {
		t.Errorf("Addr = %q and Handler = %v, want the values passed in", srv.Addr, srv.Handler)
	}
	for _, tt := range []struct {
		name      string
		got, want time.Duration
	}{
		{"ReadTimeout", srv.ReadTimeout, 5 * time.Second},
		{"WriteTimeout", srv.WriteTimeout, 10 * time.Second},
		{"IdleTimeout", srv.IdleTimeout, 120 * time.Second},
	} {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v (zero means wait forever)", tt.name, tt.got, tt.want)
		}
	}
	if shutdownGrace <= srv.WriteTimeout {
		t.Errorf("shutdownGrace (%v) is not longer than WriteTimeout (%v): shutdown would cut off requests that are still allowed to run",
			shutdownGrace, srv.WriteTimeout)
	}
}

func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return ln
}

func newClient(t *testing.T) *http.Client {
	t.Helper()
	transport := &http.Transport{}
	t.Cleanup(transport.CloseIdleConnections)
	return &http.Client{Transport: transport, Timeout: runWait}
}

func waitRun(t *testing.T, errc <-chan error) error {
	t.Helper()
	select {
	case err := <-errc:
		return err
	case <-time.After(runWait):
		t.Fatal("Run never returned: after ctx is cancelled it must call srv.Shutdown and return")
		return nil
	}
}

func TestRunServesUntilTheContextIsCancelled(t *testing.T) {
	ln := listen(t)
	addr := ln.Addr().String()
	srv := NewServer(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "alive")
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errc := make(chan error, 1)
	go func() { errc <- Run(ctx, srv, ln) }()

	resp, err := newClient(t).Get("http://" + addr + "/healthz")
	if err != nil {
		t.Fatalf("request to the running server failed: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil || string(body) != "alive" {
		t.Fatalf("body = %q (err %v), want %q", body, err, "alive")
	}

	cancel()
	if err := waitRun(t, errc); err != nil {
		t.Fatalf("Run returned %v, want nil: http.ErrServerClosed after a clean Shutdown is success", err)
	}
	if c, err := net.DialTimeout("tcp", addr, 2*time.Second); err == nil {
		c.Close()
		t.Error("the address still accepts connections after Run returned")
	}
}

// A deploy restarts the process while requests are in flight. Those requests
// must finish, not be cut off.
func TestRunDrainsInFlightRequests(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})

	ln := listen(t)
	addr := ln.Addr().String()
	srv := NewServer(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		io.WriteString(w, "finished")
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errc := make(chan error, 1)
	go func() { errc <- Run(ctx, srv, ln) }()

	type result struct {
		body string
		err  error
	}
	res := make(chan result, 1)
	go func() {
		resp, err := newClient(t).Get("http://" + addr + "/tasks")
		if err != nil {
			res <- result{err: err}
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		res <- result{body: string(body), err: err}
	}()

	select {
	case <-started:
	case r := <-res:
		t.Fatalf("the request never reached a handler (%v): Run must serve on ln", r.err)
	case <-time.After(runWait):
		t.Fatal("the request never reached a handler: Run must serve on ln")
	}
	cancel()       // shutdown begins with a request in flight
	close(release) // let the handler produce its response

	select {
	case r := <-res:
		if r.err != nil {
			t.Fatalf("in-flight request failed during shutdown: %v", r.err)
		}
		if r.body != "finished" {
			t.Errorf("in-flight response body = %q, want %q", r.body, "finished")
		}
	case <-time.After(runWait):
		t.Fatal("the in-flight request never completed")
	}
	if err := waitRun(t, errc); err != nil {
		t.Fatalf("Run returned %v, want nil after a clean shutdown", err)
	}
}

func TestRunReportsAServeFailure(t *testing.T) {
	ln := listen(t)
	addr := ln.Addr().String()
	ln.Close() // Serve on this listener fails immediately

	errc := make(chan error, 1)
	go func() { errc <- Run(context.Background(), NewServer(addr, http.NotFoundHandler()), ln) }()

	select {
	case err := <-errc:
		if err == nil {
			t.Error("Run returned nil, want the Serve error: only http.ErrServerClosed after a Shutdown is success")
		}
	case <-time.After(runWait):
		t.Fatal("Run blocked forever after Serve failed — wait on the Serve error as well as on ctx.Done()")
	}
}
