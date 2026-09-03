package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const addr = "127.0.0.1:8080"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	// Cancelled the moment Ctrl-C or SIGTERM arrives, which is all the rest
	// of the program needs to know about signals.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
