// Command docsd runs the documents service. Wiring only: the store, the
// policy, and the HTTP server shape you built in S5.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"tutor.local/authorization/api"
	"tutor.local/authorization/authz"
	"tutor.local/authorization/doc"
	"tutor.local/authorization/memstore"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	store := memstore.New(
		doc.Document{ID: "alice-doc", OwnerID: "alice", Title: "Alice's notes"},
		doc.Document{ID: "bob-doc", OwnerID: "bob", Title: "Bob's draft"},
	)
	// One policy, built once, shared by every request. The audit log is the
	// same logger the rest of the service uses.
	policy := authz.NewPolicy(authz.DefaultRules, log)
	srv := &http.Server{
		Addr:              "localhost:8080",
		Handler:           api.New(store, policy).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		log.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown failed", "err", err)
	}
}
