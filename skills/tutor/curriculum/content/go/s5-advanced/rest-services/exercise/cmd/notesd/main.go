// Command notesd wires the layers together and serves the notes API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tutor.local/rest-services/api"
	"tutor.local/rest-services/memstore"
	"tutor.local/rest-services/note"
)

const addr = ":8080"

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	// main is the only place that names a storage implementation: swapping
	// memstore for a database is this line, and nothing else.
	svc := note.NewService(memstore.New())

	// Same backstops as the http-servers lesson — a zero timeout is
	// "wait forever", and forever is how connections leak.
	srv := &http.Server{
		Addr:         addr,
		Handler:      api.Logging(api.New(svc).Routes()),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		slog.Error("listen failed", "addr", srv.Addr, "err", err)
		os.Exit(1)
	}
	slog.Info("listening", "addr", ln.Addr().String())

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("serve failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "err", err)
	}
	slog.Info("shutdown complete")
}
