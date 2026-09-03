// Command tinysvc is the reference service for the operations checks: config
// from the environment, structured logs, metrics over /metrics, health and
// readiness endpoints, and a shutdown that drains work in flight.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tutor.local/capstone-reference/internal/config"
	"tutor.local/capstone-reference/internal/httpapi"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tinysvc: bad configuration:", err)
		os.Exit(2)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))

	// The signal context is the one place termination enters the program, and
	// the process lifetime starts here — which is why the root belongs in main
	// and run is handed a context it can be tested with.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg, log); err != nil {
		log.Error("stopped with an error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	api := httpapi.New(log)
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Addr)
		api.Ready()
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case err := <-serveErr:
		if err != nil {
			return fmt.Errorf("listen on %s: %w", cfg.Addr, err)
		}
		return nil
	case <-ctx.Done():
		log.Info("signal received, draining", "timeout", cfg.ShutdownTimeout.String())
	}

	// Fail readiness first so the load balancer stops sending new requests,
	// then let the ones already in flight finish inside a bounded window. The
	// drain has to outlive the context that just ended, so it is derived with
	// WithoutCancel rather than started as a fresh root: the deadline is ours
	// on purpose, and everything else about the caller is kept.
	api.NotReady()
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	log.Info("stopped cleanly")
	return nil
}
