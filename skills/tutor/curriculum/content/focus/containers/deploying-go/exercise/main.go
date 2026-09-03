// Command timesvc is the service you containerized earlier in this pack,
// now being taught how to die politely.
//
// Configuration, routing, logging and the /now handler are finished. Four
// things are not, and they are exactly the four Kubernetes cares about: a
// readiness endpoint that tells the truth, a liveness endpoint that means
// something different from readiness, a shutdown path that drains instead of
// severing, and a process that hears SIGTERM at all. Look for TODO.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	// Embeds the IANA database so /now works on a base image with no
	// /usr/share/zoneinfo — the dockerizing-go lesson's reason, unchanged.
	_ "time/tzdata"
)

type config struct {
	port            string
	shutdownTimeout time.Duration
	logLevel        slog.Level
}

// loadConfig reads the whole configuration from the environment: one process,
// one set of env vars, no config file to mount and no reload story.
func loadConfig() (config, error) {
	cfg := config{
		port:            "8080",
		shutdownTimeout: 15 * time.Second,
		logLevel:        slog.LevelInfo,
	}
	if v := os.Getenv("PORT"); v != "" {
		cfg.port = v
	}
	if v := os.Getenv("SHUTDOWN_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return cfg, fmt.Errorf("SHUTDOWN_TIMEOUT %q: %w", v, err)
		}
		cfg.shutdownTimeout = d
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		if err := cfg.logLevel.UnmarshalText([]byte(v)); err != nil {
			return cfg, fmt.Errorf("LOG_LEVEL %q: %w", v, err)
		}
	}
	return cfg, nil
}

type app struct {
	log *slog.Logger
	// ready is false until the server is serving, and false again from the
	// moment shutdown starts. It is the only state the probes read.
	ready atomic.Bool
}

func newApp(log *slog.Logger) *app {
	return &app{log: log}
}

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.healthz)
	mux.HandleFunc("GET /readyz", a.readyz)
	mux.HandleFunc("GET /now", a.now)
	return mux
}

// healthz answers the liveness probe: "is this process still a working
// server, or has it wedged badly enough that only a restart will fix it?"
func (a *app) healthz(w http.ResponseWriter, _ *http.Request) {
	// TODO 2: this is wrong on purpose. Liveness currently mirrors readiness,
	// so the instant the pod starts draining it also tells the kubelet the
	// process is dead — and the kubelet restarts the container in the middle
	// of finishing real requests. Liveness must answer 200 for as long as this
	// process can serve at all.
	if !a.ready.Load() {
		http.Error(w, "draining", http.StatusServiceUnavailable)
		return
	}
	_, _ = io.WriteString(w, "ok\n")
}

// readyz answers the readiness probe: "should this pod be receiving new
// requests right now?"
func (a *app) readyz(w http.ResponseWriter, _ *http.Request) {
	// TODO 1: report the truth. While a.ready is false, answer 503 so the
	// platform takes this pod out of the Service's endpoints instead of
	// routing new work to a process that is on its way out.
	_, _ = io.WriteString(w, "ok\n")
}

type nowResponse struct {
	Service string `json:"service"`
	Zone    string `json:"zone"`
	Now     string `json:"now"`
}

// now is the real work. ?delay= makes a request slow on purpose, which is how
// you watch a drain actually drain.
func (a *app) now(w http.ResponseWriter, r *http.Request) {
	if v := r.URL.Query().Get("delay"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d < 0 || d > 5*time.Second {
			http.Error(w, "delay must be a duration between 0 and 5s", http.StatusBadRequest)
			return
		}
		select {
		case <-time.After(d):
		case <-r.Context().Done():
			return
		}
	}
	zone := r.URL.Query().Get("zone")
	if zone == "" {
		zone = "UTC"
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		http.Error(w, "unknown time zone: "+zone, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(nowResponse{
		Service: "timesvc",
		Zone:    zone,
		Now:     time.Now().In(loc).Format(time.RFC3339),
	})
}

// run serves on ln until ctx is cancelled, then stops the server and returns.
func (a *app) run(ctx context.Context, ln net.Listener, shutdownTimeout time.Duration) error {
	srv := &http.Server{
		Handler:           a.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveErr := make(chan error, 1)
	a.ready.Store(true)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	// TODO 3: Close severs every open connection, in-flight requests included:
	// the caller sees a reset, and your 99th percentile sees it too. Drain
	// instead — stop advertising readiness first, then let
	// http.Server.Shutdown finish the requests already accepted, bounded by
	// shutdownTimeout because the grace period is not infinite. Remember that
	// Serve returns http.ErrServerClosed on a clean stop; that is success, not
	// an error to report.
	a.log.Info("stopping")
	return srv.Close()
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(2)
	}

	// Structured logs, on stdout, one JSON object per line: the platform
	// contract from the observability lesson. No log files, no rotation.
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.logLevel}))

	// TODO 4: Kubernetes asks a container to stop by sending SIGTERM to PID 1.
	// Nothing here is listening for it, so this process ignores the request
	// entirely and runs until the grace period expires and the kubelet sends
	// SIGKILL — which no code can handle. Derive a context that is cancelled
	// on SIGTERM (and on SIGINT, so Ctrl-C behaves the same on your laptop).
	ctx := context.Background()

	ln, err := net.Listen("tcp", ":"+cfg.port)
	if err != nil {
		log.Error("listen failed", "err", err)
		os.Exit(1)
	}
	log.Info("listening",
		"addr", ln.Addr().String(),
		"shutdown_timeout", cfg.shutdownTimeout.String(),
	)

	a := newApp(log)
	if err := a.run(ctx, ln, cfg.shutdownTimeout); err != nil {
		log.Error("server stopped with an error", "err", err)
		os.Exit(1)
	}
	log.Info("stopped cleanly")
}
