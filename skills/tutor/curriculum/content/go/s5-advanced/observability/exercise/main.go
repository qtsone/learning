package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// main is provided: it wires the pieces you are about to write. Read the
// chain — the order of those five middlewares is the whole lesson in one
// line.
func main() {
	level := new(slog.LevelVar) // starts at INFO; flip to DEBUG without a redeploy
	logger := NewLogger(os.Stdout, level)
	slog.SetDefault(logger)

	reg := NewRegistry()
	tracer := &Tracer{}
	ready := NewReadiness()

	// A stand-in for the real thing: the process accepts traffic only once
	// it has warmed up. Watch /readyz flip while /healthz stays 200.
	started := time.Now()
	ready.Register("warmup", func(ctx context.Context) error {
		if time.Since(started) < 5*time.Second {
			return errors.New("still warming up")
		}
		return nil
	})

	handler := Chain(NewMux(reg, ready),
		RequestID,       // every signal needs the id, so it goes first
		RouteContext,    // plant the route box before anyone reads it
		Tracing(tracer), // start the span before the logger reads its ids
		Observe(reg, logger),
		Recover, // inside Observe: a panic is a counted, logged 500
	)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("serve failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown failed", "err", err)
	}
	logger.Info("shutdown complete")
}
