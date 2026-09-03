package httpapi

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

// shutdownGrace is how long Run gives in-flight requests to finish once
// shutdown begins. It must be longer than the request timeout, or shutdown
// cuts off requests that were still allowed to run.
const shutdownGrace = 15 * time.Second

// NewServer builds the production server: every timeout set, because a zero
// timeout means "wait forever" and a server that waits forever is a server
// whose connections belong to whoever asks for them.
func NewServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

// Run serves srv on ln until ctx is cancelled, then shuts down gracefully:
// stop accepting new connections, let in-flight requests finish (up to
// shutdownGrace), and return nil on a clean exit.
//
// This file is PROVIDED, complete — you wrote this loop in the http-servers
// lesson. What is graded here is the argument in NOTES.md: why the three
// budgets are ordered requestTimeout < WriteTimeout < shutdownGrace, and what
// a SIGTERM does to a request that is already in flight. Read the code and
// run server_test.go before you write that section.
func Run(ctx context.Context, srv *http.Server, ln net.Listener) error {
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		// Serve gave up on its own — a dead listener, say. Nobody is going
		// to cancel ctx on our behalf, so report it now.
		return ignoreServerClosed(err)
	case <-ctx.Done():
	}

	// A fresh context: the one that was just cancelled cannot bound the
	// shutdown it triggered.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	// Shutdown has returned, so Serve has already unblocked; draining the
	// channel keeps its goroutine from leaking and surfaces a real failure.
	return ignoreServerClosed(<-serveErr)
}

// ignoreServerClosed treats http.ErrServerClosed as success: it is Serve's
// way of confirming that Shutdown ran, not a failure.
func ignoreServerClosed(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
