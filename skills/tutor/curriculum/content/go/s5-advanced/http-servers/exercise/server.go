package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

// shutdownGrace is how long Run gives in-flight requests to finish once
// shutdown begins.
const shutdownGrace = 10 * time.Second

// NewServer builds the production http.Server for this service: handler
// attached, every timeout set (a zero timeout means wait forever).
func NewServer(addr string, h http.Handler) *http.Server {
	// TODO: set ReadTimeout (5s), WriteTimeout (10s), IdleTimeout (120s).
	return &http.Server{
		Addr:    addr,
		Handler: h,
	}
}

// Run serves srv on ln until ctx is cancelled, then shuts down gracefully:
// stop accepting new connections, let in-flight requests finish (up to
// shutdownGrace), and return nil on a clean exit.
func Run(ctx context.Context, srv *http.Server, ln net.Listener) error {
	// TODO:
	//   1. srv.Serve(ln) in a goroutine; send its error on a channel.
	//   2. Wait for ctx.Done() — or for Serve to fail on its own first.
	//   3. srv.Shutdown with a shutdownGrace deadline.
	//   4. Drain the Serve error: http.ErrServerClosed after a clean
	//      Shutdown means success, not failure.
	return errors.New("not implemented")
}
