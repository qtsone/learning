// Command timesvc is the service you have carried through this pack: probes
// that mean two different things, a shutdown path that drains, configuration
// from the environment. It is finished — do not change it. What is missing is
// everything around it: the image, the pipeline that builds and ships it, and
// the runbook whoever is paged at 3 a.m. will read.
//
// One line here is new: buildRevision is stamped in at link time, and GET
// /version reports it. That is the thread that ties a running pod back to a
// commit, and it is what makes "which revision is actually deployed?" a
// question with an answer.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	// Embeds the IANA database so /now works on a base image with no
	// /usr/share/zoneinfo — the dockerizing-go lesson's reason, unchanged.
	_ "time/tzdata"
)

// buildRevision is overwritten at build time with the commit the binary was
// built from:
//
//	go build -ldflags="-X main.buildRevision=$REVISION"
//
// The default is what an unstamped local build reports, and seeing "dev" in a
// cluster is itself a finding.
var buildRevision = "dev"

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
	mux.HandleFunc("GET /version", a.version)
	mux.HandleFunc("GET /now", a.now)
	return mux
}

// healthz answers the liveness probe: "is this process still a working
// server, or has it wedged badly enough that only a restart will fix it?"
func (a *app) healthz(w http.ResponseWriter, _ *http.Request) {
	_, _ = io.WriteString(w, "ok\n")
}

// readyz answers the readiness probe: "should this pod be receiving new
// requests right now?" It is 503 before the server is up and 503 for the
// whole drain, which is how the pod asks to leave the Service's endpoints.
func (a *app) readyz(w http.ResponseWriter, _ *http.Request) {
	if !a.ready.Load() {
		http.Error(w, "draining", http.StatusServiceUnavailable)
		return
	}
	_, _ = io.WriteString(w, "ok\n")
}

type versionResponse struct {
	Service  string `json:"service"`
	Revision string `json:"revision"`
}

// version reports which commit this binary was built from — the check that
// turns "the rollout finished" into "the rollout finished with my change".
func (a *app) version(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(versionResponse{Service: "timesvc", Revision: buildRevision})
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

// run serves on ln until ctx is cancelled, then drains and returns.
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

	// Stop advertising first, stop accepting second — the ordering rule from
	// the deploying-go lesson.
	a.ready.Store(false)
	a.log.Info("draining", "timeout", shutdownTimeout.String())

	stopCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(stopCtx); err != nil {
		return fmt.Errorf("drain: %w", err)
	}

	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(2)
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.logLevel}))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ln, err := net.Listen("tcp", ":"+cfg.port)
	if err != nil {
		log.Error("listen failed", "err", err)
		os.Exit(1)
	}
	// The revision goes in the first log line too: the pod's logs answer
	// "what is running here?" without anyone reaching for /version.
	log.Info("listening",
		"addr", ln.Addr().String(),
		"revision", buildRevision,
		"shutdown_timeout", cfg.shutdownTimeout.String(),
	)

	a := newApp(log)
	if err := a.run(ctx, ln, cfg.shutdownTimeout); err != nil {
		log.Error("server stopped with an error", "err", err)
		os.Exit(1)
	}
	log.Info("stopped cleanly")
}
