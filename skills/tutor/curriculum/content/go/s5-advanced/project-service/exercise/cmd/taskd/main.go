// Command taskd runs the task service: it reads configuration from the
// environment, wires the layers together, and serves until it is told to
// stop. Everything interesting lives in the packages it imports — main is a
// composition root, not a place to put logic.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"tutor.local/project-service/httpapi"
	"tutor.local/project-service/sqlitestore"
	"tutor.local/project-service/task"
)

// requestTimeout is the per-request budget. It is shorter than the server's
// WriteTimeout so a slow handler is answered with a 503 rather than having
// its connection cut mid-response.
const requestTimeout = 5 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("service stopped", "err", err)
		os.Exit(1)
	}
	logger.Info("shutdown complete")
}

func run(logger *slog.Logger) error {
	tokens, err := parseTokens(os.Getenv("TASKD_TOKENS"))
	if err != nil {
		return err
	}
	addr := envOr("TASKD_ADDR", "127.0.0.1:8080")
	dbPath := envOr("TASKD_DB", "taskd.db")

	// The signal-aware context is the whole program's stop button: Ctrl-C
	// or the SIGTERM a process manager sends before a deploy cancels it.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := sqlitestore.Open(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}
	defer store.Close()

	svc := task.NewService(store, time.Now)
	api := httpapi.New(svc, httpapi.NewMetrics(), logger)

	// Order matters: RequestID first so everything downstream can log the
	// id; AccessLog outside Recover so a panic still produces one access
	// line, with status 500; Timeout innermost so it bounds the handler
	// rather than the middleware around it.
	handler := httpapi.Chain(api.Routes(httpapi.Auth(tokens)),
		httpapi.RequestID,
		httpapi.AccessLog(logger),
		httpapi.Recover(logger),
		httpapi.Timeout(requestTimeout),
	)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	logger.Info("listening", "addr", ln.Addr().String(), "db", dbPath, "clients", len(tokens))
	return httpapi.Run(ctx, httpapi.NewServer(addr, handler), ln)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// parseTokens reads "client:token,client:token" into a map. Credentials come
// from the environment and the service refuses to start without them: a
// default token in source is a default token in production.
func parseTokens(raw string) (map[string]string, error) {
	tokens := make(map[string]string)
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		name, token, ok := strings.Cut(pair, ":")
		if !ok || name == "" || token == "" {
			return nil, fmt.Errorf(`TASKD_TOKENS: %q is not "client:token"`, pair)
		}
		tokens[name] = token
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf(`TASKD_TOKENS is empty: set it to "client:token[,client:token]"`)
	}
	return tokens, nil
}
