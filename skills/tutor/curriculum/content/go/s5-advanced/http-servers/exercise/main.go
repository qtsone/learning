package main

import (
	"context"
	"log/slog"
	"net"
	"os"
)

const addr = "127.0.0.1:8080"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	// TODO: make this context signal-aware with signal.NotifyContext so
	// Ctrl-C (os.Interrupt) and `kill` (syscall.SIGTERM) trigger a graceful
	// shutdown instead of killing the process mid-request.
	ctx := context.Background()

	mux := NewMux("0.1.0")
	handler := Chain(mux, RequestID, Logging(logger), Recover)
	srv := NewServer(addr, handler)

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		logger.Error("listen failed", "addr", srv.Addr, "err", err)
		os.Exit(1)
	}
	logger.Info("listening", "addr", ln.Addr().String())

	if err := Run(ctx, srv, ln); err != nil {
		logger.Error("server error", "err", err)
		os.Exit(1)
	}
	logger.Info("shutdown complete")
}
