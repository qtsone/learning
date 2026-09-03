package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

// runWait is a deliberately generous ceiling on "Run should have returned by
// now". It is not a performance assertion: the tests never depend on how
// long anything takes, only on it finishing at all, so the race detector's
// slowdown cannot make them flaky.
const runWait = 15 * time.Second

func TestNewServer(t *testing.T) {
	mux := http.NewServeMux()
	srv := NewServer("127.0.0.1:9999", mux)

	if srv.Addr != "127.0.0.1:9999" {
		t.Errorf("Addr = %q, want %q", srv.Addr, "127.0.0.1:9999")
	}
	if srv.Handler != http.Handler(mux) {
		t.Error("Handler is not the handler that was passed in")
	}

	timeouts := []struct {
		name string
		got  time.Duration
		want time.Duration
	}{
		{"ReadTimeout", srv.ReadTimeout, 5 * time.Second},
		{"WriteTimeout", srv.WriteTimeout, 10 * time.Second},
		{"IdleTimeout", srv.IdleTimeout, 120 * time.Second},
	}
	for _, tt := range timeouts {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v (zero means wait forever)", tt.name, tt.got, tt.want)
		}
	}
}

// listen returns a loopback listener on an OS-picked free port: no fixed
// port to collide with, no traffic beyond 127.0.0.1.
func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return ln
}

// newClient returns a client with its own connection pool, so one test's
// keep-alive connections never reach another test's server. The Timeout is a
// safety net, not an assertion: a served request comes back in microseconds,
// but a Run that never calls Serve would otherwise leave the request parked
// in the kernel's accept queue forever.
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

func TestRunServesUntilContextCancelled(t *testing.T) {
	ln := listen(t)
	addr := ln.Addr().String()
	srv := NewServer(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "alive")
	}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errc := make(chan error, 1)
	go func() { errc <- Run(ctx, srv, ln) }()

	// A correct Run does not return until ctx is cancelled, so anything on
	// errc now means it gave up early — most likely it is still the starter.
	// Catching that here fails in milliseconds instead of parking the request
	// below in the accept queue until the ceiling.
	select {
	case err := <-errc:
		t.Fatalf("Run returned %v before serving anything: it must Serve on ln and block until ctx is done", err)
	case <-time.After(50 * time.Millisecond):
	}

	client := newClient(t)

	resp, err := client.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("request to the running server failed: %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("reading body: %v", err)
	}
	if string(body) != "alive" {
		t.Errorf("body = %q, want %q", body, "alive")
	}

	cancel()
	if err := waitRun(t, errc); err != nil {
		t.Fatalf("Run returned %v, want nil: http.ErrServerClosed after a clean Shutdown is success, not failure", err)
	}

	if c, err := net.DialTimeout("tcp", addr, 2*time.Second); err == nil {
		c.Close()
		t.Error("the address still accepts connections after Run returned; Shutdown must stop the listener")
	}
}

func TestRunLetsInFlightRequestsFinish(t *testing.T) {
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

	client := newClient(t)

	type result struct {
		body string
		err  error
	}
	res := make(chan result, 1)
	go func() {
		resp, err := client.Get("http://" + addr + "/slow")
		if err != nil {
			res <- result{err: err}
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		res <- result{body: string(body), err: err}
	}()

	select {
	case <-started: // the handler is running inside the server
	case r := <-res:
		t.Fatalf("the request never reached a handler (%v): Run must serve on ln", r.err)
	case err := <-errc:
		t.Fatalf("Run returned %v while a request was pending: it must Serve on ln and block until ctx is done", err)
	case <-time.After(runWait):
		t.Fatal("the request never reached a handler: Run must serve on ln")
	}
	cancel()       // shutdown begins with a request in flight
	close(release) // let the handler produce its response

	select {
	case r := <-res:
		if r.err != nil {
			t.Fatalf("in-flight request failed during shutdown: %v (Shutdown must drain active requests, not cut them off)", r.err)
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

func TestRunReportsServeFailure(t *testing.T) {
	ln := listen(t)
	addr := ln.Addr().String()
	ln.Close() // Serve on this listener fails immediately

	srv := NewServer(addr, http.NotFoundHandler())
	errc := make(chan error, 1)
	go func() { errc <- Run(context.Background(), srv, ln) }()

	select {
	case err := <-errc:
		if err == nil {
			t.Error("Run returned nil, want the Serve error: only http.ErrServerClosed after a Shutdown counts as success")
		}
	case <-time.After(runWait):
		t.Fatal("Run blocked forever after Serve failed — wait on the Serve error as well as on ctx.Done()")
	}
}
